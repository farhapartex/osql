package shell

import (
	"fmt"
	"io"
	"os"

	"github.com/farhapartex/osql/internal/buildinfo"
)

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

func (s *Shell) Run() error {
	return s.writeBanner(s.cfg.Out)
}

func (s *Shell) writeBanner(w io.Writer) error {
	_, err := fmt.Fprintln(w, buildinfo.String(s.cfg.Version, s.cfg.Commit))
	return err
}
