package main

import (
	"fmt"
	"os"

	"github.com/farhapartex/osql/internal/reader"
	"github.com/farhapartex/osql/internal/shell"
	"github.com/farhapartex/osql/internal/state"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "osql:", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := state.DefaultRoot()
	if err != nil {
		return err
	}

	store := state.New(state.Options{
		Root:    root,
		Version: version,
		Commit:  commit,
	})
	if err := store.Ensure(); err != nil {
		return err
	}
	defer store.Close()

	history, err := store.History()
	if err != nil {
		return err
	}

	app := shell.New(shell.Config{
		Reader:  reader.NewBasic(os.Stdin, os.Stdout, history),
		Store:   store,
		Out:     os.Stdout,
		Err:     os.Stderr,
		Version: version,
		Commit:  commit,
	})

	return app.Run()
}
