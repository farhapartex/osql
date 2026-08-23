package cli

import "fmt"

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
}

const Usage = `osql — query your filesystem in SQL-like statements

usage:
  osql                  start the interactive shell
  osql init             create ~/.osql and write system.txt
  osql init --reinit    rewrite system.txt even if it exists

flags:
  --no-history          do not record this session's commands
  --version             print the version and exit
  --help                print this message`

func Parse(args []string) (Options, error) {
	opts := Options{Command: CommandShell}

	if len(args) > 0 && args[0] == "init" {
		opts.Command = CommandInit
		args = args[1:]
	}

	for _, arg := range args {
		switch arg {
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
