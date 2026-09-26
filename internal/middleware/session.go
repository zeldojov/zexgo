package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/zeldojov/zexgo/internal/session"
	"github.com/zeldojov/zexgo/internal/store"
)

func Session(st *store.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sess *session.Session
		newSession := false

		cookie, err := session.GetSessionCookie(r)

		switch {
		case errors.Is(err, session.ErrCookieNotFound):
			if r.Method == http.MethodPost {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			sess, err = session.NewSessionFromReq(r)
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			newSession = true

		case err != nil:
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return

		default:
			sess, err = st.NewSessionFromDB(r.Context(), cookie.Value)
			if err != nil {
				if errors.Is(err, session.ErrSessionNotFound) {
					if r.Method == http.MethodPost {
						http.Error(w, "forbidden", http.StatusForbidden)
						return
					}

					session.UnsetSessionCookie(w)

					sess, err = session.NewSessionFromReq(r)
					if err != nil {
						http.Error(w, "internal server error", http.StatusInternalServerError)
						return
					}
					newSession = true
				} else {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
			}
		}

		if newSession {
			if err := st.SaveSession(r.Context(), sess); err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if err := session.SetSessionCookie(w, sess.ID()); err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}

		r = session.SetSession(sess, r)

		next.ServeHTTP(w, r)

		if err := st.SaveSession(r.Context(), sess); err != nil {
			log.Printf(
				"failed to save session %q: %v",
				sess.ID(),
				err,
			)
		}
	})
}
