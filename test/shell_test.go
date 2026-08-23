package test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/farhapartex/osql/internal/shell"
)

func TestShellRunWritesBannerToInjectedWriter(t *testing.T) {
	out := &bytes.Buffer{}
	app := shell.New(shell.Config{Out: out, Version: "v0.1.0", Commit: "abc1234"})

	if err := app.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got := strings.TrimSpace(out.String())
	want := "osql v0.1.0 (abc1234)"
	if got != want {
		t.Errorf("Run() wrote %q, want %q", got, want)
	}
}

func TestShellRunFallsBackForUnstampedBuild(t *testing.T) {
	out := &bytes.Buffer{}
	app := shell.New(shell.Config{Out: out})

	if err := app.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got := strings.TrimSpace(out.String())
	want := "osql dev (none)"
	if got != want {
		t.Errorf("Run() wrote %q, want %q", got, want)
	}
}

func TestShellNewDefaultsNilWriters(t *testing.T) {
	app := shell.New(shell.Config{})

	if app == nil {
		t.Fatal("New() returned nil")
	}
	if err := app.Run(); err != nil {
		t.Errorf("Run() with nil writers error = %v; New must default them", err)
	}
}

func TestShellAcceptsFullyInjectedConfig(t *testing.T) {
	out := &bytes.Buffer{}
	store := &fakeStore{}

	cfg := shell.Config{
		Reader:   &fakeReader{lines: []string{"select files from '.'"}},
		Lexer:    &fakeLexer{},
		Parser:   &fakeParser{},
		Engine:   nil,
		Renderer: &fakeRenderer{},
		Store:    store,
		Out:      out,
		Err:      &bytes.Buffer{},
		Version:  "v1",
		Commit:   "c1",
	}

	app := shell.New(cfg)
	if err := app.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if out.Len() == 0 {
		t.Error("Run() wrote nothing to the injected Out")
	}
}
