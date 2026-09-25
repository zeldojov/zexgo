package app

import "net/http"

func AllowMethods(app *application) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodPost:
				next.ServeHTTP(w, r)
			default:
				w.Header().Set("Allow", "GET, HEAD, POST")
				app.MethodNotAllowedError(w, r)
			}
		})
	}
}
