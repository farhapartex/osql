package test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/farhapartex/osql/internal/engine"
)

func naiveMatch(pattern, name string) bool {
	if pattern == "" {
		return name == ""
	}
	if pattern[0] == '%' || pattern[0] == '*' {
		for i := 0; i <= len(name); i++ {
			if naiveMatch(pattern[1:], name[i:]) {
				return true
			}
		}
		return false
	}
	if name == "" || name[0] != pattern[0] {
		return false
	}
	return naiveMatch(pattern[1:], name[1:])
}

func TestPatternShapes(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		want    bool
	}{
		{"contains hit", "%report%", "q4-report-final.pdf", true},
		{"contains miss", "%report%", "invoice.pdf", false},
		{"prefix hit", "report%", "report-2026.pdf", true},
		{"prefix miss", "report%", "final-report.pdf", false},
		{"suffix hit", "%report", "final-report", true},
		{"suffix miss", "%report", "report-final", false},
		{"exact hit", "notes.txt", "notes.txt", true},
		{"exact miss", "notes.txt", "notes.txtx", false},
		{"exact miss shorter", "notes.txt", "notes.tx", false},
		{"middle wildcard hit", "a%b", "azzzb", true},
		{"middle wildcard empty run", "a%b", "ab", true},
		{"middle wildcard miss", "a%b", "acd", false},
		{"multiple wildcards", "a%b%c", "a1b2c", true},
		{"multiple wildcards miss order", "a%b%c", "a1c2b", false},
		{"star alias contains", "*report*", "my-report-v2", true},
		{"star alias prefix", "report*", "report-1", true},
		{"star alias suffix", "*.log", "server.log", true},
		{"mixed wildcards", "%report*", "q4-report-final", true},
		{"log extension", "%.log", "server.log", true},
		{"log extension miss", "%.log", "server.logx", false},
		{"test prefix", "test_%", "test_lexer.go", true},
		{"test prefix miss", "test_%", "lexer_test.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.CompilePattern(tt.pattern).Match(tt.input)
			if got != tt.want {
				t.Errorf("CompilePattern(%q).Match(%q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestPatternEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		want    bool
	}{
		{"empty pattern matches empty", "", "", true},
		{"empty pattern rejects non-empty", "", "a", false},
		{"lone percent matches empty", "%", "", true},
		{"lone percent matches anything", "%", "anything at all", true},
		{"lone star matches anything", "*", "anything", true},
		{"double percent matches anything", "%%", "anything", true},
		{"triple percent matches anything", "%%%", "anything", true},
		{"mixed wildcard run matches anything", "%*%", "anything", true},
		{"adjacent wildcards around literal", "%%a%%", "xxaxx", true},
		{"adjacent wildcards around literal miss", "%%a%%", "xxxx", false},
		{"exact empty input against literal", "a", "", false},
		{"wildcard then nothing", "a%", "a", true},
		{"nothing then wildcard", "%a", "a", true},
		{"repeated literal overlap", "a%a", "aa", true},
		{"repeated literal too short", "a%a", "a", false},
		{"repeated literal three", "a%a%a", "aaa", true},
		{"repeated literal three too short", "a%a%a", "aa", false},
		{"whole pattern is literal with spaces", "my file", "my file", true},
		{"wildcard spanning spaces", "my%file", "my long file", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.CompilePattern(tt.pattern).Match(tt.input)
			if got != tt.want {
				t.Errorf("CompilePattern(%q).Match(%q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestPatternIsCaseSensitive(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		{"%report%", "REPORT", false},
		{"%REPORT%", "report", false},
		{"%report%", "report", true},
		{"notes.txt", "NOTES.TXT", false},
		{"%.TXT", "a.txt", false},
	}

	for _, tt := range tests {
		got := engine.CompilePattern(tt.pattern).Match(tt.input)
		if got != tt.want {
			t.Errorf("Match(%q) against %q = %v, want %v; values are case-sensitive", tt.input, tt.pattern, got, tt.want)
		}
	}
}

func TestPatternHandlesUnicode(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		{"%語%", "日本語のファイル", true},
		{"日本%", "日本語", true},
		{"%ファイル", "日本語のファイル", true},
		{"%語%", "日本", false},
		{"café%", "café-notes", true},
		{"%é", "café", true},
	}

	for _, tt := range tests {
		got := engine.CompilePattern(tt.pattern).Match(tt.input)
		if got != tt.want {
			t.Errorf("Match(%q) against %q = %v, want %v", tt.input, tt.pattern, got, tt.want)
		}
	}
}

func TestPatternIsExact(t *testing.T) {
	tests := []struct {
		pattern string
		want    bool
	}{
		{"notes.txt", true},
		{"", true},
		{"%notes", false},
		{"notes%", false},
		{"*", false},
		{"a%b", false},
	}

	for _, tt := range tests {
		if got := engine.CompilePattern(tt.pattern).IsExact(); got != tt.want {
			t.Errorf("CompilePattern(%q).IsExact() = %v, want %v", tt.pattern, got, tt.want)
		}
	}
}

func TestPatternStringReturnsOriginal(t *testing.T) {
	for _, pattern := range []string{"%report%", "notes.txt", "", "*", "a%b%c"} {
		if got := engine.CompilePattern(pattern).String(); got != pattern {
			t.Errorf("String() = %q, want %q", got, pattern)
		}
	}
}

func TestPatternMatchesNaiveOracle(t *testing.T) {
	patterns := []string{
		"", "a", "ab", "%", "*", "%%", "a%", "%a", "%a%", "a%b", "a%a",
		"a%b%c", "%a%b%", "ab%cd", "%%a%%", "a*b", "*a*", "a%%b",
	}
	inputs := []string{
		"", "a", "b", "aa", "ab", "ba", "abc", "aab", "abb", "aba",
		"abcd", "abcabc", "xaybzc", "aaa", "cba", "ab cd",
	}

	for _, pattern := range patterns {
		for _, input := range inputs {
			want := naiveMatch(pattern, input)
			got := engine.CompilePattern(pattern).Match(input)
			if got != want {
				t.Errorf("pattern %q input %q: fast=%v oracle=%v", pattern, input, got, want)
			}
		}
	}
}

func TestPatternMatchDoesNotAllocate(t *testing.T) {
	patterns := map[string]string{
		"contains": "%report%",
		"prefix":   "report%",
		"suffix":   "%report",
		"exact":    "report.txt",
		"multi":    "a%b%c%d",
		"matchAll": "%",
	}

	for name, raw := range patterns {
		t.Run(name, func(t *testing.T) {
			p := engine.CompilePattern(raw)
			input := "a-report-b-c-d.txt"

			allocs := testing.AllocsPerRun(200, func() {
				p.Match(input)
			})
			if allocs != 0 {
				t.Errorf("Match allocated %.0f times per run, want 0", allocs)
			}
		})
	}
}

func TestPatternCompiledOncePerQueryNotPerCandidate(t *testing.T) {
	p := engine.CompilePattern("%report%")

	names := make([]string, 1000)
	for i := range names {
		names[i] = fmt.Sprintf("file-%d-report.txt", i)
	}

	allocs := testing.AllocsPerRun(20, func() {
		for _, n := range names {
			p.Match(n)
		}
	})
	if allocs != 0 {
		t.Errorf("matching 1000 candidates allocated %.0f times, want 0", allocs)
	}
}

func TestPatternPathologicalInputTerminatesQuickly(t *testing.T) {
	pattern := strings.Repeat("%a", 40) + "%b"
	input := strings.Repeat("a", 20000)

	p := engine.CompilePattern(pattern)

	done := make(chan bool, 1)
	start := time.Now()
	go func() { done <- p.Match(input) }()

	select {
	case got := <-done:
		if got {
			t.Errorf("Match returned true; input has no 'b'")
		}
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Errorf("Match took %v; the greedy scan must not backtrack exponentially", elapsed)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Match did not terminate; a backtracking matcher would blow up here")
	}
}

func TestPatternLongInputs(t *testing.T) {
	long := strings.Repeat("x", 100000) + "report" + strings.Repeat("y", 100000)

	if !engine.CompilePattern("%report%").Match(long) {
		t.Error("failed to find a literal in a 200KB name")
	}
	if engine.CompilePattern("%missing%").Match(long) {
		t.Error("false positive on a 200KB name")
	}
}

func TestPatternReuseIsStateless(t *testing.T) {
	p := engine.CompilePattern("%report%")

	inputs := []struct {
		in   string
		want bool
	}{
		{"a-report-b", true},
		{"nothing", false},
		{"report", true},
		{"", false},
		{"report", true},
	}

	for range 3 {
		for _, tt := range inputs {
			if got := p.Match(tt.in); got != tt.want {
				t.Fatalf("Match(%q) = %v, want %v; a compiled pattern must be reusable", tt.in, got, tt.want)
			}
		}
	}
}

func FuzzPatternMatchesOracle(f *testing.F) {
	seeds := [][2]string{
		{"%report%", "q4-report.pdf"},
		{"a%b", "ab"},
		{"", ""},
		{"%", "anything"},
		{"a%a", "aa"},
		{"%%a%%", "xax"},
	}
	for _, s := range seeds {
		f.Add(s[0], s[1])
	}

	f.Fuzz(func(t *testing.T, pattern, name string) {
		if len(pattern) > 12 || len(name) > 12 {
			t.Skip()
		}

		got := engine.CompilePattern(pattern).Match(name)
		want := naiveMatch(pattern, name)
		if got != want {
			t.Errorf("pattern %q name %q: fast=%v oracle=%v", pattern, name, got, want)
		}
	})
}
