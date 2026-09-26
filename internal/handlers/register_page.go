package handlers

import (
	"net/http"

	"github.com/zeldojov/zexgo/internal/session"
)

func (h *Handler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.GetSession(r)
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	errorMessage, _ := sess.GetFlash("error")

	h.RenderStatus(w, http.StatusOK, "guest/register", struct {
		Title     string
		CSRFToken string
		Error     string
	}{
		Title:     "Register",
		CSRFToken: sess.CSRFToken(),
		Error:     errorMessage,
	})
}
