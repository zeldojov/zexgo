package app

import (
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"slices"

	staticfspkg "github.com/zeldojov/zexgo/internal/staticfs"
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

	application struct {
		views  *viewspkg.Views
		static fs.FS

		mux    *http.ServeMux
		Config Config

		staticHandler         http.Handler
		NotFoundError         http.HandlerFunc
		InternalServerError   http.HandlerFunc
		MethodNotAllowedError http.HandlerFunc
		middlewares           []registeredMiddleware
	}
)

// region helpers

// endregion helpers
// region API

func NewApp(config Config, staticFS fs.FS, templatesFS fs.FS, funcs template.FuncMap) (*application, error) {

	if err := ValidateDirectoryPath(staticFS, staticPath); err != nil {
		return nil, fmt.Errorf("%q: %w", "invalid static path", err)
	}
	if err := ValidateDirectoryPath(templatesFS, templatesPath); err != nil {
		return nil, fmt.Errorf("%q: %w", "invalid templates path", err)
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

		NotFoundError: func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		},
		InternalServerError: func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		},
		MethodNotAllowedError: func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		},

		middlewares: []registeredMiddleware{},
	}

	app.middlewares = append(app.middlewares, registeredMiddleware{
		name:       "allow-methods",
		middleware: AllowMethods(app),
	})

	return app, nil
}

// endregion API
func (a *application) Handler() (http.Handler, error) {
	if a.static == nil {
		return nil, errors.New("static files are not initialized")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/static/", a.MethodNotAllowedError)
	mux.HandleFunc("/static", a.MethodNotAllowedError)
	mux.HandleFunc("GET /static", a.NotFoundError)
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

	if name == "allow-methods" {
		return errors.New("middleware name is reserved")
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
