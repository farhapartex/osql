package shell

import (
	"errors"
	"fmt"
	"slices"
	"text/tabwriter"
)

var ErrExit = errors.New("exit")

const historyDisplayLimit = 50

type BuiltinFunc func(s *Shell, args []string) error

type Builtin struct {
	Name    string
	Summary string
	Run     BuiltinFunc
}

type BuiltinRegistry struct {
	byName map[string]Builtin
}

func NewBuiltinRegistry(builtins ...Builtin) *BuiltinRegistry {
	r := &BuiltinRegistry{byName: make(map[string]Builtin, len(builtins))}
	for _, b := range builtins {
		r.Register(b)
	}
	return r
}

func (r *BuiltinRegistry) Register(b Builtin) {
	if r.byName == nil {
		r.byName = make(map[string]Builtin)
	}
	r.byName[b.Name] = b
}

func (r *BuiltinRegistry) Lookup(name string) (Builtin, bool) {
	b, ok := r.byName[name]
	return b, ok
}

func (r *BuiltinRegistry) Names() []string {
	names := make([]string, 0, len(r.byName))
	for name := range r.byName {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func (r *BuiltinRegistry) All() []Builtin {
	all := make([]Builtin, 0, len(r.byName))
	for _, name := range r.Names() {
		all = append(all, r.byName[name])
	}
	return all
}

func DefaultBuiltins() *BuiltinRegistry {
	return NewBuiltinRegistry(
		Builtin{
			Name:    "help",
			Summary: "list the available commands",
			Run:     builtinHelp,
		},
		Builtin{
			Name:    "exit",
			Summary: "leave the shell",
			Run:     builtinExit,
		},
		Builtin{
			Name:    "quit",
			Summary: "leave the shell",
			Run:     builtinExit,
		},
		Builtin{
			Name:    "clear",
			Summary: "clear the screen",
			Run:     builtinClear,
		},
		Builtin{
			Name:    "history",
			Summary: "show recent commands; \"history clear\" empties the file",
			Run:     builtinHistory,
		},
	)
}

func builtinExit(s *Shell, args []string) error {
	return ErrExit
}

func builtinClear(s *Shell, args []string) error {
	_, err := fmt.Fprint(s.cfg.Out, "\033[H\033[2J")
	return err
}

func builtinHelp(s *Shell, args []string) error {
	w := tabwriter.NewWriter(s.cfg.Out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "files|folders|all from '<path>'\tlist what is in a folder")
	fmt.Fprintln(w, "count(files|folders|all) from '<path>'\tcount what is in a folder")
	fmt.Fprintln(w, "apps [where ...]\tlist the apps installed on this machine")
	fmt.Fprintln(w, "open '<path>'\tprint what is inside a text file")
	fmt.Fprintln(w, "new file|folder '<path>' [data='...']\tmake a new file or folder")
	fmt.Fprintln(w, "summary from '<path>' [recursive]\twhat is in a folder, at a glance")
	fmt.Fprintln(w, "delete file|folder '<path>'\tmove one thing to the trash")
	fmt.Fprintln(w, "delete files|folders|all from '<path>' [where ...]\tmove matching things to the trash")
	for _, b := range s.builtins.All() {
		fmt.Fprintf(w, "%s\t%s\n", b.Name, b.Summary)
	}
	return w.Flush()
}

func builtinHistory(s *Shell, args []string) error {
	if s.cfg.Store == nil {
		return errNoStore
	}

	hist, err := s.cfg.Store.History()
	if err != nil {
		return err
	}

	if len(args) > 0 && args[0] == "clear" {
		return hist.Clear()
	}
	if len(args) > 0 {
		return fmt.Errorf("history takes no argument other than \"clear\", not %q", args[0])
	}

	lines, err := hist.Lines(0)
	if err != nil {
		return err
	}
	if len(lines) == 0 {
		fmt.Fprintln(s.cfg.Out, "No history yet.")
		return nil
	}

	start := 0
	if len(lines) > historyDisplayLimit {
		start = len(lines) - historyDisplayLimit
	}
	for i, line := range lines[start:] {
		fmt.Fprintf(s.cfg.Out, "%5d  %s\n", start+i+1, line)
	}
	return nil
}
