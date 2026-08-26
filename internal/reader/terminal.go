package reader

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

var ErrNotATerminal = errors.New("not a terminal")

type terminal struct {
	fd    uintptr
	saved syscall.Termios
}

func openTerminal(file *os.File) (*terminal, error) {
	if file == nil {
		return nil, ErrNotATerminal
	}

	fd := file.Fd()
	saved, err := readAttr(fd)
	if err != nil {
		return nil, ErrNotATerminal
	}
	return &terminal{fd: fd, saved: saved}, nil
}

func (t *terminal) enterRaw() error {
	raw := t.saved
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	raw.Iflag &^= syscall.IXON | syscall.ICRNL
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	return writeAttr(t.fd, raw)
}

func (t *terminal) restore() error {
	return writeAttr(t.fd, t.saved)
}

func readAttr(fd uintptr) (syscall.Termios, error) {
	var attr syscall.Termios
	if err := ioctl(fd, getAttr, &attr); err != nil {
		return attr, err
	}
	return attr, nil
}

func writeAttr(fd uintptr, attr syscall.Termios) error {
	return ioctl(fd, setAttr, &attr)
}

func ioctl(fd uintptr, request uintptr, attr *syscall.Termios) error {
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, request,
		uintptr(unsafe.Pointer(attr)), 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}
