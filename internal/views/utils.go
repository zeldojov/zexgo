package views

import (
	"fmt"
	"strings"
)

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
