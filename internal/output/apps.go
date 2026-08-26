package output

import (
	"fmt"
	"io"
	"strconv"

	"github.com/farhapartex/osql/internal/engine"
)

const (
	HeaderVersion = "VERSION"
	HeaderSource  = "SOURCE"
)

const (
	appNameLimit    = 34
	appVersionLimit = 18
	appColumnGap    = 2
)

type AppsRenderer struct{}

func NewApps() AppsRenderer {
	return AppsRenderer{}
}

func (AppsRenderer) Render(w io.Writer, report engine.AppReport) error {
	nameWidth := DisplayWidth(HeaderName)
	versionWidth := DisplayWidth(HeaderVersion)
	sourceWidth := DisplayWidth(HeaderSource)
	sizeWidth := DisplayWidth(HeaderSize)

	names := make([]string, len(report.Apps))
	versions := make([]string, len(report.Apps))
	sizes := make([]string, len(report.Apps))

	for i, app := range report.Apps {
		names[i] = truncateMiddle(app.Name, appNameLimit)
		versions[i] = truncateMiddle(versionOf(app), appVersionLimit)
		sizes[i] = sizeOf(app)

		nameWidth = max(nameWidth, DisplayWidth(names[i]))
		versionWidth = max(versionWidth, DisplayWidth(versions[i]))
		sourceWidth = max(sourceWidth, DisplayWidth(app.Source))
		sizeWidth = max(sizeWidth, DisplayWidth(sizes[i]))
	}

	gap := appGap()
	header := padRight(HeaderName, nameWidth) + gap +
		padRight(HeaderVersion, versionWidth) + gap +
		padRight(HeaderSource, sourceWidth) + gap
	if report.Sized {
		header += padRight(HeaderSize, sizeWidth) + gap
	}
	header += HeaderModified

	if _, err := fmt.Fprintln(w, header); err != nil {
		return err
	}

	for i, app := range report.Apps {
		line := padRight(names[i], nameWidth) + gap +
			padRight(versions[i], versionWidth) + gap +
			padRight(app.Source, sourceWidth) + gap
		if report.Sized {
			line += padRight(sizes[i], sizeWidth) + gap
		}
		line += FormatModified(app.Modified)

		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	footer := FormatAppCount(len(report.Apps))
	if report.Sized {
		footer += ", " + FormatSize(report.TotalSize) + " on disk"
	}
	if _, err := fmt.Fprintf(w, "\n%s\n", footer); err != nil {
		return err
	}

	if report.Tools > 0 {
		if _, err := fmt.Fprintln(w, HiddenToolsNote(report.Tools)); err != nil {
			return err
		}
	}
	return nil
}

func sizeOf(app engine.App) string {
	if !app.SizeKnown {
		return Absent
	}
	return FormatSize(app.Size)
}

func HiddenToolsNote(tools int) string {
	return fmt.Sprintf("Not showing %s. Add \"where source = '%s'\" to see them.",
		plural(tools, "command-line tool"), engine.SourceHomebrewCLI)
}

func versionOf(app engine.App) string {
	if app.Version == "" {
		return Absent
	}
	return app.Version
}

func FormatAppCount(n int) string {
	if n == 1 {
		return "1 app"
	}
	return strconv.Itoa(n) + " apps"
}

func appGap() string {
	out := make([]byte, appColumnGap)
	for i := range out {
		out[i] = ' '
	}
	return string(out)
}
