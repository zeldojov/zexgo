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
	// ErrTemplateWalk identifies an error while recursively walking page or
	// partial template directories.
	ErrTemplateWalk = errors.New("template walk error")
	// ErrTemplateParse identifies an error while parsing or cloning a template.
	ErrTemplateParse = errors.New("template parse error")
)

// Views contains parsed page templates keyed by their registered names.
//
// Create a Views value with New. A zero-value Views is not ready for use.
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

// New loads all layouts, partials, and pages below tmplPath.
//
// The filesystem path must use io/fs slash-separated paths. The template root
// must contain layouts, partials, and pages directories. Every layout must
// have a matching partials and pages directory, and each of those directories
// must contain at least one HTML file. Partials and pages are discovered
// recursively.
//
// tmplFuncs is applied before parsing, so every function used by a template
// must be present in the map. Errors wrapping ErrTemplateWalk or
// ErrTemplateParse can be identified with errors.Is.
func New(fsys fs.FS, tmplPath string, tmplFuncs template.FuncMap) (*Views, error) {
	if fsys == nil {
		return nil, errors.New("template filesystem is nil")
	}

	templates, err := loadViews(fsys, tmplPath, tmplFuncs)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates from %q: %w", tmplPath, err)
	}

	return &Views{templates: templates}, nil
}

// ExecuteTemplate renders the registered page named name into w.
//
// name is the page path relative to the pages root, without the .html
// extension. For example, pages/public/admin/index.html is registered as
// public/admin/index. The page is executed through its layout, so the layout
// must define a template named "layout" and the page must define "content".
//
// The rendered output is written to w. The method returns an error when Views
// is not initialized, w or name is invalid, the page is not registered, or
// template execution fails.
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
