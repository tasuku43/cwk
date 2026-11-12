package chatworkapi

import (
	"reflect"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func FuzzParseNotation(f *testing.F) {
	for _, seed := range []string{
		"plain text", "[To:12] hello", "[rp aid=7 to=42-101] reply",
		"[qt][qtmeta aid=7 time=1700000000]quote[/qt]",
		"[code][To:12][/code]", "[To:12] [qt][qtmeta aid=7]unclosed",
		"\x1b[31m\u202e\\ [rp aid=7 to=42-101]",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, body string) {
		if len(body) > 16*1024 {
			t.Skip()
		}
		recipients, replies, quotes, state := parseNotation(body)
		againRecipients, againReplies, againQuotes, againState := parseNotation(body)
		if state != againState || !reflect.DeepEqual(recipients, againRecipients) ||
			!reflect.DeepEqual(replies, againReplies) || !reflect.DeepEqual(quotes, againQuotes) {
			t.Fatal("notation parsing is not deterministic")
		}
		if state == chatwork.MessageRelationsUnknown {
			if len(recipients) != 0 || len(replies) != 0 || len(quotes) != 0 {
				t.Fatal("unknown relation set contains partial facts")
			}
			return
		}
		if state != chatwork.MessageRelationsComplete {
			t.Fatalf("unrecognized relation state: %q", state)
		}
		seen := make(map[string]bool, len(recipients))
		for _, recipient := range recipients {
			if err := chatwork.ValidateReference(chatwork.ReferenceAccount, recipient.Value); err != nil || recipient.Kind != chatwork.ReferenceAccount || seen[recipient.Value] {
				t.Fatalf("invalid or duplicate recipient: %+v", recipient)
			}
			seen[recipient.Value] = true
		}
		for _, reply := range replies {
			if reply.Kind != "reply" || reply.Resolved || reply.Target.Kind != chatwork.ReferenceMessage || chatwork.ValidateReference(chatwork.ReferenceMessage, reply.Target.Value) != nil ||
				chatwork.ValidateReference(chatwork.ReferenceRoom, reply.ExternalID) != nil {
				t.Fatalf("invalid explicit reply: %+v", reply)
			}
		}
		for _, quote := range quotes {
			if quote.Kind != "quote" || quote.Resolved || quote.Target.Kind != chatwork.ReferenceAccount || chatwork.ValidateReference(chatwork.ReferenceAccount, quote.Target.Value) != nil {
				t.Fatalf("invalid explicit quote: %+v", quote)
			}
		}
	})
}

func FuzzParseNotationPreservesExactReferences(f *testing.F) {
	f.Add("7", "42", "101")
	f.Add("12345678901234567890123456789012", "1", "9")
	f.Fuzz(func(t *testing.T, account, room, message string) {
		if chatwork.ValidateReference(chatwork.ReferenceAccount, account) != nil ||
			chatwork.ValidateReference(chatwork.ReferenceRoom, room) != nil ||
			chatwork.ValidateReference(chatwork.ReferenceMessage, message) != nil {
			t.Skip()
		}
		body := "[To:" + account + "] [rp aid=" + account + " to=" + room + "-" + message + "] " +
			"[qt][qtmeta aid=" + account + " time=1700000000]quoted text[/qt]"
		recipients, replies, quotes, state := parseNotation(body)
		if state != chatwork.MessageRelationsComplete || len(recipients) != 1 || len(replies) != 1 || len(quotes) != 1 {
			t.Fatalf("valid tags produced unexpected relation counts or state: %d, %d, %d, %q", len(recipients), len(replies), len(quotes), state)
		}
		if recipients[0].Value != account || replies[0].Target.Value != message || replies[0].ExternalID != room ||
			quotes[0].Target.Value != account || quotes[0].ExternalID != "1700000000" {
			t.Fatal("parsed references differ from the exact tag values")
		}
	})
}
