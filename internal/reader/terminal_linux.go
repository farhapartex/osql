//go:build linux

package reader

import "syscall"

const (
	getAttr = syscall.TCGETS
	setAttr = syscall.TCSETS
)
