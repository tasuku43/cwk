package chatwork

import "fmt"

// MessageQuoteCandidateState describes how a provider quote's author and
// optional timestamp compare with one bounded source window. No state proves
// that a candidate is the quoted message.
type MessageQuoteCandidateState string

const (
	MessageQuoteCandidateUniqueWindow MessageQuoteCandidateState = "unique-window-candidate"
	MessageQuoteCandidateAmbiguous    MessageQuoteCandidateState = "ambiguous-window-candidates"
	MessageQuoteCandidateNoMatch      MessageQuoteCandidateState = "no-window-candidate"
	MessageQuoteCandidateMissingTime  MessageQuoteCandidateState = "missing-timestamp"
	MessageQuoteCandidateUnknown      MessageQuoteCandidateState = "unknown"
)

type MessageQuoteCandidateScope string

const MessageQuoteCandidateSourceWindow MessageQuoteCandidateScope = "source-window"

// MessageQuoteCandidate records one exact author-and-time match in the
// bounded source window. Sequence is zero when the candidate's source record
// is not included in the visible result. IdentityProven is intentionally
// always false: quote metadata contains no message identity.
type MessageQuoteCandidate struct {
	MessageRef     Reference
	Room           Reference
	Author         Reference
	SendTime       int64
	Sequence       int
	Scope          MessageQuoteCandidateScope
	IdentityProven bool
}

// MessageQuoteMatch records the source-window search result for one quote in
// provider order. Candidates are evidence only and the quote relation remains
// unresolved for every state.
type MessageQuoteMatch struct {
	State      MessageQuoteCandidateState
	Candidates []MessageQuoteCandidate
}

func (r Result) validateMessageQuoteCandidateBindings() error {
	if r.Task != TaskMessagesList && r.Task != TaskMessagesShow {
		return nil
	}
	// Bind source provenance only when there are candidate references to check.
	// Ordinary message validation and presentation checks own duplicate visible
	// message handling; quote validation should not change their error ordering.
	hasCandidates := false
	for _, message := range r.Messages {
		for _, match := range message.QuoteMatches {
			if len(match.Candidates) != 0 {
				hasCandidates = true
				break
			}
		}
	}
	if r.MessageRelationResolution != nil {
		for _, target := range r.MessageRelationResolution.Targets {
			if target.Message == nil {
				continue
			}
			for _, match := range target.Message.QuoteMatches {
				if len(match.Candidates) != 0 {
					hasCandidates = true
					break
				}
			}
		}
	}
	if r.Task == TaskMessagesShow {
		if r.MessageShowContext != nil {
			for _, message := range r.MessageShowContext.Children {
				for _, match := range message.QuoteMatches {
					if len(match.Candidates) != 0 {
						hasCandidates = true
						break
					}
				}
			}
		}
		if r.MessageShowAround != nil {
			for _, group := range [][]Message{r.MessageShowAround.Before, r.MessageShowAround.After} {
				for _, message := range group {
					for _, match := range message.QuoteMatches {
						if len(match.Candidates) != 0 {
							hasCandidates = true
							break
						}
					}
				}
			}
		}
	}
	if !hasCandidates {
		return nil
	}
	type sourceRecord struct {
		message  Message
		sequence int
	}
	sourceByRef := make(map[Reference]sourceRecord)
	owners := make([]Message, 0, len(r.Messages))
	addSource := func(message Message, sequence int) error {
		if prior, exists := sourceByRef[message.Ref]; exists {
			if prior.message.Room != message.Room || prior.message.Sender.Ref != message.Sender.Ref || prior.message.SendTime != message.SendTime {
				return fmt.Errorf("visible quote source duplicates a message with inconsistent identity or time")
			}
			if prior.sequence > 0 && sequence > 0 && prior.sequence != sequence {
				return fmt.Errorf("visible quote source assigns multiple provider sequences to one message")
			}
			if prior.sequence == 0 && sequence > 0 {
				prior.sequence = sequence
				sourceByRef[message.Ref] = prior
			}
			return nil
		}
		sourceByRef[message.Ref] = sourceRecord{message: message, sequence: sequence}
		return nil
	}
	for index, message := range r.Messages {
		sequence := 0
		if r.Task == TaskMessagesList {
			sequence = index + 1
			if r.MessageSelection != nil && index < len(r.MessageSelection.SourceSequences) {
				sequence = r.MessageSelection.SourceSequences[index]
			}
		}
		if r.Task == TaskMessagesShow && r.MessageShowAround != nil && r.MessageShowAround.AnchorState == MessageAroundAnchorInWindow {
			sequence = r.MessageShowAround.AnchorSourceSequence
		}
		if err := addSource(message, sequence); err != nil {
			return err
		}
		owners = append(owners, message)
	}
	if r.Task == TaskMessagesShow && r.MessageShowContext != nil {
		for index, message := range r.MessageShowContext.Children {
			if index >= len(r.MessageShowContext.SourceSequences) {
				return fmt.Errorf("visible quote source has no reply-context sequence")
			}
			if err := addSource(message, r.MessageShowContext.SourceSequences[index]); err != nil {
				return err
			}
			owners = append(owners, message)
		}
	}
	if r.Task == TaskMessagesShow && r.MessageShowAround != nil {
		for index, message := range r.MessageShowAround.Before {
			if index >= len(r.MessageShowAround.BeforeSequences) {
				return fmt.Errorf("visible quote source has no before-context sequence")
			}
			if err := addSource(message, r.MessageShowAround.BeforeSequences[index]); err != nil {
				return err
			}
			owners = append(owners, message)
		}
		for index, message := range r.MessageShowAround.After {
			if index >= len(r.MessageShowAround.AfterSequences) {
				return fmt.Errorf("visible quote source has no after-context sequence")
			}
			if err := addSource(message, r.MessageShowAround.AfterSequences[index]); err != nil {
				return err
			}
			owners = append(owners, message)
		}
	}
	if r.MessageRelationResolution != nil {
		for _, target := range r.MessageRelationResolution.Targets {
			if target.Message != nil {
				owners = append(owners, *target.Message)
			}
		}
	}
	for _, owner := range owners {
		for matchIndex, match := range owner.QuoteMatches {
			if matchIndex >= len(owner.Quotes) {
				return fmt.Errorf("visible quote match does not have a quote relation")
			}
			for _, candidate := range match.Candidates {
				source, visible := sourceByRef[candidate.MessageRef]
				if !visible {
					if candidate.Sequence != 0 {
						return fmt.Errorf("hidden quote candidate retains a dangling source sequence")
					}
					continue
				}
				if candidate.Sequence != source.sequence || candidate.Room != source.message.Room ||
					candidate.Author != source.message.Sender.Ref || candidate.SendTime != source.message.SendTime {
					return fmt.Errorf("visible quote candidate evidence does not match its source message")
				}
			}
		}
	}
	return nil
}
