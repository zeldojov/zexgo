package app

import (
	"fmt"
	"io/fs"
)

func VvalidateDirectoryPath(fsys fs.FS, dirPath string, invalidPathErr error) error {
	if dirPath == "" {
		return fmt.Errorf("%w: path is empty", invalidPathErr)
	}

	if !fs.ValidPath(dirPath) {
		return fmt.Errorf("%w %q: invalid format", invalidPathErr, dirPath)
	}

	info, err := fs.Stat(fsys, dirPath)
	if err != nil {
		return fmt.Errorf("%w %q: %w", invalidPathErr, dirPath, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%w %q: path is not a directory", invalidPathErr, dirPath)
	}

	return nil
}
