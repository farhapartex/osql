package oerr

import (
	"errors"
	"fmt"
	"strings"
)

type Kind int

const (
	KindFolderMissing Kind = iota
	KindPathIsFile
	KindNoPermission
	KindUnknownVerb
	KindMissingTarget
	KindSingularTarget
	KindUnknownTarget
	KindMissingFrom
	KindMissingPath
	KindUnknownField
	KindWrongOperator
	KindCountChildOnFiles
	KindCountChildNonNumeric
	KindUnexpectedInput
	KindIncompleteQuery
	KindUnclosedQuote
)

var kindNames = map[Kind]string{
	KindFolderMissing:        "folder_missing",
	KindPathIsFile:           "path_is_file",
	KindNoPermission:         "no_permission",
	KindUnknownVerb:          "unknown_verb",
	KindMissingTarget:        "missing_target",
	KindSingularTarget:       "singular_target",
	KindUnknownTarget:        "unknown_target",
	KindMissingFrom:          "missing_from",
	KindMissingPath:          "missing_path",
	KindUnknownField:         "unknown_field",
	KindWrongOperator:        "wrong_operator",
	KindCountChildOnFiles:    "count_child_on_files",
	KindCountChildNonNumeric: "count_child_non_numeric",
	KindUnexpectedInput:      "unexpected_input",
	KindIncompleteQuery:      "incomplete_query",
	KindUnclosedQuote:        "unclosed_quote",
}

func (k Kind) String() string {
	if name, ok := kindNames[k]; ok {
		return name
	}
	return "unknown"
}

type Error struct {
	Kind    Kind
	message string
}

func (e *Error) Error() string {
	return e.message
}

func newError(kind Kind, format string, args ...any) *Error {
	return &Error{Kind: kind, message: fmt.Sprintf(format, args...)}
}

func Is(err error, kind Kind) bool {
	var oe *Error
	if errors.As(err, &oe) {
		return oe.Kind == kind
	}
	return false
}

func FolderMissing(path string) *Error {
	return newError(KindFolderMissing, "I couldn't find a folder at '%s'. Check the path and try again.", path)
}

func PathIsFile(path string) *Error {
	return newError(KindPathIsFile, "'%s' is a file, not a folder. Try: select files from 'Documents'", path)
}

func NoPermission(path string) *Error {
	return newError(KindNoPermission, "I don't have permission to read '%s'.", path)
}

func UnknownVerb(got string, known []string) *Error {
	if suggestion, ok := Suggest(got, known); ok {
		return newError(KindUnknownVerb, "I don't know how to \"%s\". Did you mean \"%s\"?", got, suggestion)
	}
	return newError(KindUnknownVerb, "I don't know how to \"%s\".", got)
}

func MissingTarget() *Error {
	return newError(KindMissingTarget, "I need \"files\", \"folders\", or \"all\" after \"select\" — for example: select files from 'Documents'")
}

func UnexpectedInput(got string) *Error {
	return newError(KindUnexpectedInput, "I don't understand \"%s\" here. Try: select files from 'Documents'", got)
}

func IncompleteAfter(keyword string) *Error {
	return newError(KindIncompleteQuery, "The query ends after \"%s\". I need more — for example: select files from 'Documents' where name = 'notes.txt'", keyword)
}

func SingularTarget(got string) *Error {
	return newError(KindSingularTarget, "Use \"%ss\", not \"%s\" — for example: select %ss from 'Documents'", got, got, got)
}

func UnknownTarget(got string) *Error {
	return newError(KindUnknownTarget, "I can select \"files\", \"folders\", or \"all\" — not \"%s\".", got)
}

func MissingFrom() *Error {
	return newError(KindMissingFrom, "I need \"from\" before the folder — for example: select files from 'Documents'")
}

func MissingPath() *Error {
	return newError(KindMissingPath, "I need a folder after \"from\" — for example: select files from 'Documents'")
}

func UnknownField(got string, known []string) *Error {
	return newError(KindUnknownField, "I don't know the field \"%s\". I understand: %s", got, strings.Join(known, ", "))
}

func WrongOperator(field string, allowed []string) *Error {
	base := fmt.Sprintf("\"%s\" only works with %s.", field, joinWithAnd(allowed))
	if field == "name_like" {
		return newError(KindWrongOperator, "%s", base)
	}
	return newError(KindWrongOperator, "%s For patterns use name_like: select files from 'Documents' where name_like = '%%report%%'", base)
}

func CountChildOnFiles() *Error {
	return newError(KindCountChildOnFiles, "count(child) describes folders, not files. Try: select folders from 'Documents' where count(child) > 10")
}

func CountChildNonNumeric() *Error {
	return newError(KindCountChildNonNumeric, "count(child) needs a number — for example: count(child) > 10")
}

func UnclosedQuote(fragment string) *Error {
	return newError(KindUnclosedQuote, "This quote is never closed: '%s — add a closing '", fragment)
}

func joinWithAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
	}
}
