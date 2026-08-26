//go:build linux

package vfs

import "path/filepath"

func trashPaths(home string) (files string, info string) {
	base := filepath.Join(home, ".local", "share", "Trash")
	return filepath.Join(base, "files"), filepath.Join(base, "info")
}
