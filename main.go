package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	"github.com/farhapartex/osql/internal/uninstall"
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

type interruptible interface {
	Interrupt() bool
}

func leaveCleanly(input reader.LineReader, store io.Closer) {
	input.Close()
	if store != nil {
		store.Close()
	}
	os.Exit(130)
}

func restoreOnSignal(input reader.LineReader, store io.Closer) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		<-signals
		leaveCleanly(input, store)
	}()
}

func stopQueryOnInterrupt(app interruptible, input reader.LineReader, store io.Closer) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)

	go func() {
		for range signals {
			if app.Interrupt() {
				continue
			}
			leaveCleanly(input, store)
		}
	}()
}

func removeOsql(opts cli.Options) error {
	stateRoot, err := state.DefaultRoot()
	if err != nil {
		return err
	}

	remover := uninstall.New(uninstall.Options{
		Files:        uninstall.SystemFiles(),
		LocateBinary: os.Executable,
		StateRoot:    stateRoot,
	})

	plan, err := remover.Plan(opts.KeepData)
	if err != nil {
		return err
	}

	renderer := output.NewUninstall()
	if !opts.Confirmed {
		if err := renderer.Preview(os.Stdout, plan); err != nil {
			return err
		}

		answer, err := reader.NewBasic(os.Stdin, os.Stdout, nil).ReadLine(output.UninstallPrompt)
		if err != nil || strings.TrimSpace(answer) != output.ConfirmWord {
			return renderer.Cancelled(os.Stdout)
		}
	}

	if err := remover.Commit(plan); err != nil {
		return err
	}
	return renderer.Result(os.Stdout, plan)
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
	case cli.CommandUninstall:
		return removeOsql(opts)
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
	restoreOnSignal(input, store)

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

	stopQueryOnInterrupt(app, input, store)

	return app.Run()
}
