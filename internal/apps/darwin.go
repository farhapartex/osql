//go:build darwin

package apps

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/farhapartex/osql/internal/engine"
)

const bundleSuffix = ".app"

var homebrewPrefixes = []string{"/opt/homebrew", "/usr/local"}

func DefaultSources(home string) []Source {
	sources := []Source{
		bundleSource{dir: "/Applications", label: engine.SourceSystem},
		bundleSource{dir: "/System/Applications", label: engine.SourceMacOS},
		bundleSource{dir: filepath.Join(home, "Applications"), label: engine.SourceUser},
	}

	for _, prefix := range homebrewPrefixes {
		sources = append(sources,
			homebrewSource{dir: filepath.Join(prefix, "Caskroom"), label: engine.SourceHomebrew},
			homebrewSource{dir: filepath.Join(prefix, "Cellar"), label: engine.SourceHomebrewCLI},
		)
	}
	return sources
}

type bundleSource struct {
	dir   string
	label string
}

func (s bundleSource) Name() string {
	return s.dir
}

func (s bundleSource) Apps(ctx context.Context) ([]engine.App, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	apps := make([]engine.App, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !strings.HasSuffix(entry.Name(), bundleSuffix) {
			continue
		}

		path := filepath.Join(s.dir, entry.Name())
		app := engine.App{
			Name:   strings.TrimSuffix(entry.Name(), bundleSuffix),
			Source: s.label,
			Path:   path,
		}
		if info, err := entry.Info(); err == nil {
			app.Modified = info.ModTime()
		}
		readBundleInfo(path, &app)
		apps = append(apps, app)
	}
	return apps, nil
}

func readBundleInfo(path string, app *engine.App) {
	file, err := os.Open(filepath.Join(path, "Contents", "Info.plist"))
	if err != nil {
		return
	}
	defer file.Close()

	values, err := ParsePlist(file)
	if err != nil {
		return
	}
	app.Version = values[KeyBundleVersion]
	app.ID = values[KeyBundleIdentifier]
}

type homebrewSource struct {
	dir   string
	label string
}

func (s homebrewSource) Name() string {
	return s.dir
}

func (s homebrewSource) Apps(ctx context.Context) ([]engine.App, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	apps := make([]engine.App, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		app := engine.App{
			Name:    entry.Name(),
			Source:  s.label,
			Path:    filepath.Join(s.dir, entry.Name()),
			Version: latestVersionDir(filepath.Join(s.dir, entry.Name())),
		}
		if info, err := entry.Info(); err == nil {
			app.Modified = info.ModTime()
		}
		apps = append(apps, app)
	}
	return apps, nil
}

func latestVersionDir(path string) string {
	entries, err := os.ReadDir(path)
	if err != nil {
		return ""
	}

	latest := ""
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.Name() > latest {
			latest = entry.Name()
		}
	}
	return latest
}
