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
	KindPathIsFolder
	KindFileMissing
	KindBinaryFile
	KindAlreadyExists
	KindDataOnFolder
	KindMissingNewTarget
	KindSingularNewTarget
	KindMissingNewPath
	KindMissingDataValue
	KindCannotCreate
	KindCannotRead
	KindNoPermission
	KindOutsideRoot
	KindUnknownVerb
	KindNoVerbNeeded
	KindMissingTarget
	KindUnclosedCount
	KindSingularTarget
	KindUnknownTarget
	KindMissingFrom
	KindWithNeedsSkipped
	KindMissingPath
	KindMissingFilePath
	KindUnknownField
	KindWrongOperator
	KindCountChildOnFiles
	KindCountChildNonNumeric
	KindUnexpectedInput
	KindIncompleteQuery
	KindUnclosedQuote
	KindBadEscape
)

var kindNames = map[Kind]string{
	KindFolderMissing:        "folder_missing",
	KindPathIsFile:           "path_is_file",
	KindPathIsFolder:         "path_is_folder",
	KindFileMissing:          "file_missing",
	KindBinaryFile:           "binary_file",
	KindAlreadyExists:        "already_exists",
	KindDataOnFolder:         "data_on_folder",
	KindMissingNewTarget:     "missing_new_target",
	KindSingularNewTarget:    "singular_new_target",
	KindMissingNewPath:       "missing_new_path",
	KindMissingDataValue:     "missing_data_value",
	KindCannotCreate:         "cannot_create",
	KindCannotRead:           "cannot_read",
	KindNoPermission:         "no_permission",
	KindOutsideRoot:          "outside_root",
	KindUnknownVerb:          "unknown_verb",
	KindNoVerbNeeded:         "no_verb_needed",
	KindMissingTarget:        "missing_target",
	KindUnclosedCount:        "unclosed_count",
	KindSingularTarget:       "singular_target",
	KindUnknownTarget:        "unknown_target",
	KindMissingFrom:          "missing_from",
	KindWithNeedsSkipped:     "with_needs_skipped",
	KindMissingPath:          "missing_path",
	KindMissingFilePath:      "missing_file_path",
	KindUnknownField:         "unknown_field",
	KindWrongOperator:        "wrong_operator",
	KindCountChildOnFiles:    "count_child_on_files",
	KindCountChildNonNumeric: "count_child_non_numeric",
	KindUnexpectedInput:      "unexpected_input",
	KindIncompleteQuery:      "incomplete_query",
	KindUnclosedQuote:        "unclosed_quote",
	KindBadEscape:            "bad_escape",
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
	return newError(KindPathIsFile, "'%s' is a file, not a folder. Try: files from 'Documents'", path)
}

func PathIsFolder(path string) *Error {
	return newError(KindPathIsFolder, "'%s' is a folder, not a file. Try: open '%s/notes.txt'", path, path)
}

func FileMissing(path string) *Error {
	return newError(KindFileMissing, "I couldn't find a file at '%s'. Check the path and try again.", path)
}

func BinaryFile(path string) *Error {
	return newError(KindBinaryFile, "'%s' looks like a binary file, so I won't print it. open only shows text.", path)
}

func AlreadyExists(path string) *Error {
	return newError(KindAlreadyExists, "'%s' already exists. Nothing was changed.", path)
}

func DataOnFolder() *Error {
	return newError(KindDataOnFolder, "A folder can't hold data. Drop the data part, or use: new file 'notes.txt' data='hello'")
}

func MissingNewTarget(got string) *Error {
	if got == "" {
		return newError(KindMissingNewTarget, "I need \"file\" or \"folder\" after \"new\" — for example: new file 'notes.txt'")
	}
	return newError(KindMissingNewTarget, "I can make a \"file\" or a \"folder\" — not \"%s\".", got)
}

func SingularNewTarget(got string) *Error {
	singular := singularOf(got)
	return newError(KindSingularNewTarget, "Use \"%s\", not \"%s\" — you make one thing at a time: new %s '%s'", singular, got, singular, exampleNameFor(singular))
}

func MissingNewPath(kind string) *Error {
	return newError(KindMissingNewPath, "I need a path after \"new %s\" — for example: new %s '%s'", kind, kind, exampleNameFor(kind))
}

func exampleNameFor(kind string) string {
	if kind == "folder" {
		return "reports"
	}
	return "notes.txt"
}

func MissingDataValue() *Error {
	return newError(KindMissingDataValue, "data needs a value in quotes — for example: data='hello there'")
}

func singularOf(plural string) string {
	return strings.TrimSuffix(plural, "s")
}

func CannotCreate(path string, reason string) *Error {
	return newError(KindCannotCreate, "I couldn't create '%s': %s", path, reason)
}

func CannotRead(path string) *Error {
	return newError(KindCannotRead, "I couldn't read '%s'. The file may have changed while I was reading it.", path)
}

func NoPermission(path string) *Error {
	return newError(KindNoPermission, "I don't have permission to read '%s'.", path)
}

func OutsideRoot(input, root string) *Error {
	return newError(KindOutsideRoot, "I can only look inside '%s'. '%s' points outside it.", root, input)
}

func UnknownVerb(got string, known []string) *Error {
	if suggestion, ok := Suggest(got, known); ok {
		return newError(KindUnknownVerb, "I don't know how to \"%s\". Did you mean \"%s\"?", got, suggestion)
	}
	return newError(KindUnknownVerb, "I don't know how to \"%s\".", got)
}

func MissingTarget() *Error {
	return newError(KindMissingTarget, "I need \"files\", \"folders\", or \"all\" to start — for example: files from 'Documents'")
}

func UnclosedCount() *Error {
	return newError(KindUnclosedCount, "count( needs a closing ) — for example: count(files) from 'Documents'")
}

func UnexpectedInput(got string) *Error {
	return newError(KindUnexpectedInput, "I don't understand \"%s\" here. Try: files from 'Documents'", got)
}

func IncompleteAfter(keyword string) *Error {
	return newError(KindIncompleteQuery, "The query ends after \"%s\". I need more — for example: files from 'Documents' where name = 'notes.txt'", keyword)
}

func NoVerbNeeded(got string) *Error {
	return newError(KindNoVerbNeeded, "Queries don't need \"%s\" — start with what you want: files from 'Documents'", got)
}

func SingularTarget(got string) *Error {
	return newError(KindSingularTarget, "Use \"%ss\", not \"%s\" — for example: %ss from 'Documents'", got, got, got)
}

func UnknownTarget(got string, known []string) *Error {
	if suggestion, ok := Suggest(got, known); ok {
		return newError(KindUnknownTarget, "I can list \"files\", \"folders\", or \"all\" — not \"%s\". Did you mean \"%s\"?", got, suggestion)
	}
	return newError(KindUnknownTarget, "I can list \"files\", \"folders\", or \"all\" — not \"%s\".", got)
}

func MissingFrom() *Error {
	return newError(KindMissingFrom, "I need \"from\" before the folder — for example: files from 'Documents'")
}

func WithNeedsSkipped(got string) *Error {
	if got == "" {
		return newError(KindWithNeedsSkipped, "\"with\" needs \"skipped\" — for example: summary from 'Documents' recursive with skipped")
	}
	return newError(KindWithNeedsSkipped, "I only know \"with skipped\", not \"with %s\".", got)
}

func MissingPath() *Error {
	return newError(KindMissingPath, "I need a folder after \"from\" — for example: files from 'Documents'")
}

func MissingFilePath() *Error {
	return newError(KindMissingFilePath, "I need a file after \"open\" — for example: open 'notes.txt'")
}

func UnknownField(got string, known []string) *Error {
	return newError(KindUnknownField, "I don't know the field \"%s\". I understand: %s", got, strings.Join(known, ", "))
}

func WrongOperator(field string, allowed []string) *Error {
	base := fmt.Sprintf("\"%s\" only works with %s.", field, joinWithAnd(allowed))
	if field == "name_like" {
		return newError(KindWrongOperator, "%s", base)
	}
	return newError(KindWrongOperator, "%s For patterns use name_like: files from 'Documents' where name_like = '%%report%%'", base)
}

func CountChildOnFiles() *Error {
	return newError(KindCountChildOnFiles, "count(child) describes folders, not files. Try: folders from 'Documents' where count(child) > 10")
}

func CountChildNonNumeric() *Error {
	return newError(KindCountChildNonNumeric, "count(child) needs a number — for example: count(child) > 10")
}

func BadEscape(got string) *Error {
	return newError(KindBadEscape, "I don't know the escape \"\\%s\". I understand: \\n, \\t, \\r, \\\\ and \\'", got)
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
