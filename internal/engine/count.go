package engine

import (
	"context"

	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/vfs"
)

type CountExecutor struct {
	scanner  *Scanner
	resolver *PathResolver
	compiler *Compiler
	skip     SkipList
}

func NewCountExecutor(fsys vfs.FileSystem, resolver *PathResolver, compiler *Compiler, skip SkipList) *CountExecutor {
	return &CountExecutor{
		scanner:  NewScanner(fsys),
		resolver: resolver,
		compiler: compiler,
		skip:     skip,
	}
}

func (e *CountExecutor) Verb() string {
	return query.VerbCount
}

func (e *CountExecutor) Execute(ctx context.Context, stmt *query.Statement, out RowSink) error {
	resolved, err := e.resolver.Resolve(stmt.Path)
	if err != nil {
		return err
	}

	matchers, err := e.compiler.CompileAll(stmt.Predicates, stmt.Target)
	if err != nil {
		return err
	}

	depth := 1
	if stmt.Recursive {
		depth = DepthUnlimited
	}

	tally := &countingSink{}
	err = e.scanner.Scan(ctx, resolved, ScanOptions{
		MaxDepth: depth,
		Target:   stmt.Target,
		Matchers: matchers,
		Skip:     e.skip,
		OmitInfo: true,
	}, tally)
	if err != nil {
		return err
	}

	return out.Push(Row{Name: stmt.Target.String(), Count: tally.total})
}

type countingSink struct {
	total int64
}

func (c *countingSink) Push(Row) error {
	c.total++
	return nil
}
