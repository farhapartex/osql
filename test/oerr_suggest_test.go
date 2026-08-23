package test

import (
	"errors"
	"strings"
	"testing"

	"github.com/farhapartex/osql/internal/oerr"
)

var errPlain = errors.New("plain")

func TestDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"select", "select", 0},
		{"", "select", 6},
		{"select", "", 6},
		{"slect", "select", 1},
		{"selct", "select", 1},
		{"seleect", "select", 1},
		{"selecct", "select", 1},
		{"xyzzy", "select", 6},
		{"fils", "files", 1},
		{"flder", "folders", 2},
		{"kitten", "sitting", 3},
		{"flaw", "lawn", 2},
		{"a", "b", 1},
		{"ab", "ba", 2},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			if got := oerr.Distance(tt.a, tt.b); got != tt.want {
				t.Errorf("Distance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDistanceIsSymmetric(t *testing.T) {
	pairs := [][2]string{
		{"slect", "select"},
		{"kitten", "sitting"},
		{"", "files"},
		{"folders", "fold"},
	}

	for _, p := range pairs {
		forward := oerr.Distance(p[0], p[1])
		backward := oerr.Distance(p[1], p[0])
		if forward != backward {
			t.Errorf("Distance(%q,%q)=%d but Distance(%q,%q)=%d", p[0], p[1], forward, p[1], p[0], backward)
		}
	}
}

func TestDistanceHandlesMultiByteRunes(t *testing.T) {
	if got := oerr.Distance("café", "cafe"); got != 1 {
		t.Errorf("Distance(\"café\", \"cafe\") = %d, want 1; must compare runes not bytes", got)
	}
	if got := oerr.Distance("日本語", "日本"); got != 1 {
		t.Errorf("Distance(\"日本語\", \"日本\") = %d, want 1", got)
	}
	if got := oerr.Distance("日本語", "日本語"); got != 0 {
		t.Errorf("Distance on identical multi-byte strings = %d, want 0", got)
	}
}

func TestDistanceLongInputDoesNotOverflowRows(t *testing.T) {
	long := strings.Repeat("a", 500)
	short := "a"

	if got := oerr.Distance(long, short); got != 499 {
		t.Errorf("Distance(500a, a) = %d, want 499", got)
	}
	if got := oerr.Distance(short, long); got != 499 {
		t.Errorf("swapped argument order changed the result: %d", got)
	}
}

func TestSuggest(t *testing.T) {
	verbs := []string{"select"}
	fields := []string{"name", "name_like", "type", "count(child)"}
	targets := []string{"all", "files", "folders"}

	tests := []struct {
		name       string
		got        string
		candidates []string
		want       string
		wantOK     bool
	}{
		{"one edit away", "slect", verbs, "select", true},
		{"two edits away", "slct", verbs, "select", true},
		{"three edits away is silent", "xyzzy", verbs, "", false},
		{"exact match suggests itself", "select", verbs, "select", true},
		{"empty input", "", verbs, "", false},
		{"no candidates", "slect", nil, "", false},
		{"empty candidate list", "slect", []string{}, "", false},
		{"field typo", "nmae", fields, "name", true},
		{"field far off", "extension", fields, "", false},
		{"target typo", "file", targets, "files", true},
		{"target typo folder", "folder", targets, "folders", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := oerr.Suggest(tt.got, tt.candidates)
			if ok != tt.wantOK {
				t.Fatalf("Suggest(%q, %v) ok = %v, want %v (got %q)", tt.got, tt.candidates, ok, tt.wantOK, got)
			}
			if got != tt.want {
				t.Errorf("Suggest(%q) = %q, want %q", tt.got, got, tt.want)
			}
		})
	}
}

func TestSuggestPrefersNearestCandidate(t *testing.T) {
	got, ok := oerr.Suggest("nam", []string{"name_like", "name"})
	if !ok {
		t.Fatal("Suggest returned no match")
	}
	if got != "name" {
		t.Errorf("Suggest(\"nam\") = %q, want \"name\" — the nearer candidate", got)
	}
}

func TestSuggestBreaksTiesDeterministically(t *testing.T) {
	candidates := []string{"zeta", "beta"}

	first, _ := oerr.Suggest("eta", candidates)
	for range 20 {
		again, _ := oerr.Suggest("eta", candidates)
		if again != first {
			t.Fatalf("Suggest is non-deterministic on ties: got %q then %q", first, again)
		}
	}
	if first != "beta" {
		t.Errorf("tie broke to %q, want \"beta\" (lexicographically smaller)", first)
	}
}

func TestSuggestDoesNotMutateCandidates(t *testing.T) {
	candidates := []string{"select", "delete", "create"}
	before := append([]string(nil), candidates...)

	oerr.Suggest("slect", candidates)

	for i := range candidates {
		if candidates[i] != before[i] {
			t.Fatalf("Suggest reordered its input: %v, want %v", candidates, before)
		}
	}
}
