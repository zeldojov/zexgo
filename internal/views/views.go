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

func validateLayoutsDir(fsys fs.FS, layoutsPath string) ([]fs.DirEntry, error) {
	info, err := fs.Stat(fsys, layoutsPath)
	if err != nil {
		return nil, fmt.Errorf("layouts directory %q: %w", layoutsPath, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("layouts path %q is not a directory", layoutsPath)
	}

	entries, err := fs.ReadDir(fsys, layoutsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q: %w", layoutsPath, err)
	}

	return entries, nil
}

func validatePagesDirs(fsys fs.FS, pagesPath string, layouts map[string]*template.Template) error {
	info, err := fs.Stat(fsys, pagesPath)
	if err != nil {
		return fmt.Errorf("pages directory %q: %w", pagesPath, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("pages path %q is not a directory", pagesPath)
	}

	entries, err := fs.ReadDir(fsys, pagesPath)
	if err != nil {
		return fmt.Errorf("failed to read %q: %w", pagesPath, err)
	}

	for layoutName := range layouts {
		layoutPagesPath := path.Join(
			pagesPath,
			layoutName,
		)

		info, err := fs.Stat(fsys, layoutPagesPath)
		if err != nil {
			return fmt.Errorf("pages directory for layout %q is missing: %w", layoutName, err)
		}

		if !info.IsDir() {
			return fmt.Errorf("pages path for layout %q is not a directory", layoutName)
		}
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			return fmt.Errorf("unexpected file in pages directory: %q", entry.Name())
		}

		if _, ok := layouts[entry.Name()]; !ok {
			return fmt.Errorf("no layout found for pages directory %q", entry.Name())
		}
	}

	return nil
}

func loadLayouts(fsys fs.FS, layoutsPath string, entries []fs.DirEntry, funcs template.FuncMap) (map[string]*template.Template, error) {

	layouts := make(map[string]*template.Template)

	for _, entry := range entries {
		if entry.IsDir() {
			return nil, fmt.Errorf("layouts directory must be flat: %q", path.Join(layoutsPath, entry.Name()))
		}

		if path.Ext(entry.Name()) != ".html" {
			continue
		}

		layoutName := strings.TrimSuffix(
			entry.Name(),
			path.Ext(entry.Name()),
		)

		layoutPath := path.Join(layoutsPath, entry.Name())

		layoutTmpl, err := template.New("layout").Funcs(funcs).ParseFS(fsys, layoutPath)
		if err != nil {
			return nil, fmt.Errorf("%w under %q: %w", ErrTemplateParse, layoutPath, err)
		}

		if layoutTmpl.Lookup("layout") == nil {
			return nil, fmt.Errorf("template %q is not defined in %q", "layout", layoutPath)
		}

		layouts[layoutName] = layoutTmpl
	}

	if len(layouts) == 0 {
		return nil, fmt.Errorf("no layouts found under %q", layoutsPath)
	}

	return layouts, nil
}

func relativeFSPath(root, target string) (string, error) {
	root = strings.TrimSuffix(root, "/")
	prefix := root + "/"

	if !strings.HasPrefix(target, prefix) {
		return "", fmt.Errorf(
			"path %q is outside root %q",
			target,
			root,
		)
	}

	relativePath := strings.TrimPrefix(target, prefix)
	if relativePath == "" {
		return "", fmt.Errorf(
			"path %q has no relative part under %q",
			target,
			root,
		)
	}

	return relativePath, nil
}

func loadPages(fsys fs.FS, pagesPath string, layouts map[string]*template.Template) (map[string]*template.Template, error) {

	templates := make(map[string]*template.Template)
	pageCounts := make(map[string]int)

	err := fs.WalkDir(
		fsys,
		pagesPath,
		func(pagePath string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if entry.IsDir() || path.Ext(pagePath) != ".html" {
				return nil
			}

			relativePath, err := relativeFSPath(pagesPath, pagePath)
			if err != nil {
				return err
			}

			parts := strings.Split(relativePath, "/")
			if len(parts) < 2 {
				return fmt.Errorf(
					"page must be inside a layout directory: %q",
					pagePath,
				)
			}

			layoutName := parts[0]
			layoutTmpl, ok := layouts[layoutName]
			if !ok {
				return fmt.Errorf(
					"no layout found for page %q",
					pagePath,
				)
			}

			pageTmpl, err := layoutTmpl.Clone()
			if err != nil {
				return fmt.Errorf(
					"%w while cloning layout %q: %w",
					ErrTemplateParse,
					layoutName,
					err,
				)
			}

			if _, err := pageTmpl.ParseFS(fsys, pagePath); err != nil {
				return fmt.Errorf(
					"%w under %q: %w",
					ErrTemplateParse,
					pagePath,
					err,
				)
			}

			if pageTmpl.Lookup("content") == nil {
				return fmt.Errorf(
					"page %q does not define %q",
					pagePath,
					"content",
				)
			}

			pageName := strings.TrimSuffix(
				relativePath,
				path.Ext(relativePath),
			)

			if _, exists := templates[pageName]; exists {
				return fmt.Errorf(
					"duplicate page template %q",
					pageName,
				)
			}

			templates[pageName] = pageTmpl
			pageCounts[layoutName]++
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w under %q: %w",
			ErrTemplateWalk,
			pagesPath,
			err,
		)
	}

	for layoutName := range layouts {
		if pageCounts[layoutName] == 0 {
			return nil, fmt.Errorf(
				"no page templates found for layout %q",
				layoutName,
			)
		}
	}

	return templates, nil
}

func loadViews(fsys fs.FS, tmplPath string, funcs template.FuncMap) (map[string]*template.Template, error) {
	layoutsPath := path.Join(tmplPath, "layouts")
	pagesPath := path.Join(tmplPath, "pages")

	layoutEntries, err := validateLayoutsDir(fsys, layoutsPath)
	if err != nil {
		return nil, err
	}

	layouts, err := loadLayouts(fsys, layoutsPath, layoutEntries, funcs)
	if err != nil {
		return nil, err
	}

	if err := validatePagesDirs(fsys, pagesPath, layouts); err != nil {
		return nil, err
	}

	return loadPages(fsys, pagesPath, layouts)
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
