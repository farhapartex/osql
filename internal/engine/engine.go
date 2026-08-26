package engine

import (
	"context"
	"errors"
	"io/fs"
	"time"

	"github.com/farhapartex/osql/internal/query"
)

var ErrStopWalk = errors.New("stop walk")

type Row struct {
	Name     string
	Ext      string
	Size     int64
	Modified time.Time
	IsDir    bool
	Count    int64
}

type Entry struct {
	DirEntry fs.DirEntry
	Path     string
	App      *App
}

func (e Entry) Name() string {
	if e.App != nil {
		return e.App.Name
	}
	if e.DirEntry == nil {
		return ""
	}
	return e.DirEntry.Name()
}

type Value struct {
	Text   string
	Number int64
	IsNum  bool
}

type RowSink interface {
	Push(Row) error
}

type Executor interface {
	Verb() string
	Execute(ctx context.Context, stmt *query.Statement, out RowSink) error
}

type FieldExtractor interface {
	Field() string
	Cost() int
	AllowedOperators() []string
	AppliesTo(query.Target) bool
	NormalizeValue(string) (Value, error)
	Extract(Entry) (Value, error)
}

type Comparator interface {
	Op() string
	Compare(got, want Value) bool
}
