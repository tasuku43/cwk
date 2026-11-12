package cli

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	appauthn "github.com/tasuku43/cwk/internal/app/authn"
	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	domainauthn "github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestMessageLinkDiscoveryLocalAndSelection(t *testing.T) {
	const url = "https://www.chatwork.com/#!rid42-101"
	h := newCommandSelectionHarness(t, strings.NewReader(""))
	h.command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
		t.Fatal("link resolution invoked authentication or provider setup")
		return nil, nil, nil
	}
	if code := runCLI(h.command, []string{"messages", "resolve-link", "--url", url}); code == ExitOK || !strings.Contains(h.stderr.String(), "command_selection_required") {
		t.Fatalf("unconfigured code=%d stderr=%q", code, h.stderr.String())
	}
	h.reset(strings.NewReader(""))
	saveCommandSelection(t, h.store, []string{"rooms list"})
	if code := runCLI(h.command, []string{"messages", "resolve-link", "--url", url}); code == ExitOK || !strings.Contains(h.stderr.String(), "command_disabled") {
		t.Fatalf("disabled code=%d stderr=%q", code, h.stderr.String())
	}
	h.reset(strings.NewReader(""))
	saveCommandSelection(t, h.store, []string{"messages resolve-link"})
	if code := runCLI(h.command, []string{"messages", "resolve-link", "--url", url}); code != ExitOK {
		t.Fatalf("exit=%d stderr=%q", code, h.stderr.String())
	}
	if got, want := h.stdout.String(), "room-ref=42 message-ref=101 existence=not-checked access=not-checked\n"; got != want {
		t.Fatalf("output=%q want=%q", got, want)
	}
	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"help"}); code != ExitOK {
		t.Fatalf("help exit=%d", code)
	}
	if output := h.stdout.String(); !strings.Contains(output, "リンクから参照を得る: 'cwk messages resolve-link --help'") || !strings.Contains(output, "人に config で messages show の有効化") || strings.Contains(output, "cwk messages show --help") {
		t.Fatalf("resolver-only help=%q", output)
	}
}

func TestMessageLinkDiscoveryRejectsHostileInputWithoutEcho(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader(""))
	saveCommandSelection(t, h.store, []string{"messages resolve-link"})
	for _, input := range []string{"https://www.chatwork.com/#!rid42-101\x1b[31mSECRET", "https://attacker.example/#!rid42-101", strings.Repeat("A", 96)} {
		h.reset(strings.NewReader(""))
		if code := runCLI(h.command, []string{"messages", "resolve-link", "--url", input}); code == ExitOK || h.stdout.Len() != 0 || strings.Contains(h.stderr.String(), input) || strings.Contains(h.stderr.String(), "SECRET") {
			t.Fatalf("unexpected result exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
		}
	}
}

func TestMessageLinkReferencesFlowToShowWithoutURLInput(t *testing.T) {
	catalog := DefaultCatalog()
	view, _, err := catalog.ActiveView([]string{"messages resolve-link", "messages show"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := view.Lookup("messages resolve-link"); !ok {
		t.Fatal("resolver absent")
	}
	show, ok := view.Lookup("messages show")
	if !ok {
		t.Fatal("show absent")
	}
	args, err := parseChatworkArguments(show, []string{"--room", "42", "--message", "101"})
	if err != nil || args.first("--room") != "42" || args.first("--message") != "101" {
		t.Fatalf("parsed=%v err=%v", args, err)
	}
	for _, argv := range [][]string{{"--room", "https://www.chatwork.com/#!rid42-101", "--message", "101"}, {"--room", "42", "--message", "https://www.chatwork.com/#!rid42-101"}} {
		if _, err := parseChatworkArguments(show, argv); err == nil {
			t.Fatal("action accepted link")
		}
	}
}

func TestMessageLinkOutputReferencesReachShowPort(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader(""))
	saveCommandSelection(t, h.store, []string{"messages resolve-link", "messages show"})
	if code := runCLI(h.command, []string{"messages", "resolve-link", "--url", "https://www.chatwork.com/#!rid42-101"}); code != ExitOK {
		t.Fatalf("resolve exit=%d stderr=%q", code, h.stderr.String())
	}
	fields := strings.Fields(h.stdout.String())
	if len(fields) < 2 {
		t.Fatalf("resolve output=%q", h.stdout.String())
	}
	room := strings.TrimPrefix(fields[0], "room-ref=")
	message := strings.TrimPrefix(fields[1], "message-ref=")
	port := &chatworkRuntimePort{result: func(request chatwork.Request) (chatwork.Result, error) {
		return chatwork.Result{Task: request.Task, Coverage: chatwork.Coverage{Kind: "single_operation", Complete: true}, Messages: []chatwork.Message{{Ref: request.Message, Room: request.Room, Sender: chatwork.Account{Ref: chatworkRuntimeRef(t, chatwork.ReferenceAccount, "7"), Name: "Aki"}, Body: "hello", SendTime: 1}}}, nil
	}}
	binding, err := domainauthn.NewBindingID("link-roundtrip")
	if err != nil {
		t.Fatal(err)
	}
	h.command.chatwork = chatworkcmd.New(port)
	h.command.chatworkAuth = appauthn.New(&chatworkRuntimeAuthenticator{binding: binding})
	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"messages", "show", "--room", room, "--message", message, "--context", "none"}); code != ExitOK {
		t.Fatalf("show exit=%d stderr=%q", code, h.stderr.String())
	}
	if port.calls != 1 || port.request.Room.Value != "42" || port.request.Message.Value != "101" {
		t.Fatalf("port calls=%d request=%+v stderr=%q", port.calls, port.request, h.stderr.String())
	}
}

func TestMessageLinkHelpRoutesWithBothCommandsEnabled(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader(""))
	saveCommandSelection(t, h.store, []string{"messages resolve-link", "messages show"})
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{[]string{"help"}, []string{"cwk messages resolve-link --help", "cwk messages show --help"}},
		{[]string{"messages", "--help"}, []string{"cwk messages resolve-link --url <message-link>", "cwk messages show"}},
		{[]string{"messages", "resolve-link", "--help"}, []string{"--url", "#!rid<room-id>-<message-id>", "未確認"}},
	} {
		h.reset(strings.NewReader(""))
		if code := runCLI(h.command, tc.args); code != ExitOK {
			t.Fatalf("args=%v exit=%d stderr=%q", tc.args, code, h.stderr.String())
		}
		for _, want := range tc.want {
			if !strings.Contains(h.stdout.String(), want) {
				t.Errorf("args=%v missing %q output=%q", tc.args, want, h.stdout.String())
			}
		}
	}
}

func TestMessageLinkStaticFallbackIsExplicit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink requires platform-specific privileges")
	}
	h := newCommandSelectionHarness(t, strings.NewReader(""))
	app := filepath.Join(h.base, "cwk")
	if err := os.Mkdir(app, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(h.base, "target")
	if err := os.WriteFile(target, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(app, "command-selection.json")); err != nil {
		t.Fatal(err)
	}
	if code := runCLI(h.command, []string{"help"}); code != ExitOK {
		t.Fatalf("exit=%d stderr=%q", code, h.stderr.String())
	}
	for _, want := range []string{staticHelpNotice, "cwk messages resolve-link --help", "cwk messages show --help"} {
		if !strings.Contains(h.stdout.String(), want) {
			t.Fatalf("fallback missing %q: %q", want, h.stdout.String())
		}
	}
}
