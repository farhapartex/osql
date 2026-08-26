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
	Name      string
	Version   string
	Source    string
	ID        string
	Path      string
	Modified  time.Time
	Size      int64
	SizeKnown bool
}

type Catalog interface {
	Apps(ctx context.Context) ([]App, error)
}

type AppSizer interface {
	Sizes(ctx context.Context, apps []App) error
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
	Apps      []App
	Tools     int
	Sized     bool
	TotalSize int64
}

type AppLister interface {
	ListApps(ctx context.Context, stmt *query.Statement) (AppReport, error)
}

type AppsExecutor struct {
	catalog  Catalog
	compiler *Compiler
	sizer    AppSizer
}

func NewAppsExecutor(catalog Catalog, compiler *Compiler, sizer AppSizer) *AppsExecutor {
	return &AppsExecutor{catalog: catalog, compiler: compiler, sizer: sizer}
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
		return e.withSizes(ctx, stmt, AppReport{Apps: found})
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
	return e.withSizes(ctx, stmt, AppReport{Apps: kept, Tools: tools})
}

func (e *AppsExecutor) withSizes(ctx context.Context, stmt *query.Statement, report AppReport) (AppReport, error) {
	if !stmt.WithSize || len(report.Apps) == 0 {
		return report, nil
	}
	if e.sizer == nil {
		return report, errNoSizer
	}

	if err := e.sizer.Sizes(ctx, report.Apps); err != nil {
		return AppReport{}, err
	}

	report.Sized = true
	for _, app := range report.Apps {
		if app.SizeKnown {
			report.TotalSize += app.Size
		}
	}
	return report, nil
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
