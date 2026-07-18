package chatworkcmd

import (
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

// ResolveMessageRelations returns a detached copy of one bounded message
// window. It promotes only explicit reply facts whose parent room and target
// are both proven by that window; presentation-shaped text is never inspected.
func ResolveMessageRelations(messages []chatwork.Message) ([]chatwork.Message, error) {
	resolved := cloneMessages(messages)
	messageIndex := make(map[string]int, len(resolved))

	for index := range resolved {
		message := &resolved[index]
		if !validReference(message.Ref, chatwork.ReferenceMessage) ||
			!validReference(message.Room, chatwork.ReferenceRoom) {
			return nil, relationshipContractFault(
				"invalid_chatwork_message_window",
				"Chatwork message window contains an invalid canonical reference",
			)
		}
		if _, exists := messageIndex[message.Ref.Value]; exists {
			return nil, relationshipContractFault(
				"duplicate_chatwork_message_reference",
				"Chatwork message window contains a duplicate message reference",
			)
		}
		messageIndex[message.Ref.Value] = index
	}

	for index := range resolved {
		message := &resolved[index]
		if message.Reply == nil {
			continue
		}

		reply := message.Reply
		targetIndex, targetExists := messageIndex[reply.Target.Value]
		targetInRoom := targetExists && resolved[targetIndex].Room == message.Room
		canResolve := reply.Kind == "reply" &&
			reply.ExternalID == message.Room.Value &&
			validReference(reply.Target, chatwork.ReferenceMessage) &&
			targetInRoom

		if reply.Resolved && !canResolve {
			return nil, relationshipContractFault(
				"inconsistent_chatwork_message_relation",
				"Chatwork message window contains an inconsistent resolved relation",
			)
		}
		if canResolve {
			reply.Resolved = true
		}
	}

	return resolved, nil
}

func cloneMessages(messages []chatwork.Message) []chatwork.Message {
	if messages == nil {
		return nil
	}
	cloned := make([]chatwork.Message, len(messages))
	copy(cloned, messages)
	for index := range messages {
		if messages[index].Recipients != nil {
			cloned[index].Recipients = append([]chatwork.Reference{}, messages[index].Recipients...)
		}
		if messages[index].Quotes != nil {
			cloned[index].Quotes = append([]chatwork.Relation{}, messages[index].Quotes...)
		}
		if messages[index].Reply != nil {
			reply := *messages[index].Reply
			cloned[index].Reply = &reply
		}
	}
	return cloned
}

func validReference(reference chatwork.Reference, kind chatwork.ReferenceKind) bool {
	return reference.Kind == kind && chatwork.ValidateReference(kind, reference.Value) == nil
}

func relationshipContractFault(code, message string) error {
	return fault.New(fault.KindContract, code, message, false)
}
