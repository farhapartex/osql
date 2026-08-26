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
	fsys     vfs.FileSystem
	dir      string
	home     string
	previous string
}

func NewPathResolver(fsys vfs.FileSystem, dir string) *PathResolver {
	return NewPathResolverAt(fsys, dir, dir)
}

func NewPathResolverAt(fsys vfs.FileSystem, dir, home string) *PathResolver {
	if dir == "" {
		dir = vfs.Separator
	}
	if home == "" {
		home = dir
	}
	clean := filepath.Clean(dir)
	return &PathResolver{
		fsys:     fsys,
		dir:      clean,
		home:     filepath.Clean(home),
		previous: clean,
	}
}

func (r *PathResolver) Dir() string {
	return r.dir
}

func (r *PathResolver) Home() string {
	return r.home
}

func (r *PathResolver) Expand(input string) string {
	path := strings.TrimSpace(input)

	switch {
	case path == "":
		return r.dir
	case path == "~":
		return r.home
	case strings.HasPrefix(path, "~/"):
		return filepath.Join(r.home, path[2:])
	case filepath.IsAbs(path):
		return filepath.Clean(path)
	default:
		return filepath.Join(r.dir, path)
	}
}

func (r *PathResolver) Chdir(input string) (string, error) {
	target := input
	if strings.TrimSpace(input) == "" {
		target = "~"
	}
	if strings.TrimSpace(input) == "-" {
		target = r.previous
	}

	resolved, err := r.Resolve(target)
	if err != nil {
		if oerr.Is(err, oerr.KindPathIsFile) {
			return "", oerr.CannotChangeDir(target, "that is a file, not a folder")
		}
		return "", err
	}

	r.previous = r.dir
	r.dir = resolved.Absolute
	return r.dir, nil
}

func (r *PathResolver) Display(path string) string {
	if path == r.home {
		return "~"
	}
	if r.home != vfs.Separator && strings.HasPrefix(path, r.home+vfs.Separator) {
		return "~" + path[len(r.home):]
	}
	return path
}

func (r *PathResolver) Resolve(input string) (Resolved, error) {
	absolute := r.Expand(input)

	fsPath, err := vfs.FSPath(absolute)
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

func (r *PathResolver) ResolveFile(input string) (Resolved, error) {
	absolute := r.Expand(input)

	fsPath, err := vfs.FSPath(absolute)
	if err != nil {
		return Resolved{}, oerr.FileMissing(input)
	}

	info, err := r.fsys.Stat(fsPath)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return Resolved{}, oerr.NoPermission(input)
		}
		return Resolved{}, oerr.FileMissing(input)
	}
	if info.IsDir() {
		return Resolved{}, oerr.PathIsFolder(input)
	}

	return Resolved{Input: input, Absolute: absolute, FSPath: fsPath}, nil
}

func (r *PathResolver) ResolveNew(input string) (Resolved, error) {
	absolute := r.Expand(input)

	fsPath, err := vfs.FSPath(absolute)
	if err != nil {
		return Resolved{}, oerr.CannotCreate(input, "that path is not usable")
	}
	if absolute == vfs.Separator {
		return Resolved{}, oerr.AlreadyExists(input)
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
