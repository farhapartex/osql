package test

import (
	"context"
	"io/fs"
	"slices"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
)

type countingFS struct {
	inner    fs.FS
	readDirs atomic.Int64
	stats    atomic.Int64
	infos    atomic.Int64
}

func newCountingFS(inner fs.FS) *countingFS {
	return &countingFS{inner: inner}
}

func (c *countingFS) Open(name string) (fs.File, error) {
	return c.inner.Open(name)
}

func (c *countingFS) Stat(name string) (fs.FileInfo, error) {
	c.stats.Add(1)
	return fs.Stat(c.inner, name)
}

func (c *countingFS) ReadDir(name string) ([]fs.DirEntry, error) {
	c.readDirs.Add(1)
	entries, err := fs.ReadDir(c.inner, name)
	if err != nil {
		return nil, err
	}
	wrapped := make([]fs.DirEntry, len(entries))
	for i, e := range entries {
		wrapped[i] = countingEntry{DirEntry: e, infos: &c.infos}
	}
	return wrapped, nil
}

type countingEntry struct {
	fs.DirEntry
	infos *atomic.Int64
}

func (c countingEntry) Info() (fs.FileInfo, error) {
	c.infos.Add(1)
	return c.DirEntry.Info()
}

func executorFor(t *testing.T, fsys fs.FS) (*engine.SelectExecutor, *engine.Compiler) {
	t.Helper()

	vf := &fakeFileSystem{fsys: fsys}
	compiler := engine.NewCompiler(engine.DefaultFields(vf), engine.DefaultOperators())
	resolver := engine.NewPathResolver(vf, "/")
	return engine.NewSelectExecutor(vf, resolver, compiler, engine.EmptySkipList()), compiler
}

func runSelect(t *testing.T, fsys fs.FS, input string) []engine.Row {
	t.Helper()

	exec, compiler := executorFor(t, fsys)

	tokens, err := query.NewLexer().Lex(input)
	if err != nil {
		t.Fatalf("Lex error = %v", err)
	}
	stmt, err := query.NewParser(compiler).Parse(tokens)
	if err != nil {
		t.Fatalf("Parse(%q) error = %v", input, err)
	}

	sink := &engine.SliceSink{}
	if err := exec.Execute(context.Background(), stmt, sink); err != nil {
		t.Fatalf("Execute(%q) error = %v", input, err)
	}
	return sink.Rows
}

func rowNames(rows []engine.Row) []string {
	names := make([]string, 0, len(rows))
	for _, r := range rows {
		names = append(names, r.Name)
	}
	slices.Sort(names)
	return names
}

func TestSelectExecutorVerb(t *testing.T) {
	exec, _ := executorFor(t, fstest.MapFS{})

	if exec.Verb() != "select" {
		t.Errorf("Verb() = %q, want \"select\"", exec.Verb())
	}
}

func TestSelectExecutorRegistersByVerb(t *testing.T) {
	exec, _ := executorFor(t, fstest.MapFS{})
	registry := engine.NewRegistry(exec)

	got, ok := registry.Lookup("select")
	if !ok {
		t.Fatal("select executor not registered under its verb")
	}
	if got != engine.Executor(exec) {
		t.Error("registry returned a different executor")
	}
}

func TestSelectSpecExamplesAgainstAFixture(t *testing.T) {
	fsys := fstest.MapFS{
		"root/notes.txt":          {Data: []byte("a")},
		"root/report.pdf":         {Data: []byte("bb")},
		"root/q4-report.txt":      {Data: []byte("ccc")},
		"root/Makefile":           {Data: []byte("d")},
		"root/sub/deep.txt":       {Data: []byte("e")},
		"root/sub/nested/far.txt": {Data: []byte("f")},
		"root/one/only.txt":       {Data: []byte("g")},
		"root/three/a.txt":        {Data: []byte("h")},
		"root/three/b.txt":        {Data: []byte("i")},
		"root/three/c.txt":        {Data: []byte("j")},
	}

	tests := []struct {
		input string
		want  []string
	}{
		{"select all from 'root'", []string{"Makefile", "notes.txt", "one", "q4-report.txt", "report.pdf", "sub", "three"}},
		{"select files from 'root'", []string{"Makefile", "notes.txt", "q4-report.txt", "report.pdf"}},
		{"select folders from 'root'", []string{"one", "sub", "three"}},
		{"select files from 'root' where type = 'txt'", []string{"notes.txt", "q4-report.txt"}},
		{"select files from 'root' where type = '.txt'", []string{"notes.txt", "q4-report.txt"}},
		{"select files from 'root' where name = 'notes.txt'", []string{"notes.txt"}},
		{"select files from 'root' where name != 'notes.txt'", []string{"Makefile", "q4-report.txt", "report.pdf"}},
		{"select files from 'root' where name_like = '%report%'", []string{"q4-report.txt", "report.pdf"}},
		{"select files from 'root' where name_like = 'q4%'", []string{"q4-report.txt"}},
		{"select files from 'root' where name_like = '%.pdf'", []string{"report.pdf"}},
		{"select files from 'root' where name_like = '%report%' and type = 'txt'", []string{"q4-report.txt"}},
		{"select folders from 'root' where count(child) > 2", []string{"three"}},
		{"select folders from 'root' where count(child) = 1", []string{"one"}},
		{"select folders from 'root' where count(child) <= 1", []string{"one"}},
		{"select folders from 'root' where count(child) = 2", []string{"sub"}},
		{"select files from 'root' recursive where type = 'txt'", []string{"notes.txt", "one/only.txt", "q4-report.txt", "sub/deep.txt", "sub/nested/far.txt", "three/a.txt", "three/b.txt", "three/c.txt"}},
		{"select files from 'root' recursive where name = 'far.txt'", []string{"sub/nested/far.txt"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := rowNames(runSelect(t, fsys, tt.input))
			want := slices.Clone(tt.want)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Errorf("%q\n got: %v\nwant: %v", tt.input, got, want)
			}
		})
	}
}

func TestSelectRecursiveIsOptIn(t *testing.T) {
	fsys := fstest.MapFS{
		"root/top.txt":       {Data: []byte("a")},
		"root/sub/below.txt": {Data: []byte("b")},
	}

	shallow := rowNames(runSelect(t, fsys, "select files from 'root'"))
	if slices.Contains(shallow, "sub/below.txt") {
		t.Error("a non-recursive select descended into a subdirectory")
	}

	deep := rowNames(runSelect(t, fsys, "select files from 'root' recursive"))
	if !slices.Contains(deep, "sub/below.txt") {
		t.Error("recursive select did not descend")
	}
}

func TestSelectInfoIsCalledOnlyForMatchedRows(t *testing.T) {
	files := fstest.MapFS{}
	for i := range 200 {
		files[fixtureName("root/file", i, ".log")] = &fstest.MapFile{Data: []byte("x")}
	}
	files["root/needle.txt"] = &fstest.MapFile{Data: []byte("found")}

	counting := newCountingFS(files)
	compiler := engine.NewCompiler(engine.DefaultFields(counting), engine.DefaultOperators())
	resolver := engine.NewPathResolver(counting, "/")
	exec := engine.NewSelectExecutor(counting, resolver, compiler, engine.EmptySkipList())

	tokens, _ := query.NewLexer().Lex("select files from 'root' where type = 'txt'")
	stmt, err := query.NewParser(compiler).Parse(tokens)
	if err != nil {
		t.Fatal(err)
	}

	sink := &engine.SliceSink{}
	if err := exec.Execute(context.Background(), stmt, sink); err != nil {
		t.Fatal(err)
	}

	if len(sink.Rows) != 1 {
		t.Fatalf("matched %d rows, want 1", len(sink.Rows))
	}
	if got := counting.infos.Load(); got != 1 {
		t.Errorf("Info() called %d times for 201 entries with 1 match; want exactly 1 — never stat to filter", got)
	}
}

func TestSelectCountChildReadsOnlyCandidateFolders(t *testing.T) {
	fsys := fstest.MapFS{
		"root/keep/a.txt":    {Data: []byte("a")},
		"root/keep/b.txt":    {Data: []byte("b")},
		"root/skipme/c.txt":  {Data: []byte("c")},
		"root/skipme/d.txt":  {Data: []byte("d")},
		"root/another/e.txt": {Data: []byte("e")},
		"root/another/f.txt": {Data: []byte("f")},
	}

	counting := newCountingFS(fsys)
	compiler := engine.NewCompiler(engine.DefaultFields(counting), engine.DefaultOperators())
	resolver := engine.NewPathResolver(counting, "/")
	exec := engine.NewSelectExecutor(counting, resolver, compiler, engine.EmptySkipList())

	tokens, _ := query.NewLexer().Lex("select folders from 'root' where name = 'keep' and count(child) = 2")
	stmt, err := query.NewParser(compiler).Parse(tokens)
	if err != nil {
		t.Fatal(err)
	}

	before := counting.readDirs.Load()
	sink := &engine.SliceSink{}
	if err := exec.Execute(context.Background(), stmt, sink); err != nil {
		t.Fatal(err)
	}
	after := counting.readDirs.Load()

	if len(sink.Rows) != 1 || sink.Rows[0].Name != "keep" {
		t.Fatalf("rows = %v, want just \"keep\"", rowNames(sink.Rows))
	}

	extra := after - before
	if extra > 3 {
		t.Errorf("ReadDir called %d times; the cheap name predicate must narrow before count(child) reads directories", extra)
	}
}

func TestSelectStreamsIntoTheSink(t *testing.T) {
	fsys := fstest.MapFS{}
	for i := range 50 {
		fsys[fixtureName("root/f", i, ".txt")] = &fstest.MapFile{Data: []byte("x")}
	}

	exec, compiler := executorFor(t, fsys)
	tokens, _ := query.NewLexer().Lex("select files from 'root'")
	stmt, _ := query.NewParser(compiler).Parse(tokens)

	sink := &engine.SliceSink{Limit: 5}
	if err := exec.Execute(context.Background(), stmt, sink); err != nil {
		t.Fatalf("Execute error = %v; a full sink is normal termination", err)
	}
	if len(sink.Rows) != 5 {
		t.Errorf("collected %d rows, want 5", len(sink.Rows))
	}
}

func TestSelectSurfacesResolverErrors(t *testing.T) {
	fsys := fstest.MapFS{"root/a.txt": {Data: []byte("a")}}
	exec, compiler := executorFor(t, fsys)

	tests := []struct {
		input string
		kind  oerr.Kind
	}{
		{"select files from 'nope'", oerr.KindFolderMissing},
		{"select files from 'root/a.txt'", oerr.KindPathIsFile},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens, _ := query.NewLexer().Lex(tt.input)
			stmt, err := query.NewParser(compiler).Parse(tokens)
			if err != nil {
				t.Fatal(err)
			}
			err = exec.Execute(context.Background(), stmt, &engine.SliceSink{})
			if err == nil {
				t.Fatalf("Execute(%q) succeeded", tt.input)
			}
			if !oerr.Is(err, tt.kind) {
				t.Errorf("error kind = %v, want %v", err, tt.kind)
			}
		})
	}
}

func TestSelectRespectsContextCancellation(t *testing.T) {
	fsys := fstest.MapFS{"root/a.txt": {Data: []byte("a")}}
	exec, compiler := executorFor(t, fsys)

	tokens, _ := query.NewLexer().Lex("select files from 'root'")
	stmt, _ := query.NewParser(compiler).Parse(tokens)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := exec.Execute(ctx, stmt, &engine.SliceSink{}); err == nil {
		t.Error("Execute ignored a cancelled context")
	}
}

func TestSelectEmptyFolderYieldsNoRows(t *testing.T) {
	fsys := fstest.MapFS{"root/sub/.keep": {Data: []byte("")}}

	got := runSelect(t, fsys, "select files from 'root'")
	if len(got) != 0 {
		t.Errorf("got %v, want no rows", rowNames(got))
	}
}

func TestSelectNoMatchYieldsNoRows(t *testing.T) {
	fsys := fstest.MapFS{"root/a.txt": {Data: []byte("a")}}

	got := runSelect(t, fsys, "select files from 'root' where type = 'pdf'")
	if len(got) != 0 {
		t.Errorf("got %v, want no rows", rowNames(got))
	}
}

func TestSliceSinkLimit(t *testing.T) {
	s := &engine.SliceSink{Limit: 2}

	if err := s.Push(engine.Row{Name: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Push(engine.Row{Name: "b"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Push(engine.Row{Name: "c"}); err == nil {
		t.Error("Push past the limit succeeded; want ErrStopWalk")
	}
	if len(s.Rows) != 2 {
		t.Errorf("collected %d rows, want 2", len(s.Rows))
	}
}

func TestSliceSinkUnlimited(t *testing.T) {
	s := &engine.SliceSink{}

	for i := range 100 {
		if err := s.Push(engine.Row{Name: fixtureName("f", i, "")}); err != nil {
			t.Fatalf("Push %d error = %v", i, err)
		}
	}
	if len(s.Rows) != 100 {
		t.Errorf("collected %d rows, want 100", len(s.Rows))
	}
}

func fixtureName(prefix string, i int, suffix string) string {
	digits := []byte{byte('0' + i/100%10), byte('0' + i/10%10), byte('0' + i%10)}
	return prefix + string(digits) + suffix
}
