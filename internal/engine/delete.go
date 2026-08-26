package engine

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/vfs"
)

type Victim struct {
	Name    string
	FSPath  string
	IsDir   bool
	Size    int64
	Files   int64
	Folders int64
}

type DeletePlan struct {
	Path      string
	Permanent bool
	Recursive bool
	Victims   []Victim
	TotalSize int64
	TotalFile int64
	TotalDir  int64
}

func (p DeletePlan) IsEmpty() bool {
	return len(p.Victims) == 0
}

type DeleteOutcome struct {
	Name   string
	Reason string
}

type DeleteResult struct {
	Deleted   []string
	Failed    []DeleteOutcome
	Permanent bool
}

type Deleter interface {
	Executor
	Plan(ctx context.Context, stmt *query.Statement) (DeletePlan, error)
	Commit(ctx context.Context, plan DeletePlan) (DeleteResult, error)
}

type DeleteExecutor struct {
	fsys     vfs.WritableFileSystem
	scanner  *Scanner
	resolver *PathResolver
	compiler *Compiler
	trash    *vfs.Trash
}

func NewDeleteExecutor(fsys vfs.WritableFileSystem, resolver *PathResolver, compiler *Compiler, trash *vfs.Trash) *DeleteExecutor {
	return &DeleteExecutor{
		fsys:     fsys,
		scanner:  NewScanner(fsys),
		resolver: resolver,
		compiler: compiler,
		trash:    trash,
	}
}

func (e *DeleteExecutor) Verb() string {
	return query.VerbDelete
}

func (e *DeleteExecutor) Execute(ctx context.Context, stmt *query.Statement, out RowSink) error {
	return errContentOnly
}

func (e *DeleteExecutor) Plan(ctx context.Context, stmt *query.Statement) (DeletePlan, error) {
	if stmt.Single {
		return e.planSingle(stmt)
	}
	return e.planBulk(ctx, stmt)
}

func (e *DeleteExecutor) planSingle(stmt *query.Statement) (DeletePlan, error) {
	resolved, err := e.resolver.Resolve(stmt.Path)
	if err == nil {
		if stmt.Kind == query.NewFile {
			return DeletePlan{}, oerr.DeleteKindMismatch(stmt.Path, "folder")
		}
		if err := e.refuseProtected(resolved.Absolute); err != nil {
			return DeletePlan{}, err
		}
		return e.planFor(stmt, []Victim{e.weigh(resolved.FSPath, path.Base(resolved.FSPath), true)}), nil
	}

	if oerr.Is(err, oerr.KindPathIsFile) {
		if stmt.Kind == query.NewFolder {
			return DeletePlan{}, oerr.DeleteKindMismatch(stmt.Path, "file")
		}
		file, ferr := e.resolver.ResolveFile(stmt.Path)
		if ferr != nil {
			return DeletePlan{}, ferr
		}
		return e.planFor(stmt, []Victim{e.weigh(file.FSPath, path.Base(file.FSPath), false)}), nil
	}

	if stmt.Kind == query.NewFile && oerr.Is(err, oerr.KindFolderMissing) {
		return DeletePlan{}, oerr.FileMissing(stmt.Path)
	}
	return DeletePlan{}, err
}

func isTopLevel(absolute, home string) bool {
	return absolute == vfs.Separator || absolute == home
}

func (e *DeleteExecutor) refuseProtected(absolute string) error {
	if isTopLevel(absolute, e.resolver.Home()) {
		return oerr.RefuseDeleteRoot(e.resolver.Display(absolute))
	}
	if absolute == e.resolver.Dir() {
		return oerr.RefuseDeleteHere(e.resolver.Display(absolute))
	}
	return nil
}

func (e *DeleteExecutor) planBulk(ctx context.Context, stmt *query.Statement) (DeletePlan, error) {
	resolved, err := e.resolver.Resolve(stmt.Path)
	if err != nil {
		return DeletePlan{}, err
	}
	if len(stmt.Predicates) == 0 && isTopLevel(resolved.Absolute, e.resolver.Home()) {
		return DeletePlan{}, oerr.RefuseDeleteRoot(e.resolver.Display(resolved.Absolute))
	}

	matchers, err := e.compiler.CompileAll(stmt.Predicates, stmt.Target)
	if err != nil {
		return DeletePlan{}, err
	}

	depth := 1
	if stmt.Recursive {
		depth = DepthUnlimited
	}

	sink := &SliceSink{}
	err = e.scanner.Scan(ctx, resolved, ScanOptions{
		MaxDepth: depth,
		Target:   stmt.Target,
		Matchers: matchers,
		Skip:     EmptySkipList(),
		OmitInfo: true,
	}, sink)
	if err != nil {
		return DeletePlan{}, err
	}

	paths := make([]string, 0, len(sink.Rows))
	dirs := make(map[string]bool, len(sink.Rows))
	for _, row := range sink.Rows {
		full := vfs.Join(resolved.FSPath, row.Name)
		paths = append(paths, full)
		dirs[full] = row.IsDir
	}

	victims := make([]Victim, 0, len(paths))
	for _, p := range outermost(paths) {
		victims = append(victims, e.weigh(p, relativeTo(resolved.FSPath, p), dirs[p]))
	}

	return e.planFor(stmt, victims), nil
}

func (e *DeleteExecutor) planFor(stmt *query.Statement, victims []Victim) DeletePlan {
	plan := DeletePlan{
		Path:      stmt.Path,
		Permanent: stmt.Permanent,
		Recursive: stmt.Recursive,
		Victims:   victims,
	}
	for _, v := range victims {
		plan.TotalSize += v.Size
		plan.TotalFile += v.Files
		plan.TotalDir += v.Folders
	}
	return plan
}

func (e *DeleteExecutor) weigh(fsPath, name string, isDir bool) Victim {
	victim := Victim{Name: name, FSPath: fsPath, IsDir: isDir}

	if !isDir {
		victim.Files = 1
		if info, err := e.fsys.Stat(fsPath); err == nil {
			victim.Size = info.Size()
		}
		return victim
	}

	victim.Folders = 1
	fs.WalkDir(e.fsys, fsPath, func(current string, entry fs.DirEntry, err error) error {
		if err != nil || current == fsPath {
			return nil
		}
		if entry.IsDir() {
			victim.Folders++
			return nil
		}
		victim.Files++
		if info, err := entry.Info(); err == nil {
			victim.Size += info.Size()
		}
		return nil
	})
	return victim
}

func (e *DeleteExecutor) Commit(ctx context.Context, plan DeletePlan) (DeleteResult, error) {
	result := DeleteResult{Permanent: plan.Permanent}

	for _, victim := range plan.Victims {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		if err := e.remove(victim, plan.Permanent); err != nil {
			result.Failed = append(result.Failed, DeleteOutcome{Name: victim.Name, Reason: reasonFor(err)})
			continue
		}
		result.Deleted = append(result.Deleted, victim.Name)
	}

	return result, nil
}

func (e *DeleteExecutor) remove(victim Victim, permanent bool) error {
	if permanent {
		return e.fsys.Remove(victim.FSPath)
	}
	if e.trash == nil {
		return errNoTrash
	}

	absolute := vfs.OSPath(vfs.Separator, victim.FSPath)
	if _, err := e.trash.Move(absolute); err != nil {
		if errors.Is(err, vfs.ErrCrossDevice) {
			return errCrossDevice
		}
		return err
	}
	return nil
}

func outermost(paths []string) []string {
	sorted := slices.Clone(paths)
	slices.Sort(sorted)

	kept := make([]string, 0, len(sorted))
	for _, p := range sorted {
		if len(kept) > 0 && isUnder(kept[len(kept)-1], p) {
			continue
		}
		kept = append(kept, p)
	}
	return kept
}

func isUnder(parent, child string) bool {
	return strings.HasPrefix(child, parent+vfs.Separator)
}

var (
	errNoTrash     = errors.New("no trash configured")
	errCrossDevice = errors.New("on another disk")
)
