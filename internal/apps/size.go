package apps

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/farhapartex/osql/internal/engine"
)

const sizeWorkers = 4

type Sizer struct {
	workers int
}

func NewSizer() *Sizer {
	return &Sizer{workers: sizeWorkers}
}

func (s *Sizer) Sizes(ctx context.Context, list []engine.App) error {
	workers := s.workers
	if workers < 1 {
		workers = 1
	}
	if workers > len(list) {
		workers = len(list)
	}

	next := make(chan int)
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				size, ok := measure(ctx, list[i].Path)
				list[i].Size = size
				list[i].SizeKnown = ok
			}
		}()
	}

	for i := range list {
		select {
		case <-ctx.Done():
			close(next)
			wg.Wait()
			return ctx.Err()
		case next <- i:
		}
	}
	close(next)
	wg.Wait()

	return ctx.Err()
}

func measure(ctx context.Context, root string) (int64, bool) {
	if root == "" {
		return 0, false
	}

	info, err := os.Lstat(root)
	if err != nil {
		return 0, false
	}
	if !info.IsDir() {
		return info.Size(), true
	}

	var total int64
	counted := false
	checks := 0

	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if entry != nil && entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		checks++
		if checks%1024 == 0 {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
		}

		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}

		if stat, statErr := entry.Info(); statErr == nil {
			total += stat.Size()
			counted = true
		}
		return nil
	})
	if err != nil {
		return total, counted
	}
	return total, true
}
