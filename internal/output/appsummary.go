package output

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/farhapartex/osql/internal/engine"
)

const (
	AppsHeading   = "Installed apps"
	sourceWidth   = 13
	appLabelWidth = 13
)

type AppSummaryRenderer struct{}

func NewAppSummary() AppSummaryRenderer {
	return AppSummaryRenderer{}
}

func (AppSummaryRenderer) Render(w io.Writer, s engine.AppSummary) error {
	if s.IsEmpty() {
		_, err := fmt.Fprintln(w, "I didn't find any installed apps.")
		return err
	}

	if _, err := fmt.Fprintf(w, "%s\n\n", AppsHeading); err != nil {
		return err
	}

	for _, section := range []func(io.Writer, engine.AppSummary) error{
		writeAppCounts, writeAppSources, writeLargestApps, writeAppModified,
	} {
		if err := section(w, s); err != nil {
			return err
		}
	}
	return writeUnmeasuredNotice(w, s)
}

func writeAppCounts(w io.Writer, s engine.AppSummary) error {
	rows := [][3]string{{"WHAT", "COUNT", "SIZE"}}
	rows = append(rows, [3]string{"apps", strconv.FormatInt(s.Apps, 10), FormatSize(s.AppsSize)})
	if s.Tools > 0 {
		rows = append(rows, [3]string{"tools", strconv.FormatInt(s.Tools, 10), FormatSize(s.ToolsSize)})
		rows = append(rows, [3]string{"total", strconv.FormatInt(s.Total(), 10), FormatSize(s.TotalSize())})
	}
	return writeAppTallyBlock(w, rows, appLabelWidth)
}

func writeAppSources(w io.Writer, s engine.AppSummary) error {
	if len(s.Sources) == 0 {
		return nil
	}

	rows := [][3]string{{"SOURCE", "COUNT", "SIZE"}}
	for _, tally := range s.Sources {
		rows = append(rows, [3]string{
			tally.Source,
			strconv.FormatInt(tally.Count, 10),
			FormatSize(tally.Size),
		})
	}
	return writeAppTallyBlock(w, rows, sourceWidth)
}

func writeAppTallyBlock(w io.Writer, rows [][3]string, width int) error {
	for _, row := range rows {
		line := indent + padRight(row[0], width) + padLeft(row[1], countWidth) + padLeft(row[2], sizeWidth)
		if _, err := fmt.Fprintln(w, strings.TrimRight(line, " ")); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

func writeLargestApps(w io.Writer, s engine.AppSummary) error {
	if len(s.Largest) == 0 {
		return nil
	}

	header := indent + padRight("LARGEST", nameWidth) + padLeft("SIZE", sizeWidth)
	if _, err := fmt.Fprintln(w, strings.TrimRight(header, " ")); err != nil {
		return err
	}

	for _, app := range s.Largest {
		line := indent + padRight(truncateMiddle(app.Name, nameWidth), nameWidth) + padLeft(FormatSize(app.Size), sizeWidth)
		if _, err := fmt.Fprintln(w, strings.TrimRight(line, " ")); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w)
	return err
}

func writeAppModified(w io.Writer, s engine.AppSummary) error {
	if s.Oldest.IsZero() || s.Newest.IsZero() {
		return nil
	}
	_, err := fmt.Fprintf(w, "%s%s%s to %s\n", indent, padRight("MODIFIED", appLabelWidth+1),
		s.Oldest.Format(DateLayout), s.Newest.Format(DateLayout))
	return err
}

func writeUnmeasuredNotice(w io.Writer, s engine.AppSummary) error {
	if s.Unmeasured == 0 {
		return nil
	}
	_, err := fmt.Fprintf(w, "\n%s%s could not be measured, so they are not in the total.\n",
		indent, plural(s.Unmeasured, "app"))
	return err
}
