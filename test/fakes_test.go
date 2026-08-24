package test

import (
	"context"
	"io"
	"io/fs"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/output"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/reader"
	"github.com/farhapartex/osql/internal/state"
	"github.com/farhapartex/osql/internal/vfs"
)

type fakeReader struct {
	lines   []string
	pos     int
	history []string
	closed  bool
}

func (f *fakeReader) ReadLine(prompt string) (string, error) {
	if f.pos >= len(f.lines) {
		return "", io.EOF
	}
	line := f.lines[f.pos]
	f.pos++
	return line, nil
}

func (f *fakeReader) AddHistory(line string) { f.history = append(f.history, line) }
func (f *fakeReader) Close() error           { f.closed = true; return nil }

type fakeLexer struct {
	tokens []query.Token
	err    error
}

func (f *fakeLexer) Lex(input string) ([]query.Token, error) { return f.tokens, f.err }

type fakeParser struct {
	stmt *query.Statement
	err  error
}

func (f *fakeParser) Parse(tokens []query.Token) (*query.Statement, error) {
	return f.stmt, f.err
}

type fakeExecutor struct {
	verb  string
	rows  []engine.Row
	err   error
	calls int
}

func (f *fakeExecutor) Verb() string { return f.verb }

func (f *fakeExecutor) Execute(ctx context.Context, stmt *query.Statement, out engine.RowSink) error {
	f.calls++
	if f.err != nil {
		return f.err
	}
	for _, r := range f.rows {
		if err := out.Push(r); err != nil {
			return err
		}
	}
	return nil
}

type sliceSink struct {
	rows  []engine.Row
	limit int
}

func (s *sliceSink) Push(r engine.Row) error {
	if s.limit > 0 && len(s.rows) >= s.limit {
		return engine.ErrStopWalk
	}
	s.rows = append(s.rows, r)
	return nil
}

type fakeFieldExtractor struct {
	field       string
	cost        int
	allowedOps  []string
	foldersOnly bool
	value       engine.Value
	err         error
}

func (f *fakeFieldExtractor) Field() string               { return f.field }
func (f *fakeFieldExtractor) Cost() int                   { return f.cost }
func (f *fakeFieldExtractor) AllowedOperators() []string  { return f.allowedOps }
func (f *fakeFieldExtractor) AppliesTo(query.Target) bool { return !f.foldersOnly }

func (f *fakeFieldExtractor) NormalizeValue(v string) (engine.Value, error) {
	return engine.Value{Text: v}, nil
}

func (f *fakeFieldExtractor) Extract(e engine.Entry) (engine.Value, error) {
	return f.value, f.err
}

type fakeComparator struct {
	op     string
	result bool
}

func (f *fakeComparator) Op() string                          { return f.op }
func (f *fakeComparator) Compare(got, want engine.Value) bool { return f.result }

type fakeRenderer struct {
	rendered []engine.Row
	err      error
}

func (f *fakeRenderer) Render(w io.Writer, rows []engine.Row) error {
	f.rendered = rows
	return f.err
}

type fakeFileSystem struct {
	fsys fs.FS
}

func (f *fakeFileSystem) Open(name string) (fs.File, error) { return f.fsys.Open(name) }

func (f *fakeFileSystem) Stat(name string) (fs.FileInfo, error) { return fs.Stat(f.fsys, name) }

func (f *fakeFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(f.fsys, name)
}

type fakeHistory struct {
	lines  []string
	closed bool
}

func (f *fakeHistory) Append(line string) error { f.lines = append(f.lines, line); return nil }

func (f *fakeHistory) Lines(limit int) ([]string, error) {
	if limit <= 0 || limit > len(f.lines) {
		return f.lines, nil
	}
	return f.lines[len(f.lines)-limit:], nil
}

func (f *fakeHistory) Clear() error { f.lines = nil; return nil }
func (f *fakeHistory) Close() error { f.closed = true; return nil }

type fakeStore struct {
	ensured int
	written int
	hist    *fakeHistory
}

func (f *fakeStore) Ensure() error { f.ensured++; return nil }

func (f *fakeStore) WriteSystemInfo(force bool) error { f.written++; return nil }

func (f *fakeStore) History() (state.History, error) {
	if f.hist == nil {
		f.hist = &fakeHistory{}
	}
	return f.hist, nil
}

var (
	_ reader.LineReader     = (*fakeReader)(nil)
	_ query.Lexer           = (*fakeLexer)(nil)
	_ query.Parser          = (*fakeParser)(nil)
	_ engine.Executor       = (*fakeExecutor)(nil)
	_ engine.RowSink        = (*sliceSink)(nil)
	_ engine.FieldExtractor = (*fakeFieldExtractor)(nil)
	_ engine.Comparator     = (*fakeComparator)(nil)
	_ output.Renderer       = (*fakeRenderer)(nil)
	_ vfs.FileSystem        = (*fakeFileSystem)(nil)
	_ state.Store           = (*fakeStore)(nil)
	_ state.History         = (*fakeHistory)(nil)
)

type failingReader struct {
	err error
}

func (f *failingReader) ReadLine(prompt string) (string, error) { return "", f.err }
func (f *failingReader) AddHistory(line string)                 {}
func (f *failingReader) Close() error                           { return nil }
