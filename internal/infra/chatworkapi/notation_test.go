package chatworkapi

import (
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestParseNotationKeepsOnlyExplicitRelations(t *testing.T) {
	body := "[To:12] A [rp aid=13 to=7-42] B [qt][qtmeta aid=14 time=1700000000][To:99][/qt] [code][To:98][/code]"
	recipients, reply, quotes, state := parseNotation(body)
	if state != chatwork.MessageRelationsComplete {
		t.Fatalf("state = %v, want complete", state)
	}
	if len(recipients) != 2 || recipients[0].Value != "12" || recipients[1].Value != "13" {
		t.Fatalf("recipients = %+v", recipients)
	}
	if reply == nil || reply.Target.Value != "42" || reply.ExternalID != "7" || reply.Resolved {
		t.Fatalf("reply = %+v", reply)
	}
	if len(quotes) != 1 || quotes[0].Target.Value != "14" || quotes[0].ExternalID != "1700000000" || quotes[0].Resolved {
		t.Fatalf("quotes = %+v", quotes)
	}
}

func TestParseNotationKeepsMalformedRecognizedTagsAsUnknown(t *testing.T) {
	for _, body := range []string{
		"[To:abc]", "[To]", "[rp aid=1 to=2-x]", "[rp]", "[qt][qtmeta aid=1]missing close", "[qt",
		"[code]missing close", "[code", "[/code]", "[/qt]", "[qtmeta aid=1]", "[rp aid=1 to=2-3][rp aid=1 to=2-4]",
	} {
		recipients, reply, quotes, state := parseNotation(body)
		if state != chatwork.MessageRelationsUnknown {
			t.Fatalf("parseNotation(%q) state = %v, want unknown", body, state)
		}
		if len(recipients) != 0 || reply != nil || len(quotes) != 0 {
			t.Fatalf("parseNotation(%q) returned partial facts: recipients=%+v reply=%+v quotes=%+v", body, recipients, reply, quotes)
		}
	}
}

func TestParseNotationDropsEarlierFactsWhenLaterTagIsMalformed(t *testing.T) {
	recipients, reply, quotes, state := parseNotation("[To:12] visible [qt][qtmeta aid=13]missing close")
	if state != chatwork.MessageRelationsUnknown || len(recipients) != 0 || reply != nil || len(quotes) != 0 {
		t.Fatalf("state=%v recipients=%+v reply=%+v quotes=%+v", state, recipients, reply, quotes)
	}
}
