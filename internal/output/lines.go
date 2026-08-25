package output

import (
	"bufio"
	"io"

	"github.com/farhapartex/osql/internal/engine"
)

type LinesRenderer struct{}

func NewLines() LinesRenderer {
	return LinesRenderer{}
}

func (LinesRenderer) Render(w io.Writer, rows []engine.Row) error {
	buf := bufio.NewWriter(w)
	for _, row := range rows {
		if _, err := buf.WriteString(row.Name); err != nil {
			return err
		}
		if err := buf.WriteByte('\n'); err != nil {
			return err
		}
	}
	return buf.Flush()
}
