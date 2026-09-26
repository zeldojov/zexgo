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
	"github.com/zeldojov/zexgo/internal/handlers"
	"github.com/zeldojov/zexgo/internal/middleware"
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
		store   *store.Store
		email   *email.Service
		handler *handlers.Handler

		views  renderer
		static fs.FS
		mux    *http.ServeMux

		Config Config

		staticHandler http.Handler
		middlewares   []registeredMiddleware
	}
)

// region helpers

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

	app := &application{
		views:  views,
		static: staticFiles,

		mux:    http.NewServeMux(),
		Config: config,

		staticHandler: http.StripPrefix("/"+staticPath+"/", http.FileServer(http.FS(staticFiles))),
		middlewares:   []registeredMiddleware{},
	}

	app.handler = handlers.NewHandler(
		app.store,
		app.email,
		app.views,
		http.HandlerFunc(app.InternalServerError),
	)

	if err := app.Middleware("allow-methods", func(app *application) Middleware {
		return Middleware(middleware.NewAllowedMethods(app.MethodNotAllowed))
	}); err != nil {
		return nil, err
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
