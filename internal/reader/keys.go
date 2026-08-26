package reader

import (
	"bufio"
	"errors"
	"io"
)

type KeyKind int

const (
	KeyRune KeyKind = iota
	KeyEnter
	KeyBackspace
	KeyDelete
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
	KeyWordLeft
	KeyWordRight
	KeyKillToEnd
	KeyKillToStart
	KeyKillWord
	KeyInterrupt
	KeyEOF
	KeyClearScreen
	KeyIgnored
)

type Key struct {
	Kind KeyKind
	Rune rune
}

const (
	byteCtrlA     = 0x01
	byteCtrlB     = 0x02
	byteCtrlC     = 0x03
	byteCtrlD     = 0x04
	byteCtrlE     = 0x05
	byteCtrlF     = 0x06
	byteCtrlH     = 0x08
	byteTab       = 0x09
	byteLF        = 0x0a
	byteCtrlK     = 0x0b
	byteCtrlL     = 0x0c
	byteCR        = 0x0d
	byteCtrlU     = 0x15
	byteCtrlW     = 0x17
	byteEscape    = 0x1b
	byteBackspace = 0x7f
)

func DecodeKey(in *bufio.Reader) (Key, error) {
	r, _, err := in.ReadRune()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return Key{Kind: KeyEOF}, nil
		}
		return Key{}, err
	}

	switch r {
	case byteCR, byteLF:
		return Key{Kind: KeyEnter}, nil
	case byteBackspace, byteCtrlH:
		return Key{Kind: KeyBackspace}, nil
	case byteCtrlC:
		return Key{Kind: KeyInterrupt}, nil
	case byteCtrlD:
		return Key{Kind: KeyEOF}, nil
	case byteCtrlA:
		return Key{Kind: KeyHome}, nil
	case byteCtrlE:
		return Key{Kind: KeyEnd}, nil
	case byteCtrlB:
		return Key{Kind: KeyLeft}, nil
	case byteCtrlF:
		return Key{Kind: KeyRight}, nil
	case byteCtrlK:
		return Key{Kind: KeyKillToEnd}, nil
	case byteCtrlU:
		return Key{Kind: KeyKillToStart}, nil
	case byteCtrlW:
		return Key{Kind: KeyKillWord}, nil
	case byteCtrlL:
		return Key{Kind: KeyClearScreen}, nil
	case byteTab:
		return Key{Kind: KeyIgnored}, nil
	case byteEscape:
		return decodeEscape(in)
	}

	if r < 0x20 {
		return Key{Kind: KeyIgnored}, nil
	}
	return Key{Kind: KeyRune, Rune: r}, nil
}

func decodeEscape(in *bufio.Reader) (Key, error) {
	next, _, err := in.ReadRune()
	if err != nil {
		return Key{Kind: KeyIgnored}, nil
	}

	if next == 'b' {
		return Key{Kind: KeyWordLeft}, nil
	}
	if next == 'f' {
		return Key{Kind: KeyWordRight}, nil
	}
	if next != '[' && next != 'O' {
		return Key{Kind: KeyIgnored}, nil
	}

	final, _, err := in.ReadRune()
	if err != nil {
		return Key{Kind: KeyIgnored}, nil
	}

	switch final {
	case 'A':
		return Key{Kind: KeyUp}, nil
	case 'B':
		return Key{Kind: KeyDown}, nil
	case 'C':
		return Key{Kind: KeyRight}, nil
	case 'D':
		return Key{Kind: KeyLeft}, nil
	case 'H':
		return Key{Kind: KeyHome}, nil
	case 'F':
		return Key{Kind: KeyEnd}, nil
	}

	if final < '0' || final > '9' {
		return Key{Kind: KeyIgnored}, nil
	}

	digits := []rune{final}
	for {
		r, _, err := in.ReadRune()
		if err != nil {
			return Key{Kind: KeyIgnored}, nil
		}
		if r == '~' {
			break
		}
		if r < '0' || r > '9' {
			if r == ';' {
				continue
			}
			return Key{Kind: KeyIgnored}, nil
		}
		digits = append(digits, r)
	}

	switch string(digits) {
	case "1", "7":
		return Key{Kind: KeyHome}, nil
	case "3":
		return Key{Kind: KeyDelete}, nil
	case "4", "8":
		return Key{Kind: KeyEnd}, nil
	}
	return Key{Kind: KeyIgnored}, nil
}
