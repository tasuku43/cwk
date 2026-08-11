package chatworkcmd

import (
	"fmt"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

// assembleMessageShowContext joins one exact anchor with one bounded latest
// window. It selects children only from explicit typed same-room reply edges;
// display text and other relation kinds never participate.
func assembleMessageShowContext(anchor chatwork.Message, recent chatwork.Result) (chatwork.Message, []chatwork.Message, *chatwork.MessageShowContext, error) {
	if recent.Task != chatwork.TaskMessagesList || recent.MessageRoom != anchor.Room {
		return chatwork.Message{}, nil, nil, fmt.Errorf("exact-message context source does not match the anchor room")
	}

	// Prefer the exact read for the anchor itself. A recent window may contain
	// an older representation of the same message, so remove that copy before
	// resolving typed edges against the exact anchor.
	combined := make([]chatwork.Message, 0, len(recent.Messages)+1)
	sequences := make([]int, 0, len(recent.Messages))
	for index, message := range recent.Messages {
		if message.Ref == anchor.Ref {
			continue
		}
		combined = append(combined, message)
		sequences = append(sequences, index+1)
	}
	combined = append(combined, anchor)
	resolved, err := ResolveMessageRelations(combined)
	if err != nil {
		return chatwork.Message{}, nil, nil, err
	}
	resolvedSource := resolved

	children := make([]chatwork.Message, 0)
	childSequences := make([]int, 0)
	for index := range resolved[:len(resolved)-1] {
		message := resolved[index]
		direct := false
		for _, reply := range message.Replies {
			if reply.Kind == "reply" && reply.ExternalID == anchor.Room.Value && reply.Target == anchor.Ref && reply.Resolved {
				direct = true
				break
			}
		}
		if direct {
			// Keep the original relation state until the final visible set is
			// known. Source-window resolution must not make a hidden target look
			// available in the exact-message projection.
			children = append(children, combined[index])
			childSequences = append(childSequences, sequences[index])
		}
	}
	visible := append(cloneMessages(children), cloneMessages([]chatwork.Message{anchor})[0])
	for index := range visible {
		for replyIndex := range visible[index].Replies {
			visible[index].Replies[replyIndex].Resolved = false
		}
	}
	visible, err = ResolveMessageRelations(visible)
	if err != nil {
		return chatwork.Message{}, nil, nil, err
	}
	children = visible[:len(visible)-1]
	resolvedAnchor := visible[len(visible)-1]
	context := &chatwork.MessageShowContext{
		Anchor:          anchor.Ref,
		Children:        children,
		SourceSequences: childSequences,
		SourceCount:     len(recent.Messages),
		Coverage:        recent.Coverage,
		Access:          recent.MessageAccess,
	}
	return resolvedAnchor, resolvedSource, context, nil
}
