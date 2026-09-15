package test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
)

func TestLimitSinkStopsAtItsLimit(t *testing.T) {
	inner := &engine.SliceSink{}
	sink := engine.NewLimitSink(inner, 2)

	if err := sink.Push(engine.Row{Name: "a"}); err != nil {
		t.Fatalf("first push = %v", err)
	}
	if sink.Filled() {
		t.Error("Filled() = true after one of two")
	}

	err := sink.Push(engine.Row{Name: "b"})
	if !errors.Is(err, engine.ErrStopWalk) {
		t.Fatalf("second push = %v, want ErrStopWalk", err)
	}
	if !sink.Filled() {
		t.Error("Filled() = false after reaching the limit")
	}
	if got := len(inner.Rows); got != 2 {
		t.Errorf("the inner sink took %d rows, want 2", got)
	}
	if got := sink.Taken(); got != 2 {
		t.Errorf("Taken() = %d, want 2", got)
	}
}

func TestLimitSinkRefusesAnythingAfterTheLimit(t *testing.T) {
	inner := &engine.SliceSink{}
	sink := engine.NewLimitSink(inner, 1)

	if err := sink.Push(engine.Row{Name: "a"}); !errors.Is(err, engine.ErrStopWalk) {
		t.Fatalf("push = %v, want ErrStopWalk", err)
	}
	if err := sink.Push(engine.Row{Name: "b"}); !errors.Is(err, engine.ErrStopWalk) {
		t.Fatalf("a later push = %v, want ErrStopWalk", err)
	}
	if got := len(inner.Rows); got != 1 {
		t.Errorf("the inner sink took %d rows, want 1", got)
	}
}

func deepFS(dirs, perDir int) fstest.MapFS {
	fsys := fstest.MapFS{}
	for d := 0; d < dirs; d++ {
		for f := 0; f < perDir; f++ {
			name := "tree/d" + strconv.Itoa(d) + "/f" + strconv.Itoa(f) + ".txt"
			fsys[name] = &fstest.MapFile{Data: []byte("x")}
		}
	}
	return fsys
}

func TestLimitEndsTheWalkInsteadOfFilteringAfterwards(t *testing.T) {
	counter := newCountingFS(deepFS(40, 20))
	vf := &fakeFileSystem{fsys: counter}

	compiler := engine.NewCompiler(engine.DefaultFields(vf), engine.DefaultOperators())
	resolver := engine.NewPathResolver(vf, "/")
	selector := engine.NewSelectExecutor(vf, resolver, compiler, engine.EmptySkipList())

	stmt, err := query.NewParser(compiler).Parse(mustLex(t, "files from 'tree' recursive limit 5"))
	if err != nil {
		t.Fatalf("Parse() = %v", err)
	}

	inner := &engine.SliceSink{}
	sink := engine.NewLimitSink(inner, stmt.Limit)
	if err := selector.Execute(context.Background(), stmt, sink); err != nil {
		t.Fatalf("Execute() = %v", err)
	}

	if got := len(inner.Rows); got != 5 {
		t.Fatalf("collected %d rows, want 5", got)
	}

	readDirs := counter.readDirs.Load()
	if readDirs > 3 {
		t.Errorf("ReadDir ran %d times over a 40-folder tree; limit must end the walk, not filter a finished list", readDirs)
	}
}

func TestLimitLargerThanTheResultsChangesNothing(t *testing.T) {
	counter := newCountingFS(deepFS(2, 3))
	vf := &fakeFileSystem{fsys: counter}

	compiler := engine.NewCompiler(engine.DefaultFields(vf), engine.DefaultOperators())
	resolver := engine.NewPathResolver(vf, "/")
	selector := engine.NewSelectExecutor(vf, resolver, compiler, engine.EmptySkipList())

	stmt, err := query.NewParser(compiler).Parse(mustLex(t, "files from 'tree' recursive limit 500"))
	if err != nil {
		t.Fatalf("Parse() = %v", err)
	}

	inner := &engine.SliceSink{}
	sink := engine.NewLimitSink(inner, stmt.Limit)
	if err := selector.Execute(context.Background(), stmt, sink); err != nil {
		t.Fatalf("Execute() = %v", err)
	}

	if got := len(inner.Rows); got != 6 {
		t.Errorf("collected %d rows, want all 6", got)
	}
	if sink.Filled() {
		t.Error("Filled() = true although the limit was never reached")
	}
}

func TestParseLimit(t *testing.T) {
	fsys := &fakeFileSystem{fsys: deepFS(1, 1)}
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"plain", "files from 'tree' limit 10", 10},
		{"after recursive", "files from 'tree' recursive limit 4", 4},
		{"after where", "files from 'tree' where type = 'txt' limit 7", 7},
		{"one", "files from 'tree' limit 1", 1},
		{"no limit", "files from 'tree'", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := parser.Parse(mustLex(t, tt.input))
			if err != nil {
				t.Fatalf("Parse(%q) = %v", tt.input, err)
			}
			if stmt.Limit != tt.want {
				t.Errorf("Limit = %d, want %d", stmt.Limit, tt.want)
			}
		})
	}
}

func TestParseLimitRejections(t *testing.T) {
	fsys := &fakeFileSystem{fsys: deepFS(1, 1)}
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		name  string
		input string
		kind  oerr.Kind
	}{
		{"zero", "files from 'tree' limit 0", oerr.KindLimitTooSmall},
		{"negative", "files from 'tree' limit -1", oerr.KindLimitTooSmall},
		{"not a number", "files from 'tree' limit abc", oerr.KindBadLimit},
		{"a decimal", "files from 'tree' limit 2.5", oerr.KindBadLimit},
		{"nothing after it", "files from 'tree' limit", oerr.KindMissingLimit},
		{"on a count", "count(files) from 'tree' limit 5", oerr.KindCountTakesNoLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(mustLex(t, tt.input))
			if !oerr.Is(err, tt.kind) {
				t.Errorf("Parse(%q) = %v, want %s", tt.input, err, tt.kind)
			}
		})
	}
}

func TestLimitMessages(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{oerr.MissingLimit(), `"limit" needs a number — for example: files from 'Documents' limit 10`},
		{oerr.BadLimit("abc"), `"limit" needs a whole number, not "abc" — for example: limit 10`},
		{oerr.LimitTooSmall(0), "A limit of 0 would show nothing. Use 1 or more, or leave the limit off to see everything."},
	}

	for _, tt := range tests {
		if got := tt.err.Error(); got != tt.want {
			t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
		}
	}

	if got, want := oerr.LimitReached(3), "Showing the first 3. Raise the limit to see more."; got != want {
		t.Errorf("LimitReached() = %q, want %q", got, want)
	}
}

func TestShellNotesWhenTheLimitBit(t *testing.T) {
	out := runShell(t, "files from 'work' limit 1\nexit\n", nil)

	if !strings.Contains(out, "Showing the first 1.") {
		t.Errorf("the shell should say the limit bit, got:\n%s", out)
	}
}

func TestShellStaysQuietWhenTheLimitDidNot(t *testing.T) {
	out := runShell(t, "files from 'work' limit 50\nexit\n", nil)

	if strings.Contains(out, "Showing the first") {
		t.Errorf("a limit that never bit must not be mentioned, got:\n%s", out)
	}
}
