package reader

import (
	"io"
	"os"
)

type LineReader interface {
	ReadLine(prompt string) (string, error)
	AddHistory(line string)
	Close() error
}

func New(file *os.File, out io.Writer, history HistoryStore) (LineReader, bool) {
	if raw, err := NewRaw(file, out, history); err == nil {
		return raw, true
	}
	return NewBasic(file, out, history), false
}
