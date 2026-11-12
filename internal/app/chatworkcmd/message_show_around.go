package chatworkcmd

import (
	"fmt"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

// assembleMessageShowAround selects positional neighbors only when the exact
// anchor appears in this one bounded latest-window source. Absence is useful
// only when the provider did not report an access limitation; no other field
// such as time, ID magnitude, or adjacency in rendered output is consulted.
func assembleMessageShowAround(anchor chatwork.Message, recent chatwork.Result, count int) (*chatwork.MessageShowAround, error) {
	if recent.Task != chatwork.TaskMessagesList || recent.MessageRoom != anchor.Room {
		return nil, fmt.Errorf("exact-message around source does not match the anchor room")
	}
	if count < 0 || count > chatwork.MaxMessageAroundCount {
		return nil, fmt.Errorf("exact-message around count is outside its declared bound")
	}

	around := &chatwork.MessageShowAround{
		Anchor:          anchor.Ref,
		AnchorState:     chatwork.MessageAroundAnchorOutside,
		Requested:       count,
		Before:          []chatwork.Message{},
		BeforeSequences: []int{},
		After:           []chatwork.Message{},
		AfterSequences:  []int{},
		SourceCount:     len(recent.Messages),
		Coverage:        recent.Coverage,
		Access:          recent.MessageAccess,
	}
	anchorIndex := -1
	for index, message := range recent.Messages {
		if message.Ref != anchor.Ref {
			continue
		}
		if anchorIndex >= 0 {
			return nil, fmt.Errorf("exact-message around source contains a duplicate anchor reference")
		}
		anchorIndex = index
	}
	if anchorIndex < 0 {
		if recent.MessageAccess != chatwork.MessageAccessNone {
			around.AnchorState = chatwork.MessageAroundAnchorUnknown
		}
		return around, nil
	}

	around.AnchorState = chatwork.MessageAroundAnchorInWindow
	around.AnchorSourceSequence = anchorIndex + 1
	first := anchorIndex - count
	if first < 0 {
		first = 0
	}
	for index := first; index < anchorIndex; index++ {
		around.Before = append(around.Before, cloneMessages([]chatwork.Message{recent.Messages[index]})[0])
		around.BeforeSequences = append(around.BeforeSequences, index+1)
	}
	last := anchorIndex + 1 + count
	if last > len(recent.Messages) {
		last = len(recent.Messages)
	}
	for index := anchorIndex + 1; index < last; index++ {
		around.After = append(around.After, cloneMessages([]chatwork.Message{recent.Messages[index]})[0])
		around.AfterSequences = append(around.AfterSequences, index+1)
	}
	around.ActualBefore = len(around.Before)
	around.ActualAfter = len(around.After)
	return around, nil
}

// resolveMessageShowVisible binds explicit reply edges only within the final
// unique visible set. The exact read owns the anchor if the same reference is
// also present in the recent source; direct children and positional neighbors
// share one canonical-reference set so a renderer can emit each message once.
func resolveMessageShowVisible(anchor chatwork.Message, replyContext *chatwork.MessageShowContext, around *chatwork.MessageShowAround) (chatwork.Message, error) {
	visible := []chatwork.Message{cloneMessages([]chatwork.Message{anchor})[0]}
	seen := map[chatwork.Reference]struct{}{anchor.Ref: {}}
	appendUnique := func(messages []chatwork.Message) {
		for _, message := range messages {
			if _, exists := seen[message.Ref]; exists {
				continue
			}
			seen[message.Ref] = struct{}{}
			visible = append(visible, cloneMessages([]chatwork.Message{message})[0])
		}
	}
	if replyContext != nil {
		appendUnique(replyContext.Children)
	}
	appendUnique(around.Before)
	appendUnique(around.After)

	resolved, err := ResolveMessageRelations(visible)
	if err != nil {
		return chatwork.Message{}, err
	}
	byRef := make(map[chatwork.Reference]chatwork.Message, len(resolved))
	for _, message := range resolved {
		byRef[message.Ref] = message
	}
	resolvedAnchor := byRef[anchor.Ref]
	if replyContext != nil {
		for index := range replyContext.Children {
			replyContext.Children[index] = byRef[replyContext.Children[index].Ref]
		}
	}
	for index := range around.Before {
		around.Before[index] = byRef[around.Before[index].Ref]
	}
	for index := range around.After {
		around.After[index] = byRef[around.After[index].Ref]
	}
	return resolvedAnchor, nil
}
