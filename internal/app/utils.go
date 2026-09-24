package app

import (
	"fmt"
	"io/fs"
)

func ValidateDirectoryPath(fsys fs.FS, dirPath string) error {
	if dirPath == "" {
		return fmt.Errorf("path is empty")
	}

	if !fs.ValidPath(dirPath) {
		return fmt.Errorf("%q: invalid format", dirPath)
	}

	info, err := fs.Stat(fsys, dirPath)
	if err != nil {
		return fmt.Errorf("%q: %w", dirPath, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%q: path is not a directory", dirPath)
	}

	return nil
}
