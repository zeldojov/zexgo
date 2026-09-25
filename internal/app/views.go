package app

import (
	"bytes"
	"io"
	"log"
	"net/http"
)

// region helpers
// endregion helpers
// region API

func (a *application) Render(w http.ResponseWriter, name string, data any) {
	var buf bytes.Buffer

	// a and a.views is guaranteed to be initialized by the constructor.

	if err := a.views.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("template render failed: %q: %v", name, err)
		a.InternalServerError(w, nil)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, &buf); err != nil {
		log.Printf("template response write failed: %q: %v", name, err)
	}
}

// endregion API
