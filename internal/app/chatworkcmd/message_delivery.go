package chatworkcmd

import (
	"fmt"
	"sort"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

// MessageDelivery is one presentation-sized prefix of an already assembled
// message result. OutputItems counts primary anchors; direct reply neighbors
// and the reachable supplemental relation closure travel with those anchors.
// The original Result, Coverage, and MessageSelection remain unchanged.
type MessageDelivery struct {
	Messages             []chatwork.Message
	SourceSequences      []int
	AnchorSequences      []int
	RelationTargets      []chatwork.MessageRelationTarget
	OutputItems          int
	OutputTotalItems     int
	OutputComplete       bool
	OutputNextStartIndex int
	OutputNextCount      int
}

// MessageDeliveryAnchorCount returns the primary-anchor count available to a
// byte-budgeted projection. An unfiltered result treats every source message
// as an anchor.
func MessageDeliveryAnchorCount(result chatwork.Result) (int, error) {
	_, anchors, err := messageDeliverySequences(result)
	return len(anchors), err
}

// PlanMessageDelivery selects the newest-first prefix of primary anchors while
// preserving provider order in the returned message slice. It performs no I/O.
func PlanMessageDelivery(result chatwork.Result, outputItems int) (MessageDelivery, error) {
	sequences, anchors, err := messageDeliverySequences(result)
	if err != nil {
		return MessageDelivery{}, err
	}
	if outputItems < 0 || outputItems > len(anchors) {
		return MessageDelivery{}, fmt.Errorf("message delivery output item count is outside its selected anchor set")
	}

	messageBySequence := make(map[int]int, len(sequences))
	for index, sequence := range sequences {
		messageBySequence[sequence] = index
	}
	rankedAnchors := append([]int(nil), anchors...)
	sort.Slice(rankedAnchors, func(left, right int) bool {
		leftMessage := result.Messages[messageBySequence[rankedAnchors[left]]]
		rightMessage := result.Messages[messageBySequence[rankedAnchors[right]]]
		if leftMessage.SendTime == rightMessage.SendTime {
			return rankedAnchors[left] > rankedAnchors[right]
		}
		return leftMessage.SendTime > rightMessage.SendTime
	})
	keptAnchorSet := make(map[int]struct{}, outputItems)
	keptAnchorRefs := make(map[chatwork.Reference]struct{}, outputItems)
	for _, sequence := range rankedAnchors[:outputItems] {
		keptAnchorSet[sequence] = struct{}{}
		keptAnchorRefs[result.Messages[messageBySequence[sequence]].Ref] = struct{}{}
	}

	keepMessage := make([]bool, len(result.Messages))
	for index, sequence := range sequences {
		_, keepMessage[index] = keptAnchorSet[sequence]
	}
	if result.MessageSelection != nil && result.MessageSelection.Filter.Context == chatwork.MessageContextReplies {
		for index, message := range result.Messages {
			if keepMessage[index] {
				continue
			}
			if messageRepliesToAny(message, keptAnchorRefs) || anyAnchorRepliesTo(result.Messages, keptAnchorSet, sequences, message.Ref) {
				keepMessage[index] = true
			}
		}
	}

	deliveredMessages := make([]chatwork.Message, 0, len(result.Messages))
	deliveredSequences := make([]int, 0, len(result.Messages))
	deliveredAnchors := make([]int, 0, outputItems)
	for index, message := range result.Messages {
		if !keepMessage[index] {
			continue
		}
		deliveredMessages = append(deliveredMessages, message)
		deliveredSequences = append(deliveredSequences, sequences[index])
		if _, anchor := keptAnchorSet[sequences[index]]; anchor {
			deliveredAnchors = append(deliveredAnchors, sequences[index])
		}
	}
	deliveredMessages = cloneMessages(deliveredMessages)
	for index := range deliveredMessages {
		for replyIndex := range deliveredMessages[index].Replies {
			deliveredMessages[index].Replies[replyIndex].Resolved = false
		}
	}
	deliveredMessages, err = ResolveMessageRelations(deliveredMessages)
	if err != nil {
		return MessageDelivery{}, fmt.Errorf("message delivery relation reset: %w", err)
	}

	deliveredTargets := planDeliveryRelationTargets(result, deliveredMessages)
	markDeliveryResolvedReplies(result.MessageRoom, deliveredMessages, deliveredTargets)

	startIndex := 1
	if result.MessageSelection != nil && result.MessageSelection.Filter.StartIndex > 0 {
		startIndex = result.MessageSelection.Filter.StartIndex
	}
	complete := outputItems == len(anchors)
	nextStartIndex := 0
	nextCount := 0
	if !complete {
		nextStartIndex = startIndex + outputItems
		if result.MessageSelection != nil && result.MessageSelection.Filter.Count > 0 {
			nextCount = len(anchors) - outputItems
		}
	}
	return MessageDelivery{
		Messages:             deliveredMessages,
		SourceSequences:      deliveredSequences,
		AnchorSequences:      deliveredAnchors,
		RelationTargets:      deliveredTargets,
		OutputItems:          outputItems,
		OutputTotalItems:     len(anchors),
		OutputComplete:       complete,
		OutputNextStartIndex: nextStartIndex,
		OutputNextCount:      nextCount,
	}, nil
}

func messageDeliverySequences(result chatwork.Result) ([]int, []int, error) {
	if result.Task != chatwork.TaskMessagesList {
		return nil, nil, fmt.Errorf("message delivery requires messages.list")
	}
	if err := result.Validate(); err != nil {
		return nil, nil, fmt.Errorf("message delivery result: %w", err)
	}
	if result.MessageSelection != nil {
		return append([]int(nil), result.MessageSelection.SourceSequences...), append([]int(nil), result.MessageSelection.AnchorSequences...), nil
	}
	sequences := make([]int, len(result.Messages))
	for index := range result.Messages {
		sequences[index] = index + 1
	}
	return sequences, append([]int(nil), sequences...), nil
}

func messageRepliesToAny(message chatwork.Message, targets map[chatwork.Reference]struct{}) bool {
	for _, reply := range message.Replies {
		if !reply.Resolved || reply.Kind != "reply" || reply.ExternalID != message.Room.Value {
			continue
		}
		if _, found := targets[reply.Target]; found {
			return true
		}
	}
	return false
}

func anyAnchorRepliesTo(messages []chatwork.Message, anchors map[int]struct{}, sequences []int, target chatwork.Reference) bool {
	for index, message := range messages {
		if _, anchor := anchors[sequences[index]]; !anchor {
			continue
		}
		for _, reply := range message.Replies {
			if reply.Resolved && reply.Kind == "reply" && reply.ExternalID == message.Room.Value && reply.Target == target {
				return true
			}
		}
	}
	return false
}

func planDeliveryRelationTargets(result chatwork.Result, messages []chatwork.Message) []chatwork.MessageRelationTarget {
	if result.MessageRelationResolution == nil {
		return nil
	}
	original := make(map[chatwork.Reference]chatwork.MessageRelationTarget, len(result.MessageRelationResolution.Targets))
	for _, target := range result.MessageRelationResolution.Targets {
		original[target.Target] = target
	}
	seen := make(map[chatwork.Reference]struct{}, len(messages)+len(original))
	for _, message := range messages {
		seen[message.Ref] = struct{}{}
	}
	queue := make([]chatwork.Reference, 0)
	enqueue := func(message chatwork.Message) {
		for _, reply := range message.Replies {
			if reply.Kind != "reply" || reply.ExternalID != result.MessageRoom.Value {
				continue
			}
			if _, duplicate := seen[reply.Target]; duplicate {
				continue
			}
			seen[reply.Target] = struct{}{}
			queue = append(queue, reply.Target)
		}
	}
	for _, message := range messages {
		enqueue(message)
	}
	delivered := make([]chatwork.MessageRelationTarget, 0, len(original))
	for index := 0; index < len(queue); index++ {
		target, found := original[queue[index]]
		if !found {
			continue
		}
		cloned := target
		if target.Message != nil {
			message := cloneMessages([]chatwork.Message{*target.Message})[0]
			cloned.Message = &message
			enqueue(message)
		}
		delivered = append(delivered, cloned)
	}
	return delivered
}

func markDeliveryResolvedReplies(room chatwork.Reference, messages []chatwork.Message, targets []chatwork.MessageRelationTarget) {
	available := make(map[chatwork.Reference]struct{}, len(messages)+len(targets))
	for _, message := range messages {
		available[message.Ref] = struct{}{}
	}
	for _, target := range targets {
		if target.Message != nil {
			available[target.Message.Ref] = struct{}{}
		}
	}
	mark := func(message *chatwork.Message) {
		for index := range message.Replies {
			reply := &message.Replies[index]
			if reply.Kind != "reply" || reply.ExternalID != room.Value {
				continue
			}
			_, reply.Resolved = available[reply.Target]
		}
	}
	for index := range messages {
		mark(&messages[index])
	}
	for index := range targets {
		if targets[index].Message != nil {
			mark(targets[index].Message)
		}
	}
}
