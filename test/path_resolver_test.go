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

func TestExpandIsRelativeToTheCurrentDirectory(t *testing.T) {
	r := engine.NewPathResolverAt(&fakeFileSystem{fsys: fstest.MapFS{}}, "/work/project", "/home/user")

	tests := []struct {
		in   string
		want string
	}{
		{"Documents", "/work/project/Documents"},
		{"./Documents", "/work/project/Documents"},
		{"Documents/", "/work/project/Documents"},
		{"Documents/deep", "/work/project/Documents/deep"},
		{"Documents/../Downloads", "/work/project/Downloads"},
		{"", "/work/project"},
		{".", "/work/project"},

		{"/Documents", "/Documents"},
		{"/Documents/", "/Documents"},
		{"//Documents//", "/Documents"},
		{"/etc/hosts", "/etc/hosts"},
		{"/", "/"},

		{"~", "/home/user"},
		{"~/Documents", "/home/user/Documents"},
		{"  ~/Documents  ", "/home/user/Documents"},

		{"..", "/work"},
		{"../..", "/"},
		{"../sibling", "/work/sibling"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := r.Expand(tt.in); got != tt.want {
				t.Errorf("Expand(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExpandTreatsTheThreeFormsAsDifferent(t *testing.T) {
	r := engine.NewPathResolverAt(&fakeFileSystem{fsys: fstest.MapFS{}}, "/work/project", "/home/user")

	bare := r.Expand("Documents")
	absolute := r.Expand("/Documents")
	tilde := r.Expand("~/Documents")

	if bare == absolute || bare == tilde || absolute == tilde {
		t.Errorf("these must now be three different places: %q, %q, %q", bare, absolute, tilde)
	}
	if bare != "/work/project/Documents" {
		t.Errorf("bare = %q, want it relative to the current directory", bare)
	}
	if absolute != "/Documents" {
		t.Errorf("absolute = %q, want the real absolute path", absolute)
	}
	if tilde != "/home/user/Documents" {
		t.Errorf("tilde = %q, want it under home", tilde)
	}
}

func TestChdirMovesTheCurrentDirectory(t *testing.T) {
	fsys := fstest.MapFS{
		"work/project/src/x.go": {Data: []byte("a")},
		"work/other/y.go":       {Data: []byte("b")},
	}
	r := engine.NewPathResolverAt(&fakeFileSystem{fsys: fsys}, "/work/project", "/home/user")

	if _, err := r.Chdir("src"); err != nil {
		t.Fatalf("Chdir(\"src\") error = %v", err)
	}
	if r.Dir() != "/work/project/src" {
		t.Errorf("Dir() = %q", r.Dir())
	}

	if _, err := r.Chdir("../../other"); err != nil {
		t.Fatalf("Chdir upward error = %v", err)
	}
	if r.Dir() != "/work/other" {
		t.Errorf("Dir() = %q, want /work/other", r.Dir())
	}

	if _, err := r.Chdir("-"); err != nil {
		t.Fatalf("Chdir(\"-\") error = %v", err)
	}
	if r.Dir() != "/work/project/src" {
		t.Errorf("cd - gave %q, want the previous directory", r.Dir())
	}
}

func TestChdirBareGoesHome(t *testing.T) {
	fsys := fstest.MapFS{"home/user/x.txt": {Data: []byte("a")}, "work/y.txt": {Data: []byte("b")}}
	r := engine.NewPathResolverAt(&fakeFileSystem{fsys: fsys}, "/work", "/home/user")

	if _, err := r.Chdir(""); err != nil {
		t.Fatalf("bare Chdir error = %v", err)
	}
	if r.Dir() != "/home/user" {
		t.Errorf("Dir() = %q, want home", r.Dir())
	}
}

func TestChdirRejectsMissingAndFiles(t *testing.T) {
	fsys := fstest.MapFS{"work/file.txt": {Data: []byte("a")}}
	r := engine.NewPathResolver(&fakeFileSystem{fsys: fsys}, "/work")

	if _, err := r.Chdir("nowhere"); err == nil {
		t.Error("Chdir into a missing folder succeeded")
	}
	if _, err := r.Chdir("file.txt"); !oerr.Is(err, oerr.KindCannotChangeDir) {
		t.Errorf("Chdir into a file gave %v, want cannot_change_dir", err)
	}
	if r.Dir() != "/work" {
		t.Errorf("a failed Chdir moved the directory to %q", r.Dir())
	}
}

func TestDisplayAbbreviatesHome(t *testing.T) {
	r := engine.NewPathResolverAt(&fakeFileSystem{fsys: fstest.MapFS{}}, "/home/user/work", "/home/user")

	tests := []struct {
		in   string
		want string
	}{
		{"/home/user", "~"},
		{"/home/user/work", "~/work"},
		{"/home/user/work/deep", "~/work/deep"},
		{"/etc", "/etc"},
		{"/home/userother", "/home/userother"},
	}

	for _, tt := range tests {
		if got := r.Display(tt.in); got != tt.want {
			t.Errorf("Display(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestResolveSucceedsFromRootWhereAllFormsAgree(t *testing.T) {
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

func TestResolveWalksAboveTheStartDirectory(t *testing.T) {
	fsys := fstest.MapFS{
		"box/inside.txt": {Data: []byte("a")},
		"sibling/x.txt":  {Data: []byte("b")},
	}
	r := engine.NewPathResolver(&fakeFileSystem{fsys: fsys}, "/box")

	tests := []struct {
		input string
		want  string
	}{
		{"..", "."},
		{"../", "."},
		{"../sibling", "sibling"},
		{"inside/../../sibling", "sibling"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := r.Resolve(tt.input)
			if err != nil {
				t.Fatalf("Resolve(%q) error = %v; \"..\" is allowed now", tt.input, err)
			}
			if got.FSPath != tt.want {
				t.Errorf("Resolve(%q).FSPath = %q, want %q", tt.input, got.FSPath, tt.want)
			}
		})
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

func TestResolverDirIsCleaned(t *testing.T) {
	r := engine.NewPathResolver(&fakeFileSystem{fsys: fstest.MapFS{}}, "/home/user/")

	if r.Dir() != "/home/user" {
		t.Errorf("Dir() = %q, want the cleaned \"/home/user\"", r.Dir())
	}
}

func TestResolverDefaultsDirToSeparator(t *testing.T) {
	r := engine.NewPathResolver(&fakeFileSystem{fsys: fstest.MapFS{}}, "")

	if r.Dir() != "/" {
		t.Errorf("Dir() = %q, want \"/\"", r.Dir())
	}
}

func TestResolveOnRealFilesystemUsesTheCurrentDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Documents", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := engine.NewPathResolver(vfs.OS(), root)

	relative, err := r.Resolve("Documents")
	if err != nil {
		t.Fatalf("Resolve(\"Documents\") error = %v", err)
	}
	if relative.Absolute != filepath.Join(root, "Documents") {
		t.Errorf("Absolute = %q, want it under the start directory", relative.Absolute)
	}

	absolute, err := r.Resolve(filepath.Join(root, "Documents", "deep"))
	if err != nil {
		t.Fatalf("an absolute path must resolve: %v", err)
	}
	if absolute.Absolute != filepath.Join(root, "Documents", "deep") {
		t.Errorf("Absolute = %q", absolute.Absolute)
	}

	if _, err := r.Resolve("/etc"); err != nil {
		t.Errorf("/etc must be reachable now: %v", err)
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

	r := engine.NewPathResolver(vfs.OS(), root)

	_, err := r.Resolve("locked/inner")
	if err == nil {
		t.Fatal("Resolve succeeded on an unreadable parent")
	}
	if !oerr.Is(err, oerr.KindNoPermission) && !oerr.Is(err, oerr.KindFolderMissing) {
		t.Errorf("error kind = %v, want no_permission or folder_missing", err)
	}
}
