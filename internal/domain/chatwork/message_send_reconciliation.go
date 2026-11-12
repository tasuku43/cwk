package chatwork

import "fmt"

// MessageSendReconciliationStatus describes only the exact matches observed in
// one bounded latest-window read. Neither value claims whether an invocation
// of messages.send was applied.
type MessageSendReconciliationStatus string

const (
	MessageSendCandidatesFound    MessageSendReconciliationStatus = "candidates_found"
	MessageSendNoCandidatesWindow MessageSendReconciliationStatus = "no_candidates_in_window"
)

// MessageSendCandidate is a body-free projection of one visible message whose
// complete provider body exactly matched the expanded send payload.
type MessageSendCandidate struct {
	Message    Reference
	Sender     Reference
	SenderName string
	SendTime   int64
}

// MessageSendReconciliation contains every exact match in one bounded recent
// source. Outcome is deliberately fixed to unknown: body equality cannot prove
// that a particular command invocation created the candidate.
type MessageSendReconciliation struct {
	Room           Reference
	Account        Reference
	Outcome        string
	Status         MessageSendReconciliationStatus
	SourceCount    int
	CandidateCount int
	Candidates     []MessageSendCandidate
}

func validateMessageSendReconciliation(value MessageSendReconciliation, room Reference, coverage Coverage, access MessageAccessLimitation) error {
	if value.Room != room {
		return fmt.Errorf("messages.reconcile-send metadata room does not match the source room")
	}
	if err := validateResultReference("send reconciliation account", value.Account, ReferenceAccount, false); err != nil {
		return err
	}
	if value.Outcome != "unknown" {
		return fmt.Errorf("messages.reconcile-send cannot declare a known send outcome")
	}
	if value.SourceCount < 0 || value.SourceCount > coverage.Limit {
		return fmt.Errorf("messages.reconcile-send source count is outside the source limit")
	}
	if value.Candidates == nil || value.CandidateCount != len(value.Candidates) || value.CandidateCount > value.SourceCount {
		return fmt.Errorf("messages.reconcile-send candidate count does not match its candidates")
	}
	if (value.CandidateCount == 0) != (value.Status == MessageSendNoCandidatesWindow) {
		return fmt.Errorf("messages.reconcile-send status does not match its candidate count")
	}
	if value.Status != MessageSendCandidatesFound && value.Status != MessageSendNoCandidatesWindow {
		return fmt.Errorf("messages.reconcile-send status is invalid")
	}
	if access == MessageAccessAll && value.SourceCount != 0 {
		return fmt.Errorf("fully restricted reconciliation source cannot contain visible messages")
	}
	seen := make(map[Reference]struct{}, len(value.Candidates))
	for index, candidate := range value.Candidates {
		if err := validateResultReference(fmt.Sprintf("send reconciliation candidate[%d] message", index), candidate.Message, ReferenceMessage, false); err != nil {
			return err
		}
		if err := validateResultReference(fmt.Sprintf("send reconciliation candidate[%d] sender", index), candidate.Sender, ReferenceAccount, false); err != nil {
			return err
		}
		if candidate.Sender != value.Account {
			return fmt.Errorf("messages.reconcile-send candidate sender does not match authenticated account")
		}
		if err := validateText(fmt.Sprintf("send reconciliation candidate[%d] sender name", index), candidate.SenderName, 65535); err != nil {
			return err
		}
		if candidate.SendTime < 0 {
			return fmt.Errorf("messages.reconcile-send candidate send time must not be negative")
		}
		if _, exists := seen[candidate.Message]; exists {
			return fmt.Errorf("messages.reconcile-send contains a duplicate candidate reference")
		}
		seen[candidate.Message] = struct{}{}
	}
	return nil
}
