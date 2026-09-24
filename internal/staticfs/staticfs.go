package staticfs

import (
	"fmt"
	"io/fs"
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

func New(fsys fs.FS, staticPath string) (fs.FS, error) {
	staticFiles, err := fs.Sub(fsys, staticPath)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to initialize static files from %q: %w",
			staticPath,
			err,
		)
	}

	return filesOnlyFS{fsys: staticFiles}, nil
}
