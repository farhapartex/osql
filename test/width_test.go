package test

import (
	"strings"
	"testing"
	"time"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/output"
)

func bigSummary() engine.Summary {
	stamp := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	return engine.Summary{
		Path:      "Documents",
		Files:     52,
		Folders:   29,
		TotalSize: 87556096,
		Types: []engine.TypeTally{
			{Ext: "pdf", Count: 20, Size: 51484426},
			{Ext: "png", Count: 9, Size: 16986931},
		},
		MoreTypes: 10,
		Largest: []engine.Row{
			rowNamed("Designing Data Intensive Applications by Martin Kleppmann.pdf", 24955781),
			rowNamed("IMG_6594_YT.png", 13631488),
		},
		Oldest: stamp,
		Newest: stamp,
	}
}

func rowNamed(name string, size int64) engine.Row {
	return engine.Row{Name: name, Size: size}
}

func TestDisplayWidth(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"café", 4},
		{"日本語", 6},
		{"日本語abc", 9},
		{"한국어", 6},
		{"ｆｕｌｌ", 8},
		{"→", 1},
		{"a→b", 3},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := output.DisplayWidth(tt.in); got != tt.want {
				t.Errorf("DisplayWidth(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestRuneWidth(t *testing.T) {
	narrow := []rune{'a', 'Z', '0', ' ', '—', 'é', '→'}
	wide := []rune{'日', '本', '한', 'Ａ'}

	for _, r := range narrow {
		if got := output.RuneWidth(r); got != 1 {
			t.Errorf("RuneWidth(%q) = %d, want 1", r, got)
		}
	}
	for _, r := range wide {
		if got := output.RuneWidth(r); got != 2 {
			t.Errorf("RuneWidth(%q) = %d, want 2", r, got)
		}
	}
}

func TestSummaryColumnsAlignAcrossBlocks(t *testing.T) {
	got := renderSummary(t, bigSummary())

	var whatHeader, typeHeader string
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "WHAT") {
			whatHeader = line
		}
		if strings.Contains(line, "TYPE") {
			typeHeader = line
		}
	}

	if whatHeader == "" || typeHeader == "" {
		t.Fatalf("headers not found:\n%s", got)
	}
	if strings.Index(whatHeader, "COUNT") != strings.Index(typeHeader, "COUNT") {
		t.Errorf("COUNT column differs between blocks:\n%q\n%q", whatHeader, typeHeader)
	}
	if strings.Index(whatHeader, "SIZE") != strings.Index(typeHeader, "SIZE") {
		t.Errorf("SIZE column differs between blocks:\n%q\n%q", whatHeader, typeHeader)
	}
}

func TestSummaryRowsAlignWithTheirHeader(t *testing.T) {
	got := renderSummary(t, bigSummary())
	lines := strings.Split(got, "\n")

	for i, line := range lines {
		if !strings.Contains(line, "WHAT") {
			continue
		}
		want := len(line)
		for _, row := range lines[i+1 : i+4] {
			if len(strings.TrimRight(row, " ")) > want+2 {
				t.Errorf("row %q is wider than its header %q", row, line)
			}
		}
	}
}

func TestSummaryLongNamesDoNotStretchTheTable(t *testing.T) {
	got := renderSummary(t, bigSummary())

	for _, line := range strings.Split(got, "\n") {
		if width := output.DisplayWidth(line); width > 72 {
			t.Errorf("line is %d columns wide, which will wrap on a normal terminal:\n%q", width, line)
		}
	}
}

func TestSummaryTruncatesLongNamesInTheMiddle(t *testing.T) {
	got := renderSummary(t, bigSummary())

	if !strings.Contains(got, "…") {
		t.Errorf("a very long name was not truncated:\n%s", got)
	}
	if strings.Contains(got, "Designing Data Intensive Applications by Martin Kleppmann.pdf") {
		t.Error("the full long name was printed, stretching the table")
	}
	if !strings.Contains(got, "Designing") {
		t.Error("truncation lost the start of the name")
	}
	if !strings.Contains(got, ".pdf") {
		t.Error("truncation lost the extension; middle truncation keeps both ends")
	}
}

func TestSummaryShortNamesAreNotTruncated(t *testing.T) {
	got := renderSummary(t, bigSummary())

	if !strings.Contains(got, "IMG_6594_YT.png") {
		t.Errorf("a short name was altered:\n%s", got)
	}
}

func TestSummaryWideNamesStayInTheirColumn(t *testing.T) {
	s := bigSummary()
	s.Largest = append(s.Largest, rowNamed("日本語のとても長いファイル名前です.txt", 100))

	got := renderSummary(t, s)

	var sizeColumn = -1
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "LARGEST") {
			sizeColumn = output.DisplayWidth(line[:strings.Index(line, "SIZE")])
			break
		}
	}
	if sizeColumn < 0 {
		t.Fatal("LARGEST header not found")
	}

	for _, line := range strings.Split(got, "\n") {
		if !strings.Contains(line, "日本語") {
			continue
		}
		if output.DisplayWidth(line) > 72 {
			t.Errorf("a wide-character name overflowed the table: %q (%d columns)", line, output.DisplayWidth(line))
		}
	}
}
