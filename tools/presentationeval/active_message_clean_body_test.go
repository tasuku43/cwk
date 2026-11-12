package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestCleanBodyFixturePreservesSemanticAnswersAndUncertainty(t *testing.T) {
	fixture := cleanBodyFixture()
	if err := fixture.Result.Validate(); err != nil {
		t.Fatal(err)
	}
	var answer struct {
		Room           string `json:"room_ref"`
		Message        string `json:"message_ref"`
		Sender         string `json:"sender_ref"`
		SourceBody     string `json:"source_body"`
		TextBefore     string `json:"text_before"`
		TextAfter      string `json:"text_after"`
		CodeText       string `json:"code_text"`
		Recipient      string `json:"recipient_ref"`
		ReplyCount     *int   `json:"reply_count"`
		QuoteAuthor    string `json:"quote_author_ref"`
		QuoteTime      string `json:"quote_time"`
		QuoteText      string `json:"quote_text"`
		NewProposal    string `json:"new_proposal"`
		CodeLiteral    string `json:"code_literal"`
		UnknownMessage string `json:"unknown_message_ref"`
		UnknownBody    string `json:"unknown_body"`
	}
	if err := json.Unmarshal(fixture.AnswerKey, &answer); err != nil {
		t.Fatal(err)
	}
	known, unknown := fixture.Result.Messages[0], fixture.Result.Messages[1]
	if fixture.Result.MessageRoom.Value != answer.Room || known.Ref.Value != answer.Message || known.Sender.Ref.Value != answer.Sender || known.Body != answer.SourceBody || len(known.Recipients) != 1 || known.Recipients[0].Value != answer.Recipient {
		t.Fatal("canonical refs or recipients differ from answer")
	}
	segments := known.BodySegments.Segments
	if len(segments) != 4 || !reflect.DeepEqual([]chatwork.MessageBodySegmentKind{segments[0].Kind, segments[1].Kind, segments[2].Kind, segments[3].Kind}, []chatwork.MessageBodySegmentKind{chatwork.MessageBodyText, chatwork.MessageBodyQuote, chatwork.MessageBodyText, chatwork.MessageBodyCode}) || segments[0].Text != answer.TextBefore || segments[1].Text != answer.QuoteText || segments[1].Quote == nil || segments[1].Quote.Author.Value != answer.QuoteAuthor || segments[1].Quote.Timestamp != answer.QuoteTime || segments[2].Text != answer.TextAfter || segments[3].Text != answer.CodeText || !strings.Contains(segments[2].Text, answer.NewProposal) || !strings.Contains(segments[3].Text, answer.CodeLiteral) {
		t.Fatalf("semantic segments = %+v", segments)
	}
	if len(known.Recipients) != 1 || answer.ReplyCount == nil || len(known.Replies) != *answer.ReplyCount || known.Quotes[0].Resolved {
		t.Fatal("code notation created a recipient or resolved quote")
	}
	if unknown.Ref.Value != answer.UnknownMessage || unknown.Body != answer.UnknownBody || unknown.RelationState != chatwork.MessageRelationsUnknown || unknown.BodySegments != nil || len(unknown.Recipients) != 0 || len(unknown.Replies) != 0 || len(unknown.Quotes) != 0 {
		t.Fatalf("uncertainty = %+v", unknown)
	}
}
