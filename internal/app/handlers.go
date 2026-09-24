package app

import "net/http"

func (a *App) StaticHandler(w http.ResponseWriter, r *http.Request) {
	handler := http.FileServer(http.FS(a.static))
	handler = http.StripPrefix("/"+staticPath+"/", handler)

	handler.ServeHTTP(w, r)
}

func (a *App) InternalServerError(w http.ResponseWriter, r *http.Request) {
	http.Error(
		w,
		http.StatusText(http.StatusInternalServerError),
		http.StatusInternalServerError,
	)
}

func (a *App) MethodNotAllowedError(w http.ResponseWriter, r *http.Request) {
	http.Error(
		w,
		http.StatusText(http.StatusMethodNotAllowed),
		http.StatusMethodNotAllowed,
	)
}

func (a *App) NotFoundError(w http.ResponseWriter, r *http.Request) {
	http.Error(
		w,
		http.StatusText(http.StatusNotFound),
		http.StatusNotFound,
	)
}
