package test

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/vfs"
)

func resolverFor(t *testing.T, home, cwd string) *engine.PathResolver {
	t.Helper()

	fsys := fstest.MapFS{
		"home/user/Documents/notes.txt": {Data: []byte("a")},
		"home/user/Downloads/x.txt":     {Data: []byte("b")},
		"work/project/main.go":          {Data: []byte("c")},
		"work/loose.txt":                {Data: []byte("d")},
	}
	return engine.NewPathResolver(&fakeFileSystem{fsys: fsys}, "/", home, cwd)
}

func TestExpandTildeAndRelativePaths(t *testing.T) {
	r := resolverFor(t, "/home/user", "/work")

	tests := []struct {
		in   string
		want string
	}{
		{"~", "/home/user"},
		{"~/Documents", "/home/user/Documents"},
		{"~/Documents/", "/home/user/Documents"},
		{"~/Documents/../Downloads", "/home/user/Downloads"},
		{"/absolute/path", "/absolute/path"},
		{"/absolute/path/", "/absolute/path"},
		{"project", "/work/project"},
		{"./project", "/work/project"},
		{"../home/user", "/home/user"},
		{".", "/work"},
		{"", "/work"},
		{"  ~/Documents  ", "/home/user/Documents"},
		{"//double//slashes//", "/double/slashes"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := r.Expand(tt.in); got != tt.want {
				t.Errorf("Expand(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExpandDoesNotExpandOtherUsersHome(t *testing.T) {
	r := resolverFor(t, "/home/user", "/work")

	got := r.Expand("~otheruser/Documents")
	if got == "/home/user/Documents" {
		t.Error("~otheruser was expanded to the current user's home")
	}
	if got != "/work/~otheruser/Documents" {
		t.Errorf("Expand(\"~otheruser/Documents\") = %q; only ~ and ~/ expand", got)
	}
}

func TestResolveSuccess(t *testing.T) {
	r := resolverFor(t, "/home/user", "/work")

	got, err := r.Resolve("~/Documents")
	if err != nil {
		t.Fatalf("Resolve error = %v", err)
	}
	if got.Input != "~/Documents" {
		t.Errorf("Input = %q, want the path as typed", got.Input)
	}
	if got.Absolute != "/home/user/Documents" {
		t.Errorf("Absolute = %q", got.Absolute)
	}
	if got.FSPath != "home/user/Documents" {
		t.Errorf("FSPath = %q, want an unrooted fs path", got.FSPath)
	}
}

func TestResolveRelativeAgainstCwd(t *testing.T) {
	r := resolverFor(t, "/home/user", "/work")

	got, err := r.Resolve("project")
	if err != nil {
		t.Fatalf("Resolve error = %v", err)
	}
	if got.FSPath != "work/project" {
		t.Errorf("FSPath = %q, want \"work/project\"", got.FSPath)
	}
}

func TestResolveClassifiesErrors(t *testing.T) {
	r := resolverFor(t, "/home/user", "/work")

	tests := []struct {
		name  string
		input string
		kind  oerr.Kind
	}{
		{"missing folder", "~/Documnets", oerr.KindFolderMissing},
		{"missing absolute", "/nope/nowhere", oerr.KindFolderMissing},
		{"path is a file", "~/Documents/notes.txt", oerr.KindPathIsFile},
		{"relative file", "loose.txt", oerr.KindPathIsFile},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.Resolve(tt.input)
			if err == nil {
				t.Fatalf("Resolve(%q) succeeded", tt.input)
			}
			if !oerr.Is(err, tt.kind) {
				t.Errorf("Resolve(%q) error kind mismatch\n got: %v\nwant: %v", tt.input, err, tt.kind)
			}
		})
	}
}

func TestResolveErrorQuotesThePathAsTyped(t *testing.T) {
	r := resolverFor(t, "/home/user", "/work")

	_, err := r.Resolve("~/Documnets")
	if err == nil {
		t.Fatal("expected an error")
	}
	want := "I couldn't find a folder at '~/Documnets'. Check the path and try again."
	if err.Error() != want {
		t.Errorf("\n got: %s\nwant: %s", err.Error(), want)
	}
}

func TestResolveNoPermissionOnRealFilesystem(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root, permission cannot be denied")
	}

	root := t.TempDir()
	locked := filepath.Join(root, "locked")
	if err := os.MkdirAll(filepath.Join(locked, "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })

	r := engine.NewPathResolver(vfs.NewOS(root), root, root, root)

	_, err := r.Resolve(filepath.Join(locked, "inner"))
	if err == nil {
		t.Fatal("Resolve succeeded on an unreadable parent")
	}
	if !oerr.Is(err, oerr.KindNoPermission) && !oerr.Is(err, oerr.KindFolderMissing) {
		t.Errorf("error kind = %v, want no_permission or folder_missing", err)
	}
}

func TestResolveOnRealFilesystem(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Documents"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := engine.NewPathResolver(vfs.NewOS(root), root, root, root)

	got, err := r.Resolve("~/Documents")
	if err != nil {
		t.Fatalf("Resolve error = %v", err)
	}
	if got.FSPath != "Documents" {
		t.Errorf("FSPath = %q, want \"Documents\"", got.FSPath)
	}
}
