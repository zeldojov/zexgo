package handlers

import (
	"bytes"
	"io"
	"net/http"

	"github.com/zeldojov/zexgo/internal/email"
	"github.com/zeldojov/zexgo/internal/store"
)

type Renderer interface {
	ExecuteTemplate(io.Writer, string, any) error
}

type Handler struct {
	store               *store.Store
	email               *email.Service
	views               Renderer
	internalServerError http.Handler
}

func NewHandler(
	st *store.Store,
	emailService *email.Service,
	views Renderer,
	internalServerError ...http.Handler,
) *Handler {
	h := &Handler{
		store: st,
		email: emailService,
		views: views,
	}

	if len(internalServerError) > 0 {
		h.internalServerError = internalServerError[0]
	}

	return h
}

func (h *Handler) failInternalServerError(w http.ResponseWriter, r *http.Request) {
	if h.internalServerError != nil {
		h.internalServerError.ServeHTTP(w, r)
		return
	}

	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func (h *Handler) RenderStatus(
	w http.ResponseWriter,
	status int,
	name string,
	data any,
) {
	var buf bytes.Buffer

	if h.views == nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.views.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.Copy(w, &buf)
}
