package oerr

import (
	"errors"
	"fmt"
	"io/fs"
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
	KindCannotDelete
	KindRefuseDeleteRoot
	KindDeleteKindMismatch
	KindTrashUnavailable
	KindTrashCrossDevice
	KindMissingDeleteTarget
	KindMissingDeletePath
	KindCannotRead
	KindNoPermission
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
	KindAppsNeedNoPath
	KindAppsNotRecursive
	KindAppsUnavailable
	KindCannotDeleteApps
	KindFieldNotForTarget
	KindWithNeedsSize
	KindWithSizeComesFirst
	KindCountHasNoSize
	KindSummaryTakesNoWhere
	KindRefuseDeleteHere
	KindCannotChangeDir
	KindPwdTakesNoPath
	KindBinaryNotFound
	KindInstalledByPackageManager
	KindCannotRemoveBinary
	KindCannotRemoveData
	KindQueryStopped
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
	KindCannotDelete:         "cannot_delete",
	KindRefuseDeleteRoot:     "refuse_delete_root",
	KindDeleteKindMismatch:   "delete_kind_mismatch",
	KindTrashUnavailable:     "trash_unavailable",
	KindTrashCrossDevice:     "trash_cross_device",
	KindMissingDeleteTarget:  "missing_delete_target",
	KindMissingDeletePath:    "missing_delete_path",
	KindCannotRead:           "cannot_read",
	KindNoPermission:         "no_permission",
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
	KindAppsNeedNoPath:       "apps_need_no_path",
	KindAppsNotRecursive:     "apps_not_recursive",
	KindAppsUnavailable:      "apps_unavailable",
	KindCannotDeleteApps:     "cannot_delete_apps",
	KindFieldNotForTarget:    "field_not_for_target",
	KindWithNeedsSize:        "with_needs_size",
	KindWithSizeComesFirst:   "with_size_comes_first",
	KindCountHasNoSize:       "count_has_no_size",
	KindSummaryTakesNoWhere:  "summary_takes_no_where",
	KindRefuseDeleteHere:     "refuse_delete_here",
	KindCannotChangeDir:      "cannot_change_dir",
	KindPwdTakesNoPath:       "pwd_takes_no_path",

	KindBinaryNotFound:            "binary_not_found",
	KindInstalledByPackageManager: "installed_by_package_manager",
	KindCannotRemoveBinary:        "cannot_remove_binary",
	KindCannotRemoveData:          "cannot_remove_data",
	KindQueryStopped:              "query_stopped",
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

func CannotDelete(path, reason string) *Error {
	return newError(KindCannotDelete, "I couldn't delete '%s': %s", path, reason)
}

func RefuseDeleteRoot(root string) *Error {
	return newError(KindRefuseDeleteRoot, "I won't empty '%s' itself. Name a folder inside it, or add a where clause.", root)
}

func DeleteKindMismatch(path, actual string) *Error {
	other := "folder"
	if actual == "folder" {
		other = "file"
	}
	return newError(KindDeleteKindMismatch, "'%s' is a %s, not a %s. Try: delete %s '%s'", path, actual, other, actual, path)
}

func TrashUnavailable(reason string) *Error {
	return newError(KindTrashUnavailable, "I couldn't reach the trash (%s). Add \"permanently\" to delete for good.", reason)
}

func TrashCrossDevice(path string) *Error {
	return newError(KindTrashCrossDevice, "'%s' is on another disk, so it can't go to the trash. Add \"permanently\" to delete it for good.", path)
}

func MissingDeleteTarget(got string) *Error {
	if got == "" {
		return newError(KindMissingDeleteTarget, "I need \"file\", \"folder\", \"files\", \"folders\" or \"all\" after \"delete\" — for example: delete file 'notes.txt'")
	}
	return newError(KindMissingDeleteTarget, "I can't delete \"%s\". Try \"delete file\", \"delete folder\", or \"delete files from\".", got)
}

func MissingDeletePath(kind string) *Error {
	return newError(KindMissingDeletePath, "I need a path after \"delete %s\" — for example: delete %s '%s'", kind, kind, exampleNameFor(kind))
}

func CannotRead(path string) *Error {
	return newError(KindCannotRead, "I couldn't read '%s'. The file may have changed while I was reading it.", path)
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
		return newError(KindUnknownTarget, "I can list \"files\", \"folders\", \"all\", or \"apps\" — not \"%s\". Did you mean \"%s\"?", got, suggestion)
	}
	return newError(KindUnknownTarget, "I can list \"files\", \"folders\", \"all\", or \"apps\" — not \"%s\".", got)
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

func FieldNotForTarget(field, target string, usable []string) *Error {
	if len(usable) == 0 {
		return newError(KindFieldNotForTarget, "\"%s\" doesn't work with \"%s\".", field, target)
	}
	return newError(KindFieldNotForTarget, "\"%s\" doesn't work with \"%s\". There you can filter on %s.", field, target, joinWithAnd(usable))
}

func WithNeedsSize(got string) *Error {
	if got == "" {
		return newError(KindWithNeedsSize, "\"with\" needs \"size\" — for example: apps with size")
	}
	return newError(KindWithNeedsSize, "After \"apps with\" I only know \"size\", not \"%s\".", got)
}

func CountHasNoSize() *Error {
	return newError(KindCountHasNoSize, "A count has no size column, and measuring every app would take a while for a number you didn't ask for. Try: apps with size")
}

func WithSizeComesFirst() *Error {
	return newError(KindWithSizeComesFirst, "\"with size\" goes before \"where\" — for example: apps with size where source = 'homebrew'")
}

func RefuseDeleteHere(path string) *Error {
	return newError(KindRefuseDeleteHere, "'%s' is the folder you are in, so I won't delete it. Move somewhere else first with \"cd ..\".", path)
}

func PwdTakesNoPath(got string) *Error {
	return newError(KindPwdTakesNoPath, "\"pwd\" just shows where you are, so it takes no folder. To move, use: cd %s", got)
}

func CannotChangeDir(path, reason string) *Error {
	return newError(KindCannotChangeDir, "I couldn't move to '%s': %s", path, reason)
}

func SummaryTakesNoWhere() *Error {
	return newError(KindSummaryTakesNoWhere, "A summary covers everything, so it takes no \"where\". Use \"apps where …\" to filter a list instead.")
}

func AppsNeedNoPath() *Error {
	return newError(KindAppsNeedNoPath, "\"apps\" already looks everywhere your system installs apps, so it needs no path. Try: apps")
}

func AppsNotRecursive() *Error {
	return newError(KindAppsNotRecursive, "\"apps\" is never recursive — looking inside an app would list the helpers it ships with as if they were apps. Try: apps")
}

func AppsUnavailable(reason string) *Error {
	return newError(KindAppsUnavailable, "I couldn't read your installed apps: %s", reason)
}

func CannotDeleteApps() *Error {
	return newError(KindCannotDeleteApps, "I won't uninstall apps — removing one properly also means its settings and background helpers, which I can't do safely. Use your system's own uninstaller.")
}

func BinaryNotFound() *Error {
	return newError(KindBinaryNotFound, "I couldn't work out where the osql binary lives, so I won't guess at what to delete. Find it with \"which osql\" and remove that file yourself.")
}

func InstalledByPackageManager(manager, command string) *Error {
	return newError(KindInstalledByPackageManager, "osql was installed by %s, so deleting the file would leave %s's records out of step. Run: %s", manager, manager, command)
}

func CannotRemoveBinary(path string) *Error {
	return newError(KindCannotRemoveBinary, "I don't have permission to remove '%s', and I won't ask for it. Run this yourself:\n\n  sudo rm '%s'", path, path)
}

func CannotRemoveData(path, reason string) *Error {
	return newError(KindCannotRemoveData, "I couldn't remove '%s': %s. Nothing was removed, so osql still works.", path, reason)
}

func Reason(err error) string {
	switch {
	case errors.Is(err, fs.ErrPermission):
		return "permission denied"
	case errors.Is(err, fs.ErrExist):
		return "something is already there"
	case errors.Is(err, fs.ErrNotExist):
		return "it is no longer there"
	default:
		return err.Error()
	}
}

func QueryStopped(found int) *Error {
	if found <= 0 {
		return newError(KindQueryStopped, "Stopped. Nothing had matched yet.")
	}
	return newError(KindQueryStopped, "Stopped after %d matches. Narrow the folder or add a where clause to make it quicker.", found)
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
