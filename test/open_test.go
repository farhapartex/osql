package test

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/vfs"
)

func openFS() fstest.MapFS {
	return fstest.MapFS{
		"notes.txt":       {Data: []byte("first line\nsecond line\n")},
		"no_newline.txt":  {Data: []byte("no trailing newline")},
		"empty.txt":       {Data: []byte("")},
		"one_char.txt":    {Data: []byte("x")},
		"binary.bin":      {Data: []byte{0x7f, 0x45, 0x4c, 0x46, 0x00, 0x01, 0x02}},
		"unicode.txt":     {Data: []byte("日本語\ncafé\n")},
		"crlf.txt":        {Data: []byte("line one\r\nline two\r\n")},
		"docs/inside.txt": {Data: []byte("nested\n")},
	}
}

func openExecutorFor(t *testing.T, fsys fs.FS) (*engine.OpenExecutor, *engine.Compiler) {
	t.Helper()

	vf := &fakeFileSystem{fsys: fsys}
	compiler := engine.NewCompiler(engine.DefaultFields(vf), engine.DefaultOperators())
	resolver := engine.NewPathResolver(vf, "/")
	return engine.NewOpenExecutor(vf, resolver), compiler
}

func runOpen(t *testing.T, fsys fs.FS, input string) (string, error) {
	t.Helper()

	exec, compiler := openExecutorFor(t, fsys)

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

func TestParseOpenForm(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		input string
		path  string
	}{
		{"open 'notes.txt'", "notes.txt"},
		{"open notes.txt", "notes.txt"},
		{"OPEN 'notes.txt'", "notes.txt"},
		{"open 'docs/inside.txt'", "docs/inside.txt"},
		{"open 'my file.txt'", "my file.txt"},
		{"open '/notes.txt'", "/notes.txt"},
		{"open '~/notes.txt'", "~/notes.txt"},
		{"open 'notes.txt';", "notes.txt"},
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
			if stmt.Verb != query.VerbOpen {
				t.Errorf("Verb = %q, want open", stmt.Verb)
			}
			if stmt.Path != tt.path {
				t.Errorf("Path = %q, want %q", stmt.Path, tt.path)
			}
			if stmt.Recursive {
				t.Error("open takes no recursive clause")
			}
			if len(stmt.Predicates) != 0 {
				t.Error("open takes no where clause")
			}
		})
	}
}

func TestParseOpenErrors(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		name  string
		input string
		kind  oerr.Kind
	}{
		{"no path", "open", oerr.KindMissingFilePath},
		{"trailing junk", "open 'a.txt' junk", oerr.KindUnexpectedInput},
		{"where is not allowed", "open 'a.txt' where name = 'b'", oerr.KindUnexpectedInput},
		{"recursive is not allowed", "open 'a.txt' recursive", oerr.KindUnexpectedInput},
		{"unclosed quote", "open 'a.txt", oerr.KindUnclosedQuote},
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

func TestOpenPrintsFileContents(t *testing.T) {
	got, err := runOpen(t, openFS(), "open 'notes.txt'")
	if err != nil {
		t.Fatalf("open error = %v", err)
	}
	if got != "first line\nsecond line\n" {
		t.Errorf("contents = %q", got)
	}
}

func TestOpenNestedFile(t *testing.T) {
	got, err := runOpen(t, openFS(), "open 'docs/inside.txt'")
	if err != nil {
		t.Fatalf("open error = %v", err)
	}
	if got != "nested\n" {
		t.Errorf("contents = %q", got)
	}
}

func TestOpenAddsAMissingTrailingNewline(t *testing.T) {
	got, err := runOpen(t, openFS(), "open 'no_newline.txt'")
	if err != nil {
		t.Fatalf("open error = %v", err)
	}
	if got != "no trailing newline\n" {
		t.Errorf("contents = %q; a missing final newline is added so the prompt starts fresh", got)
	}
}

func TestOpenDoesNotDoubleTheTrailingNewline(t *testing.T) {
	got, err := runOpen(t, openFS(), "open 'notes.txt'")
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasSuffix(got, "\n\n") {
		t.Errorf("contents end with two newlines: %q", got)
	}
}

func TestOpenEmptyFileWritesNothing(t *testing.T) {
	got, err := runOpen(t, openFS(), "open 'empty.txt'")
	if err != nil {
		t.Fatalf("open error = %v", err)
	}
	if got != "" {
		t.Errorf("contents = %q, want nothing at all", got)
	}
}

func TestOpenSingleCharacterFile(t *testing.T) {
	got, err := runOpen(t, openFS(), "open 'one_char.txt'")
	if err != nil {
		t.Fatal(err)
	}
	if got != "x\n" {
		t.Errorf("contents = %q, want \"x\\n\"", got)
	}
}

func TestOpenPreservesBytesExactly(t *testing.T) {
	tests := []struct {
		file string
		want string
	}{
		{"unicode.txt", "日本語\ncafé\n"},
		{"crlf.txt", "line one\r\nline two\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			got, err := runOpen(t, openFS(), "open '"+tt.file+"'")
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("contents = %q, want %q; open must not rewrite bytes", got, tt.want)
			}
		})
	}
}

func TestOpenRefusesFolders(t *testing.T) {
	_, err := runOpen(t, openFS(), "open 'docs'")
	if err == nil {
		t.Fatal("open on a folder succeeded")
	}
	if !oerr.Is(err, oerr.KindPathIsFolder) {
		t.Errorf("error kind = %v, want path_is_folder", err)
	}
	want := "'docs' is a folder, not a file. Try: open 'docs/notes.txt'"
	if err.Error() != want {
		t.Errorf("\n got: %s\nwant: %s", err.Error(), want)
	}
}

func TestOpenRefusesTheRootItself(t *testing.T) {
	for _, input := range []string{"open '.'", "open '/'", "open '~'"} {
		t.Run(input, func(t *testing.T) {
			_, err := runOpen(t, openFS(), input)
			if !oerr.Is(err, oerr.KindPathIsFolder) {
				t.Errorf("error = %v, want path_is_folder", err)
			}
		})
	}
}

func TestOpenMissingFile(t *testing.T) {
	_, err := runOpen(t, openFS(), "open 'nope.txt'")
	if err == nil {
		t.Fatal("open on a missing file succeeded")
	}
	if !oerr.Is(err, oerr.KindFileMissing) {
		t.Errorf("error kind = %v, want file_missing", err)
	}
	want := "I couldn't find a file at 'nope.txt'. Check the path and try again."
	if err.Error() != want {
		t.Errorf("\n got: %s\nwant: %s", err.Error(), want)
	}
}

func TestOpenRefusesBinaryFiles(t *testing.T) {
	_, err := runOpen(t, openFS(), "open 'binary.bin'")
	if err == nil {
		t.Fatal("open printed a binary file")
	}
	if !oerr.Is(err, oerr.KindBinaryFile) {
		t.Errorf("error kind = %v, want binary_file", err)
	}
}

func TestOpenWritesNothingWhenRefusingBinary(t *testing.T) {
	got, err := runOpen(t, openFS(), "open 'binary.bin'")
	if err == nil {
		t.Fatal("expected an error")
	}
	if got != "" {
		t.Errorf("wrote %d bytes before refusing; nothing should reach the terminal", len(got))
	}
}

func TestOpenDetectsBinaryPastTheFirstChunk(t *testing.T) {
	data := append(bytes.Repeat([]byte("text\n"), 4000), 0x00, 0x01)
	if len(data) <= 8192 {
		t.Fatalf("fixture is only %d bytes; it must exceed the sniff window", len(data))
	}
	fsys := fstest.MapFS{"late.bin": {Data: data}}

	got, err := runOpen(t, fsys, "open 'late.bin'")
	if err == nil {
		t.Fatal("a NUL past the first chunk was printed instead of refused")
	}
	if !oerr.Is(err, oerr.KindBinaryFile) {
		t.Errorf("error kind = %v, want binary_file", err)
	}
	if strings.Contains(got, "\x00") {
		t.Error("a NUL byte reached the writer")
	}
}

func TestOpenReadsAboveTheStartDirectory(t *testing.T) {
	fsys := fstest.MapFS{"box/inside.txt": {Data: []byte("in")}, "outside.txt": {Data: []byte("out")}}
	vf := &fakeFileSystem{fsys: fsys}
	resolver := engine.NewPathResolver(vf, "/box")
	exec := engine.NewOpenExecutor(vf, resolver)

	stmt := &query.Statement{Verb: query.VerbOpen, Path: "../outside.txt"}
	buf := &bytes.Buffer{}

	if err := exec.WriteContent(context.Background(), stmt, buf); err != nil {
		t.Fatalf("reading above the start directory must work now: %v", err)
	}
	if buf.String() != "out\n" {
		t.Errorf("content = %q, want \"out\\n\"", buf.String())
	}
}

func TestOpenRespectsContextCancellation(t *testing.T) {
	exec, _ := openExecutorFor(t, openFS())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := exec.WriteContent(ctx, &query.Statement{Verb: query.VerbOpen, Path: "notes.txt"}, &bytes.Buffer{})
	if err == nil {
		t.Error("open ignored a cancelled context")
	}
}

func TestOpenSurfacesWriteErrors(t *testing.T) {
	exec, _ := openExecutorFor(t, openFS())

	err := exec.WriteContent(context.Background(), &query.Statement{Verb: query.VerbOpen, Path: "notes.txt"}, errWriter{})
	if err == nil {
		t.Error("open swallowed a write failure")
	}
}

func TestOpenStreamsLargeFilesWithoutHoldingThem(t *testing.T) {
	big := strings.Repeat("this is a line of text\n", 50000)
	fsys := fstest.MapFS{"big.txt": {Data: []byte(big)}}

	got, err := runOpen(t, fsys, "open 'big.txt'")
	if err != nil {
		t.Fatalf("open error = %v", err)
	}
	if got != big {
		t.Errorf("large file not reproduced exactly: got %d bytes, want %d", len(got), len(big))
	}
}

func TestOpenRegistersUnderItsOwnVerb(t *testing.T) {
	opener, _ := openExecutorFor(t, openFS())
	selector, _ := executorFor(t, openFS())

	registry := engine.NewRegistry(selector, opener)

	got, ok := registry.Lookup("open")
	if !ok {
		t.Fatal("open executor not registered")
	}
	if _, isContent := got.(engine.ContentExecutor); !isContent {
		t.Error("open executor does not satisfy ContentExecutor, so the shell cannot stream it")
	}
	if opener.Verb() != "open" {
		t.Errorf("Verb() = %q, want open", opener.Verb())
	}
}

func TestOpenRowPathReportsItIsContentOnly(t *testing.T) {
	opener, _ := openExecutorFor(t, openFS())

	err := opener.Execute(context.Background(), &query.Statement{Verb: query.VerbOpen, Path: "notes.txt"}, &engine.SliceSink{})
	if err == nil {
		t.Error("Execute returned no error; open has no rows and must say so")
	}
}

func TestOpenOnRealFilesystem(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "real.txt"), []byte("on disk\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	vf := vfs.OS()
	exec := engine.NewOpenExecutor(vf, engine.NewPathResolver(vf, root))

	buf := &bytes.Buffer{}
	if err := exec.WriteContent(context.Background(), &query.Statement{Verb: query.VerbOpen, Path: "real.txt"}, buf); err != nil {
		t.Fatalf("open error = %v", err)
	}
	if buf.String() != "on disk\n" {
		t.Errorf("contents = %q", buf.String())
	}
}

func TestOpenNoPermissionOnRealFilesystem(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root, permission cannot be denied")
	}

	root := t.TempDir()
	locked := filepath.Join(root, "locked.txt")
	if err := os.WriteFile(locked, []byte("secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o644) })

	vf := vfs.OS()
	exec := engine.NewOpenExecutor(vf, engine.NewPathResolver(vf, root))

	buf := &bytes.Buffer{}
	err := exec.WriteContent(context.Background(), &query.Statement{Verb: query.VerbOpen, Path: "locked.txt"}, buf)
	if !oerr.Is(err, oerr.KindNoPermission) {
		t.Errorf("error kind = %v, want no_permission", err)
	}
	if buf.Len() != 0 {
		t.Error("wrote content from an unreadable file")
	}
}
