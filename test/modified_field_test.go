package test

import (
	"context"
	"slices"
	"testing"
	"testing/fstest"
	"time"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
)

func fixedNow() time.Time {
	return time.Date(2026, 3, 15, 13, 45, 0, 0, time.UTC)
}

func day(year int, month time.Month, d int) time.Time {
	return time.Date(year, month, d, 0, 0, 0, 0, time.UTC)
}

func TestParseWhenUnderstandsEveryForm(t *testing.T) {
	tests := []struct {
		in        string
		wantStart time.Time
		wantEnd   time.Time
		wholeDay  bool
	}{
		{"today", day(2026, 3, 15), day(2026, 3, 16), true},
		{"TODAY", day(2026, 3, 15), day(2026, 3, 16), true},
		{"yesterday", day(2026, 3, 14), day(2026, 3, 15), true},
		{"1 day ago", day(2026, 3, 14), day(2026, 3, 15), true},
		{"7 days ago", day(2026, 3, 8), day(2026, 3, 9), true},
		{"0 days ago", day(2026, 3, 15), day(2026, 3, 16), true},
		{"1 week ago", day(2026, 3, 8), day(2026, 3, 9), true},
		{"2 weeks ago", day(2026, 3, 1), day(2026, 3, 2), true},
		{"1 month ago", day(2026, 2, 15), day(2026, 2, 16), true},
		{"3 months ago", day(2025, 12, 15), day(2025, 12, 16), true},
		{"1 year ago", day(2025, 3, 15), day(2025, 3, 16), true},
		{"2026-01-31", day(2026, 1, 31), day(2026, 2, 1), true},
		{"  7   days   ago  ", day(2026, 3, 8), day(2026, 3, 9), true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := engine.ParseWhen(tt.in, fixedNow())
			if err != nil {
				t.Fatalf("ParseWhen(%q) = %v", tt.in, err)
			}
			if !got.Start.Equal(tt.wantStart) {
				t.Errorf("start = %v, want %v", got.Start, tt.wantStart)
			}
			if !got.End.Equal(tt.wantEnd) {
				t.Errorf("end = %v, want %v", got.End, tt.wantEnd)
			}
			if got.WholeDay != tt.wholeDay {
				t.Errorf("WholeDay = %v, want %v", got.WholeDay, tt.wholeDay)
			}
		})
	}
}

func TestParseWhenReadsAClockTime(t *testing.T) {
	got, err := engine.ParseWhen("2026-01-31 14:30", fixedNow())
	if err != nil {
		t.Fatalf("ParseWhen() = %v", err)
	}

	want := time.Date(2026, 1, 31, 14, 30, 0, 0, time.UTC)
	if !got.Start.Equal(want) {
		t.Errorf("start = %v, want %v", got.Start, want)
	}
	if got.WholeDay {
		t.Error("a value with a clock time is a moment, not a whole day")
	}
	if !got.End.Equal(got.Start) {
		t.Errorf("a moment should have no span, got %v to %v", got.Start, got.End)
	}
}

func TestParseWhenRejectsNonsense(t *testing.T) {
	for _, in := range []string{
		"", "   ", "yesteryear", "7 days", "days ago", "ago",
		"2026-13-01", "2026-02-30", "31-01-2026", "tomorrow",
		"-1 days ago", "many days ago", "1 fortnight ago", "1 day from now",
		"2026-01-31 25:00",
	} {
		t.Run(in, func(t *testing.T) {
			if _, err := engine.ParseWhen(in, fixedNow()); !oerr.Is(err, oerr.KindBadTimeValue) {
				t.Errorf("ParseWhen(%q) = %v, want bad_time_value", in, err)
			}
		})
	}
}

func TestModifiedFieldShape(t *testing.T) {
	field := engine.NewModifiedField(fixedNow)

	if field.Field() != "modified" {
		t.Errorf("Field() = %q", field.Field())
	}
	if field.Cost() != engine.CostStat {
		t.Errorf("Cost() = %d; modified needs a stat", field.Cost())
	}
	for _, op := range []string{"=", "!=", "<", ">", "<=", ">="} {
		if !slices.Contains(field.AllowedOperators(), op) {
			t.Errorf("modified should accept %q", op)
		}
	}
	for _, target := range []query.Target{query.TargetFiles, query.TargetFolders, query.TargetAll} {
		if !field.AppliesTo(target) {
			t.Errorf("modified should apply to %v", target)
		}
	}
	if field.AppliesTo(query.TargetApps) {
		t.Error("modified should not apply to apps")
	}
}

func TestADayComparesAsAWholeDay(t *testing.T) {
	field := engine.NewModifiedField(fixedNow)

	want, err := field.NormalizeValue("2026-03-10")
	if err != nil {
		t.Fatalf("NormalizeValue() = %v", err)
	}
	if !want.IsSpan {
		t.Fatal("a bare date must cover the whole day")
	}

	moment := func(hour int) engine.Value {
		at := time.Date(2026, 3, 10, hour, 0, 0, 0, time.UTC)
		return engine.Value{Number: at.Unix(), IsNum: true}
	}
	dayBefore := engine.Value{Number: day(2026, 3, 9).Unix(), IsNum: true}
	dayAfter := engine.Value{Number: day(2026, 3, 11).Unix(), IsNum: true}

	tests := []struct {
		name string
		op   engine.Comparator
		got  engine.Value
		want bool
	}{
		{"equal matches the start of the day", engine.EqualOp{}, moment(0), true},
		{"equal matches the end of the day", engine.EqualOp{}, moment(23), true},
		{"equal rejects the day before", engine.EqualOp{}, dayBefore, false},
		{"equal rejects the day after", engine.EqualOp{}, dayAfter, false},
		{"not equal is the opposite", engine.NotEqualOp{}, moment(12), false},
		{"not equal accepts another day", engine.NotEqualOp{}, dayAfter, true},
		{"greater skips the whole day", engine.GreaterOp{}, moment(23), false},
		{"greater takes the next day", engine.GreaterOp{}, dayAfter, true},
		{"less rejects the day itself", engine.LessOp{}, moment(0), false},
		{"less takes an earlier day", engine.LessOp{}, dayBefore, true},
		{"at least takes the day itself", engine.GreaterEqualOp{}, moment(0), true},
		{"at least rejects earlier", engine.GreaterEqualOp{}, dayBefore, false},
		{"at most takes the day itself", engine.LessEqualOp{}, moment(23), true},
		{"at most rejects the next day", engine.LessEqualOp{}, dayAfter, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.Compare(tt.got, want); got != tt.want {
				t.Errorf("%s %s day = %v, want %v", tt.name, tt.op.Op(), got, tt.want)
			}
		})
	}
}

func TestAClockTimeComparesAsAMoment(t *testing.T) {
	field := engine.NewModifiedField(fixedNow)

	want, err := field.NormalizeValue("2026-03-10 12:00")
	if err != nil {
		t.Fatalf("NormalizeValue() = %v", err)
	}
	if want.IsSpan {
		t.Fatal("a value with a clock time must not cover a whole day")
	}

	before := engine.Value{Number: time.Date(2026, 3, 10, 11, 59, 0, 0, time.UTC).Unix(), IsNum: true}
	after := engine.Value{Number: time.Date(2026, 3, 10, 12, 1, 0, 0, time.UTC).Unix(), IsNum: true}

	if !(engine.LessOp{}).Compare(before, want) {
		t.Error("11:59 should be before 12:00")
	}
	if !(engine.GreaterOp{}).Compare(after, want) {
		t.Error("12:01 should be after 12:00")
	}
}

func localAt(year int, month time.Month, d, hour, minute int) time.Time {
	return time.Date(year, month, d, hour, minute, 0, 0, time.Local)
}

func datedFS() fstest.MapFS {
	return fstest.MapFS{
		"work/old.txt":    {Data: []byte("a"), ModTime: localAt(2025, 6, 1, 9, 0)},
		"work/edge.txt":   {Data: []byte("b"), ModTime: localAt(2026, 3, 10, 23, 59)},
		"work/midday.txt": {Data: []byte("c"), ModTime: localAt(2026, 3, 10, 12, 0)},
		"work/next.txt":   {Data: []byte("d"), ModTime: localAt(2026, 3, 11, 0, 1)},
	}
}

func modifiedQueryNames(t *testing.T, q string) []string {
	t.Helper()

	fsys := &fakeFileSystem{fsys: datedFS()}
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())
	resolver := engine.NewPathResolver(fsys, "/")
	selector := engine.NewSelectExecutor(fsys, resolver, compiler, engine.EmptySkipList())

	stmt, err := query.NewParser(compiler).Parse(mustLex(t, q))
	if err != nil {
		t.Fatalf("Parse(%q) = %v", q, err)
	}

	sink := &engine.SliceSink{}
	if err := selector.Execute(context.Background(), stmt, sink); err != nil {
		t.Fatalf("Execute(%q) = %v", q, err)
	}

	names := make([]string, 0, len(sink.Rows))
	for _, row := range sink.Rows {
		names = append(names, row.Name)
	}
	return names
}

func TestModifiedFiltersByDate(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"a whole day", "files from 'work' where modified = '2026-03-10'", []string{"edge.txt", "midday.txt"}},
		{"after that day", "files from 'work' where modified > '2026-03-10'", []string{"next.txt"}},
		{"before that day", "files from 'work' where modified < '2026-03-10'", []string{"old.txt"}},
		{"that day or later", "files from 'work' where modified >= '2026-03-10'", []string{"edge.txt", "midday.txt", "next.txt"}},
		{"that day or earlier", "files from 'work' where modified <= '2026-03-10'", []string{"edge.txt", "midday.txt", "old.txt"}},
		{"any other day", "files from 'work' where modified != '2026-03-10'", []string{"next.txt", "old.txt"}},
		{"a clock time", "files from 'work' where modified > '2026-03-10 13:00'", []string{"edge.txt", "next.txt"}},
		{"nothing that old", "files from 'work' where modified < '2020-01-01'", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modifiedQueryNames(t, tt.query)
			if !slices.Equal(got, tt.want) {
				t.Errorf("%s\n got: %v\nwant: %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestModifiedCombinesWithCheaperFields(t *testing.T) {
	got := modifiedQueryNames(t, "files from 'work' where type = 'txt' and modified = '2026-03-10'")

	if !slices.Equal(got, []string{"edge.txt", "midday.txt"}) {
		t.Errorf("got %v", got)
	}
}

func TestModifiedRejectsABadDateAtParseTime(t *testing.T) {
	fsys := &fakeFileSystem{fsys: datedFS()}
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())

	_, err := query.NewParser(compiler).Parse(mustLex(t, "files from 'work' where modified = 'someday'"))
	if !oerr.Is(err, oerr.KindBadTimeValue) {
		t.Errorf("Parse() = %v, want bad_time_value", err)
	}
}
