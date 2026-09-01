package engine

import (
	"strconv"
	"strings"
	"time"

	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
)

const FieldModified = "modified"

const (
	dayLayout    = "2006-01-02"
	minuteLayout = "2006-01-02 15:04"
)

type Moment struct {
	Start    time.Time
	End      time.Time
	WholeDay bool
}

func wholeDayOf(t time.Time) Moment {
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return Moment{Start: start, End: start.AddDate(0, 0, 1), WholeDay: true}
}

func instantAt(t time.Time) Moment {
	return Moment{Start: t, End: t}
}

func ParseWhen(text string, now time.Time) (Moment, error) {
	trimmed := strings.ToLower(strings.Join(strings.Fields(text), " "))
	if trimmed == "" {
		return Moment{}, oerr.BadTimeValue(text)
	}

	switch trimmed {
	case "today":
		return wholeDayOf(now), nil
	case "yesterday":
		return wholeDayOf(now.AddDate(0, 0, -1)), nil
	}

	if moment, ok := parseAgo(trimmed, now); ok {
		return moment, nil
	}

	if at, err := time.ParseInLocation(minuteLayout, trimmed, now.Location()); err == nil {
		return instantAt(at), nil
	}
	if day, err := time.ParseInLocation(dayLayout, trimmed, now.Location()); err == nil {
		return wholeDayOf(day), nil
	}
	return Moment{}, oerr.BadTimeValue(text)
}

func parseAgo(text string, now time.Time) (Moment, bool) {
	parts := strings.Fields(text)
	if len(parts) != 3 || parts[2] != "ago" {
		return Moment{}, false
	}

	count, err := strconv.Atoi(parts[0])
	if err != nil || count < 0 {
		return Moment{}, false
	}

	switch strings.TrimSuffix(parts[1], "s") {
	case "day":
		return wholeDayOf(now.AddDate(0, 0, -count)), true
	case "week":
		return wholeDayOf(now.AddDate(0, 0, -7*count)), true
	case "month":
		return wholeDayOf(now.AddDate(0, -count, 0)), true
	case "year":
		return wholeDayOf(now.AddDate(-count, 0, 0)), true
	}
	return Moment{}, false
}

type ModifiedField struct {
	now func() time.Time
}

func NewModifiedField(now func() time.Time) ModifiedField {
	if now == nil {
		now = time.Now
	}
	return ModifiedField{now: now}
}

func (ModifiedField) Field() string              { return FieldModified }
func (ModifiedField) Cost() int                  { return CostStat }
func (ModifiedField) AllowedOperators() []string { return orderingOperators }

func (ModifiedField) AppliesTo(t query.Target) bool { return t != query.TargetApps }

func (f ModifiedField) NormalizeValue(v string) (Value, error) {
	moment, err := ParseWhen(v, f.now())
	if err != nil {
		return Value{}, err
	}
	if !moment.WholeDay {
		return Value{Number: moment.Start.Unix(), IsNum: true}, nil
	}
	return Value{
		Number: moment.Start.Unix(),
		Upper:  moment.End.Unix(),
		IsNum:  true,
		IsSpan: true,
	}, nil
}

func (ModifiedField) Extract(e Entry) (Value, error) {
	info, err := e.DirEntry.Info()
	if err != nil {
		return Value{}, err
	}
	return Value{Number: info.ModTime().Unix(), IsNum: true}, nil
}
