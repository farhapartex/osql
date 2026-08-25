package output

import (
	"math"
	"strconv"
	"time"

	"github.com/farhapartex/osql/internal/engine"
)

const (
	Absent         = "—"
	TypeFolder     = "folder"
	ModifiedLayout = "2006-01-02 15:04"

	sizeStep = 1024
)

var sizeUnits = []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}

func FormatSize(bytes int64) string {
	if bytes < 0 {
		bytes = 0
	}
	if bytes < sizeStep {
		return strconv.FormatInt(bytes, 10) + " " + sizeUnits[0]
	}

	value := float64(bytes)
	unit := 0
	for value >= sizeStep && unit < len(sizeUnits)-1 {
		value /= sizeStep
		unit++
	}
	if math.Round(value*10)/10 >= sizeStep && unit < len(sizeUnits)-1 {
		value /= sizeStep
		unit++
	}

	return strconv.FormatFloat(value, 'f', 1, 64) + " " + sizeUnits[unit]
}

func FormatType(row engine.Row) string {
	if row.IsDir {
		return TypeFolder
	}
	if row.Ext == "" {
		return Absent
	}
	return row.Ext
}

func FormatRowSize(row engine.Row) string {
	if row.IsDir {
		return Absent
	}
	return FormatSize(row.Size)
}

func FormatModified(t time.Time) string {
	if t.IsZero() {
		return Absent
	}
	return t.Format(ModifiedLayout)
}

func FormatCount(n int) string {
	return strconv.Itoa(n) + " files"
}
