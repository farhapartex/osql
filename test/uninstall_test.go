package test

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/uninstall"
)

type fakeFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return f.size }
func (f fakeFileInfo) Mode() fs.FileMode  { return 0o644 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.isDir }
func (f fakeFileInfo) Sys() any           { return nil }

type fakeFiles struct {
	present    map[string]int64
	sizes      map[string]int64
	unwritable map[string]bool
	sizeErr    error
	removeErr  error
	treeErr    error
	removed    []string
}

func newFakeFiles() *fakeFiles {
	return &fakeFiles{
		present:    map[string]int64{},
		sizes:      map[string]int64{},
		unwritable: map[string]bool{},
	}
}

func (f *fakeFiles) Stat(path string) (fs.FileInfo, error) {
	size, ok := f.present[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return fakeFileInfo{name: path, size: size}, nil
}

func (f *fakeFiles) DirectorySize(path string) (int64, error) {
	if f.sizeErr != nil {
		return 0, f.sizeErr
	}
	return f.sizes[path], nil
}

func (f *fakeFiles) CanWriteInto(directory string) bool {
	return !f.unwritable[directory]
}

func (f *fakeFiles) Remove(path string) error {
	if f.removeErr != nil {
		return f.removeErr
	}
	f.removed = append(f.removed, path)
	return nil
}

func (f *fakeFiles) RemoveTree(path string) error {
	if f.treeErr != nil {
		return f.treeErr
	}
	f.removed = append(f.removed, path)
	return nil
}

const (
	binaryPath = "/home/me/.local/bin/osql"
	statePath  = "/home/me/.osql"
)

func plannerFor(files uninstall.Files, path string) *uninstall.Uninstaller {
	return uninstall.New(uninstall.Options{
		Files:        files,
		LocateBinary: func() (string, error) { return path, nil },
		StateRoot:    statePath,
	})
}

func readyFiles() *fakeFiles {
	files := newFakeFiles()
	files.present[binaryPath] = 5_242_880
	files.present[statePath] = 0
	files.sizes[statePath] = 12_288
	return files
}

func TestPlanListsTheBinaryAndTheStateFolder(t *testing.T) {
	plan, err := plannerFor(readyFiles(), binaryPath).Plan(false)
	if err != nil {
		t.Fatalf("Plan() = %v, want no error", err)
	}

	if plan.Binary.Path != binaryPath {
		t.Errorf("binary path = %q, want %q", plan.Binary.Path, binaryPath)
	}
	if plan.Binary.Size != 5_242_880 {
		t.Errorf("binary size = %d, want 5242880", plan.Binary.Size)
	}
	if !plan.IncludesData {
		t.Fatal("plan should include the state folder")
	}
	if plan.Data.Path != statePath {
		t.Errorf("data path = %q, want %q", plan.Data.Path, statePath)
	}
	if plan.Data.Size != 12_288 {
		t.Errorf("data size = %d, want 12288", plan.Data.Size)
	}
	if got, want := plan.TotalSize(), int64(5_242_880+12_288); got != want {
		t.Errorf("TotalSize() = %d, want %d", got, want)
	}
}

func TestPlanKeepsDataWhenAsked(t *testing.T) {
	plan, err := plannerFor(readyFiles(), binaryPath).Plan(true)
	if err != nil {
		t.Fatalf("Plan() = %v, want no error", err)
	}

	if plan.IncludesData {
		t.Error("--keep-data must leave the state folder out of the plan")
	}
	if plan.Data.Path != statePath {
		t.Errorf("data path = %q, want %q so the plan can say where the notes stayed", plan.Data.Path, statePath)
	}
	if plan.Data.Size != 0 {
		t.Errorf("data size = %d; a folder we keep is never measured", plan.Data.Size)
	}
	if got, want := plan.TotalSize(), int64(5_242_880); got != want {
		t.Errorf("TotalSize() = %d, want %d", got, want)
	}
}

func TestKeepingDataIgnoresAnUnwritableStateFolder(t *testing.T) {
	files := readyFiles()
	files.unwritable[statePath] = true

	plan, err := plannerFor(files, binaryPath).Plan(true)
	if err != nil {
		t.Fatalf("a folder we are keeping needs no write permission, got %v", err)
	}
	if plan.IncludesData {
		t.Error("plan must not include the data")
	}
}

func TestPlanSkipsAStateFolderThatIsNotThere(t *testing.T) {
	files := readyFiles()
	delete(files.present, statePath)

	plan, err := plannerFor(files, binaryPath).Plan(false)
	if err != nil {
		t.Fatalf("a missing state folder is not an error, got %v", err)
	}
	if plan.IncludesData {
		t.Error("plan claims to remove a folder that does not exist")
	}
}

func TestPlanRefusesWhenTheBinaryCannotBeFound(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		locate error
	}{
		{"lookup failed", "", errors.New("no executable")},
		{"empty path", "", nil},
		{"blank path", "   ", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planner := uninstall.New(uninstall.Options{
				Files:        readyFiles(),
				LocateBinary: func() (string, error) { return tt.path, tt.locate },
				StateRoot:    statePath,
			})

			_, err := planner.Plan(false)
			if !oerr.Is(err, oerr.KindBinaryNotFound) {
				t.Errorf("Plan() = %v, want binary_not_found", err)
			}
		})
	}
}

func TestPlanRefusesWhenTheBinaryIsGone(t *testing.T) {
	files := readyFiles()
	delete(files.present, binaryPath)

	_, err := plannerFor(files, binaryPath).Plan(false)
	if !oerr.Is(err, oerr.KindBinaryNotFound) {
		t.Errorf("Plan() = %v, want binary_not_found", err)
	}
}

func TestPlanRefusesPackageManagerInstalls(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"homebrew on apple silicon", "/opt/homebrew/Cellar/osql/0.1.0/bin/osql", "brew uninstall osql"},
		{"homebrew on intel", "/usr/local/Cellar/osql/0.1.0/bin/osql", "brew uninstall osql"},
		{"nix", "/nix/store/abc123-osql-0.1.0/bin/osql", "nix profile remove osql"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := newFakeFiles()
			files.present[tt.path] = 100

			_, err := plannerFor(files, tt.path).Plan(false)
			if !oerr.Is(err, oerr.KindInstalledByPackageManager) {
				t.Fatalf("Plan() = %v, want installed_by_package_manager", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("message must name the command %q, got: %s", tt.want, err)
			}
		})
	}
}

func TestPlanRefusesAnUnwritableInstallFolder(t *testing.T) {
	files := readyFiles()
	files.unwritable["/home/me/.local/bin"] = true

	_, err := plannerFor(files, binaryPath).Plan(false)
	if !oerr.Is(err, oerr.KindCannotRemoveBinary) {
		t.Fatalf("Plan() = %v, want cannot_remove_binary", err)
	}
	if !strings.Contains(err.Error(), "sudo rm '"+binaryPath+"'") {
		t.Errorf("message must show the exact sudo line, got: %s", err)
	}
}

func TestPlanRefusesAnUnwritableStateFolder(t *testing.T) {
	files := readyFiles()
	files.unwritable[statePath] = true

	_, err := plannerFor(files, binaryPath).Plan(false)
	if !oerr.Is(err, oerr.KindCannotRemoveData) {
		t.Errorf("Plan() = %v, want cannot_remove_data", err)
	}
}

func TestPlanRefusesWhenTheStateFolderCannotBeMeasured(t *testing.T) {
	files := readyFiles()
	files.sizeErr = fs.ErrPermission

	_, err := plannerFor(files, binaryPath).Plan(false)
	if !oerr.Is(err, oerr.KindCannotRemoveData) {
		t.Fatalf("Plan() = %v, want cannot_remove_data", err)
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("message must explain why, got: %s", err)
	}
}

func TestPlanRemovesNothingByItself(t *testing.T) {
	files := readyFiles()

	if _, err := plannerFor(files, binaryPath).Plan(false); err != nil {
		t.Fatalf("Plan() = %v", err)
	}
	if len(files.removed) != 0 {
		t.Errorf("planning removed %v; it must only look", files.removed)
	}
}

func TestCommitRemovesTheStateFolderBeforeTheBinary(t *testing.T) {
	files := readyFiles()
	planner := plannerFor(files, binaryPath)

	plan, err := planner.Plan(false)
	if err != nil {
		t.Fatalf("Plan() = %v", err)
	}
	if err := planner.Commit(plan); err != nil {
		t.Fatalf("Commit() = %v", err)
	}

	want := []string{statePath, binaryPath}
	if len(files.removed) != len(want) {
		t.Fatalf("removed %v, want %v", files.removed, want)
	}
	for i := range want {
		if files.removed[i] != want[i] {
			t.Errorf("removed[%d] = %q, want %q", i, files.removed[i], want[i])
		}
	}
}

func TestCommitLeavesTheStateFolderWhenKeepingData(t *testing.T) {
	files := readyFiles()
	planner := plannerFor(files, binaryPath)

	plan, err := planner.Plan(true)
	if err != nil {
		t.Fatalf("Plan() = %v", err)
	}
	if err := planner.Commit(plan); err != nil {
		t.Fatalf("Commit() = %v", err)
	}

	if len(files.removed) != 1 || files.removed[0] != binaryPath {
		t.Errorf("removed %v, want only %q", files.removed, binaryPath)
	}
}

func TestCommitKeepsTheBinaryWhenTheStateFolderWillNotGo(t *testing.T) {
	files := readyFiles()
	planner := plannerFor(files, binaryPath)

	plan, err := planner.Plan(false)
	if err != nil {
		t.Fatalf("Plan() = %v", err)
	}

	files.treeErr = fs.ErrPermission
	err = planner.Commit(plan)

	if !oerr.Is(err, oerr.KindCannotRemoveData) {
		t.Fatalf("Commit() = %v, want cannot_remove_data", err)
	}
	if len(files.removed) != 0 {
		t.Errorf("removed %v; a half-uninstall must never happen", files.removed)
	}
	if !strings.Contains(err.Error(), "osql still works") {
		t.Errorf("message must say the tool is untouched, got: %s", err)
	}
}

func TestCommitReportsABinaryItCannotRemove(t *testing.T) {
	files := readyFiles()
	planner := plannerFor(files, binaryPath)

	plan, err := planner.Plan(true)
	if err != nil {
		t.Fatalf("Plan() = %v", err)
	}

	files.removeErr = fs.ErrPermission
	if err := planner.Commit(plan); !oerr.Is(err, oerr.KindCannotRemoveBinary) {
		t.Errorf("Commit() = %v, want cannot_remove_binary", err)
	}
}

func TestPlanWithoutAStateRootOnlyRemovesTheBinary(t *testing.T) {
	files := readyFiles()
	planner := uninstall.New(uninstall.Options{
		Files:        files,
		LocateBinary: func() (string, error) { return binaryPath, nil },
	})

	plan, err := planner.Plan(false)
	if err != nil {
		t.Fatalf("Plan() = %v", err)
	}
	if plan.IncludesData {
		t.Error("there is no state folder to remove")
	}
}
