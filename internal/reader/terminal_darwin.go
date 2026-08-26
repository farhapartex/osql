//go:build darwin

package reader

import "syscall"

const (
	getAttr = syscall.TIOCGETA
	setAttr = syscall.TIOCSETA
)
