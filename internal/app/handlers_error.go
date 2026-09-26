package app

import "net/http"

func (a *application) NotFound(w http.ResponseWriter, r *http.Request) {
	a.RenderStatus(w, http.StatusNotFound, "public/errors/404", errorPageData{
		Title:   "Page not found",
		Code:    http.StatusNotFound,
		Message: http.StatusText(http.StatusNotFound),
	})
}

func (a *application) InternalServerError(w http.ResponseWriter, r *http.Request) {
	a.RenderStatus(w, http.StatusInternalServerError, "public/errors/500", errorPageData{
		Title:   "Internal server error",
		Code:    http.StatusInternalServerError,
		Message: http.StatusText(http.StatusInternalServerError),
	})
}

func (a *application) MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	a.RenderStatus(w, http.StatusMethodNotAllowed, "public/errors/405", errorPageData{
		Title:   "Method not allowed",
		Code:    http.StatusMethodNotAllowed,
		Message: http.StatusText(http.StatusMethodNotAllowed),
	})
}
