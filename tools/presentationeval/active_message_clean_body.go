package main

import (
	"encoding/json"
	"strings"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

// messageCleanBodyFixture is shared semantic input for a future controlled
// presentation comparison; it is not itself an agent evaluation.
type messageCleanBodyFixture struct {
	Result    chatwork.Result
	AnswerKey json.RawMessage
}

func cleanBodyFixture() messageCleanBodyFixture {
	room := ref(chatwork.ReferenceRoom, "3001")
	sender := account("2001", "Aki")
	recipient := ref(chatwork.ReferenceAccount, "2002")
	quoteAuthor := ref(chatwork.ReferenceAccount, "2004")
	body := "[To:2002]確認: [qt][qtmeta aid=2004 time=1700000000]旧案[/qt] 新案 [code][To:2003][/code]"
	toEnd := len("[To:2002]")
	quoteStart := strings.Index(body, "[qt]")
	quoteText := strings.Index(body, "旧案")
	quoteEnd := quoteText + len("旧案")
	quoteCloseEnd := quoteEnd + len("[/qt]")
	codeStart := strings.Index(body, "[code]")
	quote := chatwork.Relation{Kind: "quote", Target: quoteAuthor, ExternalID: "1700000000"}
	known := message("1001", room, sender, body, 1700000001)
	known.Recipients = []chatwork.Reference{recipient}
	known.Quotes = []chatwork.Relation{quote}
	known.BodySegments = &chatwork.MessageBodySegments{
		Segments: []chatwork.MessageBodySegment{
			{Kind: chatwork.MessageBodyText, Text: body[toEnd:quoteStart], SourceStart: toEnd, SourceEnd: quoteStart},
			{Kind: chatwork.MessageBodyQuote, Text: body[quoteText:quoteEnd], SourceStart: quoteText, SourceEnd: quoteEnd, Quote: &chatwork.MessageBodyQuoteMetadata{Author: quoteAuthor, Timestamp: "1700000000"}},
			{Kind: chatwork.MessageBodyText, Text: body[quoteCloseEnd:codeStart], SourceStart: quoteCloseEnd, SourceEnd: codeStart},
			{Kind: chatwork.MessageBodyCode, Text: body[codeStart:], SourceStart: codeStart, SourceEnd: len(body)},
		},
		Removed: []chatwork.MessageBodyRemovedSpan{{SourceEnd: toEnd}, {SourceStart: quoteStart, SourceEnd: quoteText}, {SourceStart: quoteEnd, SourceEnd: quoteCloseEnd}},
	}
	unknown := message("1002", room, sender, "[To:2002]未確定[To:bad]本文", 1700000002)
	unknown.RelationState = chatwork.MessageRelationsUnknown
	return messageCleanBodyFixture{
		Result:    chatwork.Result{Task: chatwork.TaskMessagesList, MessageRoom: room, Coverage: chatwork.Coverage{Kind: "recent-window", Limit: 100}, Messages: []chatwork.Message{known, unknown}},
		AnswerKey: raw(`{"room_ref":"3001","message_ref":"1001","sender_ref":"2001","source_body":"[To:2002]確認: [qt][qtmeta aid=2004 time=1700000000]旧案[/qt] 新案 [code][To:2003][/code]","text_before":"確認: ","text_after":" 新案 ","code_text":"[code][To:2003][/code]","recipient_ref":"2002","reply_count":0,"quote_author_ref":"2004","quote_time":"1700000000","quote_text":"旧案","new_proposal":"新案","code_literal":"[To:2003]","unknown_message_ref":"1002","unknown_body":"[To:2002]未確定[To:bad]本文"}`),
	}
}
