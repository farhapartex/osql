package state

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const DefaultHistoryLimit = 10000

type fileHistory struct {
	path  string
	limit int

	mu   sync.Mutex
	file *os.File
}

func openHistory(path string, limit int) (*fileHistory, error) {
	if limit <= 0 {
		limit = DefaultHistoryLimit
	}

	h := &fileHistory{path: path, limit: limit}
	if err := h.trim(); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, historyMode)
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(historyMode); err != nil {
		file.Close()
		return nil, err
	}

	h.file = file
	return h, nil
}

func (h *fileHistory) Append(line string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.file == nil {
		return os.ErrClosed
	}
	_, err := h.file.WriteString(line + "\n")
	return err
}

func (h *fileHistory) Lines(limit int) ([]string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	lines, err := readLines(h.path)
	if err != nil {
		return nil, err
	}
	if limit > 0 && limit < len(lines) {
		lines = lines[len(lines)-limit:]
	}
	return lines, nil
}

func (h *fileHistory) Clear() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.file != nil {
		if err := h.file.Close(); err != nil {
			return err
		}
		h.file = nil
	}
	if err := os.WriteFile(h.path, nil, historyMode); err != nil {
		return err
	}

	file, err := os.OpenFile(h.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, historyMode)
	if err != nil {
		return err
	}
	h.file = file
	return nil
}

func (h *fileHistory) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.file == nil {
		return nil
	}
	err := h.file.Close()
	h.file = nil
	return err
}

func (h *fileHistory) trim() error {
	lines, err := readLines(h.path)
	if err != nil {
		return err
	}
	if len(lines) <= h.limit {
		return nil
	}

	kept := lines[len(lines)-h.limit:]
	return writeLinesAtomic(h.path, kept)
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func writeLinesAtomic(path string, lines []string) error {
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".history-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.WriteString(b.String()); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(historyMode); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, path)
}

type nopHistory struct{}

func (nopHistory) Append(string) error         { return nil }
func (nopHistory) Lines(int) ([]string, error) { return nil, nil }
func (nopHistory) Clear() error                { return nil }
func (nopHistory) Close() error                { return nil }
