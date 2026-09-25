package views

import (
	"fmt"
	"html/template"
	"io/fs"
	"path"
	"strings"
)

func loadLayouts(fsys fs.FS, layoutsPath, partialsPath string, funcs template.FuncMap) (map[string]*template.Template, error) {

	entries, err := fs.ReadDir(fsys, layoutsPath)
	if err != nil {
		return nil, fmt.Errorf("read layouts directory %q: %w", layoutsPath, err)
	}

	layouts := make(map[string]*template.Template)

	for _, entry := range entries {
		if entry.IsDir() {
			return nil, fmt.Errorf("layouts directory must be flat: %q", path.Join(layoutsPath, entry.Name()))
		}

		if path.Ext(entry.Name()) != ".html" {
			continue
		}

		layoutName := strings.TrimSuffix(entry.Name(), path.Ext(entry.Name()))
		layoutPath := path.Join(layoutsPath, entry.Name())
		layoutPartialsPath := path.Join(partialsPath, layoutName)

		partialFiles, err := loadPartials(fsys, layoutPartialsPath)
		if err != nil {
			return nil, err
		}

		files := []string{layoutPath}
		files = append(files, partialFiles...)

		layoutTmpl, err := template.New("layout").Funcs(funcs).ParseFS(fsys, files...)
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
