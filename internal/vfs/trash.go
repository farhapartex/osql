package vfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

var ErrCrossDevice = errors.New("path is on another filesystem")

type Trash struct {
	files string
	info  string
	now   func() time.Time
}

func NewTrash(home string, now func() time.Time) *Trash {
	if now == nil {
		now = time.Now
	}
	files, info := trashPaths(home)
	return &Trash{files: files, info: info, now: now}
}

func (t *Trash) FilesDir() string {
	return t.files
}

func (t *Trash) Ensure() error {
	if err := os.MkdirAll(t.files, newDirMode); err != nil {
		return err
	}
	if t.info == "" {
		return nil
	}
	return os.MkdirAll(t.info, newDirMode)
}

func (t *Trash) Move(absolute string) (string, error) {
	if err := t.Ensure(); err != nil {
		return "", err
	}

	target, err := t.freeName(filepath.Base(absolute))
	if err != nil {
		return "", err
	}

	if err := os.Rename(absolute, target); err != nil {
		if isCrossDevice(err) {
			return "", ErrCrossDevice
		}
		return "", err
	}

	if err := t.writeInfo(filepath.Base(target), absolute); err != nil {
		return target, nil
	}
	return target, nil
}

func (t *Trash) freeName(name string) (string, error) {
	candidate := filepath.Join(t.files, name)
	if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
		return candidate, nil
	}

	ext := filepath.Ext(name)
	stem := name[:len(name)-len(ext)]

	for n := 2; n < 10000; n++ {
		candidate = filepath.Join(t.files, stem+" "+strconv.Itoa(n)+ext)
		if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("the trash already holds too many copies of %q", name)
}

func (t *Trash) writeInfo(trashedName, original string) error {
	if t.info == "" {
		return nil
	}

	body := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n",
		original, t.now().Format("2006-01-02T15:04:05"))

	return os.WriteFile(filepath.Join(t.info, trashedName+".trashinfo"), []byte(body), newFileMode)
}

func isCrossDevice(err error) bool {
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		return errors.Is(linkErr.Err, syscall.EXDEV)
	}
	return errors.Is(err, syscall.EXDEV)
}
