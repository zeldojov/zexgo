package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	apppkg "github.com/zeldojov/zexgo/internal/app"
)

//go:embed templates
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

var templateFuncs = template.FuncMap{
	"upper": strings.ToUpper,
}

var config = apppkg.Config{
	Environment: "development",
}

func main() {

	app, err := apppkg.NewApp(config, staticFS, templateFS, templateFuncs)
	if err != nil {
		log.Fatal(err)
	}

	app.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Title string
			Name  string
		}{
			Title: "Home",
			Name:  "Željko",
		}

		app.Render(w, "public/index", data)
	})

	app.HandleFunc("GET /about", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Title string
			Name  string
		}{
			Title: "About",
			Name:  "Željko",
		}

		app.Render(w, "public/about", data)
	})

	handler, err := app.Handler()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Server running at http://localhost:8080")

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,

		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
