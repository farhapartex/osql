package shell

import (
	"io"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/output"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/reader"
	"github.com/farhapartex/osql/internal/state"
)

type Config struct {
	Reader        reader.LineReader
	Lexer         query.Lexer
	Parser        query.Parser
	Engine        *engine.Registry
	Renderer      output.Renderer
	CountRenderer output.Renderer
	Summary       output.SummaryRenderer
	Delete        output.DeleteRenderer
	Store         state.Store
	Out           io.Writer
	Err           io.Writer
	Editing       bool
	Version       string
	Commit        string
}
