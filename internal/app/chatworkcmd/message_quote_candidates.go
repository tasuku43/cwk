package chatworkcmd

import (
	"fmt"
	"strconv"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

// ResolveMessageQuoteCandidates annotates each explicit quote against one
// original bounded source window. A candidate requires exact same-room sender
// identity and whole timestamp equality. Quote metadata never proves message
// identity, so every quote relation remains unresolved.
func ResolveMessageQuoteCandidates(messages, source []chatwork.Message, sourceAvailable bool, sourceAccess chatwork.MessageAccessLimitation) ([]chatwork.Message, error) {
	if !sourceAvailable && len(source) != 0 {
		return nil, fmt.Errorf("unavailable quote source contains messages")
	}
	if sourceAccess != chatwork.MessageAccessNone && sourceAccess != chatwork.MessageAccessPartial && sourceAccess != chatwork.MessageAccessAll {
		return nil, fmt.Errorf("quote source access limitation is invalid")
	}
	resolved := cloneMessages(messages)
	sourceSequence := make(map[chatwork.Reference]int, len(source))
	for index, candidate := range source {
		if !validReference(candidate.Ref, chatwork.ReferenceMessage) || !validReference(candidate.Room, chatwork.ReferenceRoom) ||
			!validReference(candidate.Sender.Ref, chatwork.ReferenceAccount) {
			return nil, fmt.Errorf("quote source contains an invalid message identity")
		}
		if _, duplicate := sourceSequence[candidate.Ref]; duplicate {
			return nil, fmt.Errorf("quote source contains a duplicate message reference")
		}
		sourceSequence[candidate.Ref] = index + 1
	}
	for index := range resolved {
		message := &resolved[index]
		if !validReference(message.Ref, chatwork.ReferenceMessage) || !validReference(message.Room, chatwork.ReferenceRoom) {
			return nil, fmt.Errorf("quote-bearing message has an invalid identity")
		}
		if message.RelationState == chatwork.MessageRelationsUnknown {
			if len(message.Quotes) != 0 {
				return nil, fmt.Errorf("unknown relation set carries quote metadata")
			}
			message.QuoteMatches = nil
			continue
		}
		if message.QuoteMatches != nil {
			return nil, fmt.Errorf("quote candidates were already computed for this message")
		}
		if len(message.Quotes) == 0 {
			message.QuoteMatches = nil
			continue
		}
		message.QuoteMatches = make([]chatwork.MessageQuoteMatch, len(message.Quotes))
		for quoteIndex, quote := range message.Quotes {
			match := chatwork.MessageQuoteMatch{State: chatwork.MessageQuoteCandidateUnknown, Candidates: []chatwork.MessageQuoteCandidate{}}
			if quote.Kind != "quote" || quote.Resolved {
				return nil, fmt.Errorf("quote metadata is invalid")
			}
			if !validReference(quote.Target, chatwork.ReferenceAccount) {
				message.QuoteMatches[quoteIndex] = match
				continue
			}
			if quote.ExternalID == "" {
				match.State = chatwork.MessageQuoteCandidateMissingTime
				message.QuoteMatches[quoteIndex] = match
				continue
			}
			if !sourceAvailable || sourceAccess == chatwork.MessageAccessAll {
				message.QuoteMatches[quoteIndex] = match
				continue
			}
			for _, candidate := range source {
				if candidate.Room != message.Room || candidate.Sender.Ref != quote.Target || strconv.FormatInt(candidate.SendTime, 10) != quote.ExternalID {
					continue
				}
				match.Candidates = append(match.Candidates, chatwork.MessageQuoteCandidate{
					MessageRef: candidate.Ref, Room: candidate.Room, Author: candidate.Sender.Ref, SendTime: candidate.SendTime,
					Sequence: sourceSequence[candidate.Ref],
					Scope:    chatwork.MessageQuoteCandidateSourceWindow, IdentityProven: false,
				})
			}
			switch len(match.Candidates) {
			case 0:
				if sourceAccess == chatwork.MessageAccessPartial {
					match.State = chatwork.MessageQuoteCandidateUnknown
				} else {
					match.State = chatwork.MessageQuoteCandidateNoMatch
				}
			case 1:
				match.State = chatwork.MessageQuoteCandidateUniqueWindow
			default:
				match.State = chatwork.MessageQuoteCandidateAmbiguous
			}
			message.QuoteMatches[quoteIndex] = match
		}
	}
	return resolved, nil
}

// RemoveHiddenMessageQuoteCandidateSequences keeps a candidate's canonical
// reference when it is outside the visible message slice, while removing the
// now-dangling optional source sequence. It does not recompute matches or
// alter their source-window ambiguity.
func RemoveHiddenMessageQuoteCandidateSequences(groups ...[]chatwork.Message) {
	visible := make(map[chatwork.Reference]struct{})
	for _, messages := range groups {
		for _, message := range messages {
			visible[message.Ref] = struct{}{}
		}
	}
	for _, messages := range groups {
		removeQuoteCandidateSequences(messages, visible)
	}
}

// RemoveHiddenMessageQuoteCandidateSequencesFromTargets clears source
// sequences in supplemental message records unless the candidate is also a
// visible source-sequenced message. Relation-context records do not expose a
// source sequence of their own.
func RemoveHiddenMessageQuoteCandidateSequencesFromTargets(targets []chatwork.MessageRelationTarget, visibleGroups ...[]chatwork.Message) {
	visible := make(map[chatwork.Reference]struct{})
	for _, messages := range visibleGroups {
		for _, message := range messages {
			visible[message.Ref] = struct{}{}
		}
	}
	for index := range targets {
		if targets[index].Message == nil {
			continue
		}
		removeQuoteCandidateSequences([]chatwork.Message{*targets[index].Message}, visible)
	}
}

func removeHiddenQuoteCandidateSequencesFromResult(result *chatwork.Result) {
	if result == nil {
		return
	}
	var owners [][]chatwork.Message
	var sourceSequenced [][]chatwork.Message
	switch result.Task {
	case chatwork.TaskMessagesList:
		owners = append(owners, result.Messages)
		sourceSequenced = append(sourceSequenced, result.Messages)
	case chatwork.TaskMessagesShow:
		// The exact anchor always owns quote metadata, but it has a source
		// sequence only when around proved its position in the recent window.
		owners = append(owners, result.Messages)
		if result.MessageShowAround != nil && result.MessageShowAround.AnchorState == chatwork.MessageAroundAnchorInWindow {
			sourceSequenced = append(sourceSequenced, result.Messages)
		}
		if result.MessageShowContext != nil {
			owners = append(owners, result.MessageShowContext.Children)
			sourceSequenced = append(sourceSequenced, result.MessageShowContext.Children)
		}
		if result.MessageShowAround != nil {
			owners = append(owners, result.MessageShowAround.Before, result.MessageShowAround.After)
			sourceSequenced = append(sourceSequenced, result.MessageShowAround.Before, result.MessageShowAround.After)
		}
	}
	visible := make(map[chatwork.Reference]struct{})
	for _, messages := range sourceSequenced {
		for _, message := range messages {
			visible[message.Ref] = struct{}{}
		}
	}
	for _, messages := range owners {
		removeQuoteCandidateSequences(messages, visible)
	}
	if result.MessageRelationResolution != nil {
		for index := range result.MessageRelationResolution.Targets {
			target := &result.MessageRelationResolution.Targets[index]
			if target.Message != nil {
				removeQuoteCandidateSequences([]chatwork.Message{*target.Message}, visible)
			}
		}
	}
}

func removeQuoteCandidateSequences(messages []chatwork.Message, visible map[chatwork.Reference]struct{}) {
	for messageIndex := range messages {
		matches := messages[messageIndex].QuoteMatches
		for matchIndex := range matches {
			candidates := matches[matchIndex].Candidates
			for candidateIndex := range candidates {
				candidate := &candidates[candidateIndex]
				if _, found := visible[candidate.MessageRef]; !found {
					candidate.Sequence = 0
				}
			}
		}
	}
}
