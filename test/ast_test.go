package test

import (
	"testing"

	"github.com/farhapartex/osql/internal/query"
)

func TestTargetString(t *testing.T) {
	tests := []struct {
		target query.Target
		want   string
	}{
		{query.TargetAll, "all"},
		{query.TargetFiles, "files"},
		{query.TargetFolders, "folders"},
		{query.Target(99), "unknown"},
		{query.Target(-1), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.target.String(); got != tt.want {
			t.Errorf("Target(%d).String() = %q, want %q", int(tt.target), got, tt.want)
		}
	}
}

func TestParseTarget(t *testing.T) {
	tests := []struct {
		in     string
		want   query.Target
		wantOk bool
	}{
		{"all", query.TargetAll, true},
		{"files", query.TargetFiles, true},
		{"folders", query.TargetFolders, true},
		{"file", query.TargetAll, false},
		{"folder", query.TargetAll, false},
		{"Files", query.TargetAll, false},
		{"", query.TargetAll, false},
		{"documents", query.TargetAll, false},
	}

	for _, tt := range tests {
		got, ok := query.ParseTarget(tt.in)
		if ok != tt.wantOk {
			t.Errorf("ParseTarget(%q) ok = %v, want %v", tt.in, ok, tt.wantOk)
			continue
		}
		if ok && got != tt.want {
			t.Errorf("ParseTarget(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseTargetRejectsSingularForms(t *testing.T) {
	for _, singular := range []string{"file", "folder"} {
		if _, ok := query.ParseTarget(singular); ok {
			t.Errorf("ParseTarget(%q) accepted a singular target; plural-only per PLAN 3.2", singular)
		}
	}
}

func TestTokenKindString(t *testing.T) {
	tests := []struct {
		kind query.TokenKind
		want string
	}{
		{query.TokenEOF, "eof"},
		{query.TokenIdent, "ident"},
		{query.TokenString, "string"},
		{query.TokenOperator, "operator"},
		{query.TokenLParen, "lparen"},
		{query.TokenRParen, "rparen"},
		{query.TokenKind(42), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("TokenKind(%d).String() = %q, want %q", int(tt.kind), got, tt.want)
		}
	}
}

func TestStatementZeroValue(t *testing.T) {
	var s query.Statement

	if s.Recursive {
		t.Error("Statement.Recursive must default to false; recursion is opt-in per PLAN 3.3")
	}
	if s.Target != query.TargetAll {
		t.Errorf("zero Target = %v, want TargetAll", s.Target)
	}
	if len(s.Predicates) != 0 {
		t.Error("zero Statement must carry no predicates")
	}
}
