package vfs

import (
	"io"
	"io/fs"
)

type FileSystem interface {
	fs.FS
	fs.StatFS
	fs.ReadDirFS
}

type Remover interface {
	Remove(name string) error
}

type Creator interface {
	Create(name string) (io.WriteCloser, error)
}
