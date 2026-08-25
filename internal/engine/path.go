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
}

func NewPathResolver(fsys vfs.FileSystem, root string) *PathResolver {
	if root == "" {
		root = vfs.Separator
	}
	return &PathResolver{fsys: fsys, root: filepath.Clean(root)}
}

func (r *PathResolver) Root() string {
	return r.root
}

func (r *PathResolver) Expand(input string) string {
	path := strings.TrimSpace(input)

	switch {
	case path == "" || path == "." || path == "~" || path == vfs.Separator:
		return r.root
	case strings.HasPrefix(path, "~/"):
		path = path[2:]
	case strings.HasPrefix(path, vfs.Separator):
		path = path[1:]
	}

	return filepath.Join(r.root, path)
}

func (r *PathResolver) Resolve(input string) (Resolved, error) {
	absolute := r.Expand(input)

	fsPath, err := vfs.FSPathUnder(r.root, absolute)
	if err != nil {
		if errors.Is(err, vfs.ErrOutsideRoot) {
			return Resolved{}, oerr.OutsideRoot(input, r.root)
		}
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
	default:
		return oerr.FolderMissing(input)
	}
}
