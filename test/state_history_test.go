package test

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/farhapartex/osql/internal/state"
)

func historyFor(t *testing.T, root string, limit int) (*state.DirStore, state.History) {
	t.Helper()

	s := state.New(state.Options{Root: root, HistoryLimit: limit})
	if err := s.Ensure(); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	h, err := s.History()
	if err != nil {
		t.Fatalf("History() error = %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s, h
}

func TestHistoryAppendAndRead(t *testing.T) {
	s, h := historyFor(t, t.TempDir(), 0)

	for _, line := range []string{"files from '.'", "all from '~'", "exit"} {
		if err := h.Append(line); err != nil {
			t.Fatalf("Append(%q) error = %v", line, err)
		}
	}

	lines, err := h.Lines(0)
	if err != nil {
		t.Fatalf("Lines() error = %v", err)
	}
	want := []string{"files from '.'", "all from '~'", "exit"}
	if len(lines) != len(want) {
		t.Fatalf("Lines() returned %d lines, want %d: %v", len(lines), len(want), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}

	info, err := os.Stat(s.HistoryPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("history.txt mode = %04o, want 0600 (it holds the user's paths)", got)
	}
}

func TestHistoryLinesLimit(t *testing.T) {
	_, h := historyFor(t, t.TempDir(), 0)

	for i := range 10 {
		if err := h.Append(fmt.Sprintf("line-%d", i)); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		limit     int
		wantCount int
		wantFirst string
	}{
		{0, 10, "line-0"},
		{-1, 10, "line-0"},
		{3, 3, "line-7"},
		{1, 1, "line-9"},
		{10, 10, "line-0"},
		{50, 10, "line-0"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("limit_%d", tt.limit), func(t *testing.T) {
			lines, err := h.Lines(tt.limit)
			if err != nil {
				t.Fatal(err)
			}
			if len(lines) != tt.wantCount {
				t.Fatalf("Lines(%d) returned %d lines, want %d", tt.limit, len(lines), tt.wantCount)
			}
			if lines[0] != tt.wantFirst {
				t.Errorf("Lines(%d)[0] = %q, want %q", tt.limit, lines[0], tt.wantFirst)
			}
		})
	}
}

func TestHistoryEmptyBeforeAnyAppend(t *testing.T) {
	_, h := historyFor(t, t.TempDir(), 0)

	lines, err := h.Lines(0)
	if err != nil {
		t.Fatalf("Lines() on a fresh history error = %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("fresh history has %d lines, want 0", len(lines))
	}
}

func TestHistoryTrimKeepsNewestLines(t *testing.T) {
	root := t.TempDir()
	const limit = 100
	const written = 150

	s1 := state.New(state.Options{Root: root, HistoryLimit: limit})
	if err := s1.Ensure(); err != nil {
		t.Fatal(err)
	}
	h1, err := s1.History()
	if err != nil {
		t.Fatal(err)
	}
	for i := range written {
		if err := h1.Append(fmt.Sprintf("line-%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s1.Close(); err != nil {
		t.Fatal(err)
	}

	s2 := state.New(state.Options{Root: root, HistoryLimit: limit})
	h2, err := s2.History()
	if err != nil {
		t.Fatalf("reopening history error = %v", err)
	}
	defer s2.Close()

	lines, err := h2.Lines(0)
	if err != nil {
		t.Fatal(err)
	}

	if len(lines) != limit {
		t.Fatalf("after trim history has %d lines, want %d", len(lines), limit)
	}
	if lines[0] != fmt.Sprintf("line-%d", written-limit) {
		t.Errorf("oldest kept line = %q, want %q — trim must drop the oldest, not the newest", lines[0], fmt.Sprintf("line-%d", written-limit))
	}
	if lines[len(lines)-1] != fmt.Sprintf("line-%d", written-1) {
		t.Errorf("newest line = %q, want %q; the most recent entry must survive", lines[len(lines)-1], fmt.Sprintf("line-%d", written-1))
	}
}

func TestHistoryTrimPreservesMode(t *testing.T) {
	root := t.TempDir()

	s1 := state.New(state.Options{Root: root, HistoryLimit: 5})
	if err := s1.Ensure(); err != nil {
		t.Fatal(err)
	}
	h1, _ := s1.History()
	for i := range 20 {
		h1.Append(fmt.Sprintf("l%d", i))
	}
	s1.Close()

	s2 := state.New(state.Options{Root: root, HistoryLimit: 5})
	if _, err := s2.History(); err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	info, err := os.Stat(s2.HistoryPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("mode after atomic trim = %04o, want 0600", got)
	}
}

func TestHistoryTrimIsNoopUnderLimit(t *testing.T) {
	root := t.TempDir()

	s1 := state.New(state.Options{Root: root, HistoryLimit: 100})
	if err := s1.Ensure(); err != nil {
		t.Fatal(err)
	}
	h1, _ := s1.History()
	for i := range 10 {
		h1.Append(fmt.Sprintf("line-%d", i))
	}
	s1.Close()

	s2 := state.New(state.Options{Root: root, HistoryLimit: 100})
	h2, _ := s2.History()
	defer s2.Close()

	lines, _ := h2.Lines(0)
	if len(lines) != 10 {
		t.Errorf("under-limit history was modified: %d lines, want 10", len(lines))
	}
}

func TestHistoryDefaultLimitIsTenThousand(t *testing.T) {
	if state.DefaultHistoryLimit != 10000 {
		t.Errorf("DefaultHistoryLimit = %d, want 10000 per the plan", state.DefaultHistoryLimit)
	}
}

func TestHistoryClear(t *testing.T) {
	_, h := historyFor(t, t.TempDir(), 0)

	for i := range 5 {
		h.Append(fmt.Sprintf("line-%d", i))
	}
	if err := h.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	lines, err := h.Lines(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Errorf("after Clear() history has %d lines, want 0", len(lines))
	}

	if err := h.Append("after clear"); err != nil {
		t.Fatalf("Append after Clear() error = %v; the handle must be reusable", err)
	}
	lines, _ = h.Lines(0)
	if len(lines) != 1 || lines[0] != "after clear" {
		t.Errorf("after Clear() then Append, lines = %v, want [\"after clear\"]", lines)
	}
}

func TestHistoryAppendAfterCloseFails(t *testing.T) {
	s := state.New(state.Options{Root: t.TempDir()})
	if err := s.Ensure(); err != nil {
		t.Fatal(err)
	}
	h, _ := s.History()

	if err := h.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := h.Append("x"); err == nil {
		t.Error("Append after Close() succeeded; want an error")
	}
}

func TestHistoryCloseIsIdempotent(t *testing.T) {
	s := state.New(state.Options{Root: t.TempDir()})
	if err := s.Ensure(); err != nil {
		t.Fatal(err)
	}
	h, _ := s.History()

	if err := h.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := h.Close(); err != nil {
		t.Errorf("second Close() error = %v, want nil", err)
	}
}

func TestHistoryPreservesLinesWithSpecialCharacters(t *testing.T) {
	_, h := historyFor(t, t.TempDir(), 0)

	inputs := []string{
		"files from 'my folder'",
		"files from '~' where name_like = '%report%'",
		"folders from 'src' where count(child) <= 2",
		"日本語のファイル",
		"  leading and trailing  ",
		"tab\tseparated",
	}
	for _, in := range inputs {
		if err := h.Append(in); err != nil {
			t.Fatal(err)
		}
	}

	lines, err := h.Lines(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != len(inputs) {
		t.Fatalf("got %d lines, want %d", len(lines), len(inputs))
	}
	for i := range inputs {
		if lines[i] != inputs[i] {
			t.Errorf("line %d = %q, want %q", i, lines[i], inputs[i])
		}
	}
}

func TestHistoryHandlesVeryLongLine(t *testing.T) {
	_, h := historyFor(t, t.TempDir(), 0)

	long := "files from '" + strings.Repeat("a", 200000) + "'"
	if err := h.Append(long); err != nil {
		t.Fatalf("Append(long) error = %v", err)
	}

	lines, err := h.Lines(0)
	if err != nil {
		t.Fatalf("Lines() with a long entry error = %v", err)
	}
	if len(lines) != 1 || lines[0] != long {
		t.Errorf("long line not round-tripped intact (got %d lines, len %d)", len(lines), len(lines[0]))
	}
}

func TestHistoryConcurrentAppends(t *testing.T) {
	_, h := historyFor(t, t.TempDir(), 0)

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if err := h.Append(fmt.Sprintf("line-%d", n)); err != nil {
				t.Errorf("concurrent Append error = %v", err)
			}
		}(i)
	}
	wg.Wait()

	lines, err := h.Lines(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 50 {
		t.Errorf("concurrent appends produced %d lines, want 50", len(lines))
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "line-") {
			t.Errorf("interleaved write corrupted a line: %q", line)
		}
	}
}

func TestNoHistoryOptionDiscardsEverything(t *testing.T) {
	s := state.New(state.Options{Root: t.TempDir(), NoHistory: true})
	if err := s.Ensure(); err != nil {
		t.Fatal(err)
	}
	h, err := s.History()
	if err != nil {
		t.Fatalf("History() with NoHistory error = %v", err)
	}

	if err := h.Append("files from '.'"); err != nil {
		t.Errorf("Append with NoHistory error = %v, want nil", err)
	}
	lines, err := h.Lines(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Errorf("NoHistory returned %d lines, want 0", len(lines))
	}
	if _, err := os.Stat(s.HistoryPath()); !os.IsNotExist(err) {
		t.Error("NoHistory must not create history.txt")
	}
	if err := h.Clear(); err != nil {
		t.Errorf("Clear with NoHistory error = %v", err)
	}
	if err := h.Close(); err != nil {
		t.Errorf("Close with NoHistory error = %v", err)
	}
}

func TestHistoryReturnsSameInstance(t *testing.T) {
	s := state.New(state.Options{Root: t.TempDir()})
	if err := s.Ensure(); err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	first, err := s.History()
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.History()
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Error("History() opened a second handle; the file handle is held for the session")
	}
}
