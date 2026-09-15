package test

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/output"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/vfs"
)

func nestedFS() fstest.MapFS {
	return fstest.MapFS{
		"root/small/a.txt":         {Data: make([]byte, 100)},
		"root/small/b.txt":         {Data: make([]byte, 200)},
		"root/deep/one/two/c.txt":  {Data: make([]byte, 1000)},
		"root/deep/d.txt":          {Data: make([]byte, 500)},
		"root/empty/.keep":         {Data: []byte{}},
		"root/onlyfolders/inner/x": {Data: make([]byte, 7)},
		"root/top.txt":             {Data: make([]byte, 42)},
	}
}

func measureFolders(t *testing.T, names ...string) map[string]engine.Row {
	t.Helper()

	fsys := &fakeFileSystem{fsys: nestedFS()}
	rows := make([]engine.Row, 0, len(names))
	for _, name := range names {
		rows = append(rows, engine.Row{Name: name, IsDir: true})
	}

	sizer := engine.NewFolderSizer(fsys)
	if err := sizer.Measure(context.Background(), rows, func(r engine.Row) string {
		return path.Join("root", r.Name)
	}); err != nil {
		t.Fatalf("Measure() = %v", err)
	}

	byName := make(map[string]engine.Row, len(rows))
	for _, row := range rows {
		byName[row.Name] = row
	}
	return byName
}

func TestFolderSizeAddsUpEverythingBeneath(t *testing.T) {
	got := measureFolders(t, "small", "deep", "empty", "onlyfolders")

	tests := []struct {
		folder string
		want   int64
	}{
		{"small", 300},
		{"deep", 1500},
		{"empty", 0},
		{"onlyfolders", 7},
	}

	for _, tt := range tests {
		t.Run(tt.folder, func(t *testing.T) {
			row := got[tt.folder]
			if !row.SizeKnown {
				t.Fatalf("%s was not measured", tt.folder)
			}
			if row.Size != tt.want {
				t.Errorf("size = %d, want %d", row.Size, tt.want)
			}
		})
	}
}

func TestFolderSizeLeavesFilesAlone(t *testing.T) {
	fsys := &fakeFileSystem{fsys: nestedFS()}
	rows := []engine.Row{{Name: "top.txt", Size: 42}}

	sizer := engine.NewFolderSizer(fsys)
	if err := sizer.Measure(context.Background(), rows, func(r engine.Row) string {
		return path.Join("root", r.Name)
	}); err != nil {
		t.Fatalf("Measure() = %v", err)
	}

	if rows[0].Size != 42 {
		t.Errorf("a file's size changed to %d", rows[0].Size)
	}
	if !rows[0].SizeKnown {
		t.Error("a file's size is always known")
	}
}

func TestFolderSizeStopsWhenCancelled(t *testing.T) {
	fsys := &fakeFileSystem{fsys: nestedFS()}
	rows := []engine.Row{{Name: "deep", IsDir: true}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sizer := engine.NewFolderSizer(fsys)
	if err := sizer.Measure(ctx, rows, func(r engine.Row) string { return path.Join("root", r.Name) }); err == nil {
		t.Error("Measure() on a cancelled context returned no error")
	}
}

func TestFolderSizeDoesNotFollowSymlinks(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, "linked")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatalf("MkdirAll = %v", err)
	}

	outside := filepath.Join(root, "huge.bin")
	if err := os.WriteFile(outside, make([]byte, 5000), 0o644); err != nil {
		t.Fatalf("WriteFile = %v", err)
	}
	if err := os.WriteFile(filepath.Join(folder, "real.txt"), []byte("abc"), 0o644); err != nil {
		t.Fatalf("WriteFile = %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(folder, "link.bin")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	fsPath, err := vfs.FSPath(folder)
	if err != nil {
		t.Fatalf("FSPath = %v", err)
	}

	rows := []engine.Row{{Name: "linked", IsDir: true}}
	sizer := engine.NewFolderSizer(vfs.OS())
	if err := sizer.Measure(context.Background(), rows, func(engine.Row) string { return fsPath }); err != nil {
		t.Fatalf("Measure() = %v", err)
	}

	if rows[0].Size != 3 {
		t.Errorf("size = %d, want 3 — a symlink must not pull in its target's bytes", rows[0].Size)
	}
}

func TestFolderSizeReportsAnUnreadableFolder(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read anything")
	}

	root := t.TempDir()
	locked := filepath.Join(root, "locked")
	if err := os.MkdirAll(filepath.Join(locked, "inner"), 0o755); err != nil {
		t.Fatalf("MkdirAll = %v", err)
	}
	if err := os.WriteFile(filepath.Join(locked, "inner", "a.txt"), make([]byte, 10), 0o644); err != nil {
		t.Fatalf("WriteFile = %v", err)
	}
	if err := os.Chmod(filepath.Join(locked, "inner"), 0o000); err != nil {
		t.Fatalf("Chmod = %v", err)
	}
	t.Cleanup(func() { os.Chmod(filepath.Join(locked, "inner"), 0o755) })

	fsPath, err := vfs.FSPath(locked)
	if err != nil {
		t.Fatalf("FSPath = %v", err)
	}

	rows := []engine.Row{{Name: "locked", IsDir: true}}
	sizer := engine.NewFolderSizer(vfs.OS())
	if err := sizer.Measure(context.Background(), rows, func(engine.Row) string { return fsPath }); err != nil {
		t.Fatalf("Measure() = %v", err)
	}

	if rows[0].SizeKnown {
		t.Error("a folder osql could not fully read must not claim a size it does not know")
	}
}

func TestFourWorkersIsTheMeasuredOptimum(t *testing.T) {
	if engine.SizeWorkers != 4 {
		t.Errorf("SizeWorkers = %d, want 4 — 8 and 12 measured slower, so this is not a knob to turn", engine.SizeWorkers)
	}
}

func TestInParallelVisitsEveryIndexOnce(t *testing.T) {
	const count = 500
	seen := make([]int32, count)

	if err := engine.InParallel(context.Background(), count, func(i int) { seen[i]++ }); err != nil {
		t.Fatalf("InParallel() = %v", err)
	}

	for i, times := range seen {
		if times != 1 {
			t.Fatalf("index %d ran %d times, want once", i, times)
		}
	}
}

func TestInParallelHandlesAnEmptyList(t *testing.T) {
	if err := engine.InParallel(context.Background(), 0, func(int) { t.Error("nothing to do") }); err != nil {
		t.Errorf("InParallel() = %v", err)
	}
}

func TestUnmeasuredFolderStillRendersADash(t *testing.T) {
	if got := output.FormatRowSize(engine.Row{Name: "docs", IsDir: true}); got != output.Absent {
		t.Errorf("FormatRowSize() = %q, want %q", got, output.Absent)
	}
	measured := engine.Row{Name: "docs", IsDir: true, Size: 2048, SizeKnown: true}
	if got := output.FormatRowSize(measured); got != "2.0 KB" {
		t.Errorf("FormatRowSize() = %q, want 2.0 KB", got)
	}
}

func TestParseWithSizeOnFolders(t *testing.T) {
	fsys := &fakeFileSystem{fsys: nestedFS()}
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"folders", "folders from 'root' with size", true},
		{"all", "all from 'root' with size", true},
		{"recursive first", "folders from 'root' recursive with size", true},
		{"with a where", "folders from 'root' with size where name = 'deep'", true},
		{"with a sort", "folders from 'root' with size sorted by size desc", true},
		{"without it", "folders from 'root'", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := parser.Parse(mustLex(t, tt.input))
			if err != nil {
				t.Fatalf("Parse(%q) = %v", tt.input, err)
			}
			if stmt.WithSize != tt.want {
				t.Errorf("WithSize = %v, want %v", stmt.WithSize, tt.want)
			}
		})
	}
}

func TestParseWithSizeRejections(t *testing.T) {
	fsys := &fakeFileSystem{fsys: nestedFS()}
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		name  string
		input string
		kind  oerr.Kind
	}{
		{"files already have one", "files from 'root' with size", oerr.KindFilesAlreadyHaveSize},
		{"a count has no size", "count(folders) from 'root' with size", oerr.KindCountHasNoSize},
		{"with needs size", "folders from 'root' with", oerr.KindWithNeedsSize},
		{"with comes before where", "folders from 'root' where name = 'deep' with size", oerr.KindWithSizeComesFirst},
		{"sorting by size needs measuring", "folders from 'root' sorted by size desc", oerr.KindSortNeedsMeasuring},
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

func TestSortableFieldsGrowWhenFoldersAreMeasured(t *testing.T) {
	fsys := &fakeFileSystem{fsys: nestedFS()}
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())

	if err := compiler.ValidateSort("size", query.TargetFolders, true); err != nil {
		t.Errorf("with size, folders should sort by size, got %v", err)
	}
	if err := compiler.ValidateSort("size", query.TargetFolders, false); err == nil {
		t.Error("without measuring, folders must not claim to sort by size")
	}
	if err := compiler.ValidateSort("name", query.TargetFolders, false); err != nil {
		t.Errorf("name always sorts, got %v", err)
	}
}

func TestFoldersAreSizedOnlyAfterFiltering(t *testing.T) {
	counter := newCountingFS(nestedFS())
	vf := &fakeFileSystem{fsys: counter}

	compiler := engine.NewCompiler(engine.DefaultFields(vf), engine.DefaultOperators())
	resolver := engine.NewPathResolver(vf, "/")
	selector := engine.NewSelectExecutor(vf, resolver, compiler, engine.EmptySkipList())

	stmt, err := query.NewParser(compiler).Parse(mustLex(t, "folders from 'root' with size where name = 'small'"))
	if err != nil {
		t.Fatalf("Parse() = %v", err)
	}

	sink := &engine.SliceSink{}
	if err := selector.Execute(context.Background(), stmt, sink); err != nil {
		t.Fatalf("Execute() = %v", err)
	}

	names := make([]string, 0, len(sink.Rows))
	for _, row := range sink.Rows {
		names = append(names, row.Name)
	}
	if !slices.Equal(names, []string{"small"}) {
		t.Fatalf("the filter selected %v, want just small", names)
	}

	for _, row := range sink.Rows {
		if row.SizeKnown {
			t.Error("the scan measured a folder; sizing must happen after filtering, not during the walk")
		}
	}
}
