package shell

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/farhapartex/osql/internal/buildinfo"
	"github.com/farhapartex/osql/internal/oerr"
)

const (
	Prompt    = "osql > "
	QueryVerb = "select"
)

var (
	ErrNoReader = errors.New("no line reader configured")
	errNoStore  = errors.New("no state store configured")
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
		fmt.Fprintln(s.cfg.Out, line)
		return nil
	}

	return oerr.UnknownVerb(fields[0], s.KnownVerbs())
}
