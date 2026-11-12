package chatwork

import (
	"errors"
	"strings"
)

const messageLinkPrefix = "https://www.chatwork.com/#!rid"
const MaxMessageLinkBytes = len(messageLinkPrefix) + 32 + 1 + 32

// MessageLinkReferences are syntactically extracted, not remotely verified.
type MessageLinkReferences struct {
	Room    Reference
	Message Reference
}

// ParseMessageLink accepts only the exact reviewed provider link form.
func ParseMessageLink(link string) (MessageLinkReferences, error) {
	invalid := errors.New("invalid message link")
	if len(link) > MaxMessageLinkBytes || !strings.HasPrefix(link, messageLinkPrefix) {
		return MessageLinkReferences{}, invalid
	}
	roomPart, messagePart, found := strings.Cut(strings.TrimPrefix(link, messageLinkPrefix), "-")
	if !found || strings.Contains(messagePart, "-") {
		return MessageLinkReferences{}, invalid
	}
	room, err := NewReference(ReferenceRoom, roomPart)
	if err != nil {
		return MessageLinkReferences{}, invalid
	}
	message, err := NewReference(ReferenceMessage, messagePart)
	if err != nil {
		return MessageLinkReferences{}, invalid
	}
	return MessageLinkReferences{Room: room, Message: message}, nil
}
