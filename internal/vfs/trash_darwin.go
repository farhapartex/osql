//go:build darwin

package vfs

import "path/filepath"

func trashPaths(home string) (files string, info string) {
	return filepath.Join(home, ".Trash"), ""
}
