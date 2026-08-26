package engine

import (
	"github.com/farhapartex/osql/internal/query"
)

const (
	FieldVersion     = "version"
	FieldVersionLike = "version_like"
	FieldSource      = "source"
	FieldID          = "id"
	FieldIDLike      = "id_like"
)

type VersionField struct{}

func (VersionField) Field() string              { return FieldVersion }
func (VersionField) Cost() int                  { return CostFree }
func (VersionField) AllowedOperators() []string { return equalityOperators }

func (VersionField) AppliesTo(t query.Target) bool { return t == query.TargetApps }

func (VersionField) NormalizeValue(v string) (Value, error) {
	return Value{Text: v}, nil
}

func (VersionField) Extract(e Entry) (Value, error) {
	if e.App == nil {
		return Value{}, errNotAnApp
	}
	return Value{Text: e.App.Version}, nil
}

type VersionLikeField struct{}

func (VersionLikeField) Field() string              { return FieldVersionLike }
func (VersionLikeField) Cost() int                  { return CostFree }
func (VersionLikeField) AllowedOperators() []string { return equalityOperators }

func (VersionLikeField) AppliesTo(t query.Target) bool { return t == query.TargetApps }

func (VersionLikeField) Glob() bool { return true }

func (VersionLikeField) NormalizeValue(v string) (Value, error) {
	return Value{Text: v}, nil
}

func (VersionLikeField) Extract(e Entry) (Value, error) {
	if e.App == nil {
		return Value{}, errNotAnApp
	}
	return Value{Text: e.App.Version}, nil
}

type SourceField struct{}

func (SourceField) Field() string              { return FieldSource }
func (SourceField) Cost() int                  { return CostFree }
func (SourceField) AllowedOperators() []string { return equalityOperators }

func (SourceField) AppliesTo(t query.Target) bool { return t == query.TargetApps }

func (SourceField) NormalizeValue(v string) (Value, error) {
	return Value{Text: CanonicalSource(v)}, nil
}

func (SourceField) Extract(e Entry) (Value, error) {
	if e.App == nil {
		return Value{}, errNotAnApp
	}
	return Value{Text: e.App.Source}, nil
}

type IDField struct{}

func (IDField) Field() string              { return FieldID }
func (IDField) Cost() int                  { return CostFree }
func (IDField) AllowedOperators() []string { return equalityOperators }

func (IDField) AppliesTo(t query.Target) bool { return t == query.TargetApps }

func (IDField) NormalizeValue(v string) (Value, error) {
	return Value{Text: v}, nil
}

func (IDField) Extract(e Entry) (Value, error) {
	if e.App == nil {
		return Value{}, errNotAnApp
	}
	return Value{Text: e.App.ID}, nil
}

type IDLikeField struct{}

func (IDLikeField) Field() string              { return FieldIDLike }
func (IDLikeField) Cost() int                  { return CostFree }
func (IDLikeField) AllowedOperators() []string { return equalityOperators }

func (IDLikeField) AppliesTo(t query.Target) bool { return t == query.TargetApps }

func (IDLikeField) Glob() bool { return true }

func (IDLikeField) NormalizeValue(v string) (Value, error) {
	return Value{Text: v}, nil
}

func (IDLikeField) Extract(e Entry) (Value, error) {
	if e.App == nil {
		return Value{}, errNotAnApp
	}
	return Value{Text: e.App.ID}, nil
}
