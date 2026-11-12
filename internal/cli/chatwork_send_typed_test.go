package cli

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

func TestMessageSendParsesTypedRefsInInputOrderAndKeepsRoomAsParent(t *testing.T) {
	spec, ok := DefaultCatalog().Lookup("messages send")
	if !ok {
		t.Fatal("messages send is missing from the catalog")
	}
	manualBody := "手書き [To:77] と [rp aid=8 to=42-88] をそのまま残す"
	args := []string{"--room", "42", "--to", "9", "--to=3", "--reply-to", "101", "--body", manualBody}
	parsed, err := parseChatworkArguments(spec, args)
	if err != nil {
		t.Fatal(err)
	}
	request, err := buildChatworkRequest(spec, parsed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("typed message-send request: %v", err)
	}
	if request.Room != (chatwork.Reference{Kind: chatwork.ReferenceRoom, Value: "42"}) ||
		request.Body != manualBody || request.MessageReplyTarget == nil ||
		request.MessageReplyTarget.Message != (chatwork.Reference{Kind: chatwork.ReferenceMessage, Value: "101"}) ||
		!reflect.DeepEqual(request.MessageRecipients, []chatwork.Reference{
			{Kind: chatwork.ReferenceAccount, Value: "9"}, {Kind: chatwork.ReferenceAccount, Value: "3"},
		}) {
		t.Fatalf("request = %+v", request)
	}

	base := operation.Intent{Command: spec.Path, Effect: spec.Effect}
	executionRequest, err := buildChatworkExecutionRequest(spec, base, parsed)
	if err != nil {
		t.Fatal(err)
	}
	wantTarget := operation.TargetRef{Kind: "chatwork-message", ParentID: "42"}
	wantPreflight := []operation.PreflightRef{
		{Input: "--to", Kind: string(chatwork.ReferenceAccount), Value: "9"},
		{Input: "--to", Kind: string(chatwork.ReferenceAccount), Value: "3"},
		{Input: "--reply-to", Kind: string(chatwork.ReferenceMessage), Value: "101"},
	}
	if executionRequest.Intent.Target != wantTarget ||
		!reflect.DeepEqual(executionRequest.Intent.PreflightRefs, wantPreflight) ||
		!reflect.DeepEqual(executionRequest.ExpectedPreflightRefs, wantPreflight) {
		t.Fatalf("execution intent = %+v, expected refs = %+v", executionRequest.Intent, executionRequest.ExpectedPreflightRefs)
	}
}

func TestRunMessageSendRejectsRecipientOverflowBeforeAuthenticationOrProvider(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages send")
	args := []string{"--room", "42", "--body", "-"}
	for index := 1; index <= chatwork.MaxMessageRecipients+1; index++ {
		args = append(args, "--to", fmt.Sprint(index))
	}
	port := &chatworkRuntimePort{}
	cli, authenticator, _, stderr := chatworkRuntimeCLI(t, spec, port)
	stdin := &countingMessageInput{reader: strings.NewReader("body")}
	cli.In = stdin
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), args)
	if code != ExitUsage {
		t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
	}
	if authenticator.calls != 0 || port.calls != 0 || stdin.reads != 0 {
		t.Fatalf("recipient overflow reached stdin=%d auth=%d provider=%d", stdin.reads, authenticator.calls, port.calls)
	}
	if !strings.Contains(stderr.String(), "100") {
		t.Fatalf("recipient bound was not explained: %s", stderr.String())
	}
}

type countingMessageInput struct {
	reader *strings.Reader
	reads  int
}

func (r *countingMessageInput) Read(buffer []byte) (int, error) {
	r.reads++
	return r.reader.Read(buffer)
}

func TestMessageSendRejectsNoncanonicalAndOverBoundTypedReferences(t *testing.T) {
	spec, _ := DefaultCatalog().Lookup("messages send")
	for name, args := range map[string][]string{
		"leading-zero account":      {"--room", "42", "--to", "07", "--body", "body"},
		"leading-zero message":      {"--room", "42", "--reply-to", "0101", "--body", "body"},
		"duplicate singleton reply": {"--room", "42", "--reply-to", "101", "--reply-to", "102", "--body", "body"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseChatworkArguments(spec, args); err == nil {
				t.Fatal("noncanonical or repeated typed reference parsed successfully")
			}
		})
	}

	args := []string{"--room", "42", "--body", "body"}
	for index := 1; index <= chatwork.MaxMessageRecipients+1; index++ {
		args = append(args, "--to", fmt.Sprint(index))
	}
	parsed, err := parseChatworkArguments(spec, args)
	if err != nil {
		t.Fatalf("repeatable recipient flags failed to parse before bounded task validation: %v", err)
	}
	request, err := buildChatworkRequest(spec, parsed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := request.Validate(); err == nil {
		t.Fatal("101 typed recipients passed domain validation")
	}
}
