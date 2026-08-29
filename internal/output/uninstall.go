package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/farhapartex/osql/internal/uninstall"
)

const (
	UninstallPrompt = "uninstall> "

	uninstallPathWidth = 44
	uninstallSizeWidth = 12
)

type UninstallRenderer struct{}

func NewUninstall() UninstallRenderer {
	return UninstallRenderer{}
}

func (UninstallRenderer) Preview(w io.Writer, plan uninstall.Plan) error {
	if _, err := fmt.Fprint(w, "This will remove:\n\n"); err != nil {
		return err
	}

	if err := writeUninstallLine(w, plan.Binary.Path, "the osql binary", plan.Binary.Size); err != nil {
		return err
	}

	if plan.IncludesData {
		if err := writeUninstallLine(w, plan.Data.Path, "your history and system notes", plan.Data.Size); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "\nTotal: %s\n", FormatSize(plan.TotalSize())); err != nil {
		return err
	}

	if !plan.IncludesData {
		if _, err := fmt.Fprintln(w, "\nYour history and system notes will be left alone."); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintf(w, "\nThis cannot be undone. Type %q to go ahead, anything else to cancel.\n", ConfirmWord)
	return err
}

func (UninstallRenderer) Cancelled(w io.Writer) error {
	_, err := fmt.Fprintln(w, "Cancelled. Nothing was removed.")
	return err
}

func (UninstallRenderer) Result(w io.Writer, plan uninstall.Plan) error {
	if _, err := fmt.Fprintln(w, "osql has been removed. Thanks for trying it."); err != nil {
		return err
	}
	if plan.IncludesData || plan.Data.Path == "" {
		return nil
	}
	_, err := fmt.Fprintf(w, "Your notes are still at %s. Remove them with: rm -rf '%s'\n", plan.Data.Path, plan.Data.Path)
	return err
}

func writeUninstallLine(w io.Writer, path, description string, size int64) error {
	line := indent + padRight(truncateMiddle(path, uninstallPathWidth), uninstallPathWidth) +
		padLeft(FormatSize(size), uninstallSizeWidth) + "  " + description
	_, err := fmt.Fprintln(w, strings.TrimRight(line, " "))
	return err
}
