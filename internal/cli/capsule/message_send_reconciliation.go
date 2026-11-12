package capsule

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func validateMessageSendReconciliationProjection(result chatwork.Result) error {
	if result.Task != chatwork.TaskMessagesReconcileSend || result.MessageSendReconciliation == nil {
		return nil
	}
	reconciliation := result.MessageSendReconciliation
	if err := chatwork.ValidateReference(chatwork.ReferenceRoom, result.MessageRoom.Value); err != nil {
		return fmt.Errorf("task projection reconciliation room reference: %w", err)
	}
	if err := chatwork.ValidateReference(chatwork.ReferenceAccount, reconciliation.Account.Value); err != nil {
		return fmt.Errorf("task projection reconciliation account reference: %w", err)
	}
	for index, candidate := range reconciliation.Candidates {
		if err := chatwork.ValidateReference(chatwork.ReferenceMessage, candidate.Message.Value); err != nil {
			return fmt.Errorf("task projection reconciliation candidate[%d] message reference: %w", index, err)
		}
		if err := chatwork.ValidateReference(chatwork.ReferenceAccount, candidate.Sender.Value); err != nil {
			return fmt.Errorf("task projection reconciliation candidate[%d] sender reference: %w", index, err)
		}
		if !utf8.ValidString(candidate.SenderName) {
			return fmt.Errorf("task projection reconciliation candidate[%d] sender name is not valid UTF-8", index)
		}
	}
	return nil
}

func renderMessageSendReconciliation(output *strings.Builder, result chatwork.Result) error {
	reconciliation := result.MessageSendReconciliation
	access, err := messageAccess(result.MessageAccess)
	if err != nil {
		return fmt.Errorf("task projection reconciliation access limitation: %w", err)
	}
	line(output, "send-reconciliation room_ref=%s account_ref=%s candidate_status=%s outcome=%s candidate_count=%d source_count=%d source_limit=%d complete=%t access_limitation=%s",
		ref(result.MessageRoom), ref(reconciliation.Account), string(reconciliation.Status), reconciliation.Outcome,
		reconciliation.CandidateCount, reconciliation.SourceCount, result.Coverage.Limit,
		result.Coverage.Complete, access)
	line(output, "uncertainty=候補は送信呼び出しの証明ではありません。候補なしも未送信の証明ではありません。過去の同文、送信時と異なるアカウント、伝播遅延、上限外、閲覧制限、編集・削除、変更された本文ファイル、再指定された標準入力、変化した返信先投稿者により結果は一致しない場合があります。")
	line(output, "external-text=untrusted escaped")
	for _, candidate := range reconciliation.Candidates {
		line(output, "candidate candidate_message_ref=%s sender_ref=%s sender_name=%s send_time=%d",
			ref(candidate.Message), ref(candidate.Sender), quoted(candidate.SenderName), candidate.SendTime)
	}
	return nil
}
