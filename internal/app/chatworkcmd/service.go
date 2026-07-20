// Package chatworkcmd owns application use cases for task-oriented Chatwork
// operations. It depends on a semantic task port, not on HTTP or wire DTOs.
package chatworkcmd

import (
	"context"
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

type Service struct {
	port Port
}

func New(port Port) *Service {
	return &Service{port: port}
}

// Execute validates the task and binding before crossing the external port.
// It returns no result when cancellation or a contract failure is observed.
func (s *Service) Execute(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	if ctx == nil {
		return chatwork.Result{}, fault.New(fault.KindContract, "missing_context", "Chatwork タスクコンテキストが設定されていません", false)
	}
	if err := ctx.Err(); err != nil {
		return chatwork.Result{}, fault.Wrap(fault.KindCanceled, "operation_canceled", "実行前に Chatwork タスクがキャンセルされました", true, err)
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
	providerRequest := request
	if request.Task == chatwork.TaskMembersFind {
		// Member-name discovery is an application-owned projection of the one
		// complete room-member read. Display text never crosses the provider
		// boundary as an identity or undocumented query parameter.
		providerRequest.Task = chatwork.TaskMembersList
		providerRequest.MemberQuery = ""
	}
	providerRequest.MessageFilter = chatwork.MessageFilter{}
	providerRequest.MessageRelationFetchLimit = 0
	result, err := s.executeProvider(ctx, binding, providerRequest)
	if err != nil {
		return chatwork.Result{}, err
	}
	switch request.Task {
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
		reachability, reachabilityErr := chatwork.DeriveMessageReachability(result.Coverage, result.MessageAccess, source, request.MessageFilter.Period)
		if reachabilityErr != nil {
			return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, reachabilityErr)
		}
		messages, selection, selectionErr := assembleMessageWindow(result.Messages, request.MessageFilter)
		if selectionErr != nil {
			return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, selectionErr)
		}
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
		result.Messages = messages
	}
	if err := result.ValidateFor(request); err != nil {
		return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, err)
	}
	return result, nil
}

func (s *Service) executeProvider(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	result, err := s.port.Execute(ctx, binding, request)
	if err != nil {
		if structured, ok := fault.PublicCopy(err); ok {
			return chatwork.Result{}, structured
		}
		if ctx.Err() != nil {
			return chatwork.Result{}, fault.Wrap(fault.KindCanceled, "operation_canceled", "実行中に Chatwork タスクがキャンセルされました", true, ctx.Err())
		}
		return chatwork.Result{}, fault.New(fault.KindInternal, "unclassified_chatwork_error", "Chatwork タスクが分類不能なエラーを返しました", false)
	}
	if err := ctx.Err(); err != nil {
		return chatwork.Result{}, fault.Wrap(fault.KindCanceled, "operation_canceled", "実行後に Chatwork タスクがキャンセルされました", true, err)
	}
	if result.Task != request.Task {
		return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_result_mismatch", "Chatwork タスクアダプターが別のタスクの結果を返しました", false)
	}
	if err := result.ValidateFor(request); err != nil {
		return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, err)
	}
	return result, nil
}
