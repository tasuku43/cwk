package chatworkapi

import (
	"regexp"
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
	recipients := make([]chatwork.Reference, 0)
	seenRecipients := map[string]struct{}{}
	quotes := make([]chatwork.Relation, 0)
	replies := make([]chatwork.Relation, 0)

	for index := 0; index < len(body); {
		rest := body[index:]
		if strings.HasPrefix(rest, "[code]") {
			end := strings.Index(rest[len("[code]"):], "[/code]")
			if end < 0 {
				return unknownNotation()
			}
			index += len("[code]") + end + len("[/code]")
			continue
		}
		if strings.HasPrefix(rest, "[qt]") {
			match := quoteMeta.FindStringSubmatch(rest)
			if match == nil {
				return unknownNotation()
			}
			end := strings.Index(rest[len(match[0]):], "[/qt]")
			if end < 0 {
				return unknownNotation()
			}
			account, err := chatwork.NewReference(chatwork.ReferenceAccount, match[1])
			if err != nil {
				return unknownNotation()
			}
			quotes = append(quotes, chatwork.Relation{Kind: "quote", Target: account, Resolved: false, ExternalID: match[2]})
			index += len(match[0]) + end + len("[/qt]")
			continue
		}
		if strings.HasPrefix(rest, "[To:") {
			match := toTag.FindStringSubmatch(rest)
			if match == nil {
				return unknownNotation()
			}
			account, err := chatwork.NewReference(chatwork.ReferenceAccount, match[1])
			if err != nil {
				return unknownNotation()
			}
			if _, exists := seenRecipients[account.Value]; !exists {
				recipients = append(recipients, account)
				seenRecipients[account.Value] = struct{}{}
			}
			index += len(match[0])
			continue
		}
		if strings.HasPrefix(rest, "[rp") {
			match := replyTag.FindStringSubmatch(rest)
			if match == nil {
				return unknownNotation()
			}
			account, err := chatwork.NewReference(chatwork.ReferenceAccount, match[1])
			if err != nil {
				return unknownNotation()
			}
			message, err := chatwork.NewReference(chatwork.ReferenceMessage, match[3])
			if err != nil {
				return unknownNotation()
			}
			if err := chatwork.ValidateReference(chatwork.ReferenceRoom, match[2]); err != nil {
				return unknownNotation()
			}
			if _, exists := seenRecipients[account.Value]; !exists {
				recipients = append(recipients, account)
				seenRecipients[account.Value] = struct{}{}
			}
			replies = append(replies, chatwork.Relation{Kind: "reply", Target: message, Resolved: false, ExternalID: match[2]})
			index += len(match[0])
			continue
		}
		if strings.HasPrefix(rest, "[To") || strings.HasPrefix(rest, "[code") ||
			strings.HasPrefix(rest, "[qt") || strings.HasPrefix(rest, "[/code]") ||
			strings.HasPrefix(rest, "[/qt]") {
			return unknownNotation()
		}
		index++
	}
	return recipients, replies, quotes, chatwork.MessageRelationsComplete
}

func unknownNotation() ([]chatwork.Reference, []chatwork.Relation, []chatwork.Relation, chatwork.MessageRelationState) {
	return nil, nil, nil, chatwork.MessageRelationsUnknown
}
