package main

import (
	"encoding/json"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

// activeMessageBodyLimitFixture reuses the typed contains/context source and
// adds hostile multi-byte body text whose full value must survive semantic
// selection before the renderer applies a per-body display limit.
type activeMessageBodyLimitFixture struct {
	Source  chatwork.Result
	Request chatwork.Request
	Answer  json.RawMessage
	Limit   int
}

func messageBodyLimitFixture() activeMessageBodyLimitFixture {
	base := messageContainsFixture()
	base.Source.Messages = append([]chatwork.Message(nil), base.Source.Messages...)
	base.Source.Messages[1].Body = "Needle日🙂\n\\\u202e"
	return activeMessageBodyLimitFixture{
		Source:  base.Source,
		Request: base.Request,
		Limit:   8,
		Answer:  raw(`{"source_count":7,"candidate_count":2,"anchor_sequences":[2,4],"context_sequences":[1,3],"limited_body":{"sequence":2,"message_ref":"1302","max_body_bytes":8,"original_body_bytes":18,"visible_prefix":"Needle"},"provider_calls":1,"external_processing_calls":0,"source_complete":false}`),
	}
}
