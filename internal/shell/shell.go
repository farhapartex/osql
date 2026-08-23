package shell

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/farhapartex/osql/internal/buildinfo"
)

const Prompt = "osql > "

var ErrNoReader = errors.New("no line reader configured")

type Shell struct {
	cfg Config
}

func New(cfg Config) *Shell {
	if cfg.Out == nil {
		cfg.Out = os.Stdout
	}
	if cfg.Err == nil {
		cfg.Err = os.Stderr
	}
	return &Shell{cfg: cfg}
}

func (s *Shell) Greeting() string {
	return fmt.Sprintf("%s — Ctrl+D to exit.", buildinfo.String(s.cfg.Version, s.cfg.Commit))
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
		s.dispatch(line)
	}
}

func (s *Shell) dispatch(line string) {
	fmt.Fprintln(s.cfg.Out, line)
}
