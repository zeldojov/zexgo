package app

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
)

var (
	ErrInvalidStaticPath = errors.New("invalid static path")
)

type filesOnlyFS struct {
	fsys fs.FS
}

func (f filesOnlyFS) Open(name string) (fs.File, error) {
	if name == "." || !fs.ValidPath(name) {
		return nil, fs.ErrNotExist
	}

	file, err := f.fsys.Open(name)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	if info.IsDir() {
		file.Close()
		return nil, fs.ErrNotExist
	}

	return file, nil
}

func validateStaticPath(fsys fs.FS, staticPath string) error {
	return VvalidateDirectoryPath(fsys, staticPath, ErrInvalidStaticPath)
}

func (a *App) InitStatic(fsys fs.FS) error {
	if err := validateStaticPath(fsys, a.Config.StaticPath); err != nil {
		return err
	}

	staticFiles, err := fs.Sub(fsys, a.Config.StaticPath)
	if err != nil {
		return fmt.Errorf("application failed to initialize static files: %w", err)
	}

	a.static = filesOnlyFS{fsys: staticFiles}
	return nil
}

func (a *App) StaticHandler() http.Handler {
	return http.StripPrefix("/static/",
		http.FileServer(http.FS(a.static)),
	)
}
