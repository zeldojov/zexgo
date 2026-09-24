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
	Environment: "development",
}

func main() {

	app := apppkg.NewApp(config)

	if err := app.InitViews(templateFS, templateFuncs); err != nil {
		log.Fatal(err)
	}

	if err := app.InitStatic(staticFS); err != nil {
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

	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
