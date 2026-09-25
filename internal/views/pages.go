package views

import (
	"fmt"
	"html/template"
	"io/fs"
	"path"
	"strings"
)

func validatePagesDirs(fsys fs.FS, pagesPath string, layouts map[string]*template.Template) error {
	entries, err := fs.ReadDir(fsys, pagesPath)
	if err != nil {
		return fmt.Errorf("pages directory %q: failed to read: %w", pagesPath, err)
	}

	for layoutName := range layouts {
		layoutPagesPath := path.Join(pagesPath, layoutName)
		if _, err := fs.ReadDir(fsys, layoutPagesPath); err != nil {
			return fmt.Errorf("failed to read pages directory for layout %q: %w", layoutName, err)
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

func loadPages(fsys fs.FS, pagesPath string, layoutName string, layoutTmpl *template.Template) (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	err := fs.WalkDir(fsys, pagesPath, func(pagePath string, entry fs.DirEntry, err error) error {
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

		pageTmpl, err := layoutTmpl.Clone()
		if err != nil {
			return fmt.Errorf("%w while cloning layout %q: %w", ErrTemplateParse, layoutName, err)
		}

		if _, err := pageTmpl.ParseFS(fsys, pagePath); err != nil {
			return fmt.Errorf("%w under %q: %w", ErrTemplateParse, pagePath, err)
		}

		if pageTmpl.Lookup("content") == nil {
			return fmt.Errorf("page %q does not define %q", pagePath, "content")
		}

		pageName := path.Join(layoutName, strings.TrimSuffix(relativePath, path.Ext(relativePath)))

		templates[pageName] = pageTmpl
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w under %q: %w", ErrTemplateWalk, pagesPath, err)
	}

	if len(templates) == 0 {
		return nil, fmt.Errorf("no page templates found for layout %q", layoutName)
	}

	return templates, nil
}
