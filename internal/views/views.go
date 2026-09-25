package views

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path"
	"strings"
)

var (
	ErrTemplateWalk  = errors.New("template walk error")
	ErrTemplateParse = errors.New("template parse error")
)

type Views struct {
	templates map[string]*template.Template
}

// region helpers

func loadViews(fsys fs.FS, tmplPath string, funcs template.FuncMap) (map[string]*template.Template, error) {
	var paths []string

	err := fs.WalkDir(fsys, tmplPath, func(templatePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() || path.Ext(templatePath) != ".html" {
			return nil
		}

		paths = append(paths, templatePath)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w under %q: %w", ErrTemplateWalk, tmplPath, err)
	}

	layoutTemplates := make(map[string]*template.Template)
	templates := make(map[string]*template.Template)

	for _, pagePath := range paths {
		pageDir := path.Dir(pagePath)

		// Fajlovi direktno u templates/ su layout fajlovi.
		if pageDir == tmplPath {
			continue
		}

		layoutName := path.Base(pageDir)
		layoutPath := path.Join(tmplPath, layoutName+".html")

		layoutTmpl, ok := layoutTemplates[layoutName]
		if !ok {
			if _, err := fs.Stat(fsys, layoutPath); err != nil {
				return nil, fmt.Errorf(
					"required layout %q: %w",
					layoutPath,
					err,
				)
			}

			layoutTmpl, err = template.
				New("layout").
				Funcs(funcs).
				ParseFS(fsys, layoutPath)
			if err != nil {
				return nil, fmt.Errorf("%w under %q: %w", ErrTemplateParse, layoutPath, err)
			}

			layoutTemplates[layoutName] = layoutTmpl
		}

		pageTmpl, err := layoutTmpl.Clone()
		if err != nil {
			return nil, fmt.Errorf("%w while cloning layout %q: %w", ErrTemplateParse, layoutName, err)
		}

		if _, err := pageTmpl.ParseFS(fsys, pagePath); err != nil {
			return nil, fmt.Errorf("%w under %q: %w", ErrTemplateParse, pagePath, err)
		}

		pageName := path.Join(
			layoutName,
			strings.TrimSuffix(path.Base(pagePath), path.Ext(pagePath)),
		)

		templates[pageName] = pageTmpl
	}

	return templates, nil
}

// endregion helpers
// region API

func New(fsys fs.FS, tmplPath string, tmplFuncs template.FuncMap) (*Views, error) {
	templates, err := loadViews(fsys, tmplPath, tmplFuncs)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates from %q: %w", tmplPath, err)
	}

	return &Views{templates: templates}, nil
}

func (v *Views) ExecuteTemplate(w io.Writer, name string, data any) error {
	if v == nil {
		return errors.New("views are not initialized")
	}

	if w == nil {
		return errors.New("template writer is nil")
	}

	if name == "" {
		return errors.New("template name is empty")
	}

	tmpl, ok := v.templates[name]
	if !ok {
		return fmt.Errorf("template %q is not registered", name)
	}

	if tmpl.Lookup(name) == nil {
		return fmt.Errorf("template %q is not defined", name)
	}

	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		return fmt.Errorf("failed to execute template %q: %w", name, err)
	}

	return nil
}

// endregion API
