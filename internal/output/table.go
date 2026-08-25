package output

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/farhapartex/osql/internal/engine"
)

const (
	HeaderName     = "NAME"
	HeaderType     = "TYPE"
	HeaderSize     = "SIZE"
	HeaderModified = "MODIFIED"
)

type TableRenderer struct{}

func NewTable() TableRenderer {
	return TableRenderer{}
}

func (TableRenderer) Render(w io.Writer, rows []engine.Row) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", HeaderName, HeaderType, HeaderSize, HeaderModified); err != nil {
		return err
	}

	for _, row := range rows {
		_, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			row.Name,
			FormatType(row),
			FormatRowSize(row),
			FormatModified(row.Modified),
		)
		if err != nil {
			return err
		}
	}

	if err := tw.Flush(); err != nil {
		return err
	}

	_, err := fmt.Fprintf(w, "\n%s\n", FormatCount(len(rows)))
	return err
}
