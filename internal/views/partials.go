package views

import (
	"fmt"
	"html/template"
	"io/fs"
	"path"
)

func validatePartialsDirs(fsys fs.FS, partialsPath string, layouts map[string]*template.Template) error {
	entries, err := fs.ReadDir(fsys, partialsPath)
	if err != nil {
		return fmt.Errorf("failed to read %q: %w", partialsPath, err)
	}

	for layoutName := range layouts {
		layoutPartialsPath := path.Join(partialsPath, layoutName)
		if _, err := fs.ReadDir(fsys, layoutPartialsPath); err != nil {
			return fmt.Errorf("failed to read partials directory for layout %q: %w", layoutName, err)
		}
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			return fmt.Errorf("unexpected file in partials directory: %q", entry.Name())
		}

		if _, ok := layouts[entry.Name()]; !ok {
			return fmt.Errorf("no layout found for partials directory %q", entry.Name())
		}
	}

	return nil
}

func loadPartials(fsys fs.FS, partialsPath string) ([]string, error) {
	partialFiles := make([]string, 0)

	err := fs.WalkDir(fsys, partialsPath, func(partialPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() && path.Ext(partialPath) == ".html" {
			partialFiles = append(partialFiles, partialPath)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w under %q: %w", ErrTemplateWalk, partialsPath, err)
	}

	if len(partialFiles) == 0 {
		return nil, fmt.Errorf("no partial templates found under %q", partialsPath)
	}

	return partialFiles, nil
}
