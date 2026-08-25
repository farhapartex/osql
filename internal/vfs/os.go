package vfs

import (
	"io"
	"io/fs"
	"os"
)

const (
	newFileMode os.FileMode = 0o644
	newDirMode  os.FileMode = 0o755
)

type OSFileSystem struct {
	root string
	fsys fs.FS
}

func OS() *OSFileSystem {
	return NewOS(Separator)
}

func NewOS(root string) *OSFileSystem {
	return &OSFileSystem{root: root, fsys: os.DirFS(root)}
}

func (f *OSFileSystem) Root() string {
	return f.root
}

func (f *OSFileSystem) Open(name string) (fs.File, error) {
	return f.fsys.Open(name)
}

func (f *OSFileSystem) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(f.fsys, name)
}

func (f *OSFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(f.fsys, name)
}

func (f *OSFileSystem) OSPath(fsPath string) string {
	return OSPath(f.root, fsPath)
}

func (f *OSFileSystem) Create(name string) (io.WriteCloser, error) {
	if !fs.ValidPath(name) {
		return nil, ErrInvalidPath
	}
	return os.OpenFile(f.OSPath(name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, newFileMode)
}

func (f *OSFileSystem) MkdirAll(name string) error {
	if !fs.ValidPath(name) {
		return ErrInvalidPath
	}
	return os.MkdirAll(f.OSPath(name), newDirMode)
}
