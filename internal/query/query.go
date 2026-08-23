package query

type Lexer interface {
	Lex(input string) ([]Token, error)
}

type Parser interface {
	Parse(tokens []Token) (*Statement, error)
}
