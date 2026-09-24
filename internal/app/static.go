package app

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"

	staticfspkg "github.com/zeldojov/zexgo/internal/staticfs"
)

var (
	ErrInvalidStaticPath = errors.New("invalid static path")
	ErrInitStaticFS      = errors.New("failed to initialize static FS")
)

// region FS

// endregion FS

func (a *App) InitStatic(fsys fs.FS) error {
	if err := ValidateDirectoryPath(fsys, a.Config.StaticPath); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidStaticPath, err)
	}

	staticFiles, err := staticfspkg.New(fsys, a.Config.StaticPath)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInitStaticFS, err)
	}

	a.static = staticFiles
	return nil
}

func (a *App) StaticHandler() http.Handler {
	return http.StripPrefix("/static/",
		http.FileServer(http.FS(a.static)),
	)
}
