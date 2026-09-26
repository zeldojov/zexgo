package handlers

import (
	"net/http"

	"github.com/zeldojov/zexgo/internal/session"
)

func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.GetSession(r)
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	errorMessage, _ := sess.GetFlash("error")

	h.RenderStatus(w, http.StatusOK, "guest/login", struct {
		Title     string
		CSRFToken string
		Error     string
	}{
		Title:     "Login",
		CSRFToken: sess.CSRFToken(),
		Error:     errorMessage,
	})
}
