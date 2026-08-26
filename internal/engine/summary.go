package engine

import (
	"container/heap"
	"context"
	"slices"
	"time"

	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/vfs"
)

const (
	TopTypes   = 5
	TopLargest = 5
)

type TypeTally struct {
	Ext   string
	Count int64
	Size  int64
}

type Summary struct {
	Path       string
	Recursive  bool
	Files      int64
	Folders    int64
	TotalSize  int64
	Types      []TypeTally
	MoreTypes  int
	Largest    []Row
	Oldest     time.Time
	Newest     time.Time
	Skipped    []string
	SkipsShown bool
}

func (s Summary) IsEmpty() bool {
	return s.Files == 0 && s.Folders == 0
}

func (s Summary) HasFiles() bool {
	return s.Files > 0
}

type Summarizer interface {
	Executor
	Summarize(ctx context.Context, stmt *query.Statement) (Summary, error)
}

type SummaryExecutor struct {
	scanner  *Scanner
	resolver *PathResolver
	skip     SkipList
}

func NewSummaryExecutor(fsys vfs.FileSystem, resolver *PathResolver, skip SkipList) *SummaryExecutor {
	return &SummaryExecutor{scanner: NewScanner(fsys), resolver: resolver, skip: skip}
}

func (e *SummaryExecutor) Verb() string {
	return query.VerbSummary
}

func (e *SummaryExecutor) Execute(ctx context.Context, stmt *query.Statement, out RowSink) error {
	return errContentOnly
}

func (e *SummaryExecutor) Summarize(ctx context.Context, stmt *query.Statement) (Summary, error) {
	resolved, err := e.resolver.Resolve(stmt.Path)
	if err != nil {
		return Summary{}, err
	}

	depth := 1
	if stmt.Recursive {
		depth = DepthUnlimited
	}

	skip := e.skip
	if stmt.IncludeSkipped {
		skip = EmptySkipList()
	}

	collector := newSummaryCollector()
	opts := ScanOptions{
		MaxDepth: depth,
		Target:   query.TargetAll,
		Skip:     skip,
		OnSkip:   collector.recordSkip,
	}

	if err := e.scanner.Scan(ctx, resolved, opts, collector); err != nil {
		return Summary{}, err
	}

	summary := collector.finish()
	summary.Path = stmt.Path
	summary.Recursive = stmt.Recursive
	summary.SkipsShown = !stmt.IncludeSkipped
	return summary, nil
}

type summaryCollector struct {
	files     int64
	folders   int64
	totalSize int64
	byType    map[string]*TypeTally
	largest   *rowHeap
	oldest    time.Time
	newest    time.Time
	skipped   []string
	seenSkip  map[string]struct{}
}

func newSummaryCollector() *summaryCollector {
	return &summaryCollector{
		byType:   make(map[string]*TypeTally),
		largest:  &rowHeap{},
		seenSkip: make(map[string]struct{}),
	}
}

func (c *summaryCollector) recordSkip(name string) {
	if _, seen := c.seenSkip[name]; seen {
		return
	}
	c.seenSkip[name] = struct{}{}
	c.skipped = append(c.skipped, name)
}

func (c *summaryCollector) Push(row Row) error {
	if row.IsDir {
		c.folders++
		return nil
	}

	c.files++
	c.totalSize += row.Size

	tally, ok := c.byType[row.Ext]
	if !ok {
		tally = &TypeTally{Ext: row.Ext}
		c.byType[row.Ext] = tally
	}
	tally.Count++
	tally.Size += row.Size

	c.trackLargest(row)
	c.trackTimes(row.Modified)
	return nil
}

func (c *summaryCollector) trackLargest(row Row) {
	if c.largest.Len() < TopLargest {
		heap.Push(c.largest, row)
		return
	}
	if row.Size > (*c.largest)[0].Size {
		heap.Pop(c.largest)
		heap.Push(c.largest, row)
	}
}

func (c *summaryCollector) trackTimes(t time.Time) {
	if t.IsZero() {
		return
	}
	if c.oldest.IsZero() || t.Before(c.oldest) {
		c.oldest = t
	}
	if c.newest.IsZero() || t.After(c.newest) {
		c.newest = t
	}
}

func (c *summaryCollector) finish() Summary {
	types := make([]TypeTally, 0, len(c.byType))
	for _, t := range c.byType {
		types = append(types, *t)
	}
	slices.SortFunc(types, func(a, b TypeTally) int {
		if a.Size != b.Size {
			return int(b.Size - a.Size)
		}
		if a.Count != b.Count {
			return int(b.Count - a.Count)
		}
		if a.Ext < b.Ext {
			return -1
		}
		if a.Ext > b.Ext {
			return 1
		}
		return 0
	})

	more := 0
	if len(types) > TopTypes {
		more = len(types) - TopTypes
		types = types[:TopTypes]
	}

	largest := make([]Row, c.largest.Len())
	for i := len(largest) - 1; i >= 0; i-- {
		largest[i] = heap.Pop(c.largest).(Row)
	}

	return Summary{
		Files:     c.files,
		Folders:   c.folders,
		TotalSize: c.totalSize,
		Types:     types,
		MoreTypes: more,
		Largest:   largest,
		Oldest:    c.oldest,
		Newest:    c.newest,
		Skipped:   c.skipped,
	}
}

type rowHeap []Row

func (h rowHeap) Len() int           { return len(h) }
func (h rowHeap) Less(i, j int) bool { return h[i].Size < h[j].Size }
func (h rowHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *rowHeap) Push(x any)        { *h = append(*h, x.(Row)) }

func (h *rowHeap) Pop() any {
	old := *h
	n := len(old)
	last := old[n-1]
	*h = old[:n-1]
	return last
}
