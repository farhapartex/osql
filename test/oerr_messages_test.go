package test

import (
	"testing"

	"github.com/farhapartex/osql/internal/oerr"
)

func TestErrorMessagesMatchSpec(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			"folder missing",
			oerr.FolderMissing("Documnets"),
			"I couldn't find a folder at 'Documnets'. Check the path and try again.",
		},
		{
			"path is a file",
			oerr.PathIsFile("notes.txt"),
			"'notes.txt' is a file, not a folder. Try: select files from 'Documents'",
		},
		{
			"no permission",
			oerr.NoPermission("Library"),
			"I don't have permission to read 'Library'.",
		},
		{
			"unknown verb with suggestion",
			oerr.UnknownVerb("slect", []string{"select"}),
			`I don't know how to "slect". Did you mean "select"?`,
		},
		{
			"singular target",
			oerr.SingularTarget("file"),
			`Use "files", not "file" — for example: select files from 'Documents'`,
		},
		{
			"unknown target",
			oerr.UnknownTarget("documents"),
			`I can select "files", "folders", or "all" — not "documents".`,
		},
		{
			"missing from",
			oerr.MissingFrom(),
			`I need "from" before the folder — for example: select files from 'Documents'`,
		},
		{
			"missing path",
			oerr.MissingPath(),
			`I need a folder after "from" — for example: select files from 'Documents'`,
		},
		{
			"unknown field",
			oerr.UnknownField("extension", []string{"name", "name_like", "type", "count(child)"}),
			`I don't know the field "extension". I understand: name, name_like, type, count(child)`,
		},
		{
			"wrong operator for field",
			oerr.WrongOperator("name", []string{"=", "!="}),
			`"name" only works with = and !=. For patterns use name_like: select files from 'Documents' where name_like = '%report%'`,
		},
		{
			"count(child) on files",
			oerr.CountChildOnFiles(),
			"count(child) describes folders, not files. Try: select folders from 'Documents' where count(child) > 10",
		},
		{
			"count(child) non-numeric",
			oerr.CountChildNonNumeric(),
			"count(child) needs a number — for example: count(child) > 10",
		},
		{
			"unclosed quote",
			oerr.UnclosedQuote("Documents"),
			"This quote is never closed: 'Documents — add a closing '",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("message mismatch\n got: %s\nwant: %s", got, tt.want)
			}
		})
	}
}

func TestOutcomeMessagesMatchSpec(t *testing.T) {
	if got, want := oerr.NoMatches(), "No files matched."; got != want {
		t.Errorf("NoMatches() = %q, want %q", got, want)
	}
	if got, want := oerr.EmptyFolder("Documents"), "'Documents' is empty."; got != want {
		t.Errorf("EmptyFolder() = %q, want %q", got, want)
	}
}

func TestUnknownVerbWithoutSuggestion(t *testing.T) {
	err := oerr.UnknownVerb("xyzzy", []string{"select"})

	want := `I don't know how to "xyzzy".`
	if got := err.Error(); got != want {
		t.Errorf("far-off input must not guess\n got: %s\nwant: %s", got, want)
	}
}

func TestUnknownVerbWithNoCandidates(t *testing.T) {
	err := oerr.UnknownVerb("select", nil)

	want := `I don't know how to "select".`
	if got := err.Error(); got != want {
		t.Errorf("empty candidate list must not panic or suggest\n got: %s\nwant: %s", got, want)
	}
}

func TestSingularTargetPluralises(t *testing.T) {
	tests := []struct {
		got  string
		want string
	}{
		{"file", `Use "files", not "file" — for example: select files from 'Documents'`},
		{"folder", `Use "folders", not "folder" — for example: select folders from 'Documents'`},
	}

	for _, tt := range tests {
		if got := oerr.SingularTarget(tt.got).Error(); got != tt.want {
			t.Errorf("SingularTarget(%q)\n got: %s\nwant: %s", tt.got, got, tt.want)
		}
	}
}

func TestWrongOperatorOmitsPatternHintForNameLike(t *testing.T) {
	err := oerr.WrongOperator("name_like", []string{"=", "!="})

	want := `"name_like" only works with = and !=.`
	if got := err.Error(); got != want {
		t.Errorf("advising name_like when the field already is name_like is nonsense\n got: %s\nwant: %s", got, want)
	}
}

func TestWrongOperatorJoinsOperatorLists(t *testing.T) {
	tests := []struct {
		name    string
		allowed []string
		want    string
	}{
		{"single", []string{"="}, `"count(child)" only works with =.`},
		{"pair", []string{"=", "!="}, `"count(child)" only works with = and !=.`},
		{"many", []string{"=", "!=", "<", ">"}, `"count(child)" only works with =, !=, <, and >.`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := oerr.WrongOperator("count(child)", tt.allowed).Error()
			want := tt.want + " For patterns use name_like: select files from 'Documents' where name_like = '%report%'"
			if got != want {
				t.Errorf("\n got: %s\nwant: %s", got, want)
			}
		})
	}
}

func TestUnknownFieldWithSingleField(t *testing.T) {
	err := oerr.UnknownField("size", []string{"name"})

	want := `I don't know the field "size". I understand: name`
	if got := err.Error(); got != want {
		t.Errorf("\n got: %s\nwant: %s", got, want)
	}
}

func TestErrorsCarryTheirKind(t *testing.T) {
	tests := []struct {
		err  error
		kind oerr.Kind
	}{
		{oerr.FolderMissing("x"), oerr.KindFolderMissing},
		{oerr.PathIsFile("x"), oerr.KindPathIsFile},
		{oerr.NoPermission("x"), oerr.KindNoPermission},
		{oerr.UnknownVerb("x", nil), oerr.KindUnknownVerb},
		{oerr.SingularTarget("file"), oerr.KindSingularTarget},
		{oerr.UnknownTarget("x"), oerr.KindUnknownTarget},
		{oerr.MissingFrom(), oerr.KindMissingFrom},
		{oerr.MissingPath(), oerr.KindMissingPath},
		{oerr.UnknownField("x", nil), oerr.KindUnknownField},
		{oerr.WrongOperator("name", nil), oerr.KindWrongOperator},
		{oerr.CountChildOnFiles(), oerr.KindCountChildOnFiles},
		{oerr.CountChildNonNumeric(), oerr.KindCountChildNonNumeric},
		{oerr.UnclosedQuote("x"), oerr.KindUnclosedQuote},
	}

	for _, tt := range tests {
		t.Run(tt.kind.String(), func(t *testing.T) {
			if !oerr.Is(tt.err, tt.kind) {
				t.Errorf("Is(%v, %v) = false, want true", tt.err, tt.kind)
			}
			if oerr.Is(tt.err, oerr.Kind(999)) {
				t.Error("matched a kind it does not carry")
			}
		})
	}
}

func TestIsOnForeignErrors(t *testing.T) {
	if oerr.Is(nil, oerr.KindFolderMissing) {
		t.Error("Is(nil, ...) = true, want false")
	}
	if oerr.Is(errPlain, oerr.KindFolderMissing) {
		t.Error("Is matched a non-oerr error")
	}
}

func TestKindString(t *testing.T) {
	tests := []struct {
		kind oerr.Kind
		want string
	}{
		{oerr.KindFolderMissing, "folder_missing"},
		{oerr.KindUnclosedQuote, "unclosed_quote"},
		{oerr.Kind(999), "unknown"},
		{oerr.Kind(-1), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("Kind(%d).String() = %q, want %q", int(tt.kind), got, tt.want)
		}
	}
}

func TestEveryKindHasAName(t *testing.T) {
	for k := oerr.KindFolderMissing; k <= oerr.KindUnclosedQuote; k++ {
		if k.String() == "unknown" {
			t.Errorf("Kind(%d) has no name; every declared kind needs one", int(k))
		}
	}
}
