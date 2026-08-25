package test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/farhapartex/osql/internal/reader"
)

type errAppender struct {
	err   error
	lines []string
}

func (e *errAppender) Append(line string) error {
	e.lines = append(e.lines, line)
	return e.err
}

type errWriter struct{}

func (errWriter) Write(p []byte) (int, error) { return 0, errors.New("writer closed") }

func TestBasicReadLineReadsSequentially(t *testing.T) {
	in := strings.NewReader("first\nsecond\nthird\n")
	out := &bytes.Buffer{}
	r := reader.NewBasic(in, out, nil)

	for _, want := range []string{"first", "second", "third"} {
		got, err := r.ReadLine("osql > ")
		if err != nil {
			t.Fatalf("ReadLine() error = %v", err)
		}
		if got != want {
			t.Errorf("ReadLine() = %q, want %q", got, want)
		}
	}

	if _, err := r.ReadLine("osql > "); !errors.Is(err, io.EOF) {
		t.Errorf("ReadLine() after input exhausted error = %v, want io.EOF", err)
	}
}

func TestBasicReadLineWritesPromptEachCall(t *testing.T) {
	in := strings.NewReader("a\nb\n")
	out := &bytes.Buffer{}
	r := reader.NewBasic(in, out, nil)

	r.ReadLine("osql > ")
	r.ReadLine("osql > ")

	if got, want := out.String(), "osql > osql > "; got != want {
		t.Errorf("prompt output = %q, want %q", got, want)
	}
}

func TestBasicReadLineStripsLineEndings(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"unix newline", "abc\n", "abc"},
		{"windows newline", "abc\r\n", "abc"},
		{"bare carriage return at end", "abc\r\n", "abc"},
		{"inner spaces preserved", "a b c\n", "a b c"},
		{"leading whitespace preserved", "   abc\n", "   abc"},
		{"tab preserved", "a\tb\n", "a\tb"},
		{"empty line", "\n", ""},
		{"unicode", "日本語\n", "日本語"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := reader.NewBasic(strings.NewReader(tt.input), &bytes.Buffer{}, nil)
			got, err := r.ReadLine("")
			if err != nil {
				t.Fatalf("ReadLine() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ReadLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBasicReadLineReturnsFinalLineWithoutNewline(t *testing.T) {
	r := reader.NewBasic(strings.NewReader("no trailing newline"), &bytes.Buffer{}, nil)

	got, err := r.ReadLine("")
	if err != nil {
		t.Fatalf("ReadLine() error = %v; unterminated final line must not be discarded", err)
	}
	if got != "no trailing newline" {
		t.Errorf("ReadLine() = %q, want %q", got, "no trailing newline")
	}

	if _, err := r.ReadLine(""); !errors.Is(err, io.EOF) {
		t.Errorf("second ReadLine() error = %v, want io.EOF", err)
	}
}

func TestBasicReadLineOnEmptyInput(t *testing.T) {
	r := reader.NewBasic(strings.NewReader(""), &bytes.Buffer{}, nil)

	if _, err := r.ReadLine("osql > "); !errors.Is(err, io.EOF) {
		t.Errorf("ReadLine() on empty input error = %v, want io.EOF", err)
	}
}

func TestBasicReadLineHandlesVeryLongLine(t *testing.T) {
	long := strings.Repeat("x", 500000)
	r := reader.NewBasic(strings.NewReader(long+"\n"), &bytes.Buffer{}, nil)

	got, err := r.ReadLine("")
	if err != nil {
		t.Fatalf("ReadLine() on a long line error = %v", err)
	}
	if got != long {
		t.Errorf("long line truncated: got %d bytes, want %d", len(got), len(long))
	}
}

func TestBasicReadLineSurfacesWriteError(t *testing.T) {
	r := reader.NewBasic(strings.NewReader("abc\n"), errWriter{}, nil)

	if _, err := r.ReadLine("osql > "); err == nil {
		t.Error("ReadLine() swallowed a prompt write failure")
	}
}

func TestBasicAddHistoryForwardsLines(t *testing.T) {
	hist := &errAppender{}
	r := reader.NewBasic(strings.NewReader(""), &bytes.Buffer{}, hist)

	r.AddHistory("files from '.'")
	r.AddHistory("exit")

	if len(hist.lines) != 2 {
		t.Fatalf("history received %d lines, want 2", len(hist.lines))
	}
	if hist.lines[0] != "files from '.'" || hist.lines[1] != "exit" {
		t.Errorf("history lines = %v", hist.lines)
	}
	if r.Err() != nil {
		t.Errorf("Err() = %v, want nil", r.Err())
	}
}

func TestBasicAddHistoryToleratesNilAppender(t *testing.T) {
	r := reader.NewBasic(strings.NewReader(""), &bytes.Buffer{}, nil)

	r.AddHistory("files from '.'")

	if r.Err() != nil {
		t.Errorf("Err() = %v, want nil with no history configured", r.Err())
	}
}

func TestBasicAddHistoryRecordsFailureWithoutPanicking(t *testing.T) {
	hist := &errAppender{err: errors.New("disk full")}
	r := reader.NewBasic(strings.NewReader(""), &bytes.Buffer{}, hist)

	r.AddHistory("files from '.'")

	if r.Err() == nil {
		t.Error("Err() = nil after a failed append; the failure must be observable")
	}
}

func TestBasicCloseIsSafe(t *testing.T) {
	r := reader.NewBasic(strings.NewReader(""), &bytes.Buffer{}, nil)

	if err := r.Close(); err != nil {
		t.Errorf("Close() = %v, want nil", err)
	}
	if err := r.Close(); err != nil {
		t.Errorf("second Close() = %v, want nil", err)
	}
}
