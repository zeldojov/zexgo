package app

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"

	viewspkg "github.com/zeldojov/zexgo/internal/views"
)

var (
	ErrTemplateRender = errors.New("template render error")
	ErrTemplateWrite  = errors.New("template write error")
)

type App struct {
	views  *viewspkg.Views
	Config Config
	State  State
}

type Config struct {
	Environment   string
	TemplatesPath string
}

type State struct {
	// aplikaciono stanje
}

// region helpers

func (a *App) renderWithStatus(w http.ResponseWriter, name string, data any, statusCode int) error {
	var buf bytes.Buffer

	if a.views == nil {
		return fmt.Errorf("%w %q: %w", ErrTemplateRender, name, errors.New("views not initialized"))
	}
	if err := a.views.ExecuteTemplate(&buf, name, data); err != nil {
		return fmt.Errorf("%w %q: %w", ErrTemplateRender, name, err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)

	if _, err := io.Copy(w, &buf); err != nil {
		return fmt.Errorf("%w %q: %w", ErrTemplateWrite, name, err)
	}

	return nil
}

func handleRenderError(w http.ResponseWriter, err error, fallbackStatus int) {
	log.Printf("render failed: %v", err)

	if errors.Is(err, ErrTemplateRender) {
		http.Error(w, http.StatusText(fallbackStatus), fallbackStatus)
	}
}

// endregion helpers
// region API

func NewApp(config Config) *App {
	return &App{
		Config: config,
	}
}

func (a *App) InitViews(fsys fs.FS, funcs template.FuncMap) error {
	views, err := viewspkg.NewViews(fsys, a.Config.TemplatesPath, funcs)
	if err != nil {
		return fmt.Errorf("application failed to initialize views: %w", err)
	}

	a.views = views
	return nil
}

func (a *App) Render(w http.ResponseWriter, name string, data any) {
	if err := a.renderWithStatus(w, name, data, http.StatusOK); err != nil {
		handleRenderError(w, err, http.StatusInternalServerError)
	}
}

func (a *App) InternalServerError(w http.ResponseWriter) {
	if err := a.renderWithStatus(w, "errors/internalServerError", nil, http.StatusInternalServerError); err != nil {
		handleRenderError(w, err, http.StatusInternalServerError)
	}
}

func (a *App) NotFound(w http.ResponseWriter) {
	if err := a.renderWithStatus(w, "errors/notFound", nil, http.StatusNotFound); err != nil {
		handleRenderError(w, err, http.StatusNotFound)
	}
}

// endregion API
