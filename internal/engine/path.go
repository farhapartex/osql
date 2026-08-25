package engine

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/vfs"
)

type Resolved struct {
	Input    string
	Absolute string
	FSPath   string
}

type PathResolver struct {
	fsys vfs.FileSystem
	root string
	home string
	cwd  string
}

func NewPathResolver(fsys vfs.FileSystem, root, home, cwd string) *PathResolver {
	if root == "" {
		root = vfs.Separator
	}
	return &PathResolver{fsys: fsys, root: root, home: home, cwd: cwd}
}

func (r *PathResolver) Expand(input string) string {
	path := strings.TrimSpace(input)
	if path == "" {
		path = "."
	}

	switch {
	case path == "~":
		path = r.home
	case strings.HasPrefix(path, "~/"):
		path = filepath.Join(r.home, path[2:])
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(r.cwd, path)
	}
	return filepath.Clean(path)
}

func (r *PathResolver) Resolve(input string) (Resolved, error) {
	absolute := r.Expand(input)

	fsPath, err := vfs.FSPathUnder(r.root, absolute)
	if err != nil {
		return Resolved{}, oerr.FolderMissing(input)
	}

	info, err := r.fsys.Stat(fsPath)
	if err != nil {
		return Resolved{}, classifyStatError(input, err)
	}
	if !info.IsDir() {
		return Resolved{}, oerr.PathIsFile(input)
	}

	return Resolved{Input: input, Absolute: absolute, FSPath: fsPath}, nil
}

func classifyStatError(input string, err error) error {
	switch {
	case errors.Is(err, fs.ErrPermission):
		return oerr.NoPermission(input)
	case errors.Is(err, fs.ErrNotExist):
		return oerr.FolderMissing(input)
	default:
		return oerr.FolderMissing(input)
	}
}
