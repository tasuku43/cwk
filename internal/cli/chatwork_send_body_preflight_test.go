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

type failedBodyPreflightPort struct {
	prepareCalls int
	executeCalls int
	prepared     error
}

func (p *failedBodyPreflightPort) Execute(context.Context, authn.BindingID, chatwork.Request) (chatwork.Result, error) {
	p.executeCalls++
	return chatwork.Result{}, fault.New(fault.KindContract, "test_unexpected_provider_call", "unexpected provider call", false)
}

func (p *failedBodyPreflightPort) PrepareMessageBody(context.Context, authn.BindingID, chatwork.Request) (chatwork.PreparedMessageBody, error) {
	p.prepareCalls++
	return nil, p.prepared
}

func (*failedBodyPreflightPort) ExecutePreparedMessageSend(context.Context, authn.BindingID, chatwork.Request, chatwork.PreparedMessageBody) (chatwork.Result, error) {
	return chatwork.Result{}, fault.New(fault.KindContract, "test_unexpected_prepared_send", "unexpected prepared send", false)
}

func (*failedBodyPreflightPort) DiscardPreparedMessageBody(chatwork.PreparedMessageBody) {}

func TestRunMessagesSendBodyPreflightFaultRecoversWithoutReplyReadOrPost(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages send")
	underlying := &chatworkRuntimePort{}
	cli, authenticator, _, stderr := chatworkRuntimeCLI(t, spec, underlying)
	port := &failedBodyPreflightPort{prepared: fault.New(
		fault.KindInvalidInput,
		"chatwork_message_body_file_unreadable",
		"メッセージ本文ファイルを読み取れませんでした",
		false,
	)}
	cli.chatwork = chatworkcmd.New(port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--room", "42", "--reply-to", "101", "--body-file", "private-body-path",
	})
	if code != ExitUsage {
		t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.prepareCalls != 1 || port.executeCalls != 0 || underlying.calls != 0 {
		t.Fatalf("calls = auth %d prepare %d task %d base-provider %d", authenticator.calls, port.prepareCalls, port.executeCalls, underlying.calls)
	}
	for _, want := range []string{"chatwork_message_body_file_unreadable", "next_action: cwk help messages send"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr lacks %q: %s", want, stderr.String())
		}
	}
	if strings.Contains(stderr.String(), "private-body-path") {
		t.Fatal("body-file path leaked into the public error")
	}
}
