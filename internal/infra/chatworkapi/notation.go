package chatworkapi

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

var (
	toTag     = regexp.MustCompile(`^\[To:([1-9][0-9]{0,31})\]`)
	replyTag  = regexp.MustCompile(`^\[rp aid=([1-9][0-9]{0,31}) to=([1-9][0-9]{0,31})-([1-9][0-9]{0,31})\]`)
	quoteMeta = regexp.MustCompile(`^\[qt\]\[qtmeta aid=([1-9][0-9]{0,31})(?: time=([1-9][0-9]{0,18}))?\]`)
)

// parseNotation recognizes only the reviewed provider forms needed by the
// semantic boundary. It does not infer relations from prose, names, layout, or
// time proximity, and it does not interpret copied tags inside quote/code data.
func parseNotation(body string) ([]chatwork.Reference, []chatwork.Relation, []chatwork.Relation, chatwork.MessageRelationState) {
	recipients, replies, quotes, state, _ := parseNotationWithBody(body)
	return recipients, replies, quotes, state
}

func parseNotationWithBody(body string) ([]chatwork.Reference, []chatwork.Relation, []chatwork.Relation, chatwork.MessageRelationState, *chatwork.MessageBodySegments) {
	recipients := make([]chatwork.Reference, 0)
	seenRecipients := map[string]struct{}{}
	quotes := make([]chatwork.Relation, 0)
	replies := make([]chatwork.Relation, 0)
	segments := &chatwork.MessageBodySegments{}
	textStart := 0
	appendText := func(end int) {
		if textStart < end {
			segments.Segments = append(segments.Segments, chatwork.MessageBodySegment{Kind: chatwork.MessageBodyText, Text: body[textStart:end], SourceStart: textStart, SourceEnd: end})
		}
	}

	for index := 0; index < len(body); {
		rest := body[index:]
		if strings.HasPrefix(rest, "[code]") {
			end := strings.Index(rest[len("[code]"):], "[/code]")
			if end < 0 {
				return nil, nil, nil, chatwork.MessageRelationsUnknown, nil
			}
			appendText(index)
			endIndex := index + len("[code]") + end + len("[/code]")
			segments.Segments = append(segments.Segments, chatwork.MessageBodySegment{Kind: chatwork.MessageBodyCode, Text: body[index:endIndex], SourceStart: index, SourceEnd: endIndex})
			index = endIndex
			textStart = index
			continue
		}
		if strings.HasPrefix(rest, "[qt]") {
			match := quoteMeta.FindStringSubmatch(rest)
			if match == nil {
				return nil, nil, nil, chatwork.MessageRelationsUnknown, nil
			}
			if match[2] != "" {
				if _, err := strconv.ParseInt(match[2], 10, 64); err != nil {
					return nil, nil, nil, chatwork.MessageRelationsUnknown, nil
				}
			}
			end := strings.Index(rest[len(match[0]):], "[/qt]")
			if end < 0 {
				return nil, nil, nil, chatwork.MessageRelationsUnknown, nil
			}
			account, err := chatwork.NewReference(chatwork.ReferenceAccount, match[1])
			if err != nil {
				return nil, nil, nil, chatwork.MessageRelationsUnknown, nil
			}
			quote := chatwork.Relation{Kind: "quote", Target: account, Resolved: false, ExternalID: match[2]}
			quotes = append(quotes, quote)
			appendText(index)
			quoteStart := index + len(match[0])
			segments.Removed = append(segments.Removed, chatwork.MessageBodyRemovedSpan{SourceStart: index, SourceEnd: quoteStart})
			segments.Removed = append(segments.Removed, chatwork.MessageBodyRemovedSpan{SourceStart: quoteStart + end, SourceEnd: quoteStart + end + len("[/qt]")})
			segments.Segments = append(segments.Segments, chatwork.MessageBodySegment{Kind: chatwork.MessageBodyQuote, Text: body[quoteStart : quoteStart+end], SourceStart: quoteStart, SourceEnd: quoteStart + end, Quote: &chatwork.MessageBodyQuoteMetadata{Author: account, Timestamp: match[2]}})
			index += len(match[0]) + end + len("[/qt]")
			textStart = index
			continue
		}
		if strings.HasPrefix(rest, "[To:") {
			match := toTag.FindStringSubmatch(rest)
			if match == nil {
				return nil, nil, nil, chatwork.MessageRelationsUnknown, nil
			}
			account, err := chatwork.NewReference(chatwork.ReferenceAccount, match[1])
			if err != nil {
				return nil, nil, nil, chatwork.MessageRelationsUnknown, nil
			}
			if _, exists := seenRecipients[account.Value]; !exists {
				recipients = append(recipients, account)
				seenRecipients[account.Value] = struct{}{}
			}
			appendText(index)
			segments.Removed = append(segments.Removed, chatwork.MessageBodyRemovedSpan{SourceStart: index, SourceEnd: index + len(match[0])})
			index += len(match[0])
			textStart = index
			continue
		}
		if strings.HasPrefix(rest, "[rp") {
			match := replyTag.FindStringSubmatch(rest)
			if match == nil {
				return nil, nil, nil, chatwork.MessageRelationsUnknown, nil
			}
			account, err := chatwork.NewReference(chatwork.ReferenceAccount, match[1])
			if err != nil {
				return nil, nil, nil, chatwork.MessageRelationsUnknown, nil
			}
			message, err := chatwork.NewReference(chatwork.ReferenceMessage, match[3])
			if err != nil {
				return nil, nil, nil, chatwork.MessageRelationsUnknown, nil
			}
			if err := chatwork.ValidateReference(chatwork.ReferenceRoom, match[2]); err != nil {
				return nil, nil, nil, chatwork.MessageRelationsUnknown, nil
			}
			if _, exists := seenRecipients[account.Value]; !exists {
				recipients = append(recipients, account)
				seenRecipients[account.Value] = struct{}{}
			}
			replies = append(replies, chatwork.Relation{Kind: "reply", Target: message, Resolved: false, ExternalID: match[2]})
			appendText(index)
			segments.Removed = append(segments.Removed, chatwork.MessageBodyRemovedSpan{SourceStart: index, SourceEnd: index + len(match[0])})
			index += len(match[0])
			textStart = index
			continue
		}
		if strings.HasPrefix(rest, "[To") || strings.HasPrefix(rest, "[code") ||
			strings.HasPrefix(rest, "[qt") || strings.HasPrefix(rest, "[/code]") ||
			strings.HasPrefix(rest, "[/qt]") {
			return nil, nil, nil, chatwork.MessageRelationsUnknown, nil
		}
		index++
	}
	appendText(len(body))
	return recipients, replies, quotes, chatwork.MessageRelationsComplete, segments
}
