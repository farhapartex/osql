package output

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/farhapartex/osql/internal/engine"
)

const (
	ScopeOneLevel  = "one level"
	ScopeAllLevels = "all levels"
	DateLayout     = "2006-01-02"

	indent     = "  "
	labelWidth = 9
	countWidth = 7
	sizeWidth  = 11
	nameWidth  = 46
)

type SummaryRenderer struct{}

func NewSummary() SummaryRenderer {
	return SummaryRenderer{}
}

func (SummaryRenderer) Render(w io.Writer, s engine.Summary) error {
	if s.IsEmpty() {
		_, err := fmt.Fprintf(w, "'%s' is empty.\n", s.Path)
		return err
	}

	if _, err := fmt.Fprintf(w, "%s — %s\n\n", s.Path, scopeOf(s)); err != nil {
		return err
	}

	if !s.HasFiles() {
		if _, err := fmt.Fprintf(w, "%sContains %s, and no files.\n", indent, plural(int(s.Folders), "folder")); err != nil {
			return err
		}
		return writeSkipNotice(w, s)
	}

	for _, section := range []func(io.Writer, engine.Summary) error{
		writeCounts, writeTypes, writeLargest, writeModified,
	} {
		if err := section(w, s); err != nil {
			return err
		}
	}
	return writeSkipNotice(w, s)
}

func writeCounts(w io.Writer, s engine.Summary) error {
	rows := [][3]string{
		{"WHAT", "COUNT", "SIZE"},
		{"files", strconv.FormatInt(s.Files, 10), FormatSize(s.TotalSize)},
		{"folders", strconv.FormatInt(s.Folders, 10), Absent},
		{"total", strconv.FormatInt(s.Files+s.Folders, 10), FormatSize(s.TotalSize)},
	}
	return writeTallyBlock(w, rows)
}

func writeTypes(w io.Writer, s engine.Summary) error {
	if len(s.Types) == 0 {
		return nil
	}

	rows := [][3]string{{"TYPE", "COUNT", "SIZE"}}
	for _, t := range s.Types {
		name := t.Ext
		if name == "" {
			name = Absent
		}
		rows = append(rows, [3]string{name, strconv.FormatInt(t.Count, 10), FormatSize(t.Size)})
	}

	if err := writeTallyRows(w, rows); err != nil {
		return err
	}
	if s.MoreTypes > 0 {
		if _, err := fmt.Fprintf(w, "%sand %d more %s\n", indent, s.MoreTypes, pluralWord(s.MoreTypes, "type")); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

func writeTallyBlock(w io.Writer, rows [][3]string) error {
	if err := writeTallyRows(w, rows); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w)
	return err
}

func writeTallyRows(w io.Writer, rows [][3]string) error {
	for _, row := range rows {
		line := indent + padRight(row[0], labelWidth) + padLeft(row[1], countWidth) + padLeft(row[2], sizeWidth)
		if _, err := fmt.Fprintln(w, strings.TrimRight(line, " ")); err != nil {
			return err
		}
	}
	return nil
}

func writeLargest(w io.Writer, s engine.Summary) error {
	if len(s.Largest) == 0 {
		return nil
	}

	header := indent + padRight("LARGEST", nameWidth) + padLeft("SIZE", sizeWidth)
	if _, err := fmt.Fprintln(w, strings.TrimRight(header, " ")); err != nil {
		return err
	}

	for _, row := range s.Largest {
		line := indent + padRight(truncateMiddle(row.Name, nameWidth), nameWidth) + padLeft(FormatSize(row.Size), sizeWidth)
		if _, err := fmt.Fprintln(w, strings.TrimRight(line, " ")); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w)
	return err
}

func writeModified(w io.Writer, s engine.Summary) error {
	if s.Oldest.IsZero() || s.Newest.IsZero() {
		return nil
	}
	_, err := fmt.Fprintf(w, "%s%s%s to %s\n", indent, padRight("MODIFIED", labelWidth+1),
		s.Oldest.Format(DateLayout), s.Newest.Format(DateLayout))
	return err
}

func writeSkipNotice(w io.Writer, s engine.Summary) error {
	if !s.SkipsShown || len(s.Skipped) == 0 {
		return nil
	}

	if _, err := fmt.Fprintf(w, "\n%sSkipped %s: %s\n", indent, plural(len(s.Skipped), "folder"), strings.Join(s.Skipped, ", ")); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "%sAdd \"with skipped\" to include them — it will take longer.\n", indent)
	return err
}

func scopeOf(s engine.Summary) string {
	if s.Recursive {
		return ScopeAllLevels
	}
	return ScopeOneLevel
}

func plural(n int, word string) string {
	return strconv.Itoa(n) + " " + pluralWord(n, word)
}

func pluralWord(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
