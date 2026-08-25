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
	app := shell.New(withPipeline(t, shell.Config{Out: out, Err: out}, pipelineFS()))

	if err := app.Dispatch("files from 'work'"); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if !strings.Contains(out.String(), "notes.txt") {
		t.Errorf("query produced no results: %q", out.String())
	}
}

func TestDispatchIsCaseInsensitiveOnKeywords(t *testing.T) {
	tests := []string{"HELP", "Help", "hELp", "files from 'work'", "EXIT"}

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
	app := shell.New(withPipeline(t, shell.Config{Out: out, Err: out}, pipelineFS()))

	err := app.Dispatch("filez from '.'")
	if err == nil {
		t.Fatal("Dispatch accepted an unknown word")
	}
	if !oerr.Is(err, oerr.KindUnknownTarget) {
		t.Errorf("error kind = %v, want unknown_target", err)
	}
	if !strings.Contains(err.Error(), `Did you mean "files"?`) {
		t.Errorf("no suggestion offered: %v", err)
	}
}

func TestDispatchUnknownVerbStaysSilentWhenFarOff(t *testing.T) {
	out := &bytes.Buffer{}
	app := shell.New(withPipeline(t, shell.Config{Out: out, Err: out}, pipelineFS()))

	err := app.Dispatch("frobnicate everything")
	if err == nil {
		t.Fatal("Dispatch accepted an unknown word")
	}
	if strings.Contains(err.Error(), "Did you mean") {
		t.Errorf("guessed at a far-off token: %v", err)
	}
}

func TestKnownWordsIncludesTargetsAndBuiltins(t *testing.T) {
	app := shell.New(shell.Config{})

	words := app.KnownWords()
	for _, target := range []string{"all", "files", "folders"} {
		if !slices.Contains(words, target) {
			t.Errorf("KnownWords() omits %q, so typos would never suggest it", target)
		}
	}
	if slices.Contains(words, "select") {
		t.Error("KnownWords() still offers select; the verb was removed")
	}
	for _, name := range app.Builtins().Names() {
		if !slices.Contains(words, name) {
			t.Errorf("KnownWords() omits builtin %q", name)
		}
	}
}

func TestLegacySelectVerbIsRejectedHelpfully(t *testing.T) {
	out := &bytes.Buffer{}
	app := shell.New(withPipeline(t, shell.Config{Out: out, Err: out}, pipelineFS()))

	err := app.Dispatch("select files from 'work'")
	if err == nil {
		t.Fatal("the removed select verb was accepted")
	}
	if !oerr.Is(err, oerr.KindNoVerbNeeded) {
		t.Errorf("error kind = %v, want no_verb_needed", err)
	}
	if !strings.Contains(err.Error(), "files from 'Documents'") {
		t.Errorf("error does not show the new form: %v", err)
	}
}

func TestRunStopsOnExitBuiltin(t *testing.T) {
	out := &bytes.Buffer{}
	app := shell.New(shell.Config{
		Reader:  reader.NewBasic(strings.NewReader("exit\nfiles from '.'\n"), out, nil),
		Out:     out,
		Err:     out,
		Version: "v1",
		Commit:  "c1",
	})

	if err := app.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if strings.Contains(out.String(), "files from '.'") {
		t.Error("Run kept reading after exit")
	}
}

func TestRunWritesErrorsToErrNotOut(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cfg := withPipeline(t, shell.Config{Out: stdout, Err: stderr}, pipelineFS())
	cfg.Reader = reader.NewBasic(strings.NewReader("filez from '.'\n"), stdout, nil)
	app := shell.New(cfg)

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
	cfg := withPipeline(t, shell.Config{Out: out, Err: errBuf}, pipelineFS())
	cfg.Reader = reader.NewBasic(strings.NewReader("slect x\nfiles from 'work'\nexit\n"), out, nil)

	app := shell.New(cfg)
	if err := app.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "notes.txt") {
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
