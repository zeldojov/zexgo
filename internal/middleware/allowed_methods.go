package middleware

import (
	"net/http"
)

func NewAllowedMethods(methodNotAllowedHandler func(w http.ResponseWriter, r *http.Request)) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodPost:
				next.ServeHTTP(w, r)

			default:
				w.Header().Set("Allow", "GET, HEAD, POST")
				methodNotAllowedHandler(w, r)
			}

		})
	}
}
