package chatworkapi

import (
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestMessageBodySegmentsKeepTextQuoteAndCodeInOrder(t *testing.T) {
	body := "前[To:12][rp aid=13 to=7-42]間[qt][qtmeta aid=14 time=1700000000]引用 [To:99][/qt][code][To:98][/code]後"
	recipients, replies, quotes, state, segments := parseNotationWithBody(body)
	if state != chatwork.MessageRelationsComplete || segments == nil || len(recipients) != 2 || len(replies) != 1 || len(quotes) != 1 {
		t.Fatalf("state=%v recipients=%v replies=%v quotes=%v segments=%+v", state, recipients, replies, quotes, segments)
	}
	wantKinds := []chatwork.MessageBodySegmentKind{chatwork.MessageBodyText, chatwork.MessageBodyText, chatwork.MessageBodyQuote, chatwork.MessageBodyCode, chatwork.MessageBodyText}
	wantText := []string{"前", "間", "引用 [To:99]", "[code][To:98][/code]", "後"}
	if len(segments.Segments) != len(wantKinds) {
		t.Fatalf("segments=%+v", segments.Segments)
	}
	for i, segment := range segments.Segments {
		if segment.Kind != wantKinds[i] || segment.Text != wantText[i] || body[segment.SourceStart:segment.SourceEnd] != wantText[i] {
			t.Fatalf("segment %d = %+v", i, segment)
		}
	}
	wantRemoved := []string{"[To:12]", "[rp aid=13 to=7-42]", "[qt][qtmeta aid=14 time=1700000000]", "[/qt]"}
	if len(segments.Removed) != len(wantRemoved) {
		t.Fatalf("removed=%+v", segments.Removed)
	}
	for i, span := range segments.Removed {
		if got := body[span.SourceStart:span.SourceEnd]; got != wantRemoved[i] {
			t.Fatalf("removed %d = %q, want %q", i, got, wantRemoved[i])
		}
	}
	quote := segments.Segments[2].Quote
	if quote == nil || quote.Author != quotes[0].Target || quote.Timestamp != quotes[0].ExternalID || !strings.Contains(body[segments.Segments[2].SourceEnd:], "[/qt]") {
		t.Fatalf("quote metadata=%+v", quote)
	}
}

func TestMessageBodySegmentsUnknownIsNotPartial(t *testing.T) {
	for _, body := range []string{"[To:12]text[rp]", "[qt][qtmeta aid=1]missing", "[code]missing", "[qt][qtmeta aid=1 time=9999999999999999999]quote[/qt]"} {
		_, _, _, state, segments := parseNotationWithBody(body)
		if state != chatwork.MessageRelationsUnknown || segments != nil {
			t.Fatalf("%q: state=%v segments=%+v", body, state, segments)
		}
	}
}

func TestMessageBodySegmentsRetainOriginalInMappedMessage(t *testing.T) {
	room, err := chatwork.NewReference(chatwork.ReferenceRoom, "7")
	if err != nil {
		t.Fatal(err)
	}
	body := "[To:12]日本語\n\u202e\\[unknown]"
	messages, err := mapMessages(room, []messageDTO{{MessageID: "42", Account: accountDTO{AccountID: "1", Name: "Example"}, Body: body}})
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].Body != body || messages[0].BodySegments == nil || len(messages[0].BodySegments.Segments) != 1 || messages[0].BodySegments.Segments[0].Text != "日本語\n\u202e\\[unknown]" {
		t.Fatalf("mapped=%+v", messages)
	}
}
