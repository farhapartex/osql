package test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/output"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/vfs"
)

type deleteFixture struct {
	root  string
	exec  *engine.DeleteExecutor
	trash string
}

func newDeleteFixture(t *testing.T) deleteFixture {
	t.Helper()

	root := t.TempDir()
	write := func(rel, body string) {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("Downloads/cache.tmp", "aaa")
	write("Downloads/build.tmp", "bb")
	write("Downloads/keep.txt", "c")
	write("Downloads/report.pdf", "dddd")
	write("tree/top.txt", "t")
	write("tree/sub/inner.txt", "i")
	write("tree/sub/deeper/leaf.txt", "l")
	if err := os.MkdirAll(filepath.Join(root, "emptydir"), 0o755); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	fsys := vfs.OS()
	resolver := engine.NewPathResolverAt(fsys, root, home)
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())
	trash := vfs.NewTrash(root, func() time.Time { return time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC) })

	return deleteFixture{
		root:  root,
		exec:  engine.NewDeleteExecutor(fsys, resolver, compiler, trash),
		trash: trash.FilesDir(),
	}
}

func (f deleteFixture) plan(t *testing.T, input string) (engine.DeletePlan, error) {
	t.Helper()

	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	tokens, err := query.NewLexer().Lex(input)
	if err != nil {
		return engine.DeletePlan{}, err
	}
	stmt, err := query.NewParser(compiler).Parse(tokens)
	if err != nil {
		return engine.DeletePlan{}, err
	}
	return f.exec.Plan(context.Background(), stmt)
}

func (f deleteFixture) run(t *testing.T, input string) engine.DeleteResult {
	t.Helper()

	plan, err := f.plan(t, input)
	if err != nil {
		t.Fatalf("Plan(%q) error = %v", input, err)
	}
	result, err := f.exec.Commit(context.Background(), plan)
	if err != nil {
		t.Fatalf("Commit error = %v", err)
	}
	return result
}

func (f deleteFixture) exists(rel string) bool {
	_, err := os.Lstat(filepath.Join(f.root, rel))
	return err == nil
}

func (f deleteFixture) inTrash(name string) bool {
	_, err := os.Lstat(filepath.Join(f.trash, name))
	return err == nil
}

func TestParseDeleteForms(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		input      string
		single     bool
		kind       query.NewKind
		target     query.Target
		path       string
		recursive  bool
		permanent  bool
		predicates int
	}{
		{input: "delete file 'notes.txt'", single: true, kind: query.NewFile, path: "notes.txt"},
		{input: "delete folder 'old'", single: true, kind: query.NewFolder, path: "old"},
		{input: "delete file 'notes.txt' permanently", single: true, kind: query.NewFile, path: "notes.txt", permanent: true},
		{input: "delete files from 'Downloads'", target: query.TargetFiles, path: "Downloads"},
		{input: "delete folders from 'src'", target: query.TargetFolders, path: "src"},
		{input: "delete all from 'temp'", target: query.TargetAll, path: "temp"},
		{input: "delete files from 'd' recursive", target: query.TargetFiles, path: "d", recursive: true},
		{input: "delete files from 'd' where type = 'tmp'", target: query.TargetFiles, path: "d", predicates: 1},
		{input: "delete files from 'd' where name_like = '%.tmp'", target: query.TargetFiles, path: "d", predicates: 1},
		{input: "delete files from 'd' recursive where type = 'tmp' permanently", target: query.TargetFiles, path: "d", recursive: true, predicates: 1, permanent: true},
		{input: "DELETE FILES FROM 'd' PERMANENTLY", target: query.TargetFiles, path: "d", permanent: true},
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
			if stmt.Verb != query.VerbDelete {
				t.Errorf("Verb = %q, want delete", stmt.Verb)
			}
			if stmt.Single != tt.single {
				t.Errorf("Single = %v, want %v", stmt.Single, tt.single)
			}
			if tt.single && stmt.Kind != tt.kind {
				t.Errorf("Kind = %v, want %v", stmt.Kind, tt.kind)
			}
			if !tt.single && stmt.Target != tt.target {
				t.Errorf("Target = %v, want %v", stmt.Target, tt.target)
			}
			if stmt.Path != tt.path {
				t.Errorf("Path = %q, want %q", stmt.Path, tt.path)
			}
			if stmt.Recursive != tt.recursive {
				t.Errorf("Recursive = %v, want %v", stmt.Recursive, tt.recursive)
			}
			if stmt.Permanent != tt.permanent {
				t.Errorf("Permanent = %v, want %v", stmt.Permanent, tt.permanent)
			}
			if len(stmt.Predicates) != tt.predicates {
				t.Errorf("got %d predicates, want %d", len(stmt.Predicates), tt.predicates)
			}
		})
	}
}

func TestDeleteFilterMatchesTheListingFilter(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	pairs := [][2]string{
		{"files from 'd' where type = 'tmp'", "delete files from 'd' where type = 'tmp'"},
		{"files from 'd' recursive where name_like = '%.log'", "delete files from 'd' recursive where name_like = '%.log'"},
		{"folders from 'd' where count(child) = 0", "delete folders from 'd' where count(child) = 0"},
	}

	for _, pair := range pairs {
		t.Run(pair[0], func(t *testing.T) {
			listed := mustParse(t, parser, pair[0])
			deleted := mustParse(t, parser, pair[1])

			if listed.Target != deleted.Target {
				t.Errorf("target differs: %v vs %v", listed.Target, deleted.Target)
			}
			if listed.Path != deleted.Path {
				t.Errorf("path differs: %q vs %q", listed.Path, deleted.Path)
			}
			if listed.Recursive != deleted.Recursive {
				t.Errorf("recursive differs")
			}
			if len(listed.Predicates) != len(deleted.Predicates) {
				t.Fatalf("predicate count differs")
			}
			for i := range listed.Predicates {
				if listed.Predicates[i] != deleted.Predicates[i] {
					t.Errorf("predicate %d differs: %+v vs %+v", i, listed.Predicates[i], deleted.Predicates[i])
				}
			}
		})
	}
}

func mustParse(t *testing.T, p query.Parser, input string) *query.Statement {
	t.Helper()

	tokens, err := query.NewLexer().Lex(input)
	if err != nil {
		t.Fatal(err)
	}
	stmt, err := p.Parse(tokens)
	if err != nil {
		t.Fatalf("Parse(%q) error = %v", input, err)
	}
	return stmt
}

func TestParseDeleteErrors(t *testing.T) {
	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	parser := query.NewParser(compiler)

	tests := []struct {
		name  string
		input string
		kind  oerr.Kind
	}{
		{"no target", "delete", oerr.KindMissingDeleteTarget},
		{"unknown target", "delete thing 'x'", oerr.KindMissingDeleteTarget},
		{"no single path", "delete file", oerr.KindMissingDeletePath},
		{"no folder path", "delete folder", oerr.KindMissingDeletePath},
		{"bulk without from", "delete files 'x'", oerr.KindMissingFrom},
		{"bulk without path", "delete files from", oerr.KindMissingPath},
		{"trailing junk", "delete files from 'x' junk", oerr.KindUnexpectedInput},
		{"junk after permanently", "delete file 'x' permanently junk", oerr.KindUnexpectedInput},
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

func TestDeleteMovesAFileToTheTrash(t *testing.T) {
	f := newDeleteFixture(t)

	result := f.run(t, "delete file 'Downloads/keep.txt'")

	if len(result.Deleted) != 1 || len(result.Failed) != 0 {
		t.Fatalf("result = %+v", result)
	}
	if f.exists("Downloads/keep.txt") {
		t.Error("the file is still there")
	}
	if !f.inTrash("keep.txt") {
		t.Error("the file did not reach the trash")
	}
	if result.Permanent {
		t.Error("Permanent = true for a default delete")
	}
}

func TestDeletePermanentlySkipsTheTrash(t *testing.T) {
	f := newDeleteFixture(t)

	result := f.run(t, "delete file 'Downloads/keep.txt' permanently")

	if f.exists("Downloads/keep.txt") {
		t.Error("the file is still there")
	}
	if f.inTrash("keep.txt") {
		t.Error("a permanent delete put the file in the trash")
	}
	if !result.Permanent {
		t.Error("Permanent = false")
	}
}

func TestDeleteBulkWithFilter(t *testing.T) {
	f := newDeleteFixture(t)

	result := f.run(t, "delete files from 'Downloads' where type = 'tmp'")

	if len(result.Deleted) != 2 {
		t.Fatalf("deleted %v, want the two tmp files", result.Deleted)
	}
	if f.exists("Downloads/cache.tmp") || f.exists("Downloads/build.tmp") {
		t.Error("a tmp file survived")
	}
	if !f.exists("Downloads/keep.txt") || !f.exists("Downloads/report.pdf") {
		t.Error("an unmatched file was deleted")
	}
}

func TestDeleteBulkWithPattern(t *testing.T) {
	f := newDeleteFixture(t)

	f.run(t, "delete files from 'Downloads' where name_like = '%.tmp'")

	if f.exists("Downloads/cache.tmp") {
		t.Error("pattern delete missed a file")
	}
	if !f.exists("Downloads/keep.txt") {
		t.Error("pattern delete removed too much")
	}
}

func TestDeleteFolderTakesItsContents(t *testing.T) {
	f := newDeleteFixture(t)

	f.run(t, "delete folder 'tree'")

	if f.exists("tree") {
		t.Error("the folder is still there")
	}
	if !f.inTrash("tree") {
		t.Error("the folder did not reach the trash")
	}
}

func TestDeletePlanWeighsAFolder(t *testing.T) {
	f := newDeleteFixture(t)

	plan, err := f.plan(t, "delete folder 'tree'")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Victims) != 1 {
		t.Fatalf("got %d victims, want 1", len(plan.Victims))
	}

	victim := plan.Victims[0]
	if !victim.IsDir {
		t.Error("victim not marked as a folder")
	}
	if victim.Files != 3 {
		t.Errorf("Files = %d, want 3; the preview must show the real weight", victim.Files)
	}
	if victim.Folders != 3 {
		t.Errorf("Folders = %d, want 3 (tree, sub, deeper)", victim.Folders)
	}
	if victim.Size != 3 {
		t.Errorf("Size = %d, want 3", victim.Size)
	}
}

func TestDeleteSelectsOnlyTheOutermostPaths(t *testing.T) {
	f := newDeleteFixture(t)

	plan, err := f.plan(t, "delete all from 'tree' recursive")
	if err != nil {
		t.Fatal(err)
	}

	names := make([]string, 0, len(plan.Victims))
	for _, v := range plan.Victims {
		names = append(names, v.Name)
	}

	if len(plan.Victims) != 2 {
		t.Fatalf("selected %v, want just sub and top.txt; children of a selected folder are redundant", names)
	}
	for _, name := range names {
		if strings.Contains(name, "/") {
			t.Errorf("selected %q, which lives inside another selected folder", name)
		}
	}
}

func TestDeleteIgnoresTheSkipList(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "proj", "node_modules", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "proj", "node_modules", "pkg", "x.js"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	fsys := vfs.OS()
	exec := engine.NewDeleteExecutor(fsys, engine.NewPathResolver(fsys, root),
		engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators()),
		vfs.NewTrash(root, nil))

	compiler := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())
	tokens, _ := query.NewLexer().Lex("delete all from 'proj'")
	stmt, err := query.NewParser(compiler).Parse(tokens)
	if err != nil {
		t.Fatal(err)
	}

	plan, err := exec.Plan(context.Background(), stmt)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, v := range plan.Victims {
		if v.Name == "node_modules" {
			found = true
		}
	}
	if !found {
		t.Error("node_modules was skipped; delete must see everything or it silently leaves files behind")
	}
}

func TestDeleteRefusesTheRoot(t *testing.T) {
	f := newDeleteFixture(t)

	for _, input := range []string{
		"delete all from '/'",
		"delete all from '~'",
		"delete files from '/'",
		"delete folder '/'",
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := f.plan(t, input); !oerr.Is(err, oerr.KindRefuseDeleteRoot) {
				t.Errorf("error = %v, want refuse_delete_root", err)
			}
		})
	}
}

func TestDeleteAllowsTheRootWithAFilter(t *testing.T) {
	f := newDeleteFixture(t)

	if _, err := f.plan(t, "delete files from '/' where type = 'nothing'"); err != nil {
		t.Errorf("a filtered delete at the root should be allowed, got %v", err)
	}
}

func TestDeleteKindMismatchSuggestsTheOtherForm(t *testing.T) {
	f := newDeleteFixture(t)

	_, err := f.plan(t, "delete file 'emptydir'")
	if !oerr.Is(err, oerr.KindDeleteKindMismatch) {
		t.Fatalf("error = %v, want delete_kind_mismatch", err)
	}
	if !strings.Contains(err.Error(), "delete folder 'emptydir'") {
		t.Errorf("error does not offer the right command: %v", err)
	}

	_, err = f.plan(t, "delete folder 'Downloads/keep.txt'")
	if !strings.Contains(err.Error(), "delete file 'Downloads/keep.txt'") {
		t.Errorf("error does not offer the right command: %v", err)
	}
}

func TestDeleteMissingPath(t *testing.T) {
	f := newDeleteFixture(t)

	if _, err := f.plan(t, "delete file 'nope.txt'"); !oerr.Is(err, oerr.KindFileMissing) {
		t.Errorf("error = %v, want file_missing", err)
	}
	if _, err := f.plan(t, "delete folder 'nope'"); !oerr.Is(err, oerr.KindFolderMissing) {
		t.Errorf("error = %v, want folder_missing", err)
	}
}

func TestDeleteEmptySelection(t *testing.T) {
	f := newDeleteFixture(t)

	plan, err := f.plan(t, "delete files from 'Downloads' where type = 'nothing'")
	if err != nil {
		t.Fatal(err)
	}
	if !plan.IsEmpty() {
		t.Errorf("plan has %d victims, want none", len(plan.Victims))
	}
}

func TestDeleteCanPlanAboveTheStartDirectory(t *testing.T) {
	f := newDeleteFixture(t)

	outside := filepath.Join(filepath.Dir(f.root), "outside.txt")
	if err := os.WriteFile(outside, []byte("o"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(outside) })

	plan, err := f.plan(t, "delete file '../outside.txt'")
	if err != nil {
		t.Fatalf("planning above the start directory must work now: %v", err)
	}
	if len(plan.Victims) != 1 {
		t.Fatalf("planned %d victims, want 1", len(plan.Victims))
	}
	if _, err := os.Stat(outside); err != nil {
		t.Error("planning must not delete anything")
	}
}

func TestDeleteRefusesTheFolderYouAreIn(t *testing.T) {
	f := newDeleteFixture(t)

	if _, err := f.plan(t, "delete folder '.'"); !oerr.Is(err, oerr.KindRefuseDeleteHere) {
		t.Errorf("error = %v, want refuse_delete_here", err)
	}
}

func TestDeleteCanEmptyTheFolderYouAreIn(t *testing.T) {
	f := newDeleteFixture(t)

	plan, err := f.plan(t, "delete all from '.'")
	if err != nil {
		t.Fatalf("emptying the current folder is allowed with a confirmation: %v", err)
	}
	if len(plan.Victims) == 0 {
		t.Error("planned nothing")
	}
}

func TestDeleteReportsPerItemFailures(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root, permission cannot be denied")
	}

	f := newDeleteFixture(t)
	locked := filepath.Join(f.root, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "stuck.txt"), []byte("s"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "also.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })

	result := f.run(t, "delete files from 'locked'")

	if len(result.Failed) != 2 {
		t.Fatalf("failures = %+v, want both files to fail", result.Failed)
	}
	for _, failure := range result.Failed {
		if failure.Reason == "" {
			t.Errorf("failure %q has no reason", failure.Name)
		}
	}
}

func TestDeleteContinuesAfterAFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root, permission cannot be denied")
	}

	f := newDeleteFixture(t)
	locked := filepath.Join(f.root, "mixed")
	if err := os.MkdirAll(filepath.Join(locked, "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "inner", "stuck.txt"), []byte("s"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "free.txt"), []byte("f"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(locked, "inner"), 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(filepath.Join(locked, "inner"), 0o755) })

	result := f.run(t, "delete all from 'mixed'")

	if len(result.Deleted) == 0 {
		t.Error("nothing was deleted; a single failure must not abort the rest")
	}
}

func TestDeleteRespectsContextCancellation(t *testing.T) {
	f := newDeleteFixture(t)

	plan, err := f.plan(t, "delete files from 'Downloads' where type = 'tmp'")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := f.exec.Commit(ctx, plan); err == nil {
		t.Error("Commit ignored a cancelled context")
	}
	if !f.exists("Downloads/cache.tmp") {
		t.Error("a file was deleted after cancellation")
	}
}

func TestDeleteRegistersUnderItsOwnVerb(t *testing.T) {
	f := newDeleteFixture(t)

	registry := engine.NewRegistry(f.exec)
	got, ok := registry.Lookup("delete")
	if !ok {
		t.Fatal("delete executor not registered")
	}
	if _, isDeleter := got.(engine.Deleter); !isDeleter {
		t.Error("delete executor does not satisfy Deleter")
	}
}

func TestDeleteRowPathReportsItIsContentOnly(t *testing.T) {
	f := newDeleteFixture(t)

	err := f.exec.Execute(context.Background(), &query.Statement{Verb: query.VerbDelete}, &engine.SliceSink{})
	if err == nil {
		t.Error("Execute returned no error; delete has no rows")
	}
}

func TestTrashRenamesOnCollision(t *testing.T) {
	root := t.TempDir()
	trash := vfs.NewTrash(root, nil)

	for i, body := range []string{"first", "second", "third"} {
		dir := filepath.Join(root, "src", string(rune('a'+i)))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(dir, "same.txt")
		if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := trash.Move(file); err != nil {
			t.Fatalf("Move %d error = %v", i, err)
		}
	}

	entries, err := os.ReadDir(trash.FilesDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("trash holds %v, want three distinct entries; a collision overwrote a deleted file", names)
	}

	bodies := map[string]bool{}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(trash.FilesDir(), e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		bodies[string(data)] = true
	}
	for _, want := range []string{"first", "second", "third"} {
		if !bodies[want] {
			t.Errorf("the trash lost the file containing %q", want)
		}
	}
}

func TestTrashCreatesItsFolder(t *testing.T) {
	root := t.TempDir()
	trash := vfs.NewTrash(root, nil)

	if err := trash.Ensure(); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	info, err := os.Stat(trash.FilesDir())
	if err != nil {
		t.Fatalf("trash folder not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("trash path is not a folder")
	}
}

func renderPreview(t *testing.T, plan engine.DeletePlan) string {
	t.Helper()

	buf := &bytes.Buffer{}
	if err := output.NewDelete().Preview(buf, plan); err != nil {
		t.Fatalf("Preview error = %v", err)
	}
	return buf.String()
}

func TestPreviewWording(t *testing.T) {
	one := renderPreview(t, engine.DeletePlan{Victims: []engine.Victim{{Name: "a.txt", Size: 10}}})
	many := renderPreview(t, engine.DeletePlan{Victims: []engine.Victim{{Name: "a.txt"}, {Name: "b.txt"}}})
	folder := renderPreview(t, engine.DeletePlan{Victims: []engine.Victim{{Name: "old", IsDir: true, Files: 5, Folders: 2}}})

	if !strings.Contains(one, "This file will be deleted") {
		t.Errorf("single file wording:\n%s", one)
	}
	if !strings.Contains(one, "It will be moved to the trash") {
		t.Errorf("single subject wording:\n%s", one)
	}
	if !strings.Contains(many, "These 2 items will be deleted") {
		t.Errorf("plural wording:\n%s", many)
	}
	if !strings.Contains(many, "They will be moved") {
		t.Errorf("plural subject wording:\n%s", many)
	}
	if !strings.Contains(folder, "'old' and everything in it") {
		t.Errorf("folder wording:\n%s", folder)
	}
	if !strings.Contains(folder, "5 files") {
		t.Errorf("folder weight missing:\n%s", folder)
	}
}

func TestPreviewSaysWhenItIsPermanent(t *testing.T) {
	plan := engine.DeletePlan{Permanent: true, Victims: []engine.Victim{{Name: "a.txt"}}}

	got := renderPreview(t, plan)
	if !strings.Contains(got, "for good") {
		t.Errorf("a permanent delete must say so:\n%s", got)
	}
	if strings.Contains(got, "trash") {
		t.Errorf("a permanent delete should not mention the trash:\n%s", got)
	}
}

func TestPreviewAsksForTheWordYes(t *testing.T) {
	got := renderPreview(t, engine.DeletePlan{Victims: []engine.Victim{{Name: "a.txt"}}})

	if !strings.Contains(got, `Type "yes"`) {
		t.Errorf("the prompt must ask for a typed word:\n%s", got)
	}
	if output.ConfirmWord != "yes" {
		t.Errorf("ConfirmWord = %q, want \"yes\"", output.ConfirmWord)
	}
}

func TestPreviewShowsATotalOnlyForMultipleItems(t *testing.T) {
	one := renderPreview(t, engine.DeletePlan{Victims: []engine.Victim{{Name: "a.txt", Size: 10}}, TotalSize: 10})
	many := renderPreview(t, engine.DeletePlan{Victims: []engine.Victim{{Name: "a.txt"}, {Name: "b.txt"}}, TotalSize: 20})

	if strings.Contains(one, "total") {
		t.Errorf("a single item needs no total line:\n%s", one)
	}
	if !strings.Contains(many, "total") {
		t.Errorf("several items should be totalled:\n%s", many)
	}
}

func TestDeleteResultWording(t *testing.T) {
	buf := &bytes.Buffer{}
	trashed := engine.DeleteResult{Deleted: []string{"a", "b"}}
	if err := output.NewDelete().Result(buf, trashed); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Moved 2 items to the trash") {
		t.Errorf("trash wording:\n%s", buf.String())
	}

	buf.Reset()
	permanent := engine.DeleteResult{Deleted: []string{"a"}, Permanent: true}
	if err := output.NewDelete().Result(buf, permanent); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Deleted 1 item.") {
		t.Errorf("permanent wording:\n%s", buf.String())
	}
	if strings.Contains(buf.String(), "trash") {
		t.Errorf("permanent result should not mention the trash:\n%s", buf.String())
	}
}

func TestDeleteResultMentionsSudoOnFailure(t *testing.T) {
	buf := &bytes.Buffer{}
	result := engine.DeleteResult{
		Deleted: []string{"a"},
		Failed:  []engine.DeleteOutcome{{Name: "b", Reason: "permission denied"}},
	}

	if err := output.NewDelete().Result(buf, result); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	for _, want := range []string{"Moved 1 item", "Could not delete 1 item", "permission denied", "sudo"} {
		if !strings.Contains(got, want) {
			t.Errorf("result missing %q:\n%s", want, got)
		}
	}
}

func TestDeleteCancelledMessage(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := output.NewDelete().Cancelled(buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Nothing was deleted") {
		t.Errorf("cancel wording: %q", buf.String())
	}
}
