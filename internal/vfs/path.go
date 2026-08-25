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
	ErrOutsideRoot = errors.New("path is outside the filesystem root")
)

func FSPath(absolute string) (string, error) {
	return FSPathUnder(Separator, absolute)
}

func FSPathUnder(root, absolute string) (string, error) {
	if root == "" || absolute == "" {
		return "", ErrNotAbsolute
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(absolute) {
		return "", ErrNotAbsolute
	}

	root = filepath.Clean(root)
	absolute = filepath.Clean(absolute)
	if absolute == root {
		return RootPath, nil
	}

	prefix := root
	if prefix != Separator {
		prefix += Separator
	}
	if !strings.HasPrefix(absolute, prefix) {
		return "", ErrOutsideRoot
	}

	relative := absolute[len(prefix):]
	if !fs.ValidPath(relative) {
		return "", ErrInvalidPath
	}
	return relative, nil
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
