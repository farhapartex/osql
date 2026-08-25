package test

import (
	"bytes"
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/output"
	"github.com/farhapartex/osql/internal/query"
)

func countFS() fstest.MapFS {
	return fstest.MapFS{
		"root/a.txt":            {Data: []byte("a")},
		"root/b.txt":            {Data: []byte("b")},
		"root/c.log":            {Data: []byte("c")},
		"root/Makefile":         {Data: []byte("m")},
		"root/one/x.txt":        {Data: []byte("x")},
		"root/two/y.txt":        {Data: []byte("y")},
		"root/two/deep/z.txt":   {Data: []byte("z")},
		"elsewhere/ignored.txt": {Data: []byte("i")},
	}
}

func countExecutorFor(t *testing.T, fsys fs.FS) (*engine.CountExecutor, *engine.Compiler) {
	t.Helper()

	vf := &fakeFileSystem{fsys: fsys}
	compiler := engine.NewCompiler(engine.DefaultFields(vf), engine.DefaultOperators())
	resolver := engine.NewPathResolver(vf, "/")
	return engine.NewCountExecutor(vf, resolver, compiler, engine.EmptySkipList()), compiler
}

func runCount(t *testing.T, fsys fs.FS, input string) engine.Row {
	t.Helper()

	exec, compiler := countExecutorFor(t, fsys)

	tokens, err := query.NewLexer().Lex(input)
	if err != nil {
		t.Fatalf("Lex error = %v", err)
	}
	stmt, err := query.NewParser(compiler).Parse(tokens)
	if err != nil {
		t.Fatalf("Parse(%q) error = %v", input, err)
	}
	if stmt.Verb != query.VerbCount {
		t.Fatalf("Parse(%q) produced verb %q, want count", input, stmt.Verb)
	}

	sink := &engine.SliceSink{}
	if err := exec.Execute(context.Background(), stmt, sink); err != nil {
		t.Fatalf("Execute(%q) error = %v", input, err)
	}
	if len(sink.Rows) != 1 {
		t.Fatalf("count produced %d rows, want exactly 1", len(sink.Rows))
	}
	return sink.Rows[0]
}

func TestParseCountForm(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		input     string
		target    query.Target
		path      string
		recursive bool
		predicate int
	}{
		{input: "count(files) from 'root'", target: query.TargetFiles, path: "root"},
		{input: "count(folders) from 'root'", target: query.TargetFolders, path: "root"},
		{input: "count(all) from 'root'", target: query.TargetAll, path: "root"},
		{input: "COUNT(FILES) FROM 'root'", target: query.TargetFiles, path: "root"},
		{input: "count( files ) from 'root'", target: query.TargetFiles, path: "root"},
		{input: "count(files) from 'root';", target: query.TargetFiles, path: "root"},
		{input: "count(files) from 'root' recursive", target: query.TargetFiles, path: "root", recursive: true},
		{input: "count(files) from 'root' where type = 'txt'", target: query.TargetFiles, path: "root", predicate: 1},
		{input: "count(files) from 'root' recursive where type = 'txt'", target: query.TargetFiles, path: "root", recursive: true, predicate: 1},
		{input: "count(folders) from 'root' where count(child) > 1", target: query.TargetFolders, path: "root", predicate: 1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens, err := query.NewLexer().Lex(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			stmt, err := parser.Parse(tokens)
			if err != nil {
				t.Fatalf("Parse error = %v", err)
			}
			if stmt.Verb != query.VerbCount {
				t.Errorf("Verb = %q, want count", stmt.Verb)
			}
			if stmt.Target != tt.target {
				t.Errorf("Target = %v, want %v", stmt.Target, tt.target)
			}
			if stmt.Path != tt.path {
				t.Errorf("Path = %q, want %q", stmt.Path, tt.path)
			}
			if stmt.Recursive != tt.recursive {
				t.Errorf("Recursive = %v, want %v", stmt.Recursive, tt.recursive)
			}
			if len(stmt.Predicates) != tt.predicate {
				t.Errorf("got %d predicates, want %d", len(stmt.Predicates), tt.predicate)
			}
		})
	}
}

func TestParsePlainFormStillUsesSelectVerb(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	tokens, _ := query.NewLexer().Lex("files from 'root'")

	stmt, err := query.NewParser(compiler).Parse(tokens)
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Verb != query.VerbSelect {
		t.Errorf("Verb = %q, want select; only count( switches verbs", stmt.Verb)
	}
}

func TestParseCountErrors(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		name  string
		input string
		kind  oerr.Kind
	}{
		{"singular target", "count(file) from 'root'", oerr.KindSingularTarget},
		{"singular folder", "count(folder) from 'root'", oerr.KindSingularTarget},
		{"unknown target", "count(bogus) from 'root'", oerr.KindUnknownTarget},
		{"child is not a target", "count(child) from 'root'", oerr.KindUnknownTarget},
		{"unclosed paren", "count(files from 'root'", oerr.KindUnclosedCount},
		{"empty parens", "count() from 'root'", oerr.KindMissingTarget},
		{"missing from", "count(files) 'root'", oerr.KindMissingFrom},
		{"missing path", "count(files) from", oerr.KindMissingPath},
		{"trailing junk", "count(files) from 'root' junk", oerr.KindUnexpectedInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := query.NewLexer().Lex(tt.input)
			if err != nil {
				if !oerr.Is(err, tt.kind) {
					t.Fatalf("Lex(%q) error = %v, want %v", tt.input, err, tt.kind)
				}
				return
			}
			_, err = parser.Parse(tokens)
			if err == nil {
				t.Fatalf("Parse(%q) succeeded", tt.input)
			}
			if !oerr.Is(err, tt.kind) {
				t.Errorf("Parse(%q)\n got: %v\nwant kind: %v", tt.input, err, tt.kind)
			}
		})
	}
}

func TestCountBareWordIsStillAField(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	tokens, _ := query.NewLexer().Lex("folders from 'root' where count(child) > 1")

	stmt, err := query.NewParser(compiler).Parse(tokens)
	if err != nil {
		t.Fatalf("count(child) inside where broke: %v", err)
	}
	if stmt.Verb != query.VerbSelect {
		t.Errorf("Verb = %q; count(child) in a where clause is a field, not the count form", stmt.Verb)
	}
	if stmt.Predicates[0].Field != "count(child)" {
		t.Errorf("Field = %q", stmt.Predicates[0].Field)
	}
}

func TestCountTotals(t *testing.T) {
	fsys := countFS()

	tests := []struct {
		input string
		what  string
		want  int64
	}{
		{"count(files) from 'root'", "files", 4},
		{"count(folders) from 'root'", "folders", 2},
		{"count(all) from 'root'", "all", 6},
		{"count(files) from 'root' where type = 'txt'", "files", 2},
		{"count(files) from 'root' where type = 'log'", "files", 1},
		{"count(files) from 'root' recursive", "files", 7},
		{"count(all) from 'root' recursive", "all", 10},
		{"count(files) from 'root' recursive where type = 'txt'", "files", 5},
		{"count(folders) from 'root' recursive", "folders", 3},
		{"count(files) from 'root' where type = 'zzz'", "files", 0},
		{"count(files) from 'root/one'", "files", 1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			row := runCount(t, fsys, tt.input)
			if row.Count != tt.want {
				t.Errorf("%q = %d, want %d", tt.input, row.Count, tt.want)
			}
			if row.Name != tt.what {
				t.Errorf("Name = %q, want %q", row.Name, tt.what)
			}
		})
	}
}

func TestCountAllEqualsFilesPlusFolders(t *testing.T) {
	fsys := countFS()

	files := runCount(t, fsys, "count(files) from 'root' recursive").Count
	folders := runCount(t, fsys, "count(folders) from 'root' recursive").Count
	all := runCount(t, fsys, "count(all) from 'root' recursive").Count

	if files+folders != all {
		t.Errorf("files(%d) + folders(%d) = %d, but all = %d", files, folders, files+folders, all)
	}
}

func TestCountMatchesTheNumberOfListedRows(t *testing.T) {
	fsys := countFS()

	listed := runSelect(t, fsys, "files from 'root' recursive where type = 'txt'")
	counted := runCount(t, fsys, "count(files) from 'root' recursive where type = 'txt'")

	if int64(len(listed)) != counted.Count {
		t.Errorf("listing returned %d rows but count said %d", len(listed), counted.Count)
	}
}

func TestCountNeverReadsFileInfo(t *testing.T) {
	files := fstest.MapFS{}
	for i := range 100 {
		files[fixtureName("root/f", i, ".txt")] = &fstest.MapFile{Data: []byte("x")}
	}

	counting := newCountingFS(files)
	compiler := engine.NewCompiler(engine.DefaultFields(counting), engine.DefaultOperators())
	resolver := engine.NewPathResolver(counting, "/")
	exec := engine.NewCountExecutor(counting, resolver, compiler, engine.EmptySkipList())

	tokens, _ := query.NewLexer().Lex("count(files) from 'root'")
	stmt, err := query.NewParser(compiler).Parse(tokens)
	if err != nil {
		t.Fatal(err)
	}

	sink := &engine.SliceSink{}
	if err := exec.Execute(context.Background(), stmt, sink); err != nil {
		t.Fatal(err)
	}

	if sink.Rows[0].Count != 100 {
		t.Fatalf("counted %d, want 100", sink.Rows[0].Count)
	}
	if got := counting.infos.Load(); got != 0 {
		t.Errorf("Info() called %d times while counting; a count needs no size or timestamp", got)
	}
}

func TestCountSurfacesResolverErrors(t *testing.T) {
	fsys := countFS()
	exec, compiler := countExecutorFor(t, fsys)

	for _, tt := range []struct {
		input string
		kind  oerr.Kind
	}{
		{"count(files) from 'nope'", oerr.KindFolderMissing},
		{"count(files) from 'root/a.txt'", oerr.KindPathIsFile},
	} {
		t.Run(tt.input, func(t *testing.T) {
			tokens, _ := query.NewLexer().Lex(tt.input)
			stmt, err := query.NewParser(compiler).Parse(tokens)
			if err != nil {
				t.Fatal(err)
			}
			if err := exec.Execute(context.Background(), stmt, &engine.SliceSink{}); !oerr.Is(err, tt.kind) {
				t.Errorf("error = %v, want %v", err, tt.kind)
			}
		})
	}
}

func TestCountRegistersUnderItsOwnVerb(t *testing.T) {
	counter, _ := countExecutorFor(t, countFS())
	selector, _ := executorFor(t, countFS())

	registry := engine.NewRegistry(selector, counter)

	if _, ok := registry.Lookup("count"); !ok {
		t.Error("count executor not registered")
	}
	if _, ok := registry.Lookup("select"); !ok {
		t.Error("select executor no longer registered")
	}
	if counter.Verb() != "count" {
		t.Errorf("Verb() = %q, want count", counter.Verb())
	}
}

func TestCountRendererOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	rows := []engine.Row{{Name: "files", Count: 52}}

	if err := output.NewCount().Render(buf, rows); err != nil {
		t.Fatalf("Render error = %v", err)
	}

	want := "WHAT   COUNT\nfiles  52\n"
	if buf.String() != want {
		t.Errorf("\n got: %q\nwant: %q", buf.String(), want)
	}
}

func TestCountRendererZeroAndLargeValues(t *testing.T) {
	tests := []struct {
		row  engine.Row
		want string
	}{
		{engine.Row{Name: "files", Count: 0}, "0"},
		{engine.Row{Name: "folders", Count: 1}, "1"},
		{engine.Row{Name: "all", Count: 1234567}, "1234567"},
	}

	for _, tt := range tests {
		buf := &bytes.Buffer{}
		if err := output.NewCount().Render(buf, []engine.Row{tt.row}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), tt.want) {
			t.Errorf("Render(%+v) = %q, want it to contain %q", tt.row, buf.String(), tt.want)
		}
		if !strings.Contains(buf.String(), tt.row.Name) {
			t.Errorf("Render(%+v) omitted the target name", tt.row)
		}
	}
}

func TestCountRendererHeaderAndNoFooter(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := output.NewCount().Render(buf, []engine.Row{{Name: "files", Count: 3}}); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	if !strings.HasPrefix(got, output.HeaderWhat) {
		t.Errorf("output does not start with the WHAT header: %q", got)
	}
	if !strings.Contains(got, output.HeaderCount) {
		t.Errorf("output has no COUNT header: %q", got)
	}
	if strings.Contains(got, "files\n\n") || strings.Contains(got, " files\n") {
		t.Errorf("count output should carry no row-count footer: %q", got)
	}
}

func TestCountRendererSurfacesWriteErrors(t *testing.T) {
	if err := output.NewCount().Render(errWriter{}, []engine.Row{{Name: "files", Count: 1}}); err == nil {
		t.Error("Render swallowed a write failure")
	}
}

func TestCountRendererSatisfiesTheInterface(t *testing.T) {
	var _ output.Renderer = output.NewCount()
}
