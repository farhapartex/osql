package test

import (
	"bufio"
	"strings"
	"testing"

	"github.com/farhapartex/osql/internal/reader"
)

func lineWith(text string, cursor int) *reader.Line {
	l := &reader.Line{}
	l.Set(text)
	for l.Cursor() > cursor {
		l.Left()
	}
	return l
}

func TestLineInsert(t *testing.T) {
	l := &reader.Line{}
	for _, r := range "abc" {
		l.Insert(r)
	}

	if l.String() != "abc" {
		t.Errorf("String() = %q", l.String())
	}
	if l.Cursor() != 3 {
		t.Errorf("Cursor() = %d, want 3", l.Cursor())
	}
}

func TestLineInsertInTheMiddle(t *testing.T) {
	l := lineWith("abc", 1)
	l.Insert('X')

	if l.String() != "aXbc" {
		t.Errorf("String() = %q, want \"aXbc\"", l.String())
	}
	if l.Cursor() != 2 {
		t.Errorf("Cursor() = %d, want 2", l.Cursor())
	}
}

func TestLineBackspace(t *testing.T) {
	tests := []struct {
		text   string
		cursor int
		want   string
		ok     bool
	}{
		{"abc", 3, "ab", true},
		{"abc", 2, "ac", true},
		{"abc", 1, "bc", true},
		{"abc", 0, "abc", false},
		{"", 0, "", false},
	}

	for _, tt := range tests {
		l := lineWith(tt.text, tt.cursor)
		if got := l.Backspace(); got != tt.ok {
			t.Errorf("Backspace on %q at %d = %v, want %v", tt.text, tt.cursor, got, tt.ok)
		}
		if l.String() != tt.want {
			t.Errorf("Backspace on %q at %d gave %q, want %q", tt.text, tt.cursor, l.String(), tt.want)
		}
	}
}

func TestLineDeleteForward(t *testing.T) {
	l := lineWith("abc", 1)
	if !l.DeleteForward() {
		t.Error("DeleteForward returned false mid-line")
	}
	if l.String() != "ac" {
		t.Errorf("String() = %q, want \"ac\"", l.String())
	}
	if l.Cursor() != 1 {
		t.Errorf("Cursor moved to %d; delete-forward leaves it alone", l.Cursor())
	}

	end := lineWith("abc", 3)
	if end.DeleteForward() {
		t.Error("DeleteForward at the end should do nothing")
	}
}

func TestLineMovement(t *testing.T) {
	l := lineWith("abc", 3)

	if !l.Left() || l.Cursor() != 2 {
		t.Errorf("Left gave cursor %d", l.Cursor())
	}
	if !l.Right() || l.Cursor() != 3 {
		t.Errorf("Right gave cursor %d", l.Cursor())
	}
	if l.Right() {
		t.Error("Right past the end should return false")
	}

	l.Home()
	if l.Cursor() != 0 {
		t.Errorf("Home gave cursor %d", l.Cursor())
	}
	if l.Left() {
		t.Error("Left before the start should return false")
	}

	l.End()
	if l.Cursor() != 3 {
		t.Errorf("End gave cursor %d", l.Cursor())
	}
}

func TestLineKilling(t *testing.T) {
	toEnd := lineWith("abcdef", 3)
	toEnd.KillToEnd()
	if toEnd.String() != "abc" {
		t.Errorf("KillToEnd = %q, want \"abc\"", toEnd.String())
	}

	toStart := lineWith("abcdef", 3)
	toStart.KillToStart()
	if toStart.String() != "def" {
		t.Errorf("KillToStart = %q, want \"def\"", toStart.String())
	}
	if toStart.Cursor() != 0 {
		t.Errorf("KillToStart left cursor at %d, want 0", toStart.Cursor())
	}
}

func TestLineKillWord(t *testing.T) {
	tests := []struct {
		text   string
		cursor int
		want   string
	}{
		{"one two", 7, "one "},
		{"one two ", 8, "one "},
		{"one", 3, ""},
		{"", 0, ""},
		{"one two three", 7, "one  three"},
	}

	for _, tt := range tests {
		l := lineWith(tt.text, tt.cursor)
		l.KillWord()
		if l.String() != tt.want {
			t.Errorf("KillWord on %q at %d = %q, want %q", tt.text, tt.cursor, l.String(), tt.want)
		}
	}
}

func TestLineWordMovement(t *testing.T) {
	l := lineWith("one two three", 13)

	l.WordLeft()
	if l.Cursor() != 8 {
		t.Errorf("WordLeft gave cursor %d, want 8", l.Cursor())
	}
	l.WordLeft()
	if l.Cursor() != 4 {
		t.Errorf("second WordLeft gave cursor %d, want 4", l.Cursor())
	}
	l.WordRight()
	if l.Cursor() != 7 {
		t.Errorf("WordRight gave cursor %d, want 7", l.Cursor())
	}
}

func TestLineHandlesWideCharacters(t *testing.T) {
	l := &reader.Line{}
	for _, r := range "日本語" {
		l.Insert(r)
	}

	if l.Len() != 3 {
		t.Errorf("Len() = %d, want 3 runes", l.Len())
	}
	l.Left()
	l.Insert('X')
	if l.String() != "日本X語" {
		t.Errorf("String() = %q, want 日本X語", l.String())
	}
}

func TestLineSetAndClear(t *testing.T) {
	l := &reader.Line{}
	l.Set("recalled")

	if l.String() != "recalled" || l.Cursor() != 8 {
		t.Errorf("Set gave %q cursor %d", l.String(), l.Cursor())
	}

	l.Clear()
	if l.String() != "" || l.Cursor() != 0 {
		t.Errorf("Clear gave %q cursor %d", l.String(), l.Cursor())
	}
}

func decode(t *testing.T, input string) []reader.Key {
	t.Helper()

	in := bufio.NewReader(strings.NewReader(input))
	keys := []reader.Key{}
	for {
		key, err := reader.DecodeKey(in)
		if err != nil {
			t.Fatalf("DecodeKey error = %v", err)
		}
		if key.Kind == reader.KeyEOF {
			return keys
		}
		keys = append(keys, key)
	}
}

func TestDecodeArrowKeys(t *testing.T) {
	tests := []struct {
		input string
		want  reader.KeyKind
	}{
		{"\x1b[A", reader.KeyUp},
		{"\x1b[B", reader.KeyDown},
		{"\x1b[C", reader.KeyRight},
		{"\x1b[D", reader.KeyLeft},
		{"\x1bOA", reader.KeyUp},
		{"\x1bOD", reader.KeyLeft},
		{"\x1b[H", reader.KeyHome},
		{"\x1b[F", reader.KeyEnd},
		{"\x1b[1~", reader.KeyHome},
		{"\x1b[4~", reader.KeyEnd},
		{"\x1b[3~", reader.KeyDelete},
	}

	for _, tt := range tests {
		t.Run(strings.ReplaceAll(tt.input, "\x1b", "ESC"), func(t *testing.T) {
			keys := decode(t, tt.input)
			if len(keys) != 1 {
				t.Fatalf("decoded %d keys, want 1: %+v", len(keys), keys)
			}
			if keys[0].Kind != tt.want {
				t.Errorf("Kind = %v, want %v", keys[0].Kind, tt.want)
			}
		})
	}
}

func TestDecodeControlKeys(t *testing.T) {
	tests := []struct {
		input string
		want  reader.KeyKind
	}{
		{"\r", reader.KeyEnter},
		{"\n", reader.KeyEnter},
		{"\x7f", reader.KeyBackspace},
		{"\x08", reader.KeyBackspace},
		{"\x03", reader.KeyInterrupt},
		{"\x01", reader.KeyHome},
		{"\x05", reader.KeyEnd},
		{"\x02", reader.KeyLeft},
		{"\x06", reader.KeyRight},
		{"\x0b", reader.KeyKillToEnd},
		{"\x15", reader.KeyKillToStart},
		{"\x17", reader.KeyKillWord},
		{"\x0c", reader.KeyClearScreen},
		{"\x1bb", reader.KeyWordLeft},
		{"\x1bf", reader.KeyWordRight},
	}

	for _, tt := range tests {
		keys := decode(t, tt.input)
		if len(keys) != 1 {
			t.Fatalf("decoded %d keys for %q", len(keys), tt.input)
		}
		if keys[0].Kind != tt.want {
			t.Errorf("%q decoded as %v, want %v", tt.input, keys[0].Kind, tt.want)
		}
	}
}

func TestDecodePlainText(t *testing.T) {
	keys := decode(t, "abc")

	if len(keys) != 3 {
		t.Fatalf("decoded %d keys, want 3", len(keys))
	}
	for i, want := range []rune{'a', 'b', 'c'} {
		if keys[i].Kind != reader.KeyRune || keys[i].Rune != want {
			t.Errorf("key %d = %+v, want rune %q", i, keys[i], want)
		}
	}
}

func TestDecodeMultiByteRunes(t *testing.T) {
	keys := decode(t, "日本")

	if len(keys) != 2 {
		t.Fatalf("decoded %d keys, want 2", len(keys))
	}
	if keys[0].Rune != '日' || keys[1].Rune != '本' {
		t.Errorf("decoded %+v; multi-byte runes must arrive whole", keys)
	}
}

func TestDecodeMixedSequence(t *testing.T) {
	keys := decode(t, "ab\x1b[Dc\x7f\r")

	want := []reader.KeyKind{
		reader.KeyRune, reader.KeyRune, reader.KeyLeft,
		reader.KeyRune, reader.KeyBackspace, reader.KeyEnter,
	}
	if len(keys) != len(want) {
		t.Fatalf("decoded %d keys, want %d: %+v", len(keys), len(want), keys)
	}
	for i := range want {
		if keys[i].Kind != want[i] {
			t.Errorf("key %d = %v, want %v", i, keys[i].Kind, want[i])
		}
	}
}

func TestDecodeUnknownEscapesAreIgnoredNotPrinted(t *testing.T) {
	for _, input := range []string{"\x1b[Z", "\x1b[200~", "\x1bX", "\x1b"} {
		keys := decode(t, input)
		for _, key := range keys {
			if key.Kind == reader.KeyRune {
				t.Errorf("%q produced a printable rune %q; unknown escapes must not leak into the line", input, key.Rune)
			}
		}
	}
}

func TestDecodeTabIsIgnoredForNow(t *testing.T) {
	keys := decode(t, "\t")

	if len(keys) != 1 || keys[0].Kind != reader.KeyIgnored {
		t.Errorf("tab decoded as %+v, want ignored", keys)
	}
}
