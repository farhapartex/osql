package test

import (
	"strings"
	"testing"

	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
)

type tok struct {
	kind  query.TokenKind
	value string
}

func lexTokens(t *testing.T, input string) []tok {
	t.Helper()

	got, err := query.NewLexer().Lex(input)
	if err != nil {
		t.Fatalf("Lex(%q) error = %v", input, err)
	}
	if len(got) == 0 {
		t.Fatalf("Lex(%q) returned no tokens; EOF is always appended", input)
	}
	last := got[len(got)-1]
	if last.Kind != query.TokenEOF {
		t.Fatalf("Lex(%q) last token = %v, want EOF", input, last.Kind)
	}

	out := make([]tok, 0, len(got)-1)
	for _, tk := range got[:len(got)-1] {
		out = append(out, tok{tk.Kind, tk.Value})
	}
	return out
}

func assertTokens(t *testing.T, input string, want []tok) {
	t.Helper()

	got := lexTokens(t, input)
	if len(got) != len(want) {
		t.Fatalf("Lex(%q) produced %d tokens, want %d\ngot:  %v\nwant: %v", input, len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Lex(%q) token %d = {%v %q}, want {%v %q}", input, i, got[i].kind, got[i].value, want[i].kind, want[i].value)
		}
	}
}

func TestLexFullStatements(t *testing.T) {
	ident := query.TokenIdent
	str := query.TokenString
	op := query.TokenOperator
	lp := query.TokenLParen
	rp := query.TokenRParen

	tests := []struct {
		name  string
		input string
		want  []tok
	}{
		{
			"simple query",
			"files from 'Documents'",
			[]tok{{ident, "files"}, {ident, "from"}, {str, "Documents"}},
		},
		{
			"bare word path",
			"files from Documents",
			[]tok{{ident, "files"}, {ident, "from"}, {ident, "Documents"}},
		},
		{
			"where clause",
			"files from 'd' where type = 'txt'",
			[]tok{{ident, "files"}, {ident, "from"}, {str, "d"}, {ident, "where"}, {ident, "type"}, {op, "="}, {str, "txt"}},
		},
		{
			"recursive",
			"files from '~' recursive",
			[]tok{{ident, "files"}, {ident, "from"}, {str, "~"}, {ident, "recursive"}},
		},
		{
			"count child",
			"folders from 'src' where count(child) > 10",
			[]tok{{ident, "folders"}, {ident, "from"}, {str, "src"}, {ident, "where"}, {ident, "count"}, {lp, "("}, {ident, "child"}, {rp, ")"}, {op, ">"}, {ident, "10"}},
		},
		{
			"two predicates",
			"files from 's' where name_like = 'test_%' and type = 'go'",
			[]tok{{ident, "files"}, {ident, "from"}, {str, "s"}, {ident, "where"}, {ident, "name_like"}, {op, "="}, {str, "test_%"}, {ident, "and"}, {ident, "type"}, {op, "="}, {str, "go"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.input, tt.want)
		})
	}
}

func TestLexOperatorsDoNotNeedSpaces(t *testing.T) {
	spaced := lexTokens(t, "type = 'txt'")
	tight := lexTokens(t, "type='txt'")

	if len(spaced) != len(tight) {
		t.Fatalf("token counts differ: spaced %d, tight %d", len(spaced), len(tight))
	}
	for i := range spaced {
		if spaced[i] != tight[i] {
			t.Errorf("token %d differs: spaced {%v %q}, tight {%v %q}", i, spaced[i].kind, spaced[i].value, tight[i].kind, tight[i].value)
		}
	}
	if len(tight) != 3 {
		t.Errorf("type='txt' produced %d tokens, want 3", len(tight))
	}
}

func TestLexTwoCharacterOperatorsAreGreedy(t *testing.T) {
	tests := []struct {
		input string
		want  []tok
	}{
		{"count(child)<=2", []tok{{query.TokenIdent, "count"}, {query.TokenLParen, "("}, {query.TokenIdent, "child"}, {query.TokenRParen, ")"}, {query.TokenOperator, "<="}, {query.TokenIdent, "2"}}},
		{"a>=1", []tok{{query.TokenIdent, "a"}, {query.TokenOperator, ">="}, {query.TokenIdent, "1"}}},
		{"a!='b'", []tok{{query.TokenIdent, "a"}, {query.TokenOperator, "!="}, {query.TokenString, "b"}}},
		{"a<1", []tok{{query.TokenIdent, "a"}, {query.TokenOperator, "<"}, {query.TokenIdent, "1"}}},
		{"a>1", []tok{{query.TokenIdent, "a"}, {query.TokenOperator, ">"}, {query.TokenIdent, "1"}}},
		{"a=1", []tok{{query.TokenIdent, "a"}, {query.TokenOperator, "="}, {query.TokenIdent, "1"}}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assertTokens(t, tt.input, tt.want)
		})
	}
}

func TestLexLoneBangIsAnOperatorForTheParserToReject(t *testing.T) {
	got := lexTokens(t, "a ! b")

	if len(got) != 3 {
		t.Fatalf("got %d tokens, want 3", len(got))
	}
	if got[1].kind != query.TokenOperator || got[1].value != "!" {
		t.Errorf("token 1 = {%v %q}, want an operator \"!\"; validity is the parser's job", got[1].kind, got[1].value)
	}
}

func TestLexQuotedStringsKeepTheirContents(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"spaces preserved", "'my folder'", "my folder"},
		{"empty string", "''", ""},
		{"single space", "' '", " "},
		{"percent wildcards", "'%report%'", "%report%"},
		{"star", "'*.log'", "*.log"},
		{"path with slashes", "'~/Documents/goupp'", "~/Documents/goupp"},
		{"operators inside quotes are literal", "'a=b<c>d!e'", "a=b<c>d!e"},
		{"parens inside quotes are literal", "'count(child)'", "count(child)"},
		{"semicolon inside quotes is literal", "'a;b'", "a;b"},
		{"unicode", "'日本語'", "日本語"},
		{"leading and trailing spaces", "'  padded  '", "  padded  "},
		{"tab inside", "'a\tb'", "a\tb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lexTokens(t, tt.input)
			if len(got) != 1 {
				t.Fatalf("got %d tokens, want 1", len(got))
			}
			if got[0].kind != query.TokenString {
				t.Errorf("kind = %v, want string", got[0].kind)
			}
			if got[0].value != tt.want {
				t.Errorf("value = %q, want %q", got[0].value, tt.want)
			}
		})
	}
}

func TestLexPreservesCase(t *testing.T) {
	got := lexTokens(t, "FILES FROM 'Documents' WHERE type = '.TXT'")

	if got[0].value != "FILES" {
		t.Errorf("keyword case not preserved: %q; folding is the parser's job", got[0].value)
	}
	for _, tk := range got {
		if tk.kind == query.TokenString && tk.value == ".txt" {
			t.Error("string value was lowercased; paths and values are case-sensitive")
		}
	}
	if got[len(got)-1].value != ".TXT" {
		t.Errorf("value = %q, want \".TXT\"", got[len(got)-1].value)
	}
}

func TestLexCollapsesWhitespaceRuns(t *testing.T) {
	tests := []string{
		"  files\tfrom\n'a'",
		"  files from 'a'  ",
		"files\t\t from \r\n 'a'",
	}

	want := []tok{
		{query.TokenIdent, "files"},
		{query.TokenIdent, "from"},
		{query.TokenString, "a"},
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			assertTokens(t, input, want)
		})
	}
}

func TestLexWhitespaceInsideParens(t *testing.T) {
	assertTokens(t, "count( child )", []tok{
		{query.TokenIdent, "count"},
		{query.TokenLParen, "("},
		{query.TokenIdent, "child"},
		{query.TokenRParen, ")"},
	})
}

func TestLexSemicolonTerminatesStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []tok
	}{
		{"trailing", "files from 'a';", []tok{{query.TokenIdent, "files"}, {query.TokenIdent, "from"}, {query.TokenString, "a"}}},
		{"trailing with space", "files from 'a' ;", []tok{{query.TokenIdent, "files"}, {query.TokenIdent, "from"}, {query.TokenString, "a"}}},
		{"repeated", "files from 'a';;;", []tok{{query.TokenIdent, "files"}, {query.TokenIdent, "from"}, {query.TokenString, "a"}}},
		{"text after is ignored", "files from 'a'; junk here", []tok{{query.TokenIdent, "files"}, {query.TokenIdent, "from"}, {query.TokenString, "a"}}},
		{"leading semicolon yields nothing", ";select files", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.input, tt.want)
		})
	}
}

func TestLexEmptyAndWhitespaceOnlyInput(t *testing.T) {
	for _, input := range []string{"", " ", "\t", "\n", "   \t\r\n  "} {
		got, err := query.NewLexer().Lex(input)
		if err != nil {
			t.Fatalf("Lex(%q) error = %v", input, err)
		}
		if len(got) != 1 || got[0].Kind != query.TokenEOF {
			t.Errorf("Lex(%q) = %v, want a lone EOF token", input, got)
		}
	}
}

func TestLexUnclosedQuote(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"open at end", "files from 'Documents"},
		{"lone quote", "'"},
		{"odd number of quotes", "files from 'a' where name = 'b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := query.NewLexer().Lex(tt.input)
			if err == nil {
				t.Fatalf("Lex(%q) accepted an unclosed quote", tt.input)
			}
			if !oerr.Is(err, oerr.KindUnclosedQuote) {
				t.Errorf("error kind = %v, want unclosed_quote", err)
			}
		})
	}
}

func TestLexUnclosedQuoteReportsTheFragment(t *testing.T) {
	_, err := query.NewLexer().Lex("files from 'Documents")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "Documents") {
		t.Errorf("error does not quote the unterminated fragment: %v", err)
	}
}

func TestLexTokenPositions(t *testing.T) {
	got, err := query.NewLexer().Lex("files from 'a'")
	if err != nil {
		t.Fatal(err)
	}

	wantPos := []int{0, 6, 11, 14}
	if len(got) != len(wantPos) {
		t.Fatalf("got %d tokens, want %d", len(got), len(wantPos))
	}
	for i, want := range wantPos {
		if got[i].Pos != want {
			t.Errorf("token %d (%q) Pos = %d, want %d", i, got[i].Value, got[i].Pos, want)
		}
	}
}

func TestLexBareWordsAcceptPathCharacters(t *testing.T) {
	tests := []string{
		"~/Documents",
		"./relative",
		"/absolute/path",
		"file-with-dashes.txt",
		"file_with_underscores",
		"%wildcard%",
		"*.log",
		"dir.with.dots",
		"10",
		"日本語",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got := lexTokens(t, input)
			if len(got) != 1 {
				t.Fatalf("Lex(%q) produced %d tokens, want 1: %v", input, len(got), got)
			}
			if got[0].kind != query.TokenIdent || got[0].value != input {
				t.Errorf("Lex(%q) = {%v %q}, want a single ident", input, got[0].kind, got[0].value)
			}
		})
	}
}

func TestLexAdjacentQuotedStrings(t *testing.T) {
	assertTokens(t, "'a''b'", []tok{
		{query.TokenString, "a"},
		{query.TokenString, "b"},
	})
}

func TestLexLongInput(t *testing.T) {
	long := "files from '" + strings.Repeat("a", 100000) + "'"

	got := lexTokens(t, long)
	if len(got) != 3 {
		t.Fatalf("got %d tokens, want 3", len(got))
	}
	if len(got[2].value) != 100000 {
		t.Errorf("long path truncated to %d bytes", len(got[3].value))
	}
}

func TestTokenIsKeyword(t *testing.T) {
	tests := []struct {
		token query.Token
		match string
		want  bool
	}{
		{query.Token{Kind: query.TokenIdent, Value: "select"}, "select", true},
		{query.Token{Kind: query.TokenIdent, Value: "SELECT"}, "select", true},
		{query.Token{Kind: query.TokenIdent, Value: "Select"}, "select", true},
		{query.Token{Kind: query.TokenIdent, Value: "select"}, "from", false},
		{query.Token{Kind: query.TokenString, Value: "select"}, "select", false},
		{query.Token{Kind: query.TokenOperator, Value: "="}, "=", false},
		{query.Token{Kind: query.TokenIdent, Value: ""}, "", true},
	}

	for _, tt := range tests {
		if got := tt.token.IsKeyword(tt.match); got != tt.want {
			t.Errorf("Token{%v %q}.IsKeyword(%q) = %v, want %v", tt.token.Kind, tt.token.Value, tt.match, got, tt.want)
		}
	}
}

func FuzzLex(f *testing.F) {
	seeds := []string{
		"files from 'Documents'",
		"folders from 'src' where count(child) <= 2",
		"type='txt'",
		"'unterminated",
		";",
		"",
		"a!b",
		"''''",
		"count( child )>=1",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	lexer := query.NewLexer()

	f.Fuzz(func(t *testing.T, input string) {
		tokens, err := lexer.Lex(input)
		if err != nil {
			if !oerr.Is(err, oerr.KindUnclosedQuote) {
				t.Fatalf("Lex(%q) returned a non-oerr error: %v", input, err)
			}
			if tokens != nil {
				t.Errorf("Lex(%q) returned both tokens and an error", input)
			}
			return
		}

		if len(tokens) == 0 {
			t.Fatalf("Lex(%q) returned no tokens and no error", input)
		}
		if tokens[len(tokens)-1].Kind != query.TokenEOF {
			t.Errorf("Lex(%q) did not end with EOF", input)
		}
		for i, tk := range tokens[:len(tokens)-1] {
			if tk.Pos < 0 || tk.Pos > len(input) {
				t.Errorf("token %d Pos = %d, outside input of length %d", i, tk.Pos, len(input))
			}
			if tk.Kind == query.TokenEOF {
				t.Errorf("token %d is EOF before the end", i)
			}
		}
	})
}
