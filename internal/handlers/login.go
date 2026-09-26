package handlers

import (
	"errors"
	"net/http"

	"github.com/zeldojov/zexgo/internal/session"
	"github.com/zeldojov/zexgo/internal/user"
)

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.GetSession(r)
	if !ok {
		h.failInternalServerError(w, r)
		return
	}

	username := r.PostFormValue("username")
	pass := r.PostFormValue("password")

	foundUser, err := h.store.GetUserByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			sess.SetFlash("error", "invalid credentials")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		h.failInternalServerError(w, r)
		return
	}

	if !foundUser.PasswordHash().Verify(pass) {
		sess.SetFlash("error", "invalid credentials")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Session fixation protection.
	if err := h.store.DeleteSession(r.Context(), sess.ID()); err != nil {
		h.failInternalServerError(w, r)
		return
	}

	session.UnsetSessionCookie(w)

	if err := sess.Authenticate(foundUser.ID(), r); err != nil {
		h.failInternalServerError(w, r)
		return
	}

	if err := h.store.SaveSession(r.Context(), sess); err != nil {
		h.failInternalServerError(w, r)
		return
	}

	if err := session.SetSessionCookie(w, sess.ID()); err != nil {
		h.failInternalServerError(w, r)
		return
	}

	http.Redirect(w, r, "/user/home", http.StatusSeeOther)
}
