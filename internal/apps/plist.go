package apps

import (
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

const (
	KeyBundleName        = "CFBundleName"
	KeyBundleDisplayName = "CFBundleDisplayName"
	KeyBundleVersion     = "CFBundleShortVersionString"
	KeyBundleIdentifier  = "CFBundleIdentifier"
)

var binaryMagic = []byte("bplist")

var ErrBinaryPlist = errors.New("binary plist")

func IsBinaryPlist(head []byte) bool {
	if len(head) < len(binaryMagic) {
		return false
	}
	for i := range binaryMagic {
		if head[i] != binaryMagic[i] {
			return false
		}
	}
	return true
}

func ParsePlist(r io.Reader) (map[string]string, error) {
	head := make([]byte, len(binaryMagic))
	n, err := io.ReadFull(r, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return nil, err
	}
	head = head[:n]

	if IsBinaryPlist(head) {
		return nil, ErrBinaryPlist
	}

	return parseXMLPlist(io.MultiReader(strings.NewReader(string(head)), r))
}

func parseXMLPlist(r io.Reader) (map[string]string, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = false

	if err := seekTopDict(dec); err != nil {
		return nil, err
	}

	values := map[string]string{}
	for {
		token, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return values, nil
			}
			return values, err
		}

		switch element := token.(type) {
		case xml.EndElement:
			if element.Name.Local == "dict" {
				return values, nil
			}
		case xml.StartElement:
			if element.Name.Local != "key" {
				continue
			}
			var key string
			if err := dec.DecodeElement(&key, &element); err != nil {
				return values, nil
			}
			value, err := readPlistValue(dec)
			if err != nil {
				return values, nil
			}
			values[strings.TrimSpace(key)] = value
		}
	}
}

func seekTopDict(dec *xml.Decoder) error {
	for {
		token, err := dec.Token()
		if err != nil {
			return err
		}
		if start, ok := token.(xml.StartElement); ok && start.Name.Local == "dict" {
			return nil
		}
	}
}

func readPlistValue(dec *xml.Decoder) (string, error) {
	for {
		token, err := dec.Token()
		if err != nil {
			return "", err
		}

		start, ok := token.(xml.StartElement)
		if !ok {
			if end, isEnd := token.(xml.EndElement); isEnd && end.Name.Local == "dict" {
				return "", io.EOF
			}
			continue
		}

		switch start.Name.Local {
		case "string", "integer", "real", "date":
			var text string
			if err := dec.DecodeElement(&text, &start); err != nil {
				return "", err
			}
			return strings.TrimSpace(text), nil
		case "true", "false":
			if err := dec.Skip(); err != nil {
				return "", err
			}
			return start.Name.Local, nil
		default:
			if err := dec.Skip(); err != nil {
				return "", err
			}
			return "", nil
		}
	}
}
