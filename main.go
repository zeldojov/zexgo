package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	apppkg "github.com/zeldojov/zexgo/internal/app"
)

//go:embed templates
var templateFS embed.FS

var templateFuncs = template.FuncMap{
	"upper": strings.ToUpper,
}

var config = apppkg.Config{
	Environment:   "development",
	TemplatesPath: "templates",
}

func main() {

	app := apppkg.NewApp(config)

	if err := app.InitViews(templateFS, templateFuncs); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

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

	fmt.Println("Server running at http://localhost:8080")
	http.ListenAndServe(":8080", mux)
}
