package test

import (
	"testing"

	"github.com/farhapartex/osql/internal/buildinfo"
)

func TestBuildInfoString(t *testing.T) {
	tests := []struct {
		name    string
		version string
		commit  string
		want    string
	}{
		{"build defaults", "dev", "none", "osql dev (none)"},
		{"stamped by ldflags", "v0.1.0", "abc1234", "osql v0.1.0 (abc1234)"},
		{"describe output with dirty suffix", "v0.1.0-2-gabc1234-dirty", "abc1234", "osql v0.1.0-2-gabc1234-dirty (abc1234)"},
		{"empty version falls back", "", "abc1234", "osql dev (abc1234)"},
		{"empty commit falls back", "v0.1.0", "", "osql v0.1.0 (none)"},
		{"both empty fall back", "", "", "osql dev (none)"},
		{"whitespace is preserved not trimmed", " v1 ", " abc ", "osql  v1  ( abc )"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildinfo.String(tt.version, tt.commit)
			if got != tt.want {
				t.Errorf("String(%q, %q) = %q, want %q", tt.version, tt.commit, got, tt.want)
			}
		})
	}
}
