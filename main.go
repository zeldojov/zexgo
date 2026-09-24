package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"strings"

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
	Environment:   "development",
	TemplatesPath: "templates",
	StaticPath:    "static",
}

func main() {

	app := apppkg.NewApp(config)

	if err := app.InitViews(templateFS, templateFuncs); err != nil {
		log.Fatal(err)
	}

	if err := app.InitStatic(staticFS); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/static", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	mux.Handle("/static/", app.StaticHandler())

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Title string
			Name  string
		}{
			Title: "Home",
			Name:  "Željko",
		}

		app.Render(w, "public/index", data)
	})

	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Title string
			Name  string
		}{
			Title: "About",
			Name:  "Željko",
		}

		app.Render(w, "public/about", data)
	})

	log.Println("Server running at http://localhost:8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
