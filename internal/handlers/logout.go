package handlers

import (
	"net/http"

	"github.com/zeldojov/zexgo/internal/session"
)

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.GetSession(r)
	if !ok {
		h.failInternalServerError(w, r)
		return
	}

	if err := h.store.DeleteSession(r.Context(), sess.ID()); err != nil {
		h.failInternalServerError(w, r)
		return
	}

	session.UnsetSessionCookie(w)

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
