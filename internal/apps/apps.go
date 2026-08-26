package apps

import (
	"context"
	"strings"
	"unicode"

	"github.com/farhapartex/osql/internal/engine"
)

type Source interface {
	Name() string
	Apps(ctx context.Context) ([]engine.App, error)
}

type Catalog struct {
	sources []Source
}

func NewCatalog(sources ...Source) *Catalog {
	return &Catalog{sources: sources}
}

func NormalizeName(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

var sourceRank = map[string]int{
	engine.SourceHomebrew:    4,
	engine.SourceHomebrewCLI: 3,
	engine.SourceApt:         3,
	engine.SourceFlatpak:     3,
	engine.SourceSnap:        3,
	engine.SourceUser:        2,
	engine.SourceSystem:      1,
	engine.SourceMacOS:       0,
}

func (c *Catalog) Apps(ctx context.Context) ([]engine.App, error) {
	byKey := make(map[string]*engine.App)
	order := make([]string, 0, 128)

	for _, source := range c.sources {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		found, err := source.Apps(ctx)
		if err != nil {
			continue
		}

		for _, app := range found {
			key := NormalizeName(app.Name)
			if key == "" {
				continue
			}

			existing, ok := byKey[key]
			if !ok {
				stored := app
				byKey[key] = &stored
				order = append(order, key)
				continue
			}
			merge(existing, app)
		}
	}

	apps := make([]engine.App, 0, len(order))
	for _, key := range order {
		apps = append(apps, *byKey[key])
	}

	engine.SortApps(apps)
	return apps, nil
}

func merge(into *engine.App, from engine.App) {
	if into.Version == "" {
		into.Version = from.Version
	}
	if into.ID == "" {
		into.ID = from.ID
	}
	if into.Path == "" {
		into.Path = from.Path
	}
	if into.Modified.IsZero() {
		into.Modified = from.Modified
	}
	if sourceRank[from.Source] > sourceRank[into.Source] {
		into.Source = from.Source
	}
}
