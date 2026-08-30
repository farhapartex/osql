package test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/shell"
)

type blockingExecutor struct {
	started   chan struct{}
	rowsFirst int
}

func (b *blockingExecutor) Verb() string { return query.VerbSelect }

func (b *blockingExecutor) Execute(ctx context.Context, stmt *query.Statement, out engine.RowSink) error {
	for i := 0; i < b.rowsFirst; i++ {
		if err := out.Push(engine.Row{Name: "found.txt"}); err != nil {
			return err
		}
	}
	close(b.started)
	<-ctx.Done()
	return ctx.Err()
}

func shellAwaitingInterrupt(t *testing.T, rowsFirst int) (*shell.Shell, *blockingExecutor, *bytes.Buffer) {
	t.Helper()

	executor := &blockingExecutor{started: make(chan struct{}), rowsFirst: rowsFirst}
	out := &bytes.Buffer{}

	app := shell.New(shell.Config{
		Out:      out,
		Err:      out,
		Lexer:    query.NewLexer(),
		Parser:   query.NewParser(engine.NewCompiler(engine.DefaultFields(&fakeFileSystem{fsys: pipelineFS()}), engine.DefaultOperators())),
		Engine:   engine.NewRegistry(executor),
		Renderer: &fakeRenderer{},
	})
	return app, executor, out
}

func TestInterruptReportsNothingWhenNoQueryIsRunning(t *testing.T) {
	app, _, _ := shellAwaitingInterrupt(t, 0)

	if app.Interrupt() {
		t.Error("Interrupt() = true with no query running, want false")
	}
}

func TestInterruptStopsARunningQuery(t *testing.T) {
	app, executor, _ := shellAwaitingInterrupt(t, 0)

	failed := make(chan error, 1)
	go func() { failed <- app.Dispatch("files from 'work'") }()

	<-executor.started

	if !app.Interrupt() {
		t.Fatal("Interrupt() = false while a query was running, want true")
	}

	select {
	case err := <-failed:
		if !oerr.Is(err, oerr.KindQueryStopped) {
			t.Fatalf("Dispatch() = %v, want query_stopped", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the query did not stop after Interrupt()")
	}
}

func TestStoppedQuerySaysWhatItHadFound(t *testing.T) {
	app, executor, _ := shellAwaitingInterrupt(t, 3)

	failed := make(chan error, 1)
	go func() { failed <- app.Dispatch("files from 'work'") }()

	<-executor.started
	app.Interrupt()

	select {
	case err := <-failed:
		if !strings.Contains(err.Error(), "3 matches") {
			t.Errorf("message must report the partial count, got: %s", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the query did not stop")
	}
}

func TestInterruptIsSpentOnceTheQueryEnds(t *testing.T) {
	app, executor, _ := shellAwaitingInterrupt(t, 0)

	done := make(chan error, 1)
	go func() { done <- app.Dispatch("files from 'work'") }()

	<-executor.started
	app.Interrupt()
	<-done

	if app.Interrupt() {
		t.Error("Interrupt() = true after the query ended, want false")
	}
}

func TestASecondQueryRunsAfterAnInterrupt(t *testing.T) {
	app, executor, _ := shellAwaitingInterrupt(t, 0)

	done := make(chan error, 1)
	go func() { done <- app.Dispatch("files from 'work'") }()
	<-executor.started
	app.Interrupt()
	<-done

	if app.Interrupt() {
		t.Error("the shell should be idle again")
	}
}

func TestQueryStoppedMessages(t *testing.T) {
	tests := []struct {
		name  string
		found int
		want  string
	}{
		{"nothing found", 0, "Stopped. Nothing had matched yet."},
		{"negative is treated as nothing", -1, "Stopped. Nothing had matched yet."},
		{"some found", 12, "Stopped after 12 matches. Narrow the folder or add a where clause to make it quicker."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := oerr.QueryStopped(tt.found).Error(); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}

	if !oerr.Is(oerr.QueryStopped(1), oerr.KindQueryStopped) {
		t.Error("QueryStopped does not carry its kind")
	}
}

func TestScannerStopsOnACancelledContext(t *testing.T) {
	fsys := &fakeFileSystem{fsys: pipelineFS()}
	resolver := engine.NewPathResolver(fsys, "/")
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())
	selector := engine.NewSelectExecutor(fsys, resolver, compiler, engine.EmptySkipList())

	stmt, err := query.NewParser(compiler).Parse(mustLex(t, "files from 'work' recursive"))
	if err != nil {
		t.Fatalf("Parse() = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sink := &engine.SliceSink{}
	if err := selector.Execute(ctx, stmt, sink); err == nil {
		t.Fatal("Execute() on a cancelled context returned no error")
	}
	if len(sink.Rows) != 0 {
		t.Errorf("a cancelled scan pushed %d rows, want 0", len(sink.Rows))
	}
}
