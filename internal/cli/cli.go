package cli

import (
	"fmt"
	"strings"
)

type Command int

const (
	CommandShell Command = iota
	CommandVersion
	CommandInit
	CommandHelp
)

var commandNames = map[Command]string{
	CommandShell:   "shell",
	CommandVersion: "version",
	CommandInit:    "init",
	CommandHelp:    "help",
}

func (c Command) String() string {
	if name, ok := commandNames[c]; ok {
		return name
	}
	return "unknown"
}

type Options struct {
	Command   Command
	Reinit    bool
	NoHistory bool
	Root      string
}

const Usage = `osql — query your filesystem in SQL-like statements

usage:
  osql                  start the interactive shell
  osql init             create ~/.osql and write system.txt
  osql init --reinit    rewrite system.txt even if it exists

flags:
  --root <path>         anchor queries at <path> instead of your home directory
  --no-history          do not record this session's commands
  --version             print the version and exit
  --help                print this message

queries:
  files from '<path>'                       one level, files only
  folders from '<path>'                     one level, folders only
  all from '<path>'                         one level, everything
  files from '<path>' recursive             the whole subtree
  files from '<path>' where type = 'txt'    filtered
  count(files) from '<path>'                how many, instead of which

Every path is resolved inside the root, so 'Documents', '/Documents' and
'~/Documents' all mean the same folder.`

func Parse(args []string) (Options, error) {
	opts := Options{Command: CommandShell}

	if len(args) > 0 && args[0] == "init" {
		opts.Command = CommandInit
		args = args[1:]
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if value, ok := strings.CutPrefix(arg, "--root="); ok {
			if opts.Command != CommandShell {
				return Options{}, fmt.Errorf("--root only applies to the interactive shell")
			}
			if value == "" {
				return Options{}, fmt.Errorf("--root needs a path, for example: osql --root /")
			}
			opts.Root = value
			continue
		}

		switch arg {
		case "--root":
			if opts.Command != CommandShell {
				return Options{}, fmt.Errorf("--root only applies to the interactive shell")
			}
			if i+1 >= len(args) {
				return Options{}, fmt.Errorf("--root needs a path, for example: osql --root /")
			}
			i++
			opts.Root = args[i]
		case "--version", "-v":
			if opts.Command != CommandShell {
				return Options{}, fmt.Errorf("--version cannot be combined with %q", opts.Command)
			}
			opts.Command = CommandVersion
		case "--help", "-h":
			opts.Command = CommandHelp
		case "--reinit":
			if opts.Command != CommandInit {
				return Options{}, fmt.Errorf("--reinit only applies to \"osql init\"")
			}
			opts.Reinit = true
		case "--no-history":
			if opts.Command != CommandShell {
				return Options{}, fmt.Errorf("--no-history only applies to the interactive shell")
			}
			opts.NoHistory = true
		default:
			return Options{}, fmt.Errorf("I don't recognise %q. Run \"osql --help\" to see the options", arg)
		}
	}

	return opts, nil
}
