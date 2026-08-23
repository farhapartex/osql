package state

import (
	"os"
	"path/filepath"
	"time"
)

const (
	DirName     = ".osql"
	SystemFile  = "system.txt"
	HistoryFile = "history.txt"

	dirMode     os.FileMode = 0o700
	systemMode  os.FileMode = 0o644
	historyMode os.FileMode = 0o600
)

type Options struct {
	Root         string
	Version      string
	Commit       string
	HistoryLimit int
	NoHistory    bool
	Now          func() time.Time
}

type DirStore struct {
	opts Options
	hist History
}

func New(opts Options) *DirStore {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.HistoryLimit <= 0 {
		opts.HistoryLimit = DefaultHistoryLimit
	}
	return &DirStore{opts: opts}
}

func DefaultRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DirName), nil
}

func (s *DirStore) Root() string { return s.opts.Root }

func (s *DirStore) SystemPath() string {
	return filepath.Join(s.opts.Root, SystemFile)
}

func (s *DirStore) HistoryPath() string {
	return filepath.Join(s.opts.Root, HistoryFile)
}

func (s *DirStore) Ensure() error {
	if err := os.MkdirAll(s.opts.Root, dirMode); err != nil {
		return err
	}
	if err := os.Chmod(s.opts.Root, dirMode); err != nil {
		return err
	}
	return s.WriteSystemInfo(false)
}

func (s *DirStore) WriteSystemInfo(force bool) error {
	path := s.SystemPath()
	if !force {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	info := CollectSystemInfo(s.opts.Version, s.opts.Commit, s.opts.Now())
	if err := os.WriteFile(path, []byte(FormatSystemInfo(info)), systemMode); err != nil {
		return err
	}
	return os.Chmod(path, systemMode)
}

func (s *DirStore) History() (History, error) {
	if s.hist != nil {
		return s.hist, nil
	}
	if s.opts.NoHistory {
		s.hist = nopHistory{}
		return s.hist, nil
	}

	hist, err := openHistory(s.HistoryPath(), s.opts.HistoryLimit)
	if err != nil {
		return nil, err
	}
	s.hist = hist
	return s.hist, nil
}

func (s *DirStore) Close() error {
	if s.hist == nil {
		return nil
	}
	err := s.hist.Close()
	s.hist = nil
	return err
}
