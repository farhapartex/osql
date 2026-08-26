package test

import (
	"errors"
	"strings"
	"testing"

	"github.com/farhapartex/osql/internal/apps"
)

const chromePlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>Chrome</string>
	<key>CFBundleShortVersionString</key>
	<string>140.0.1</string>
	<key>CFBundleIdentifier</key>
	<string>com.google.Chrome</string>
	<key>LSMinimumSystemVersion</key>
	<string>11.0</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
</dict>
</plist>
`

func TestParsePlistReadsTopLevelStrings(t *testing.T) {
	values, err := apps.ParsePlist(strings.NewReader(chromePlist))
	if err != nil {
		t.Fatalf("ParsePlist error = %v", err)
	}

	want := map[string]string{
		apps.KeyBundleName:       "Chrome",
		apps.KeyBundleVersion:    "140.0.1",
		apps.KeyBundleIdentifier: "com.google.Chrome",
	}
	for key, expected := range want {
		if values[key] != expected {
			t.Errorf("values[%q] = %q, want %q", key, values[key], expected)
		}
	}
}

func TestParsePlistIgnoresNestedDictKeys(t *testing.T) {
	nested := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>CFBundleShortVersionString</key>
	<string>2.0</string>
	<key>CFBundleURLTypes</key>
	<array>
		<dict>
			<key>CFBundleShortVersionString</key>
			<string>SHOULD-NOT-WIN</string>
		</dict>
	</array>
	<key>CFBundleIdentifier</key>
	<string>com.example.App</string>
</dict>
</plist>`

	values, err := apps.ParsePlist(strings.NewReader(nested))
	if err != nil {
		t.Fatalf("ParsePlist error = %v", err)
	}

	if values[apps.KeyBundleVersion] != "2.0" {
		t.Errorf("version = %q, want 2.0 from the top-level dict", values[apps.KeyBundleVersion])
	}
	if values[apps.KeyBundleIdentifier] != "com.example.App" {
		t.Errorf("identifier = %q; parsing must resume after a nested array", values[apps.KeyBundleIdentifier])
	}
}

func TestParsePlistHandlesNonStringValues(t *testing.T) {
	mixed := `<plist version="1.0">
<dict>
	<key>LSUIElement</key>
	<true/>
	<key>CFBundleNumericThing</key>
	<integer>42</integer>
	<key>CFBundleShortVersionString</key>
	<string>3.1</string>
</dict>
</plist>`

	values, err := apps.ParsePlist(strings.NewReader(mixed))
	if err != nil {
		t.Fatalf("ParsePlist error = %v", err)
	}

	if values[apps.KeyBundleVersion] != "3.1" {
		t.Errorf("version = %q, want 3.1 after a bool and an integer", values[apps.KeyBundleVersion])
	}
	if values["CFBundleNumericThing"] != "42" {
		t.Errorf("integer value = %q, want 42", values["CFBundleNumericThing"])
	}
}

func TestParsePlistRejectsBinaryPlist(t *testing.T) {
	binary := "bplist00\xd1\x01\x02\x5fSomething"

	if _, err := apps.ParsePlist(strings.NewReader(binary)); !errors.Is(err, apps.ErrBinaryPlist) {
		t.Errorf("ParsePlist on a binary plist gave %v, want ErrBinaryPlist", err)
	}
}

func TestIsBinaryPlist(t *testing.T) {
	tests := []struct {
		head string
		want bool
	}{
		{"bplist00", true},
		{"bplist", true},
		{"<?xml v", false},
		{"bpli", false},
		{"", false},
		{"BPLIST00", false},
	}

	for _, tt := range tests {
		if got := apps.IsBinaryPlist([]byte(tt.head)); got != tt.want {
			t.Errorf("IsBinaryPlist(%q) = %v, want %v", tt.head, got, tt.want)
		}
	}
}

func TestParsePlistOnGarbageDoesNotPanic(t *testing.T) {
	inputs := []string{
		"",
		"not xml at all",
		"<plist>",
		"<plist><dict>",
		"<plist><dict><key>Truncated</key>",
		"<plist><dict><key>A</key><string>1</string>",
		"<<<<>>>>",
		strings.Repeat("<dict>", 100),
	}

	for _, input := range inputs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("ParsePlist(%q) panicked: %v", input, r)
				}
			}()
			apps.ParsePlist(strings.NewReader(input))
		}()
	}
}

func TestParsePlistTruncatedStillReturnsWhatItRead(t *testing.T) {
	truncated := `<plist version="1.0">
<dict>
	<key>CFBundleShortVersionString</key>
	<string>9.9</string>
	<key>CFBundleIdentifier</key>`

	values, _ := apps.ParsePlist(strings.NewReader(truncated))
	if values[apps.KeyBundleVersion] != "9.9" {
		t.Errorf("version = %q, want 9.9 salvaged from a truncated plist", values[apps.KeyBundleVersion])
	}
}
