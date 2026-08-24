package vfs

import (
	"io/fs"
	"os"
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
