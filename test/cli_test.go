package test

import (
	"strings"
	"testing"

	"github.com/farhapartex/osql/internal/cli"
)

func TestParseValidInvocations(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want cli.Options
	}{
		{"no args starts the shell", nil, cli.Options{Command: cli.CommandShell}},
		{"empty args starts the shell", []string{}, cli.Options{Command: cli.CommandShell}},
		{"version long", []string{"--version"}, cli.Options{Command: cli.CommandVersion}},
		{"version short", []string{"-v"}, cli.Options{Command: cli.CommandVersion}},
		{"help long", []string{"--help"}, cli.Options{Command: cli.CommandHelp}},
		{"help short", []string{"-h"}, cli.Options{Command: cli.CommandHelp}},
		{"init", []string{"init"}, cli.Options{Command: cli.CommandInit}},
		{"init reinit", []string{"init", "--reinit"}, cli.Options{Command: cli.CommandInit, Reinit: true}},
		{"no history", []string{"--no-history"}, cli.Options{Command: cli.CommandShell, NoHistory: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cli.Parse(tt.args)
			if err != nil {
				t.Fatalf("Parse(%v) error = %v", tt.args, err)
			}
			if got != tt.want {
				t.Errorf("Parse(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestParseRejectsInvalidInvocations(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"unknown flag", []string{"--frobnicate"}},
		{"unknown subcommand", []string{"start"}},
		{"bare path", []string{"Documents"}},
		{"reinit without init", []string{"--reinit"}},
		{"no-history with init", []string{"init", "--no-history"}},
		{"version with init", []string{"init", "--version"}},
		{"single dash", []string{"-"}},
		{"double dash", []string{"--"}},
		{"empty string arg", []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := cli.Parse(tt.args); err == nil {
				t.Errorf("Parse(%v) accepted an invalid invocation", tt.args)
			}
		})
	}
}

func TestParseErrorsAreReadable(t *testing.T) {
	_, err := cli.Parse([]string{"--frobnicate"})
	if err == nil {
		t.Fatal("expected an error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "--frobnicate") {
		t.Errorf("error does not quote the offending argument: %q", msg)
	}
	if !strings.Contains(msg, "--help") {
		t.Errorf("error does not say what to do next: %q", msg)
	}
}

func TestParseInitAcceptsReinitOnce(t *testing.T) {
	got, err := cli.Parse([]string{"init", "--reinit", "--reinit"})
	if err != nil {
		t.Fatalf("repeating a flag should be harmless, got %v", err)
	}
	if !got.Reinit {
		t.Error("Reinit not set")
	}
}

func TestCommandString(t *testing.T) {
	tests := []struct {
		cmd  cli.Command
		want string
	}{
		{cli.CommandShell, "shell"},
		{cli.CommandVersion, "version"},
		{cli.CommandInit, "init"},
		{cli.CommandHelp, "help"},
		{cli.Command(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.cmd.String(); got != tt.want {
			t.Errorf("Command(%d).String() = %q, want %q", int(tt.cmd), got, tt.want)
		}
	}
}

func TestUsageMentionsEveryFlag(t *testing.T) {
	for _, flag := range []string{"init", "--reinit", "--no-history", "--version", "--help"} {
		if !strings.Contains(cli.Usage, flag) {
			t.Errorf("usage text does not document %q", flag)
		}
	}
}
