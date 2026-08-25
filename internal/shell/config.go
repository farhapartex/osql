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
	Store         state.Store
	Out           io.Writer
	Err           io.Writer
	Version       string
	Commit        string
}
