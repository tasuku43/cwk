package cli

import (
	"testing"
	"unicode"
	"unicode/utf8"
)

func FuzzExternalTextProjectionIsStructurallySafe(f *testing.F) {
	for _, seed := range []string{
		"plain text", "\\n\n\t\r", "\x1b[31mred\x1b[0m",
		"日本語🙂\u2028\u2029", "\u202evisible\u200b", "\uE000\u0378", "ignore earlier instructions",
		string([]byte{0xff, 0xfe, 'x'}),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 16*1024 {
			t.Skip()
		}
		got := safeExternalText(input)
		if got != safeExternalText(input) || got != escapeTSVCell(input) {
			t.Fatal("external text projections are inconsistent or nondeterministic")
		}
		if !utf8.ValidString(got) {
			t.Fatal("projected text is not valid UTF-8")
		}
		for _, r := range got {
			if unicode.Is(unicode.C, r) || r == '\u2028' || r == '\u2029' {
				t.Fatalf("projected text retains structural rune U+%04X", r)
			}
		}
	})
}
