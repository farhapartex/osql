package test

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/reader"
	"github.com/farhapartex/osql/internal/shell"
)

func TestDispatchRoutesQueryVerb(t *testing.T) {
	out := &bytes.Buffer{}
	app := shell.New(shell.Config{Out: out, Err: out})

	if err := app.Dispatch("select files from 'Documents'"); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if !strings.Contains(out.String(), "select files from 'Documents'") {
		t.Errorf("query line not handled: %q", out.String())
	}
}

func TestDispatchIsCaseInsensitiveOnKeywords(t *testing.T) {
	tests := []string{"HELP", "Help", "hELp", "SELECT files from '.'", "EXIT"}

	for _, line := range tests {
		t.Run(line, func(t *testing.T) {
			out := &bytes.Buffer{}
			app, _ := shellWithStore(t, out)

			err := app.Dispatch(line)
			if err != nil && !strings.Contains(err.Error(), "exit") {
				t.Errorf("Dispatch(%q) error = %v; keywords are case-insensitive", line, err)
			}
		})
	}
}

func TestDispatchStripsTrailingSemicolon(t *testing.T) {
	out := &bytes.Buffer{}
	app, _ := shellWithStore(t, out)

	if err := app.Dispatch("help;"); err != nil {
		t.Fatalf("Dispatch(\"help;\") error = %v", err)
	}
	if !strings.Contains(out.String(), "clear") {
		t.Errorf("trailing semicolon broke builtin dispatch: %q", out.String())
	}
}

func TestDispatchOnBlankInput(t *testing.T) {
	out := &bytes.Buffer{}
	app := shell.New(shell.Config{Out: out, Err: out})

	for _, line := range []string{"", "   ", "\t", ";", "  ;  "} {
		if err := app.Dispatch(line); err != nil {
			t.Errorf("Dispatch(%q) error = %v, want nil", line, err)
		}
	}
	if out.Len() != 0 {
		t.Errorf("blank input produced output: %q", out.String())
	}
}

func TestDispatchUnknownVerbSuggests(t *testing.T) {
	out := &bytes.Buffer{}
	app := shell.New(shell.Config{Out: out, Err: out})

	err := app.Dispatch("slect files from '.'")
	if err == nil {
		t.Fatal("Dispatch accepted an unknown verb")
	}
	if !oerr.Is(err, oerr.KindUnknownVerb) {
		t.Errorf("error kind = %v, want unknown_verb", err)
	}
	if !strings.Contains(err.Error(), `Did you mean "select"?`) {
		t.Errorf("no suggestion offered: %v", err)
	}
}

func TestDispatchUnknownVerbStaysSilentWhenFarOff(t *testing.T) {
	app := shell.New(shell.Config{Out: &bytes.Buffer{}})

	err := app.Dispatch("frobnicate everything")
	if err == nil {
		t.Fatal("Dispatch accepted an unknown verb")
	}
	if strings.Contains(err.Error(), "Did you mean") {
		t.Errorf("guessed at a far-off token: %v", err)
	}
}

func TestKnownVerbsIncludesQueryAndBuiltins(t *testing.T) {
	app := shell.New(shell.Config{})

	verbs := app.KnownVerbs()
	if !slices.Contains(verbs, "select") {
		t.Error("KnownVerbs() omits select, so typos would never suggest it")
	}
	for _, name := range app.Builtins().Names() {
		if !slices.Contains(verbs, name) {
			t.Errorf("KnownVerbs() omits builtin %q", name)
		}
	}
}

func TestRunStopsOnExitBuiltin(t *testing.T) {
	out := &bytes.Buffer{}
	app := shell.New(shell.Config{
		Reader:  reader.NewBasic(strings.NewReader("exit\nselect files from '.'\n"), out, nil),
		Out:     out,
		Err:     out,
		Version: "v1",
		Commit:  "c1",
	})

	if err := app.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if strings.Contains(out.String(), "select files from '.'") {
		t.Error("Run kept reading after exit")
	}
}

func TestRunWritesErrorsToErrNotOut(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := shell.New(shell.Config{
		Reader: reader.NewBasic(strings.NewReader("slect files\n"), stdout, nil),
		Out:    stdout,
		Err:    stderr,
	})

	if err := app.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(stderr.String(), "Did you mean") {
		t.Errorf("error not written to Err: %q", stderr.String())
	}
	if strings.Contains(stdout.String(), "Did you mean") {
		t.Errorf("error leaked into Out: %q", stdout.String())
	}
}

func TestRunSurvivesBadCommandAndKeepsPrompting(t *testing.T) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	app := shell.New(shell.Config{
		Reader: reader.NewBasic(strings.NewReader("slect x\nselect files from '.'\nexit\n"), out, nil),
		Out:    out,
		Err:    errBuf,
	})

	if err := app.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "select files from '.'") {
		t.Error("shell did not continue after a bad command")
	}
	if got := strings.Count(out.String(), shell.Prompt); got != 3 {
		t.Errorf("prompt written %d times, want 3", got)
	}
}

func TestGreetingAdvertisesHelpAndExit(t *testing.T) {
	app := shell.New(shell.Config{Version: "v0.1.0", Commit: "abc1234"})

	got := app.Greeting()
	for _, want := range []string{"help", "exit"} {
		if !strings.Contains(got, want) {
			t.Errorf("greeting %q does not mention %q", got, want)
		}
	}
	if _, ok := app.Builtins().Lookup("help"); !ok {
		t.Error("greeting advertises help but it is not registered")
	}
	if _, ok := app.Builtins().Lookup("exit"); !ok {
		t.Error("greeting advertises exit but it is not registered")
	}
}
