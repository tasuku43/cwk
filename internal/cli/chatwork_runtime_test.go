package cli

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	appauthn "github.com/tasuku43/cwk/internal/app/authn"
	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	domainauthn "github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

type chatworkRuntimeAuthenticator struct {
	binding domainauthn.BindingID
	calls   int
}

func (a *chatworkRuntimeAuthenticator) Authenticate(_ context.Context, requirement domainauthn.Requirement) (domainauthn.Session, error) {
	a.calls++
	method := domainauthn.MethodPAT
	if len(requirement.Methods) > 0 {
		method = requirement.Methods[0]
	}
	return domainauthn.Session{
		Method:              method,
		Authority:           requirement.Authority,
		Audience:            requirement.Audience,
		SubjectID:           "synthetic-account",
		BindingID:           a.binding,
		GrantedCapabilities: append([]string(nil), requirement.RequiredCapabilities...),
	}, nil
}

type chatworkRuntimePort struct {
	calls   int
	request chatwork.Request
	result  func(chatwork.Request) (chatwork.Result, error)
}

func (p *chatworkRuntimePort) Execute(_ context.Context, _ domainauthn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	p.calls++
	p.request = request
	if p.result != nil {
		return p.result(request)
	}
	return chatwork.Result{Task: request.Task, Coverage: chatwork.Coverage{Kind: "bounded", Complete: true}}, nil
}

func TestRunChatworkRendersResolvedMessageContextWithoutPostProcessing(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "7")
	parent := chatworkRuntimeRef(t, chatwork.ReferenceMessage, "10")
	child := chatworkRuntimeRef(t, chatwork.ReferenceMessage, "11")
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		return chatwork.Result{
			Task:        request.Task,
			MessageRoom: request.Room,
			Coverage:    chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false, Description: "latest bounded window"},
			Messages: []chatwork.Message{
				{Ref: parent, Room: room, Sender: chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "1")}, Body: "parent"},
				{Ref: child, Room: room, Sender: chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "2")}, Body: "child", Reply: &chatwork.Relation{Kind: "reply", Target: parent, ExternalID: room.Value}},
			},
		}, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)

	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{"--room", "7", "--window=recent"})
	if code != ExitOK {
		t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("calls = auth %d, port %d; want 1, 1", authenticator.calls, port.calls)
	}
	if port.request.Room != room || !port.request.ForceRecent {
		t.Fatalf("request = %+v, want exact room and recent window", port.request)
	}
	// The flat presentation must preserve both exact references and project the
	// typed resolved relation through the provider-sequence edge.
	for _, want := range []string{"#1 10 ", "#2 11 ", "reply=#1"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("output does not contain %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunChatworkFiltersRepeatedSendersWithBoundedReplyContext(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "7")
	account := func(value, name string) chatwork.Account {
		return chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, value), Name: name}
	}
	message := func(value string, sender chatwork.Account, body string) chatwork.Message {
		return chatwork.Message{Ref: chatworkRuntimeRef(t, chatwork.ReferenceMessage, value), Room: room, Sender: sender, Body: body}
	}
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		unrelated := message("9", account("4", "Omitted"), "unrelated before")
		parent := message("10", account("1", "Aki"), "anchor parent")
		child := message("11", account("2", "Beni"), "anchor reply")
		child.Reply = &chatwork.Relation{Kind: "reply", Target: parent.Ref, ExternalID: room.Value}
		contextChild := message("12", account("3", "Chika"), "direct reply context")
		contextChild.Reply = &chatwork.Relation{Kind: "reply", Target: child.Ref, ExternalID: room.Value}
		trailing := message("13", account("4", "Omitted"), "unrelated after")
		return chatwork.Result{
			Task: request.Task, MessageRoom: request.Room,
			Coverage: chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false},
			Messages: []chatwork.Message{unrelated, parent, child, contextChild, trailing},
		}, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)

	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--room", "7", "--window=recent", "--sender", "1", "--sender=2", "--context", "replies",
	})
	if code != ExitOK {
		t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("calls = auth %d, port %d; want 1, 1", authenticator.calls, port.calls)
	}
	if len(port.request.MessageFilter.Senders) != 0 || port.request.MessageFilter.Context != "" {
		t.Fatalf("local message filter crossed the provider port: %+v", port.request.MessageFilter)
	}
	for _, want := range []string{
		"selection source-count=5 senders=[1,2] context=replies anchors=[#2,#3]",
		"#2 10 ", "#3 11 ", "reply=#2", "#4 12 ", "reply=#3",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("filtered output does not contain %q:\n%s", want, stdout.String())
		}
	}
	for _, forbidden := range []string{"#1 9 ", "#5 13 ", `name="Omitted"`} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Errorf("filtered output contains omitted source data %q:\n%s", forbidden, stdout.String())
		}
	}
}

func TestBuildChatworkMessageFilterDefaultsContextAndPreservesSenderOrder(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	parsed, err := parseChatworkArguments(spec, []string{"--room", "7", "--sender", "2", "--sender=1"})
	if err != nil {
		t.Fatal(err)
	}
	request, err := buildChatworkRequest(spec, parsed)
	if err != nil {
		t.Fatal(err)
	}
	if len(request.MessageFilter.Senders) != 2 ||
		request.MessageFilter.Senders[0].Value != "2" || request.MessageFilter.Senders[1].Value != "1" ||
		request.MessageFilter.Context != chatwork.MessageContextNone {
		t.Fatalf("message filter = %+v", request.MessageFilter)
	}
}

func TestRunChatworkRejectsInvalidMessageFilterBeforeAuthentication(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	cases := [][]string{
		{"--room", "7", "--context", "replies"},
		{"--room", "7", "--context", "none"},
		{"--room", "7", "--sender", "1", "--sender", "1"},
		{"--room", "7", "--sender", "07"},
		{"--room", "7", "--sender", "1", "--context", "thread"},
	}
	excessive := []string{"--room", "7"}
	for sender := 1; sender <= 101; sender++ {
		excessive = append(excessive, "--sender", strconv.Itoa(sender))
	}
	cases = append(cases, excessive)
	for _, args := range cases {
		port := &chatworkRuntimePort{}
		cli, authenticator, _, stderr := chatworkRuntimeCLI(t, spec, port)
		code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), args)
		if code != ExitUsage || !strings.Contains(stderr.String(), "code: invalid_arguments") {
			t.Errorf("args %v: code = %d, stderr = %s", args, code, stderr.String())
		}
		if authenticator.calls != 0 || port.calls != 0 {
			t.Errorf("args %v: calls = auth %d, port %d; want zero", args, authenticator.calls, port.calls)
		}
	}
}

func TestBuildChatworkRequestCoversEveryCatalogTask(t *testing.T) {
	specs := chatworkCommandSpecs()
	if len(specs) != 33 {
		t.Fatalf("Chatwork specs = %d, want fixed 33-task contract", len(specs))
	}
	seen := make(map[chatwork.Task]struct{}, len(specs))
	for _, spec := range specs {
		args := make([]string, 0, len(spec.Agent.Inputs)*2)
		for _, input := range spec.Agent.Inputs {
			if !input.Required {
				continue
			}
			value := chatworkRuntimeInputValue(input)
			args = append(args, input.Name, value)
		}
		parsed, err := parseChatworkArguments(spec, args)
		if err != nil {
			t.Errorf("%s parse error = %v", spec.Path, err)
			continue
		}
		request, err := buildChatworkRequest(spec, parsed)
		if err != nil {
			t.Errorf("%s build error = %v", spec.Path, err)
			continue
		}
		if err := request.Validate(); err != nil {
			t.Errorf("%s request validation error = %v", spec.Path, err)
		}
		if request.Task != spec.chatwork.Task {
			t.Errorf("%s task = %s, want %s", spec.Path, request.Task, spec.chatwork.Task)
		}
		if _, duplicate := seen[request.Task]; duplicate {
			t.Errorf("duplicate task mapping %s", request.Task)
		}
		seen[request.Task] = struct{}{}
	}
}

func TestRunChatworkConfirmationFailsBeforeAuthentication(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "rooms delete")
	for _, args := range [][]string{{"--room", "7"}, {"--room", "7", "--confirm=access-change"}} {
		port := &chatworkRuntimePort{}
		cli, authenticator, _, stderr := chatworkRuntimeCLI(t, spec, port)
		code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), args)
		if code != ExitRejected || !strings.Contains(stderr.String(), "code: mutation_rejected") {
			t.Errorf("args %v: code = %d, stderr = %s", args, code, stderr.String())
		}
		if authenticator.calls != 0 || port.calls != 0 {
			t.Errorf("args %v: calls = auth %d, port %d; want zero", args, authenticator.calls, port.calls)
		}
	}
}

func TestRunChatworkMapsRepeatedReferencesAndExecutesConfirmedMutationOnce(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "rooms create")
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		return chatwork.Result{
			Task:     request.Task,
			Coverage: chatwork.Coverage{Kind: "confirmed", Complete: true},
			Created:  []chatwork.Reference{chatworkRuntimeRef(t, chatwork.ReferenceRoom, "99")},
		}, nil
	}}
	cli, authenticator, _, stderr := chatworkRuntimeCLI(t, spec, port)
	args := []string{
		"--owner", "1", "--name", "project", "--admin", "2", "--admin=3",
		"--member", "4", "--readonly", "5", "--invite-approval", "required",
		"--confirm=access-change",
	}
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), args)
	if code != ExitOK {
		t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("calls = auth %d, port %d; want 1, 1", authenticator.calls, port.calls)
	}
	request := port.request
	if request.Account.Value != "1" || request.Name != "project" || len(request.Admins) != 2 ||
		request.Admins[0].Value != "2" || request.Admins[1].Value != "3" ||
		len(request.Members) != 1 || request.Members[0].Value != "4" ||
		len(request.ReadonlyMembers) != 1 || request.ReadonlyMembers[0].Value != "5" ||
		!request.InviteEnabled || !request.InviteApprovalSet || !request.InviteNeedsApproval {
		t.Fatalf("mapped request = %+v", request)
	}
}

func TestRunChatworkRejectsMalformedOrDuplicateInputsBeforeAuthentication(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages show")
	for _, args := range [][]string{
		{"--room", "07", "--message", "8"},
		{"--room", "7", "--room", "8", "--message", "9"},
		{"--room", "7", "--message", "8", "--unknown", "9"},
	} {
		port := &chatworkRuntimePort{}
		cli, authenticator, _, stderr := chatworkRuntimeCLI(t, spec, port)
		code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), args)
		if code != ExitUsage || !strings.Contains(stderr.String(), "code: invalid_arguments") {
			t.Errorf("args %v: code = %d, stderr = %s", args, code, stderr.String())
		}
		if authenticator.calls != 0 || port.calls != 0 {
			t.Errorf("args %v: calls = auth %d, port %d; want zero", args, authenticator.calls, port.calls)
		}
	}
}

func TestRunChatworkMutationPreservesUnclassifiedOutcome(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages send")
	port := &chatworkRuntimePort{result: func(chatwork.Request) (chatwork.Result, error) {
		return chatwork.Result{}, errors.New("private transport detail")
	}}
	cli, _, _, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{"--room", "7", "--body", "hello"})
	if code != ExitContract || !strings.Contains(stderr.String(), "code: unclassified_mutation_outcome") {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	if port.calls != 1 {
		t.Fatalf("port calls = %d, want 1", port.calls)
	}
	if strings.Contains(stderr.String(), "private transport detail") {
		t.Fatal("private mutation error leaked to stderr")
	}
}

func TestRunChatworkRejectsWrongResultVariantAfterConfirmedMutation(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages send")
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		return chatwork.Result{
			Task:     request.Task,
			Coverage: chatwork.Coverage{Kind: "confirmed", Complete: true},
			Messages: []chatwork.Message{{
				Ref: chatworkRuntimeRef(t, chatwork.ReferenceMessage, "8"), Room: chatworkRuntimeRef(t, chatwork.ReferenceRoom, "7"),
				Sender: chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "1")},
				Body:   strings.Repeat("x", maxChatworkOutputBytes+1),
			}},
		}, nil
	}}
	cli, _, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{"--room", "7", "--body", "hello"})
	if code != ExitContract || !strings.Contains(stderr.String(), "code: chatwork_result_invalid") {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	if port.calls != 1 {
		t.Fatalf("port calls = %d, want a single attempted mutation", port.calls)
	}
	if stdout.Len() != 0 {
		t.Fatalf("invalid result wrote %d bytes", stdout.Len())
	}
}

func chatworkRuntimeCLI(t *testing.T, spec CommandSpec, port *chatworkRuntimePort) (*CLI, *chatworkRuntimeAuthenticator, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	binding, err := domainauthn.NewBindingID("synthetic-runtime-binding")
	if err != nil {
		t.Fatalf("NewBindingID() error = %v", err)
	}
	authenticator := &chatworkRuntimeAuthenticator{binding: binding}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	return &CLI{
		Out: stdout, Err: stderr, catalog: NewCatalog(spec),
		chatwork: chatworkcmd.New(port), chatworkAuth: appauthn.New(authenticator),
	}, authenticator, stdout, stderr
}

func chatworkRuntimeSpec(t *testing.T, path string) CommandSpec {
	t.Helper()
	for _, spec := range chatworkCommandSpecs() {
		if spec.Path == path {
			return spec
		}
	}
	t.Fatalf("Chatwork spec %q not found", path)
	return CommandSpec{}
}

func chatworkRuntimeContext(spec CommandSpec) context.Context {
	return withCommandPath(withErrorFormat(context.Background(), errorFormatText), spec.Path)
}

func chatworkRuntimeIntent(spec CommandSpec) operation.Intent {
	return operation.Intent{Command: spec.Path, Effect: spec.Effect}
}

func chatworkRuntimeRef(t *testing.T, kind chatwork.ReferenceKind, value string) chatwork.Reference {
	t.Helper()
	ref, err := chatwork.NewReference(kind, value)
	if err != nil {
		t.Fatalf("NewReference(%s, %q) error = %v", kind, value, err)
	}
	return ref
}

func chatworkRuntimeInputValue(input CommandInput) string {
	if len(input.AllowedValues) > 0 {
		return input.AllowedValues[0]
	}
	if input.ReferenceKind != "" {
		return "1"
	}
	switch input.Name {
	case "--limit":
		return "1700000000"
	case "--path":
		return "fixture.bin"
	default:
		return "synthetic"
	}
}
