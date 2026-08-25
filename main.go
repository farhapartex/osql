package main

import (
	"fmt"
	"os"

	"github.com/farhapartex/osql/internal/buildinfo"
	"github.com/farhapartex/osql/internal/cli"
	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/output"
	"github.com/farhapartex/osql/internal/query"
	"github.com/farhapartex/osql/internal/reader"
	"github.com/farhapartex/osql/internal/shell"
	"github.com/farhapartex/osql/internal/state"
	"github.com/farhapartex/osql/internal/vfs"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "osql:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	opts, err := cli.Parse(args)
	if err != nil {
		return err
	}

	switch opts.Command {
	case cli.CommandVersion:
		fmt.Fprintln(os.Stdout, buildinfo.String(version, commit))
		return nil
	case cli.CommandHelp:
		fmt.Fprintln(os.Stdout, cli.Usage)
		return nil
	}

	queryRoot := opts.Root
	if queryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		queryRoot = home
	}

	root, err := state.DefaultRoot()
	if err != nil {
		return err
	}

	store := state.New(state.Options{
		Root:      root,
		Version:   version,
		Commit:    commit,
		NoHistory: opts.NoHistory,
	})
	if err := store.Ensure(); err != nil {
		return err
	}
	defer store.Close()

	if opts.Command == cli.CommandInit {
		if err := store.WriteSystemInfo(opts.Reinit); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "ready: %s\n", store.Root())
		return nil
	}

	history, err := store.History()
	if err != nil {
		return err
	}

	fsys := vfs.NewOS(queryRoot)
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())
	resolver := engine.NewPathResolver(fsys, fsys.Root())
	skip := engine.DefaultSkipList()
	selector := engine.NewSelectExecutor(fsys, resolver, compiler, skip)
	counter := engine.NewCountExecutor(fsys, resolver, compiler, skip)

	app := shell.New(shell.Config{
		Reader:        reader.NewBasic(os.Stdin, os.Stdout, history),
		Lexer:         query.NewLexer(),
		Parser:        query.NewParser(compiler),
		Engine:        engine.NewRegistry(selector, counter),
		Renderer:      output.NewTable(),
		CountRenderer: output.NewCount(),
		Store:         store,
		Out:           os.Stdout,
		Err:           os.Stderr,
		Version:       version,
		Commit:        commit,
	})

	return app.Run()
}
