package middleware

import (
	"net/http"

	"github.com/zeldojov/zexgo/internal/session"
	"github.com/zeldojov/zexgo/internal/store"
)

func ValidateSession(st *store.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := session.GetSession(r)
		if !ok {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if sess.IsExpired() ||
			!sess.MatchUserAgent(r) ||
			!sess.MatchIP(r) {

			if r.Method == http.MethodGet {
				if err := st.DeleteSession(r.Context(), sess.ID()); err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}

				session.UnsetSessionCookie(w)

				newSession, err := session.NewSessionFromReq(r)
				if err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				sess = newSession

				if err := st.SaveSession(r.Context(), sess); err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}

				if err := session.SetSessionCookie(w, sess.ID()); err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
			} else {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		if sess.ShouldRefresh() {
			sess.Touch()
		}

		r = session.SetSession(sess, r)
		next.ServeHTTP(w, r)
	})
}
