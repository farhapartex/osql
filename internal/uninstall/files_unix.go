//go:build darwin || linux

package uninstall

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

const writePermission uint32 = 0o2

type systemFiles struct{}

func SystemFiles() Files {
	return systemFiles{}
}

func (systemFiles) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

func (systemFiles) DirectorySize(path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func (systemFiles) CanWriteInto(directory string) bool {
	return syscall.Access(directory, writePermission) == nil
}

func (systemFiles) Remove(path string) error {
	return os.Remove(path)
}

func (systemFiles) RemoveTree(path string) error {
	return os.RemoveAll(path)
}
