package query

import (
	"strings"

	"github.com/farhapartex/osql/internal/oerr"
)

const (
	VerbSelect       = "select"
	LegacyVerb       = "select"
	KeywordFrom      = "from"
	KeywordRecursive = "recursive"
	KeywordWhere     = "where"
	KeywordAnd       = "and"
	KeywordCount     = "count"
	KeywordChild     = "child"
)

var singularTargets = []string{"file", "folder"}

var targetNamesInOrder = []string{"all", "files", "folders"}

var structuralKeywords = []string{KeywordFrom, KeywordRecursive, KeywordWhere, KeywordAnd}

type PredicateValidator interface {
	Validate(p Predicate, target Target) error
}

type stdParser struct {
	validator PredicateValidator
}

func NewParser(validator PredicateValidator) Parser {
	return stdParser{validator: validator}
}

type cursor struct {
	tokens []Token
	pos    int
}

func (c *cursor) peek() Token {
	if c.pos >= len(c.tokens) {
		return Token{Kind: TokenEOF}
	}
	return c.tokens[c.pos]
}

func (c *cursor) next() Token {
	t := c.peek()
	if c.pos < len(c.tokens) {
		c.pos++
	}
	return t
}

func (c *cursor) atEOF() bool {
	return c.peek().Kind == TokenEOF
}

func (p stdParser) Parse(tokens []Token) (*Statement, error) {
	c := &cursor{tokens: tokens}

	if c.atEOF() {
		return nil, oerr.MissingTarget()
	}
	if c.peek().IsKeyword(LegacyVerb) {
		return nil, oerr.NoVerbNeeded(c.peek().Value)
	}

	target, err := parseTarget(c)
	if err != nil {
		return nil, err
	}

	if c.atEOF() {
		return nil, oerr.MissingFrom()
	}
	if !c.peek().IsKeyword(KeywordFrom) {
		return nil, oerr.MissingFrom()
	}
	c.next()

	path, err := parsePath(c)
	if err != nil {
		return nil, err
	}

	stmt := &Statement{Verb: VerbSelect, Target: target, Path: path}

	if c.peek().IsKeyword(KeywordRecursive) {
		c.next()
		stmt.Recursive = true
	}

	if c.peek().IsKeyword(KeywordWhere) {
		c.next()
		stmt.Predicates, err = parseCondition(c)
		if err != nil {
			return nil, err
		}
	}

	if !c.atEOF() {
		return nil, oerr.UnexpectedInput(c.peek().Value)
	}

	if p.validator != nil {
		for _, pred := range stmt.Predicates {
			if err := p.validator.Validate(pred, target); err != nil {
				return nil, err
			}
		}
	}

	return stmt, nil
}

func parseTarget(c *cursor) (Target, error) {
	t := c.peek()

	if t.Kind == TokenEOF || isStructuralKeyword(t) {
		return TargetAll, oerr.MissingTarget()
	}
	if t.Kind != TokenIdent {
		return TargetAll, oerr.MissingTarget()
	}

	for _, singular := range singularTargets {
		if t.IsKeyword(singular) {
			c.next()
			return TargetAll, oerr.SingularTarget(singular)
		}
	}

	for _, name := range targetNamesInOrder {
		if t.IsKeyword(name) {
			c.next()
			target, _ := ParseTarget(name)
			return target, nil
		}
	}

	c.next()
	return TargetAll, oerr.UnknownTarget(t.Value, targetNamesInOrder)
}

func TargetNames() []string {
	return append([]string(nil), targetNamesInOrder...)
}

func parsePath(c *cursor) (string, error) {
	t := c.peek()

	if t.Kind == TokenEOF {
		return "", oerr.MissingPath()
	}
	if t.Kind == TokenString {
		c.next()
		return t.Value, nil
	}
	if t.Kind == TokenIdent && !isStructuralKeyword(t) {
		c.next()
		return t.Value, nil
	}

	return "", oerr.MissingPath()
}

func parseCondition(c *cursor) ([]Predicate, error) {
	predicates := make([]Predicate, 0, 2)

	for {
		p, err := parsePredicate(c)
		if err != nil {
			return nil, err
		}
		predicates = append(predicates, p)

		if !c.peek().IsKeyword(KeywordAnd) {
			return predicates, nil
		}
		c.next()
		if c.atEOF() {
			return nil, oerr.IncompleteAfter(KeywordAnd)
		}
	}
}

func parsePredicate(c *cursor) (Predicate, error) {
	if c.atEOF() {
		return Predicate{}, oerr.IncompleteAfter(KeywordWhere)
	}

	field, err := parseField(c)
	if err != nil {
		return Predicate{}, err
	}

	op := c.peek()
	if op.Kind == TokenEOF {
		return Predicate{}, oerr.IncompleteAfter(field)
	}
	if op.Kind != TokenOperator {
		return Predicate{}, oerr.UnexpectedInput(op.Value)
	}
	c.next()

	value := c.peek()
	if value.Kind == TokenEOF {
		return Predicate{}, oerr.IncompleteAfter(op.Value)
	}
	if value.Kind != TokenString && value.Kind != TokenIdent {
		return Predicate{}, oerr.UnexpectedInput(value.Value)
	}
	c.next()

	return Predicate{Field: field, Op: op.Value, Value: value.Value}, nil
}

func parseField(c *cursor) (string, error) {
	t := c.peek()
	if t.Kind != TokenIdent {
		return "", oerr.UnexpectedInput(t.Value)
	}
	c.next()

	if !t.IsKeyword(KeywordCount) || c.peek().Kind != TokenLParen {
		return strings.ToLower(t.Value), nil
	}
	c.next()

	arg := c.peek()
	if arg.Kind != TokenIdent {
		return "", oerr.UnexpectedInput(arg.Value)
	}
	c.next()

	if c.peek().Kind != TokenRParen {
		return "", oerr.UnexpectedInput(c.peek().Value)
	}
	c.next()

	return strings.ToLower(t.Value) + "(" + strings.ToLower(arg.Value) + ")", nil
}

func isStructuralKeyword(t Token) bool {
	for _, kw := range structuralKeywords {
		if t.IsKeyword(kw) {
			return true
		}
	}
	return false
}
