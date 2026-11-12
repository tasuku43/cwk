// Package chatworkcmd owns application use cases for task-oriented Chatwork
// operations. It depends on a semantic task port, not on HTTP or wire DTOs.
package chatworkcmd

import (
	"context"
	"errors"
	"strings"

	"github.com/tasuku43/cwk/internal/app/portcheck"
	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

// Port executes one validated semantic task using the exact infrastructure
// authentication binding admitted by the application authentication gate.
type Port interface {
	Execute(context.Context, authn.BindingID, chatwork.Request) (chatwork.Result, error)
}

// MessageBodyPreflightPort provides a controlled, task-specific path for
// reading a body file before an optional reply preflight. The prepared value
// is opaque to the application and is consumed once by the same adapter.
type MessageBodyPreflightPort interface {
	PrepareMessageBody(context.Context, authn.BindingID, chatwork.Request) (chatwork.PreparedMessageBody, error)
	ExecutePreparedMessageSend(context.Context, authn.BindingID, chatwork.Request, chatwork.PreparedMessageBody) (chatwork.Result, error)
	DiscardPreparedMessageBody(chatwork.PreparedMessageBody)
}

type Service struct {
	port Port
}

func New(port Port) *Service {
	return &Service{port: port}
}

// Execute validates the task and binding before crossing the external port.
// Read cancellation prevents a result from being returned. A valid mutation
// acknowledgement is preserved even if cancellation arrives after the port
// has confirmed it.
func (s *Service) Execute(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	if ctx == nil {
		return chatwork.Result{}, fault.New(fault.KindContract, "missing_context", "Chatwork タスクコンテキストが設定されていません", false)
	}
	if err := ctx.Err(); err != nil && !mutationTask(request.Task) {
		return chatwork.Result{}, readContextFault(err, nil, "実行前に Chatwork タスクがキャンセルされました")
	}
	if err := binding.Validate(); err != nil {
		return chatwork.Result{}, fault.New(fault.KindAuthentication, "invalid_authentication_binding", "Chatwork 認証バインドは無効です", false)
	}
	if err := request.Validate(); err != nil {
		return chatwork.Result{}, fault.Wrap(fault.KindInvalidInput, "invalid_chatwork_task", "Chatwork タスク入力は無効です", false, err)
	}
	if s == nil || portcheck.IsNil(s.port) {
		return chatwork.Result{}, fault.New(fault.KindContract, "missing_chatwork_port", "Chatwork タスクアダプターが設定されていません", false)
	}
	if request.Task == chatwork.TaskMessagesResources {
		return s.executeMessageResources(ctx, binding, request)
	}
	if request.Task == chatwork.TaskMessagesSend {
		return s.executeMessageSend(ctx, binding, request)
	}
	if request.Task == chatwork.TaskMessagesReconcileSend {
		return s.executeMessageSendReconciliation(ctx, binding, request)
	}
	providerRequest := request
	if request.Task == chatwork.TaskMembersFind {
		// Member-name discovery is an application-owned projection of the one
		// complete room-member read. Display text never crosses the provider
		// boundary as an identity or undocumented query parameter.
		providerRequest.Task = chatwork.TaskMembersList
		providerRequest.MemberQuery = ""
	}
	// Room-name discovery is a local projection over the one complete joined-
	// room collection. The query is display text, never a provider parameter.
	providerRequest.RoomQuery = ""
	providerRequest.MessageFilter = chatwork.MessageFilter{}
	providerRequest.MessageRelationFetchLimit = 0
	providerRequest.MessageAround = nil
	result, err := s.executeProvider(ctx, binding, providerRequest)
	if err != nil {
		return chatwork.Result{}, err
	}
	switch request.Task {
	case chatwork.TaskRoomsList:
		if request.RoomQuery != "" {
			sourceCount := len(result.Rooms)
			candidates := make([]chatwork.Room, 0, sourceCount)
			for _, room := range result.Rooms {
				if chatwork.ContainsFold(room.Name, request.RoomQuery) {
					candidates = append(candidates, room)
				}
			}
			result.Rooms = candidates
			result.RoomSelection = &chatwork.RoomSelection{
				Query:          request.RoomQuery,
				SourceCount:    sourceCount,
				CandidateCount: len(candidates),
			}
		}
	case chatwork.TaskMembersFind:
		sourceCount := len(result.Accounts)
		candidates := make([]chatwork.Account, 0, sourceCount)
		for _, account := range result.Accounts {
			if strings.Contains(account.Name, request.MemberQuery) {
				candidates = append(candidates, account)
			}
		}
		result.Task = chatwork.TaskMembersFind
		result.Accounts = candidates
		result.MemberSelection = &chatwork.MemberSelection{
			Query:       request.MemberQuery,
			SourceCount: sourceCount,
		}
	case chatwork.TaskMessagesList:
		source, sourceErr := ResolveMessageRelations(result.Messages)
		if sourceErr != nil {
			return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, sourceErr)
		}
		source, sourceErr = ResolveMessageQuoteCandidates(source, source, true, result.MessageAccess)
		if sourceErr != nil {
			return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, sourceErr)
		}
		reachability, reachabilityErr := chatwork.DeriveMessageReachability(result.Coverage, result.MessageAccess, source, request.MessageFilter.Period)
		if reachabilityErr != nil {
			return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, reachabilityErr)
		}
		messages, selection, selectionErr := assembleMessageWindow(source, request.MessageFilter)
		if selectionErr != nil {
			return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, selectionErr)
		}
		RemoveHiddenMessageQuoteCandidateSequences(messages)
		result.Messages = messages
		result.MessageSelection = selection
		result.MessageReachability = &reachability
		if request.MessageRelationFetchLimit > 0 {
			resolved, resolution, resolutionErr := s.resolveMessageRelations(ctx, binding, request.Room, source, result.Messages, request.MessageRelationFetchLimit)
			if resolutionErr != nil {
				return chatwork.Result{}, resolutionErr
			}
			result.Messages = resolved
			result.MessageRelationResolution = resolution
		}
	case chatwork.TaskMessagesShow:
		messages, resolutionErr := ResolveMessageRelations(result.Messages)
		if resolutionErr != nil {
			return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, resolutionErr)
		}
		if request.MessageFilter.Context == chatwork.MessageContextReplies || request.MessageAround != nil {
			recentRequest := chatwork.Request{
				Task:        chatwork.TaskMessagesList,
				Room:        request.Room,
				ForceRecent: true,
			}
			recent, recentErr := s.executeProvider(ctx, binding, recentRequest)
			if recentErr != nil {
				return chatwork.Result{}, recentErr
			}
			recent.Messages, resolutionErr = ResolveMessageQuoteCandidates(recent.Messages, recent.Messages, true, recent.MessageAccess)
			if resolutionErr != nil {
				return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, resolutionErr)
			}
			messages, resolutionErr = ResolveMessageQuoteCandidates(messages, recent.Messages, true, recent.MessageAccess)
			if resolutionErr != nil {
				return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, resolutionErr)
			}
			anchor := messages[0]
			var source []chatwork.Message
			if request.MessageFilter.Context == chatwork.MessageContextReplies {
				var showContext *chatwork.MessageShowContext
				var contextErr error
				anchor, source, showContext, contextErr = assembleMessageShowContext(anchor, recent)
				if contextErr != nil {
					return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, contextErr)
				}
				result.MessageShowContext = showContext
			}
			if request.MessageAround != nil {
				around, aroundErr := assembleMessageShowAround(messages[0], recent, request.MessageAround.Count)
				if aroundErr != nil {
					return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, aroundErr)
				}
				result.MessageShowAround = around
				anchor, aroundErr = resolveMessageShowVisible(anchor, result.MessageShowContext, around)
				if aroundErr != nil {
					return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, aroundErr)
				}
				result.MessageShowAround = around
			}
			result.Messages = []chatwork.Message{anchor}
			visibleSource := make([][]chatwork.Message, 0, 4)
			if result.MessageShowAround != nil && result.MessageShowAround.AnchorState == chatwork.MessageAroundAnchorInWindow {
				visibleSource = append(visibleSource, result.Messages)
			}
			if result.MessageShowContext != nil {
				visibleSource = append(visibleSource, result.MessageShowContext.Children)
			}
			if result.MessageShowAround != nil {
				visibleSource = append(visibleSource, result.MessageShowAround.Before, result.MessageShowAround.After)
			}
			RemoveHiddenMessageQuoteCandidateSequences(visibleSource...)
			if request.MessageFilter.Context != chatwork.MessageContextReplies {
				break
			}
			if request.MessageRelationFetchLimit == 0 {
				break
			}
			resolved, resolution, relationErr := s.resolveMessageRelations(
				ctx,
				binding,
				request.Room,
				source,
				result.Messages,
				request.MessageRelationFetchLimit,
			)
			if relationErr != nil {
				return chatwork.Result{}, relationErr
			}
			result.Messages = resolved
			result.MessageRelationResolution = resolution
		} else {
			messages, resolutionErr = ResolveMessageQuoteCandidates(messages, nil, false, chatwork.MessageAccessNone)
			if resolutionErr != nil {
				return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, resolutionErr)
			}
			result.Messages = messages
		}
	}
	removeHiddenQuoteCandidateSequencesFromResult(&result)
	if err := result.ValidateFor(request); err != nil {
		return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, err)
	}
	return result, nil
}

func (s *Service) executeMessageSend(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	providerRequest := request
	var preparedBody chatwork.PreparedMessageBody
	var bodyPort MessageBodyPreflightPort
	if request.MessageReplyTarget != nil && request.BodyFile != "" {
		candidate, ok := s.port.(MessageBodyPreflightPort)
		if !ok {
			return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_request_contract_invalid", "返信先を読む前に本文ファイルを検証できる Chatwork アダプターが必要です", false)
		}
		bodyPort = candidate
		prepared, prepareErr := bodyPort.PrepareMessageBody(ctx, binding, request)
		if prepareErr != nil {
			return chatwork.Result{}, classifyMessageBodyPreparationFailure(prepareErr)
		}
		if portcheck.IsNil(prepared) {
			return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_request_contract_invalid", "本文準備アダプターが空の本文ハンドルを返しました", false)
		}
		preparedBody = prepared
		defer bodyPort.DiscardPreparedMessageBody(preparedBody)
	}
	if request.MessageReplyTarget != nil {
		replyRequest := chatwork.Request{
			Task:    chatwork.TaskMessagesShow,
			Room:    request.Room,
			Message: request.MessageReplyTarget.Message,
		}
		preflight, err := s.executeProvider(ctx, binding, replyRequest)
		if err != nil {
			return chatwork.Result{}, classifyReplyPreflightFailure(err)
		}
		if len(preflight.Messages) != 1 {
			return chatwork.Result{}, invalidReplyPreflightResult()
		}
		message := preflight.Messages[0]
		if message.Ref != request.MessageReplyTarget.Message || message.Room != request.Room ||
			message.Sender.Ref.Kind != chatwork.ReferenceAccount ||
			chatwork.ValidateReference(chatwork.ReferenceAccount, message.Sender.Ref.Value) != nil {
			return chatwork.Result{}, invalidReplyPreflightResult()
		}
		replyTarget := *request.MessageReplyTarget
		replyTarget.Sender = message.Sender.Ref
		providerRequest.MessageReplyTarget = &replyTarget
	}
	var result chatwork.Result
	var err error
	if preparedBody != nil {
		result, err = s.executePreparedProvider(ctx, binding, providerRequest, bodyPort, preparedBody)
	} else {
		result, err = s.executeProvider(ctx, binding, providerRequest)
	}
	if err != nil {
		return chatwork.Result{}, err
	}
	return result, nil
}

func classifyMessageBodyPreparationFailure(err error) error {
	structured, ok := fault.PublicCopy(err)
	if !ok || structured.Code == "unclassified_chatwork_error" {
		return fault.New(fault.KindContract, "chatwork_send_body_preparation_failed", "送信前に本文ファイルを検証できませんでした。メッセージは送信していません", false,
			fault.NextAction{Command: "help messages send", Reason: "本文ファイルの読み取り契約を確認してください。"})
	}
	if structured.Code == "operation_canceled" || structured.Code == "chatwork_mutation_not_dispatched" {
		return fault.New(fault.KindCanceled, "chatwork_send_body_preflight_not_completed", "本文ファイルの送信前確認がキャンセルされました。メッセージは送信していません", true,
			fault.NextAction{Command: "messages send", Reason: "本文ファイルの確認後に同じ入力を再実行できます。"})
	}
	return structured
}

func classifyReplyPreflightFailure(err error) error {
	structured, ok := fault.PublicCopy(err)
	if !ok {
		return retryableReplyPreflightFailure()
	}
	switch structured.Code {
	case "operation_canceled":
		return fault.New(fault.KindCanceled, "chatwork_send_preflight_not_completed", "返信先の確認が送信前にキャンセルされました。メッセージは送信していません", true,
			fault.NextAction{Command: "messages send", Reason: "返信先の確認と送信をもう一度実行できます。"})
	case "unclassified_chatwork_error":
		return retryableReplyPreflightFailure()
	case "chatwork_result_invalid", "chatwork_result_mismatch":
		return invalidReplyPreflightResult()
	default:
		return structured
	}
}

func retryableReplyPreflightFailure() error {
	return fault.New(fault.KindUnavailable, "chatwork_reply_preflight_failed", "返信先を送信前に確認できませんでした。メッセージは送信していません", true,
		fault.NextAction{Command: "messages send", Reason: "返信先の確認が成功した後に一度だけ送信できます。"})
}

func invalidReplyPreflightResult() error {
	return fault.New(fault.KindContract, "chatwork_reply_preflight_invalid", "返信先の取得結果が指定したルーム、メッセージ、または投稿者と一致しません。メッセージは送信していません", false,
		fault.NextAction{Command: "help messages send", Reason: "返信先と同じルーム参照を使う送信契約を確認してください。"})
}

func (s *Service) executeProvider(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	return s.executeProviderCall(ctx, binding, request, func(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
		return s.port.Execute(ctx, binding, request)
	})
}

func (s *Service) executePreparedProvider(ctx context.Context, binding authn.BindingID, request chatwork.Request, bodyPort MessageBodyPreflightPort, prepared chatwork.PreparedMessageBody) (chatwork.Result, error) {
	return s.executeProviderCall(ctx, binding, request, func(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
		return bodyPort.ExecutePreparedMessageSend(ctx, binding, request, prepared)
	})
}

func (s *Service) executeProviderCall(ctx context.Context, binding authn.BindingID, request chatwork.Request, execute func(context.Context, authn.BindingID, chatwork.Request) (chatwork.Result, error)) (chatwork.Result, error) {
	result, err := execute(ctx, binding, request)
	if err != nil {
		if structured, ok := fault.PublicCopy(err); ok {
			return chatwork.Result{}, structured
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return chatwork.Result{}, readContextFault(ctxErr, err, "実行中に Chatwork タスクがキャンセルされました")
		}
		if !mutationTask(request.Task) && errors.Is(err, context.DeadlineExceeded) {
			return chatwork.Result{}, readContextFault(nil, err, "実行中に Chatwork タスクがキャンセルされました")
		}
		if !mutationTask(request.Task) && errors.Is(err, context.Canceled) {
			return chatwork.Result{}, readContextFault(nil, err, "実行中に Chatwork タスクがキャンセルされました")
		}
		return chatwork.Result{}, fault.New(fault.KindInternal, "unclassified_chatwork_error", "Chatwork タスクが分類不能なエラーを返しました", false)
	}
	if err := ctx.Err(); err != nil && !mutationTask(request.Task) {
		return chatwork.Result{}, readContextFault(err, nil, "実行後に Chatwork タスクがキャンセルされました")
	}
	if result.Task != request.Task {
		return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_result_mismatch", "Chatwork タスクアダプターが別のタスクの結果を返しました", false)
	}
	if err := result.ValidateFor(request); err != nil {
		return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, err)
	}
	return result, nil
}

func readContextFault(contextErr, operationErr error, canceledMessage string) error {
	if errors.Is(contextErr, context.Canceled) {
		return fault.Wrap(fault.KindCanceled, "operation_canceled", canceledMessage, true, contextErr)
	}
	if errors.Is(contextErr, context.DeadlineExceeded) || errors.Is(operationErr, context.DeadlineExceeded) {
		return fault.New(fault.KindCanceled, "operation_canceled", "Chatwork の読み取りが時間内に完了しませんでした", true)
	}
	if errors.Is(operationErr, context.Canceled) {
		return fault.Wrap(fault.KindCanceled, "operation_canceled", canceledMessage, true, operationErr)
	}
	return fault.New(fault.KindUnavailable, "chatwork_unavailable", "Chatwork は一時的に利用できません", true)
}

// mutationTask lets the service preserve a provider-confirmed mutation result
// if cancellation arrives after the port acknowledges it. For a canceled
// mutation before dispatch, the infrastructure adapter owns the decision
// because it alone can prove that HTTP Do was not reached.
func mutationTask(task chatwork.Task) bool {
	switch task {
	case chatwork.TaskRoomsCreate, chatwork.TaskRoomsUpdate, chatwork.TaskRoomsLeave,
		chatwork.TaskRoomsDelete, chatwork.TaskMembersReplace, chatwork.TaskMessagesSend,
		chatwork.TaskMessagesMarkRead, chatwork.TaskMessagesMarkUnread, chatwork.TaskMessagesUpdate,
		chatwork.TaskMessagesDelete, chatwork.TaskRoomTasksCreate, chatwork.TaskRoomTasksSetStatus,
		chatwork.TaskFilesUpload, chatwork.TaskInviteLinkCreate, chatwork.TaskInviteLinkUpdate,
		chatwork.TaskInviteLinkDelete, chatwork.TaskContactRequestsAccept, chatwork.TaskContactRequestsReject:
		return true
	default:
		return false
	}
}
