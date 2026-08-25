package engine

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"

	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/vfs"
)

const binarySniffSize = 8192

type ContentExecutor interface {
	Executor
	WriteContent(ctx context.Context, stmt *query.Statement, w io.Writer) error
}

type OpenExecutor struct {
	fsys     vfs.FileSystem
	resolver *PathResolver
}

func NewOpenExecutor(fsys vfs.FileSystem, resolver *PathResolver) *OpenExecutor {
	return &OpenExecutor{fsys: fsys, resolver: resolver}
}

func (e *OpenExecutor) Verb() string {
	return query.VerbOpen
}

func (e *OpenExecutor) Execute(ctx context.Context, stmt *query.Statement, out RowSink) error {
	return errContentOnly
}

func (e *OpenExecutor) WriteContent(ctx context.Context, stmt *query.Statement, w io.Writer) error {
	resolved, err := e.resolver.ResolveFile(stmt.Path)
	if err != nil {
		return err
	}

	file, err := e.fsys.Open(resolved.FSPath)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return oerr.NoPermission(stmt.Path)
		}
		return oerr.FileMissing(stmt.Path)
	}
	defer file.Close()

	head := make([]byte, binarySniffSize)
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return classifyReadError(stmt.Path, err)
	}
	head = head[:n]

	if bytes.IndexByte(head, 0) >= 0 {
		return oerr.BinaryFile(stmt.Path)
	}

	lastByte, err := writeChunk(w, head, 0)
	if err != nil {
		return err
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	buf := make([]byte, 32*1024)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		read, readErr := file.Read(buf)
		if read > 0 {
			if bytes.IndexByte(buf[:read], 0) >= 0 {
				return oerr.BinaryFile(stmt.Path)
			}
			lastByte, err = writeChunk(w, buf[:read], lastByte)
			if err != nil {
				return err
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return classifyReadError(stmt.Path, readErr)
		}
	}

	if lastByte != 0 && lastByte != '\n' {
		if _, err := w.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
	return nil
}

func writeChunk(w io.Writer, chunk []byte, lastByte byte) (byte, error) {
	if len(chunk) == 0 {
		return lastByte, nil
	}
	if _, err := w.Write(chunk); err != nil {
		return lastByte, err
	}
	return chunk[len(chunk)-1], nil
}

func classifyReadError(input string, err error) error {
	if errors.Is(err, fs.ErrPermission) {
		return oerr.NoPermission(input)
	}
	return oerr.CannotRead(input)
}

var errContentOnly = errors.New("open streams content and has no rows")
