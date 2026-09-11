package shell

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/farhapartex/osql/internal/buildinfo"
	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/output"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/textwidth"
)

const (
	Prompt          = "osql > "
	PromptName      = "osql"
	PromptTail      = " > "
	promptPathWidth = 40
)

var (
	ErrNoReader   = errors.New("no line reader configured")
	errNoStore    = errors.New("no state store configured")
	errNoPipeline = errors.New("no query pipeline configured")
)

type Shell struct {
	cfg      Config
	builtins *BuiltinRegistry

	mu          sync.Mutex
	stopRunning context.CancelFunc
}

func New(cfg Config) *Shell {
	if cfg.Out == nil {
		cfg.Out = os.Stdout
	}
	if cfg.Err == nil {
		cfg.Err = os.Stderr
	}
	if cfg.Renderer == nil {
		cfg.Renderer = output.NewTable()
	}
	if cfg.CountRenderer == nil {
		cfg.CountRenderer = output.NewCount()
	}
	return &Shell{cfg: cfg, builtins: DefaultBuiltins()}
}

func (s *Shell) Builtins() *BuiltinRegistry {
	return s.builtins
}

func (s *Shell) Prompt() string {
	if s.cfg.Resolver == nil {
		return Prompt
	}
	where := s.cfg.Resolver.Display(s.cfg.Resolver.Dir())
	return PromptName + " " + textwidth.TruncateMiddle(where, promptPathWidth) + PromptTail
}

func (s *Shell) Greeting() string {
	return fmt.Sprintf("%s — type \"help\" for commands, \"exit\" to quit.", buildinfo.String(s.cfg.Version, s.cfg.Commit))
}

func (s *Shell) Editing() bool {
	return s.cfg.Editing
}

func (s *Shell) KnownWords() []string {
	return append(query.TargetNames(), s.builtins.Names()...)
}

func (s *Shell) Run() error {
	if s.cfg.Reader == nil {
		return ErrNoReader
	}

	fmt.Fprintln(s.cfg.Out, s.Greeting())

	for {
		line, err := s.cfg.Reader.ReadLine(s.Prompt())
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Fprintln(s.cfg.Out)
				return nil
			}
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		s.cfg.Reader.AddHistory(line)

		if err := s.Dispatch(line); err != nil {
			if errors.Is(err, ErrExit) {
				return nil
			}
			fmt.Fprintln(s.cfg.Err, err)
		}
	}
}

func (s *Shell) beginQuery() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.stopRunning = cancel
	s.mu.Unlock()
	return ctx
}

func (s *Shell) endQuery() {
	s.mu.Lock()
	stop := s.stopRunning
	s.stopRunning = nil
	s.mu.Unlock()
	if stop != nil {
		stop()
	}
}

func (s *Shell) Interrupt() bool {
	s.mu.Lock()
	stop := s.stopRunning
	s.mu.Unlock()
	if stop == nil {
		return false
	}
	stop()
	return true
}

func stoppedEarly(err error, found, scanned int) error {
	if errors.Is(err, context.Canceled) {
		return oerr.QueryStopped(found, scanned)
	}
	return err
}

func (s *Shell) runQuery(line string) error {
	if s.cfg.Lexer == nil || s.cfg.Parser == nil || s.cfg.Engine == nil {
		return errNoPipeline
	}

	tokens, err := s.cfg.Lexer.Lex(line)
	if err != nil {
		return err
	}

	stmt, err := s.cfg.Parser.Parse(tokens)
	if err != nil {
		return err
	}

	verb := stmt.Verb
	if stmt.Target == query.TargetApps {
		verb = query.VerbApps
	}

	executor, ok := s.cfg.Engine.Lookup(verb)
	if !ok {
		return oerr.UnknownVerb(stmt.Verb, s.KnownWords())
	}

	ctx := s.beginQuery()
	defer s.endQuery()

	progress := &scanProgress{}
	if s.cfg.Editing {
		progress.out = s.cfg.Err
	}
	ctx = engine.WithProgress(ctx, progress.report)
	defer progress.erase()

	if deleter, ok := executor.(engine.Deleter); ok {
		return s.runDelete(ctx, deleter, stmt, progress)
	}

	if summarizer, ok := executor.(engine.AppSummarizer); ok && stmt.Verb == query.VerbSummary {
		summary, err := summarizer.SummarizeApps(ctx, stmt)
		if err != nil {
			return stoppedEarly(err, 0, progress.scanned)
		}
		return s.cfg.AppSummary.Render(s.cfg.Out, summary)
	}

	if lister, ok := executor.(engine.AppLister); ok && stmt.Verb != query.VerbCount {
		report, err := lister.ListApps(ctx, stmt)
		if err != nil {
			return stoppedEarly(err, 0, progress.scanned)
		}
		if len(report.Apps) == 0 {
			if len(stmt.Predicates) == 0 {
				fmt.Fprintln(s.cfg.Out, oerr.NoApps())
			} else {
				fmt.Fprintln(s.cfg.Out, oerr.NoAppsMatched())
			}
			return nil
		}
		return s.cfg.Apps.Render(s.cfg.Out, report)
	}

	if summarizer, ok := executor.(engine.Summarizer); ok {
		summary, err := summarizer.Summarize(ctx, stmt)
		if err != nil {
			return stoppedEarly(err, 0, progress.scanned)
		}
		return s.cfg.Summary.Render(s.cfg.Out, summary)
	}

	if content, ok := executor.(engine.ContentExecutor); ok {
		return stoppedEarly(content.WriteContent(ctx, stmt, s.cfg.Out), 0, progress.scanned)
	}

	sink := &engine.SliceSink{}
	var out engine.RowSink = sink
	var limited *engine.LimitSink
	var sorted *engine.SortSink

	if stmt.SortField != "" {
		field, ok := s.sortableField(stmt.SortField)
		if !ok {
			return oerr.UnsortableField(stmt.SortField, nil)
		}
		sorted = engine.NewSortSink(field, stmt.SortDesc, stmt.Limit)
	}

	switch {
	case stmt.WithSize:
		out = sink
	case sorted != nil:
		out = sorted
	case stmt.Limit > 0:
		limited = engine.NewLimitSink(sink, stmt.Limit)
		out = limited
	}

	if err := executor.Execute(ctx, stmt, out); err != nil {
		return stoppedEarly(err, len(sink.Rows), progress.scanned)
	}

	rows := sink.Rows
	trimmed := false

	if stmt.WithSize {
		if err := s.measureFolders(ctx, stmt, rows); err != nil {
			return stoppedEarly(err, len(rows), progress.scanned)
		}
		switch {
		case sorted != nil:
			for _, row := range rows {
				if err := sorted.Push(row); err != nil {
					return err
				}
			}
			rows = sorted.Rows()
		case stmt.Limit > 0 && len(rows) > stmt.Limit:
			rows = rows[:stmt.Limit]
			trimmed = true
		}
	} else if sorted != nil {
		rows = sorted.Rows()
	}

	if stmt.Verb == query.VerbCount {
		return s.cfg.CountRenderer.Render(s.cfg.Out, rows)
	}

	if len(rows) == 0 {
		if len(stmt.Predicates) == 0 {
			fmt.Fprintln(s.cfg.Out, oerr.EmptyFolder(stmt.Path))
		} else {
			fmt.Fprintln(s.cfg.Out, oerr.NoMatches())
		}
		return nil
	}

	if err := s.cfg.Renderer.Render(s.cfg.Out, rows); err != nil {
		return err
	}
	if trimmed || (limited != nil && limited.Filled()) {
		fmt.Fprintln(s.cfg.Out, oerr.LimitReached(stmt.Limit))
	}
	if sorted != nil && stmt.Limit == 0 && sorted.Overflowed() {
		fmt.Fprintln(s.cfg.Out, oerr.SortTruncated(engine.SortCap))
	}
	return nil
}

func (s *Shell) measureFolders(ctx context.Context, stmt *query.Statement, rows []engine.Row) error {
	if s.cfg.FolderSizes == nil || s.cfg.Resolver == nil {
		return nil
	}

	root, err := s.cfg.Resolver.Resolve(stmt.Path)
	if err != nil {
		return err
	}
	return s.cfg.FolderSizes.Measure(ctx, rows, func(row engine.Row) string {
		return path.Join(root.FSPath, row.Name)
	})
}

func (s *Shell) sortableField(name string) (engine.SortableField, bool) {
	if s.cfg.Fields == nil {
		return nil, false
	}
	field, ok := s.cfg.Fields.Lookup(name)
	if !ok {
		return nil, false
	}
	sortable, ok := field.(engine.SortableField)
	return sortable, ok
}

func (s *Shell) runDelete(ctx context.Context, deleter engine.Deleter, stmt *query.Statement, progress *scanProgress) error {
	plan, err := deleter.Plan(ctx, stmt)
	if err != nil {
		return stoppedEarly(err, 0, progress.scanned)
	}
	if plan.IsEmpty() {
		return s.cfg.Delete.Nothing(s.cfg.Out)
	}

	if err := s.cfg.Delete.Preview(s.cfg.Out, plan); err != nil {
		return err
	}

	answer, err := s.cfg.Reader.ReadLine(output.ConfirmPrompt)
	if err != nil || strings.TrimSpace(answer) != output.ConfirmWord {
		return s.cfg.Delete.Cancelled(s.cfg.Out)
	}

	if ctx.Err() != nil {
		return s.cfg.Delete.Cancelled(s.cfg.Out)
	}

	result, err := deleter.Commit(context.WithoutCancel(ctx), plan)
	if err != nil {
		return err
	}
	return s.cfg.Delete.Result(s.cfg.Out, result)
}

func (s *Shell) Dispatch(line string) error {
	line = strings.TrimSuffix(strings.TrimSpace(line), ";")

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}

	name := strings.ToLower(fields[0])

	if b, ok := s.builtins.Lookup(name); ok {
		args := SplitArgs(line)
		return b.Run(s, args[1:])
	}
	return s.runQuery(line)
}
