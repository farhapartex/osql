package output

import (
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"github.com/farhapartex/osql/internal/engine"
)

const (
	HeaderWhat  = "WHAT"
	HeaderCount = "COUNT"
)

type CountRenderer struct{}

func NewCount() CountRenderer {
	return CountRenderer{}
}

func (CountRenderer) Render(w io.Writer, rows []engine.Row) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	if _, err := fmt.Fprintf(tw, "%s\t%s\n", HeaderWhat, HeaderCount); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(tw, "%s\t%s\n", row.Name, strconv.FormatInt(row.Count, 10)); err != nil {
			return err
		}
	}

	return tw.Flush()
}
