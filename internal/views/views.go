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
	templates map[string]*template.Template
}

// region helpers

func loadViews(fsys fs.FS, tmplPath string, funcs template.FuncMap) (map[string]*template.Template, error) {
	layoutsPath := path.Join(tmplPath, "layouts")
	partialsPath := path.Join(tmplPath, "partials")
	pagesPath := path.Join(tmplPath, "pages")

	layouts, err := loadLayouts(fsys, layoutsPath, partialsPath, funcs)
	if err != nil {
		return nil, err
	}

	if err := validatePagesDirs(fsys, pagesPath, layouts); err != nil {
		return nil, err
	}

	if err := validatePartialsDirs(fsys, partialsPath, layouts); err != nil {
		return nil, err
	}

	templates := make(map[string]*template.Template)
	for layoutName, layoutTmpl := range layouts {
		layoutPagesPath := path.Join(pagesPath, layoutName)

		layoutTemplates, err := loadPages(
			fsys,
			layoutPagesPath,
			layoutName,
			layoutTmpl,
		)
		if err != nil {
			return nil, err
		}

		for pageName, pageTmpl := range layoutTemplates {
			templates[pageName] = pageTmpl
		}
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
	if v == nil || v.templates == nil {
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

	if tmpl.Lookup("layout") == nil {
		return errors.New(`template "layout" is not defined`)
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		return fmt.Errorf("failed to execute template %q: %w", name, err)
	}

	return nil
}

// endregion API
