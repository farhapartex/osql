package test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/output"
)

func render(t *testing.T, rows []engine.Row) string {
	t.Helper()

	buf := &bytes.Buffer{}
	if err := output.NewTable().Render(buf, rows); err != nil {
		t.Fatalf("Render error = %v", err)
	}
	return buf.String()
}

func sampleRows() []engine.Row {
	stamp := time.Date(2026, 8, 20, 14, 2, 0, 0, time.UTC)
	return []engine.Row{
		{Name: "notes.txt", Ext: "txt", Size: 4300, Modified: stamp},
		{Name: "report.pdf", Ext: "pdf", Size: 1153434, Modified: stamp.Add(-time.Hour)},
		{Name: "archive.tar.gz", Ext: "gz", Size: 2469606195, Modified: stamp.Add(-48 * time.Hour)},
		{Name: "goupp", IsDir: true, Modified: stamp.Add(-72 * time.Hour)},
	}
}

func TestTableRendersExactOutput(t *testing.T) {
	got := render(t, sampleRows())

	want := strings.Join([]string{
		"NAME            TYPE    SIZE    MODIFIED",
		"notes.txt       txt     4.2 KB  2026-08-20 14:02",
		"report.pdf      pdf     1.1 MB  2026-08-20 13:02",
		"archive.tar.gz  gz      2.3 GB  2026-08-18 14:02",
		"goupp           folder  —       2026-08-17 14:02",
		"",
		"4 files",
		"",
	}, "\n")

	if got != want {
		t.Errorf("table mismatch\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestTableHeaderColumns(t *testing.T) {
	got := render(t, sampleRows())
	header := strings.Split(got, "\n")[0]

	for _, column := range []string{"NAME", "TYPE", "SIZE", "MODIFIED"} {
		if !strings.Contains(header, column) {
			t.Errorf("header %q missing column %q", header, column)
		}
	}
	if strings.Contains(header, "PATH") {
		t.Error("header carries a PATH column; recursive results reuse NAME")
	}
	if len(strings.Fields(header)) != 4 {
		t.Errorf("header has %d columns, want exactly 4", len(strings.Fields(header)))
	}
}

func TestTableColumnsAlign(t *testing.T) {
	got := render(t, sampleRows())
	lines := strings.Split(got, "\n")

	typeColumn := strings.Index(lines[0], "TYPE")
	if typeColumn < 0 {
		t.Fatal("TYPE column not found")
	}

	for _, line := range lines[1:5] {
		if len(line) <= typeColumn {
			t.Fatalf("row shorter than the TYPE column offset: %q", line)
		}
		if line[typeColumn-1] != ' ' {
			t.Errorf("row %q does not align at the TYPE column", line)
		}
	}
}

func TestTableFooterIsSeparatedByABlankLine(t *testing.T) {
	got := render(t, sampleRows())
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")

	if len(lines) < 2 {
		t.Fatalf("output too short: %q", got)
	}
	if lines[len(lines)-1] != "4 files" {
		t.Errorf("last line = %q, want \"4 files\"", lines[len(lines)-1])
	}
	if lines[len(lines)-2] != "" {
		t.Errorf("footer is not preceded by a blank line: %q", lines[len(lines)-2])
	}
}

func TestTableSingleRowSaysOneFiles(t *testing.T) {
	got := render(t, sampleRows()[:1])

	if !strings.Contains(got, "1 files") {
		t.Errorf("footer should read \"1 files\" whatever the count:\n%s", got)
	}
	if strings.Contains(got, "1 file\n") {
		t.Error("footer was singularised")
	}
}

func TestTableEmptyRowsStillRendersHeaderAndFooter(t *testing.T) {
	got := render(t, nil)

	if !strings.Contains(got, "NAME") {
		t.Errorf("empty result dropped the header:\n%s", got)
	}
	if !strings.Contains(got, "0 files") {
		t.Errorf("empty result dropped the footer:\n%s", got)
	}
}

func TestTableRecursiveNamesKeepTheirPath(t *testing.T) {
	rows := []engine.Row{
		{Name: "2025/q4-report.xlsx", Ext: "xlsx", Size: 90521},
		{Name: "report.pdf", Ext: "pdf", Size: 1153434},
	}

	got := render(t, rows)
	if !strings.Contains(got, "2025/q4-report.xlsx") {
		t.Errorf("relative path was not rendered in NAME:\n%s", got)
	}
}

func TestTableHandlesUnicodeAndWideNames(t *testing.T) {
	rows := []engine.Row{
		{Name: "日本語のファイル.txt", Ext: "txt", Size: 1024},
		{Name: "short.txt", Ext: "txt", Size: 1024},
		{Name: strings.Repeat("long", 30) + ".txt", Ext: "txt", Size: 1024},
	}

	got := render(t, rows)
	for _, row := range rows {
		if !strings.Contains(got, row.Name) {
			t.Errorf("name %q missing from output", row.Name)
		}
	}
	if !strings.Contains(got, "3 files") {
		t.Error("footer count wrong for unicode rows")
	}
}

func TestTableMissingModifiedShowsAbsent(t *testing.T) {
	rows := []engine.Row{{Name: "vanished.txt", Ext: "txt", Size: 10}}

	got := render(t, rows)
	if !strings.Contains(got, "—") {
		t.Errorf("a zero timestamp should render as an em dash:\n%s", got)
	}
}

func TestTableSurfacesWriteErrors(t *testing.T) {
	if err := output.NewTable().Render(errWriter{}, sampleRows()); err == nil {
		t.Error("Render swallowed a write failure")
	}
}

func TestLinesRendererStillWorks(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := output.NewLines().Render(buf, sampleRows()); err != nil {
		t.Fatalf("Render error = %v", err)
	}

	want := "notes.txt\nreport.pdf\narchive.tar.gz\ngoupp\n"
	if buf.String() != want {
		t.Errorf("lines output = %q, want %q", buf.String(), want)
	}
}

func TestLinesRendererSurfacesWriteErrors(t *testing.T) {
	if err := output.NewLines().Render(errWriter{}, sampleRows()); err == nil {
		t.Error("Render swallowed a write failure")
	}
}

func TestBothRenderersSatisfyTheInterface(t *testing.T) {
	renderers := []output.Renderer{output.NewTable(), output.NewLines()}

	for _, r := range renderers {
		buf := &bytes.Buffer{}
		if err := r.Render(buf, sampleRows()); err != nil {
			t.Errorf("%T.Render error = %v", r, err)
		}
		if buf.Len() == 0 {
			t.Errorf("%T wrote nothing", r)
		}
	}
}
