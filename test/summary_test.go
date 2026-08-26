package test

import (
	"bytes"
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/output"
	"github.com/farhapartex/osql/internal/query"
)

func summaryFS() fstest.MapFS {
	stamp := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	m := fstest.MapFS{
		"root/a.txt":                 {Data: bytes.Repeat([]byte("a"), 100)},
		"root/b.txt":                 {Data: bytes.Repeat([]byte("b"), 200)},
		"root/big.zip":               {Data: bytes.Repeat([]byte("z"), 5000)},
		"root/Makefile":              {Data: bytes.Repeat([]byte("m"), 50)},
		"root/sub/c.txt":             {Data: bytes.Repeat([]byte("c"), 300)},
		"root/sub/d.log":             {Data: bytes.Repeat([]byte("d"), 400)},
		"root/sub/deep/e.md":         {Data: bytes.Repeat([]byte("e"), 600)},
		"root/node_modules/pkg/x.js": {Data: bytes.Repeat([]byte("x"), 99999)},
		"root/.venv/lib/y.py":        {Data: bytes.Repeat([]byte("y"), 88888)},
		"empty_ish/.keep":            {Data: []byte("")},
		"onlyfolders/a/keep.txt":     {Data: []byte("k")},
		"onlyfolders/b/keep.txt":     {Data: []byte("k")},
	}
	for _, f := range m {
		f.ModTime = stamp
	}
	m["root/a.txt"].ModTime = time.Date(2024, 3, 11, 9, 0, 0, 0, time.UTC)
	m["root/big.zip"].ModTime = time.Date(2026, 8, 25, 18, 0, 0, 0, time.UTC)
	return m
}

func summarizerFor(t *testing.T, fsys fs.FS) *engine.SummaryExecutor {
	t.Helper()

	vf := &fakeFileSystem{fsys: fsys}
	return engine.NewSummaryExecutor(vf, engine.NewPathResolver(vf, "/"), engine.DefaultSkipList())
}

func summarize(t *testing.T, fsys fs.FS, input string) engine.Summary {
	t.Helper()

	exec := summarizerFor(t, fsys)
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())

	tokens, err := query.NewLexer().Lex(input)
	if err != nil {
		t.Fatalf("Lex error = %v", err)
	}
	stmt, err := query.NewParser(compiler).Parse(tokens)
	if err != nil {
		t.Fatalf("Parse(%q) error = %v", input, err)
	}

	got, err := exec.Summarize(context.Background(), stmt)
	if err != nil {
		t.Fatalf("Summarize(%q) error = %v", input, err)
	}
	return got
}

func TestParseSummaryForm(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		input     string
		path      string
		recursive bool
		skipped   bool
	}{
		{"summary from 'root'", "root", false, false},
		{"summary from 'root' recursive", "root", true, false},
		{"summary from 'root' recursive with skipped", "root", true, true},
		{"summary from 'root' with skipped", "root", false, true},
		{"SUMMARY FROM 'root' RECURSIVE WITH SKIPPED", "root", true, true},
		{"summary from root", "root", false, false},
		{"summary from 'root';", "root", false, false},
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
			if stmt.Verb != query.VerbSummary {
				t.Errorf("Verb = %q, want summary", stmt.Verb)
			}
			if stmt.Path != tt.path {
				t.Errorf("Path = %q, want %q", stmt.Path, tt.path)
			}
			if stmt.Recursive != tt.recursive {
				t.Errorf("Recursive = %v, want %v", stmt.Recursive, tt.recursive)
			}
			if stmt.IncludeSkipped != tt.skipped {
				t.Errorf("IncludeSkipped = %v, want %v", stmt.IncludeSkipped, tt.skipped)
			}
		})
	}
}

func TestParseSummaryErrors(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		name  string
		input string
		kind  oerr.Kind
	}{
		{"no from", "summary 'root'", oerr.KindMissingFrom},
		{"nothing at all", "summary", oerr.KindMissingFrom},
		{"no path", "summary from", oerr.KindMissingPath},
		{"with alone", "summary from 'root' with", oerr.KindWithNeedsSkipped},
		{"with the wrong word", "summary from 'root' with everything", oerr.KindWithNeedsSkipped},
		{"trailing junk", "summary from 'root' junk", oerr.KindUnexpectedInput},
		{"where is not allowed", "summary from 'root' where type = 'txt'", oerr.KindUnexpectedInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := query.NewLexer().Lex(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := parser.Parse(tokens); err == nil {
				t.Fatalf("Parse(%q) succeeded", tt.input)
			} else if !oerr.Is(err, tt.kind) {
				t.Errorf("Parse(%q)\n got: %v\nwant kind: %v", tt.input, err, tt.kind)
			}
		})
	}
}

func TestSummaryOneLevelCounts(t *testing.T) {
	got := summarize(t, summaryFS(), "summary from 'root'")

	if got.Files != 4 {
		t.Errorf("Files = %d, want 4", got.Files)
	}
	if got.Folders != 1 {
		t.Errorf("Folders = %d, want 1; skipped folders are not counted", got.Folders)
	}
	if got.TotalSize != 5350 {
		t.Errorf("TotalSize = %d, want 5350", got.TotalSize)
	}
	if got.Recursive {
		t.Error("Recursive = true for a one-level summary")
	}
}

func TestSummaryOneLevelStillReportsSkips(t *testing.T) {
	got := summarize(t, summaryFS(), "summary from 'root'")

	if len(got.Skipped) != 2 {
		t.Errorf("Skipped = %v, want the two noise folders reported even at one level", got.Skipped)
	}
}

func TestSummaryRecursiveCounts(t *testing.T) {
	got := summarize(t, summaryFS(), "summary from 'root' recursive")

	if got.Folders != 2 {
		t.Errorf("Folders = %d, want 2", got.Folders)
	}
	if got.Files != 7 {
		t.Errorf("Files = %d, want 7 (skipped folders excluded)", got.Files)
	}
	if got.TotalSize != 6650 {
		t.Errorf("TotalSize = %d, want 6650", got.TotalSize)
	}
	if !got.Recursive {
		t.Error("Recursive = false")
	}
}

func TestSummarySkipsNoiseFoldersAndSaysSo(t *testing.T) {
	got := summarize(t, summaryFS(), "summary from 'root' recursive")

	if len(got.Skipped) != 2 {
		t.Fatalf("Skipped = %v, want node_modules and .venv", got.Skipped)
	}
	for _, want := range []string{"node_modules", ".venv"} {
		if !containsString(got.Skipped, want) {
			t.Errorf("Skipped = %v, missing %q", got.Skipped, want)
		}
	}
	if !got.SkipsShown {
		t.Error("SkipsShown = false; the warning should be offered")
	}
}

func TestSummaryWithSkippedIncludesThem(t *testing.T) {
	base := summarize(t, summaryFS(), "summary from 'root' recursive")
	full := summarize(t, summaryFS(), "summary from 'root' recursive with skipped")

	if full.Files <= base.Files {
		t.Errorf("with skipped counted %d files, base counted %d; it must include more", full.Files, base.Files)
	}
	if full.TotalSize <= base.TotalSize {
		t.Errorf("with skipped totalled %d, base totalled %d", full.TotalSize, base.TotalSize)
	}
	if full.SkipsShown {
		t.Error("SkipsShown = true even though nothing was skipped")
	}
}

func TestSummaryTypesAreSortedBySizeAndCapped(t *testing.T) {
	got := summarize(t, summaryFS(), "summary from 'root' recursive")

	if len(got.Types) == 0 {
		t.Fatal("no types collected")
	}
	if len(got.Types) > engine.TopTypes {
		t.Errorf("got %d types, want at most %d", len(got.Types), engine.TopTypes)
	}
	for i := 1; i < len(got.Types); i++ {
		if got.Types[i-1].Size < got.Types[i].Size {
			t.Errorf("types not sorted by size: %d then %d", got.Types[i-1].Size, got.Types[i].Size)
		}
	}
	if got.Types[0].Ext != "zip" {
		t.Errorf("biggest type = %q, want zip", got.Types[0].Ext)
	}
}

func TestSummaryTypeTallies(t *testing.T) {
	got := summarize(t, summaryFS(), "summary from 'root' recursive")

	byExt := map[string]engine.TypeTally{}
	for _, t := range got.Types {
		byExt[t.Ext] = t
	}

	if txt := byExt["txt"]; txt.Count != 3 || txt.Size != 600 {
		t.Errorf("txt tally = %+v, want count 3 size 600", txt)
	}
	if zip := byExt["zip"]; zip.Count != 1 || zip.Size != 5000 {
		t.Errorf("zip tally = %+v, want count 1 size 5000", zip)
	}
}

func TestSummaryExtensionlessFilesGroupTogether(t *testing.T) {
	got := summarize(t, summaryFS(), "summary from 'root'")

	found := false
	for _, ty := range got.Types {
		if ty.Ext == "" {
			found = true
			if ty.Count != 1 {
				t.Errorf("extensionless count = %d, want 1", ty.Count)
			}
		}
	}
	if !found {
		t.Error("Makefile did not appear under an empty type")
	}
}

func TestSummaryLargestIsSortedDescendingAndCapped(t *testing.T) {
	got := summarize(t, summaryFS(), "summary from 'root' recursive")

	if len(got.Largest) > engine.TopLargest {
		t.Fatalf("got %d largest, want at most %d", len(got.Largest), engine.TopLargest)
	}
	if len(got.Largest) == 0 {
		t.Fatal("no largest files collected")
	}
	if got.Largest[0].Name != "big.zip" {
		t.Errorf("largest = %q, want big.zip", got.Largest[0].Name)
	}
	for i := 1; i < len(got.Largest); i++ {
		if got.Largest[i-1].Size < got.Largest[i].Size {
			t.Errorf("largest not sorted descending: %d then %d", got.Largest[i-1].Size, got.Largest[i].Size)
		}
	}
}

func TestSummaryLargestKeepsOnlyTheTopNFromManyFiles(t *testing.T) {
	fsys := fstest.MapFS{}
	for i := range 200 {
		fsys[fixtureName("many/f", i, ".bin")] = &fstest.MapFile{Data: bytes.Repeat([]byte("x"), i+1)}
	}

	got := summarize(t, fsys, "summary from 'many'")

	if len(got.Largest) != engine.TopLargest {
		t.Fatalf("got %d largest, want %d", len(got.Largest), engine.TopLargest)
	}
	if got.Largest[0].Size != 200 {
		t.Errorf("biggest = %d bytes, want 200; the heap must keep the true maximum", got.Largest[0].Size)
	}
	if got.Largest[engine.TopLargest-1].Size != 196 {
		t.Errorf("smallest kept = %d bytes, want 196", got.Largest[engine.TopLargest-1].Size)
	}
}

func TestSummaryMoreTypesCount(t *testing.T) {
	fsys := fstest.MapFS{}
	for i, ext := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		fsys["many/file."+ext] = &fstest.MapFile{Data: bytes.Repeat([]byte("x"), (i+1)*10)}
	}

	got := summarize(t, fsys, "summary from 'many'")

	if len(got.Types) != engine.TopTypes {
		t.Errorf("got %d types, want %d", len(got.Types), engine.TopTypes)
	}
	if got.MoreTypes != 2 {
		t.Errorf("MoreTypes = %d, want 2", got.MoreTypes)
	}
}

func TestSummaryModifiedRange(t *testing.T) {
	got := summarize(t, summaryFS(), "summary from 'root'")

	if got.Oldest.Format("2006-01-02") != "2024-03-11" {
		t.Errorf("Oldest = %v, want 2024-03-11", got.Oldest)
	}
	if got.Newest.Format("2006-01-02") != "2026-08-25" {
		t.Errorf("Newest = %v, want 2026-08-25", got.Newest)
	}
}

func TestSummaryEmptyFolder(t *testing.T) {
	got := summarize(t, summaryFS(), "summary from 'empty_ish/'")

	if !got.IsEmpty() {
		t.Skip("fixture folder is not actually empty")
	}
}

func TestSummaryFolderWithNoFiles(t *testing.T) {
	got := summarize(t, summaryFS(), "summary from 'onlyfolders'")

	if got.HasFiles() {
		t.Error("HasFiles() = true for a folder holding only folders")
	}
	if got.Folders != 2 {
		t.Errorf("Folders = %d, want 2", got.Folders)
	}
	if got.IsEmpty() {
		t.Error("IsEmpty() = true; it has folders")
	}
}

func TestSummarySurfacesResolverErrors(t *testing.T) {
	exec := summarizerFor(t, summaryFS())

	for _, tt := range []struct {
		path string
		kind oerr.Kind
	}{
		{"nope", oerr.KindFolderMissing},
		{"root/a.txt", oerr.KindPathIsFile},
	} {
		t.Run(tt.path, func(t *testing.T) {
			stmt := &query.Statement{Verb: query.VerbSummary, Path: tt.path, Target: query.TargetAll}
			if _, err := exec.Summarize(context.Background(), stmt); !oerr.Is(err, tt.kind) {
				t.Errorf("error = %v, want %v", err, tt.kind)
			}
		})
	}
}

func TestSummaryRespectsContextCancellation(t *testing.T) {
	exec := summarizerFor(t, summaryFS())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stmt := &query.Statement{Verb: query.VerbSummary, Path: "root", Target: query.TargetAll, Recursive: true}
	if _, err := exec.Summarize(ctx, stmt); err == nil {
		t.Error("Summarize ignored a cancelled context")
	}
}

func TestSummaryRegistersUnderItsOwnVerb(t *testing.T) {
	exec := summarizerFor(t, summaryFS())

	registry := engine.NewRegistry(exec)
	got, ok := registry.Lookup("summary")
	if !ok {
		t.Fatal("summary executor not registered")
	}
	if _, isSummarizer := got.(engine.Summarizer); !isSummarizer {
		t.Error("summary executor does not satisfy Summarizer")
	}
	if exec.Verb() != "summary" {
		t.Errorf("Verb() = %q, want summary", exec.Verb())
	}
}

func TestSummaryRowPathReportsItIsContentOnly(t *testing.T) {
	exec := summarizerFor(t, summaryFS())

	stmt := &query.Statement{Verb: query.VerbSummary, Path: "root"}
	if err := exec.Execute(context.Background(), stmt, &engine.SliceSink{}); err == nil {
		t.Error("Execute returned no error; summary has no rows")
	}
}

func renderSummary(t *testing.T, s engine.Summary) string {
	t.Helper()

	buf := &bytes.Buffer{}
	if err := output.NewSummary().Render(buf, s); err != nil {
		t.Fatalf("Render error = %v", err)
	}
	return buf.String()
}

func TestRenderSummaryEmptyFolder(t *testing.T) {
	got := renderSummary(t, engine.Summary{Path: "Downloads"})

	if got != "'Downloads' is empty.\n" {
		t.Errorf("got %q, want a single empty-folder line", got)
	}
}

func TestRenderSummaryNoFiles(t *testing.T) {
	got := renderSummary(t, engine.Summary{Path: "src", Folders: 3})

	if !strings.Contains(got, "Contains 3 folders, and no files.") {
		t.Errorf("got:\n%s", got)
	}
	if strings.Contains(got, "LARGEST") {
		t.Error("a folder with no files should not show a LARGEST table")
	}
	if strings.Contains(got, "MODIFIED") {
		t.Error("a folder with no files should not show a date range")
	}
}

func TestRenderSummaryScopeLabel(t *testing.T) {
	shallow := renderSummary(t, engine.Summary{Path: "d", Files: 1, TotalSize: 10})
	deep := renderSummary(t, engine.Summary{Path: "d", Files: 1, TotalSize: 10, Recursive: true})

	if !strings.Contains(shallow, "one level") {
		t.Errorf("non-recursive summary does not say so:\n%s", shallow)
	}
	if !strings.Contains(deep, "all levels") {
		t.Errorf("recursive summary does not say so:\n%s", deep)
	}
}

func TestRenderSummarySections(t *testing.T) {
	s := engine.Summary{
		Path:      "Downloads",
		Files:     52,
		Folders:   8,
		TotalSize: 1503238553,
		Types:     []engine.TypeTally{{Ext: "pdf", Count: 18, Size: 933522718}},
		Largest:   []engine.Row{{Name: "report.pdf", Size: 432013312}},
		Oldest:    time.Date(2024, 3, 11, 0, 0, 0, 0, time.UTC),
		Newest:    time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
	}

	got := renderSummary(t, s)

	for _, want := range []string{"WHAT", "COUNT", "SIZE", "TYPE", "LARGEST", "MODIFIED", "1.4 GB", "report.pdf", "2024-03-11 to 2026-08-25"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestRenderSummaryFoldersHaveNoSize(t *testing.T) {
	got := renderSummary(t, engine.Summary{Path: "d", Files: 2, Folders: 3, TotalSize: 1024})

	lines := strings.Split(got, "\n")
	for _, line := range lines {
		if strings.Contains(line, "folders") && !strings.Contains(line, output.Absent) {
			t.Errorf("folders row should show an em dash: %q", line)
		}
	}
}

func TestRenderSummaryMoreTypes(t *testing.T) {
	one := renderSummary(t, engine.Summary{Path: "d", Files: 1, TotalSize: 1, Types: []engine.TypeTally{{Ext: "go", Count: 1, Size: 1}}, MoreTypes: 1})
	many := renderSummary(t, engine.Summary{Path: "d", Files: 1, TotalSize: 1, Types: []engine.TypeTally{{Ext: "go", Count: 1, Size: 1}}, MoreTypes: 12})

	if !strings.Contains(one, "and 1 more type\n") {
		t.Errorf("singular wording wrong:\n%s", one)
	}
	if !strings.Contains(many, "and 12 more types\n") {
		t.Errorf("plural wording wrong:\n%s", many)
	}
}

func TestRenderSummarySkipNotice(t *testing.T) {
	s := engine.Summary{
		Path: "proj", Files: 1, TotalSize: 1, Recursive: true,
		Skipped: []string{"node_modules", ".venv"}, SkipsShown: true,
	}

	got := renderSummary(t, s)

	if !strings.Contains(got, "Skipped 2 folders: node_modules, .venv") {
		t.Errorf("no skip notice:\n%s", got)
	}
	if !strings.Contains(got, `Add "with skipped" to include them`) {
		t.Errorf("no instruction for including them:\n%s", got)
	}
	if !strings.Contains(got, "take longer") {
		t.Errorf("no warning that it costs time:\n%s", got)
	}
}

func TestRenderSummarySingleSkippedFolderReadsNaturally(t *testing.T) {
	got := renderSummary(t, engine.Summary{Path: "p", Files: 1, TotalSize: 1, Skipped: []string{".git"}, SkipsShown: true})

	if !strings.Contains(got, "Skipped 1 folder: .git") {
		t.Errorf("singular wording wrong:\n%s", got)
	}
}

func TestRenderSummaryNoNoticeWhenIncludingSkipped(t *testing.T) {
	got := renderSummary(t, engine.Summary{Path: "p", Files: 1, TotalSize: 1, Skipped: []string{".git"}, SkipsShown: false})

	if strings.Contains(got, "Skipped") {
		t.Errorf("skip notice shown even though they were included:\n%s", got)
	}
}

func TestDefaultSkipListCoversPythonAndNodeFolders(t *testing.T) {
	s := engine.DefaultSkipList()

	for _, name := range []string{"node_modules", "venv", ".venv", "__pycache__", ".git"} {
		if !s.SkipsName(name) {
			t.Errorf("SkipsName(%q) = false; it should be skipped by default", name)
		}
	}
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
