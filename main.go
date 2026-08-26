package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/farhapartex/osql/internal/apps"
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

func restoreOnSignal(input reader.LineReader) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		<-signals
		input.Close()
		os.Exit(130)
	}()
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

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	startDir := opts.Dir
	if startDir == "" {
		startDir, err = os.Getwd()
		if err != nil {
			startDir = home
		}
	}
	if abs, aerr := filepath.Abs(startDir); aerr == nil {
		startDir = abs
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

	fsys := vfs.OS()
	compiler := engine.NewCompiler(engine.DefaultFields(fsys), engine.DefaultOperators())
	resolver := engine.NewPathResolverAt(fsys, startDir, home)
	skip := engine.DefaultSkipList()
	selector := engine.NewSelectExecutor(fsys, resolver, compiler, skip)
	counter := engine.NewCountExecutor(fsys, resolver, compiler, skip)
	opener := engine.NewOpenExecutor(fsys, resolver)
	maker := engine.NewNewExecutor(fsys, resolver)
	summarizer := engine.NewSummaryExecutor(fsys, resolver, skip)
	remover := engine.NewDeleteExecutor(fsys, resolver, compiler, vfs.NewTrash(home, nil))
	lister := engine.NewAppsExecutor(apps.NewCatalog(apps.DefaultSources(home)...), compiler, apps.NewSizer())

	input, interactive := reader.New(os.Stdin, os.Stdout, history)
	defer input.Close()
	restoreOnSignal(input)

	app := shell.New(shell.Config{
		Reader:        input,
		Editing:       interactive,
		Lexer:         query.NewLexer(),
		Parser:        query.NewParser(compiler),
		Engine:        engine.NewRegistry(selector, counter, opener, maker, summarizer, remover, lister),
		Renderer:      output.NewTable(),
		CountRenderer: output.NewCount(),
		Apps:          output.NewApps(),
		AppSummary:    output.NewAppSummary(),
		Summary:       output.NewSummary(),
		Delete:        output.NewDelete(),
		Resolver:      resolver,
		Store:         store,
		Out:           os.Stdout,
		Err:           os.Stderr,
		Version:       version,
		Commit:        commit,
	})

	return app.Run()
}
