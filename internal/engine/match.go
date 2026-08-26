package engine

import (
	"errors"
	"slices"
	"sort"

	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/vfs"
)

var errNoFileSystem = errors.New("no filesystem configured")

var errNoCatalog = errors.New("no app catalog configured")

var errNotAnApp = errors.New("field applies only to apps")

type GlobField interface {
	Glob() bool
}

type FieldRegistry struct {
	byName map[string]FieldExtractor
	order  []string
}

func NewFieldRegistry(fields ...FieldExtractor) *FieldRegistry {
	r := &FieldRegistry{byName: make(map[string]FieldExtractor, len(fields))}
	for _, f := range fields {
		r.Register(f)
	}
	return r
}

func DefaultFields(fsys vfs.FileSystem) *FieldRegistry {
	return NewFieldRegistry(
		NameField{},
		NameLikeField{},
		TypeField{},
		NewCountChildField(fsys),
		VersionField{},
		VersionLikeField{},
		SourceField{},
		IDField{},
		IDLikeField{},
	)
}

func (r *FieldRegistry) Register(f FieldExtractor) {
	if r.byName == nil {
		r.byName = make(map[string]FieldExtractor)
	}
	name := f.Field()
	if _, exists := r.byName[name]; !exists {
		r.order = append(r.order, name)
	}
	r.byName[name] = f
}

func (r *FieldRegistry) Lookup(name string) (FieldExtractor, bool) {
	f, ok := r.byName[name]
	return f, ok
}

func (r *FieldRegistry) Names() []string {
	return slices.Clone(r.order)
}

type OperatorRegistry struct {
	byOp  map[string]Comparator
	order []string
}

func NewOperatorRegistry(ops ...Comparator) *OperatorRegistry {
	r := &OperatorRegistry{byOp: make(map[string]Comparator, len(ops))}
	for _, o := range ops {
		r.Register(o)
	}
	return r
}

func DefaultOperators() *OperatorRegistry {
	return NewOperatorRegistry(
		EqualOp{},
		NotEqualOp{},
		LessOp{},
		GreaterOp{},
		LessEqualOp{},
		GreaterEqualOp{},
	)
}

func (r *OperatorRegistry) Register(c Comparator) {
	if r.byOp == nil {
		r.byOp = make(map[string]Comparator)
	}
	op := c.Op()
	if _, exists := r.byOp[op]; !exists {
		r.order = append(r.order, op)
	}
	r.byOp[op] = c
}

func (r *OperatorRegistry) Lookup(op string) (Comparator, bool) {
	c, ok := r.byOp[op]
	return c, ok
}

func (r *OperatorRegistry) Ops() []string {
	return slices.Clone(r.order)
}

type Matcher struct {
	field   FieldExtractor
	compare Comparator
	want    Value
	pattern Pattern
	isGlob  bool
	negate  bool
}

func (m Matcher) Cost() int {
	return m.field.Cost()
}

func (m Matcher) Field() string {
	return m.field.Field()
}

func (m Matcher) Match(e Entry) (bool, error) {
	got, err := m.field.Extract(e)
	if err != nil {
		return false, err
	}

	if m.isGlob {
		hit := m.pattern.Match(got.Text)
		if m.negate {
			return !hit, nil
		}
		return hit, nil
	}

	return m.compare.Compare(got, m.want), nil
}

type Compiler struct {
	fields *FieldRegistry
	ops    *OperatorRegistry
}

func NewCompiler(fields *FieldRegistry, ops *OperatorRegistry) *Compiler {
	return &Compiler{fields: fields, ops: ops}
}

func (c *Compiler) Fields() *FieldRegistry { return c.fields }

func (c *Compiler) Operators() *OperatorRegistry { return c.ops }

func (c *Compiler) fieldsFor(target query.Target) []string {
	names := make([]string, 0, len(c.fields.Names()))
	for _, name := range c.fields.Names() {
		if field, ok := c.fields.Lookup(name); ok && field.AppliesTo(target) {
			names = append(names, name)
		}
	}
	return names
}

func (c *Compiler) Validate(p query.Predicate, target query.Target) error {
	field, ok := c.fields.Lookup(p.Field)
	if !ok {
		return oerr.UnknownField(p.Field, c.fieldsFor(target))
	}
	if !field.AppliesTo(target) {
		if p.Field == FieldCountChild && target == query.TargetFiles {
			return oerr.CountChildOnFiles()
		}
		return oerr.FieldNotForTarget(p.Field, target.String(), c.fieldsFor(target))
	}
	if !slices.Contains(field.AllowedOperators(), p.Op) {
		return oerr.WrongOperator(p.Field, field.AllowedOperators())
	}
	if _, ok := c.ops.Lookup(p.Op); !ok {
		return oerr.WrongOperator(p.Field, field.AllowedOperators())
	}
	if _, err := field.NormalizeValue(p.Value); err != nil {
		return err
	}
	return nil
}

func (c *Compiler) Compile(p query.Predicate, target query.Target) (Matcher, error) {
	if err := c.Validate(p, target); err != nil {
		return Matcher{}, err
	}

	field, _ := c.fields.Lookup(p.Field)
	compare, _ := c.ops.Lookup(p.Op)

	want, err := field.NormalizeValue(p.Value)
	if err != nil {
		return Matcher{}, err
	}

	m := Matcher{field: field, compare: compare, want: want}
	if glob, ok := field.(GlobField); ok && glob.Glob() {
		m.isGlob = true
		m.pattern = CompilePattern(want.Text)
		m.negate = p.Op == OpNotEqual
	}
	return m, nil
}

func (c *Compiler) CompileAll(predicates []query.Predicate, target query.Target) ([]Matcher, error) {
	matchers := make([]Matcher, 0, len(predicates))
	for _, p := range predicates {
		m, err := c.Compile(p, target)
		if err != nil {
			return nil, err
		}
		matchers = append(matchers, m)
	}
	SortMatchers(matchers)
	return matchers, nil
}

func SortMatchers(matchers []Matcher) {
	sort.SliceStable(matchers, func(i, j int) bool {
		return matchers[i].Cost() < matchers[j].Cost()
	})
}

func MatchAll(matchers []Matcher, e Entry) (bool, error) {
	for _, m := range matchers {
		ok, err := m.Match(e)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}
