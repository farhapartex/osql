package test

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/farhapartex/osql/internal/shell"
	"github.com/farhapartex/osql/internal/state"
)

func shellWithStore(t *testing.T, out *bytes.Buffer) (*shell.Shell, state.History) {
	t.Helper()

	store := state.New(state.Options{Root: t.TempDir()})
	if err := store.Ensure(); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	t.Cleanup(func() { store.Close() })

	hist, err := store.History()
	if err != nil {
		t.Fatalf("History() error = %v", err)
	}

	app := shell.New(shell.Config{Store: store, Out: out, Err: out})
	return app, hist
}

func TestBuiltinRegistryLookup(t *testing.T) {
	r := shell.NewBuiltinRegistry(shell.Builtin{Name: "ping", Summary: "s"})

	if _, ok := r.Lookup("ping"); !ok {
		t.Error("Lookup(\"ping\") = false, want true")
	}
	if _, ok := r.Lookup("pong"); ok {
		t.Error("Lookup(\"pong\") = true, want false")
	}
	if _, ok := r.Lookup(""); ok {
		t.Error("Lookup(\"\") = true, want false")
	}
}

func TestBuiltinRegistryZeroValueIsUsable(t *testing.T) {
	var r shell.BuiltinRegistry

	if _, ok := r.Lookup("help"); ok {
		t.Error("zero-value registry returned a builtin")
	}
	r.Register(shell.Builtin{Name: "help"})
	if _, ok := r.Lookup("help"); !ok {
		t.Error("Register on a zero-value registry did not take effect")
	}
}

func TestBuiltinRegistryNamesAreSorted(t *testing.T) {
	r := shell.NewBuiltinRegistry(
		shell.Builtin{Name: "quit"},
		shell.Builtin{Name: "clear"},
		shell.Builtin{Name: "help"},
	)

	got := r.Names()
	want := []string{"clear", "help", "quit"}
	if !slices.Equal(got, want) {
		t.Errorf("Names() = %v, want %v", got, want)
	}
}

func TestBuiltinRegistryOverwritesSameName(t *testing.T) {
	r := shell.NewBuiltinRegistry(
		shell.Builtin{Name: "help", Summary: "first"},
		shell.Builtin{Name: "help", Summary: "second"},
	)

	b, _ := r.Lookup("help")
	if b.Summary != "second" {
		t.Errorf("Summary = %q, want \"second\"", b.Summary)
	}
	if len(r.Names()) != 1 {
		t.Errorf("Names() = %v, want one entry", r.Names())
	}
}

func TestBuiltinRegistryAllMatchesNames(t *testing.T) {
	r := shell.DefaultBuiltins()

	all := r.All()
	names := r.Names()
	if len(all) != len(names) {
		t.Fatalf("All() has %d entries, Names() has %d", len(all), len(names))
	}
	for i := range all {
		if all[i].Name != names[i] {
			t.Errorf("All()[%d].Name = %q, want %q", i, all[i].Name, names[i])
		}
		if all[i].Summary == "" {
			t.Errorf("builtin %q has no summary; help would show a blank line", all[i].Name)
		}
		if all[i].Run == nil {
			t.Errorf("builtin %q has no Run function", all[i].Name)
		}
	}
}

func TestDefaultBuiltinsRegistersExpectedCommands(t *testing.T) {
	r := shell.DefaultBuiltins()

	for _, name := range []string{"help", "exit", "quit", "clear", "history"} {
		if _, ok := r.Lookup(name); !ok {
			t.Errorf("default builtins missing %q", name)
		}
	}
}

func TestHelpIsGeneratedFromTheRegistry(t *testing.T) {
	out := &bytes.Buffer{}
	app, _ := shellWithStore(t, out)

	if err := app.Dispatch("help"); err != nil {
		t.Fatalf("help error = %v", err)
	}

	got := out.String()
	for _, b := range app.Builtins().All() {
		if !strings.Contains(got, b.Name) {
			t.Errorf("help output omits builtin %q", b.Name)
		}
		if !strings.Contains(got, b.Summary) {
			t.Errorf("help output omits summary for %q", b.Name)
		}
	}
	if !strings.Contains(got, "select") {
		t.Error("help output does not mention the select statement")
	}
}

func TestExitAndQuitSignalShutdown(t *testing.T) {
	for _, name := range []string{"exit", "quit"} {
		t.Run(name, func(t *testing.T) {
			out := &bytes.Buffer{}
			app, _ := shellWithStore(t, out)

			err := app.Dispatch(name)
			if !errors.Is(err, shell.ErrExit) {
				t.Errorf("Dispatch(%q) = %v, want ErrExit", name, err)
			}
		})
	}
}

func TestClearWritesAnsiEscape(t *testing.T) {
	out := &bytes.Buffer{}
	app, _ := shellWithStore(t, out)

	if err := app.Dispatch("clear"); err != nil {
		t.Fatalf("clear error = %v", err)
	}
	if got := out.String(); got != "\033[H\033[2J" {
		t.Errorf("clear wrote %q, want the ANSI home-and-erase sequence", got)
	}
}

func TestHistoryBuiltinOnEmptyHistory(t *testing.T) {
	out := &bytes.Buffer{}
	app, _ := shellWithStore(t, out)

	if err := app.Dispatch("history"); err != nil {
		t.Fatalf("history error = %v", err)
	}
	if !strings.Contains(out.String(), "No history yet.") {
		t.Errorf("expected an empty-history notice, got %q", out.String())
	}
}

func TestHistoryBuiltinNumbersLines(t *testing.T) {
	out := &bytes.Buffer{}
	app, hist := shellWithStore(t, out)

	for _, line := range []string{"select files from '.'", "select all from '~'"} {
		if err := hist.Append(line); err != nil {
			t.Fatal(err)
		}
	}

	if err := app.Dispatch("history"); err != nil {
		t.Fatalf("history error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "1  select files from '.'") {
		t.Errorf("first entry not numbered 1:\n%s", got)
	}
	if !strings.Contains(got, "2  select all from '~'") {
		t.Errorf("second entry not numbered 2:\n%s", got)
	}
}

func TestHistoryBuiltinNumbersAreAbsoluteWhenTruncated(t *testing.T) {
	out := &bytes.Buffer{}
	app, hist := shellWithStore(t, out)

	for i := range 60 {
		if err := hist.Append(lineNumbered(i)); err != nil {
			t.Fatal(err)
		}
	}

	if err := app.Dispatch("history"); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 50 {
		t.Fatalf("history printed %d lines, want the last 50", len(lines))
	}
	if !strings.Contains(lines[0], "11  ") {
		t.Errorf("first shown line should be numbered 11, got %q", lines[0])
	}
	if !strings.Contains(lines[49], "60  ") {
		t.Errorf("last shown line should be numbered 60, got %q", lines[49])
	}
}

func TestHistoryClearEmptiesTheFile(t *testing.T) {
	out := &bytes.Buffer{}
	app, hist := shellWithStore(t, out)

	hist.Append("select files from '.'")

	if err := app.Dispatch("history clear"); err != nil {
		t.Fatalf("history clear error = %v", err)
	}

	lines, err := hist.Lines(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Errorf("history still holds %d lines after clear", len(lines))
	}
}

func TestHistoryRejectsUnknownArgument(t *testing.T) {
	out := &bytes.Buffer{}
	app, _ := shellWithStore(t, out)

	err := app.Dispatch("history purge")
	if err == nil {
		t.Fatal("history accepted an unknown argument")
	}
	if !strings.Contains(err.Error(), "purge") {
		t.Errorf("error does not quote the argument: %v", err)
	}
}

func TestHistoryWithoutStoreFails(t *testing.T) {
	app := shell.New(shell.Config{Out: &bytes.Buffer{}})

	if err := app.Dispatch("history"); err == nil {
		t.Error("history without a store succeeded; want an error")
	}
}

func lineNumbered(i int) string {
	return fmt.Sprintf("select files from 'dir-%d'", i)
}
