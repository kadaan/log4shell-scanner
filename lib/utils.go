package lib

import (
	"github.com/mitchellh/go-homedir"
	"path/filepath"
)

func AbsolutePath(path string) (string, error) {
	expandedPath, err := homedir.Expand(path)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return "", err
	}
	return absPath, nil
}
