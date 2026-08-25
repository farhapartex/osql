package engine

import (
	"context"

	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/vfs"
)

type SelectExecutor struct {
	scanner  *Scanner
	resolver *PathResolver
	compiler *Compiler
	skip     SkipList
}

func NewSelectExecutor(fsys vfs.FileSystem, resolver *PathResolver, compiler *Compiler, skip SkipList) *SelectExecutor {
	return &SelectExecutor{
		scanner:  NewScanner(fsys),
		resolver: resolver,
		compiler: compiler,
		skip:     skip,
	}
}

func (e *SelectExecutor) Verb() string {
	return query.VerbSelect
}

func (e *SelectExecutor) Execute(ctx context.Context, stmt *query.Statement, out RowSink) error {
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

	return e.scanner.Scan(ctx, resolved, ScanOptions{
		MaxDepth: depth,
		Target:   stmt.Target,
		Matchers: matchers,
		Skip:     e.skip,
	}, out)
}

type SliceSink struct {
	Rows  []Row
	Limit int
}

func (s *SliceSink) Push(r Row) error {
	if s.Limit > 0 && len(s.Rows) >= s.Limit {
		return ErrStopWalk
	}
	s.Rows = append(s.Rows, r)
	return nil
}
