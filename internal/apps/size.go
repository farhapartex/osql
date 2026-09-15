package apps

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/farhapartex/osql/internal/engine"
)

type Sizer struct{}

func NewSizer() *Sizer {
	return &Sizer{}
}

func (s *Sizer) Sizes(ctx context.Context, list []engine.App) error {
	return engine.InParallel(ctx, len(list), func(i int) {
		size, ok := measure(ctx, list[i].Path)
		list[i].Size = size
		list[i].SizeKnown = ok
	})
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
