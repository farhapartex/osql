package main

import (
	"fmt"
	"os"

	"github.com/farhapartex/osql/internal/shell"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	app := shell.New(shell.Config{
		Out:     os.Stdout,
		Err:     os.Stderr,
		Version: version,
		Commit:  commit,
	})

	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "osql:", err)
		os.Exit(1)
	}
}
