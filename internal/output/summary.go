package output

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/farhapartex/osql/internal/engine"
)

const (
	ScopeOneLevel  = "one level"
	ScopeAllLevels = "all levels"
	DateLayout     = "2006-01-02"
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
		if _, err := fmt.Fprintf(w, "%s, and no files.\n", folderPhrase(s.Folders)); err != nil {
			return err
		}
		return writeSkipNotice(w, s)
	}

	if err := writeCounts(w, s); err != nil {
		return err
	}
	if err := writeTypes(w, s); err != nil {
		return err
	}
	if err := writeLargest(w, s); err != nil {
		return err
	}
	if err := writeModified(w, s); err != nil {
		return err
	}
	return writeSkipNotice(w, s)
}

func writeCounts(w io.Writer, s engine.Summary) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.AlignRight)

	fmt.Fprintf(tw, "WHAT\tCOUNT\tSIZE\t\n")
	fmt.Fprintf(tw, "files\t%d\t%s\t\n", s.Files, FormatSize(s.TotalSize))
	fmt.Fprintf(tw, "folders\t%d\t%s\t\n", s.Folders, Absent)
	fmt.Fprintf(tw, "total\t%d\t%s\t\n", s.Files+s.Folders, FormatSize(s.TotalSize))

	if err := tw.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w)
	return err
}

func writeTypes(w io.Writer, s engine.Summary) error {
	if len(s.Types) == 0 {
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintf(tw, "TYPE\tCOUNT\tSIZE\t\n")
	for _, t := range s.Types {
		name := t.Ext
		if name == "" {
			name = Absent
		}
		fmt.Fprintf(tw, "%s\t%d\t%s\t\n", name, t.Count, FormatSize(t.Size))
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	if s.MoreTypes > 0 {
		if _, err := fmt.Fprintf(w, "and %d more %s\n", s.MoreTypes, pluralWord(s.MoreTypes, "type")); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

func writeLargest(w io.Writer, s engine.Summary) error {
	if len(s.Largest) == 0 {
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "LARGEST\tSIZE\n")
	for _, row := range s.Largest {
		fmt.Fprintf(tw, "%s\t%s\n", row.Name, FormatSize(row.Size))
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w)
	return err
}

func writeModified(w io.Writer, s engine.Summary) error {
	if s.Oldest.IsZero() || s.Newest.IsZero() {
		return nil
	}
	_, err := fmt.Fprintf(w, "MODIFIED  %s to %s\n", s.Oldest.Format(DateLayout), s.Newest.Format(DateLayout))
	return err
}

func writeSkipNotice(w io.Writer, s engine.Summary) error {
	if !s.SkipsShown || len(s.Skipped) == 0 {
		return nil
	}

	if _, err := fmt.Fprintf(w, "\nSkipped %s: %s\n", plural(len(s.Skipped), "folder"), strings.Join(s.Skipped, ", ")); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, "Add \"with skipped\" to include them — it will take longer.")
	return err
}

func scopeOf(s engine.Summary) string {
	if s.Recursive {
		return ScopeAllLevels
	}
	return ScopeOneLevel
}

func folderPhrase(n int64) string {
	return "Contains " + plural(int(n), "folder")
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
