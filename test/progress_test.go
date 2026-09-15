package test

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/shell"
)

func crowdedFS(files int) fstest.MapFS {
	fsys := make(fstest.MapFS, files)
	for i := 0; i < files; i++ {
		fsys["big/f"+strconv.Itoa(i)+".txt"] = &fstest.MapFile{Data: []byte("x")}
	}
	return fsys
}

func scanWithProgress(t *testing.T, files int, ctxFor func(context.Context, *[]int) context.Context) []int {
	t.Helper()

	fsys := &fakeFileSystem{fsys: crowdedFS(files)}
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())
	resolver := engine.NewPathResolver(fsys, "/")
	selector := engine.NewSelectExecutor(fsys, resolver, compiler, engine.EmptySkipList())

	stmt, err := query.NewParser(compiler).Parse(mustLex(t, "files from 'big' recursive"))
	if err != nil {
		t.Fatalf("Parse() = %v", err)
	}

	var seen []int
	ctx := ctxFor(context.Background(), &seen)

	if err := selector.Execute(ctx, stmt, &engine.SliceSink{}); err != nil {
		t.Fatalf("Execute() = %v", err)
	}
	return seen
}

func TestScanReportsProgressAtTheInterval(t *testing.T) {
	files := engine.ProgressInterval*2 + 100

	seen := scanWithProgress(t, files, func(ctx context.Context, seen *[]int) context.Context {
		return engine.WithProgress(ctx, func(scanned int) { *seen = append(*seen, scanned) })
	})

	want := []int{engine.ProgressInterval, engine.ProgressInterval * 2}
	if len(seen) != len(want) {
		t.Fatalf("reported %v, want %v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Errorf("report %d = %d, want %d", i, seen[i], want[i])
		}
	}
}

func TestShortScanReportsNoProgress(t *testing.T) {
	seen := scanWithProgress(t, 10, func(ctx context.Context, seen *[]int) context.Context {
		return engine.WithProgress(ctx, func(scanned int) { *seen = append(*seen, scanned) })
	})

	if len(seen) != 0 {
		t.Errorf("a short scan reported %v; nothing should be shown", seen)
	}
}

func TestScanWithoutAReporterStillWorks(t *testing.T) {
	seen := scanWithProgress(t, engine.ProgressInterval+10, func(ctx context.Context, _ *[]int) context.Context {
		return engine.WithProgress(ctx, nil)
	})

	if len(seen) != 0 {
		t.Errorf("a nil reporter produced %v", seen)
	}
}

func shellOverCrowdedFS(t *testing.T, editing bool, files int) (*shell.Shell, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	fsys := &fakeFileSystem{fsys: crowdedFS(files)}
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())
	resolver := engine.NewPathResolver(fsys, "/")

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	app := shell.New(shell.Config{
		Out:      out,
		Err:      errOut,
		Editing:  editing,
		Lexer:    query.NewLexer(),
		Parser:   query.NewParser(compiler),
		Engine:   engine.NewRegistry(engine.NewSelectExecutor(fsys, resolver, compiler, engine.EmptySkipList())),
		Renderer: &fakeRenderer{},
		Reader:   &fakeReader{},
	})
	return app, out, errOut
}

func TestProgressIsShownWhileInteractive(t *testing.T) {
	app, _, errOut := shellOverCrowdedFS(t, true, engine.ProgressInterval+50)

	if err := app.Dispatch("files from 'big' recursive"); err != nil {
		t.Fatalf("Dispatch() = %v", err)
	}

	got := errOut.String()
	if !strings.Contains(got, "scanned "+strconv.Itoa(engine.ProgressInterval)) {
		t.Errorf("no progress on stderr, got %q", got)
	}
}

func TestProgressIsSilentWhenNotInteractive(t *testing.T) {
	app, _, errOut := shellOverCrowdedFS(t, false, engine.ProgressInterval+50)

	if err := app.Dispatch("files from 'big' recursive"); err != nil {
		t.Fatalf("Dispatch() = %v", err)
	}

	if got := errOut.String(); got != "" {
		t.Errorf("a piped session must print no progress, got %q", got)
	}
}

func TestProgressNeverTouchesStdout(t *testing.T) {
	app, out, _ := shellOverCrowdedFS(t, true, engine.ProgressInterval+50)

	if err := app.Dispatch("files from 'big' recursive"); err != nil {
		t.Fatalf("Dispatch() = %v", err)
	}

	if strings.Contains(out.String(), "scanned") {
		t.Errorf("progress leaked into stdout, which would corrupt a pipe: %q", out.String())
	}
}

func TestProgressIsErasedWhenTheQueryEnds(t *testing.T) {
	app, _, errOut := shellOverCrowdedFS(t, true, engine.ProgressInterval+50)

	if err := app.Dispatch("files from 'big' recursive"); err != nil {
		t.Fatalf("Dispatch() = %v", err)
	}

	if !strings.HasSuffix(errOut.String(), "\r\033[K") {
		t.Errorf("the progress line was not erased, stderr ends with %q", tailOf(errOut.String()))
	}
}

func TestProgressCountsWhatWasScannedNotWhatMatched(t *testing.T) {
	fsys := &fakeFileSystem{fsys: crowdedFS(engine.ProgressInterval * 2)}
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())
	resolver := engine.NewPathResolver(fsys, "/")
	selector := engine.NewSelectExecutor(fsys, resolver, compiler, engine.EmptySkipList())

	stmt, err := query.NewParser(compiler).Parse(mustLex(t, "files from 'big' recursive where name = 'nothing.txt'"))
	if err != nil {
		t.Fatalf("Parse() = %v", err)
	}

	var seen []int
	ctx := engine.WithProgress(context.Background(), func(scanned int) { seen = append(seen, scanned) })

	sink := &engine.SliceSink{}
	if err := selector.Execute(ctx, stmt, sink); err != nil {
		t.Fatalf("Execute() = %v", err)
	}

	if len(sink.Rows) != 0 {
		t.Fatalf("the query was meant to match nothing, matched %d", len(sink.Rows))
	}
	want := []int{engine.ProgressInterval, engine.ProgressInterval * 2}
	if len(seen) != len(want) || seen[0] != want[0] {
		t.Errorf("progress reported %v, want %v even though nothing matched", seen, want)
	}
}

func tailOf(s string) string {
	if len(s) <= 12 {
		return s
	}
	return fmt.Sprintf("…%q", s[len(s)-12:])
}
