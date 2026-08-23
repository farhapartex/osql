package query

import "strings"

type TokenKind int

const (
	TokenEOF TokenKind = iota
	TokenIdent
	TokenString
	TokenOperator
	TokenLParen
	TokenRParen
)

var tokenKindNames = map[TokenKind]string{
	TokenEOF:      "eof",
	TokenIdent:    "ident",
	TokenString:   "string",
	TokenOperator: "operator",
	TokenLParen:   "lparen",
	TokenRParen:   "rparen",
}

func (k TokenKind) String() string {
	if name, ok := tokenKindNames[k]; ok {
		return name
	}
	return "unknown"
}

type Token struct {
	Kind  TokenKind
	Value string
	Pos   int
}

func (t Token) IsKeyword(keyword string) bool {
	return t.Kind == TokenIdent && strings.EqualFold(t.Value, keyword)
}
