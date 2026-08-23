//go:build darwin

package state

import "syscall"

func kernelRelease() string {
	release, err := syscall.Sysctl("kern.osrelease")
	if err != nil {
		return "unknown"
	}
	return release
}
