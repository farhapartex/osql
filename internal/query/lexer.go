package query

import (
	"strings"

	"github.com/farhapartex/osql/internal/oerr"
)

const quote = '\''

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
			end := strings.IndexByte(input[i+1:], quote)
			if end < 0 {
				return nil, oerr.UnclosedQuote(input[i+1:])
			}
			tokens = append(tokens, Token{Kind: TokenString, Value: input[i+1 : i+1+end], Pos: i})
			i += end + 2

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
