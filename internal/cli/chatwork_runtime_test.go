package cli

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	appauthn "github.com/tasuku43/cwk/internal/app/authn"
	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	domainauthn "github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

type chatworkRuntimeAuthenticator struct {
	binding     domainauthn.BindingID
	calls       int
	requirement domainauthn.Requirement
	err         error
}

func (a *chatworkRuntimeAuthenticator) Authenticate(_ context.Context, requirement domainauthn.Requirement) (domainauthn.Session, error) {
	a.calls++
	a.requirement = requirement.Clone()
	if a.err != nil {
		return domainauthn.Session{}, a.err
	}
	method := domainauthn.MethodPAT
	if len(requirement.Methods) > 0 {
		method = requirement.Methods[0]
	}
	return domainauthn.Session{
		Method:              method,
		Authority:           requirement.Authority,
		Audience:            requirement.Audience,
		SubjectID:           "synthetic-account",
		AccountID:           requirement.AccountID,
		BindingID:           a.binding,
		GrantedCapabilities: append([]string(nil), requirement.RequiredCapabilities...),
	}, nil
}

type chatworkRuntimePort struct {
	calls    int
	request  chatwork.Request
	requests []chatwork.Request
	result   func(chatwork.Request) (chatwork.Result, error)
}

func (p *chatworkRuntimePort) Execute(_ context.Context, _ domainauthn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	p.calls++
	p.request = request
	p.requests = append(p.requests, request)
	if p.result != nil {
		return p.result(request)
	}
	return chatwork.Result{Task: request.Task, Coverage: chatwork.Coverage{Kind: "bounded", Complete: true}}, nil
}

func TestRunChatworkFindsMemberCandidatesBeforeExactSenderUse(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "members find")
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		if request.Task != chatwork.TaskMembersList || request.MemberQuery != "" {
			t.Fatalf("provider request = %+v", request)
		}
		return chatwork.Result{
			Task:     request.Task,
			Coverage: chatwork.Coverage{Kind: "provider_collection", Complete: true},
			Accounts: []chatwork.Account{
				{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "2501"), Name: "篠原 花子", Role: "member"},
				{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "2502"), Name: "山田 太郎", Role: "admin"},
				{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "2503"), Name: "篠原 太郎", Role: "readonly"},
			},
		}, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)

	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{"--room", "7", "--query", "篠原"})
	if code != ExitOK {
		t.Fatalf("runChatwork() code=%d stderr=%s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("calls = auth %d provider %d", authenticator.calls, port.calls)
	}
	for _, want := range []string{
		`member-candidates query="篠原" source-count=3 candidate-count=2 complete=true`,
		`2501 "篠原 花子" "member"`,
		`2503 "篠原 太郎" "readonly"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("candidate output lacks %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "2502") || strings.Contains(stdout.String(), "山田") {
		t.Fatalf("candidate output leaked non-match:\n%s", stdout.String())
	}
}

func TestRunChatworkPassesDiscoveredAccountReferenceUnchangedIntoSenderSelection(t *testing.T) {
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "3501")
	selected := chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "2501"), Name: "篠原 花子", Role: "member"}
	other := chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "2502"), Name: "山田 太郎", Role: "member"}
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		switch request.Task {
		case chatwork.TaskMembersList:
			return chatwork.Result{
				Task: request.Task, Coverage: chatwork.Coverage{Kind: "provider_collection", Complete: true},
				Accounts: []chatwork.Account{other, selected},
			}, nil
		case chatwork.TaskMessagesList:
			return chatwork.Result{
				Task: request.Task, MessageRoom: room,
				Coverage: chatwork.Coverage{Kind: "latest_window", Limit: 100, Complete: false},
				Messages: []chatwork.Message{
					{Ref: chatworkRuntimeRef(t, chatwork.ReferenceMessage, "1102"), Room: room, Sender: selected, Body: "selected"},
					{Ref: chatworkRuntimeRef(t, chatwork.ReferenceMessage, "1103"), Room: room, Sender: other, Body: "other"},
				},
			}, nil
		default:
			return chatwork.Result{}, errors.New("unexpected task")
		}
	}}

	findSpec := chatworkRuntimeSpec(t, "members find")
	findCLI, _, findOut, findErr := chatworkRuntimeCLI(t, findSpec, port)
	if code := runChatwork(
		chatworkRuntimeContext(findSpec),
		findCLI,
		findSpec,
		chatworkRuntimeIntent(findSpec),
		[]string{"--room", room.Value, "--query", selected.Name},
	); code != ExitOK {
		t.Fatalf("members find code = %d, stderr = %s", code, findErr.String())
	}

	discovered := ""
	lines := strings.Split(strings.TrimSpace(findOut.String()), "\n")
	for index, line := range lines {
		if strings.HasPrefix(line, "schema: ") && index+1 < len(lines) {
			fields := strings.Fields(lines[index+1])
			if len(fields) > 0 {
				discovered = fields[0]
			}
			break
		}
	}
	if discovered != selected.Ref.Value {
		t.Fatalf("discovered reference = %q, output = %q", discovered, findOut.String())
	}

	listSpec := chatworkRuntimeSpec(t, "messages list")
	listCLI, _, listOut, listErr := chatworkRuntimeCLI(t, listSpec, port)
	if code := runChatwork(
		chatworkRuntimeContext(listSpec),
		listCLI,
		listSpec,
		chatworkRuntimeIntent(listSpec),
		[]string{"--room", room.Value, "--sender", discovered, "--resolve-relations", "0"},
	); code != ExitOK {
		t.Fatalf("messages list code = %d, stderr = %s", code, listErr.String())
	}
	if !strings.Contains(listOut.String(), "#1 1102 ") || strings.Contains(listOut.String(), "1103") {
		t.Fatalf("sender-selected output = %s", listOut.String())
	}
	if port.calls != 2 {
		t.Fatalf("provider calls = %d, want 2", port.calls)
	}
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
				{Ref: child, Room: room, Sender: chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "2")}, Body: "child", Replies: []chatwork.Relation{{Kind: "reply", Target: parent, ExternalID: room.Value}}},
			},
		}, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)

	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{"--room", "7"})
	if code != ExitOK {
		t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("calls = auth %d, port %d; want 1, 1", authenticator.calls, port.calls)
	}
	if port.request.Room != room || !port.request.ForceRecent || port.request.MessageFilter.Count != 0 {
		t.Fatalf("request = %+v, want exact room and recent window", port.request)
	}
	if !strings.Contains(stdout.String(), "source-limit=100") || strings.Contains(stdout.String(), "selection ") {
		t.Fatalf("unlimited output changed its source-bound or selection contract:\n%s", stdout.String())
	}
	// The flat presentation must preserve both exact references and project the
	// typed resolved relation through the provider-sequence edge.
	for _, want := range []string{"#1 10 ", "#2 11 ", "reply=#1"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("output does not contain %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunChatworkExplicitChangesPreservesDifferentialWindow(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "7")
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		kind := "latest_window"
		description := "latest bounded window"
		if !request.ForceRecent {
			kind = "differential_window"
			description = "bounded provider changes"
		}
		return chatwork.Result{
			Task:        request.Task,
			MessageRoom: request.Room,
			Coverage:    chatwork.Coverage{Kind: kind, Limit: 100, Complete: false, Description: description},
			Messages:    []chatwork.Message{},
		}, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)

	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{"--room", "7", "--window", "changes"})
	if code != ExitOK {
		t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("calls = auth %d, port %d; want 1, 1", authenticator.calls, port.calls)
	}
	if port.request.Room != room || port.request.ForceRecent {
		t.Fatalf("request = %+v, want exact room and explicit differential window", port.request)
	}
	if !strings.Contains(stdout.String(), "messages room-ref=7 count=0 window=changes source-limit=100") {
		t.Fatalf("explicit changes output lost its typed window:\n%s", stdout.String())
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
		child.Replies = []chatwork.Relation{{Kind: "reply", Target: parent.Ref, ExternalID: room.Value}}
		contextChild := message("12", account("3", "Chika"), "direct reply context")
		contextChild.Replies = []chatwork.Relation{{Kind: "reply", Target: child.Ref, ExternalID: room.Value}}
		trailing := message("13", account("4", "Omitted"), "unrelated after")
		return chatwork.Result{
			Task: request.Task, MessageRoom: request.Room,
			Coverage: chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false},
			Messages: []chatwork.Message{unrelated, parent, child, contextChild, trailing},
		}, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)

	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--room", "7", "--sender", "1", "--sender=2", "--context", "replies",
	})
	if code != ExitOK {
		t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("calls = auth %d, port %d; want 1, 1", authenticator.calls, port.calls)
	}
	if !port.request.ForceRecent || len(port.request.MessageFilter.Senders) != 0 || port.request.MessageFilter.Context != "" ||
		port.request.MessageFilter.StartIndex != 0 || port.request.MessageFilter.Count != 0 {
		t.Fatalf("default window or provider request is wrong: %+v", port.request)
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

func TestRunChatworkCountsNewestCandidatesAndKeepsProviderOrder(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "7")
	account := chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "1"), Name: "Aki"}
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		message := func(ref string, sent int64) chatwork.Message {
			return chatwork.Message{
				Ref: chatworkRuntimeRef(t, chatwork.ReferenceMessage, ref), Room: room,
				Sender: account, Body: ref, SendTime: sent,
			}
		}
		return chatwork.Result{
			Task: request.Task, MessageRoom: request.Room,
			Coverage: chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false},
			// The newest candidates are provider positions one and three. Output
			// must select by send time without reordering those source positions.
			Messages: []chatwork.Message{message("10", 30), message("11", 10), message("12", 20)},
		}, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)

	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--room", "7", "--count", "2",
	})
	if code != ExitOK {
		t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("calls = auth %d, port %d; want 1, 1", authenticator.calls, port.calls)
	}
	if !port.request.ForceRecent || len(port.request.MessageFilter.Senders) != 0 || port.request.MessageFilter.Context != "" ||
		port.request.MessageFilter.StartIndex != 0 || port.request.MessageFilter.Count != 0 {
		t.Fatalf("default window or provider request is wrong: %+v", port.request)
	}
	for _, want := range []string{
		"messages room-ref=7 count=2 window=recent source-limit=100",
		"selection source-count=3 candidate-count=3 start-index=1 count=2 items-per-page=2 next-start-index=3",
		"#1 10 ",
		"#3 12 ",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("limited output lacks %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "#2 11 ") || strings.Index(stdout.String(), "#1 10 ") > strings.Index(stdout.String(), "#3 12 ") {
		t.Fatalf("limited output retained the wrong candidate or reordered provider positions:\n%s", stdout.String())
	}
}

func TestRunChatworkResolvesTokyoTodayOnceAndFiltersBeforeContext(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "7")
	account := chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "1"), Name: "Aki"}
	period, err := chatwork.NewMessageDayPeriod("2026-07-17")
	if err != nil {
		t.Fatal(err)
	}
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		message := func(ref string, sent int64) chatwork.Message {
			return chatwork.Message{Ref: chatworkRuntimeRef(t, chatwork.ReferenceMessage, ref), Room: room, Sender: account, Body: ref, SendTime: sent}
		}
		parent := message("10", period.Since-1)
		child := message("11", period.Since)
		child.Replies = []chatwork.Relation{{Kind: "reply", Target: parent.Ref, ExternalID: room.Value}}
		return chatwork.Result{
			Task: request.Task, MessageRoom: request.Room,
			Coverage: chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false},
			Messages: []chatwork.Message{
				parent,
				child,
				message("12", period.Until-1),
				message("13", period.Until),
			},
		}, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
	clockCalls := 0
	cli.now = func() time.Time {
		clockCalls++
		return time.Date(2026, 7, 17, 14, 59, 0, 0, time.UTC)
	}

	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--room", "7", "--on", "today", "--context", "replies",
	})
	if code != ExitOK {
		t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
	}
	if clockCalls != 1 || authenticator.calls != 1 || port.calls != 1 {
		t.Fatalf("calls = clock %d, auth %d, port %d", clockCalls, authenticator.calls, port.calls)
	}
	if !reflect.DeepEqual(port.request.MessageFilter, chatwork.MessageFilter{}) {
		t.Fatalf("local period leaked to provider: %+v", port.request.MessageFilter)
	}
	for _, want := range []string{
		"selection source-count=4 since=" + strconv.FormatInt(period.Since, 10),
		" until=" + strconv.FormatInt(period.Until, 10),
		" on=2026-07-17 time-zone=Asia/Tokyo candidate-count=2 context=replies anchors=[#2,#3]",
		"#1 10 ", "#2 11 ", "reply=#1", "#3 12 ",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("period output does not contain %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "#4 13 ") {
		t.Fatalf("exclusive until record was rendered:\n%s", stdout.String())
	}
}

func TestBuildMessagePeriodResolvesYesterdayWithFixedTokyoClock(t *testing.T) {
	calls := 0
	period, err := buildMessagePeriod(chatworkArguments{"--on": {"yesterday"}}, func() time.Time {
		calls++
		return time.Date(2026, 7, 17, 15, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || period.Day != "2026-07-17" || period.TimeZone != chatwork.MessageDayTimeZone {
		t.Fatalf("calls=%d period=%+v", calls, period)
	}
}

func TestRunChatworkDefaultsFiveRelationFetchesAndSupportsExplicitZero(t *testing.T) {
	for _, test := range []struct {
		name      string
		args      []string
		wantCalls int
		resolved  bool
	}{
		{name: "default five", args: []string{"--room", "7"}, wantCalls: 2, resolved: true},
		{name: "explicit zero", args: []string{"--room", "7", "--resolve-relations", "0"}, wantCalls: 1, resolved: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			spec := chatworkRuntimeSpec(t, "messages list")
			room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "7")
			account := chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "1"), Name: "Aki"}
			target := chatworkRuntimeRef(t, chatwork.ReferenceMessage, "99")
			child := chatwork.Message{
				Ref: chatworkRuntimeRef(t, chatwork.ReferenceMessage, "10"), Room: room,
				Sender: account, Body: "child", SendTime: 200,
				Replies: []chatwork.Relation{{Kind: "reply", Target: target, ExternalID: room.Value}},
			}
			parent := chatwork.Message{Ref: target, Room: room, Sender: account, Body: "parent", SendTime: 100}
			port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
				switch request.Task {
				case chatwork.TaskMessagesList:
					if request.MessageRelationFetchLimit != 0 {
						t.Fatalf("local relation budget leaked to list request: %+v", request)
					}
					return chatwork.Result{
						Task: request.Task, MessageRoom: request.Room,
						Coverage: chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false},
						Messages: []chatwork.Message{child},
					}, nil
				case chatwork.TaskMessagesShow:
					if request.Room != room || request.Message != target {
						t.Fatalf("exact relation request = %+v", request)
					}
					return chatwork.Result{
						Task: request.Task, Coverage: chatwork.Coverage{Kind: "single_operation", Complete: true},
						Messages: []chatwork.Message{parent},
					}, nil
				default:
					t.Fatalf("unexpected task %s", request.Task)
					return chatwork.Result{}, nil
				}
			}}
			cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
			code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), test.args)
			if code != ExitOK {
				t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
			}
			if authenticator.calls != 1 || port.calls != test.wantCalls {
				t.Fatalf("calls = auth %d provider %d", authenticator.calls, port.calls)
			}
			if test.resolved {
				for _, want := range []string{
					"relation-resolution fetch-limit=5 fetch-attempts=1 targets=1",
					"reply=message-ref:99",
					"relation-context provenance=fetched message-ref=99",
				} {
					if !strings.Contains(stdout.String(), want) {
						t.Errorf("default relation output lacks %q:\n%s", want, stdout.String())
					}
				}
			} else if !strings.Contains(stdout.String(), "unresolved-relations=1") || strings.Contains(stdout.String(), "relation-resolution ") {
				t.Fatalf("explicit zero did not preserve unresolved no-fetch output:\n%s", stdout.String())
			}
		})
	}
}

func TestRunChatworkReportsPeriodOutsideReachableRecentWindow(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "7")
	account := chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "1"), Name: "Aki"}
	oldest := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC).Unix()
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		return chatwork.Result{
			Task: request.Task, MessageRoom: request.Room,
			Coverage: chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false},
			Messages: []chatwork.Message{{
				Ref: chatworkRuntimeRef(t, chatwork.ReferenceMessage, "10"), Room: room,
				Sender: account, Body: "reachable", SendTime: oldest,
			}},
		}, nil
	}}
	cli, _, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{"--room", "7", "--on", "2026-07-08"})
	if code != ExitOK {
		t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{
		"count=0 window=recent",
		"oldest-reachable-message-ref=10 oldest-reachable-send-time=" + strconv.FormatInt(oldest, 10),
		"period-reachability=out-of-reachable-window",
		"selection source-count=1",
		"candidate-count=0",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("unreachable-period output lacks %q:\n%s", want, stdout.String())
		}
	}
	if port.calls != 1 {
		t.Fatalf("provider calls = %d, want one list call", port.calls)
	}
}

func TestBuildChatworkRequestRejectsRelationFetchBudgetOutsideZeroToHundred(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	for _, value := range []string{"-1", "101", "not-a-number"} {
		parsed, err := parseChatworkArguments(spec, []string{"--room", "7", "--resolve-relations", value})
		if err != nil {
			continue
		}
		if _, err := buildChatworkRequest(spec, parsed, nil); err == nil {
			t.Fatalf("invalid relation budget %q passed", value)
		}
	}
}

func TestRunChatworkStartIndexAndCountSelectAnUnambiguousOrdinalSlice(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	room := chatworkRuntimeRef(t, chatwork.ReferenceRoom, "7")
	account := chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "1"), Name: "Aki"}
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		message := func(ref string, sent int64) chatwork.Message {
			return chatwork.Message{Ref: chatworkRuntimeRef(t, chatwork.ReferenceMessage, ref), Room: room, Sender: account, Body: ref, SendTime: sent}
		}
		return chatwork.Result{
			Task: request.Task, MessageRoom: request.Room,
			Coverage: chatwork.Coverage{Kind: "recent-window", Limit: 100, Complete: false},
			Messages: []chatwork.Message{message("10", 30), message("11", 10), message("12", 20)},
		}, nil
	}}
	cli, authenticator, stdout, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--room", "7", "--start-index", "2", "--count", "1",
	})
	if code != ExitOK {
		t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 || !reflect.DeepEqual(port.request.MessageFilter, chatwork.MessageFilter{}) {
		t.Fatalf("calls/filter = auth %d port %d %+v", authenticator.calls, port.calls, port.request.MessageFilter)
	}
	for _, want := range []string{
		"selection source-count=3 candidate-count=3 start-index=2 count=1 items-per-page=1 next-start-index=3",
		"#3 12 ",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("continued output lacks %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "#1 10 ") || strings.Contains(stdout.String(), "#2 11 ") {
		t.Fatalf("continued output did not select only newest rank 2:\n%s", stdout.String())
	}
}

func TestBuildChatworkMessageFilterDefaultsContextAndPreservesSenderOrder(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	parsed, err := parseChatworkArguments(spec, []string{"--room", "7", "--sender", "2", "--sender=1"})
	if err != nil {
		t.Fatal(err)
	}
	request, err := buildChatworkRequest(spec, parsed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(request.MessageFilter.Senders) != 2 ||
		request.MessageFilter.Senders[0].Value != "2" || request.MessageFilter.Senders[1].Value != "1" ||
		request.MessageFilter.Context != chatwork.MessageContextNone || !request.ForceRecent {
		t.Fatalf("message request = %+v", request)
	}
}

func TestBuildChatworkMessageWindowDefaultsRecentAndAcceptsExplicitValues(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	for name, test := range map[string]struct {
		args        []string
		forceRecent bool
	}{
		"omitted":          {args: []string{"--room", "7"}, forceRecent: true},
		"explicit recent":  {args: []string{"--room", "7", "--window", "recent"}, forceRecent: true},
		"explicit changes": {args: []string{"--room", "7", "--window", "changes"}, forceRecent: false},
	} {
		t.Run(name, func(t *testing.T) {
			parsed, err := parseChatworkArguments(spec, test.args)
			if err != nil {
				t.Fatal(err)
			}
			request, err := buildChatworkRequest(spec, parsed, nil)
			if err != nil {
				t.Fatal(err)
			}
			if request.ForceRecent != test.forceRecent {
				t.Fatalf("ForceRecent = %t, want %t", request.ForceRecent, test.forceRecent)
			}
		})
	}
}

func TestBuildChatworkMessageCountAcceptsInclusiveBoundsAndDefaultsStartIndexAndContext(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	for _, value := range []string{"1", "100"} {
		parsed, err := parseChatworkArguments(spec, []string{"--room", "7", "--count", value})
		if err != nil {
			t.Fatalf("count %s parse error = %v", value, err)
		}
		request, err := buildChatworkRequest(spec, parsed, nil)
		if err != nil {
			t.Fatalf("count %s build error = %v", value, err)
		}
		want, _ := strconv.Atoi(value)
		if request.MessageFilter.StartIndex != 1 || request.MessageFilter.Count != want ||
			request.MessageFilter.Context != chatwork.MessageContextNone || request.Limit != 0 || !request.ForceRecent {
			t.Fatalf("count %s request = %+v", value, request)
		}
	}
	parsed, err := parseChatworkArguments(spec, []string{"--room", "7", "--count", "10", "--context", "replies"})
	if err != nil {
		t.Fatal(err)
	}
	request, err := buildChatworkRequest(spec, parsed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if request.MessageFilter.StartIndex != 1 || request.MessageFilter.Count != 10 ||
		request.MessageFilter.Context != chatwork.MessageContextReplies || !request.ForceRecent {
		t.Fatalf("count-only reply context request = %+v", request)
	}
}

func TestBuildChatworkStartIndexAcceptsInclusiveBoundsAndStatesCountSemantics(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "messages list")
	for _, value := range []string{"1", "100"} {
		parsed, err := parseChatworkArguments(spec, []string{"--room", "7", "--start-index", value, "--count", "20"})
		if err != nil {
			t.Fatalf("start-index %s parse error = %v", value, err)
		}
		request, err := buildChatworkRequest(spec, parsed, nil)
		if err != nil {
			t.Fatalf("start-index %s build error = %v", value, err)
		}
		want, _ := strconv.Atoi(value)
		if request.MessageFilter.StartIndex != want || request.MessageFilter.Count != 20 ||
			request.MessageFilter.Context != chatwork.MessageContextNone || !request.ForceRecent {
			t.Fatalf("start-index %s request = %+v", value, request)
		}
	}
}

func TestBuildRoomTaskDeadlineRemainsIndependentFromMessageIndexSelection(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "room-tasks create")
	parsed, err := parseChatworkArguments(spec, []string{
		"--room", "7", "--body", "task", "--assignee", "1", "--limit", "1700000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	request, err := buildChatworkRequest(spec, parsed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if request.Limit != 1700000000 || len(request.MessageFilter.Senders) != 0 ||
		request.MessageFilter.Context != "" || request.MessageFilter.StartIndex != 0 || request.MessageFilter.Count != 0 {
		t.Fatalf("room task deadline/message filter = %d/%+v", request.Limit, request.MessageFilter)
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
		{"--room", "7", "--count", "0"},
		{"--room", "7", "--count", "-1"},
		{"--room", "7", "--count", "101"},
		{"--room", "7", "--count", "ten"},
		{"--room", "7", "--count", "10", "--count", "20"},
		{"--room", "7", "--start-index", "0"},
		{"--room", "7", "--start-index", "-1"},
		{"--room", "7", "--start-index", "101"},
		{"--room", "7", "--start-index", "ten"},
		{"--room", "7", "--start-index", "10", "--start-index", "20"},
		{"--room", "7", "--on", "today"},
		{"--room", "7", "--on", "7/17"},
		{"--room", "7", "--on", "2026-02-30"},
		{"--room", "7", "--on", "2026-07-17", "--since", "2026-07-17T00:00:00+09:00"},
		{"--room", "7", "--on", "2026-07-17", "--until", "2026-07-18T00:00:00+09:00"},
		{"--room", "7", "--since", "2026-07-17"},
		{"--room", "7", "--since", "2026-07-17T00:00:00"},
		{"--room", "7", "--since", "2026-07-17T00:00:00.5+09:00"},
		{"--room", "7", "--since", "2026-07-18T00:00:00+09:00", "--until", "2026-07-17T00:00:00+09:00"},
		{"--room", "7", "--since", "2026-07-17T00:00:00+09:00", "--until", "2026-07-17T00:00:00+09:00"},
		{"--room", "7", "--limit", "10"},
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
	if len(specs) != 34 {
		t.Fatalf("Chatwork specs = %d, want fixed 34-task contract", len(specs))
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
		if spec.chatwork.Task == chatwork.TaskInviteLinkUpdate {
			args = append(args, "--regenerate-code")
		}
		parsed, err := parseChatworkArguments(spec, args)
		if err != nil {
			t.Errorf("%s parse error = %v", spec.Path, err)
			continue
		}
		request, err := buildChatworkRequest(spec, parsed, nil)
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

func TestContactRequestAcceptanceRequiresAccessChangeConfirmationBeforeAuthentication(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "contact-requests accept")
	for _, args := range [][]string{
		{"--request", "7"},
		{"--request", "7", "--confirm=destructive"},
	} {
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
		"--account", "1", "--name", "project", "--admin", "2", "--admin=3",
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
	if authenticator.requirement.AccountID != "1" {
		t.Fatalf("authentication account = %q, want 1", authenticator.requirement.AccountID)
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

func TestRunChatworkRejectsInvalidOfficialRoomFieldsBeforeAuthentication(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "rooms create")
	for name, args := range map[string][]string{
		"oversized name": {"--account", "1", "--name", strings.Repeat("名", 256), "--admin", "1", "--confirm=access-change"},
		"unknown icon":   {"--account", "1", "--name", "project", "--admin", "1", "--icon", "arbitrary", "--confirm=access-change"},
	} {
		t.Run(name, func(t *testing.T) {
			port := &chatworkRuntimePort{}
			cli, authenticator, _, stderr := chatworkRuntimeCLI(t, spec, port)
			code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), args)
			if code != ExitUsage || !strings.Contains(stderr.String(), "code: invalid_arguments") {
				t.Fatalf("code = %d, stderr = %s", code, stderr.String())
			}
			if authenticator.calls != 0 || port.calls != 0 {
				t.Fatalf("calls = auth %d, port %d; want zero", authenticator.calls, port.calls)
			}
		})
	}
}

func TestRunChatworkRejectsIncompleteOrInvalidInviteLinkReplacementBeforeAuthentication(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "invite-link update")
	valid := []string{"--invite", "7", "--approval", "required", "--description", "replacement", "--confirm=access-change"}
	tests := [][]string{
		valid,
		append(append([]string{}, valid...), "--code", "valid-code", "--regenerate-code"),
		{"--invite", "7", "--code", "invalid!", "--approval", "required", "--description", "replacement", "--confirm=access-change"},
		{"--invite", "7", "--code", strings.Repeat("a", 51), "--approval", "required", "--description", "replacement", "--confirm=access-change"},
		{"--invite", "7", "--code", "valid-code", "--description", "replacement", "--confirm=access-change"},
		{"--invite", "7", "--code", "valid-code", "--approval", "required", "--confirm=access-change"},
		{"--invite", "7", "--code", "valid-code", "--approval", "required", "--description", "", "--confirm=access-change"},
	}

	for _, args := range tests {
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

func TestRunChatworkMapsExplicitInviteLinkRegeneration(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "invite-link update")
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		return chatwork.Result{
			Task: request.Task, Coverage: chatwork.Coverage{Kind: "confirmed", Complete: true},
			InviteLink: &chatwork.InviteLink{Ref: request.Invite, Public: true, Description: request.Description},
		}, nil
	}}
	cli, authenticator, _, stderr := chatworkRuntimeCLI(t, spec, port)
	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--invite", "7", "--regenerate-code", "--approval", "not-required",
		"--description", "replacement", "--confirm=access-change",
	})
	if code != ExitOK {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	if authenticator.calls != 1 || port.calls != 1 || !port.request.InviteRegenerateCode ||
		port.request.InviteCode != "" || !port.request.InviteApprovalSet ||
		port.request.InviteNeedsApproval || !port.request.DescriptionSet || port.request.Description != "replacement" {
		t.Fatalf("calls/request = auth %d port %d request %+v", authenticator.calls, port.calls, port.request)
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

func TestRunChatworkPreservesCancellationBeforeMutationPortStarts(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "rooms create")
	port := &chatworkRuntimePort{}
	cli, authenticator, _, stderr := chatworkRuntimeCLI(t, spec, port)
	authenticator.err = fault.New(fault.KindCanceled, "operation_canceled", "認証アカウントの確認がキャンセルされました", true)

	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--account", "1", "--name", "project", "--admin", "1", "--confirm=access-change",
	})
	if code != ExitCanceled || !strings.Contains(stderr.String(), "code: operation_canceled") {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	if port.calls != 0 {
		t.Fatalf("mutation port calls = %d, want zero", port.calls)
	}
	if strings.Contains(stderr.String(), "unclassified_mutation_outcome") {
		t.Fatal("pre-mutation cancellation was reported as an unknown mutation outcome")
	}
}

func TestRunChatworkConservativelyClassifiesCancellationAfterMutationPortStarts(t *testing.T) {
	spec := chatworkRuntimeSpec(t, "rooms create")
	port := &chatworkRuntimePort{result: func(chatwork.Request) (chatwork.Result, error) {
		return chatwork.Result{}, fault.New(fault.KindCanceled, "operation_canceled", "変更送信後にキャンセルされました", true)
	}}
	cli, _, _, stderr := chatworkRuntimeCLI(t, spec, port)

	code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), []string{
		"--account", "1", "--name", "project", "--admin", "1", "--confirm=access-change",
	})
	if code != ExitContract || !strings.Contains(stderr.String(), "code: unclassified_mutation_outcome") {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	if port.calls != 1 {
		t.Fatalf("mutation port calls = %d, want one", port.calls)
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

func TestRunChatworkRateLimitSeparatesReadAndMutationRecovery(t *testing.T) {
	const privateCanary = "private-rate-limit-body"
	tests := []struct {
		name           string
		path           string
		args           []string
		code           string
		retryable      bool
		retryAfter     time.Duration
		retryAfterText string
		nextCommand    string
	}{
		{
			name: "read timing unknown", path: "messages list", args: []string{"--room", "7"},
			code: "chatwork_rate_limited", retryable: true, retryAfterText: "retry_after: unknown",
			nextCommand: "cwk messages list",
		},
		{
			name: "mutation timing is advisory", path: "messages send", args: []string{"--room", "7", "--body", "hello"},
			code: "chatwork_mutation_rate_limited", retryable: false, retryAfter: 10 * time.Second,
			retryAfterText: "retry_after: 10s", nextCommand: "cwk help messages send",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := chatworkRuntimeSpec(t, test.path)
			port := &chatworkRuntimePort{result: func(chatwork.Request) (chatwork.Result, error) {
				providerFault := fault.Wrap(
					fault.KindRateLimited,
					test.code,
					"Chatwork のレート制限に達しました。",
					test.retryable,
					errors.New(privateCanary),
				)
				providerFault.RetryAfter = test.retryAfter
				return chatwork.Result{}, providerFault
			}}
			cli, _, _, stderr := chatworkRuntimeCLI(t, spec, port)

			code := runChatwork(chatworkRuntimeContext(spec), cli, spec, chatworkRuntimeIntent(spec), test.args)
			if code != ExitRateLimited {
				t.Fatalf("runChatwork() code = %d, stderr = %s", code, stderr.String())
			}
			for _, want := range []string{
				"code: " + test.code,
				"retryable: " + strconv.FormatBool(test.retryable),
				test.retryAfterText,
				"next_action: " + test.nextCommand,
			} {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("stderr does not contain %q:\n%s", want, stderr.String())
				}
			}
			if strings.Contains(stderr.String(), privateCanary) {
				t.Fatal("private provider error body leaked to stderr")
			}
		})
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
	case "--start-index", "--count":
		return "1"
	case "--path":
		return "fixture.bin"
	default:
		return "synthetic"
	}
}
