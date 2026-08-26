package shell

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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

	ctx := context.Background()

	if deleter, ok := executor.(engine.Deleter); ok {
		return s.runDelete(ctx, deleter, stmt)
	}

	if summarizer, ok := executor.(engine.AppSummarizer); ok && stmt.Verb == query.VerbSummary {
		summary, err := summarizer.SummarizeApps(ctx, stmt)
		if err != nil {
			return err
		}
		return s.cfg.AppSummary.Render(s.cfg.Out, summary)
	}

	if lister, ok := executor.(engine.AppLister); ok && stmt.Verb != query.VerbCount {
		report, err := lister.ListApps(ctx, stmt)
		if err != nil {
			return err
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
			return err
		}
		return s.cfg.Summary.Render(s.cfg.Out, summary)
	}

	if content, ok := executor.(engine.ContentExecutor); ok {
		return content.WriteContent(ctx, stmt, s.cfg.Out)
	}

	sink := &engine.SliceSink{}
	if err := executor.Execute(ctx, stmt, sink); err != nil {
		return err
	}

	if stmt.Verb == query.VerbCount {
		return s.cfg.CountRenderer.Render(s.cfg.Out, sink.Rows)
	}

	if len(sink.Rows) == 0 {
		if len(stmt.Predicates) == 0 {
			fmt.Fprintln(s.cfg.Out, oerr.EmptyFolder(stmt.Path))
		} else {
			fmt.Fprintln(s.cfg.Out, oerr.NoMatches())
		}
		return nil
	}

	return s.cfg.Renderer.Render(s.cfg.Out, sink.Rows)
}

func (s *Shell) runDelete(ctx context.Context, deleter engine.Deleter, stmt *query.Statement) error {
	plan, err := deleter.Plan(ctx, stmt)
	if err != nil {
		return err
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

	result, err := deleter.Commit(ctx, plan)
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
