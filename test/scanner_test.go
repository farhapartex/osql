package test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"
	"time"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/vfs"
)

var fixtureModTime = time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)

func treeFS() fstest.MapFS {
	m := fstest.MapFS{
		"root/a.txt":                      {Data: []byte("a")},
		"root/b.log":                      {Data: []byte("bb")},
		"root/Makefile":                   {Data: []byte("m")},
		"root/sub/c.txt":                  {Data: []byte("ccc")},
		"root/sub/d.log":                  {Data: []byte("dddd")},
		"root/sub/deep/e.txt":             {Data: []byte("eeeee")},
		"root/sub/deep/deeper/f.txt":      {Data: []byte("f")},
		"root/node_modules/pkg/g.txt":     {Data: []byte("g")},
		"root/.git/objects/h":             {Data: []byte("h")},
		"root/empty_dir/.keep":            {Data: []byte("")},
		"elsewhere/should_not_appear.txt": {Data: []byte("x")},
	}
	for _, f := range m {
		f.ModTime = fixtureModTime
	}
	return m
}

func scanNames(t *testing.T, fsys fstest.MapFS, path string, opts engine.ScanOptions) []string {
	t.Helper()

	vf := &fakeFileSystem{fsys: fsys}
	resolver := engine.NewPathResolver(vf, "/")
	resolved, err := resolver.Resolve("/" + path)
	if err != nil {
		t.Fatalf("Resolve(%q) error = %v", path, err)
	}

	sink := &sliceSink{}
	if err := engine.NewScanner(vf).Scan(context.Background(), resolved, opts, sink); err != nil {
		t.Fatalf("Scan error = %v", err)
	}

	names := make([]string, 0, len(sink.rows))
	for _, r := range sink.rows {
		names = append(names, r.Name)
	}
	slices.Sort(names)
	return names
}

func TestScanDepthOneListsDirectChildrenOnly(t *testing.T) {
	got := scanNames(t, treeFS(), "root", engine.ScanOptions{
		MaxDepth: 1,
		Target:   query.TargetAll,
		Skip:     engine.EmptySkipList(),
	})

	want := []string{".git", "Makefile", "a.txt", "b.log", "empty_dir", "node_modules", "sub"}
	if !slices.Equal(got, want) {
		t.Errorf("depth 1 listing = %v\nwant %v", got, want)
	}
}

func TestScanUnlimitedDepthWalksWholeSubtree(t *testing.T) {
	got := scanNames(t, treeFS(), "root", engine.ScanOptions{
		MaxDepth: engine.DepthUnlimited,
		Target:   query.TargetFiles,
		Skip:     engine.EmptySkipList(),
	})

	want := []string{
		".git/objects/h",
		"Makefile",
		"a.txt",
		"b.log",
		"empty_dir/.keep",
		"node_modules/pkg/g.txt",
		"sub/c.txt",
		"sub/d.log",
		"sub/deep/deeper/f.txt",
		"sub/deep/e.txt",
	}
	if !slices.Equal(got, want) {
		t.Errorf("recursive listing = %v\nwant %v", got, want)
	}
}

func TestScanRecursiveNamesAreRelativeToTheRoot(t *testing.T) {
	got := scanNames(t, treeFS(), "root", engine.ScanOptions{
		Target: query.TargetFiles,
		Skip:   engine.EmptySkipList(),
	})

	for _, name := range got {
		if filepath.IsAbs(name) {
			t.Errorf("name %q is absolute; recursive rows carry a relative path", name)
		}
		if len(name) > 5 && name[:5] == "root/" {
			t.Errorf("name %q still carries the root prefix", name)
		}
	}
}

func TestScanIntermediateDepths(t *testing.T) {
	tests := []struct {
		depth int
		want  []string
	}{
		{1, []string{"a.txt", "b.log", "Makefile"}},
		{2, []string{"a.txt", "b.log", "Makefile", "sub/c.txt", "sub/d.log", "empty_dir/.keep"}},
		{3, []string{"a.txt", "b.log", "Makefile", "sub/c.txt", "sub/d.log", "empty_dir/.keep", ".git/objects/h", "node_modules/pkg/g.txt", "sub/deep/e.txt"}},
	}

	for _, tt := range tests {
		got := scanNames(t, treeFS(), "root", engine.ScanOptions{
			MaxDepth: tt.depth,
			Target:   query.TargetFiles,
			Skip:     engine.EmptySkipList(),
		})
		want := slices.Clone(tt.want)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("depth %d = %v\nwant %v", tt.depth, got, want)
		}
	}
}

func TestScanTargetFiltering(t *testing.T) {
	tests := []struct {
		target query.Target
		want   []string
	}{
		{query.TargetFiles, []string{"Makefile", "a.txt", "b.log"}},
		{query.TargetFolders, []string{".git", "empty_dir", "node_modules", "sub"}},
		{query.TargetAll, []string{".git", "Makefile", "a.txt", "b.log", "empty_dir", "node_modules", "sub"}},
	}

	for _, tt := range tests {
		t.Run(tt.target.String(), func(t *testing.T) {
			got := scanNames(t, treeFS(), "root", engine.ScanOptions{
				MaxDepth: 1,
				Target:   tt.target,
				Skip:     engine.EmptySkipList(),
			})
			if !slices.Equal(got, tt.want) {
				t.Errorf("target %v = %v\nwant %v", tt.target, got, tt.want)
			}
		})
	}
}

func TestScanSkipListPrunesSubtreesButKeepsTheDirectory(t *testing.T) {
	got := scanNames(t, treeFS(), "root", engine.ScanOptions{
		Target: query.TargetAll,
		Skip:   engine.DefaultSkipList(),
	})

	if slices.Contains(got, "node_modules") {
		t.Error("a skipped directory should not be emitted")
	}
	for _, name := range got {
		if len(name) >= 12 && name[:12] == "node_modules" {
			t.Errorf("skip-list did not prune the subtree: %q", name)
		}
		if len(name) >= 4 && name[:4] == ".git" {
			t.Errorf("skip-list did not prune .git: %q", name)
		}
	}
	if !slices.Contains(got, "sub/deep/e.txt") {
		t.Error("skip-list pruned an unrelated subtree")
	}
}

func TestScanSkipListByPath(t *testing.T) {
	fsys := fstest.MapFS{
		"System/Library/a.txt": {Data: []byte("a")},
		"Users/me/b.txt":       {Data: []byte("b")},
		"src/dev/keep.txt":     {Data: []byte("c")},
	}

	got := scanNames(t, fsys, ".", engine.ScanOptions{
		Target: query.TargetFiles,
		Skip:   engine.DefaultSkipList(),
	})

	if slices.Contains(got, "System/Library/a.txt") {
		t.Error("System was not pruned by path")
	}
	if !slices.Contains(got, "src/dev/keep.txt") {
		t.Error("a directory named dev deep in a project must not be pruned; only the top-level /dev is")
	}
	if !slices.Contains(got, "Users/me/b.txt") {
		t.Error("an unrelated path was pruned")
	}
}

func TestScanAppliesMatchers(t *testing.T) {
	fsys := treeFS()
	vf := &fakeFileSystem{fsys: fsys}
	compiler := engine.NewCompiler(engine.DefaultFields(vf), engine.DefaultOperators())

	matchers, err := compiler.CompileAll([]query.Predicate{
		{Field: "type", Op: "=", Value: "txt"},
	}, query.TargetFiles)
	if err != nil {
		t.Fatal(err)
	}

	got := scanNames(t, fsys, "root", engine.ScanOptions{
		Target:   query.TargetFiles,
		Matchers: matchers,
		Skip:     engine.DefaultSkipList(),
	})

	want := []string{"sub/c.txt", "sub/deep/deeper/f.txt", "sub/deep/e.txt", "a.txt"}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("type = txt matched %v\nwant %v", got, want)
	}
}

func TestScanPopulatesRowFields(t *testing.T) {
	fsys := treeFS()
	vf := &fakeFileSystem{fsys: fsys}
	resolver := engine.NewPathResolver(vf, "/")
	resolved, _ := resolver.Resolve("/root")

	sink := &sliceSink{}
	err := engine.NewScanner(vf).Scan(context.Background(), resolved, engine.ScanOptions{
		MaxDepth: 1,
		Target:   query.TargetAll,
		Skip:     engine.EmptySkipList(),
	}, sink)
	if err != nil {
		t.Fatal(err)
	}

	byName := map[string]engine.Row{}
	for _, r := range sink.rows {
		byName[r.Name] = r
	}

	file := byName["b.log"]
	if file.IsDir {
		t.Error("b.log marked as a directory")
	}
	if file.Ext != "log" {
		t.Errorf("Ext = %q, want \"log\"", file.Ext)
	}
	if file.Size != 2 {
		t.Errorf("Size = %d, want 2", file.Size)
	}
	if !file.Modified.Equal(fixtureModTime) {
		t.Errorf("Modified = %v, want %v; Info() must be read for a matched row", file.Modified, fixtureModTime)
	}

	dir := byName["sub"]
	if !dir.IsDir {
		t.Error("sub not marked as a directory")
	}
	if dir.Ext != "" {
		t.Errorf("directory Ext = %q, want empty", dir.Ext)
	}

	noExt := byName["Makefile"]
	if noExt.Ext != "" {
		t.Errorf("Makefile Ext = %q, want empty", noExt.Ext)
	}
}

func TestScanStopsOnSinkLimit(t *testing.T) {
	fsys := treeFS()
	vf := &fakeFileSystem{fsys: fsys}
	resolver := engine.NewPathResolver(vf, "/")
	resolved, _ := resolver.Resolve("/root")

	sink := &sliceSink{limit: 3}
	err := engine.NewScanner(vf).Scan(context.Background(), resolved, engine.ScanOptions{
		Target: query.TargetAll,
		Skip:   engine.EmptySkipList(),
	}, sink)

	if err != nil {
		t.Fatalf("Scan returned %v; ErrStopWalk is normal termination, not a failure", err)
	}
	if len(sink.rows) != 3 {
		t.Errorf("collected %d rows, want 3", len(sink.rows))
	}
}

func TestScanRespectsContextCancellation(t *testing.T) {
	fsys := treeFS()
	vf := &fakeFileSystem{fsys: fsys}
	resolver := engine.NewPathResolver(vf, "/")
	resolved, _ := resolver.Resolve("/root")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := engine.NewScanner(vf).Scan(ctx, resolved, engine.ScanOptions{
		Target: query.TargetAll,
		Skip:   engine.EmptySkipList(),
	}, &sliceSink{})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Scan error = %v, want context.Canceled", err)
	}
}

func TestScanEmptyDirectory(t *testing.T) {
	fsys := fstest.MapFS{"empty/.keep": {Data: []byte("")}}

	got := scanNames(t, fsys, "empty", engine.ScanOptions{
		MaxDepth: 1,
		Target:   query.TargetFolders,
		Skip:     engine.EmptySkipList(),
	})

	if len(got) != 0 {
		t.Errorf("got %v, want no folders", got)
	}
}

func TestScanRootIsNeverEmitted(t *testing.T) {
	got := scanNames(t, treeFS(), "root", engine.ScanOptions{
		Target: query.TargetAll,
		Skip:   engine.EmptySkipList(),
	})

	for _, name := range got {
		if name == "root" || name == "." || name == "" {
			t.Errorf("the searched folder itself was emitted as %q", name)
		}
	}
}

func TestScanUnreadableSubtreeIsSkippedNotFatal(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root, permission cannot be denied")
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "visible.txt"), []byte("v"), 0o644); err != nil {
		t.Fatal(err)
	}
	locked := filepath.Join(root, "locked")
	if err := os.MkdirAll(filepath.Join(locked, "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "hidden.txt"), []byte("h"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })

	vf := vfs.OS()
	resolver := engine.NewPathResolver(vf, root)
	resolved, err := resolver.Resolve(".")
	if err != nil {
		t.Fatal(err)
	}

	sink := &sliceSink{}
	err = engine.NewScanner(vf).Scan(context.Background(), resolved, engine.ScanOptions{
		Target: query.TargetFiles,
		Skip:   engine.EmptySkipList(),
	}, sink)

	if err != nil {
		t.Fatalf("an unreadable subtree must be skipped, not fatal; got %v", err)
	}

	found := false
	for _, r := range sink.rows {
		if r.Name == "visible.txt" {
			found = true
		}
		if r.Name == "locked/hidden.txt" {
			t.Error("read a file inside an unreadable directory")
		}
	}
	if !found {
		t.Error("the walk aborted before reaching a readable sibling")
	}
}

func TestScanUnreadableRootIsAnError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root, permission cannot be denied")
	}

	root := t.TempDir()
	locked := filepath.Join(root, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })

	vf := vfs.OS()
	resolver := engine.NewPathResolver(vf, root)
	resolved, err := resolver.Resolve("locked")
	if err != nil {
		t.Skipf("resolver rejected the locked root first: %v", err)
	}

	err = engine.NewScanner(vf).Scan(context.Background(), resolved, engine.ScanOptions{
		Target: query.TargetFiles,
		Skip:   engine.EmptySkipList(),
	}, &sliceSink{})

	if err == nil {
		t.Fatal("scanning an unreadable root succeeded")
	}
	if !oerr.Is(err, oerr.KindNoPermission) {
		t.Errorf("error = %v, want no_permission", err)
	}
}

func TestExtensionOf(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"notes.txt", "txt"},
		{"archive.tar.gz", "gz"},
		{"Makefile", ""},
		{".gitignore", "gitignore"},
		{"a.", ""},
		{"", ""},
		{"report.PDF", "PDF"},
	}

	for _, tt := range tests {
		if got := engine.ExtensionOf(tt.name); got != tt.want {
			t.Errorf("ExtensionOf(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestSkipListLookups(t *testing.T) {
	s := engine.DefaultSkipList()

	for _, name := range []string{".git", "node_modules", ".Trash"} {
		if !s.SkipsName(name) {
			t.Errorf("SkipsName(%q) = false", name)
		}
	}
	for _, name := range []string{"src", "Documents", "dev", "sys"} {
		if s.SkipsName(name) {
			t.Errorf("SkipsName(%q) = true; only top-level paths skip those", name)
		}
	}
	for _, path := range []string{"System", "Volumes", "proc", "dev"} {
		if !s.SkipsPath(path) {
			t.Errorf("SkipsPath(%q) = false", path)
		}
	}
	if s.SkipsPath("src/dev") {
		t.Error("SkipsPath(\"src/dev\") = true; only the exact top-level path is skipped")
	}

	empty := engine.EmptySkipList()
	if empty.SkipsName(".git") || empty.SkipsPath("System") {
		t.Error("EmptySkipList skips something")
	}
}
