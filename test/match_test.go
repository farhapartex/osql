package test

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
)

func testCompiler(fsys fstest.MapFS) *engine.Compiler {
	return engine.NewCompiler(
		engine.DefaultFields(&fakeFileSystem{fsys: fsys}),
		engine.DefaultOperators(),
	)
}

func TestComparators(t *testing.T) {
	text := func(s string) engine.Value { return engine.Value{Text: s} }
	num := func(n int64) engine.Value { return engine.Value{Number: n, IsNum: true} }

	tests := []struct {
		op         engine.Comparator
		got, want  engine.Value
		wantResult bool
	}{
		{engine.EqualOp{}, text("a"), text("a"), true},
		{engine.EqualOp{}, text("a"), text("b"), false},
		{engine.EqualOp{}, text("A"), text("a"), false},
		{engine.EqualOp{}, num(3), num(3), true},
		{engine.EqualOp{}, num(3), num(4), false},
		{engine.NotEqualOp{}, text("a"), text("b"), true},
		{engine.NotEqualOp{}, text("a"), text("a"), false},
		{engine.NotEqualOp{}, num(3), num(3), false},
		{engine.LessOp{}, num(2), num(3), true},
		{engine.LessOp{}, num(3), num(3), false},
		{engine.LessOp{}, num(4), num(3), false},
		{engine.GreaterOp{}, num(4), num(3), true},
		{engine.GreaterOp{}, num(3), num(3), false},
		{engine.LessEqualOp{}, num(3), num(3), true},
		{engine.LessEqualOp{}, num(2), num(3), true},
		{engine.LessEqualOp{}, num(4), num(3), false},
		{engine.GreaterEqualOp{}, num(3), num(3), true},
		{engine.GreaterEqualOp{}, num(4), num(3), true},
		{engine.GreaterEqualOp{}, num(2), num(3), false},
		{engine.LessOp{}, text("a"), text("b"), false},
		{engine.GreaterOp{}, text("b"), text("a"), false},
		{engine.LessOp{}, num(1), text("2"), false},
		{engine.LessEqualOp{}, text("a"), text("a"), false},
		{engine.LessEqualOp{}, num(1), text("2"), false},
		{engine.LessEqualOp{}, text("1"), num(2), false},
		{engine.GreaterEqualOp{}, text("a"), text("a"), false},
		{engine.GreaterEqualOp{}, num(2), text("1"), false},
		{engine.GreaterEqualOp{}, text("2"), num(1), false},
		{engine.GreaterOp{}, num(2), text("1"), false},
		{engine.LessOp{}, text("1"), num(2), false},
	}

	for _, tt := range tests {
		t.Run(tt.op.Op(), func(t *testing.T) {
			if got := tt.op.Compare(tt.got, tt.want); got != tt.wantResult {
				t.Errorf("%s.Compare(%+v, %+v) = %v, want %v", tt.op.Op(), tt.got, tt.want, got, tt.wantResult)
			}
		})
	}
}

func TestFieldRegistryNamesFollowDeclarationOrder(t *testing.T) {
	r := engine.DefaultFields(nil)

	got := r.Names()
	want := []string{
		"name", "name_like", "type", "size", "count(child)",
		"version", "version_like", "source", "id", "id_like",
	}
	if !slices.Equal(got, want) {
		t.Errorf("Names() = %v, want %v — the order feeds the spec's unknown-field message", got, want)
	}
}

func TestOperatorRegistryOpsFollowDeclarationOrder(t *testing.T) {
	r := engine.DefaultOperators()

	got := r.Ops()
	want := []string{"=", "!=", "<", ">", "<=", ">="}
	if !slices.Equal(got, want) {
		t.Errorf("Ops() = %v, want %v", got, want)
	}
}

func TestFieldRegistryLookup(t *testing.T) {
	r := engine.DefaultFields(nil)

	for _, name := range []string{"name", "name_like", "type", "size", "count(child)"} {
		if _, ok := r.Lookup(name); !ok {
			t.Errorf("Lookup(%q) = false", name)
		}
	}
	for _, name := range []string{"modified", "extension", "", "NAME"} {
		if _, ok := r.Lookup(name); ok {
			t.Errorf("Lookup(%q) = true, want false", name)
		}
	}
}

func TestFieldRegistryZeroValueAndOverwrite(t *testing.T) {
	var r engine.FieldRegistry

	if _, ok := r.Lookup("name"); ok {
		t.Error("zero-value registry returned a field")
	}
	r.Register(engine.NameField{})
	r.Register(engine.NameField{})

	if _, ok := r.Lookup("name"); !ok {
		t.Error("Register did not take effect")
	}
	if got := r.Names(); len(got) != 1 {
		t.Errorf("Names() = %v, want one entry after re-registering the same field", got)
	}
}

func TestOperatorRegistryZeroValueAndOverwrite(t *testing.T) {
	var r engine.OperatorRegistry

	r.Register(engine.EqualOp{})
	r.Register(engine.EqualOp{})

	if _, ok := r.Lookup("="); !ok {
		t.Error("Register did not take effect")
	}
	if got := r.Ops(); len(got) != 1 {
		t.Errorf("Ops() = %v, want one entry", got)
	}
}

func TestValidityMatrix(t *testing.T) {
	c := testCompiler(nil)

	allowed := map[string][]string{
		"name":         {"=", "!="},
		"name_like":    {"=", "!="},
		"type":         {"=", "!="},
		"count(child)": {"=", "!=", "<", ">", "<=", ">="},
	}
	everyOp := []string{"=", "!=", "<", ">", "<=", ">="}

	for field, ops := range allowed {
		for _, op := range everyOp {
			value := "x"
			if field == "count(child)" {
				value = "1"
			}
			p := query.Predicate{Field: field, Op: op, Value: value}

			err := c.Validate(p, query.TargetFolders)
			shouldPass := slices.Contains(ops, op)

			if shouldPass && err != nil {
				t.Errorf("%s %s should be allowed, got %v", field, op, err)
			}
			if !shouldPass {
				if err == nil {
					t.Errorf("%s %s should be rejected", field, op)
					continue
				}
				if !oerr.Is(err, oerr.KindWrongOperator) {
					t.Errorf("%s %s error kind = %v, want wrong_operator", field, op, err)
				}
			}
		}
	}
}

func TestValidateUnknownField(t *testing.T) {
	c := testCompiler(nil)

	err := c.Validate(query.Predicate{Field: "extension", Op: "=", Value: "txt"}, query.TargetFiles)
	if err == nil {
		t.Fatal("unknown field accepted")
	}
	if !oerr.Is(err, oerr.KindUnknownField) {
		t.Errorf("error kind = %v, want unknown_field", err)
	}
	for _, name := range []string{"name", "name_like", "type"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error message does not list %q: %v", name, err)
		}
	}
	if strings.Contains(err.Error(), "count(child)") {
		t.Errorf("count(child) does not work on files, so it must not be offered: %v", err)
	}
	if strings.Contains(err.Error(), "version") {
		t.Errorf("app-only fields must not be offered for a files query: %v", err)
	}
}

func TestUnknownFieldListsOnlyFieldsForThatTarget(t *testing.T) {
	c := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())

	folders := c.Validate(query.Predicate{Field: "extension", Op: "=", Value: "txt"}, query.TargetFolders)
	if !strings.Contains(folders.Error(), "count(child)") {
		t.Errorf("count(child) works on folders, so it must be offered: %v", folders)
	}

	apps := c.Validate(query.Predicate{Field: "extension", Op: "=", Value: "txt"}, query.TargetApps)
	for _, name := range []string{"version", "source", "id"} {
		if !strings.Contains(apps.Error(), name) {
			t.Errorf("apps query must offer %q: %v", name, apps)
		}
	}
	if strings.Contains(apps.Error(), "count(child)") {
		t.Errorf("count(child) must not be offered for apps: %v", apps)
	}
}

func TestUnknownFieldMessageGrowsWithTheRegistry(t *testing.T) {
	fields := engine.DefaultFields(nil)
	fields.Register(&fakeFieldExtractor{field: "size", allowedOps: []string{"="}})

	c := engine.NewCompiler(fields, engine.DefaultOperators())
	err := c.Validate(query.Predicate{Field: "nope", Op: "=", Value: "x"}, query.TargetFiles)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "size") {
		t.Errorf("a newly registered field did not appear in the message: %v", err)
	}
}

func TestValidateCountChildOnFiles(t *testing.T) {
	c := testCompiler(nil)

	err := c.Validate(query.Predicate{Field: "count(child)", Op: ">", Value: "10"}, query.TargetFiles)
	if err == nil {
		t.Fatal("count(child) accepted for select files")
	}
	if !oerr.Is(err, oerr.KindCountChildOnFiles) {
		t.Errorf("error kind = %v, want count_child_on_files", err)
	}
}

func TestValidateCountChildNonNumeric(t *testing.T) {
	c := testCompiler(nil)

	err := c.Validate(query.Predicate{Field: "count(child)", Op: ">", Value: "many"}, query.TargetFolders)
	if err == nil {
		t.Fatal("count(child) accepted a non-numeric value")
	}
	if !oerr.Is(err, oerr.KindCountChildNonNumeric) {
		t.Errorf("error kind = %v, want count_child_non_numeric", err)
	}
}

func TestCompileSortsByCostCheapestFirst(t *testing.T) {
	c := testCompiler(sampleFS())

	predicates := []query.Predicate{
		{Field: "count(child)", Op: ">", Value: "1"},
		{Field: "name", Op: "=", Value: "a"},
		{Field: "type", Op: "=", Value: "txt"},
	}

	matchers, err := c.CompileAll(predicates, query.TargetFolders)
	if err != nil {
		t.Fatal(err)
	}

	if len(matchers) != 3 {
		t.Fatalf("got %d matchers, want 3", len(matchers))
	}
	if matchers[len(matchers)-1].Field() != "count(child)" {
		t.Errorf("last matcher = %q, want count(child); the ReadDir predicate must run last", matchers[len(matchers)-1].Field())
	}
	for i := 1; i < len(matchers); i++ {
		if matchers[i-1].Cost() > matchers[i].Cost() {
			t.Errorf("matchers not ordered by cost: %d then %d", matchers[i-1].Cost(), matchers[i].Cost())
		}
	}
}

func TestCompileSortIsStableForEqualCosts(t *testing.T) {
	c := testCompiler(nil)

	predicates := []query.Predicate{
		{Field: "type", Op: "=", Value: "txt"},
		{Field: "name", Op: "=", Value: "a"},
		{Field: "name_like", Op: "=", Value: "%a%"},
	}

	matchers, err := c.CompileAll(predicates, query.TargetFiles)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"type", "name", "name_like"}
	for i, w := range want {
		if matchers[i].Field() != w {
			t.Errorf("matcher %d = %q, want %q; equal costs must keep the typed order", i, matchers[i].Field(), w)
		}
	}
}

func TestCompileAllStopsAtFirstInvalidPredicate(t *testing.T) {
	c := testCompiler(nil)

	predicates := []query.Predicate{
		{Field: "name", Op: "=", Value: "a"},
		{Field: "bogus", Op: "=", Value: "b"},
	}

	if _, err := c.CompileAll(predicates, query.TargetFiles); err == nil {
		t.Fatal("CompileAll accepted an invalid predicate")
	}
}

func TestCompileAllOnEmptyPredicates(t *testing.T) {
	c := testCompiler(nil)

	matchers, err := c.CompileAll(nil, query.TargetAll)
	if err != nil {
		t.Fatalf("CompileAll(nil) error = %v", err)
	}
	if len(matchers) != 0 {
		t.Errorf("got %d matchers, want 0", len(matchers))
	}
}

func TestMatcherOnName(t *testing.T) {
	fsys := sampleFS()
	c := testCompiler(fsys)

	tests := []struct {
		field string
		op    string
		value string
		name  string
		want  bool
	}{
		{"name", "=", "notes.txt", "notes.txt", true},
		{"name", "=", "notes.txt", "report.PDF", false},
		{"name", "!=", "notes.txt", "report.PDF", true},
		{"name", "!=", "notes.txt", "notes.txt", false},
		{"type", "=", "txt", "notes.txt", true},
		{"type", "=", ".txt", "notes.txt", true},
		{"type", "=", "txt", "report.PDF", false},
		{"type", "!=", "txt", "report.PDF", true},
		{"name_like", "=", "%report%", "report.PDF", true},
		{"name_like", "=", "%report%", "notes.txt", false},
		{"name_like", "=", "notes%", "notes.txt", true},
		{"name_like", "=", "%.txt", "notes.txt", true},
		{"name_like", "!=", "%report%", "notes.txt", true},
		{"name_like", "!=", "%report%", "report.PDF", false},
	}

	for _, tt := range tests {
		t.Run(tt.field+tt.op+tt.value+"_"+tt.name, func(t *testing.T) {
			m, err := c.Compile(query.Predicate{Field: tt.field, Op: tt.op, Value: tt.value}, query.TargetFiles)
			if err != nil {
				t.Fatal(err)
			}
			got, err := m.Match(entryFor(t, fsys, ".", tt.name))
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("%s %s %q against %q = %v, want %v", tt.field, tt.op, tt.value, tt.name, got, tt.want)
			}
		})
	}
}

func TestMatcherOnCountChild(t *testing.T) {
	fsys := sampleFS()
	c := testCompiler(fsys)

	tests := []struct {
		op    string
		value string
		dir   string
		want  bool
	}{
		{">", "2", "three", true},
		{">", "3", "three", false},
		{"=", "3", "three", true},
		{"=", "0", "three", false},
		{"<", "5", "three", true},
		{"<=", "3", "three", true},
		{">=", "3", "three", true},
		{"!=", "3", "three", false},
		{"!=", "1", "three", true},
		{"=", "1", "one", true},
	}

	for _, tt := range tests {
		t.Run(tt.dir+tt.op+tt.value, func(t *testing.T) {
			m, err := c.Compile(query.Predicate{Field: "count(child)", Op: tt.op, Value: tt.value}, query.TargetFolders)
			if err != nil {
				t.Fatal(err)
			}
			got, err := m.Match(entryFor(t, fsys, ".", tt.dir))
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("count(child) %s %s on %q = %v, want %v", tt.op, tt.value, tt.dir, got, tt.want)
			}
		})
	}
}

func TestMatchAllRequiresEveryMatcher(t *testing.T) {
	fsys := sampleFS()
	c := testCompiler(fsys)

	matchers, err := c.CompileAll([]query.Predicate{
		{Field: "type", Op: "=", Value: "txt"},
		{Field: "name_like", Op: "=", Value: "note%"},
	}, query.TargetFiles)
	if err != nil {
		t.Fatal(err)
	}

	hit, err := engine.MatchAll(matchers, entryFor(t, fsys, ".", "notes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !hit {
		t.Error("notes.txt should satisfy both predicates")
	}

	miss, err := engine.MatchAll(matchers, entryFor(t, fsys, ".", "report.PDF"))
	if err != nil {
		t.Fatal(err)
	}
	if miss {
		t.Error("report.PDF satisfies neither predicate")
	}
}

func TestMatchAllOnNoMatchersAcceptsEverything(t *testing.T) {
	fsys := sampleFS()

	hit, err := engine.MatchAll(nil, entryFor(t, fsys, ".", "notes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !hit {
		t.Error("an unfiltered query must accept every entry")
	}
}

func TestMatchAllPropagatesExtractError(t *testing.T) {
	fsys := sampleFS()
	c := engine.NewCompiler(engine.DefaultFields(nil), engine.DefaultOperators())

	m, err := c.Compile(query.Predicate{Field: "count(child)", Op: "=", Value: "1"}, query.TargetFolders)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := engine.MatchAll([]engine.Matcher{m}, entryFor(t, fsys, ".", "three")); err == nil {
		t.Error("MatchAll swallowed an extraction error")
	}
}
