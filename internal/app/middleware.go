package app

import (
	"errors"
	"log"
	"net/http"

	"github.com/zeldojov/zexgo/internal/ratelimiter"
	"github.com/zeldojov/zexgo/internal/session"
	"github.com/zeldojov/zexgo/internal/utils"
)

func (a *application) AllowedMethods(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodPost:
			next.ServeHTTP(w, r)
		default:
			w.Header().Set("Allow", "GET, HEAD, POST")
			a.MethodNotAllowed(w, r)
		}
	})
}

func (a *application) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := session.GetSession(r)
		if !ok {
			log.Printf("session missing from request")
			a.InternalServerError(w, r)
			return
		}

		if !sess.IsAuthenticated() {
			if r.Method == http.MethodGet {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *application) Guest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := session.GetSession(r)
		if !ok {
			log.Printf("session missing from request")
			a.InternalServerError(w, r)
			return
		}

		if sess.IsAuthenticated() {
			http.Redirect(w, r, "/user/home", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *application) CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			sess, ok := session.GetSession(r)
			if !ok {
				log.Printf("session missing from request")
				a.InternalServerError(w, r)
				return
			}

			if !sess.ValidateCSRFToken(r) {
				log.Println("csrf token failed validation")
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (a *application) RateLimiter(limiter *ratelimiter.RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow(utils.GetClientIP(r)) {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *application) Session(next http.Handler) http.Handler {
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
				a.InternalServerError(w, r)
				return
			}
			newSession = true
		case err != nil:
			a.InternalServerError(w, r)
			return
		default:
			sess, err = a.store.NewSessionFromDB(r.Context(), cookie.Value)
			if err != nil {
				if !errors.Is(err, session.ErrSessionNotFound) || r.Method == http.MethodPost {
					if errors.Is(err, session.ErrSessionNotFound) {
						http.Error(w, "forbidden", http.StatusForbidden)
					} else {
						a.InternalServerError(w, r)
					}
					return
				}
				session.UnsetSessionCookie(w)
				sess, err = session.NewSessionFromReq(r)
				if err != nil {
					a.InternalServerError(w, r)
					return
				}
				newSession = true
			}
		}

		if newSession {
			if err := a.store.SaveSession(r.Context(), sess); err != nil {
				a.InternalServerError(w, r)
				return
			}
			if err := session.SetSessionCookie(w, sess.ID()); err != nil {
				a.InternalServerError(w, r)
				return
			}
		}

		r = session.SetSession(sess, r)
		next.ServeHTTP(w, r)

		if err := a.store.SaveSession(r.Context(), sess); err != nil {
			log.Printf("failed to save session %q: %v", sess.ID(), err)
		}
	})
}

func (a *application) ValidateSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := session.GetSession(r)
		if !ok {
			a.InternalServerError(w, r)
			return
		}

		if sess.IsExpired() || !sess.MatchUserAgent(r) || !sess.MatchIP(r) {
			if r.Method != http.MethodGet {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			if err := a.store.DeleteSession(r.Context(), sess.ID()); err != nil && !errors.Is(err, session.ErrSessionNotFound) {
				a.InternalServerError(w, r)
				return
			}
			session.UnsetSessionCookie(w)
			var err error
			sess, err = session.NewSessionFromReq(r)
			if err != nil {
				a.InternalServerError(w, r)
				return
			}
			if err := a.store.SaveSession(r.Context(), sess); err != nil {
				a.InternalServerError(w, r)
				return
			}
			if err := session.SetSessionCookie(w, sess.ID()); err != nil {
				a.InternalServerError(w, r)
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
