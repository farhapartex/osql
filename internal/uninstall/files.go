package uninstall

import "io/fs"

type Files interface {
	Stat(path string) (fs.FileInfo, error)
	DirectorySize(path string) (int64, error)
	CanWriteInto(directory string) bool
	Remove(path string) error
	RemoveTree(path string) error
}
