package test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/farhapartex/osql/internal/reader"
	"github.com/farhapartex/osql/internal/shell"
)

func runShell(t *testing.T, input string, hist reader.HistoryAppender) string {
	t.Helper()

	out := &bytes.Buffer{}
	app := shell.New(shell.Config{
		Reader:  reader.NewBasic(strings.NewReader(input), out, hist),
		Out:     out,
		Version: "v0.1.0",
		Commit:  "abc1234",
	})

	if err := app.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	return out.String()
}

func TestShellGreetingCarriesVersion(t *testing.T) {
	app := shell.New(shell.Config{Version: "v0.1.0", Commit: "abc1234"})

	want := "osql v0.1.0 (abc1234) — Ctrl+D to exit."
	if got := app.Greeting(); got != want {
		t.Errorf("Greeting() = %q, want %q", got, want)
	}
}

func TestShellGreetingFallsBackForUnstampedBuild(t *testing.T) {
	app := shell.New(shell.Config{})

	want := "osql dev (none) — Ctrl+D to exit."
	if got := app.Greeting(); got != want {
		t.Errorf("Greeting() = %q, want %q", got, want)
	}
}

func TestShellPromptIsOsqlAngle(t *testing.T) {
	if shell.Prompt != "osql > " {
		t.Errorf("Prompt = %q, want %q", shell.Prompt, "osql > ")
	}
}

func TestShellRunGreetsThenPromptsThenExitsOnEOF(t *testing.T) {
	got := runShell(t, "", nil)

	if !strings.HasPrefix(got, "osql v0.1.0 (abc1234) — Ctrl+D to exit.\n") {
		t.Errorf("output did not start with the greeting:\n%q", got)
	}
	if !strings.Contains(got, shell.Prompt) {
		t.Errorf("output has no prompt:\n%q", got)
	}
}

func TestShellRunEchoesEachLine(t *testing.T) {
	got := runShell(t, "select files from '.'\nselect all from '~'\n", nil)

	for _, want := range []string{"select files from '.'", "select all from '~'"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing echoed line %q:\n%s", want, got)
		}
	}
}

func TestShellRunSkipsBlankLines(t *testing.T) {
	hist := &errAppender{}
	out := runShell(t, "\n\n   \n\t\nselect files from '.'\n", hist)

	if len(hist.lines) != 1 {
		t.Errorf("history recorded %d lines, want 1; blank and whitespace-only lines are not commands", len(hist.lines))
	}
	if len(hist.lines) == 1 && hist.lines[0] != "select files from '.'" {
		t.Errorf("history recorded %q", hist.lines[0])
	}

	promptCount := strings.Count(out, shell.Prompt)
	if promptCount != 6 {
		t.Errorf("prompt written %d times, want 6 (one per line plus the EOF read)", promptCount)
	}
}

func TestShellRunTrimsSurroundingWhitespaceBeforeRecording(t *testing.T) {
	hist := &errAppender{}
	runShell(t, "   select files from '.'   \n", hist)

	if len(hist.lines) != 1 {
		t.Fatalf("history recorded %d lines, want 1", len(hist.lines))
	}
	if hist.lines[0] != "select files from '.'" {
		t.Errorf("history recorded %q, want the trimmed line", hist.lines[0])
	}
}

func TestShellRunRecordsInvalidLinesToo(t *testing.T) {
	hist := &errAppender{}
	runShell(t, "not a query at all\nslect files\n", hist)

	if len(hist.lines) != 2 {
		t.Fatalf("history recorded %d lines, want 2; a typo is exactly what you want to recall", len(hist.lines))
	}
}

func TestShellRunSurvivesHistoryFailure(t *testing.T) {
	hist := &errAppender{err: errors.New("disk full")}

	got := runShell(t, "select files from '.'\n", hist)

	if !strings.Contains(got, "select files from '.'") {
		t.Error("a history write failure must not stop the shell from working")
	}
}

func TestShellRunWithoutReaderFails(t *testing.T) {
	app := shell.New(shell.Config{Out: &bytes.Buffer{}})

	err := app.Run()
	if !errors.Is(err, shell.ErrNoReader) {
		t.Errorf("Run() with no reader error = %v, want ErrNoReader", err)
	}
}

func TestShellRunPropagatesReaderError(t *testing.T) {
	out := &bytes.Buffer{}
	app := shell.New(shell.Config{
		Reader: &failingReader{err: errors.New("terminal exploded")},
		Out:    out,
	})

	if err := app.Run(); err == nil {
		t.Error("Run() swallowed a non-EOF reader error")
	}
}

func TestShellNewDefaultsNilWriters(t *testing.T) {
	app := shell.New(shell.Config{})

	if app == nil {
		t.Fatal("New() returned nil")
	}
	if got := app.Greeting(); got == "" {
		t.Error("Greeting() empty after New with a zero Config")
	}
}

func TestShellAcceptsFullyInjectedConfig(t *testing.T) {
	out := &bytes.Buffer{}
	store := &fakeStore{}

	app := shell.New(shell.Config{
		Reader:   reader.NewBasic(strings.NewReader("select files from '.'\n"), out, nil),
		Lexer:    &fakeLexer{},
		Parser:   &fakeParser{},
		Renderer: &fakeRenderer{},
		Store:    store,
		Out:      out,
		Err:      &bytes.Buffer{},
		Version:  "v1",
		Commit:   "c1",
	})

	if err := app.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if out.Len() == 0 {
		t.Error("Run() wrote nothing to the injected Out")
	}
}
