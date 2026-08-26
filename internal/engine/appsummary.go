package engine

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/farhapartex/osql/internal/query"
)

const TopLargestApps = 5

type SourceTally struct {
	Source string
	Count  int64
	Size   int64
}

type AppSummary struct {
	Apps       int64
	AppsSize   int64
	Tools      int64
	ToolsSize  int64
	Sources    []SourceTally
	Largest    []App
	Oldest     time.Time
	Newest     time.Time
	Unmeasured int
}

func (s AppSummary) IsEmpty() bool {
	return s.Apps == 0 && s.Tools == 0
}

func (s AppSummary) Total() int64 {
	return s.Apps + s.Tools
}

func (s AppSummary) TotalSize() int64 {
	return s.AppsSize + s.ToolsSize
}

type AppSummarizer interface {
	Executor
	SummarizeApps(ctx context.Context, stmt *query.Statement) (AppSummary, error)
}

func (e *AppsExecutor) SummarizeApps(ctx context.Context, stmt *query.Statement) (AppSummary, error) {
	if e.catalog == nil {
		return AppSummary{}, errNoCatalog
	}

	found, err := e.catalog.Apps(ctx)
	if err != nil {
		return AppSummary{}, err
	}
	if len(found) == 0 {
		return AppSummary{}, nil
	}

	if e.sizer == nil {
		return AppSummary{}, errNoSizer
	}
	if err := e.sizer.Sizes(ctx, found); err != nil {
		return AppSummary{}, err
	}

	return tallyApps(found), nil
}

func tallyApps(found []App) AppSummary {
	summary := AppSummary{}
	bySource := make(map[string]*SourceTally, len(sourceAliases))

	for _, app := range found {
		if !app.SizeKnown {
			summary.Unmeasured++
		}

		if app.Source == SourceHomebrewCLI {
			summary.Tools++
			summary.ToolsSize += app.Size
		} else {
			summary.Apps++
			summary.AppsSize += app.Size
		}

		tally, ok := bySource[app.Source]
		if !ok {
			tally = &SourceTally{Source: app.Source}
			bySource[app.Source] = tally
		}
		tally.Count++
		tally.Size += app.Size

		if !app.Modified.IsZero() {
			if summary.Oldest.IsZero() || app.Modified.Before(summary.Oldest) {
				summary.Oldest = app.Modified
			}
			if summary.Newest.IsZero() || app.Modified.After(summary.Newest) {
				summary.Newest = app.Modified
			}
		}
	}

	summary.Sources = make([]SourceTally, 0, len(bySource))
	for _, tally := range bySource {
		summary.Sources = append(summary.Sources, *tally)
	}
	slices.SortFunc(summary.Sources, func(a, b SourceTally) int {
		if a.Size != b.Size {
			return int(b.Size - a.Size)
		}
		return strings.Compare(a.Source, b.Source)
	})

	summary.Largest = largestApps(found)
	return summary
}

func largestApps(found []App) []App {
	sized := make([]App, 0, len(found))
	for _, app := range found {
		if app.SizeKnown && app.Size > 0 {
			sized = append(sized, app)
		}
	}

	slices.SortFunc(sized, func(a, b App) int {
		if a.Size != b.Size {
			if b.Size > a.Size {
				return 1
			}
			return -1
		}
		return strings.Compare(a.Name, b.Name)
	})

	if len(sized) > TopLargestApps {
		sized = sized[:TopLargestApps]
	}
	return sized
}
