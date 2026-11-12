package chatworkcmd

import (
	"context"

	"github.com/tasuku43/cwk/internal/app/portcheck"
	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

// MessageSendReconciliationPort owns one validated body snapshot and compares
// it with a bounded batch of provider message bodies without exposing that
// snapshot to application code. Implementations bind the one-use handle to
// their client, authentication binding, and exact request.
type MessageSendReconciliationPort interface {
	PrepareMessageSendReconciliation(context.Context, authn.BindingID, chatwork.Request) (chatwork.PreparedMessageBody, error)
	FinalizeMessageSendReconciliation(context.Context, authn.BindingID, chatwork.Request, chatwork.PreparedMessageBody) error
	MatchMessageSendReconciliation(context.Context, authn.BindingID, chatwork.Request, chatwork.PreparedMessageBody, []string) ([]bool, error)
	DiscardMessageSendReconciliation(chatwork.PreparedMessageBody)
}

func (s *Service) executeMessageSendReconciliation(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	bodyPort, ok := s.port.(MessageSendReconciliationPort)
	if !ok || portcheck.IsNil(bodyPort) {
		return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_request_contract_invalid", "送信本文を安全に照合できる Chatwork アダプターが必要です", false,
			fault.NextAction{Command: "help messages reconcile-send", Reason: "読み取り専用の送信結果照合契約を確認してください。"})
	}

	prepared, err := bodyPort.PrepareMessageSendReconciliation(ctx, binding, request)
	if err != nil {
		return chatwork.Result{}, messageSendReconciliationAdapterError(err)
	}
	if portcheck.IsNil(prepared) {
		return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_request_contract_invalid", "送信本文照合アダプターが空の本文ハンドルを返しました", false,
			fault.NextAction{Command: "help messages reconcile-send", Reason: "本文照合アダプターの契約を確認してください。"})
	}
	defer bodyPort.DiscardMessageSendReconciliation(prepared)

	resolvedRequest := request
	if request.MessageReplyTarget != nil {
		replyRequest := chatwork.Request{
			Task:    chatwork.TaskMessagesShow,
			Room:    request.Room,
			Message: request.MessageReplyTarget.Message,
		}
		preflight, preflightErr := s.executeProvider(ctx, binding, replyRequest)
		if preflightErr != nil {
			return chatwork.Result{}, preflightErr
		}
		if len(preflight.Messages) != 1 {
			return chatwork.Result{}, invalidMessageSendReconciliationReply()
		}
		target := preflight.Messages[0]
		if target.Ref != request.MessageReplyTarget.Message || target.Room != request.Room ||
			target.Sender.Ref.Kind != chatwork.ReferenceAccount ||
			chatwork.ValidateReference(chatwork.ReferenceAccount, target.Sender.Ref.Value) != nil {
			return chatwork.Result{}, invalidMessageSendReconciliationReply()
		}
		resolvedRequest.MessageReplyTarget = &chatwork.MessageReplyTarget{
			Message: request.MessageReplyTarget.Message,
			Sender:  target.Sender.Ref,
		}
	}

	if err := bodyPort.FinalizeMessageSendReconciliation(ctx, binding, resolvedRequest, prepared); err != nil {
		return chatwork.Result{}, messageSendReconciliationAdapterError(err)
	}
	if err := ctx.Err(); err != nil {
		return chatwork.Result{}, fault.Wrap(fault.KindCanceled, "operation_canceled", "送信結果の読み取り専用照合がキャンセルされました", true, err,
			fault.NextAction{Command: "messages reconcile-send", Reason: "呼び出し元の準備ができたら読み取り専用照合を再実行してください。"})
	}

	identity, err := s.executeProvider(ctx, binding, chatwork.Request{Task: chatwork.TaskAccountShow})
	if err != nil {
		return chatwork.Result{}, err
	}
	self := identity.Account.Ref

	// This is the only window read: latest, bounded to the documented 100
	// messages, and tied to the same binding and context as any reply lookup.
	sourceRequest := chatwork.Request{Task: chatwork.TaskMessagesList, Room: request.Room, ForceRecent: true}
	source, err := s.executeProvider(ctx, binding, sourceRequest)
	if err != nil {
		return chatwork.Result{}, err
	}
	bodies := make([]string, len(source.Messages))
	for index := range source.Messages {
		bodies[index] = source.Messages[index].Body
	}
	matches, err := bodyPort.MatchMessageSendReconciliation(ctx, binding, resolvedRequest, prepared, bodies)
	for index := range source.Messages {
		// The reconciliation result is body-free. Drop references to the raw
		// provider bodies as soon as the adapter has compared the batch.
		source.Messages[index].Body = ""
		source.Messages[index].BodySegments = nil
		bodies[index] = ""
	}
	if err != nil {
		return chatwork.Result{}, messageSendReconciliationAdapterError(err)
	}
	if len(matches) != len(source.Messages) {
		return chatwork.Result{}, fault.New(fault.KindContract, "chatwork_result_invalid", "送信結果照合アダプターの候補数が取得範囲と一致しません", false,
			fault.NextAction{Command: "help messages reconcile-send", Reason: "本文照合アダプターの件数契約を確認してください。"})
	}

	candidates := make([]chatwork.MessageSendCandidate, 0)
	for index, matched := range matches {
		if !matched || source.Messages[index].Sender.Ref != self {
			continue
		}
		message := source.Messages[index]
		candidates = append(candidates, chatwork.MessageSendCandidate{
			Message: message.Ref, Sender: message.Sender.Ref, SenderName: message.Sender.Name, SendTime: message.SendTime,
		})
	}
	status := chatwork.MessageSendNoCandidatesWindow
	if len(candidates) > 0 {
		status = chatwork.MessageSendCandidatesFound
	}
	result := chatwork.Result{
		Task:          chatwork.TaskMessagesReconcileSend,
		Coverage:      source.Coverage,
		MessageRoom:   source.MessageRoom,
		MessageAccess: source.MessageAccess,
		MessageSendReconciliation: &chatwork.MessageSendReconciliation{
			Room:           request.Room,
			Account:        self,
			Outcome:        "unknown",
			Status:         status,
			SourceCount:    len(source.Messages),
			CandidateCount: len(candidates),
			Candidates:     candidates,
		},
	}
	if err := result.ValidateFor(request); err != nil {
		return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork 送信結果照合が型付き結果契約を満たしません", false, err,
			fault.NextAction{Command: "help messages reconcile-send", Reason: "読み取り専用照合の結果契約を確認してください。"})
	}
	return result, nil
}

func invalidMessageSendReconciliationReply() error {
	return fault.New(fault.KindContract, "chatwork_reconciliation_reply_invalid", "指定した返信先の取得結果がルーム、メッセージ、または投稿者と一致しません。送信結果を照合できませんでした", false,
		fault.NextAction{Command: "help messages reconcile-send", Reason: "返信先と同じルーム参照を使う照合契約を確認してください。"})
}

func messageSendReconciliationAdapterError(err error) error {
	if structured, ok := fault.PublicCopy(err); ok {
		return structured
	}
	return fault.New(fault.KindContract, "chatwork_reconciliation_body_failed", "送信本文を安全に照合できませんでした", false,
		fault.NextAction{Command: "help messages reconcile-send", Reason: "読み取り専用の送信本文照合契約を確認してください。"})
}
