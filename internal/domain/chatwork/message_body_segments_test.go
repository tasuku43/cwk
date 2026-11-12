package chatwork

import "testing"

func TestMessageBodySegmentValidation(t *testing.T) {
	messageRef, _ := NewReference(ReferenceMessage, "42")
	roomRef, _ := NewReference(ReferenceRoom, "7")
	accountRef, _ := NewReference(ReferenceAccount, "14")
	quote := Relation{Kind: "quote", Target: accountRef, ExternalID: "123"}
	base := Message{Ref: messageRef, Room: roomRef, Sender: Account{Ref: accountRef}, Body: "abc", Quotes: []Relation{quote}, BodySegments: &MessageBodySegments{Segments: []MessageBodySegment{{Kind: MessageBodyQuote, Text: "abc", SourceStart: 0, SourceEnd: 3, Quote: &MessageBodyQuoteMetadata{Author: accountRef, Timestamp: "123"}}}}}
	if err := validateResultMessage("message", base); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Message){
		"unknown with segments": func(m *Message) { m.RelationState = MessageRelationsUnknown; m.Quotes = nil },
		"unknown kind":          func(m *Message) { m.BodySegments.Segments[0].Kind = "unknown" },
		"metadata on text":      func(m *Message) { m.BodySegments.Segments[0].Kind = MessageBodyText },
		"wrong quote":           func(m *Message) { m.BodySegments.Segments[0].Quote.Timestamp = "124" },
		"uncovered source":      func(m *Message) { m.BodySegments.Segments[0].Text = "a"; m.BodySegments.Segments[0].SourceEnd = 1 },
		"wrong source":          func(m *Message) { m.BodySegments.Segments[0].SourceEnd = 2 },
		"missing quote":         func(m *Message) { m.BodySegments.Segments = nil },
	} {
		t.Run(name, func(t *testing.T) {
			m := base
			segment := base.BodySegments.Segments[0]
			metadata := *segment.Quote
			segment.Quote = &metadata
			m.BodySegments = &MessageBodySegments{Segments: []MessageBodySegment{segment}}
			mutate(&m)
			if err := validateResultMessage("message", m); err == nil {
				t.Fatal("accepted invalid message body segments")
			}
		})
	}
}

func TestMessageBodySegmentCoverageWithRemovedSpans(t *testing.T) {
	messageRef, _ := NewReference(ReferenceMessage, "42")
	roomRef, _ := NewReference(ReferenceRoom, "7")
	accountRef, _ := NewReference(ReferenceAccount, "14")
	message := Message{Ref: messageRef, Room: roomRef, Sender: Account{Ref: accountRef}, Body: "xAy", BodySegments: &MessageBodySegments{
		Segments: []MessageBodySegment{{Kind: MessageBodyText, Text: "x", SourceEnd: 1}, {Kind: MessageBodyText, Text: "y", SourceStart: 2, SourceEnd: 3}},
		Removed:  []MessageBodyRemovedSpan{{SourceStart: 1, SourceEnd: 2}},
	}}
	if err := validateResultMessage("message", message); err != nil {
		t.Fatal(err)
	}
	for name, spans := range map[string][]MessageBodyRemovedSpan{
		"missing":   nil,
		"overlap":   {{SourceStart: 0, SourceEnd: 2}},
		"shifted":   {{SourceStart: 2, SourceEnd: 3}},
		"duplicate": {{SourceStart: 1, SourceEnd: 2}, {SourceStart: 1, SourceEnd: 2}},
	} {
		t.Run(name, func(t *testing.T) {
			copyMessage := message
			copyBody := *message.BodySegments
			copyBody.Removed = spans
			copyMessage.BodySegments = &copyBody
			if err := validateResultMessage("message", copyMessage); err == nil {
				t.Fatal("accepted invalid source coverage")
			}
		})
	}
}
