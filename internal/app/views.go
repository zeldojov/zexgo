package app

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
)

// region helpers
// endregion helpers
// region API

func (a *application) Render(w http.ResponseWriter, name string, data any) {
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
