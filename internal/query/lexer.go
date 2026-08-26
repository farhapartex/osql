package query

import (
	"strings"
	"unicode/utf8"

	"github.com/farhapartex/osql/internal/oerr"
)

const (
	quote  = '\''
	escape = '\\'
)

type stdLexer struct{}

func NewLexer() Lexer {
	return stdLexer{}
}

func (stdLexer) Lex(input string) ([]Token, error) {
	tokens := make([]Token, 0, len(input)/4+1)

	i := 0
	for i < len(input) {
		c := input[i]

		switch {
		case isSpace(c):
			i++

		case c == ';':
			i = len(input)

		case c == quote:
			value, width, err := scanString(input, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, Token{Kind: TokenString, Value: value, Pos: i})
			i += width

		case c == '(':
			tokens = append(tokens, Token{Kind: TokenLParen, Value: "(", Pos: i})
			i++

		case c == ')':
			tokens = append(tokens, Token{Kind: TokenRParen, Value: ")", Pos: i})
			i++

		case isOperatorByte(c):
			width := operatorWidth(input, i)
			tokens = append(tokens, Token{Kind: TokenOperator, Value: input[i : i+width], Pos: i})
			i += width

		default:
			end := i
			for end < len(input) && isBareWordByte(input[end]) {
				end++
			}
			tokens = append(tokens, Token{Kind: TokenIdent, Value: input[i:end], Pos: i})
			i = end
		}
	}

	return append(tokens, Token{Kind: TokenEOF, Pos: len(input)}), nil
}

func scanString(input string, start int) (string, int, error) {
	hasEscape := false

	for i := start + 1; i < len(input); {
		switch input[i] {
		case escape:
			if i+1 >= len(input) {
				return "", 0, oerr.UnclosedQuote(input[start+1:])
			}
			hasEscape = true
			i += 2
		case quote:
			raw := input[start+1 : i]
			width := i - start + 1
			if !hasEscape {
				return raw, width, nil
			}
			value, err := unescape(raw)
			if err != nil {
				return "", 0, err
			}
			return value, width, nil
		default:
			i++
		}
	}

	return "", 0, oerr.UnclosedQuote(input[start+1:])
}

func unescape(raw string) (string, error) {
	var out strings.Builder
	out.Grow(len(raw))

	for i := 0; i < len(raw); i++ {
		if raw[i] != escape {
			out.WriteByte(raw[i])
			continue
		}

		i++
		switch raw[i] {
		case 'n':
			out.WriteByte('\n')
		case 't':
			out.WriteByte('\t')
		case 'r':
			out.WriteByte('\r')
		case escape:
			out.WriteByte(escape)
		case quote:
			out.WriteByte(quote)
		default:
			r, _ := utf8.DecodeRuneInString(raw[i:])
			return "", oerr.BadEscape(string(r))
		}
	}

	return out.String(), nil
}

func operatorWidth(input string, i int) int {
	if i+1 < len(input) && input[i+1] == '=' {
		switch input[i] {
		case '!', '<', '>':
			return 2
		}
	}
	return 1
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\v' || c == '\f'
}

func isOperatorByte(c byte) bool {
	return c == '=' || c == '!' || c == '<' || c == '>'
}

func isBareWordByte(c byte) bool {
	return !isSpace(c) && !isOperatorByte(c) && c != quote && c != '(' && c != ')' && c != ';'
}
