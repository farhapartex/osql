package vfs

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
)

const (
	Separator = "/"
	RootPath  = "."
)

var (
	ErrNotAbsolute = errors.New("path is not absolute")
	ErrInvalidPath = errors.New("path is not representable as an fs path")
)

func FSPath(absolute string) (string, error) {
	if absolute == "" {
		return "", ErrNotAbsolute
	}
	if !filepath.IsAbs(absolute) {
		return "", ErrNotAbsolute
	}

	trimmed := strings.TrimPrefix(filepath.Clean(absolute), Separator)
	if trimmed == "" {
		return RootPath, nil
	}
	if !fs.ValidPath(trimmed) {
		return "", ErrInvalidPath
	}
	return trimmed, nil
}

func OSPath(root, fsPath string) string {
	if fsPath == RootPath || fsPath == "" {
		return root
	}
	return filepath.Join(root, fsPath)
}

func Join(fsPath, name string) string {
	if fsPath == RootPath || fsPath == "" {
		return name
	}
	return fsPath + Separator + name
}
