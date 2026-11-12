package capsule

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestRenderMessageBodyLimitGolden(t *testing.T) {
	result := messageFixture()
	result.Messages[0].Body = "A日🙂Z"
	result.Messages[1].Body = "OK"
	got, err := RenderWithOptions(result, Options{MaxBodyBytes: 4, LimitMessageBodies: true})
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/messages-body-limited.golden")
	if err != nil {
		t.Fatalf("read body-limit golden: %v\n--- generated output ---\n%s", err, got)
	}
	if got != string(want) {
		t.Fatalf("body-limited projection mismatch\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}

func TestRenderMessageBodyLimitUsesUTF8BytePrefixAndReportsOriginalLength(t *testing.T) {
	result := messageFixture()
	result.Messages[0].Body = "A日本🙂Z"
	result.Messages[1].Body = "Acknowledged"
	originalBytes := len(result.Messages[0].Body)
	tests := []struct {
		name string
		max  int
		body string
		cut  bool
	}{
		{name: "zero", max: 0, body: "", cut: true},
		{name: "one byte keeps only first ASCII rune", max: 1, body: "A", cut: true},
		{name: "inside Japanese rune", max: 2, body: "A", cut: true},
		{name: "exact Japanese boundary", max: 4, body: "A日", cut: true},
		{name: "inside emoji", max: 6, body: "A日", cut: true},
		{name: "full value boundary", max: originalBytes, body: "A日本🙂Z", cut: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := RenderWithOptions(result, Options{MaxBodyBytes: test.max, LimitMessageBodies: true})
			if err != nil {
				t.Fatal(err)
			}
			var firstRecord string
			for _, outputLine := range strings.Split(got, "\n") {
				if strings.HasPrefix(outputLine, "#1 100 ") {
					firstRecord = outputLine
					break
				}
			}
			wantBody := ` ` + quoted(test.body)
			if !strings.Contains(firstRecord, wantBody) {
				t.Fatalf("prefix %q is absent from first message record %q:\n%s", test.body, firstRecord, got)
			}
			marker := fmt.Sprintf("body-truncated=true original-body-bytes=%d", originalBytes)
			if test.cut && !strings.Contains(firstRecord, marker) {
				t.Fatalf("truncation metadata %q is absent:\n%s", marker, got)
			}
			if !test.cut && strings.Contains(firstRecord, "body-truncated=true") {
				t.Fatalf("untruncated body received truncation metadata:\n%s", got)
			}
		})
	}
}

func TestRenderMessageBodyLimitEscapesOnlyAfterRawByteTruncation(t *testing.T) {
	result := messageFixture()
	result.Messages[0].Body = "x\\\n\u202eY"
	got, err := RenderWithOptions(result, Options{MaxBodyBytes: 6, LimitMessageBodies: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "body-truncated=true original-body-bytes=7") {
		t.Fatalf("original raw UTF-8 byte count was not preserved:\n%s", got)
	}
	if strings.Contains(got, "\u202e") || strings.Contains(got, "x\\\n\u202e") {
		t.Fatalf("hostile body controls or bidi text were emitted raw:\n%s", got)
	}
	if !strings.Contains(got, `\\`) || !strings.Contains(got, `\u202E`) {
		t.Fatalf("backslash or bidi text was not visibly escaped:\n%s", got)
	}
}

func TestRenderMessageBodyLimitLeavesTypedRelationsCompletenessAndDefaultBytesAlone(t *testing.T) {
	result := messageFixture()
	baseline, err := Render(result)
	if err != nil {
		t.Fatal(err)
	}
	unlimited, err := RenderWithOptions(result, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if baseline != unlimited {
		t.Fatal("unset body limit changed default output")
	}

	result.Messages[0].Body = "needle body longer than the presentation cap"
	originalBytes := len(result.Messages[0].Body)
	result.MessageSelection = &chatwork.MessageSelection{
		Filter:          chatwork.MessageFilter{Contains: []string{"needle"}, Context: chatwork.MessageContextReplies},
		SourceCount:     2,
		CandidateCount:  1,
		SourceSequences: []int{1, 2},
		AnchorSequences: []int{1},
	}
	result.Messages[1].Replies = []chatwork.Relation{{Kind: "reply", Target: result.Messages[0].Ref, Resolved: true}}
	got, err := RenderWithOptions(result, Options{MaxBodyBytes: 1, LimitMessageBodies: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		" complete=false ",
		"candidate-count=1",
		"contains=[\"needle\"] context=replies anchors=[#1]",
		"#2 101 a2 1720000010 reply=#1",
		fmt.Sprintf("body-truncated=true original-body-bytes=%d", originalBytes),
	} {
		if !strings.Contains(got, want) {
			t.Errorf("truncated output lacks unchanged semantic fact %q:\n%s", want, got)
		}
	}
}

func TestRenderMessageBodyLimitAlsoCoversEmittedRelationContext(t *testing.T) {
	result := messageFixture()
	child := result.Messages[1]
	parentRef := reference(chatwork.ReferenceMessage, "99")
	child.Replies = []chatwork.Relation{{Kind: "reply", Target: parentRef, ExternalID: result.MessageRoom.Value, Resolved: true}}
	result.Messages = []chatwork.Message{child}
	parent := chatwork.Message{
		Ref: parentRef, Room: result.MessageRoom, Sender: child.Sender,
		Body: "parent日本語", SendTime: 1719999990,
	}
	result.MessageRelationResolution = &chatwork.MessageRelationResolution{
		FetchLimit: 1, FetchAttempts: 1,
		Targets: []chatwork.MessageRelationTarget{{
			Target: parentRef, State: chatwork.MessageRelationResolvedByFetch, Message: &parent,
		}},
	}

	got, err := RenderWithOptions(result, Options{MaxBodyBytes: 5, LimitMessageBodies: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `relation-context provenance=fetched message-ref=99`) ||
		!strings.Contains(got, "body-truncated=true original-body-bytes=15 body=untrusted:\"paren\"") {
		t.Fatalf("supplemental relation body did not use the same display limit:\n%s", got)
	}
}
