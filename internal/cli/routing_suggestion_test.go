package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	appauthn "github.com/tasuku43/cwk/internal/app/authn"
	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/app/configcmd"
	"github.com/tasuku43/cwk/internal/domain/commandselection"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

type routingSuggestionStore struct {
	profile    commandselection.Profile
	configured bool
	loads      int
}

func (s *routingSuggestionStore) Load(context.Context) (commandselection.Profile, bool, error) {
	s.loads++
	return s.profile, s.configured, nil
}

func (*routingSuggestionStore) Save(context.Context, commandselection.Profile) error {
	return errors.New("routing suggestion test must not save command selection")
}

func TestUnknownRootOptionSuggestsBeforeSelectionOrAuthentication(t *testing.T) {
	tests := []struct {
		args       []string
		suggestion string
	}{
		{args: []string{"--eror-format", "json", "doctor"}, suggestion: "--error-format"},
		{args: []string{"--hlep"}, suggestion: "--help"},
		{args: []string{"--versoin"}, suggestion: "--version"},
	}
	for _, test := range tests {
		t.Run(test.args[0], func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			command := newCLI(strings.NewReader(""), &stdout, &stderr, DefaultCatalog(), passingInspector("unused"))
			store := &routingSuggestionStore{}
			command.commandSelection = configcmd.New(store)
			factoryCalls := 0
			command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
				factoryCalls++
				return nil, nil, errors.New("root option rejection must precede authentication")
			}

			if code := runCLI(command, test.args); code != ExitUsage {
				t.Fatalf("Run(%v) exit=%d stderr=%q", test.args, code, stderr.String())
			}
			if stdout.Len() != 0 || !strings.Contains(stderr.String(), "code: invalid_root_options") ||
				!strings.Contains(stderr.String(), test.args[0]) || !strings.Contains(stderr.String(), test.suggestion) {
				t.Fatalf("Run(%v) stdout=%q stderr=%q", test.args, stdout.String(), stderr.String())
			}
			if store.loads != 0 || factoryCalls != 0 {
				t.Fatalf("Run(%v) selection loads=%d factory calls=%d", test.args, store.loads, factoryCalls)
			}
		})
	}
}

func TestHelpTypoNamesRootHelpAlternativeWithoutExecution(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader(""), &stdout, &stderr, DefaultCatalog(), passingInspector("unused"))
	factoryCalls := 0
	command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
		factoryCalls++
		return nil, nil, errors.New("invalid help must not initialize Chatwork")
	}

	if code := runCLI(command, []string{"help", "--hlep"}); code != ExitUsage {
		t.Fatalf("help typo exit=%d stderr=%q", code, stderr.String())
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), `help フラグ "--hlep" は不明です`) ||
		!strings.Contains(stderr.String(), "cwk --help") || factoryCalls != 0 {
		t.Fatalf("stdout=%q stderr=%q factory calls=%d", stdout.String(), stderr.String(), factoryCalls)
	}
}

func TestUnknownCommandSuggestsNearestActiveCanonicalPathWithoutExecution(t *testing.T) {
	show := utilitySpec("messages show")
	calls := 0
	show.handler = func(context.Context, *CLI, CommandSpec, operation.Intent, []string) int {
		calls++
		return ExitOK
	}
	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader(""), &stdout, &stderr, NewCatalog(show), passingInspector("unused"))

	if code := runCLI(command, []string{"messages", "shwo"}); code != ExitUsage {
		t.Fatalf("unknown command exit=%d stderr=%q", code, stderr.String())
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), `コマンド "messages shwo" は不明です`) ||
		!strings.Contains(stderr.String(), "cwk messages show") || calls != 0 {
		t.Fatalf("stdout=%q stderr=%q handler calls=%d", stdout.String(), stderr.String(), calls)
	}
}

func TestUnknownCommandDoesNotSuggestNearestMutationInDefaultCatalog(t *testing.T) {
	commands := DefaultCatalog().Commands()
	if got := closestCommandPath([]string{"messages", "search"}, commands); got != "messages send" {
		t.Fatalf("nearest candidate for messages search = %q, want messages send", got)
	}
	candidate, found := DefaultCatalog().Lookup("messages send")
	if !found || candidate.Effect != operation.EffectCreate {
		t.Fatalf("messages send candidate = %#v, found=%t; want create effect", candidate, found)
	}

	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader(""), &stdout, &stderr, DefaultCatalog(), passingInspector("unused"))
	factoryCalls := 0
	command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
		factoryCalls++
		return nil, nil, errors.New("an unknown command must not initialize Chatwork")
	}

	if code := runCLI(command, []string{"messages", "search"}); code != ExitUsage {
		t.Fatalf("unknown command exit=%d stderr=%q", code, stderr.String())
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), `code: unknown_command`) ||
		!strings.Contains(stderr.String(), "--contains") || !strings.Contains(stderr.String(), "cwk help messages list") ||
		strings.Contains(stderr.String(), "もしかして") || strings.Contains(stderr.String(), "cwk messages send") || factoryCalls != 0 {
		t.Fatalf("stdout=%q stderr=%q factory calls=%d", stdout.String(), stderr.String(), factoryCalls)
	}
}

func TestMessagesSearchGuidanceRequiresActiveBoundedListTask(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader(""), &stdout, &stderr, DefaultCatalog(), passingInspector("unused"))
	factoryCalls := 0
	command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
		factoryCalls++
		return nil, nil, errors.New("unknown path must not initialize Chatwork")
	}

	if code := runCLI(command, []string{"messages", "search"}); code != ExitUsage {
		t.Fatalf("active messages search exit=%d stderr=%q", code, stderr.String())
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "--contains") ||
		!strings.Contains(stderr.String(), "cwk help messages list") ||
		!strings.Contains(stderr.String(), "最大100件") || strings.Contains(stderr.String(), "cwk messages search") || factoryCalls != 0 {
		t.Fatalf("active search guidance stdout=%q stderr=%q factory calls=%d", stdout.String(), stderr.String(), factoryCalls)
	}
	if strings.Contains(stderr.String(), "messages send") {
		t.Fatalf("messages search suggested an unrelated task: %q", stderr.String())
	}

	hiddenCommand, hiddenStdout, hiddenStderr := newGuidanceCLI(t, []string{"rooms list"}, true)
	factoryCalls = 0
	hiddenCommand.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
		factoryCalls++
		return nil, nil, errors.New("unknown path must not initialize Chatwork")
	}
	if code := runCLI(hiddenCommand, []string{"messages", "search"}); code != ExitUsage {
		t.Fatalf("hidden messages search exit=%d stderr=%q", code, hiddenStderr.String())
	}
	if hiddenStdout.Len() != 0 || !strings.Contains(hiddenStderr.String(), "cwk --help") ||
		strings.Contains(hiddenStderr.String(), "messages list") || strings.Contains(hiddenStderr.String(), "--contains") ||
		strings.Contains(hiddenStderr.String(), "messages show") || factoryCalls != 0 {
		t.Fatalf("hidden search disclosed inactive guidance: stdout=%q stderr=%q factory calls=%d", hiddenStdout.String(), hiddenStderr.String(), factoryCalls)
	}

	unconfiguredCommand, unconfiguredStdout, unconfiguredStderr := newGuidanceCLI(t, nil, false)
	unconfiguredFactoryCalls := 0
	unconfiguredCommand.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
		unconfiguredFactoryCalls++
		return nil, nil, errors.New("unknown path must not initialize Chatwork")
	}
	if code := runCLI(unconfiguredCommand, []string{"messages", "search"}); code != ExitUsage {
		t.Fatalf("unconfigured messages search exit=%d stderr=%q", code, unconfiguredStderr.String())
	}
	if unconfiguredStdout.Len() != 0 || !strings.Contains(unconfiguredStderr.String(), "code: unknown_command") ||
		!strings.Contains(unconfiguredStderr.String(), "cwk --help") || strings.Contains(unconfiguredStderr.String(), "messages list") ||
		strings.Contains(unconfiguredStderr.String(), "--contains") || unconfiguredFactoryCalls != 0 {
		t.Fatalf("unconfigured search leaked a task path: stdout=%q stderr=%q factory calls=%d", unconfiguredStdout.String(), unconfiguredStderr.String(), unconfiguredFactoryCalls)
	}
}

func TestMessagesSearchGuidanceDoesNotReflectTrailingArguments(t *testing.T) {
	secret := "SEARCH-SECRET-CANARY\x1b\n--message messages show"
	tests := []struct {
		name        string
		state       string
		args        []string
		json        bool
		wantCode    string
		wantMessage []string
		forbidden   []string
	}{
		{
			name:        "active query canary text",
			state:       "active",
			args:        []string{"messages", "search", "--query", secret},
			wantCode:    "unknown_command",
			wantMessage: []string{"--contains", "cwk help messages list", "最大100件"},
			forbidden:   []string{secret, "messages show"},
		},
		{
			name:        "active unknown option json",
			state:       "active",
			args:        []string{"messages", "search", "--unknown-option", secret},
			json:        true,
			wantCode:    "unknown_command",
			wantMessage: []string{`"schema_version":1`, `"command":"help"`, "--contains", "最大100件"},
			forbidden:   []string{secret, "messages show"},
		},
		{
			name:      "hidden list",
			state:     "hidden",
			args:      []string{"messages", "search", "--query", secret},
			wantCode:  "unknown_command",
			forbidden: []string{secret, "messages list", "--contains", "messages show"},
		},
		{
			name:      "unconfigured",
			state:     "unconfigured",
			args:      []string{"messages", "search", "--unknown-option", secret},
			wantCode:  "unknown_command",
			forbidden: []string{secret, "messages list", "--contains", "messages show"},
		},
		{
			name:      "unsafe selection",
			state:     "unsafe",
			args:      []string{"messages", "search", "--query", secret},
			wantCode:  "command_selection_unsafe",
			forbidden: []string{secret, "messages list", "--contains", "messages show"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var command *CLI
			var stdout, stderr *bytes.Buffer
			switch test.state {
			case "active":
				var out, errOut bytes.Buffer
				command = newCLI(strings.NewReader(""), &out, &errOut, DefaultCatalog(), passingInspector("unused"))
				stdout, stderr = &out, &errOut
			case "hidden":
				command, stdout, stderr = newGuidanceCLI(t, []string{"rooms list"}, true)
			case "unconfigured":
				command, stdout, stderr = newGuidanceCLI(t, nil, false)
			case "unsafe":
				var out, errOut bytes.Buffer
				command = newCLI(strings.NewReader(""), &out, &errOut, DefaultCatalog(), passingInspector("unused"))
				command.commandSelection = configcmd.New(failingCommandSelectionStore{err: fault.New(fault.KindUnavailable, "command_selection_unsafe", "unsafe", false)})
				stdout, stderr = &out, &errOut
			default:
				t.Fatalf("unknown selection state %q", test.state)
			}
			factoryCalls := 0
			command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
				factoryCalls++
				return nil, nil, errors.New("unknown path must not initialize Chatwork")
			}
			args := append([]string(nil), test.args...)
			if test.json {
				args = append([]string{"--error-format", "json"}, args...)
			}
			wantExit := ExitUsage
			if test.state == "unsafe" {
				wantExit = ExitUnavailable
			}
			if code := runCLI(command, args); code != wantExit {
				t.Fatalf("Run(%v) exit=%d want=%d stderr=%q", args, code, wantExit, stderr.String())
			}
			if stdout.Len() != 0 || factoryCalls != 0 || !strings.Contains(stderr.String(), "code: "+test.wantCode) && !test.json {
				t.Fatalf("guidance crossed a boundary or has wrong code: stdout=%q stderr=%q factory calls=%d", stdout.String(), stderr.String(), factoryCalls)
			}
			if test.json && !strings.Contains(stderr.String(), `"code":"`+test.wantCode+`"`) {
				t.Fatalf("JSON error code mismatch: %q", stderr.String())
			}
			for _, want := range test.wantMessage {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("stderr lacks %q: %s", want, stderr.String())
				}
			}
			for _, forbidden := range test.forbidden {
				if strings.Contains(stderr.String(), forbidden) {
					t.Errorf("stderr reflected secret or inactive guidance %q: %s", forbidden, stderr.String())
				}
			}
		})
	}
}

func TestChatworkUnsupportedFlagGuidanceUsesOnlyActivePathsAndNeverEchoesValues(t *testing.T) {
	hostile := "SECRET-CANARY\x1b[31m\nmessages show --format json"
	tests := []struct {
		name       string
		paths      []string
		args       []string
		want       []string
		forbidden  []string
		wantFormat errorFormat
	}{
		{
			name:      "around points to visible show help",
			args:      []string{"messages", "list", "--room", "7", "--around", hostile},
			want:      []string{"cwk help messages show", "--around"},
			forbidden: []string{"SECRET-CANARY", "\x1b", "--message"},
		},
		{
			name:      "message points to visible show help",
			args:      []string{"messages", "list", "--room", "7", "--message", hostile},
			want:      []string{"cwk help messages show", "--message"},
			forbidden: []string{"SECRET-CANARY", "\x1b", "--around"},
		},
		{
			name:      "contains explains local bounded matching",
			args:      []string{"messages", "list", "--room", "7", "--query", hostile},
			want:      []string{"--contains", "最大100件", "recent", "API検索なし", "cwk help messages list"},
			forbidden: []string{"SECRET-CANARY", "\x1b", "messages show"},
		},
		{
			name:       "task format explains text and error format",
			args:       []string{"--error-format", "json", "messages", "list", "--room", "7", "--format", hostile},
			want:       []string{"--format", "成功結果は text", "--error-format json"},
			forbidden:  []string{"SECRET-CANARY", "\x1b"},
			wantFormat: errorFormatJSON,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var command *CLI
			var stdout, stderr *bytes.Buffer
			if test.paths == nil {
				command = newCLI(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, DefaultCatalog(), passingInspector("unused"))
				stdout, stderr = command.Out.(*bytes.Buffer), command.Err.(*bytes.Buffer)
			} else {
				command, stdout, stderr = newGuidanceCLI(t, test.paths, true)
			}
			factoryCalls := 0
			command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
				factoryCalls++
				return nil, nil, errors.New("invalid arguments must not initialize Chatwork")
			}

			if code := runCLI(command, test.args); code != ExitUsage {
				t.Fatalf("Run(%v) exit=%d stdout=%q stderr=%q", test.args, code, stdout.String(), stderr.String())
			}
			for _, want := range test.want {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("stderr lacks %q: %s", want, stderr.String())
				}
			}
			for _, forbidden := range test.forbidden {
				if strings.Contains(stderr.String(), forbidden) {
					t.Errorf("stderr contains forbidden %q: %s", forbidden, stderr.String())
				}
			}
			if stdout.Len() != 0 || factoryCalls != 0 {
				t.Fatalf("invalid input crossed a boundary or changed recovery: stdout=%q stderr=%q factory calls=%d", stdout.String(), stderr.String(), factoryCalls)
			}
			if test.wantFormat == errorFormatJSON {
				if !strings.HasPrefix(stderr.String(), `{"schema_version":1,"error":`) ||
					!strings.Contains(stderr.String(), `"code":"invalid_arguments"`) ||
					!strings.Contains(stderr.String(), `"command":"help messages list"`) {
					t.Fatalf("--error-format json did not preserve structured error recovery: %q", stderr.String())
				}
			} else if !strings.Contains(stderr.String(), "next_action: cwk help messages list") {
				t.Fatalf("text error changed its declared recovery: %q", stderr.String())
			}
		})
	}
}

func TestUnsupportedMessageAndMemberGuidanceDoesNotNameHiddenPaths(t *testing.T) {
	tests := []struct {
		name      string
		paths     []string
		args      []string
		want      string
		forbidden string
	}{
		{
			name:  "message show hidden",
			paths: []string{"rooms list", "messages list"},
			args:  []string{"messages", "list", "--room", "7", "--around", "9"},
			want:  "cwk help messages list", forbidden: "messages show",
		},
		{
			name:  "message target help hidden",
			paths: []string{"rooms list", "messages list"},
			args:  []string{"messages", "list", "--room", "7", "--message", "hostile\x1b\nvalue"},
			want:  "cwk help messages list", forbidden: "messages show",
		},
		{
			name:  "member find active",
			paths: []string{"rooms list", "members list", "members find"},
			args:  []string{"members", "list", "--room", "7", "--query", "hostile\x1b\nvalue"},
			want:  "cwk help members find", forbidden: "hostile",
		},
		{
			name:  "member find hidden",
			paths: []string{"rooms list", "members list"},
			args:  []string{"members", "list", "--room", "7", "--query", "hostile\x1b\nvalue"},
			want:  "cwk help members list", forbidden: "members find",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command, stdout, stderr := newGuidanceCLI(t, test.paths, true)
			factoryCalls := 0
			command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
				factoryCalls++
				return nil, nil, errors.New("invalid argument must not initialize Chatwork")
			}
			if code := runCLI(command, test.args); code != ExitUsage {
				t.Fatalf("Run(%v) exit=%d stderr=%q", test.args, code, stderr.String())
			}
			if stdout.Len() != 0 || !strings.Contains(stderr.String(), test.want) ||
				strings.Contains(stderr.String(), test.forbidden) || strings.Contains(stderr.String(), "hostile") ||
				strings.Contains(stderr.String(), "\x1b") || factoryCalls != 0 {
				t.Fatalf("guidance leaked a hidden path/value or crossed a boundary: stdout=%q stderr=%q factory calls=%d", stdout.String(), stderr.String(), factoryCalls)
			}
		})
	}
}

func TestMessageListAroundAndMessageHintsUseOnlyTheSubmittedFlag(t *testing.T) {
	secret := "FLAG-SECRET-CANARY\x1b\n--format json"
	for _, flag := range []string{"--around", "--message"} {
		for _, showActive := range []bool{true, false} {
			for _, jsonOutput := range []bool{false, true} {
				name := flag + "/hidden-show"
				if showActive {
					name = flag + "/active-show"
				}
				if jsonOutput {
					name += "/json"
				} else {
					name += "/text"
				}
				t.Run(name, func(t *testing.T) {
					var command *CLI
					var stdout, stderr *bytes.Buffer
					if showActive {
						var out, errOut bytes.Buffer
						command = newCLI(strings.NewReader(""), &out, &errOut, DefaultCatalog(), passingInspector("unused"))
						stdout, stderr = &out, &errOut
					} else {
						command, stdout, stderr = newGuidanceCLI(t, []string{"rooms list", "messages list"}, true)
					}
					factoryCalls := 0
					command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
						factoryCalls++
						return nil, nil, errors.New("invalid flag must not initialize Chatwork")
					}
					args := []string{"messages", "list", "--room", "7", flag, secret}
					if jsonOutput {
						args = append([]string{"--error-format", "json"}, args...)
					}
					if code := runCLI(command, args); code != ExitUsage {
						t.Fatalf("Run(%v) exit=%d stderr=%q", args, code, stderr.String())
					}
					if stdout.Len() != 0 || factoryCalls != 0 || strings.Contains(stderr.String(), secret) ||
						strings.Contains(stderr.String(), "\x1b") {
						t.Fatalf("flag guidance reflected value or crossed a boundary: stdout=%q stderr=%q factory calls=%d", stdout.String(), stderr.String(), factoryCalls)
					}
					otherFlag := "--message"
					if flag == "--message" {
						otherFlag = "--around"
					}
					if showActive {
						if !strings.Contains(stderr.String(), "cwk help messages show") || !strings.Contains(stderr.String(), flag) ||
							strings.Contains(stderr.String(), otherFlag) {
							t.Fatalf("active show hint does not name only %s: %q", flag, stderr.String())
						}
					} else if !strings.Contains(stderr.String(), "cwk help messages list") || !strings.Contains(stderr.String(), flag) ||
						strings.Contains(stderr.String(), "messages show") || strings.Contains(stderr.String(), otherFlag) {
						t.Fatalf("hidden show hint must name only submitted flag %s without exposing hidden path: %q", flag, stderr.String())
					}
					if jsonOutput {
						if !strings.HasPrefix(stderr.String(), `{"schema_version":1,"error":`) ||
							!strings.Contains(stderr.String(), `"code":"invalid_arguments"`) ||
							!strings.Contains(stderr.String(), `"command":"help messages list"`) {
							t.Fatalf("JSON error lost declared exact recovery: %q", stderr.String())
						}
					} else if !strings.Contains(stderr.String(), "next_action: cwk help messages list") {
						t.Fatalf("text error lost declared exact recovery: %q", stderr.String())
					}
				})
			}
		}
	}
}

func TestSelectionStatePrecedesUnsupportedInputGuidance(t *testing.T) {
	tests := []struct {
		name  string
		store configcmd.StorePort
		code  string
		args  []string
	}{
		{
			name: "unconfigured",
			code: "command_selection_required",
			args: []string{"messages", "list", "--room", "7", "--query", "needle"},
		},
		{
			name: "disabled task",
			code: "command_disabled",
			args: []string{"messages", "list", "--room", "7", "--query", "needle"},
		},
		{
			name:  "unsafe state",
			store: failingCommandSelectionStore{err: fault.New(fault.KindUnavailable, "command_selection_unsafe", "unsafe", false)},
			code:  "command_selection_unsafe",
			args:  []string{"messages", "list", "--room", "7", "--query", "needle"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			command := newCLI(strings.NewReader(""), &stdout, &stderr, DefaultCatalog(), passingInspector("unused"))
			command.commandSelection = configcmd.New(test.store)
			if test.name == "disabled task" {
				profile, err := commandselection.New([]string{"rooms list"})
				if err != nil {
					t.Fatalf("commandselection.New(): %v", err)
				}
				command.commandSelection = configcmd.New(&routingSuggestionStore{profile: profile, configured: true})
			}
			if test.name == "unconfigured" {
				command.commandSelection = configcmd.New(&routingSuggestionStore{configured: false})
			}
			factoryCalls := 0
			command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
				factoryCalls++
				return nil, nil, errors.New("selection failure must precede Chatwork")
			}
			wantExit := ExitRejected
			if test.code == "command_selection_unsafe" {
				wantExit = ExitUnavailable
			}
			if code := runCLI(command, test.args); code != wantExit {
				t.Fatalf("Run(%v) exit=%d want=%d stderr=%q", test.args, code, wantExit, stderr.String())
			}
			if stdout.Len() != 0 || !strings.Contains(stderr.String(), "code: "+test.code) ||
				strings.Contains(stderr.String(), "--contains") || strings.Contains(stderr.String(), "messages show") || factoryCalls != 0 {
				t.Fatalf("selection state did not take precedence: stdout=%q stderr=%q factory calls=%d", stdout.String(), stderr.String(), factoryCalls)
			}
		})
	}
}

func TestHelpTasksUsesOnlyActiveTaskNamespaces(t *testing.T) {
	tests := []struct {
		name        string
		activePaths []string
		args        []string
		want        []string
		forbidden   []string
		fullCatalog bool
	}{
		{
			name:        "both active routes to root",
			fullCatalog: true,
			args:        []string{"help", "tasks"},
			want:        []string{"複数のタスク名前空間", "cwk help"},
			forbidden:   []string{"room-tasks", "personal-tasks"},
		},
		{
			name:        "only room tasks routes to exact namespace",
			activePaths: []string{"rooms list", "room-tasks list"},
			args:        []string{"help", "tasks"},
			want:        []string{"cwk help room-tasks"},
			forbidden:   []string{"personal-tasks"},
		},
		{
			name:        "only personal tasks routes to exact namespace",
			activePaths: []string{"personal-tasks list"},
			args:        []string{"help", "tasks"},
			want:        []string{"cwk help personal-tasks"},
			forbidden:   []string{"room-tasks"},
		},
		{
			name:        "no active task namespace stays generic",
			activePaths: []string{"rooms list"},
			args:        []string{"help", "tasks"},
			want:        []string{"cwk --help"},
			forbidden:   []string{"room-tasks", "personal-tasks"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var command *CLI
			var stdout, stderr *bytes.Buffer
			if test.fullCatalog {
				var out, errOut bytes.Buffer
				command = newCLI(strings.NewReader(""), &out, &errOut, DefaultCatalog(), passingInspector("unused"))
				stdout, stderr = &out, &errOut
			} else {
				command, stdout, stderr = newGuidanceCLI(t, test.activePaths, true)
			}
			factoryCalls := 0
			command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
				factoryCalls++
				return nil, nil, errors.New("help selection must not initialize Chatwork")
			}
			if code := runCLI(command, test.args); code != ExitUsage {
				t.Fatalf("Run(%v) exit=%d stdout=%q stderr=%q", test.args, code, stdout.String(), stderr.String())
			}
			if stdout.Len() != 0 || factoryCalls != 0 || !strings.Contains(stderr.String(), "code: invalid_arguments") {
				t.Fatalf("help tasks was rewritten or executed: stdout=%q stderr=%q factory calls=%d", stdout.String(), stderr.String(), factoryCalls)
			}
			for _, want := range test.want {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("stderr lacks %q: %s", want, stderr.String())
				}
			}
			for _, forbidden := range test.forbidden {
				if strings.Contains(stderr.String(), forbidden) {
					t.Errorf("stderr disclosed inactive task namespace %q: %s", forbidden, stderr.String())
				}
			}
		})
	}
}

func TestHelpTasksSelectionFaultsPrecedeNamespaceGuidance(t *testing.T) {
	tests := []struct {
		name      string
		store     configcmd.StorePort
		code      string
		wantExit  int
		wantNext  string
		wantPaths []string
	}{
		{
			name:      "unconfigured",
			store:     &routingSuggestionStore{},
			code:      "command_selection_required",
			wantExit:  ExitRejected,
			wantNext:  "next_action: cwk config",
			wantPaths: []string{"room-tasks", "personal-tasks"},
		},
		{
			name:     "unsafe",
			store:    failingCommandSelectionStore{err: fault.New(fault.KindUnavailable, "command_selection_unsafe", "unsafe", false)},
			code:     "invalid_arguments",
			wantExit: ExitUsage,
			wantNext: "next_action: cwk help",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			command := newCLI(strings.NewReader(""), &stdout, &stderr, DefaultCatalog(), passingInspector("unused"))
			command.commandSelection = configcmd.New(test.store)
			factoryCalls := 0
			command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
				factoryCalls++
				return nil, nil, errors.New("help selection state must be resolved before Chatwork")
			}
			if code := runCLI(command, []string{"help", "tasks"}); code != test.wantExit {
				t.Fatalf("help tasks exit=%d want=%d stderr=%q", code, test.wantExit, stderr.String())
			}
			if stdout.Len() != 0 || !strings.Contains(stderr.String(), "code: "+test.code) ||
				!strings.Contains(stderr.String(), test.wantNext) || factoryCalls != 0 {
				t.Fatalf("selection state did not take precedence: stdout=%q stderr=%q factory calls=%d", stdout.String(), stderr.String(), factoryCalls)
			}
			for _, path := range test.wantPaths {
				if strings.Contains(stderr.String(), path) {
					t.Errorf("unconfigured state exposed task namespace %q: %s", path, stderr.String())
				}
			}
			if test.name == "unsafe" && (strings.Contains(stderr.String(), "room-tasks") || strings.Contains(stderr.String(), "personal-tasks")) {
				t.Fatalf("unsafe fallback exposed candidate namespace: %q", stderr.String())
			}
		})
	}
}

func newGuidanceCLI(t *testing.T, paths []string, configured bool) (*CLI, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	profile, err := commandselection.New(paths)
	if err != nil {
		t.Fatalf("commandselection.New(%v): %v", paths, err)
	}
	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader(""), &stdout, &stderr, DefaultCatalog(), passingInspector("unused"))
	command.commandSelection = configcmd.New(&routingSuggestionStore{profile: profile, configured: configured})
	return command, &stdout, &stderr
}

func TestUnknownCommandSuppressesNonReadNearestCandidateWithoutPromotingRead(t *testing.T) {
	tests := []struct {
		name   string
		effect operation.Effect
	}{
		{name: "create", effect: operation.EffectCreate},
		{name: "write", effect: operation.EffectWrite},
		{name: "unknown", effect: operation.EffectUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nearest := utilitySpec("rooms list")
			nearest.Effect = test.effect
			fartherRead := utilitySpec("rooms lists")
			if got := closestCommandPath([]string{"rooms", "lst"}, []CommandSpec{nearest, fartherRead}); got != nearest.Path {
				t.Fatalf("nearest candidate = %q, want %q", got, nearest.Path)
			}
			if got := closestCommandPath([]string{"rooms", "lst"}, []CommandSpec{fartherRead}); got != fartherRead.Path {
				t.Fatalf("farther read candidate = %q, want %q", got, fartherRead.Path)
			}
			got := unknownCommandMessage([]string{"rooms", "lst"}, NewCatalog(nearest, fartherRead))
			want := `コマンド "rooms lst" は不明です。`
			if got != want {
				t.Fatalf("unknownCommandMessage() = %q, want %q", got, want)
			}
		})
	}
}

func TestUnknownCommandSuggestionDoesNotLeakHiddenFullCatalogPath(t *testing.T) {
	visible := utilitySpec("rooms list")
	hidden := utilitySpec("messages show")
	hidden.Configurable = true
	profile, err := commandselection.New([]string{})
	if err != nil {
		t.Fatalf("commandselection.New(): %v", err)
	}
	store := &routingSuggestionStore{profile: profile, configured: true}
	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader(""), &stdout, &stderr, NewCatalog(visible, hidden), passingInspector("unused"))
	command.commandSelection = configcmd.New(store)

	if code := runCLI(command, []string{"messages", "shwo"}); code != ExitUsage {
		t.Fatalf("hidden typo exit=%d stderr=%q", code, stderr.String())
	}
	if stdout.Len() != 0 || strings.Contains(stderr.String(), "messages show") ||
		!strings.Contains(stderr.String(), "code: unknown_command") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestDisabledCommandAliasReturnsCanonicalDisabledFault(t *testing.T) {
	visible := utilitySpec("rooms list")
	hidden := utilitySpec("messages show")
	hidden.Configurable = true
	hidden.DeprecatedAliases = []string{"messages get"}
	profile, err := commandselection.New([]string{})
	if err != nil {
		t.Fatalf("commandselection.New(): %v", err)
	}
	store := &routingSuggestionStore{profile: profile, configured: true}
	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader(""), &stdout, &stderr, NewCatalog(visible, hidden), passingInspector("unused"))
	command.commandSelection = configcmd.New(store)

	if code := runCLI(command, []string{"messages", "get", "--room", "7"}); code != ExitRejected {
		t.Fatalf("hidden alias exit=%d stderr=%q", code, stderr.String())
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "code: command_disabled") ||
		!strings.Contains(stderr.String(), `コマンド "messages show" は現在無効です`) ||
		!strings.Contains(stderr.String(), "next_action: cwk config") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
