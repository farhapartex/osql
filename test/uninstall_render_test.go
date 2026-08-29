package test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/farhapartex/osql/internal/output"
	"github.com/farhapartex/osql/internal/uninstall"
)

func renderUninstallPreview(t *testing.T, plan uninstall.Plan) string {
	t.Helper()

	buf := &bytes.Buffer{}
	if err := output.NewUninstall().Preview(buf, plan); err != nil {
		t.Fatalf("Preview() = %v", err)
	}
	return buf.String()
}

func fullPlan() uninstall.Plan {
	return uninstall.Plan{
		Binary:       uninstall.Target{Path: binaryPath, Size: 5_242_880},
		Data:         uninstall.Target{Path: statePath, Size: 12_288},
		IncludesData: true,
	}
}

func binaryOnlyPlan() uninstall.Plan {
	return uninstall.Plan{
		Binary: uninstall.Target{Path: binaryPath, Size: 5_242_880},
		Data:   uninstall.Target{Path: statePath},
	}
}

func TestUninstallPreviewNamesEverythingItWillRemove(t *testing.T) {
	got := renderUninstallPreview(t, fullPlan())

	for _, want := range []string{
		"This will remove:",
		binaryPath,
		"the osql binary",
		statePath,
		"your history and system notes",
		"Total:",
		`Type "yes" to go ahead`,
		"cannot be undone",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("preview is missing %q, got:\n%s", want, got)
		}
	}
}

func TestUninstallPreviewShowsSizes(t *testing.T) {
	got := renderUninstallPreview(t, fullPlan())

	for _, want := range []string{"5.0 MB", "12.0 KB"} {
		if !strings.Contains(got, want) {
			t.Errorf("preview is missing the size %q, got:\n%s", want, got)
		}
	}
}

func TestUninstallPreviewSaysWhenDataStays(t *testing.T) {
	got := renderUninstallPreview(t, binaryOnlyPlan())

	if strings.Contains(got, "your history and system notes") {
		t.Errorf("preview lists data it will not remove, got:\n%s", got)
	}
	if !strings.Contains(got, "left alone") {
		t.Errorf("preview must say the notes stay, got:\n%s", got)
	}
}

func TestUninstallPreviewTotalsOnlyWhatItRemoves(t *testing.T) {
	if got := renderUninstallPreview(t, binaryOnlyPlan()); !strings.Contains(got, "Total: 5.0 MB") {
		t.Errorf("total must exclude kept data, got:\n%s", got)
	}
}

func TestUninstallCancelledSaysNothingHappened(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := output.NewUninstall().Cancelled(buf); err != nil {
		t.Fatalf("Cancelled() = %v", err)
	}

	if got, want := buf.String(), "Cancelled. Nothing was removed.\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestUninstallResultConfirmsTheRemoval(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := output.NewUninstall().Result(buf, fullPlan()); err != nil {
		t.Fatalf("Result() = %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "osql has been removed") {
		t.Errorf("got %q", got)
	}
	if strings.Contains(got, "rm -rf") {
		t.Errorf("nothing is left over, so there is nothing to tell them to remove, got %q", got)
	}
}

func TestUninstallResultPointsAtDataItLeftBehind(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := output.NewUninstall().Result(buf, binaryOnlyPlan()); err != nil {
		t.Fatalf("Result() = %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, statePath) {
		t.Errorf("result must say where the notes are, got %q", got)
	}
	if !strings.Contains(got, "rm -rf '"+statePath+"'") {
		t.Errorf("result must give the exact line to remove them, got %q", got)
	}
}

func TestUninstallResultSaysNothingAboutDataThatNeverExisted(t *testing.T) {
	buf := &bytes.Buffer{}
	plan := uninstall.Plan{Binary: uninstall.Target{Path: binaryPath, Size: 100}}

	if err := output.NewUninstall().Result(buf, plan); err != nil {
		t.Fatalf("Result() = %v", err)
	}

	if got := buf.String(); strings.Contains(got, "rm -rf") {
		t.Errorf("there was no state folder to mention, got %q", got)
	}
}

func TestUninstallPromptIsItsOwn(t *testing.T) {
	if output.UninstallPrompt == output.ConfirmPrompt {
		t.Error("the uninstall prompt must not say \"delete>\"")
	}
	if output.UninstallPrompt != "uninstall> " {
		t.Errorf("UninstallPrompt = %q", output.UninstallPrompt)
	}
}
