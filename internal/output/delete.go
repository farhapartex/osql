package output

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/farhapartex/osql/internal/engine"
)

const (
	ConfirmWord   = "yes"
	ConfirmPrompt = "delete> "

	victimWidth = 40
	detailWidth = 18
)

type DeleteRenderer struct{}

func NewDelete() DeleteRenderer {
	return DeleteRenderer{}
}

func (DeleteRenderer) Preview(w io.Writer, plan engine.DeletePlan) error {
	if _, err := fmt.Fprintf(w, "%s will be deleted:\n\n", headline(plan)); err != nil {
		return err
	}

	for _, victim := range plan.Victims {
		label := truncateMiddle(victim.Name, victimWidth)
		line := indent + padRight(label, victimWidth) + padLeft(detailFor(victim), detailWidth)
		if _, err := fmt.Fprintln(w, strings.TrimRight(line, " ")); err != nil {
			return err
		}
	}

	if len(plan.Victims) > 1 {
		line := indent + padRight("total", victimWidth) + padLeft(FormatSize(plan.TotalSize), detailWidth)
		if _, err := fmt.Fprintln(w, strings.TrimRight(line, " ")); err != nil {
			return err
		}
	}

	subject := "They"
	if len(plan.Victims) == 1 {
		subject = "It"
	}
	destination := "moved to the trash"
	if plan.Permanent {
		destination = "deleted for good, with no way back"
	}
	_, err := fmt.Fprintf(w, "\n%s will be %s.\nType %q to go ahead, anything else to cancel.\n", subject, destination, ConfirmWord)
	return err
}

func (DeleteRenderer) Cancelled(w io.Writer) error {
	_, err := fmt.Fprintln(w, "Cancelled. Nothing was deleted.")
	return err
}

func (DeleteRenderer) Nothing(w io.Writer) error {
	_, err := fmt.Fprintln(w, "Nothing matched, so there is nothing to delete.")
	return err
}

func (DeleteRenderer) Result(w io.Writer, result engine.DeleteResult) error {
	verb := "Moved"
	suffix := " to the trash"
	if result.Permanent {
		verb = "Deleted"
		suffix = ""
	}

	if len(result.Deleted) > 0 {
		if _, err := fmt.Fprintf(w, "%s %s%s.\n", verb, plural(len(result.Deleted), "item"), suffix); err != nil {
			return err
		}
	}

	if len(result.Failed) == 0 {
		return nil
	}

	if _, err := fmt.Fprintf(w, "Could not delete %s:\n", plural(len(result.Failed), "item")); err != nil {
		return err
	}
	for _, failure := range result.Failed {
		line := indent + padRight(truncateMiddle(failure.Name, victimWidth), victimWidth) + failure.Reason
		if _, err := fmt.Fprintln(w, strings.TrimRight(line, " ")); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, "Try running osql with sudo.")
	return err
}

func headline(plan engine.DeletePlan) string {
	if len(plan.Victims) == 1 && plan.Victims[0].IsDir {
		return fmt.Sprintf("'%s' and everything in it", plan.Victims[0].Name)
	}
	if len(plan.Victims) == 1 {
		return "This file"
	}
	return "These " + strconv.Itoa(len(plan.Victims)) + " items"
}

func detailFor(v engine.Victim) string {
	if !v.IsDir {
		return FormatSize(v.Size)
	}
	if v.Files == 0 && v.Folders <= 1 {
		return "empty folder"
	}
	return plural(int(v.Files), "file") + ", " + FormatSize(v.Size)
}
