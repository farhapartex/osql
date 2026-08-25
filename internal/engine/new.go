package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/vfs"
)

type NewExecutor struct {
	fsys     vfs.WritableFileSystem
	resolver *PathResolver
}

func NewNewExecutor(fsys vfs.WritableFileSystem, resolver *PathResolver) *NewExecutor {
	return &NewExecutor{fsys: fsys, resolver: resolver}
}

func (e *NewExecutor) Verb() string {
	return query.VerbNew
}

func (e *NewExecutor) Execute(ctx context.Context, stmt *query.Statement, out RowSink) error {
	return errContentOnly
}

func (e *NewExecutor) WriteContent(ctx context.Context, stmt *query.Statement, w io.Writer) error {
	target, err := e.resolver.ResolveNew(stmt.Path)
	if err != nil {
		return err
	}

	if _, err := e.fsys.Stat(target.FSPath); err == nil {
		return oerr.AlreadyExists(stmt.Path)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return classifyStatError(stmt.Path, err)
	}

	made, err := e.ensureParents(target.FSPath)
	if err != nil {
		return oerr.CannotCreate(stmt.Path, reasonFor(err))
	}

	if stmt.Kind == query.NewFolder {
		if err := e.fsys.MkdirAll(target.FSPath); err != nil {
			return oerr.CannotCreate(stmt.Path, reasonFor(err))
		}
	} else if err := e.writeFile(target.FSPath, stmt.Data); err != nil {
		return oerr.CannotCreate(stmt.Path, reasonFor(err))
	}

	return report(w, stmt.Path, made)
}

func (e *NewExecutor) ensureParents(fsPath string) ([]string, error) {
	parent := path.Dir(fsPath)
	if parent == vfs.RootPath || parent == "" {
		return nil, nil
	}

	missing := []string{}
	for current := parent; current != vfs.RootPath && current != ""; current = path.Dir(current) {
		if _, err := e.fsys.Stat(current); err == nil {
			break
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		missing = append([]string{current}, missing...)
	}

	if len(missing) == 0 {
		return nil, nil
	}
	if err := e.fsys.MkdirAll(parent); err != nil {
		return nil, err
	}
	return missing, nil
}

func (e *NewExecutor) writeFile(fsPath, data string) error {
	file, err := e.fsys.Create(fsPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if data == "" {
		return nil
	}
	_, err = io.WriteString(file, data)
	return err
}

func report(w io.Writer, input string, alsoMade []string) error {
	if _, err := fmt.Fprintf(w, "Created '%s'\n", input); err != nil {
		return err
	}
	if len(alsoMade) == 0 {
		return nil
	}
	_, err := fmt.Fprintf(w, "  also created: %s\n", strings.Join(alsoMade, ", "))
	return err
}

func reasonFor(err error) string {
	switch {
	case errors.Is(err, fs.ErrPermission):
		return "permission denied"
	case errors.Is(err, fs.ErrExist):
		return "something is already there"
	default:
		return err.Error()
	}
}
