package reader

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

type HistoryAppender interface {
	Append(line string) error
}

type Basic struct {
	in      *bufio.Reader
	out     io.Writer
	history HistoryAppender
	lastErr error
}

func NewBasic(in io.Reader, out io.Writer, history HistoryAppender) *Basic {
	return &Basic{
		in:      bufio.NewReader(in),
		out:     out,
		history: history,
	}
}

func (b *Basic) ReadLine(prompt string) (string, error) {
	if _, err := io.WriteString(b.out, prompt); err != nil {
		return "", err
	}

	line, err := b.in.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) && line != "" {
			return trimLineEnding(line), nil
		}
		return "", err
	}
	return trimLineEnding(line), nil
}

func (b *Basic) AddHistory(line string) {
	if b.history == nil {
		return
	}
	if err := b.history.Append(line); err != nil {
		b.lastErr = err
	}
}

func (b *Basic) Err() error {
	return b.lastErr
}

func (b *Basic) Close() error {
	return nil
}

func trimLineEnding(line string) string {
	return strings.TrimRight(line, "\r\n")
}
