package engine

import (
	"context"
	"io/fs"

	"github.com/farhapartex/osql/internal/vfs"
)

const sizeCheckInterval = 1024

type FolderSizer struct {
	fsys vfs.FileSystem
}

func NewFolderSizer(fsys vfs.FileSystem) *FolderSizer {
	return &FolderSizer{fsys: fsys}
}

func (s *FolderSizer) Measure(ctx context.Context, rows []Row, pathOf func(Row) string) error {
	return InParallel(ctx, len(rows), func(i int) {
		if !rows[i].IsDir {
			rows[i].SizeKnown = true
			return
		}
		total, ok := s.totalUnder(ctx, pathOf(rows[i]))
		rows[i].Size = total
		rows[i].SizeKnown = ok
	})
}

func (s *FolderSizer) totalUnder(ctx context.Context, root string) (int64, bool) {
	if root == "" {
		return 0, false
	}

	var total int64
	seen := 0
	readable := true

	err := fs.WalkDir(s.fsys, root, func(_ string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			readable = false
			if entry != nil && entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		seen++
		if seen%sizeCheckInterval == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}

		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			readable = false
			return nil
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return total, false
	}
	return total, readable
}
