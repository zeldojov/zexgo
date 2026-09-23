package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path"
	"strings"
)

//go:embed templates
var templateFS embed.FS

type Templates struct {
	pages map[string]*template.Template
}

type TemplateSets struct {
	Public *Templates
	Guest  *Templates
	Auth   *Templates
}

func main() {
	fmt.Println("Hello, World!")

	funcs := template.FuncMap{
		"upper": strings.ToUpper,
	}

	templates, err := loadTemplates(funcs)
	if err != nil {
		panic(err)
	}
}

func parseTemplateSet(name string, funcs template.FuncMap) (*Templates, error) {
	basePath := fmt.Sprintf("templates/%s/base.html", name)
	pagePattern := fmt.Sprintf("templates/%s/pages/*", name)

	pagePaths, err := fs.Glob(templateFS, pagePattern)
	if err != nil {
		return nil, err
	}

	result := &Templates{
		pages: make(map[string]*template.Template),
	}

	for _, pagePath := range pagePaths {
		pageName := strings.TrimSuffix(path.Base(pagePath), path.Ext(pagePath))

		tmpl, err := template.
			New(pageName).
			Funcs(funcs).
			ParseFS(templateFS, basePath, pagePath)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", pagePath, err)
		}

		result.pages[pageName] = tmpl
	}

	return result, nil
}

func loadTemplates(funcs template.FuncMap) (*TemplateSets, error) {
	public, err := parseTemplateSet("public", funcs)
	if err != nil {
		return nil, err
	}

	guest, err := parseTemplateSet("guest", funcs)
	if err != nil {
		return nil, err
	}

	auth, err := parseTemplateSet("auth", funcs)
	if err != nil {
		return nil, err
	}

	return &TemplateSets{
		Public: public,
		Guest:  guest,
		Auth:   auth,
	}, nil
}
