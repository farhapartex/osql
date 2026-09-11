package test

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
)

func pushAll(t *testing.T, sink *engine.SortSink, rows []engine.Row) {
	t.Helper()

	for _, row := range rows {
		if err := sink.Push(row); err != nil {
			t.Fatalf("Push(%s) = %v", row.Name, err)
		}
	}
}

func namesOf(rows []engine.Row) []string {
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.Name)
	}
	return names
}

func sizedRows() []engine.Row {
	return []engine.Row{
		{Name: "medium.txt", Size: 500},
		{Name: "huge.txt", Size: 9000},
		{Name: "tiny.txt", Size: 1},
		{Name: "big.txt", Size: 4000},
		{Name: "small.txt", Size: 50},
	}
}

func TestSortAscendingAndDescending(t *testing.T) {
	tests := []struct {
		name       string
		descending bool
		want       []string
	}{
		{"ascending", false, []string{"tiny.txt", "small.txt", "medium.txt", "big.txt", "huge.txt"}},
		{"descending", true, []string{"huge.txt", "big.txt", "medium.txt", "small.txt", "tiny.txt"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := engine.NewSortSink(engine.SizeField{}, tt.descending, 0)
			pushAll(t, sink, sizedRows())

			if got := namesOf(sink.Rows()); !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortKeepsTheTopNRegardlessOfArrivalOrder(t *testing.T) {
	sink := engine.NewSortSink(engine.SizeField{}, true, 2)
	pushAll(t, sink, sizedRows())

	want := []string{"huge.txt", "big.txt"}
	if got := namesOf(sink.Rows()); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v — the heap decides membership, not arrival order", got, want)
	}
}

func TestSortKeepsTheBottomNWhenAscending(t *testing.T) {
	sink := engine.NewSortSink(engine.SizeField{}, false, 2)
	pushAll(t, sink, sizedRows())

	want := []string{"tiny.txt", "small.txt"}
	if got := namesOf(sink.Rows()); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSortBreaksTiesByName(t *testing.T) {
	rows := []engine.Row{
		{Name: "zebra.txt", Size: 100},
		{Name: "apple.txt", Size: 100},
		{Name: "Mango.txt", Size: 100},
	}

	sink := engine.NewSortSink(engine.SizeField{}, true, 0)
	pushAll(t, sink, rows)

	want := []string{"apple.txt", "Mango.txt", "zebra.txt"}
	if got := namesOf(sink.Rows()); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v — equal keys must order by name so output is reproducible", got, want)
	}
}

func TestSortByNameAndByType(t *testing.T) {
	rows := []engine.Row{
		{Name: "b.md", Ext: "md"},
		{Name: "a.txt", Ext: "txt"},
		{Name: "c.go", Ext: "go"},
		{Name: "folder", IsDir: true},
	}

	byName := engine.NewSortSink(engine.NameField{}, false, 0)
	pushAll(t, byName, rows)
	if got, want := namesOf(byName.Rows()), []string{"a.txt", "b.md", "c.go", "folder"}; !slices.Equal(got, want) {
		t.Errorf("by name: got %v, want %v", got, want)
	}

	byType := engine.NewSortSink(engine.TypeField{}, false, 0)
	pushAll(t, byType, rows)
	if got, want := namesOf(byType.Rows()), []string{"folder", "c.go", "b.md", "a.txt"}; !slices.Equal(got, want) {
		t.Errorf("by type: got %v, want %v", got, want)
	}
}

func TestSortByModified(t *testing.T) {
	rows := []engine.Row{
		{Name: "new.txt", Modified: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
		{Name: "old.txt", Modified: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Name: "mid.txt", Modified: time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)},
	}

	sink := engine.NewSortSink(engine.NewModifiedField(nil), true, 0)
	pushAll(t, sink, rows)

	want := []string{"new.txt", "mid.txt", "old.txt"}
	if got := namesOf(sink.Rows()); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSortOverflowsOnlyPastItsCap(t *testing.T) {
	small := engine.NewSortSink(engine.SizeField{}, true, 3)
	pushAll(t, small, sizedRows())
	if !small.Overflowed() {
		t.Error("five rows into a heap of three should overflow")
	}

	roomy := engine.NewSortSink(engine.SizeField{}, true, 50)
	pushAll(t, roomy, sizedRows())
	if roomy.Overflowed() {
		t.Error("five rows into a heap of fifty must not overflow")
	}
}

func TestSortWithNoLimitHoldsAtMostTheCap(t *testing.T) {
	sink := engine.NewSortSink(engine.SizeField{}, true, 0)

	rows := make([]engine.Row, 0, engine.SortCap+50)
	for i := 0; i < engine.SortCap+50; i++ {
		rows = append(rows, engine.Row{Name: "f" + strconv.Itoa(i) + ".txt", Size: int64(i)})
	}
	pushAll(t, sink, rows)

	ordered := sink.Rows()
	if len(ordered) != engine.SortCap {
		t.Fatalf("held %d rows, want the cap of %d", len(ordered), engine.SortCap)
	}
	if !sink.Overflowed() {
		t.Error("Overflowed() = false although rows were dropped")
	}
	if ordered[0].Size != int64(engine.SortCap+49) {
		t.Errorf("the biggest row is %d; a capped sort must still keep the real top", ordered[0].Size)
	}
}

func TestSortRunsTheWholeWalkUnlikeLimit(t *testing.T) {
	counter := newCountingFS(deepFS(20, 5))
	vf := &fakeFileSystem{fsys: counter}

	compiler := engine.NewCompiler(engine.DefaultFields(vf), engine.DefaultOperators())
	resolver := engine.NewPathResolver(vf, "/")
	selector := engine.NewSelectExecutor(vf, resolver, compiler, engine.EmptySkipList())

	stmt, err := query.NewParser(compiler).Parse(mustLex(t, "files from 'tree' recursive sorted by size desc limit 2"))
	if err != nil {
		t.Fatalf("Parse() = %v", err)
	}

	sink := engine.NewSortSink(engine.SizeField{}, stmt.SortDesc, stmt.Limit)
	if err := selector.Execute(context.Background(), stmt, sink); err != nil {
		t.Fatalf("Execute() = %v", err)
	}

	if got := len(sink.Rows()); got != 2 {
		t.Fatalf("kept %d rows, want 2", got)
	}
	if readDirs := counter.readDirs.Load(); readDirs < 20 {
		t.Errorf("ReadDir ran %d times; a sort must see every candidate before it knows the top", readDirs)
	}
}

func TestParseSortedBy(t *testing.T) {
	fsys := &fakeFileSystem{fsys: sizedFS()}
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		name      string
		input     string
		wantField string
		wantDesc  bool
		wantLimit int
	}{
		{"ascending by default", "files from 'work' sorted by size", "size", false, 0},
		{"descending", "files from 'work' sorted by size desc", "size", true, 0},
		{"explicit ascending", "files from 'work' sorted by size asc", "size", false, 0},
		{"with a limit", "files from 'work' sorted by modified desc limit 5", "modified", true, 5},
		{"after where", "files from 'work' where type = 'txt' sorted by name", "name", false, 0},
		{"after recursive", "files from 'work' recursive sorted by name desc", "name", true, 0},
		{"no sort", "files from 'work'", "", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := parser.Parse(mustLex(t, tt.input))
			if err != nil {
				t.Fatalf("Parse(%q) = %v", tt.input, err)
			}
			if stmt.SortField != tt.wantField {
				t.Errorf("SortField = %q, want %q", stmt.SortField, tt.wantField)
			}
			if stmt.SortDesc != tt.wantDesc {
				t.Errorf("SortDesc = %v, want %v", stmt.SortDesc, tt.wantDesc)
			}
			if stmt.Limit != tt.wantLimit {
				t.Errorf("Limit = %d, want %d", stmt.Limit, tt.wantLimit)
			}
		})
	}
}

func TestParseSortedByRejections(t *testing.T) {
	fsys := &fakeFileSystem{fsys: sizedFS()}
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		name  string
		input string
		kind  oerr.Kind
	}{
		{"no by", "files from 'work' sorted", oerr.KindMissingSortBy},
		{"wrong word after sorted", "files from 'work' sorted size", oerr.KindMissingSortBy},
		{"no field", "files from 'work' sorted by", oerr.KindMissingSortField},
		{"a pattern field", "files from 'work' sorted by name_like", oerr.KindUnsortableField},
		{"an unknown field", "files from 'work' sorted by colour", oerr.KindUnsortableField},
		{"child count is not sortable", "folders from 'work' sorted by count(child)", oerr.KindUnsortableField},
		{"folders need measuring first", "folders from 'work' sorted by size", oerr.KindSortNeedsMeasuring},
		{"apps cannot sort", "apps sorted by name", oerr.KindAppsNotSortable},
		{"a count cannot sort", "count(files) from 'work' sorted by size", oerr.KindCountTakesNoSort},
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

func TestUnsortableFieldNamesWhatWorks(t *testing.T) {
	fsys := &fakeFileSystem{fsys: sizedFS()}
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())

	_, err := query.NewParser(compiler).Parse(mustLex(t, "files from 'work' sorted by name_like"))
	if err == nil {
		t.Fatal("Parse() accepted an unsortable field")
	}

	want := `I can't sort by "name_like". I can sort by name, type, size, and modified.`
	if err.Error() != want {
		t.Errorf("got:\n%s\nwant:\n%s", err, want)
	}
}

func TestSortTruncatedMessage(t *testing.T) {
	want := "Sorted the first 10000 matches. Add a limit to be sure you are seeing the top."
	if got := oerr.SortTruncated(engine.SortCap); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestShellSortsAndSaysNothingWhenALimitMakesItExact(t *testing.T) {
	out := runShell(t, "files from 'work' sorted by name desc limit 1\nexit\n", nil)

	if strings.Contains(out, "Showing the first") {
		t.Errorf("a sorted query with a limit is exact, so the limit note must not appear:\n%s", out)
	}
	if strings.Contains(out, "Sorted the first") {
		t.Errorf("nothing was truncated:\n%s", out)
	}
}
