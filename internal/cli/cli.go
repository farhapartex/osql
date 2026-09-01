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
	CommandUninstall
)

var commandNames = map[Command]string{
	CommandShell:     "shell",
	CommandVersion:   "version",
	CommandInit:      "init",
	CommandHelp:      "help",
	CommandUninstall: "uninstall",
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
	KeepData  bool
	Confirmed bool
	Dir       string
}

const Usage = `osql — query your filesystem in SQL-like statements

usage:
  osql                  start the interactive shell
  osql init             create ~/.osql and write system.txt
  osql init --reinit    rewrite system.txt even if it exists
  osql uninstall        remove osql and ~/.osql from this machine

flags:
  --dir <path>          start in <path> instead of the folder you ran osql from
  --no-history          do not record this session's commands
  --keep-data           uninstall, but leave ~/.osql where it is
  --yes                 uninstall without asking first
  --version             print the version and exit
  --help                print this message

queries:
  files from '<path>'                       one level, files only
  folders from '<path>'                     one level, folders only
  all from '<path>'                         one level, everything
  files from '<path>' recursive             the whole subtree
  files from '<path>' where type = 'txt'    filtered
  files from '<path>' where size > 10mb     filtered by size
  count(files) from '<path>'                how many, instead of which
  open '<path>'                             print a text file
  new file '<path>' data='hello'            make a file, optionally with content
  new folder '<path>'                       make a folder
  summary from '<path>' [recursive]         counts, sizes and types at a glance
  apps [with size] [where ...]              the apps installed on this machine
  summary apps                              your installed apps, at a glance
  delete file '<path>'                      move one file to the trash
  delete files from '<path>' where ...      move matching files to the trash

in the shell:
  cd '<path>'                               move to another folder
  pwd                                       show which folder you are in

Paths mean what they mean in a terminal: 'Documents' is inside the folder you
are in, '/etc' is absolute, and '~' is your home folder.`

func Parse(args []string) (Options, error) {
	opts := Options{Command: CommandShell}

	if len(args) > 0 {
		switch args[0] {
		case "init":
			opts.Command = CommandInit
			args = args[1:]
		case "uninstall":
			opts.Command = CommandUninstall
			args = args[1:]
		}
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if value, ok := strings.CutPrefix(arg, "--dir="); ok {
			if opts.Command != CommandShell {
				return Options{}, fmt.Errorf("--dir only applies to the interactive shell")
			}
			if value == "" {
				return Options{}, fmt.Errorf("--dir needs a path, for example: osql --dir /tmp")
			}
			opts.Dir = value
			continue
		}

		switch arg {
		case "--dir", "--root":
			if opts.Command != CommandShell {
				return Options{}, fmt.Errorf("--dir only applies to the interactive shell")
			}
			if i+1 >= len(args) {
				return Options{}, fmt.Errorf("--dir needs a path, for example: osql --dir /tmp")
			}
			i++
			opts.Dir = args[i]
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
		case "--keep-data":
			if opts.Command != CommandUninstall {
				return Options{}, fmt.Errorf("--keep-data only applies to \"osql uninstall\"")
			}
			opts.KeepData = true
		case "--yes", "-y":
			if opts.Command != CommandUninstall {
				return Options{}, fmt.Errorf("--yes only applies to \"osql uninstall\"")
			}
			opts.Confirmed = true
		default:
			return Options{}, fmt.Errorf("I don't recognise %q. Run \"osql --help\" to see the options", arg)
		}
	}

	return opts, nil
}
