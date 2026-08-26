package reader

import "unicode"

type Line struct {
	runes  []rune
	cursor int
}

func (l *Line) String() string {
	return string(l.runes)
}

func (l *Line) Cursor() int {
	return l.cursor
}

func (l *Line) Len() int {
	return len(l.runes)
}

func (l *Line) Set(text string) {
	l.runes = []rune(text)
	l.cursor = len(l.runes)
}

func (l *Line) Clear() {
	l.runes = l.runes[:0]
	l.cursor = 0
}

func (l *Line) Insert(r rune) {
	l.runes = append(l.runes, 0)
	copy(l.runes[l.cursor+1:], l.runes[l.cursor:])
	l.runes[l.cursor] = r
	l.cursor++
}

func (l *Line) Backspace() bool {
	if l.cursor == 0 {
		return false
	}
	l.runes = append(l.runes[:l.cursor-1], l.runes[l.cursor:]...)
	l.cursor--
	return true
}

func (l *Line) DeleteForward() bool {
	if l.cursor >= len(l.runes) {
		return false
	}
	l.runes = append(l.runes[:l.cursor], l.runes[l.cursor+1:]...)
	return true
}

func (l *Line) Left() bool {
	if l.cursor == 0 {
		return false
	}
	l.cursor--
	return true
}

func (l *Line) Right() bool {
	if l.cursor >= len(l.runes) {
		return false
	}
	l.cursor++
	return true
}

func (l *Line) Home() {
	l.cursor = 0
}

func (l *Line) End() {
	l.cursor = len(l.runes)
}

func (l *Line) KillToEnd() {
	l.runes = l.runes[:l.cursor]
}

func (l *Line) KillToStart() {
	l.runes = append([]rune{}, l.runes[l.cursor:]...)
	l.cursor = 0
}

func (l *Line) KillWord() {
	end := l.cursor
	for end > 0 && unicode.IsSpace(l.runes[end-1]) {
		end--
	}
	for end > 0 && !unicode.IsSpace(l.runes[end-1]) {
		end--
	}
	l.runes = append(l.runes[:end], l.runes[l.cursor:]...)
	l.cursor = end
}

func (l *Line) WordLeft() {
	for l.cursor > 0 && unicode.IsSpace(l.runes[l.cursor-1]) {
		l.cursor--
	}
	for l.cursor > 0 && !unicode.IsSpace(l.runes[l.cursor-1]) {
		l.cursor--
	}
}

func (l *Line) WordRight() {
	for l.cursor < len(l.runes) && unicode.IsSpace(l.runes[l.cursor]) {
		l.cursor++
	}
	for l.cursor < len(l.runes) && !unicode.IsSpace(l.runes[l.cursor]) {
		l.cursor++
	}
}
