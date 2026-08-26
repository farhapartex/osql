package engine

import (
	"strconv"
	"strings"

	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/vfs"
)

const (
	CostFree    = 0
	CostStat    = 10
	CostReadDir = 100
)

const (
	FieldName       = "name"
	FieldNameLike   = "name_like"
	FieldType       = "type"
	FieldCountChild = "count(child)"

	TypeFolder = "folder"
)

var equalityOperators = []string{OpEqual, OpNotEqual}

var orderingOperators = []string{OpEqual, OpNotEqual, OpLess, OpGreater, OpLessEqual, OpGreaterEqual}

type NameField struct{}

func (NameField) Field() string               { return FieldName }
func (NameField) Cost() int                   { return CostFree }
func (NameField) AllowedOperators() []string  { return equalityOperators }
func (NameField) AppliesTo(query.Target) bool { return true }

func (NameField) NormalizeValue(v string) (Value, error) {
	return Value{Text: v}, nil
}

func (NameField) Extract(e Entry) (Value, error) {
	return Value{Text: e.Name()}, nil
}

type NameLikeField struct{}

func (NameLikeField) Field() string               { return FieldNameLike }
func (NameLikeField) Cost() int                   { return CostFree }
func (NameLikeField) AllowedOperators() []string  { return equalityOperators }
func (NameLikeField) AppliesTo(query.Target) bool { return true }
func (NameLikeField) Glob() bool                  { return true }

func (NameLikeField) NormalizeValue(v string) (Value, error) {
	return Value{Text: v}, nil
}

func (NameLikeField) Extract(e Entry) (Value, error) {
	return Value{Text: e.Name()}, nil
}

type TypeField struct{}

func (TypeField) Field() string              { return FieldType }
func (TypeField) Cost() int                  { return CostFree }
func (TypeField) AllowedOperators() []string { return equalityOperators }

func (TypeField) AppliesTo(t query.Target) bool { return t != query.TargetApps }

func (TypeField) NormalizeValue(v string) (Value, error) {
	return Value{Text: strings.TrimPrefix(v, ".")}, nil
}

func (TypeField) Extract(e Entry) (Value, error) {
	if e.DirEntry.IsDir() {
		return Value{Text: TypeFolder}, nil
	}
	return Value{Text: ExtensionOf(e.DirEntry.Name())}, nil
}

type CountChildField struct {
	fsys vfs.FileSystem
}

func NewCountChildField(fsys vfs.FileSystem) CountChildField {
	return CountChildField{fsys: fsys}
}

func (CountChildField) Field() string              { return FieldCountChild }
func (CountChildField) Cost() int                  { return CostReadDir }
func (CountChildField) AllowedOperators() []string { return orderingOperators }

func (CountChildField) AppliesTo(t query.Target) bool {
	return t != query.TargetFiles && t != query.TargetApps
}

func (CountChildField) NormalizeValue(v string) (Value, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return Value{}, oerr.CountChildNonNumeric()
	}
	return Value{Number: n, IsNum: true}, nil
}

func (f CountChildField) Extract(e Entry) (Value, error) {
	if !e.DirEntry.IsDir() {
		return Value{Number: 0, IsNum: true}, nil
	}
	if f.fsys == nil {
		return Value{}, errNoFileSystem
	}

	entries, err := f.fsys.ReadDir(e.Path)
	if err != nil {
		return Value{}, err
	}
	return Value{Number: int64(len(entries)), IsNum: true}, nil
}
