package test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/vfs"
)

func newExecutorFor(t *testing.T) (*engine.NewExecutor, string) {
	t.Helper()

	root := t.TempDir()
	fsys := vfs.NewOS(root)
	return engine.NewNewExecutor(fsys, engine.NewPathResolver(fsys, root)), root
}

func execNew(t *testing.T, exec *engine.NewExecutor, input string) (string, error) {
	t.Helper()

	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	tokens, err := query.NewLexer().Lex(input)
	if err != nil {
		return "", err
	}
	stmt, err := query.NewParser(compiler).Parse(tokens)
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}
	err = exec.WriteContent(context.Background(), stmt, buf)
	return buf.String(), err
}

func runNewIn(t *testing.T, exec *engine.NewExecutor, input string) string {
	t.Helper()

	out, err := execNew(t, exec, input)
	if err != nil {
		t.Fatalf("%q error = %v", input, err)
	}
	return out
}

func TestParseNewForm(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		input   string
		kind    query.NewKind
		path    string
		data    string
		hasData bool
	}{
		{input: "new file 'notes.txt'", kind: query.NewFile, path: "notes.txt"},
		{input: "new folder 'reports'", kind: query.NewFolder, path: "reports"},
		{input: "NEW FILE 'notes.txt'", kind: query.NewFile, path: "notes.txt"},
		{input: "new file notes.txt", kind: query.NewFile, path: "notes.txt"},
		{input: "new file 'a/b/c.txt'", kind: query.NewFile, path: "a/b/c.txt"},
		{input: "new file '/notes.txt'", kind: query.NewFile, path: "/notes.txt"},
		{input: "new file '~/notes.txt'", kind: query.NewFile, path: "~/notes.txt"},
		{input: "new file 'notes.txt';", kind: query.NewFile, path: "notes.txt"},
		{input: "new file 'notes.txt' data='hello'", kind: query.NewFile, path: "notes.txt", data: "hello", hasData: true},
		{input: "new file 'notes.txt' data='hello hello line testing'", kind: query.NewFile, path: "notes.txt", data: "hello hello line testing", hasData: true},
		{input: "new file 'notes.txt' DATA='hello'", kind: query.NewFile, path: "notes.txt", data: "hello", hasData: true},
		{input: "new file 'notes.txt' data=''", kind: query.NewFile, path: "notes.txt", data: "", hasData: true},
		{input: "new file 'notes.txt' data = 'hello'", kind: query.NewFile, path: "notes.txt", data: "hello", hasData: true},
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
			if stmt.Verb != query.VerbNew {
				t.Errorf("Verb = %q, want new", stmt.Verb)
			}
			if stmt.Kind != tt.kind {
				t.Errorf("Kind = %v, want %v", stmt.Kind, tt.kind)
			}
			if stmt.Path != tt.path {
				t.Errorf("Path = %q, want %q", stmt.Path, tt.path)
			}
			if stmt.Data != tt.data {
				t.Errorf("Data = %q, want %q", stmt.Data, tt.data)
			}
			if stmt.HasData != tt.hasData {
				t.Errorf("HasData = %v, want %v", stmt.HasData, tt.hasData)
			}
		})
	}
}

func TestParseNewErrors(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		name  string
		input string
		kind  oerr.Kind
	}{
		{"no kind", "new", oerr.KindMissingNewTarget},
		{"unknown kind", "new thing 'a'", oerr.KindMissingNewTarget},
		{"plural file", "new files 'a.txt'", oerr.KindSingularNewTarget},
		{"plural folder", "new folders 'a'", oerr.KindSingularNewTarget},
		{"no path", "new file", oerr.KindMissingNewPath},
		{"no folder path", "new folder", oerr.KindMissingNewPath},
		{"data with no value", "new file 'a.txt' data=", oerr.KindMissingDataValue},
		{"data with no equals", "new file 'a.txt' data 'x'", oerr.KindUnexpectedInput},
		{"data on a folder", "new folder 'a' data='x'", oerr.KindDataOnFolder},
		{"trailing junk", "new file 'a.txt' junk", oerr.KindUnexpectedInput},
		{"where is not allowed", "new file 'a.txt' where name = 'b'", oerr.KindUnexpectedInput},
		{"recursive is not allowed", "new file 'a.txt' recursive", oerr.KindUnexpectedInput},
		{"unclosed quote", "new file 'a.txt", oerr.KindUnclosedQuote},
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
			if _, err = parser.Parse(tokens); err == nil {
				t.Fatalf("Parse(%q) succeeded", tt.input)
			} else if !oerr.Is(err, tt.kind) {
				t.Errorf("Parse(%q)\n got: %v\nwant kind: %v", tt.input, err, tt.kind)
			}
		})
	}
}

func TestNewCreatesAnEmptyFile(t *testing.T) {
	exec, root := newExecutorFor(t)

	out := runNewIn(t, exec, "new file 'notes.txt'")
	if !strings.Contains(out, "Created 'notes.txt'") {
		t.Errorf("output = %q", out)
	}

	data, err := os.ReadFile(filepath.Join(root, "notes.txt"))
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("file has %d bytes, want an empty file", len(data))
	}
}

func TestNewCreatesAFolder(t *testing.T) {
	exec, root := newExecutorFor(t)

	runNewIn(t, exec, "new folder 'example'")

	info, err := os.Stat(filepath.Join(root, "example"))
	if err != nil {
		t.Fatalf("folder not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("created a file, not a folder")
	}
}

func TestNewWritesData(t *testing.T) {
	exec, root := newExecutorFor(t)

	runNewIn(t, exec, "new file 'hello.txt' data='hello hello line testing'")

	data, err := os.ReadFile(filepath.Join(root, "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello hello line testing" {
		t.Errorf("contents = %q", data)
	}
}

func TestNewWritesDataExactlyWithNoExtraNewline(t *testing.T) {
	exec, root := newExecutorFor(t)

	runNewIn(t, exec, "new file 'exact.txt' data='no newline'")

	data, _ := os.ReadFile(filepath.Join(root, "exact.txt"))
	if strings.HasSuffix(string(data), "\n") {
		t.Errorf("contents = %q; data is written exactly as given", data)
	}
}

func TestNewEmptyDataMakesAnEmptyFile(t *testing.T) {
	exec, root := newExecutorFor(t)

	runNewIn(t, exec, "new file 'blank.txt' data=''")

	data, err := os.ReadFile(filepath.Join(root, "blank.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Errorf("contents = %q, want empty", data)
	}
}

func TestNewCreatesMissingParents(t *testing.T) {
	exec, root := newExecutorFor(t)

	out := runNewIn(t, exec, "new file 'Documents/goupp/test1/test2/deep.txt'")

	if _, err := os.Stat(filepath.Join(root, "Documents", "goupp", "test1", "test2", "deep.txt")); err != nil {
		t.Fatalf("nested file not created: %v", err)
	}
	if !strings.Contains(out, "also created:") {
		t.Errorf("output does not report the folders it made:\n%s", out)
	}
	for _, want := range []string{"Documents", "test1", "test2"} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not mention %q:\n%s", want, out)
		}
	}
}

func TestNewCreatesMissingParentsForFolders(t *testing.T) {
	exec, root := newExecutorFor(t)

	runNewIn(t, exec, "new folder 'a/b/c'")

	info, err := os.Stat(filepath.Join(root, "a", "b", "c"))
	if err != nil {
		t.Fatalf("nested folder not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("created a file, not a folder")
	}
}

func TestNewDoesNotReportParentsThatAlreadyExisted(t *testing.T) {
	exec, root := newExecutorFor(t)
	if err := os.MkdirAll(filepath.Join(root, "existing"), 0o755); err != nil {
		t.Fatal(err)
	}

	out := runNewIn(t, exec, "new file 'existing/notes.txt'")

	if strings.Contains(out, "also created:") {
		t.Errorf("reported creating a folder that was already there:\n%s", out)
	}
}

func TestNewRefusesAnExistingFile(t *testing.T) {
	exec, root := newExecutorFor(t)
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := execNew(t, exec, "new file 'keep.txt'")
	if !oerr.Is(err, oerr.KindAlreadyExists) {
		t.Errorf("error kind = %v, want already_exists", err)
	}

	data, _ := os.ReadFile(filepath.Join(root, "keep.txt"))
	if string(data) != "keep me\n" {
		t.Errorf("existing file was modified: %q", data)
	}
}

func TestNewWithDataNeverOverwrites(t *testing.T) {
	exec, root := newExecutorFor(t)
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := execNew(t, exec, "new file 'keep.txt' data='destroyed'")
	if err == nil {
		t.Fatal("data overwrote an existing file")
	}
	if !oerr.Is(err, oerr.KindAlreadyExists) {
		t.Errorf("error kind = %v, want already_exists", err)
	}

	data, _ := os.ReadFile(filepath.Join(root, "keep.txt"))
	if string(data) != "original\n" {
		t.Errorf("contents changed to %q; refusing must never touch the file", data)
	}
}

func TestNewRefusesAnExistingFolder(t *testing.T) {
	exec, root := newExecutorFor(t)
	if err := os.MkdirAll(filepath.Join(root, "existing"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := execNew(t, exec, "new folder 'existing'")
	if !oerr.Is(err, oerr.KindAlreadyExists) {
		t.Errorf("error kind = %v, want already_exists", err)
	}
}

func TestNewRefusesTheRootItself(t *testing.T) {
	exec, _ := newExecutorFor(t)

	for _, input := range []string{"new folder '.'", "new folder '/'", "new folder '~'"} {
		t.Run(input, func(t *testing.T) {
			if _, err := execNew(t, exec, input); !oerr.Is(err, oerr.KindAlreadyExists) {
				t.Errorf("error = %v, want already_exists", err)
			}
		})
	}
}

func TestNewRefusesToEscapeTheRoot(t *testing.T) {
	exec, root := newExecutorFor(t)

	for _, input := range []string{"new file '../escape.txt'", "new folder '../../escape'"} {
		t.Run(input, func(t *testing.T) {
			_, err := execNew(t, exec, input)
			if !oerr.Is(err, oerr.KindOutsideRoot) {
				t.Errorf("error kind = %v, want outside_root", err)
			}
		})
	}

	if _, err := os.Stat(filepath.Join(filepath.Dir(root), "escape.txt")); err == nil {
		t.Error("a file was created outside the root")
	}
}

func TestNewFileModeIsReadableAndWritable(t *testing.T) {
	exec, root := newExecutorFor(t)

	runNewIn(t, exec, "new file 'notes.txt'")
	runNewIn(t, exec, "new folder 'reports'")

	file, err := os.Stat(filepath.Join(root, "notes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got := file.Mode().Perm(); got != 0o644 {
		t.Errorf("file mode = %04o, want 0644", got)
	}

	dir, err := os.Stat(filepath.Join(root, "reports"))
	if err != nil {
		t.Fatal(err)
	}
	if got := dir.Mode().Perm(); got != 0o755 {
		t.Errorf("folder mode = %04o, want 0755", got)
	}
}

func TestNewSurfacesPermissionErrors(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root, permission cannot be denied")
	}

	exec, root := newExecutorFor(t)
	locked := filepath.Join(root, "locked")
	if err := os.MkdirAll(locked, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })

	_, err := execNew(t, exec, "new file 'locked/nope.txt'")
	if err == nil {
		t.Fatal("created a file in a read-only folder")
	}
	if !strings.Contains(err.Error(), "locked/nope.txt") {
		t.Errorf("error does not name the path: %v", err)
	}
}

func TestNewIsUsableAfterCreating(t *testing.T) {
	exec, _ := newExecutorFor(t)

	runNewIn(t, exec, "new folder 'reports'")
	runNewIn(t, exec, "new file 'reports/q4.txt' data='numbers'")

	out := runNewIn(t, exec, "new file 'reports/q3.txt'")
	if !strings.Contains(out, "Created") {
		t.Errorf("second file in an existing folder failed: %q", out)
	}
}

func TestNewRegistersUnderItsOwnVerb(t *testing.T) {
	exec, _ := newExecutorFor(t)

	registry := engine.NewRegistry(exec)
	got, ok := registry.Lookup("new")
	if !ok {
		t.Fatal("new executor not registered")
	}
	if _, isContent := got.(engine.ContentExecutor); !isContent {
		t.Error("new executor does not satisfy ContentExecutor")
	}
	if exec.Verb() != "new" {
		t.Errorf("Verb() = %q, want new", exec.Verb())
	}
}

func TestNewRowPathReportsItIsContentOnly(t *testing.T) {
	exec, _ := newExecutorFor(t)

	err := exec.Execute(context.Background(), &query.Statement{Verb: query.VerbNew, Path: "x"}, &engine.SliceSink{})
	if err == nil {
		t.Error("Execute returned no error; new has no rows")
	}
}

func TestNewKindString(t *testing.T) {
	tests := []struct {
		kind query.NewKind
		want string
	}{
		{query.NewFile, "file"},
		{query.NewFolder, "folder"},
		{query.NewKind(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("NewKind(%d).String() = %q, want %q", int(tt.kind), got, tt.want)
		}
	}
}

func TestParseNewKind(t *testing.T) {
	for _, tt := range []struct {
		in     string
		want   query.NewKind
		wantOk bool
	}{
		{"file", query.NewFile, true},
		{"folder", query.NewFolder, true},
		{"files", query.NewFile, false},
		{"File", query.NewFile, false},
		{"", query.NewFile, false},
	} {
		got, ok := query.ParseNewKind(tt.in)
		if ok != tt.wantOk {
			t.Errorf("ParseNewKind(%q) ok = %v, want %v", tt.in, ok, tt.wantOk)
			continue
		}
		if ok && got != tt.want {
			t.Errorf("ParseNewKind(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
