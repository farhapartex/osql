package engine

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"strings"

	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/vfs"
)

const DepthUnlimited = 0

type ScanOptions struct {
	MaxDepth int
	Target   query.Target
	Matchers []Matcher
	Skip     SkipList
	OmitInfo bool
}

type Scanner struct {
	fsys vfs.FileSystem
}

func NewScanner(fsys vfs.FileSystem) *Scanner {
	return &Scanner{fsys: fsys}
}

func (s *Scanner) Scan(ctx context.Context, root Resolved, opts ScanOptions, sink RowSink) error {
	stopped := false

	err := fs.WalkDir(s.fsys, root.FSPath, func(current string, entry fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if walkErr != nil {
			if current == root.FSPath {
				return walkErr
			}
			if entry != nil && entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if current == root.FSPath {
			return nil
		}

		relative := relativeTo(root.FSPath, current)
		depth := strings.Count(relative, vfs.Separator) + 1
		isDir := entry.IsDir()

		if isDir && opts.Skip.Skips(current, entry.Name()) {
			return fs.SkipDir
		}

		if matchesTarget(opts.Target, isDir) {
			ok, err := MatchAll(opts.Matchers, Entry{DirEntry: entry, Path: current})
			if err != nil {
				return err
			}
			if ok {
				if err := pushRow(sink, relative, entry, opts.OmitInfo); err != nil {
					if errors.Is(err, ErrStopWalk) {
						stopped = true
						return fs.SkipAll
					}
					return err
				}
			}
		}

		if isDir && opts.MaxDepth != DepthUnlimited && depth >= opts.MaxDepth {
			return fs.SkipDir
		}
		return nil
	})

	if stopped {
		return nil
	}
	if err != nil {
		return classifyWalkError(root.Input, err)
	}
	return nil
}

func pushRow(sink RowSink, name string, entry fs.DirEntry, omitInfo bool) error {
	isDir := entry.IsDir()

	row := Row{Name: name, IsDir: isDir}
	if !isDir {
		row.Ext = ExtensionOf(entry.Name())
	}

	if !omitInfo {
		if info, err := entry.Info(); err == nil {
			row.Size = info.Size()
			row.Modified = info.ModTime()
		}
	}

	return sink.Push(row)
}

func matchesTarget(target query.Target, isDir bool) bool {
	switch target {
	case query.TargetFiles:
		return !isDir
	case query.TargetFolders:
		return isDir
	default:
		return true
	}
}

func relativeTo(root, current string) string {
	if root == vfs.RootPath {
		return current
	}
	if len(current) > len(root) && current[len(root)] == '/' {
		return current[len(root)+1:]
	}
	return current
}

func classifyWalkError(input string, err error) error {
	var oe *oerr.Error
	if errors.As(err, &oe) {
		return err
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	switch {
	case errors.Is(err, fs.ErrPermission):
		return oerr.NoPermission(input)
	case errors.Is(err, fs.ErrNotExist):
		return oerr.FolderMissing(input)
	default:
		return err
	}
}

func ExtensionOf(name string) string {
	return strings.TrimPrefix(path.Ext(name), ".")
}
