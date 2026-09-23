package views

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path"
)

var (
	ErrTemplateWalk  = errors.New("template walk error")
	ErrTemplateParse = errors.New("template parse error")
)

type Views struct {
	tmpl *template.Template
}

// region helpers

func loadViews(fsys fs.FS, tmplPath string, funcs template.FuncMap) (*template.Template, error) {
	var paths []string

	err := fs.WalkDir(fsys, tmplPath, func(templatePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || path.Ext(templatePath) != ".html" {
			return nil
		}

		paths = append(paths, templatePath)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w under %q: %w", ErrTemplateWalk, tmplPath, err)

	}

	tmpl, err := template.
		New("views").
		Funcs(funcs).
		ParseFS(fsys, paths...)
	if err != nil {
		return nil, fmt.Errorf("%w under %q: %w", ErrTemplateParse, tmplPath, err)
	}

	return tmpl, nil
}

// endregion helpers
// region API

func NewViews(fsys fs.FS, tmplPath string, tmplFuncs template.FuncMap) (*Views, error) {

	tmpl, err := loadViews(fsys, tmplPath, tmplFuncs)
	if err != nil {
		return nil, err
	}

	required := []struct {
		filename     string
		templateName string
	}{
		{"internalServerError.html", "errors/internalServerError"},
		{"notFound.html", "errors/notFound"},
	}

	for _, item := range required {
		templatePath := path.Join(tmplPath, "errors", item.filename)

		if _, err := fs.Stat(fsys, templatePath); err != nil {
			return nil, fmt.Errorf("required error template %q: %w", templatePath, err)
		}

		if tmpl.Lookup(item.templateName) == nil {
			return nil, fmt.Errorf("required template %q is not defined in %q", item.templateName, templatePath)
		}
	}

	return &Views{
		tmpl: tmpl,
	}, nil
}

func (v *Views) ExecuteTemplate(w io.Writer, name string, data any) error {
	return v.tmpl.ExecuteTemplate(w, name, data)
}

// endregion API
