package chatwork

import (
	"strings"
	"unicode/utf8"
)

// ContainsFold reports whether text contains query under Unicode simple case
// folding. It deliberately performs no Unicode normalization: canonically
// equivalent but byte/rune-distinct text remains distinct unless Unicode case
// folding itself relates the runes.
func ContainsFold(text, query string) bool {
	if query == "" {
		return true
	}
	if !utf8.ValidString(text) || !utf8.ValidString(query) {
		return false
	}

	textRunes := []rune(text)
	queryRunes := []rune(query)
	if len(queryRunes) > len(textRunes) {
		return false
	}
	for start := 0; start+len(queryRunes) <= len(textRunes); start++ {
		if strings.EqualFold(string(textRunes[start:start+len(queryRunes)]), query) {
			return true
		}
	}
	return false
}
