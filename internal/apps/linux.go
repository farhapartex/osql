//go:build linux

package apps

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/farhapartex/osql/internal/engine"
)

const desktopSuffix = ".desktop"

const dpkgStatusPath = "/var/lib/dpkg/status"

func DefaultSources(home string) []Source {
	versions := readDpkgVersions(dpkgStatusPath)

	return []Source{
		desktopSource{dir: "/usr/share/applications", label: engine.SourceSystem, versions: versions},
		desktopSource{dir: "/usr/local/share/applications", label: engine.SourceSystem, versions: versions},
		desktopSource{dir: filepath.Join(home, ".local/share/applications"), label: engine.SourceUser, versions: versions},
		desktopSource{dir: "/var/lib/flatpak/exports/share/applications", label: engine.SourceFlatpak},
		desktopSource{dir: filepath.Join(home, ".local/share/flatpak/exports/share/applications"), label: engine.SourceFlatpak},
		desktopSource{dir: "/var/lib/snapd/desktop/applications", label: engine.SourceSnap},
	}
}

type desktopSource struct {
	dir      string
	label    string
	versions map[string]string
}

func (s desktopSource) Name() string {
	return s.dir
}

func (s desktopSource) Apps(ctx context.Context) ([]engine.App, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	apps := make([]engine.App, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), desktopSuffix) {
			continue
		}

		path := filepath.Join(s.dir, entry.Name())
		entryName, visible := readDesktopEntry(path)
		if !visible {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), desktopSuffix)
		if entryName == "" {
			entryName = id
		}

		app := engine.App{
			Name:   entryName,
			Source: s.label,
			ID:     id,
			Path:   path,
		}
		if info, err := entry.Info(); err == nil {
			app.Modified = info.ModTime()
		}
		if version, ok := s.lookupVersion(entryName, id); ok {
			app.Version = version
			if s.label == engine.SourceSystem {
				app.Source = engine.SourceApt
			}
		}
		apps = append(apps, app)
	}
	return apps, nil
}

func (s desktopSource) lookupVersion(name, id string) (string, bool) {
	if s.versions == nil {
		return "", false
	}
	if version, ok := s.versions[NormalizeName(id)]; ok {
		return version, true
	}
	version, ok := s.versions[NormalizeName(name)]
	return version, ok
}

func readDesktopEntry(path string) (string, bool) {
	file, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer file.Close()

	name := ""
	kind := ""
	inEntry := false
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inEntry = line == "[Desktop Entry]"
			continue
		}
		if !inEntry {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "Name":
			if name == "" {
				name = value
			}
		case "Type":
			kind = value
		case "NoDisplay", "Hidden":
			if strings.EqualFold(value, "true") {
				return "", false
			}
		}
	}

	if kind != "" && kind != "Application" {
		return "", false
	}
	return name, true
}

func readDpkgVersions(path string) map[string]string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	versions := make(map[string]string, 1024)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	pkg := ""
	version := ""
	installed := false

	flush := func() {
		if pkg != "" && version != "" && installed {
			versions[NormalizeName(pkg)] = version
		}
		pkg, version, installed = "", "", false
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}

		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		value = strings.TrimSpace(value)

		switch key {
		case "Package":
			pkg = value
		case "Version":
			version = value
		case "Status":
			installed = strings.Contains(value, "installed") && !strings.HasPrefix(value, "deinstall")
		}
	}
	flush()

	return versions
}
