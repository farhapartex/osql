package query

import (
	"strings"

	"github.com/farhapartex/osql/internal/oerr"
)

const (
	VerbSelect       = "select"
	VerbCount        = "count"
	VerbOpen         = "open"
	VerbNew          = "new"
	VerbSummary      = "summary"
	VerbDelete       = "delete"
	VerbApps         = "apps"
	KeywordPermanent = "permanently"
	KeywordWith      = "with"
	KeywordSkipped   = "skipped"
	KeywordSize      = "size"
	KeywordData      = "data"
	LegacyVerb       = "select"
	KeywordFrom      = "from"
	KeywordRecursive = "recursive"
	KeywordWhere     = "where"
	KeywordAnd       = "and"
	KeywordCount     = "count"
	KeywordChild     = "child"
)

var singularTargets = []string{"file", "folder"}

var targetNamesInOrder = []string{"all", "files", "folders", "apps"}

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

func (c *cursor) peekAt(offset int) Token {
	if c.pos+offset >= len(c.tokens) {
		return Token{Kind: TokenEOF}
	}
	return c.tokens[c.pos+offset]
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

	if c.peek().IsKeyword(VerbOpen) {
		c.next()
		return parseOpen(c)
	}

	if c.peek().IsKeyword(VerbNew) {
		c.next()
		return parseNew(c)
	}

	if c.peek().IsKeyword(VerbSummary) {
		c.next()
		return parseSummary(c)
	}

	if c.peek().IsKeyword(VerbDelete) {
		c.next()
		return parseDelete(c)
	}

	if c.peek().IsKeyword(VerbApps) {
		c.next()
		return parseAppsTail(c, VerbApps)
	}

	verb := VerbSelect
	if c.peek().IsKeyword(KeywordCount) && c.peekAt(1).Kind == TokenLParen {
		verb = VerbCount
		c.next()
		c.next()
	}

	target, err := parseTarget(c)
	if err != nil {
		return nil, err
	}

	if verb == VerbCount {
		if c.peek().Kind != TokenRParen {
			return nil, oerr.UnclosedCount()
		}
		c.next()
	}

	if target == TargetApps {
		return parseAppsTail(c, verb)
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

	stmt := &Statement{Verb: verb, Target: target, Path: path}

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

func parseSummaryApps(c *cursor) (*Statement, error) {
	if c.peek().IsKeyword(KeywordFrom) {
		return nil, oerr.AppsNeedNoPath()
	}
	if c.peek().IsKeyword(KeywordRecursive) {
		return nil, oerr.AppsNotRecursive()
	}
	if c.peek().IsKeyword(KeywordWhere) {
		return nil, oerr.SummaryTakesNoWhere()
	}
	if !c.atEOF() {
		return nil, oerr.UnexpectedInput(c.peek().Value)
	}
	return &Statement{Verb: VerbSummary, Target: TargetApps}, nil
}

func parseAppsTail(c *cursor, verb string) (*Statement, error) {
	if c.peek().IsKeyword(KeywordFrom) {
		return nil, oerr.AppsNeedNoPath()
	}
	if c.peek().IsKeyword(KeywordRecursive) {
		return nil, oerr.AppsNotRecursive()
	}

	stmt := &Statement{Verb: verb, Target: TargetApps}

	if c.peek().IsKeyword(KeywordWith) {
		c.next()
		if !c.peek().IsKeyword(KeywordSize) {
			return nil, oerr.WithNeedsSize(c.peek().Value)
		}
		c.next()
		if verb == VerbCount {
			return nil, oerr.CountHasNoSize()
		}
		stmt.WithSize = true
	}

	if c.peek().IsKeyword(KeywordWhere) {
		c.next()
		predicates, err := parseCondition(c)
		if err != nil {
			return nil, err
		}
		stmt.Predicates = predicates
	}

	if c.peek().IsKeyword(KeywordWith) {
		return nil, oerr.WithSizeComesFirst()
	}

	if !c.atEOF() {
		return nil, oerr.UnexpectedInput(c.peek().Value)
	}
	return stmt, nil
}

func parseOpen(c *cursor) (*Statement, error) {
	path, err := parsePath(c)
	if err != nil {
		return nil, oerr.MissingFilePath()
	}
	if !c.atEOF() {
		return nil, oerr.UnexpectedInput(c.peek().Value)
	}
	return &Statement{Verb: VerbOpen, Path: path}, nil
}

func parseDelete(c *cursor) (*Statement, error) {
	word := c.peek()
	if word.Kind != TokenIdent {
		return nil, oerr.MissingDeleteTarget("")
	}

	if kind, ok := ParseNewKind(strings.ToLower(word.Value)); ok {
		c.next()
		return parseDeleteSingle(c, kind)
	}

	if word.IsKeyword(VerbApps) {
		return nil, oerr.CannotDeleteApps()
	}

	target, ok := ParseTarget(strings.ToLower(word.Value))
	if !ok {
		return nil, oerr.MissingDeleteTarget(word.Value)
	}
	c.next()
	return parseDeleteBulk(c, target)
}

func parseDeleteSingle(c *cursor, kind NewKind) (*Statement, error) {
	path, err := parsePath(c)
	if err != nil {
		return nil, oerr.MissingDeletePath(kind.String())
	}

	stmt := &Statement{Verb: VerbDelete, Kind: kind, Path: path, Single: true, Target: TargetAll}
	if err := parsePermanently(c, stmt); err != nil {
		return nil, err
	}
	return stmt, nil
}

func parseDeleteBulk(c *cursor, target Target) (*Statement, error) {
	if !c.peek().IsKeyword(KeywordFrom) {
		return nil, oerr.MissingFrom()
	}
	c.next()

	path, err := parsePath(c)
	if err != nil {
		return nil, err
	}

	stmt := &Statement{Verb: VerbDelete, Target: target, Path: path}

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
	if err := parsePermanently(c, stmt); err != nil {
		return nil, err
	}
	return stmt, nil
}

func parsePermanently(c *cursor, stmt *Statement) error {
	if c.peek().IsKeyword(KeywordPermanent) {
		c.next()
		stmt.Permanent = true
	}
	if !c.atEOF() {
		return oerr.UnexpectedInput(c.peek().Value)
	}
	return nil
}

func parseSummary(c *cursor) (*Statement, error) {
	if c.atEOF() {
		return nil, oerr.MissingFrom()
	}
	if c.peek().IsKeyword(VerbApps) {
		c.next()
		return parseSummaryApps(c)
	}
	if !c.peek().IsKeyword(KeywordFrom) {
		return nil, oerr.MissingFrom()
	}
	c.next()

	path, err := parsePath(c)
	if err != nil {
		return nil, err
	}

	stmt := &Statement{Verb: VerbSummary, Path: path, Target: TargetAll}

	if c.peek().IsKeyword(KeywordRecursive) {
		c.next()
		stmt.Recursive = true
	}

	if c.peek().IsKeyword(KeywordWith) {
		c.next()
		if !c.peek().IsKeyword(KeywordSkipped) {
			return nil, oerr.WithNeedsSkipped(c.peek().Value)
		}
		c.next()
		stmt.IncludeSkipped = true
	}

	if !c.atEOF() {
		return nil, oerr.UnexpectedInput(c.peek().Value)
	}
	return stmt, nil
}

func parseNew(c *cursor) (*Statement, error) {
	kindToken := c.peek()
	if kindToken.Kind != TokenIdent {
		return nil, oerr.MissingNewTarget("")
	}
	kind, ok := ParseNewKind(strings.ToLower(kindToken.Value))
	if !ok {
		if _, plural := ParseTarget(strings.ToLower(kindToken.Value)); plural {
			return nil, oerr.SingularNewTarget(kindToken.Value)
		}
		return nil, oerr.MissingNewTarget(kindToken.Value)
	}
	c.next()

	path, err := parsePath(c)
	if err != nil {
		return nil, oerr.MissingNewPath(kind.String())
	}

	stmt := &Statement{Verb: VerbNew, Kind: kind, Path: path}

	if c.peek().IsKeyword(KeywordData) {
		c.next()
		if c.peek().Kind != TokenOperator || c.peek().Value != "=" {
			return nil, oerr.UnexpectedInput(c.peek().Value)
		}
		c.next()

		value := c.peek()
		if value.Kind != TokenString && value.Kind != TokenIdent {
			return nil, oerr.MissingDataValue()
		}
		c.next()

		if kind == NewFolder {
			return nil, oerr.DataOnFolder()
		}
		stmt.Data = value.Value
		stmt.HasData = true
	}

	if !c.atEOF() {
		return nil, oerr.UnexpectedInput(c.peek().Value)
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
