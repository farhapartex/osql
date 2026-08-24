package test

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/farhapartex/osql/internal/vfs"
)

func conformanceLayout() map[string]string {
	return map[string]string{
		"notes.txt":           "hello",
		"report.pdf":          "pdf",
		"sub/a.txt":           "a",
		"sub/b.txt":           "b",
		"sub/deep/c.txt":      "c",
		"empty_looking/.keep": "",
		"space dir/file.txt":  "s",
		"unicode/日本語.txt":     "u",
	}
}

func mapFSFixture(t *testing.T) vfs.FileSystem {
	t.Helper()

	m := fstest.MapFS{}
	for path, content := range conformanceLayout() {
		m[path] = &fstest.MapFile{Data: []byte(content)}
	}
	return &fakeFileSystem{fsys: m}
}

func osFSFixture(t *testing.T) vfs.FileSystem {
	t.Helper()

	root := t.TempDir()
	for path, content := range conformanceLayout() {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return vfs.NewOS(root)
}

func TestFileSystemConformance(t *testing.T) {
	impls := map[string]func(*testing.T) vfs.FileSystem{
		"MapFS": mapFSFixture,
		"os":    osFSFixture,
	}

	for name, build := range impls {
		t.Run(name, func(t *testing.T) {
			fsys := build(t)

			t.Run("stat file", func(t *testing.T) {
				info, err := fsys.Stat("notes.txt")
				if err != nil {
					t.Fatalf("Stat error = %v", err)
				}
				if info.IsDir() {
					t.Error("IsDir() = true for a file")
				}
				if info.Name() != "notes.txt" {
					t.Errorf("Name() = %q", info.Name())
				}
				if info.Size() != int64(len("hello")) {
					t.Errorf("Size() = %d, want %d", info.Size(), len("hello"))
				}
			})

			t.Run("stat dir", func(t *testing.T) {
				info, err := fsys.Stat("sub")
				if err != nil {
					t.Fatalf("Stat error = %v", err)
				}
				if !info.IsDir() {
					t.Error("IsDir() = false for a directory")
				}
			})

			t.Run("stat root", func(t *testing.T) {
				info, err := fsys.Stat(".")
				if err != nil {
					t.Fatalf("Stat(\".\") error = %v", err)
				}
				if !info.IsDir() {
					t.Error("root is not a directory")
				}
			})

			t.Run("readdir sorted", func(t *testing.T) {
				entries, err := fsys.ReadDir("sub")
				if err != nil {
					t.Fatalf("ReadDir error = %v", err)
				}

				names := make([]string, 0, len(entries))
				for _, e := range entries {
					names = append(names, e.Name())
				}
				want := []string{"a.txt", "b.txt", "deep"}
				if !slices.Equal(names, want) {
					t.Errorf("ReadDir(\"sub\") = %v, want %v in sorted order", names, want)
				}
			})

			t.Run("readdir marks directories", func(t *testing.T) {
				entries, err := fsys.ReadDir("sub")
				if err != nil {
					t.Fatal(err)
				}
				for _, e := range entries {
					if e.Name() == "deep" && !e.IsDir() {
						t.Error("deep should be a directory")
					}
					if e.Name() == "a.txt" && e.IsDir() {
						t.Error("a.txt should not be a directory")
					}
				}
			})

			t.Run("readdir root", func(t *testing.T) {
				entries, err := fsys.ReadDir(".")
				if err != nil {
					t.Fatalf("ReadDir(\".\") error = %v", err)
				}
				if len(entries) == 0 {
					t.Error("root listing is empty")
				}
			})

			t.Run("open and read", func(t *testing.T) {
				file, err := fsys.Open("notes.txt")
				if err != nil {
					t.Fatalf("Open error = %v", err)
				}
				defer file.Close()

				data, err := io.ReadAll(file)
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != "hello" {
					t.Errorf("contents = %q, want \"hello\"", data)
				}
			})

			t.Run("paths with spaces and unicode", func(t *testing.T) {
				if _, err := fsys.Stat("space dir/file.txt"); err != nil {
					t.Errorf("Stat on a path with a space: %v", err)
				}
				if _, err := fsys.Stat("unicode/日本語.txt"); err != nil {
					t.Errorf("Stat on a unicode path: %v", err)
				}
			})

			t.Run("missing path", func(t *testing.T) {
				_, err := fsys.Stat("nope.txt")
				if err == nil {
					t.Fatal("Stat on a missing path succeeded")
				}
				if !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("error = %v, want fs.ErrNotExist", err)
				}
			})

			t.Run("missing directory readdir", func(t *testing.T) {
				if _, err := fsys.ReadDir("nope"); err == nil {
					t.Error("ReadDir on a missing directory succeeded")
				}
			})

			t.Run("readdir on a file", func(t *testing.T) {
				if _, err := fsys.ReadDir("notes.txt"); err == nil {
					t.Error("ReadDir on a file succeeded")
				}
			})

			t.Run("invalid fs paths are rejected", func(t *testing.T) {
				for _, bad := range []string{"/notes.txt", "../notes.txt", "sub/../notes.txt", "", "./notes.txt"} {
					if _, err := fsys.Stat(bad); err == nil {
						t.Errorf("Stat(%q) succeeded; io/fs paths must be unrooted and clean", bad)
					}
				}
			})
		})
	}
}

func TestFSPath(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"/", "."},
		{"//", "."},
		{"/..", "."},
		{"/Users/x", "Users/x"},
		{"/Users/x/", "Users/x"},
		{"/Users//x", "Users/x"},
		{"/Users/./x", "Users/x"},
		{"/Users/y/../x", "Users/x"},
		{"/Users/x/..", "Users"},
		{"/a", "a"},
		{"/space dir/file.txt", "space dir/file.txt"},
		{"/unicode/日本語.txt", "unicode/日本語.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := vfs.FSPath(tt.in)
			if err != nil {
				t.Fatalf("FSPath(%q) error = %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("FSPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
			if !fs.ValidPath(got) {
				t.Errorf("FSPath(%q) = %q which fs.ValidPath rejects", tt.in, got)
			}
		})
	}
}

func TestFSPathRejectsRelativeInput(t *testing.T) {
	for _, in := range []string{"", "Documents", "./Documents", "../Documents", "~", "~/Documents"} {
		t.Run(in, func(t *testing.T) {
			if _, err := vfs.FSPath(in); !errors.Is(err, vfs.ErrNotAbsolute) {
				t.Errorf("FSPath(%q) error = %v, want ErrNotAbsolute; expansion happens before this layer", in, err)
			}
		})
	}
}

func TestFSPathOutputAlwaysSurvivesValidPath(t *testing.T) {
	inputs := []string{
		"/", "/a", "/a/b/c", "/a/../b", "/./a", "/a/b/../../c",
		"/very/deeply/nested/path/that/goes/on",
	}

	for _, in := range inputs {
		got, err := vfs.FSPath(in)
		if err != nil {
			t.Errorf("FSPath(%q) error = %v", in, err)
			continue
		}
		if !fs.ValidPath(got) {
			t.Errorf("FSPath(%q) produced %q, rejected by fs.ValidPath", in, got)
		}
	}
}

func TestOSPathIsTheInverseOfFSPath(t *testing.T) {
	tests := []struct {
		root   string
		fsPath string
		want   string
	}{
		{"/", ".", "/"},
		{"/", "Users/x", "/Users/x"},
		{"/tmp/root", ".", "/tmp/root"},
		{"/tmp/root", "a/b", "/tmp/root/a/b"},
		{"/tmp/root", "", "/tmp/root"},
	}

	for _, tt := range tests {
		if got := vfs.OSPath(tt.root, tt.fsPath); got != tt.want {
			t.Errorf("OSPath(%q, %q) = %q, want %q", tt.root, tt.fsPath, got, tt.want)
		}
	}
}

func TestFSPathRoundTrip(t *testing.T) {
	for _, absolute := range []string{"/", "/Users/x", "/a/b/c", "/space dir/f.txt"} {
		fsPath, err := vfs.FSPath(absolute)
		if err != nil {
			t.Fatalf("FSPath(%q) error = %v", absolute, err)
		}
		if got := vfs.OSPath("/", fsPath); got != filepath.Clean(absolute) {
			t.Errorf("round trip of %q gave %q", absolute, got)
		}
	}
}

func TestJoin(t *testing.T) {
	tests := []struct {
		base string
		name string
		want string
	}{
		{".", "notes.txt", "notes.txt"},
		{"", "notes.txt", "notes.txt"},
		{"sub", "a.txt", "sub/a.txt"},
		{"sub/deep", "c.txt", "sub/deep/c.txt"},
	}

	for _, tt := range tests {
		got := vfs.Join(tt.base, tt.name)
		if got != tt.want {
			t.Errorf("Join(%q, %q) = %q, want %q", tt.base, tt.name, got, tt.want)
		}
		if !fs.ValidPath(got) {
			t.Errorf("Join(%q, %q) = %q, rejected by fs.ValidPath", tt.base, tt.name, got)
		}
	}
}

func TestOSFileSystemRootAndOSPath(t *testing.T) {
	root := t.TempDir()
	f := vfs.NewOS(root)

	if f.Root() != root {
		t.Errorf("Root() = %q, want %q", f.Root(), root)
	}
	if got, want := f.OSPath("a/b"), filepath.Join(root, "a", "b"); got != want {
		t.Errorf("OSPath(\"a/b\") = %q, want %q", got, want)
	}
	if got := f.OSPath("."); got != root {
		t.Errorf("OSPath(\".\") = %q, want %q", got, root)
	}
}

func TestOSDefaultsToFilesystemRoot(t *testing.T) {
	f := vfs.OS()

	if f.Root() != "/" {
		t.Errorf("OS().Root() = %q, want \"/\"", f.Root())
	}
	if _, err := f.Stat("."); err != nil {
		t.Errorf("Stat(\".\") on the real root error = %v", err)
	}
}

func TestOSFileSystemSatisfiesInterface(t *testing.T) {
	var _ vfs.FileSystem = (*vfs.OSFileSystem)(nil)
	var _ fs.StatFS = (*vfs.OSFileSystem)(nil)
	var _ fs.ReadDirFS = (*vfs.OSFileSystem)(nil)
}

func TestOSFileSystemReadsRealFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "real.txt"), []byte("on disk"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := vfs.NewOS(root)

	info, err := f.Stat("real.txt")
	if err != nil {
		t.Fatalf("Stat error = %v", err)
	}
	if info.Size() != int64(len("on disk")) {
		t.Errorf("Size() = %d, want %d", info.Size(), len("on disk"))
	}

	entries, err := f.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "real.txt" {
		t.Errorf("ReadDir(\".\") = %v", entries)
	}
}

func TestOSFileSystemDoesNotEscapeItsRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "inner")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(filepath.Dir(root), "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := vfs.NewOS(root)

	for _, escape := range []string{"../outside.txt", "..", "/etc/passwd"} {
		if _, err := f.Stat(escape); err == nil {
			t.Errorf("Stat(%q) succeeded; the root must be a boundary", escape)
		}
	}
}
