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

func TestHiddenCommandAliasDoesNotLeakItsCanonicalPath(t *testing.T) {
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

	if code := runCLI(command, []string{"messages", "get", "--room", "7"}); code != ExitUsage {
		t.Fatalf("hidden alias exit=%d stderr=%q", code, stderr.String())
	}
	if stdout.Len() != 0 || strings.Contains(stderr.String(), "messages show") ||
		!strings.Contains(stderr.String(), `コマンド "messages get --room 7" は不明です`) {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
