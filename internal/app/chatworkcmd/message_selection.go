package chatworkcmd

import (
	"fmt"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

// assembleMessageWindow resolves one complete bounded source window before
// applying an optional sender selection. Reply context is exactly one hop from
// the original sender matches; it is never expanded transitively.
func assembleMessageWindow(messages []chatwork.Message, filter chatwork.MessageFilter) ([]chatwork.Message, *chatwork.MessageSelection, error) {
	resolvedSource, err := ResolveMessageRelations(messages)
	if err != nil {
		return nil, nil, err
	}
	if len(filter.Senders) == 0 {
		return resolvedSource, nil, nil
	}

	senders := make(map[string]struct{}, len(filter.Senders))
	for _, sender := range filter.Senders {
		senders[sender.Value] = struct{}{}
	}
	anchors := make([]bool, len(resolvedSource))
	included := make([]bool, len(resolvedSource))
	messageIndex := make(map[string]int, len(resolvedSource))
	for index, message := range resolvedSource {
		messageIndex[message.Ref.Value] = index
		_, matches := senders[message.Sender.Ref.Value]
		anchors[index] = matches
		included[index] = matches
	}

	switch filter.Context {
	case chatwork.MessageContextNone:
		// Sender matches are the complete selected result.
	case chatwork.MessageContextReplies:
		for index, message := range resolvedSource {
			if message.Reply == nil || !message.Reply.Resolved {
				continue
			}
			parentIndex, found := messageIndex[message.Reply.Target.Value]
			if !found {
				return nil, nil, fmt.Errorf("resolved Chatwork reply target is absent from its source window")
			}
			if anchors[index] {
				included[parentIndex] = true
			}
			if anchors[parentIndex] {
				included[index] = true
			}
		}
	default:
		return nil, nil, fmt.Errorf("Chatwork message context is invalid")
	}

	selected := make([]chatwork.Message, 0, len(resolvedSource))
	sourceSequences := make([]int, 0, len(resolvedSource))
	anchorSequences := make([]int, 0, len(filter.Senders))
	for index, message := range resolvedSource {
		if !included[index] {
			continue
		}
		selected = append(selected, message)
		sourceSequences = append(sourceSequences, index+1)
		if anchors[index] {
			anchorSequences = append(anchorSequences, index+1)
		}
	}

	// Resolution is relative to the displayed result. Reset the source-window
	// proof before resolving the selected subset so an omitted parent is shown
	// as unresolved instead of retaining a false in-result edge.
	selected = cloneMessages(selected)
	for index := range selected {
		if selected[index].Reply != nil {
			selected[index].Reply.Resolved = false
		}
	}
	selected, err = ResolveMessageRelations(selected)
	if err != nil {
		return nil, nil, err
	}

	filterCopy := filter
	filterCopy.Senders = append([]chatwork.Reference(nil), filter.Senders...)
	selection := &chatwork.MessageSelection{
		Filter:          filterCopy,
		SourceCount:     len(resolvedSource),
		SourceSequences: sourceSequences,
		AnchorSequences: anchorSequences,
	}
	return selected, selection, nil
}
