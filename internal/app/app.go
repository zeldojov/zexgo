package app

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"slices"

	"github.com/zeldojov/zexgo/internal/email"
	staticfspkg "github.com/zeldojov/zexgo/internal/staticfs"
	"github.com/zeldojov/zexgo/internal/store"
	viewspkg "github.com/zeldojov/zexgo/internal/views"
)

const (
	templatesPath = "templates"
	staticPath    = "static"
)

type (
	Middleware           func(http.Handler) http.Handler
	MiddlewareFactory    func(*application) Middleware
	registeredMiddleware struct {
		name       string
		middleware Middleware
	}

	Config struct {
		Environment string
		Database    DatabaseConfig
	}

	DatabaseConfig struct {
		Username string
		Password string
		Address  string
		Port     string
		Name     string
		Options  store.DBConfig
	}

	errorPageData struct {
		Title   string
		Code    int
		Message string
	}

	renderer interface {
		ExecuteTemplate(io.Writer, string, any) error
	}

	application struct {
		store *store.Store
		email *email.Service

		views  renderer
		static fs.FS
		mux    *http.ServeMux

		Config Config

		staticHandler http.Handler
		middlewares   []registeredMiddleware
	}
)

// region helpers

func initializeStore(config DatabaseConfig) (*store.Store, error) {
	dbOptions := config.Options
	if dbOptions == (store.DBConfig{}) {
		dbOptions = store.DefaultDBConfig()
	}

	db, err := store.ConnectWithConfig(
		config.Username,
		config.Password,
		config.Address,
		config.Port,
		config.Name,
		dbOptions,
	)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "failed to connect database", err)
	}

	st, err := store.NewStore(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("%q: %w", "failed to initialize store", err)
	}

	return st, nil
}

// endregion helpers
// region API

func NewApp(config Config, staticFS fs.FS, templatesFS fs.FS, funcs template.FuncMap) (*application, error) {

	if err := ValidateDirectoryPath(staticFS, staticPath); err != nil {
		return nil, fmt.Errorf("%q: %w", "invalid static path", err)
	}

	staticFiles, err := staticfspkg.New(staticFS, staticPath)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "failed to initialize static FS", err)
	}

	views, err := viewspkg.New(templatesFS, templatesPath, funcs)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "failed to initialize views", err)
	}

	st, err := initializeStore(config.Database)
	if err != nil {
		return nil, err
	}

	app := &application{
		store:  st,
		views:  views,
		static: staticFiles,

		mux:    http.NewServeMux(),
		Config: config,

		staticHandler: http.StripPrefix("/"+staticPath+"/", http.FileServer(http.FS(staticFiles))),
		middlewares:   []registeredMiddleware{},
	}

	app.HandleFunc("GET /register", app.RegisterPage)
	app.HandleFunc("POST /register", app.Register)
	app.HandleFunc("GET /login", app.LoginPage)
	app.HandleFunc("POST /login", app.Login)
	app.HandleFunc("POST /logout", app.Logout)
	app.HandleFunc("GET /user/home", app.UserHome)

	for _, middleware := range []struct {
		name string
		fn   func(http.Handler) http.Handler
	}{
		{name: "csrf", fn: app.CSRF},
		{name: "validate-session", fn: app.ValidateSession},
		{name: "session", fn: app.Session},
		{name: "allow-methods", fn: app.AllowedMethods},
	} {
		if err := app.Middleware(middleware.name, func(*application) Middleware {
			return Middleware(middleware.fn)
		}); err != nil {
			return nil, err
		}
	}

	return app, nil
}

// endregion API
func (a *application) Handler() (http.Handler, error) {
	if a.static == nil {
		return nil, errors.New("static files are not initialized")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/static/", a.MethodNotAllowed)
	mux.HandleFunc("/static", a.MethodNotAllowed)
	mux.HandleFunc("GET /static", a.NotFound)
	mux.Handle("GET /static/", a.staticHandler)

	// Sve ostale rute prosleđujemo main mux-u.
	mux.Handle("/", a.mux)

	var handler http.Handler = mux

	for _, middleware := range slices.Backward(a.middlewares) {
		handler = middleware.middleware(handler)
	}

	return handler, nil
}

func (a *application) Handle(pattern string, handler http.Handler) {
	a.mux.Handle(pattern, handler)
}

func (a *application) HandleFunc(pattern string, handler http.HandlerFunc) {
	a.mux.HandleFunc(pattern, handler)
}

func (a *application) CreateChain(middlewares ...Middleware) Middleware {
	return func(handler http.Handler) http.Handler {
		for _, middleware := range slices.Backward(middlewares) {
			handler = middleware(handler)
		}

		return handler
	}
}

func (a *application) Middleware(
	name string,
	factory MiddlewareFactory,
) error {
	if name == "" {
		return errors.New("middleware name is empty")
	}

	if factory == nil {
		return fmt.Errorf("middleware %q is nil", name)
	}

	for _, middleware := range a.middlewares {
		if middleware.name == name {
			return fmt.Errorf("middleware %q is already registered", name)
		}
	}

	a.middlewares = append(a.middlewares, registeredMiddleware{
		name:       name,
		middleware: factory(a),
	})

	return nil
}
