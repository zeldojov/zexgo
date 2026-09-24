package app

import (
	"fmt"
	"io/fs"

	staticfspkg "github.com/zeldojov/zexgo/internal/staticfs"
)

// region FS

// endregion FS

func validateStaticPath(fsys fs.FS, staticPath string) error {
	if err := ValidateDirectoryPath(fsys, staticPath); err != nil {
		return fmt.Errorf("%q: %w", "invalid static path", err)
	}
	return nil
}

func (a *App) InitStatic(fsys fs.FS) error {

	if a.views == nil {
		return fmt.Errorf("%q", "views not initialized")
	}

	if err := validateStaticPath(fsys, staticPath); err != nil {
		return fmt.Errorf("%q: %w", "failed to initialize static FS", err)
	}

	staticFiles, err := staticfspkg.New(fsys, staticPath)
	if err != nil {
		return fmt.Errorf("%q: %w", "failed to initialize static FS", err)
	}

	a.static = staticFiles
	return nil
}
