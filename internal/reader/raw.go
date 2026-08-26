package reader

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/farhapartex/osql/internal/textwidth"
)

const historyLoadLimit = 1000

type HistoryStore interface {
	HistoryAppender
	Lines(limit int) ([]string, error)
}

type Raw struct {
	term    *terminal
	in      *bufio.Reader
	out     io.Writer
	history HistoryStore
	past    []string
	browse  int
	draft   string
	lastErr error
}

func NewRaw(file *os.File, out io.Writer, history HistoryStore) (*Raw, error) {
	term, err := openTerminal(file)
	if err != nil {
		return nil, err
	}

	r := &Raw{
		term:    term,
		in:      bufio.NewReader(file),
		out:     out,
		history: history,
	}
	r.loadHistory()
	r.browse = len(r.past)
	return r, nil
}

func (r *Raw) loadHistory() {
	if r.history == nil {
		return
	}
	lines, err := r.history.Lines(historyLoadLimit)
	if err != nil {
		return
	}
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			r.past = append(r.past, line)
		}
	}
}

func (r *Raw) ReadLine(prompt string) (string, error) {
	if err := r.term.enterRaw(); err != nil {
		return "", err
	}
	defer r.term.restore()

	line := &Line{}
	r.browse = len(r.past)
	r.draft = ""
	r.render(prompt, line)

	for {
		key, err := DecodeKey(r.in)
		if err != nil {
			return "", err
		}

		switch key.Kind {
		case KeyEnter:
			fmt.Fprint(r.out, "\r\n")
			return line.String(), nil

		case KeyEOF:
			if line.Len() > 0 {
				line.DeleteForward()
				break
			}
			fmt.Fprint(r.out, "\r\n")
			return "", io.EOF

		case KeyInterrupt:
			fmt.Fprint(r.out, "^C\r\n")
			line.Clear()
			r.browse = len(r.past)
			r.render(prompt, line)
			continue

		case KeyRune:
			line.Insert(key.Rune)
		case KeyBackspace:
			line.Backspace()
		case KeyDelete:
			line.DeleteForward()
		case KeyLeft:
			line.Left()
		case KeyRight:
			line.Right()
		case KeyHome:
			line.Home()
		case KeyEnd:
			line.End()
		case KeyWordLeft:
			line.WordLeft()
		case KeyWordRight:
			line.WordRight()
		case KeyKillToEnd:
			line.KillToEnd()
		case KeyKillToStart:
			line.KillToStart()
		case KeyKillWord:
			line.KillWord()
		case KeyUp:
			r.recallOlder(line)
		case KeyDown:
			r.recallNewer(line)
		case KeyClearScreen:
			fmt.Fprint(r.out, "\033[H\033[2J")
		case KeyIgnored:
			continue
		}

		r.render(prompt, line)
	}
}

func (r *Raw) recallOlder(line *Line) {
	if len(r.past) == 0 || r.browse == 0 {
		return
	}
	if r.browse == len(r.past) {
		r.draft = line.String()
	}
	r.browse--
	line.Set(r.past[r.browse])
}

func (r *Raw) recallNewer(line *Line) {
	if r.browse >= len(r.past) {
		return
	}
	r.browse++
	if r.browse == len(r.past) {
		line.Set(r.draft)
		return
	}
	line.Set(r.past[r.browse])
}

func (r *Raw) render(prompt string, line *Line) {
	text := line.String()
	fmt.Fprint(r.out, "\r\033[K"+prompt+text)

	if back := textwidth.Of(text) - textwidth.Of(string([]rune(text)[:line.Cursor()])); back > 0 {
		fmt.Fprintf(r.out, "\033[%dD", back)
	}
}

func (r *Raw) AddHistory(line string) {
	if len(r.past) == 0 || r.past[len(r.past)-1] != line {
		r.past = append(r.past, line)
	}
	r.browse = len(r.past)

	if r.history == nil {
		return
	}
	if err := r.history.Append(line); err != nil {
		r.lastErr = err
	}
}

func (r *Raw) Err() error {
	return r.lastErr
}

func (r *Raw) Close() error {
	return r.term.restore()
}
