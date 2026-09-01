package engine

import (
	"math"
	"strconv"
	"strings"

	"github.com/farhapartex/osql/internal/oerr"
	"github.com/farhapartex/osql/internal/query"
)

const FieldSize = "size"

const sizeStep = 1024

var sizeUnits = []struct {
	suffix string
	scale  float64
}{
	{"tb", sizeStep * sizeStep * sizeStep * sizeStep},
	{"gb", sizeStep * sizeStep * sizeStep},
	{"mb", sizeStep * sizeStep},
	{"kb", sizeStep},
	{"b", 1},
}

func ParseSize(text string) (int64, error) {
	trimmed := strings.ToLower(strings.TrimSpace(text))
	if trimmed == "" {
		return 0, oerr.BadSizeValue(text)
	}

	digits := trimmed
	scale := float64(1)
	for _, unit := range sizeUnits {
		if rest, found := strings.CutSuffix(trimmed, unit.suffix); found {
			digits = strings.TrimSpace(rest)
			scale = unit.scale
			break
		}
	}

	if digits == "" {
		return 0, oerr.BadSizeValue(text)
	}

	amount, err := strconv.ParseFloat(digits, 64)
	if err != nil || math.IsNaN(amount) || math.IsInf(amount, 0) || amount < 0 {
		return 0, oerr.BadSizeValue(text)
	}

	bytes := amount * scale
	if bytes > math.MaxInt64 {
		return 0, oerr.SizeTooLarge(text)
	}
	return int64(bytes), nil
}

type SizeField struct{}

func (SizeField) Field() string              { return FieldSize }
func (SizeField) Cost() int                  { return CostStat }
func (SizeField) AllowedOperators() []string { return orderingOperators }

func (SizeField) AppliesTo(t query.Target) bool {
	return t == query.TargetFiles || t == query.TargetAll
}

func (SizeField) NormalizeValue(v string) (Value, error) {
	bytes, err := ParseSize(v)
	if err != nil {
		return Value{}, err
	}
	return Value{Number: bytes, IsNum: true}, nil
}

func (SizeField) Extract(e Entry) (Value, error) {
	if e.DirEntry.IsDir() {
		return Value{Number: 0, IsNum: true}, nil
	}

	info, err := e.DirEntry.Info()
	if err != nil {
		return Value{}, err
	}
	return Value{Number: info.Size(), IsNum: true}, nil
}
