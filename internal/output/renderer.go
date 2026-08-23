package output

import (
	"io"

	"github.com/farhapartex/osql/internal/engine"
)

type Renderer interface {
	Render(w io.Writer, rows []engine.Row) error
}
