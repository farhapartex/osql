package test

import (
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
)

func entryFor(t *testing.T, fsys fs.FS, dir, name string) engine.Entry {
	t.Helper()

	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		t.Fatalf("ReadDir(%q) error = %v", dir, err)
	}
	for _, e := range entries {
		if e.Name() == name {
			full := name
			if dir != "." {
				full = dir + "/" + name
			}
			return engine.Entry{DirEntry: e, Path: full}
		}
	}
	t.Fatalf("%q not found in %q", name, dir)
	return engine.Entry{}
}

func sampleFS() fstest.MapFS {
	return fstest.MapFS{
		"notes.txt":            {Data: []byte("a")},
		"report.PDF":           {Data: []byte("b")},
		"Makefile":             {Data: []byte("c")},
		"archive.tar.gz":       {Data: []byte("d")},
		"empty/.keep":          {Data: []byte("")},
		"one/a.txt":            {Data: []byte("")},
		"three/a.txt":          {Data: []byte("")},
		"three/b.txt":          {Data: []byte("")},
		"three/c/d.txt":        {Data: []byte("")},
		"nested/deep/file.txt": {Data: []byte("")},
	}
}

func TestNameFieldExtract(t *testing.T) {
	fsys := sampleFS()
	f := engine.NameField{}

	got, err := f.Extract(entryFor(t, fsys, ".", "notes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "notes.txt" {
		t.Errorf("Extract() = %q, want \"notes.txt\"", got.Text)
	}
	if got.IsNum {
		t.Error("name must not be numeric")
	}
}

func TestNameFieldMetadata(t *testing.T) {
	f := engine.NameField{}

	if f.Field() != "name" {
		t.Errorf("Field() = %q", f.Field())
	}
	if f.Cost() != engine.CostFree {
		t.Errorf("Cost() = %d, want free; the name comes from the directory read", f.Cost())
	}
	if !slices.Equal(f.AllowedOperators(), []string{"=", "!="}) {
		t.Errorf("AllowedOperators() = %v, want [= !=]", f.AllowedOperators())
	}
	for _, target := range []query.Target{query.TargetAll, query.TargetFiles, query.TargetFolders} {
		if !f.AppliesTo(target) {
			t.Errorf("AppliesTo(%v) = false, want true", target)
		}
	}
}

func TestTypeFieldExtract(t *testing.T) {
	fsys := sampleFS()
	f := engine.TypeField{}

	tests := []struct {
		dir  string
		name string
		want string
	}{
		{".", "notes.txt", "txt"},
		{".", "report.PDF", "PDF"},
		{".", "Makefile", ""},
		{".", "archive.tar.gz", "gz"},
		{".", "empty", "folder"},
		{".", "nested", "folder"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := f.Extract(entryFor(t, fsys, tt.dir, tt.name))
			if err != nil {
				t.Fatal(err)
			}
			if got.Text != tt.want {
				t.Errorf("Extract(%q) = %q, want %q", tt.name, got.Text, tt.want)
			}
		})
	}
}

func TestTypeFieldNormalizesLeadingDot(t *testing.T) {
	f := engine.TypeField{}

	withDot, _ := f.NormalizeValue(".txt")
	withoutDot, _ := f.NormalizeValue("txt")

	if withDot.Text != withoutDot.Text {
		t.Errorf("NormalizeValue(\".txt\") = %q but NormalizeValue(\"txt\") = %q; they must be one query", withDot.Text, withoutDot.Text)
	}
	if withDot.Text != "txt" {
		t.Errorf("normalised to %q, want \"txt\"", withDot.Text)
	}
}

func TestTypeFieldPreservesValueCase(t *testing.T) {
	f := engine.TypeField{}

	got, _ := f.NormalizeValue(".TXT")
	if got.Text != "TXT" {
		t.Errorf("NormalizeValue(\".TXT\") = %q, want \"TXT\"; values are case-sensitive", got.Text)
	}
}

func TestTypeFieldExtractMatchesDisplayedColumn(t *testing.T) {
	fsys := sampleFS()
	f := engine.TypeField{}

	extracted, _ := f.Extract(entryFor(t, fsys, ".", "notes.txt"))
	normalised, _ := f.NormalizeValue("txt")

	if extracted.Text != normalised.Text {
		t.Errorf("extracted %q but a query for %q normalises to %q; a value read off the table must be pasteable", extracted.Text, "txt", normalised.Text)
	}
}

func TestNameLikeFieldExtractsBaseName(t *testing.T) {
	fsys := sampleFS()
	f := engine.NameLikeField{}

	got, err := f.Extract(entryFor(t, fsys, "nested", "deep"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "deep" {
		t.Errorf("Extract() = %q, want \"deep\"", got.Text)
	}
}

func TestCountChildFieldExtract(t *testing.T) {
	fsys := sampleFS()
	f := engine.NewCountChildField(&fakeFileSystem{fsys: fsys})

	tests := []struct {
		name string
		want int64
	}{
		{"empty", 1},
		{"one", 1},
		{"three", 3},
		{"nested", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := f.Extract(entryFor(t, fsys, ".", tt.name))
			if err != nil {
				t.Fatal(err)
			}
			if !got.IsNum {
				t.Fatal("count(child) must produce a numeric value")
			}
			if got.Number != tt.want {
				t.Errorf("Extract(%q) = %d, want %d", tt.name, got.Number, tt.want)
			}
		})
	}
}

func TestCountChildFieldOnFileIsZero(t *testing.T) {
	fsys := sampleFS()
	f := engine.NewCountChildField(&fakeFileSystem{fsys: fsys})

	got, err := f.Extract(entryFor(t, fsys, ".", "notes.txt"))
	if err != nil {
		t.Fatalf("Extract on a file error = %v", err)
	}
	if !got.IsNum || got.Number != 0 {
		t.Errorf("Extract on a file = %+v, want numeric 0", got)
	}
}

func TestCountChildFieldMetadata(t *testing.T) {
	f := engine.NewCountChildField(nil)

	if f.Cost() != engine.CostReadDir {
		t.Errorf("Cost() = %d, want the ReadDir cost; it is the expensive predicate", f.Cost())
	}
	if f.Cost() <= (engine.NameField{}).Cost() {
		t.Error("count(child) must cost more than name so it sorts last")
	}
	if !slices.Equal(f.AllowedOperators(), []string{"=", "!=", "<", ">", "<=", ">="}) {
		t.Errorf("AllowedOperators() = %v", f.AllowedOperators())
	}
	if f.AppliesTo(query.TargetFiles) {
		t.Error("AppliesTo(files) = true; count(child) describes folders")
	}
	if !f.AppliesTo(query.TargetFolders) || !f.AppliesTo(query.TargetAll) {
		t.Error("count(child) must apply to folders and all")
	}
}

func TestCountChildFieldNormalizeValue(t *testing.T) {
	f := engine.NewCountChildField(nil)

	tests := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"10", 10, false},
		{"0", 0, false},
		{"  7  ", 7, false},
		{"-3", -3, false},
		{"9223372036854775807", 9223372036854775807, false},
		{"abc", 0, true},
		{"", 0, true},
		{"1.5", 0, true},
		{"1e3", 0, true},
		{"9223372036854775808", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := f.NormalizeValue(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizeValue(%q) accepted a non-number", tt.in)
				}
				if !oerr.Is(err, oerr.KindCountChildNonNumeric) {
					t.Errorf("error kind = %v, want count_child_non_numeric", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeValue(%q) error = %v", tt.in, err)
			}
			if !got.IsNum || got.Number != tt.want {
				t.Errorf("NormalizeValue(%q) = %+v, want numeric %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestCountChildFieldWithoutFileSystemFails(t *testing.T) {
	fsys := sampleFS()
	f := engine.NewCountChildField(nil)

	if _, err := f.Extract(entryFor(t, fsys, ".", "three")); err == nil {
		t.Error("Extract with no filesystem succeeded; want an error")
	}
}

func TestCountChildFieldPropagatesReadDirError(t *testing.T) {
	fsys := fstest.MapFS{"dir/a.txt": {Data: []byte("")}}
	f := engine.NewCountChildField(&fakeFileSystem{fsys: fsys})

	entry := entryFor(t, fsys, ".", "dir")
	broken := engine.Entry{DirEntry: entry.DirEntry, Path: "does/not/exist"}

	if _, err := f.Extract(broken); err == nil {
		t.Error("Extract on an unreadable directory succeeded; want the error surfaced")
	}
}
