package chatworkcmd

import (
	"context"

	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

// resolveMessageRelations completes explicit same-room reply chains in
// breadth-first first-reference order. The original source is reused before
// one exact-message request consumes a caller-declared fetch slot. The finite
// budget and a visited set bound recursive context discovery and cycles.
func (s *Service) resolveMessageRelations(
	ctx context.Context,
	binding authn.BindingID,
	room chatwork.Reference,
	source []chatwork.Message,
	displayed []chatwork.Message,
	limit int,
) ([]chatwork.Message, *chatwork.MessageRelationResolution, error) {
	resolved := cloneMessages(displayed)
	resolution := &chatwork.MessageRelationResolution{
		FetchLimit: limit,
		Targets:    []chatwork.MessageRelationTarget{},
	}
	sourceByRef := make(map[chatwork.Reference]chatwork.Message, len(source))
	displayedRefs := make(map[chatwork.Reference]struct{}, len(resolved))
	for _, message := range source {
		sourceByRef[message.Ref] = message
	}
	for _, message := range resolved {
		displayedRefs[message.Ref] = struct{}{}
	}

	targets := make([]chatwork.Reference, 0)
	seen := make(map[chatwork.Reference]struct{}, len(displayedRefs))
	available := make(map[chatwork.Reference]struct{}, len(displayedRefs))
	for ref := range displayedRefs {
		seen[ref] = struct{}{}
		available[ref] = struct{}{}
	}
	enqueueReply := func(message *chatwork.Message) {
		if message.Reply == nil || message.Reply.Kind != "reply" || message.Reply.ExternalID != room.Value {
			return
		}
		target := message.Reply.Target
		if _, present := available[target]; present {
			message.Reply.Resolved = true
			return
		}
		if _, duplicate := seen[target]; duplicate {
			return
		}
		seen[target] = struct{}{}
		targets = append(targets, target)
	}
	for index := range resolved {
		enqueueReply(&resolved[index])
	}

	for targetIndex := 0; targetIndex < len(targets); targetIndex++ {
		target := targets[targetIndex]
		outcome := chatwork.MessageRelationTarget{Target: target}
		if message, present := sourceByRef[target]; present {
			contextMessage := cloneMessages([]chatwork.Message{message})[0]
			outcome.State = chatwork.MessageRelationResolvedFromSource
			outcome.Message = &contextMessage
			resolution.Targets = append(resolution.Targets, outcome)
			available[target] = struct{}{}
			markResolvedReplyTarget(resolved, resolution.Targets, room, target)
			enqueueReply(resolution.Targets[len(resolution.Targets)-1].Message)
			continue
		}
		if resolution.FetchAttempts >= limit {
			outcome.State = chatwork.MessageRelationBudgetExhausted
			resolution.Targets = append(resolution.Targets, outcome)
			continue
		}

		resolution.FetchAttempts++
		exactRequest := chatwork.Request{Task: chatwork.TaskMessagesShow, Room: room, Message: target}
		exact, err := s.executeProvider(ctx, binding, exactRequest)
		if err != nil {
			state, retained := retainedRelationFailure(err)
			if !retained {
				return nil, nil, err
			}
			outcome.State = state
			resolution.Targets = append(resolution.Targets, outcome)
			continue
		}
		contextMessages, err := ResolveMessageRelations(exact.Messages)
		if err != nil {
			return nil, nil, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, err)
		}
		contextMessage := contextMessages[0]
		outcome.State = chatwork.MessageRelationResolvedByFetch
		outcome.Message = &contextMessage
		resolution.Targets = append(resolution.Targets, outcome)
		available[target] = struct{}{}
		markResolvedReplyTarget(resolved, resolution.Targets, room, target)
		enqueueReply(resolution.Targets[len(resolution.Targets)-1].Message)
	}

	return resolved, resolution, nil
}

func markResolvedReplyTarget(messages []chatwork.Message, targets []chatwork.MessageRelationTarget, room, target chatwork.Reference) {
	for index := range messages {
		reply := messages[index].Reply
		if reply != nil && reply.Kind == "reply" && reply.ExternalID == room.Value && reply.Target == target {
			reply.Resolved = true
		}
	}
	for index := range targets {
		if targets[index].Message == nil {
			continue
		}
		reply := targets[index].Message.Reply
		if reply != nil && reply.Kind == "reply" && reply.ExternalID == room.Value && reply.Target == target {
			reply.Resolved = true
		}
	}
}

func retainedRelationFailure(err error) (chatwork.MessageRelationResolutionState, bool) {
	public, ok := fault.PublicCopy(err)
	if !ok {
		return "", false
	}
	switch public.Code {
	case "chatwork_not_found":
		return chatwork.MessageRelationNotFound, true
	case "chatwork_message_restricted":
		return chatwork.MessageRelationRestricted, true
	default:
		return "", false
	}
}
