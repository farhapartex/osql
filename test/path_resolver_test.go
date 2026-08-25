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

func rootedResolver(t *testing.T) *engine.PathResolver {
	t.Helper()

	fsys := fstest.MapFS{
		"Documents/notes.txt":  {Data: []byte("a")},
		"Documents/deep/x.txt": {Data: []byte("b")},
		"Downloads/y.txt":      {Data: []byte("c")},
		"loose.txt":            {Data: []byte("d")},
	}
	return engine.NewPathResolver(&fakeFileSystem{fsys: fsys}, "/")
}

func TestExpandIsAlwaysRootRelative(t *testing.T) {
	r := engine.NewPathResolver(&fakeFileSystem{fsys: fstest.MapFS{}}, "/home/user")

	tests := []struct {
		in   string
		want string
	}{
		{"Documents", "/home/user/Documents"},
		{"/Documents", "/home/user/Documents"},
		{"~/Documents", "/home/user/Documents"},
		{"./Documents", "/home/user/Documents"},
		{"Documents/", "/home/user/Documents"},
		{"/Documents/", "/home/user/Documents"},
		{"//Documents//", "/home/user/Documents"},
		{"Documents/deep", "/home/user/Documents/deep"},
		{"/Documents/deep", "/home/user/Documents/deep"},
		{"  ~/Documents  ", "/home/user/Documents"},
		{"~", "/home/user"},
		{".", "/home/user"},
		{"/", "/home/user"},
		{"", "/home/user"},
		{"Documents/../Downloads", "/home/user/Downloads"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := r.Expand(tt.in); got != tt.want {
				t.Errorf("Expand(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExpandTreatsTheThreeFormsAsOne(t *testing.T) {
	r := engine.NewPathResolver(&fakeFileSystem{fsys: fstest.MapFS{}}, "/home/user")

	bare := r.Expand("Documents")
	rooted := r.Expand("/Documents")
	tilde := r.Expand("~/Documents")

	if bare != rooted || bare != tilde {
		t.Errorf("the three forms diverged: %q, %q, %q", bare, rooted, tilde)
	}
}

func TestExpandIgnoresTheWorkingDirectory(t *testing.T) {
	r := engine.NewPathResolver(&fakeFileSystem{fsys: fstest.MapFS{}}, "/home/user")

	cwd, err := os.Getwd()
	if err != nil {
		t.Skip(err)
	}

	got := r.Expand("Documents")
	if got == filepath.Join(cwd, "Documents") {
		t.Error("Expand consulted the working directory; paths are root-relative")
	}
	if got != "/home/user/Documents" {
		t.Errorf("Expand(\"Documents\") = %q", got)
	}
}

func TestResolveSucceedsForAllThreeForms(t *testing.T) {
	r := rootedResolver(t)

	for _, input := range []string{"Documents", "/Documents", "~/Documents", "./Documents"} {
		t.Run(input, func(t *testing.T) {
			got, err := r.Resolve(input)
			if err != nil {
				t.Fatalf("Resolve(%q) error = %v", input, err)
			}
			if got.FSPath != "Documents" {
				t.Errorf("FSPath = %q, want \"Documents\"", got.FSPath)
			}
			if got.Input != input {
				t.Errorf("Input = %q, want the path as typed", got.Input)
			}
		})
	}
}

func TestResolveRootItself(t *testing.T) {
	r := rootedResolver(t)

	for _, input := range []string{".", "~", "/", ""} {
		got, err := r.Resolve(input)
		if err != nil {
			t.Fatalf("Resolve(%q) error = %v", input, err)
		}
		if got.FSPath != "." {
			t.Errorf("Resolve(%q).FSPath = %q, want \".\"", input, got.FSPath)
		}
	}
}

func TestResolveNestedPath(t *testing.T) {
	r := rootedResolver(t)

	got, err := r.Resolve("/Documents/deep")
	if err != nil {
		t.Fatalf("Resolve error = %v", err)
	}
	if got.FSPath != "Documents/deep" {
		t.Errorf("FSPath = %q, want \"Documents/deep\"", got.FSPath)
	}
}

func TestResolveClassifiesErrors(t *testing.T) {
	r := rootedResolver(t)

	tests := []struct {
		name  string
		input string
		kind  oerr.Kind
	}{
		{"missing folder", "Documnets", oerr.KindFolderMissing},
		{"missing rooted folder", "/Documnets", oerr.KindFolderMissing},
		{"missing nested", "Documents/nope", oerr.KindFolderMissing},
		{"path is a file", "loose.txt", oerr.KindPathIsFile},
		{"rooted path is a file", "/Documents/notes.txt", oerr.KindPathIsFile},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.Resolve(tt.input)
			if err == nil {
				t.Fatalf("Resolve(%q) succeeded", tt.input)
			}
			if !oerr.Is(err, tt.kind) {
				t.Errorf("Resolve(%q)\n got: %v\nwant kind: %v", tt.input, err, tt.kind)
			}
		})
	}
}

func TestResolveRefusesToEscapeTheRoot(t *testing.T) {
	fsys := fstest.MapFS{"box/inside.txt": {Data: []byte("a")}, "outside.txt": {Data: []byte("b")}}
	r := engine.NewPathResolver(&fakeFileSystem{fsys: fsys}, "/box")

	for _, input := range []string{"..", "../", "../outside.txt", "../..", "inside/../../elsewhere"} {
		t.Run(input, func(t *testing.T) {
			_, err := r.Resolve(input)
			if err == nil {
				t.Fatalf("Resolve(%q) escaped the root", input)
			}
			if !oerr.Is(err, oerr.KindOutsideRoot) {
				t.Errorf("Resolve(%q) error kind = %v, want outside_root", input, err)
			}
		})
	}
}

func TestOutsideRootErrorNamesTheRoot(t *testing.T) {
	r := engine.NewPathResolver(&fakeFileSystem{fsys: fstest.MapFS{}}, "/home/user")

	_, err := r.Resolve("../etc")
	if err == nil {
		t.Fatal("expected an error")
	}
	want := "I can only look inside '/home/user'. '../etc' points outside it."
	if err.Error() != want {
		t.Errorf("\n got: %s\nwant: %s", err.Error(), want)
	}
}

func TestResolveErrorQuotesThePathAsTyped(t *testing.T) {
	r := rootedResolver(t)

	_, err := r.Resolve("/Documnets")
	if err == nil {
		t.Fatal("expected an error")
	}
	want := "I couldn't find a folder at '/Documnets'. Check the path and try again."
	if err.Error() != want {
		t.Errorf("\n got: %s\nwant: %s", err.Error(), want)
	}
}

func TestResolverRootIsCleaned(t *testing.T) {
	r := engine.NewPathResolver(&fakeFileSystem{fsys: fstest.MapFS{}}, "/home/user/")

	if r.Root() != "/home/user" {
		t.Errorf("Root() = %q, want the cleaned \"/home/user\"", r.Root())
	}
}

func TestResolverDefaultsRootToSeparator(t *testing.T) {
	r := engine.NewPathResolver(&fakeFileSystem{fsys: fstest.MapFS{}}, "")

	if r.Root() != "/" {
		t.Errorf("Root() = %q, want \"/\"", r.Root())
	}
}

func TestResolveOnRealFilesystemAnchoredAtRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Documents", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := engine.NewPathResolver(vfs.NewOS(root), root)

	for _, input := range []string{"Documents", "/Documents", "~/Documents"} {
		got, err := r.Resolve(input)
		if err != nil {
			t.Fatalf("Resolve(%q) error = %v", input, err)
		}
		if got.FSPath != "Documents" {
			t.Errorf("Resolve(%q).FSPath = %q, want \"Documents\"", input, got.FSPath)
		}
	}

	if _, err := r.Resolve("/etc"); err == nil {
		t.Error("an absolute system path resolved; everything is root-relative now")
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

	r := engine.NewPathResolver(vfs.NewOS(root), root)

	_, err := r.Resolve("locked/inner")
	if err == nil {
		t.Fatal("Resolve succeeded on an unreadable parent")
	}
	if !oerr.Is(err, oerr.KindNoPermission) && !oerr.Is(err, oerr.KindFolderMissing) {
		t.Errorf("error kind = %v, want no_permission or folder_missing", err)
	}
}
