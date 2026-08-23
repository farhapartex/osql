//go:build linux

package state

import "syscall"

func kernelRelease() string {
	var uts syscall.Utsname
	if err := syscall.Uname(&uts); err != nil {
		return "unknown"
	}

	buf := make([]byte, 0, len(uts.Release))
	for _, c := range uts.Release {
		if c == 0 {
			break
		}
		buf = append(buf, byte(c))
	}
	if len(buf) == 0 {
		return "unknown"
	}
	return string(buf)
}
