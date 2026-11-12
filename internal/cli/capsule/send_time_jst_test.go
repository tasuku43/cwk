package capsule

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestFormatSendTimeJSTUsesFixedOffsetAndFourDigitRFC3339Bounds(t *testing.T) {
	if minimumJSTRFC3339UnixTime != -62167251600 {
		t.Fatalf("minimum fixed-JST Unix second = %d, want -62167251600", minimumJSTRFC3339UnixTime)
	}
	if maximumJSTRFC3339UnixTime != 253402268400 {
		t.Fatalf("exclusive maximum fixed-JST Unix second = %d, want 253402268400", maximumJSTRFC3339UnixTime)
	}
	tests := []struct {
		name string
		unix int64
		want string
	}{
		{name: "ordinary", unix: 1700000000, want: "2023-11-15T07:13:20+09:00"},
		{name: "JST midnight", unix: time.Date(2024, time.January, 1, 0, 0, 0, 0, fixedJSTLocation).Unix(), want: "2024-01-01T00:00:00+09:00"},
		{name: "minimum four digit year", unix: minimumJSTRFC3339UnixTime, want: "0000-01-01T00:00:00+09:00"},
		{name: "maximum four digit year", unix: maximumJSTRFC3339UnixTime - 1, want: "9999-12-31T23:59:59+09:00"},
		{name: "before minimum", unix: minimumJSTRFC3339UnixTime - 1, want: "unavailable"},
		{name: "at exclusive maximum", unix: maximumJSTRFC3339UnixTime, want: "unavailable"},
		{name: "int64 minimum", unix: math.MinInt64, want: "unavailable"},
		{name: "int64 maximum", unix: math.MaxInt64, want: "unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatSendTimeJST(test.unix); got != test.want {
				t.Fatalf("formatSendTimeJST(%d) = %q, want %q", test.unix, got, test.want)
			}
		})
	}
	if minimumJSTRFC3339UnixTime != time.Date(0, time.January, 1, 0, 0, 0, 0, fixedJSTLocation).Unix() {
		t.Fatal("lower bound does not begin at the earliest four-digit RFC 3339 JST date")
	}
	if maximumJSTRFC3339UnixTime != time.Date(10000, time.January, 1, 0, 0, 0, 0, fixedJSTLocation).Unix() {
		t.Fatal("upper bound is not exclusive at the first five-digit JST date")
	}
}

func TestFormatSendTimeJSTIgnoresAmbientLocalZone(t *testing.T) {
	original := time.Local
	time.Local = time.FixedZone("synthetic-host-zone", -11*60*60)
	defer func() { time.Local = original }()

	if got := formatSendTimeJST(1700000000); got != "2023-11-15T07:13:20+09:00" {
		t.Fatalf("formatSendTimeJST with a non-JST host zone = %q", got)
	}
}

func TestRenderMessagesListJSTIsOptionalAndCoversRowsAndRelationContext(t *testing.T) {
	result := messageFixture()
	result.Messages[0].SendTime = 1700000000
	result.Messages[0].Replies = []chatwork.Relation{{
		Kind: "reply", Target: reference(chatwork.ReferenceMessage, "102"), ExternalID: "42", Resolved: true,
	}}
	result.Messages[1].SendTime = time.Date(2024, time.January, 1, 0, 0, 0, 0, fixedJSTLocation).Unix()
	result.Messages[0].Quotes = []chatwork.Relation{{
		Kind: "quote", Target: result.Messages[1].Sender.Ref, ExternalID: "1704034800",
	}}
	result.Messages[0].QuoteMatches = []chatwork.MessageQuoteMatch{{
		State: chatwork.MessageQuoteCandidateUniqueWindow,
		Candidates: []chatwork.MessageQuoteCandidate{{
			MessageRef: result.Messages[1].Ref, Room: result.MessageRoom, Author: result.Messages[1].Sender.Ref,
			SendTime: result.Messages[1].SendTime, Sequence: 2, Scope: chatwork.MessageQuoteCandidateSourceWindow,
		}},
	}}
	result.MessageReachability = &chatwork.MessageReachability{
		OldestMessage: result.Messages[0].Ref, OldestSendTime: result.Messages[0].SendTime,
	}
	context := result.Messages[0]
	context.Ref = reference(chatwork.ReferenceMessage, "102")
	context.SendTime = math.MaxInt64
	context.Body = "fetched context"
	context.Replies = nil
	result.MessageRelationResolution = &chatwork.MessageRelationResolution{
		FetchLimit: 5, FetchAttempts: 1,
		Targets: []chatwork.MessageRelationTarget{{
			Target: context.Ref, State: chatwork.MessageRelationResolvedByFetch, Message: &context,
		}},
	}

	defaultOutput, err := Render(result)
	if err != nil {
		t.Fatal(err)
	}
	defaultWithOptions, err := RenderWithOptions(result, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if defaultOutput != defaultWithOptions || strings.Contains(defaultOutput, "send-time-jst=") {
		t.Fatal("omitted JST option changed the existing output")
	}

	got, err := RenderWithOptions(result, Options{IncludeSendTimeJST: true, MaxBodyBytes: 4, LimitMessageBodies: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`schema: #sequence message-ref actor sent [send-time-jst] [reply] [to] [quote] [quote-candidate] [relation-state] [body-truncated=true original-body-bytes=N] "body"`,
		"#1 100 a1 1700000000 send-time-jst=2023-11-15T07:13:20+09:00",
		"#2 101 a2 1704034800 send-time-jst=2024-01-01T00:00:00+09:00",
		"relation-context provenance=fetched message-ref=102 room-ref=42 sender-ref=7 sender-name=untrusted:\"Aki\" send-time=9223372036854775807 send-time-jst=unavailable",
		"body-truncated=true original-body-bytes=15",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("JST projection is missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "oldest-reachable-send-time=unavailable") {
		t.Fatal("JST projection rewrote the reachability boundary")
	}
	if !strings.Contains(got, "oldest-reachable-send-time=1700000000") || strings.Contains(got, "oldest-reachable-send-time-jst=") {
		t.Fatalf("reachability boundary did not retain its Unix-only contract:\n%s", got)
	}
	for _, outputLine := range strings.Split(got, "\n") {
		if !strings.HasPrefix(outputLine, "#1 ") {
			continue
		}
		candidate := strings.Index(outputLine, "quote-candidate=")
		if candidate < 0 || strings.Contains(outputLine[candidate:], "send-time-jst=") {
			t.Fatalf("quote candidate invented a displayed message time: %q", outputLine)
		}
	}
}

func TestRenderMessagesShowJSTCoversAnchorReplyAroundAndRelationContext(t *testing.T) {
	result := resultForTask(chatwork.TaskMessagesShow)
	anchor := result.Messages[0]
	anchor.SendTime = 1700000000
	anchor.Replies = []chatwork.Relation{{
		Kind: "reply", Target: reference(chatwork.ReferenceMessage, "102"), ExternalID: "42", Resolved: true,
	}}
	result.Messages[0] = anchor
	child := anchor
	child.Ref = reference(chatwork.ReferenceMessage, "101")
	child.SendTime = time.Date(2024, time.January, 1, 0, 0, 0, 0, fixedJSTLocation).Unix()
	child.Replies = []chatwork.Relation{{Kind: "reply", Target: anchor.Ref, ExternalID: "42", Resolved: true}}
	result.MessageShowContext = &chatwork.MessageShowContext{
		Anchor: anchor.Ref, Children: []chatwork.Message{child}, SourceSequences: []int{3}, SourceCount: 4,
		Coverage: chatwork.Coverage{Kind: "latest_window", Limit: 100, Complete: false}, Access: chatwork.MessageAccessNone,
	}
	prior := child
	prior.Ref = reference(chatwork.ReferenceMessage, "99")
	prior.SendTime = minimumJSTRFC3339UnixTime
	after := child
	after.Ref = reference(chatwork.ReferenceMessage, "103")
	after.SendTime = maximumJSTRFC3339UnixTime - 1
	result.MessageShowAround = &chatwork.MessageShowAround{
		Anchor: anchor.Ref, AnchorState: chatwork.MessageAroundAnchorInWindow, AnchorSourceSequence: 2,
		Requested: 2, Before: []chatwork.Message{prior}, BeforeSequences: []int{1}, ActualBefore: 1,
		After: []chatwork.Message{child, after}, AfterSequences: []int{3, 4}, ActualAfter: 2, SourceCount: 4,
		Coverage: chatwork.Coverage{Kind: "latest_window", Limit: 100, Complete: false}, Access: chatwork.MessageAccessNone,
	}
	fetched := child
	fetched.Ref = reference(chatwork.ReferenceMessage, "102")
	fetched.SendTime = math.MinInt64
	result.MessageRelationResolution = &chatwork.MessageRelationResolution{
		FetchLimit: 5, FetchAttempts: 1,
		Targets: []chatwork.MessageRelationTarget{{
			Target: fetched.Ref, State: chatwork.MessageRelationResolvedByFetch, Message: &fetched,
		}},
	}

	got, err := RenderWithOptions(result, Options{IncludeSendTimeJST: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"message message-ref=100 room-ref=42 sender-ref=7 sender-name=untrusted:\"Synthetic Account\" send-time=1700000000 send-time-jst=2023-11-15T07:13:20+09:00",
		"around-before source-sequence=#1 message-ref=99 room-ref=42 sender-ref=7 sender-name=untrusted:\"Synthetic Account\" send-time=",
		"send-time-jst=0000-01-01T00:00:00+09:00",
		"reply-child around-position=after source-sequence=#3 message-ref=101",
		"send-time-jst=2024-01-01T00:00:00+09:00",
		"around-after source-sequence=#4 message-ref=103",
		"send-time-jst=9999-12-31T23:59:59+09:00",
		"relation-context provenance=fetched message-ref=102 room-ref=42 sender-ref=7 sender-name=untrusted:\"Synthetic Account\" send-time=-9223372036854775808 send-time-jst=unavailable",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("show JST projection is missing %q:\n%s", want, got)
		}
	}

	// Replies without an around window use a separate rendering path.
	result.MessageShowAround = nil
	repliesOnly, err := RenderWithOptions(result, Options{IncludeSendTimeJST: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(repliesOnly, "reply-child source-sequence=#3 message-ref=101 room-ref=42 sender-ref=7 sender-name=untrusted:\"Synthetic Account\" send-time=1704034800 send-time-jst=2024-01-01T00:00:00+09:00") {
		t.Fatalf("replies-only projection lacks the child's JST timestamp:\n%s", repliesOnly)
	}
}
