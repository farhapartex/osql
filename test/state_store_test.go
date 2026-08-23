package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/farhapartex/osql/internal/state"
)

func newStore(t *testing.T, mutate func(*state.Options)) *state.DirStore {
	t.Helper()

	opts := state.Options{
		Root:    t.TempDir(),
		Version: "v0.1.0",
		Commit:  "abc1234",
		Now:     func() time.Time { return time.Date(2026, 8, 23, 22, 0, 0, 0, time.UTC) },
	}
	if mutate != nil {
		mutate(&opts)
	}
	return state.New(opts)
}

func TestEnsureCreatesDirectoryAndSystemFile(t *testing.T) {
	s := newStore(t, nil)

	if err := s.Ensure(); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	dir, err := os.Stat(s.Root())
	if err != nil {
		t.Fatalf("root not created: %v", err)
	}
	if !dir.IsDir() {
		t.Fatal("root is not a directory")
	}
	if got := dir.Mode().Perm(); got != 0o700 {
		t.Errorf("root mode = %04o, want 0700", got)
	}

	sys, err := os.Stat(s.SystemPath())
	if err != nil {
		t.Fatalf("system.txt not created: %v", err)
	}
	if got := sys.Mode().Perm(); got != 0o644 {
		t.Errorf("system.txt mode = %04o, want 0644", got)
	}
}

func TestEnsureIsIdempotent(t *testing.T) {
	s := newStore(t, nil)

	if err := s.Ensure(); err != nil {
		t.Fatalf("first Ensure() error = %v", err)
	}
	first, err := os.ReadFile(s.SystemPath())
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Ensure(); err != nil {
		t.Fatalf("second Ensure() error = %v", err)
	}
	second, err := os.ReadFile(s.SystemPath())
	if err != nil {
		t.Fatal(err)
	}

	if string(first) != string(second) {
		t.Error("second Ensure() rewrote system.txt; it must be a no-op when the file exists")
	}
}

func TestEnsureOnNestedRoot(t *testing.T) {
	base := t.TempDir()
	s := state.New(state.Options{Root: filepath.Join(base, "a", "b", ".osql")})

	if err := s.Ensure(); err != nil {
		t.Fatalf("Ensure() on a nested path error = %v", err)
	}
	if _, err := os.Stat(s.Root()); err != nil {
		t.Errorf("nested root not created: %v", err)
	}
}

func TestWriteSystemInfoForceRewrites(t *testing.T) {
	calls := 0
	s := newStore(t, func(o *state.Options) {
		o.Now = func() time.Time {
			calls++
			return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(calls) * time.Hour)
		}
	})

	if err := s.Ensure(); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(s.SystemPath())

	if err := s.WriteSystemInfo(true); err != nil {
		t.Fatalf("WriteSystemInfo(true) error = %v", err)
	}
	after, _ := os.ReadFile(s.SystemPath())

	if string(before) == string(after) {
		t.Error("WriteSystemInfo(true) did not rewrite the file")
	}
}

func TestSystemFileIsKeyValueLines(t *testing.T) {
	s := newStore(t, nil)
	if err := s.Ensure(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(s.SystemPath())
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		t.Fatal("system.txt is empty")
	}
	for _, line := range lines {
		if !strings.Contains(line, ": ") {
			t.Errorf("line %q is not in \"key: value\" form", line)
		}
	}

	want := []string{"version: v0.1.0", "commit: abc1234", "created_at: 2026-08-23T22:00:00Z"}
	for _, w := range want {
		if !strings.Contains(string(data), w) {
			t.Errorf("system.txt missing %q\ngot:\n%s", w, data)
		}
	}
}

func TestFormatSystemInfoIsStableAndOrdered(t *testing.T) {
	info := state.SystemInfo{
		Version:   "v1.2.3",
		Commit:    "deadbee",
		CreatedAt: time.Date(2026, 8, 23, 22, 0, 0, 0, time.UTC),
		OS:        "darwin",
		Arch:      "arm64",
		Kernel:    "22.6.0",
		CPUs:      8,
		GoVersion: "go1.26.7",
		Hostname:  "testbox",
		Username:  "tester",
		UID:       "501",
		Home:      "/Users/tester",
		Shell:     "/bin/zsh",
	}

	want := strings.Join([]string{
		"version: v1.2.3",
		"commit: deadbee",
		"created_at: 2026-08-23T22:00:00Z",
		"os: darwin",
		"arch: arm64",
		"kernel: 22.6.0",
		"cpus: 8",
		"go: go1.26.7",
		"hostname: testbox",
		"user: tester",
		"uid: 501",
		"home: /Users/tester",
		"shell: /bin/zsh",
	}, "\n") + "\n"

	if got := state.FormatSystemInfo(info); got != want {
		t.Errorf("FormatSystemInfo mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatSystemInfoConvertsTimeToUTC(t *testing.T) {
	zone := time.FixedZone("IST", 6*3600)
	info := state.SystemInfo{CreatedAt: time.Date(2026, 8, 23, 4, 0, 0, 0, zone)}

	got := state.FormatSystemInfo(info)
	if !strings.Contains(got, "created_at: 2026-08-22T22:00:00Z") {
		t.Errorf("timestamp not normalised to UTC:\n%s", got)
	}
}

func TestCollectSystemInfoFillsEveryField(t *testing.T) {
	now := time.Date(2026, 8, 23, 22, 0, 0, 0, time.UTC)
	info := state.CollectSystemInfo("v1", "c1", now)

	checks := map[string]string{
		"OS":        info.OS,
		"Arch":      info.Arch,
		"Kernel":    info.Kernel,
		"GoVersion": info.GoVersion,
		"Hostname":  info.Hostname,
		"Username":  info.Username,
		"UID":       info.UID,
		"Home":      info.Home,
		"Shell":     info.Shell,
	}
	for name, value := range checks {
		if value == "" {
			t.Errorf("%s is empty; it must fall back rather than render blank", name)
		}
	}
	if info.CPUs < 1 {
		t.Errorf("CPUs = %d, want at least 1", info.CPUs)
	}
	if !info.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want the injected clock value %v", info.CreatedAt, now)
	}
}

func TestDefaultRootEndsInDotOsql(t *testing.T) {
	root, err := state.DefaultRoot()
	if err != nil {
		t.Skipf("no home directory available: %v", err)
	}
	if filepath.Base(root) != ".osql" {
		t.Errorf("DefaultRoot() = %q, want it to end in .osql", root)
	}
	if !filepath.IsAbs(root) {
		t.Errorf("DefaultRoot() = %q, want an absolute path", root)
	}
}
