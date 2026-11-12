package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

type failedReconciliationBodyFilePort struct {
	providerCalls int
	prepareCalls  int
	privatePath   string
}

func (p *failedReconciliationBodyFilePort) Execute(context.Context, authn.BindingID, chatwork.Request) (chatwork.Result, error) {
	p.providerCalls++
	return chatwork.Result{}, fault.New(fault.KindContract, "test_unexpected_provider_call", "unexpected provider call", false)
}

func (p *failedReconciliationBodyFilePort) PrepareMessageSendReconciliation(_ context.Context, _ authn.BindingID, request chatwork.Request) (chatwork.PreparedMessageBody, error) {
	p.prepareCalls++
	p.privatePath = request.BodyFile
	return nil, fault.New(fault.KindInvalidInput, "chatwork_message_body_file_unreadable", "メッセージ本文ファイルを読み取れませんでした", false)
}

func (*failedReconciliationBodyFilePort) FinalizeMessageSendReconciliation(context.Context, authn.BindingID, chatwork.Request, chatwork.PreparedMessageBody) error {
	return fault.New(fault.KindContract, "test_unexpected_finalize", "unexpected finalize", false)
}

func (*failedReconciliationBodyFilePort) MatchMessageSendReconciliation(context.Context, authn.BindingID, chatwork.Request, chatwork.PreparedMessageBody, []string) ([]bool, error) {
	return nil, fault.New(fault.KindContract, "test_unexpected_match", "unexpected match", false)
}

func (*failedReconciliationBodyFilePort) DiscardMessageSendReconciliation(chatwork.PreparedMessageBody) {
}

func TestMessagesReconcileSendBodyFileFaultsBeforeReadsAndUsesExactReadOnlyRecovery(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages reconcile-send")
	underlying := &chatworkRuntimePort{}
	cli, authenticator, _, stderr := chatworkRuntimeCLI(t, spec, underlying)
	port := &failedReconciliationBodyFilePort{}
	cli.chatwork = chatworkcmd.New(port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--room", "42", "--reply-to", "101", "--to", "9", "--body-file", "private-reconcile-path",
	})
	if code != ExitUsage {
		t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.prepareCalls != 1 || port.providerCalls != 0 || underlying.calls != 0 {
		t.Fatalf("calls = auth %d prepare %d task %d base-provider %d", authenticator.calls, port.prepareCalls, port.providerCalls, underlying.calls)
	}
	for _, want := range []string{"chatwork_message_body_file_unreadable", "next_action: cwk help messages reconcile-send"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr lacks %q: %s", want, stderr.String())
		}
	}
	if port.privatePath != "private-reconcile-path" || strings.Contains(stderr.String(), "private-reconcile-path") {
		t.Fatal("body-file path did not stay inside the provider adapter error boundary")
	}
}

func TestMessagesReconcileSendInvalidLiteralFailsBeforeAuthentication(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages reconcile-send")
	port := &chatworkRuntimePort{}
	cli, authenticator, _, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--room", "42", "--body", "invalid\x00body",
	})
	if code != ExitUsage || authenticator.calls != 0 || port.calls != 0 {
		t.Fatalf("runChatwork() code=%d auth=%d calls=%d stderr=%s", code, authenticator.calls, port.calls, stderr.String())
	}
	if strings.Contains(stderr.String(), "invalid\x00body") {
		t.Fatal("literal body leaked into invalid-argument error")
	}
}
