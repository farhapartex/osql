package test

import (
	"strings"
	"testing"
	"time"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/output"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{512, "512 B"},
		{938, "938 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1025, "1.0 KB"},
		{1536, "1.5 KB"},
		{4300, "4.2 KB"},
		{123456, "120.6 KB"},
		{1048575, "1.0 MB"},
		{1048576, "1.0 MB"},
		{1153434, "1.1 MB"},
		{123500000, "117.8 MB"},
		{1073741823, "1.0 GB"},
		{1073741824, "1.0 GB"},
		{2469606195, "2.3 GB"},
		{1099511627776, "1.0 TB"},
		{1539270819840, "1.4 TB"},
		{1125899906842624, "1.0 PB"},
		{1152921504606846976, "1.0 EB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := output.FormatSize(tt.bytes); got != tt.want {
				t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestFormatSizeNeverShows1024OfAUnit(t *testing.T) {
	boundaries := []int64{
		1048575, 1048576,
		1073741823, 1073741824,
		1099511627775, 1099511627776,
	}

	for _, b := range boundaries {
		got := output.FormatSize(b)
		if strings.HasPrefix(got, "1024.0") {
			t.Errorf("FormatSize(%d) = %q; 1024 of a unit is one of the next", b, got)
		}
	}
}

func TestFormatSizeExhaustiveBoundarySweep(t *testing.T) {
	var step int64 = 1024
	for unit := range 5 {
		below := step - 1
		exact := step

		gotBelow := output.FormatSize(below)
		gotExact := output.FormatSize(exact)

		if strings.HasPrefix(gotBelow, "1024") {
			t.Errorf("FormatSize(%d) = %q at unit %d", below, gotBelow, unit)
		}
		if strings.HasPrefix(gotExact, "1024") {
			t.Errorf("FormatSize(%d) = %q at unit %d", exact, gotExact, unit)
		}
		step *= 1024
	}
}

func TestFormatSizeClampsNegative(t *testing.T) {
	if got := output.FormatSize(-1); got != "0 B" {
		t.Errorf("FormatSize(-1) = %q, want \"0 B\"", got)
	}
	if got := output.FormatSize(-999999); got != "0 B" {
		t.Errorf("FormatSize(-999999) = %q, want \"0 B\"", got)
	}
}

func TestFormatSizeBytesHaveNoDecimal(t *testing.T) {
	for _, b := range []int64{0, 1, 100, 1023} {
		got := output.FormatSize(b)
		if strings.Contains(got, ".") {
			t.Errorf("FormatSize(%d) = %q; byte values carry no decimal", b, got)
		}
	}
}

func TestFormatSizeAboveBytesHasOneDecimal(t *testing.T) {
	for _, b := range []int64{1024, 5000, 1048576, 1073741824} {
		got := output.FormatSize(b)
		number := strings.Fields(got)[0]
		dot := strings.Index(number, ".")
		if dot < 0 {
			t.Errorf("FormatSize(%d) = %q; want one decimal place", b, got)
			continue
		}
		if len(number)-dot-1 != 1 {
			t.Errorf("FormatSize(%d) = %q; want exactly one decimal place", b, got)
		}
	}
}

func TestFormatType(t *testing.T) {
	tests := []struct {
		name string
		row  engine.Row
		want string
	}{
		{"file with extension", engine.Row{Ext: "txt"}, "txt"},
		{"case preserved", engine.Row{Ext: "PDF"}, "PDF"},
		{"extensionless file", engine.Row{Ext: ""}, "—"},
		{"folder", engine.Row{IsDir: true}, "folder"},
		{"folder ignores extension", engine.Row{IsDir: true, Ext: "git"}, "folder"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := output.FormatType(tt.row); got != tt.want {
				t.Errorf("FormatType(%+v) = %q, want %q", tt.row, got, tt.want)
			}
		})
	}
}

func TestFormatTypeMatchesWhatTypeFieldQueries(t *testing.T) {
	field := engine.TypeField{}

	for _, name := range []string{"notes.txt", "report.PDF", "archive.tar.gz"} {
		ext := engine.ExtensionOf(name)
		displayed := output.FormatType(engine.Row{Ext: ext})
		queried, _ := field.NormalizeValue(displayed)

		if queried.Text != ext {
			t.Errorf("%q displays TYPE %q which queries back as %q, want %q; a value read off the table must be pasteable", name, displayed, queried.Text, ext)
		}
	}
}

func TestFormatRowSize(t *testing.T) {
	if got := output.FormatRowSize(engine.Row{IsDir: true, Size: 4096}); got != "—" {
		t.Errorf("folder size = %q, want \"—\"; summing a subtree would read the whole disk", got)
	}
	if got := output.FormatRowSize(engine.Row{Size: 1024}); got != "1.0 KB" {
		t.Errorf("file size = %q, want \"1.0 KB\"", got)
	}
	if got := output.FormatRowSize(engine.Row{Size: 0}); got != "0 B" {
		t.Errorf("empty file size = %q, want \"0 B\"", got)
	}
}

func TestFormatModified(t *testing.T) {
	stamp := time.Date(2026, 8, 20, 14, 2, 30, 0, time.UTC)

	if got := output.FormatModified(stamp); got != "2026-08-20 14:02" {
		t.Errorf("FormatModified = %q, want \"2026-08-20 14:02\"", got)
	}
	if got := output.FormatModified(time.Time{}); got != "—" {
		t.Errorf("zero time = %q, want \"—\"", got)
	}
}

func TestFormatModifiedPadsSingleDigits(t *testing.T) {
	stamp := time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC)

	got := output.FormatModified(stamp)
	if got != "2026-01-02 03:04" {
		t.Errorf("FormatModified = %q, want zero-padded \"2026-01-02 03:04\"", got)
	}
	if len(got) != len("2026-01-02 03:04") {
		t.Errorf("timestamp width varies: %q", got)
	}
}

func TestFormatCountAlwaysSaysFiles(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0 files"},
		{1, "1 files"},
		{2, "2 files"},
		{100, "100 files"},
	}

	for _, tt := range tests {
		if got := output.FormatCount(tt.n); got != tt.want {
			t.Errorf("FormatCount(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
