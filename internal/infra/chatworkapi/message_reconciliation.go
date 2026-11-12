package chatworkapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"sync"

	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

type preparedMessageSendReconciliation struct {
	mu                sync.Mutex
	owner             *Client
	binding           authn.BindingID
	sourceFingerprint [32]byte
	finalFingerprint  [32]byte
	body              []byte
	finalized         bool
	used              bool
}

func (*preparedMessageSendReconciliation) IsPreparedMessageBody() {}

func (*preparedMessageSendReconciliation) String() string {
	return "[prepared Chatwork send reconciliation]"
}

func (*preparedMessageSendReconciliation) GoString() string {
	return "[prepared Chatwork send reconciliation]"
}

func (c *Client) messageBodyBytes(ctx context.Context, input chatwork.Request) ([]byte, error) {
	if input.BodyFile == "" {
		body := []byte(input.Body)
		if err := chatwork.ValidateMessageBodyBytes(body); err != nil {
			clear(body)
			return nil, fault.New(fault.KindInvalidInput, "chatwork_message_body_invalid", "メッセージ本文は空でない65535バイト以下の有効なUTF-8が必要です", false)
		}
		return body, nil
	}
	if c == nil || c.readBodyFile == nil {
		return nil, fault.New(fault.KindInvalidInput, "chatwork_message_body_file_unreadable", "メッセージ本文ファイルを読み取れませんでした", false)
	}
	data, err := c.readBodyFile(ctx, input.BodyFile)
	if err != nil {
		if ctx != nil && ctx.Err() != nil {
			return nil, fault.Wrap(fault.KindCanceled, "operation_canceled", "メッセージ本文ファイルの読み取りをキャンセルしました", true, ctx.Err())
		}
		return nil, fault.New(fault.KindInvalidInput, "chatwork_message_body_file_unreadable", "メッセージ本文ファイルを読み取れませんでした", false)
	}
	if len(data) > chatwork.MaxMessageBodyBytes {
		clear(data)
		return nil, fault.New(fault.KindInvalidInput, "chatwork_message_body_too_large", "メッセージ本文が 65535 バイトの上限を超えています", false)
	}
	if err := chatwork.ValidateMessageBodyBytes(data); err != nil {
		clear(data)
		return nil, fault.New(fault.KindInvalidInput, "chatwork_message_body_invalid", "メッセージ本文ファイルには空でない65535バイト以下の有効なUTF-8が必要です", false)
	}
	return data, nil
}

// PrepareMessageSendReconciliation snapshots the exact validated source body
// before any provider read. A body-file is dereferenced only by the existing
// bounded infrastructure reader, and its path and bytes stay inside this
// client-owned one-use handle.
func (c *Client) PrepareMessageSendReconciliation(ctx context.Context, binding authn.BindingID, input chatwork.Request) (chatwork.PreparedMessageBody, error) {
	if ctx == nil {
		return nil, fault.New(fault.KindContract, "missing_context", "Chatwork タスクコンテキストが設定されていません", false)
	}
	if err := ctx.Err(); err != nil {
		return nil, fault.Wrap(fault.KindCanceled, "operation_canceled", "送信本文の照合準備がキャンセルされました", true, err)
	}
	if input.Task != chatwork.TaskMessagesReconcileSend {
		return nil, invalidRequest("send reconciliation body preparation requires messages.reconcile-send")
	}
	if err := input.Validate(); err != nil {
		return nil, fault.Wrap(fault.KindInvalidInput, "invalid_chatwork_request", "Chatwork タスク入力は無効です", false, err)
	}
	if _, err := c.resolve(binding); err != nil {
		return nil, err
	}
	body, err := c.messageBodyBytes(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		clear(body)
		return nil, fault.Wrap(fault.KindCanceled, "operation_canceled", "送信本文の照合準備がキャンセルされました", true, err)
	}
	return &preparedMessageSendReconciliation{
		owner:             c,
		binding:           binding,
		sourceFingerprint: messageSendReconciliationFingerprint(input, false),
		body:              body,
	}, nil
}

// FinalizeMessageSendReconciliation applies the same expanded-body builder as
// messages.send after the optional reply read has established the exact reply
// author. The caller still receives only an opaque matching handle.
func (c *Client) FinalizeMessageSendReconciliation(ctx context.Context, binding authn.BindingID, input chatwork.Request, prepared chatwork.PreparedMessageBody) error {
	if ctx == nil {
		return fault.New(fault.KindContract, "missing_context", "Chatwork タスクコンテキストが設定されていません", false)
	}
	if err := ctx.Err(); err != nil {
		return fault.Wrap(fault.KindCanceled, "operation_canceled", "送信本文の照合準備がキャンセルされました", true, err)
	}
	body, ok := prepared.(*preparedMessageSendReconciliation)
	if !ok || body == nil {
		return invalidRequest("prepared send reconciliation handle is invalid")
	}
	body.mu.Lock()
	defer body.mu.Unlock()
	if body.owner != c || body.binding != binding || body.used || body.finalized ||
		body.sourceFingerprint != messageSendReconciliationFingerprint(input, false) {
		return invalidRequest("prepared send reconciliation handle does not match this request")
	}
	if input.Task != chatwork.TaskMessagesReconcileSend {
		return invalidRequest("send reconciliation finalization requires messages.reconcile-send")
	}
	if err := input.Validate(); err != nil {
		return fault.Wrap(fault.KindInvalidInput, "invalid_chatwork_request", "Chatwork タスク入力は無効です", false, err)
	}
	if input.MessageReplyTarget != nil && input.MessageReplyTarget.Sender.Kind != chatwork.ReferenceAccount {
		return invalidRequest("resolved reply target sender is required for typed send reconciliation")
	}
	expanded, err := expandMessageBodyBytes(input, body.body)
	if err != nil {
		return err
	}
	clear(body.body)
	body.body = expanded
	body.finalFingerprint = messageSendReconciliationFingerprint(input, true)
	body.finalized = true
	return nil
}

// MatchMessageSendReconciliation consumes one finalized matcher over at most
// one provider source. Only equality booleans cross back to application code;
// the stored expanded payload is zeroed before this method returns.
func (c *Client) MatchMessageSendReconciliation(ctx context.Context, binding authn.BindingID, input chatwork.Request, prepared chatwork.PreparedMessageBody, candidates []string) ([]bool, error) {
	if ctx == nil {
		return nil, fault.New(fault.KindContract, "missing_context", "Chatwork タスクコンテキストが設定されていません", false)
	}
	if err := ctx.Err(); err != nil {
		return nil, fault.Wrap(fault.KindCanceled, "operation_canceled", "送信本文の照合がキャンセルされました", true, err)
	}
	if len(candidates) > chatwork.MaxMessageSelectionCount {
		return nil, invalidRequest("send reconciliation matcher exceeds the provider source bound")
	}
	body, ok := prepared.(*preparedMessageSendReconciliation)
	if !ok || body == nil {
		return nil, invalidRequest("prepared send reconciliation handle is invalid")
	}
	body.mu.Lock()
	defer body.mu.Unlock()
	if body.owner != c || body.binding != binding || body.used || !body.finalized ||
		body.finalFingerprint != messageSendReconciliationFingerprint(input, true) {
		return nil, invalidRequest("prepared send reconciliation handle does not match this request")
	}
	defer body.zero()
	matches := make([]bool, len(candidates))
	for index, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return nil, fault.Wrap(fault.KindCanceled, "operation_canceled", "送信本文の照合がキャンセルされました", true, err)
		}
		candidateBytes := []byte(candidate)
		matches[index] = bytes.Equal(body.body, candidateBytes)
		clear(candidateBytes)
	}
	body.used = true
	return matches, nil
}

// DiscardMessageSendReconciliation releases an unused body snapshot. It is
// safe to call after a successful one-use match.
func (c *Client) DiscardMessageSendReconciliation(prepared chatwork.PreparedMessageBody) {
	body, ok := prepared.(*preparedMessageSendReconciliation)
	if !ok || body == nil {
		return
	}
	body.mu.Lock()
	defer body.mu.Unlock()
	if body.owner != c {
		return
	}
	body.zero()
}

func (body *preparedMessageSendReconciliation) zero() {
	clear(body.body)
	body.body = nil
	body.sourceFingerprint = [32]byte{}
	body.finalFingerprint = [32]byte{}
	body.binding = authn.BindingID{}
	body.owner = nil
	body.finalized = false
	body.used = true
}

func messageSendReconciliationFingerprint(input chatwork.Request, includeResolvedSender bool) [32]byte {
	value := sha256.New()
	writeFingerprintString(value, string(input.Task))
	writeFingerprintString(value, string(input.Room.Kind))
	writeFingerprintString(value, input.Room.Value)
	writeFingerprintString(value, input.Body)
	writeFingerprintString(value, input.BodyFile)
	writeFingerprintString(value, fmt.Sprint(input.SelfUnread))
	writeFingerprintInt(value, uint64(len(input.MessageRecipients)))
	for _, recipient := range input.MessageRecipients {
		writeFingerprintString(value, string(recipient.Kind))
		writeFingerprintString(value, recipient.Value)
	}
	if input.MessageReplyTarget == nil {
		writeFingerprintString(value, "no-reply-target")
	} else {
		writeFingerprintString(value, "reply-target")
		writeFingerprintString(value, string(input.MessageReplyTarget.Message.Kind))
		writeFingerprintString(value, input.MessageReplyTarget.Message.Value)
		if includeResolvedSender {
			writeFingerprintString(value, string(input.MessageReplyTarget.Sender.Kind))
			writeFingerprintString(value, input.MessageReplyTarget.Sender.Value)
		}
	}
	var fingerprint [32]byte
	copy(fingerprint[:], value.Sum(nil))
	return fingerprint
}

func writeFingerprintString(value hash.Hash, text string) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(text)))
	_, _ = value.Write(length[:])
	bytes := []byte(text)
	_, _ = value.Write(bytes)
	clear(bytes)
}

func writeFingerprintInt(value hash.Hash, number uint64) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], number)
	_, _ = value.Write(encoded[:])
}
