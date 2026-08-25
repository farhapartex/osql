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
)

const (
	Prompt    = "osql > "
	QueryVerb = "select"
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
		cfg.Renderer = output.NewLines()
	}
	return &Shell{cfg: cfg, builtins: DefaultBuiltins()}
}

func (s *Shell) Builtins() *BuiltinRegistry {
	return s.builtins
}

func (s *Shell) Greeting() string {
	return fmt.Sprintf("%s — type \"help\" for commands, \"exit\" to quit.", buildinfo.String(s.cfg.Version, s.cfg.Commit))
}

func (s *Shell) KnownVerbs() []string {
	return append([]string{QueryVerb}, s.builtins.Names()...)
}

func (s *Shell) Run() error {
	if s.cfg.Reader == nil {
		return ErrNoReader
	}

	fmt.Fprintln(s.cfg.Out, s.Greeting())

	for {
		line, err := s.cfg.Reader.ReadLine(Prompt)
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

	executor, ok := s.cfg.Engine.Lookup(stmt.Verb)
	if !ok {
		return oerr.UnknownVerb(stmt.Verb, s.KnownVerbs())
	}

	sink := &engine.SliceSink{}
	if err := executor.Execute(context.Background(), stmt, sink); err != nil {
		return err
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

func (s *Shell) Dispatch(line string) error {
	line = strings.TrimSuffix(strings.TrimSpace(line), ";")

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}

	name := strings.ToLower(fields[0])

	if b, ok := s.builtins.Lookup(name); ok {
		return b.Run(s, fields[1:])
	}
	if name == QueryVerb {
		return s.runQuery(line)
	}

	return oerr.UnknownVerb(fields[0], s.KnownVerbs())
}
