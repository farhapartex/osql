package engine

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/farhapartex/osql/internal/query"
)

const (
	SourceMacOS       = "macos"
	SourceSystem      = "system"
	SourceUser        = "user"
	SourceHomebrew    = "homebrew"
	SourceHomebrewCLI = "homebrew-cli"
	SourceApt         = "apt"
	SourceFlatpak     = "flatpak"
	SourceSnap        = "snap"
)

type App struct {
	Name     string
	Version  string
	Source   string
	ID       string
	Path     string
	Modified time.Time
}

type Catalog interface {
	Apps(ctx context.Context) ([]App, error)
}

func SortApps(apps []App) {
	slices.SortStableFunc(apps, func(a, b App) int {
		if order := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); order != 0 {
			return order
		}
		return strings.Compare(a.Source, b.Source)
	})
}

func FilterApps(ctx context.Context, catalog Catalog, matchers []Matcher) ([]App, error) {
	if catalog == nil {
		return nil, errNoCatalog
	}

	all, err := catalog.Apps(ctx)
	if err != nil {
		return nil, err
	}

	kept := make([]App, 0, len(all))
	for i := range all {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		ok, err := MatchAll(matchers, Entry{App: &all[i]})
		if err != nil {
			return nil, err
		}
		if ok {
			kept = append(kept, all[i])
		}
	}

	SortApps(kept)
	return kept, nil
}

type AppReport struct {
	Apps  []App
	Tools int
}

type AppLister interface {
	ListApps(ctx context.Context, stmt *query.Statement) (AppReport, error)
}

type AppsExecutor struct {
	catalog  Catalog
	compiler *Compiler
}

func NewAppsExecutor(catalog Catalog, compiler *Compiler) *AppsExecutor {
	return &AppsExecutor{catalog: catalog, compiler: compiler}
}

func (e *AppsExecutor) Verb() string {
	return query.VerbApps
}

func (e *AppsExecutor) ListApps(ctx context.Context, stmt *query.Statement) (AppReport, error) {
	matchers, err := e.compiler.CompileAll(stmt.Predicates, query.TargetApps)
	if err != nil {
		return AppReport{}, err
	}

	found, err := FilterApps(ctx, e.catalog, matchers)
	if err != nil {
		return AppReport{}, err
	}

	if mentionsSource(stmt.Predicates) {
		return AppReport{Apps: found}, nil
	}

	kept := make([]App, 0, len(found))
	tools := 0
	for _, app := range found {
		if app.Source == SourceHomebrewCLI {
			tools++
			continue
		}
		kept = append(kept, app)
	}
	return AppReport{Apps: kept, Tools: tools}, nil
}

func (e *AppsExecutor) Execute(ctx context.Context, stmt *query.Statement, out RowSink) error {
	report, err := e.ListApps(ctx, stmt)
	if err != nil {
		return err
	}
	return out.Push(Row{Name: query.TargetApps.String(), Count: int64(len(report.Apps))})
}

func mentionsSource(predicates []query.Predicate) bool {
	for _, p := range predicates {
		if p.Field == FieldSource {
			return true
		}
	}
	return false
}
