package app

import (
	"errors"
	"io/fs"
	"net/http"

	viewspkg "github.com/zeldojov/zexgo/internal/views"
)

const (
	templatesPath = "templates"
	staticPath    = "static"
)

type App struct {
	views  *viewspkg.Views
	static fs.FS
	mux    *http.ServeMux
	Config Config
	State  State
}

type Config struct {
	Environment string
}

type State struct {
	// aplikaciono stanje
}

// region helpers

// endregion helpers
// region API

func NewApp(config Config) *App {
	return &App{
		Config: config,
		mux:    http.NewServeMux(),
	}
}

// endregion API
func (a *App) Handler() (http.Handler, error) {
	if a.static == nil {
		return nil, errors.New("static files are not initialized")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/static/", a.MethodNotAllowedError)
	mux.HandleFunc("/static", a.MethodNotAllowedError)
	mux.HandleFunc("GET /static", a.NotFoundError)
	mux.HandleFunc("GET /static/", a.StaticHandler)

	// Sve ostale rute prosleđujemo main mux-u.
	mux.Handle("/", a.mux)

	return mux, nil
}

func (a *App) Handle(pattern string, handler http.Handler) {
	a.mux.Handle(pattern, handler)
}

func (a *App) HandleFunc(pattern string, handler http.HandlerFunc) {
	a.mux.HandleFunc(pattern, handler)
}
