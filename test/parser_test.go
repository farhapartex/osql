package test

import (
	"errors"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
)

func parse(t *testing.T, input string) (*query.Statement, error) {
	t.Helper()

	tokens, err := query.NewLexer().Lex(input)
	if err != nil {
		return nil, err
	}
	validator := engine.NewCompiler(engine.DefaultFields(&fakeFileSystem{fsys: fstest.MapFS{}}), engine.DefaultOperators())
	return query.NewParser(validator).Parse(tokens)
}

func parseNoValidation(t *testing.T, input string) (*query.Statement, error) {
	t.Helper()

	tokens, err := query.NewLexer().Lex(input)
	if err != nil {
		return nil, err
	}
	return query.NewParser(nil).Parse(tokens)
}

func TestParseSpecExamples(t *testing.T) {
	tests := []struct {
		input      string
		target     query.Target
		path       string
		recursive  bool
		predicates []query.Predicate
	}{
		{input: "all from '~/Downloads'", target: query.TargetAll, path: "~/Downloads"},
		{input: "files from 'Documents'", target: query.TargetFiles, path: "Documents"},
		{input: "folders from '/'", target: query.TargetFolders, path: "/"},
		{input: "files from Documents", target: query.TargetFiles, path: "Documents"},
		{input: "FILES FROM 'Documents';", target: query.TargetFiles, path: "Documents"},
		{
			input: "files from 'Documents' where type = 'txt'", target: query.TargetFiles, path: "Documents",
			predicates: []query.Predicate{{Field: "type", Op: "=", Value: "txt"}},
		},
		{
			input: "files from 'Documents' where type = '.txt'", target: query.TargetFiles, path: "Documents",
			predicates: []query.Predicate{{Field: "type", Op: "=", Value: ".txt"}},
		},
		{
			input: "files from 'Documents' where name = 'notes.txt'", target: query.TargetFiles, path: "Documents",
			predicates: []query.Predicate{{Field: "name", Op: "=", Value: "notes.txt"}},
		},
		{
			input: "files from 'Documents' where name != 'secret.txt'", target: query.TargetFiles, path: "Documents",
			predicates: []query.Predicate{{Field: "name", Op: "!=", Value: "secret.txt"}},
		},
		{
			input: "files from 'Documents' where name_like = '%report%'", target: query.TargetFiles, path: "Documents",
			predicates: []query.Predicate{{Field: "name_like", Op: "=", Value: "%report%"}},
		},
		{
			input: "files from 'Documents' where name_like = 'report%'", target: query.TargetFiles, path: "Documents",
			predicates: []query.Predicate{{Field: "name_like", Op: "=", Value: "report%"}},
		},
		{
			input: "files from 'Documents' where name_like = '%report'", target: query.TargetFiles, path: "Documents",
			predicates: []query.Predicate{{Field: "name_like", Op: "=", Value: "%report"}},
		},
		{
			input: "files from 'src' where name_like = 'test_%' and type = 'go'", target: query.TargetFiles, path: "src",
			predicates: []query.Predicate{
				{Field: "name_like", Op: "=", Value: "test_%"},
				{Field: "type", Op: "=", Value: "go"},
			},
		},
		{
			input: "folders from 'src' where count(child) > 10", target: query.TargetFolders, path: "src",
			predicates: []query.Predicate{{Field: "count(child)", Op: ">", Value: "10"}},
		},
		{
			input: "folders from 'src' where count(child) = 0", target: query.TargetFolders, path: "src",
			predicates: []query.Predicate{{Field: "count(child)", Op: "=", Value: "0"}},
		},
		{
			input: "folders from '~' where count(child) <= 2", target: query.TargetFolders, path: "~",
			predicates: []query.Predicate{{Field: "count(child)", Op: "<=", Value: "2"}},
		},
		{
			input: "files from '~' recursive where name = 'notes.txt'", target: query.TargetFiles, path: "~", recursive: true,
			predicates: []query.Predicate{{Field: "name", Op: "=", Value: "notes.txt"}},
		},
		{
			input: "files from '.' recursive where name_like = '%.log'", target: query.TargetFiles, path: ".", recursive: true,
			predicates: []query.Predicate{{Field: "name_like", Op: "=", Value: "%.log"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			stmt, err := parse(t, tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			if stmt.Verb != "select" {
				t.Errorf("Verb = %q, want select", stmt.Verb)
			}
			if stmt.Target != tt.target {
				t.Errorf("Target = %v, want %v", stmt.Target, tt.target)
			}
			if stmt.Path != tt.path {
				t.Errorf("Path = %q, want %q", stmt.Path, tt.path)
			}
			if stmt.Recursive != tt.recursive {
				t.Errorf("Recursive = %v, want %v", stmt.Recursive, tt.recursive)
			}
			if !slices.Equal(stmt.Predicates, tt.predicates) {
				t.Errorf("Predicates = %+v, want %+v", stmt.Predicates, tt.predicates)
			}
		})
	}
}

func TestParseErrorKinds(t *testing.T) {
	tests := []struct {
		name  string
		input string
		kind  oerr.Kind
	}{
		{"legacy select verb", "select files from 'a'", oerr.KindNoVerbNeeded},
		{"unknown target", "slect files from 'a'", oerr.KindUnknownTarget},
		{"empty input", "", oerr.KindMissingTarget},
		{"missing target before from", "from 'a'", oerr.KindMissingTarget},
		{"bare from", "from", oerr.KindMissingTarget},
		{"missing target before where", "where name = 'a'", oerr.KindMissingTarget},
		{"singular file", "file from 'a'", oerr.KindSingularTarget},
		{"singular folder", "folder from 'a'", oerr.KindSingularTarget},
		{"unknown word", "documents from 'a'", oerr.KindUnknownTarget},
		{"missing from", "files 'a'", oerr.KindMissingFrom},
		{"missing from at end", "files", oerr.KindMissingFrom},
		{"missing path", "files from", oerr.KindMissingPath},
		{"missing path before where", "files from where name = 'a'", oerr.KindMissingPath},
		{"unknown field", "files from 'a' where extension = 'txt'", oerr.KindUnknownField},
		{"wrong operator for name", "files from 'a' where name < 'b'", oerr.KindWrongOperator},
		{"wrong operator for type", "files from 'a' where type > 'b'", oerr.KindWrongOperator},
		{"lone bang operator", "files from 'a' where name ! 'b'", oerr.KindWrongOperator},
		{"count child on files", "files from 'a' where count(child) > 1", oerr.KindCountChildOnFiles},
		{"count child non numeric", "folders from 'a' where count(child) > 'many'", oerr.KindCountChildNonNumeric},
		{"unclosed quote", "files from 'a", oerr.KindUnclosedQuote},
		{"trailing junk", "files from 'a' junk", oerr.KindUnexpectedInput},
		{"junk after recursive", "files from 'a' recursive junk", oerr.KindUnexpectedInput},
		{"where with nothing", "files from 'a' where", oerr.KindIncompleteQuery},
		{"and with nothing", "files from 'a' where name = 'b' and", oerr.KindIncompleteQuery},
		{"field with no operator", "files from 'a' where name", oerr.KindIncompleteQuery},
		{"operator with no value", "files from 'a' where name =", oerr.KindIncompleteQuery},
		{"or is not supported", "files from 'a' where name = 'b' or type = 'c'", oerr.KindUnexpectedInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parse(t, tt.input)
			if err == nil {
				t.Fatalf("Parse(%q) succeeded; want %v", tt.input, tt.kind)
			}
			if !oerr.Is(err, tt.kind) {
				t.Errorf("Parse(%q) error kind mismatch\n got: %v\nwant: %v", tt.input, err, tt.kind)
			}
		})
	}
}

func TestParseSingularTargetSuggestsPlural(t *testing.T) {
	_, err := parse(t, "file from 'a'")
	if err == nil {
		t.Fatal("expected an error")
	}
	want := `Use "files", not "file" — for example: files from 'Documents'`
	if err.Error() != want {
		t.Errorf("\n got: %s\nwant: %s", err.Error(), want)
	}
}

func TestParseUnknownFieldListsRegisteredFields(t *testing.T) {
	_, err := parse(t, "files from 'a' where extension = 'txt'")
	if err == nil {
		t.Fatal("expected an error")
	}
	want := `I don't know the field "extension". I understand: name, name_like, type, size`
	if err.Error() != want {
		t.Errorf("\n got: %s\nwant: %s", err.Error(), want)
	}

	_, err = parse(t, "folders from 'a' where extension = 'txt'")
	if err == nil {
		t.Fatal("expected an error")
	}
	want = `I don't know the field "extension". I understand: name, name_like, type, count(child)`
	if err.Error() != want {
		t.Errorf("\n got: %s\nwant: %s", err.Error(), want)
	}
}

func TestParseMalformedCountReportsUnknownField(t *testing.T) {
	tests := []string{
		"folders from 'a' where count(parent) > 1",
		"folders from 'a' where count = 1",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := parse(t, input)
			if err == nil {
				t.Fatalf("Parse(%q) succeeded", input)
			}
			if !oerr.Is(err, oerr.KindUnknownField) {
				t.Errorf("error kind = %v, want unknown_field", err)
			}
		})
	}
}

func TestParseKeywordsAreCaseInsensitive(t *testing.T) {
	inputs := []string{
		"FILES FROM 'a' WHERE NAME = 'b'",
		"Files From 'a' Where Name = 'b'",
		"fIlEs FrOm 'a' ReCuRsIvE",
		"FOLDERS from 'a' where COUNT(CHILD) > 1",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			if _, err := parse(t, input); err != nil {
				t.Errorf("Parse(%q) error = %v; keywords are case-insensitive", input, err)
			}
		})
	}
}

func TestParseNormalisesFieldNamesButNotValues(t *testing.T) {
	stmt, err := parse(t, "files from 'a' where NAME = 'MixedCase.TXT'")
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Predicates[0].Field != "name" {
		t.Errorf("Field = %q, want lowercased \"name\"", stmt.Predicates[0].Field)
	}
	if stmt.Predicates[0].Value != "MixedCase.TXT" {
		t.Errorf("Value = %q, want the original case preserved", stmt.Predicates[0].Value)
	}
}

func TestParsePathPreservesCase(t *testing.T) {
	stmt, err := parse(t, "files from '/Users/Nazmul/Documents'")
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Path != "/Users/Nazmul/Documents" {
		t.Errorf("Path = %q, want the original case", stmt.Path)
	}
}

func TestParseRecursiveIsOptOut(t *testing.T) {
	stmt, err := parse(t, "files from 'a'")
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Recursive {
		t.Error("Recursive defaults to true; it must be opt-in per the plan")
	}
}

func TestParseRecursiveMustPrecedeWhere(t *testing.T) {
	if _, err := parse(t, "files from 'a' where name = 'b' recursive"); err == nil {
		t.Error("recursive after where was accepted; the grammar places it before where")
	}
}

func TestParseBareWordValues(t *testing.T) {
	stmt, err := parse(t, "files from 'a' where type = txt")
	if err != nil {
		t.Fatalf("bare word value rejected: %v", err)
	}
	if stmt.Predicates[0].Value != "txt" {
		t.Errorf("Value = %q, want \"txt\"", stmt.Predicates[0].Value)
	}
}

func TestParseTightOperatorSpacing(t *testing.T) {
	spaced, err := parse(t, "files from 'a' where type = 'txt'")
	if err != nil {
		t.Fatal(err)
	}
	tight, err := parse(t, "files from 'a' where type='txt'")
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(spaced.Predicates, tight.Predicates) {
		t.Errorf("spacing changed the parse: %+v vs %+v", spaced.Predicates, tight.Predicates)
	}
}

func TestParseThreePredicates(t *testing.T) {
	stmt, err := parse(t, "files from 'a' where name_like = '%a%' and type = 'go' and name != 'b'")
	if err != nil {
		t.Fatal(err)
	}
	if len(stmt.Predicates) != 3 {
		t.Fatalf("got %d predicates, want 3", len(stmt.Predicates))
	}
}

func TestParsePredicateOrderIsPreserved(t *testing.T) {
	stmt, err := parse(t, "files from 'a' where type = 'go' and name = 'b'")
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Predicates[0].Field != "type" || stmt.Predicates[1].Field != "name" {
		t.Errorf("predicate order changed: %+v; the executor sorts by cost later", stmt.Predicates)
	}
}

func TestParseWithoutValidatorSkipsFieldChecks(t *testing.T) {
	stmt, err := parseNoValidation(t, "files from 'a' where whatever = 'x'")
	if err != nil {
		t.Fatalf("a nil validator must skip semantic checks, got %v", err)
	}
	if stmt.Predicates[0].Field != "whatever" {
		t.Errorf("Field = %q", stmt.Predicates[0].Field)
	}
}

func TestParseNilAndEmptyTokens(t *testing.T) {
	p := query.NewParser(nil)

	for _, tokens := range [][]query.Token{nil, {}, {{Kind: query.TokenEOF}}} {
		if _, err := p.Parse(tokens); err == nil {
			t.Errorf("Parse(%v) succeeded; want an error", tokens)
		}
	}
}

func FuzzParse(f *testing.F) {
	seeds := []string{
		"files from 'Documents'",
		"folders from 'src' where count(child) <= 2",
		"files from '~' recursive where name_like = '%a%'",
		"select",
		"",
		"files from 'a' where",
		"file from 'a'",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	lexer := query.NewLexer()
	validator := engine.NewCompiler(engine.DefaultFields(&fakeFileSystem{fsys: fstest.MapFS{}}), engine.DefaultOperators())
	parser := query.NewParser(validator)

	f.Fuzz(func(t *testing.T, input string) {
		tokens, err := lexer.Lex(input)
		if err != nil {
			return
		}

		stmt, err := parser.Parse(tokens)
		if err != nil {
			var oe *oerr.Error
			if !errors.As(err, &oe) {
				t.Fatalf("Parse(%q) returned a non-oerr error: %v", input, err)
			}
			if stmt != nil {
				t.Errorf("Parse(%q) returned both a statement and an error", input)
			}
			return
		}

		if stmt == nil {
			t.Fatalf("Parse(%q) returned no statement and no error", input)
		}
		if stmt.Verb != "select" {
			t.Errorf("Parse(%q) produced verb %q", input, stmt.Verb)
		}
		switch stmt.Target {
		case query.TargetAll, query.TargetFiles, query.TargetFolders:
		default:
			t.Errorf("Parse(%q) produced an invalid target %v", input, stmt.Target)
		}
	})
}
