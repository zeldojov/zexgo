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

// region helpers
// endregion helpers
// region API

func (a *App) InitViews(fsys fs.FS, funcs template.FuncMap) error {

	if err := ValidateDirectoryPath(fsys, templatesPath); err != nil {
		return fmt.Errorf("%q: %w", "invalid templates path", err)
	}

	views, err := viewspkg.New(fsys, templatesPath, funcs)
	if err != nil {
		return fmt.Errorf("application failed to initialize views: %w", err)
	}

	a.views = views
	return nil
}

func (a *App) Render(w http.ResponseWriter, name string, data any) {
	var buf bytes.Buffer

	if a.views == nil {
		err := errors.New("views not initialized")

		log.Printf("template render failed: %q: %v", name, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := a.views.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("template render failed: %q: %v", name, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, &buf); err != nil {
		log.Printf("template response write failed: %q: %v", name, err)
	}
}

// endregion API
