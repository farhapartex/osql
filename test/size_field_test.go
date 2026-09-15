package test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
)

func TestParseSizeAcceptsEveryUnit(t *testing.T) {
	const kb = 1024

	tests := []struct {
		in   string
		want int64
	}{
		{"0", 0},
		{"100", 100},
		{"100b", 100},
		{"100B", 100},
		{"1kb", kb},
		{"1KB", kb},
		{"1Kb", kb},
		{"10kb", 10 * kb},
		{"1mb", kb * kb},
		{"1gb", kb * kb * kb},
		{"1tb", kb * kb * kb * kb},
		{"1.5mb", 1572864},
		{"0.5kb", 512},
		{"  20kb  ", 20 * kb},
		{"20 kb", 20 * kb},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := engine.ParseSize(tt.in)
			if err != nil {
				t.Fatalf("ParseSize(%q) = %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ParseSize(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseSizeRejectsNonsense(t *testing.T) {
	tests := []struct {
		name string
		in   string
		kind oerr.Kind
	}{
		{"empty", "", oerr.KindBadSizeValue},
		{"only spaces", "   ", oerr.KindBadSizeValue},
		{"unknown unit", "10xb", oerr.KindBadSizeValue},
		{"unit with no number", "mb", oerr.KindBadSizeValue},
		{"negative", "-5", oerr.KindBadSizeValue},
		{"negative with unit", "-5kb", oerr.KindBadSizeValue},
		{"two decimal points", "1.2.3", oerr.KindBadSizeValue},
		{"words", "big", oerr.KindBadSizeValue},
		{"number then words", "10 apples", oerr.KindBadSizeValue},
		{"overflow", "99999999tb", oerr.KindSizeTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.ParseSize(tt.in)
			if !oerr.Is(err, tt.kind) {
				t.Errorf("ParseSize(%q) = %v, want %s", tt.in, err, tt.kind)
			}
		})
	}
}

func TestSizeFieldShape(t *testing.T) {
	field := engine.SizeField{}

	if field.Field() != "size" {
		t.Errorf("Field() = %q", field.Field())
	}
	if field.Cost() <= engine.CostFree {
		t.Errorf("Cost() = %d; size needs a stat so it must cost more than a free field", field.Cost())
	}
	if field.Cost() >= engine.CostReadDir {
		t.Errorf("Cost() = %d; a stat is cheaper than a ReadDir", field.Cost())
	}

	for _, op := range []string{"=", "!=", "<", ">", "<=", ">="} {
		if !slices.Contains(field.AllowedOperators(), op) {
			t.Errorf("size should accept %q", op)
		}
	}
}

func TestSizeAppliesOnlyWhereAFileHasOne(t *testing.T) {
	field := engine.SizeField{}

	tests := []struct {
		target query.Target
		want   bool
	}{
		{query.TargetFiles, true},
		{query.TargetAll, true},
		{query.TargetFolders, false},
		{query.TargetApps, false},
	}

	for _, tt := range tests {
		if got := field.AppliesTo(tt.target); got != tt.want {
			t.Errorf("AppliesTo(%v) = %v, want %v", tt.target, got, tt.want)
		}
	}
}

func sizedFS() fstest.MapFS {
	return fstest.MapFS{
		"work/tiny.txt":     {Data: []byte("ab")},
		"work/small.txt":    {Data: make([]byte, 500)},
		"work/big.txt":      {Data: make([]byte, 4096)},
		"work/huge.log":     {Data: make([]byte, 40960)},
		"work/nested/x.txt": {Data: make([]byte, 2048)},
	}
}

func sizeQueryNames(t *testing.T, q string) []string {
	t.Helper()

	fsys := &fakeFileSystem{fsys: sizedFS()}
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())
	resolver := engine.NewPathResolver(fsys, "/")
	selector := engine.NewSelectExecutor(fsys, resolver, compiler, engine.EmptySkipList())

	stmt, err := query.NewParser(compiler).Parse(mustLex(t, q))
	if err != nil {
		t.Fatalf("Parse(%q) = %v", q, err)
	}

	sink := &engine.SliceSink{}
	if err := selector.Execute(context.Background(), stmt, sink); err != nil {
		t.Fatalf("Execute(%q) = %v", q, err)
	}

	names := make([]string, 0, len(sink.Rows))
	for _, row := range sink.Rows {
		names = append(names, row.Name)
	}
	return names
}

func TestSizeFiltersFiles(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"greater than", "files from 'work' where size > 1kb", []string{"big.txt", "huge.log"}},
		{"less than", "files from 'work' where size < 1kb", []string{"small.txt", "tiny.txt"}},
		{"at least", "files from 'work' where size >= 4kb", []string{"big.txt", "huge.log"}},
		{"at most", "files from 'work' where size <= 500", []string{"small.txt", "tiny.txt"}},
		{"exactly", "files from 'work' where size = 4096", []string{"big.txt"}},
		{"not equal", "files from 'work' where size != 2", []string{"big.txt", "huge.log", "small.txt"}},
		{"recursive", "files from 'work' recursive where size = 2kb", []string{"nested/x.txt"}},
		{"nothing that big", "files from 'work' where size > 1gb", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sizeQueryNames(t, tt.query)
			if !slices.Equal(got, tt.want) {
				t.Errorf("%s\n got: %v\nwant: %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestSizeTreatsFoldersAsEmptyUnderAll(t *testing.T) {
	got := sizeQueryNames(t, "all from 'work' where size > 0")

	for _, name := range got {
		if name == "nested" {
			t.Error("a folder was reported as having a size; folders wait for Task 24")
		}
	}
	if len(got) == 0 {
		t.Error("no files came back at all")
	}
}

func TestSizeIsRejectedForFolders(t *testing.T) {
	fsys := &fakeFileSystem{fsys: sizedFS()}
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())

	_, err := query.NewParser(compiler).Parse(mustLex(t, "folders from 'work' where size > 1kb"))
	if err == nil {
		t.Fatal("Parse() accepted size on folders")
	}
	if !strings.Contains(err.Error(), "size") {
		t.Errorf("the message should name the field, got: %s", err)
	}
}

func TestSizeNeverStatsAnEntryACheaperFieldAlreadyRejected(t *testing.T) {
	counter := newCountingFS(sizedFS())

	vf := &fakeFileSystem{fsys: counter}
	compiler := engine.NewCompiler(engine.DefaultFields(vf), engine.DefaultOperators())
	resolver := engine.NewPathResolver(vf, "/")
	selector := engine.NewSelectExecutor(vf, resolver, compiler, engine.EmptySkipList())

	stmt, err := query.NewParser(compiler).Parse(mustLex(t, "files from 'work' where type = 'log' and size > 1kb"))
	if err != nil {
		t.Fatalf("Parse() = %v", err)
	}

	sink := &engine.SliceSink{}
	if err := selector.Execute(context.Background(), stmt, sink); err != nil {
		t.Fatalf("Execute() = %v", err)
	}

	if len(sink.Rows) != 1 || sink.Rows[0].Name != "huge.log" {
		t.Fatalf("matched %v, want just huge.log", sink.Rows)
	}

	statted := counter.infos.Load()
	if statted > 2 {
		t.Errorf("Info() ran %d times; only the one .log file should be stat-ed, so type must be evaluated before size", statted)
	}
}
