package chatworkapi

import (
	"regexp"
	"strings"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

var (
	toTag     = regexp.MustCompile(`^\[To:([1-9][0-9]{0,31})\]`)
	replyTag  = regexp.MustCompile(`^\[rp aid=([1-9][0-9]{0,31}) to=([1-9][0-9]{0,31})-([1-9][0-9]{0,31})\]`)
	quoteMeta = regexp.MustCompile(`^\[qt\]\[qtmeta aid=([1-9][0-9]{0,31})(?: time=([1-9][0-9]{0,18}))?\]`)
)

// parseNotation recognizes only the reviewed provider forms needed by the
// semantic boundary. It does not infer relations from prose, names, layout, or
// time proximity, and it does not interpret copied tags inside quote/code data.
func parseNotation(body string) ([]chatwork.Reference, *chatwork.Relation, []chatwork.Relation, error) {
	recipients := make([]chatwork.Reference, 0)
	seenRecipients := map[string]struct{}{}
	quotes := make([]chatwork.Relation, 0)
	var reply *chatwork.Relation

	for index := 0; index < len(body); {
		rest := body[index:]
		if strings.HasPrefix(rest, "[code]") {
			end := strings.Index(rest[len("[code]"):], "[/code]")
			if end < 0 {
				return nil, nil, nil, notationFault()
			}
			index += len("[code]") + end + len("[/code]")
			continue
		}
		if strings.HasPrefix(rest, "[qt]") {
			match := quoteMeta.FindStringSubmatch(rest)
			if match == nil {
				return nil, nil, nil, notationFault()
			}
			end := strings.Index(rest[len(match[0]):], "[/qt]")
			if end < 0 {
				return nil, nil, nil, notationFault()
			}
			account, err := chatwork.NewReference(chatwork.ReferenceAccount, match[1])
			if err != nil {
				return nil, nil, nil, notationFault()
			}
			quotes = append(quotes, chatwork.Relation{Kind: "quote", Target: account, Resolved: false, ExternalID: match[2]})
			index += len(match[0]) + end + len("[/qt]")
			continue
		}
		if strings.HasPrefix(rest, "[To:") {
			match := toTag.FindStringSubmatch(rest)
			if match == nil {
				return nil, nil, nil, notationFault()
			}
			account, err := chatwork.NewReference(chatwork.ReferenceAccount, match[1])
			if err != nil {
				return nil, nil, nil, notationFault()
			}
			if _, exists := seenRecipients[account.Value]; !exists {
				recipients = append(recipients, account)
				seenRecipients[account.Value] = struct{}{}
			}
			index += len(match[0])
			continue
		}
		if strings.HasPrefix(rest, "[rp ") {
			match := replyTag.FindStringSubmatch(rest)
			if match == nil || reply != nil {
				return nil, nil, nil, notationFault()
			}
			account, err := chatwork.NewReference(chatwork.ReferenceAccount, match[1])
			if err != nil {
				return nil, nil, nil, notationFault()
			}
			message, err := chatwork.NewReference(chatwork.ReferenceMessage, match[3])
			if err != nil {
				return nil, nil, nil, notationFault()
			}
			if err := chatwork.ValidateReference(chatwork.ReferenceRoom, match[2]); err != nil {
				return nil, nil, nil, notationFault()
			}
			if _, exists := seenRecipients[account.Value]; !exists {
				recipients = append(recipients, account)
				seenRecipients[account.Value] = struct{}{}
			}
			reply = &chatwork.Relation{Kind: "reply", Target: message, Resolved: false, ExternalID: match[2]}
			index += len(match[0])
			continue
		}
		index++
	}
	return recipients, reply, quotes, nil
}

func notationFault() error {
	return fault.New(fault.KindContract, "chatwork_notation_malformed", "Chatwork message notation is malformed or unsupported", false)
}
