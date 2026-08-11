package cli

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	appauthn "github.com/tasuku43/cwk/internal/app/authn"
	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/app/configcmd"
	"github.com/tasuku43/cwk/internal/domain/commandselection"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

func TestStaticCommandAliasesNormalizeOnlyDeclaredExactPrefixes(t *testing.T) {
	catalog := DefaultCatalog()
	tests := []struct {
		args []string
		want []string
	}{
		{args: []string{"messages", "get", "--room", "123", "--message", "456"}, want: []string{"messages", "show", "--room", "123", "--message", "456"}},
		{args: []string{"rooms", "members", "find", "--room", "123"}, want: []string{"members", "find", "--room", "123"}},
		{args: []string{"rooms", "members", "list", "--room", "123"}, want: []string{"members", "list", "--room", "123"}},
		{args: []string{"rooms", "members", "replace", "--room", "123"}, want: []string{"members", "replace", "--room", "123"}},
		{args: []string{"messages", "getty", "--help"}, want: []string{"messages", "getty", "--help"}},
		{args: []string{"rooms", "members", "show", "--help"}, want: []string{"rooms", "members", "show", "--help"}},
	}
	for _, test := range tests {
		if got := normalizeCommandAlias(catalog, test.args); !reflect.DeepEqual(got, test.want) {
			t.Errorf("normalizeCommandAlias(%v)=%v want=%v", test.args, got, test.want)
		}
	}
}

func TestStaticCommandAliasesRouteToCanonicalHelpWithoutAuthentication(t *testing.T) {
	tests := []struct {
		alias     []string
		canonical []string
	}{
		{alias: []string{"messages", "get", "--help"}, canonical: []string{"messages", "show", "--help"}},
		{alias: []string{"rooms", "members", "find", "--help"}, canonical: []string{"members", "find", "--help"}},
		{alias: []string{"rooms", "members", "list", "--help"}, canonical: []string{"members", "list", "--help"}},
		{alias: []string{"rooms", "members", "replace", "--help"}, canonical: []string{"members", "replace", "--help"}},
	}
	for _, test := range tests {
		t.Run(strings.Join(test.alias, " "), func(t *testing.T) {
			aliasOut, aliasErr := &bytes.Buffer{}, &bytes.Buffer{}
			alias := newCLI(strings.NewReader(""), aliasOut, aliasErr, DefaultCatalog(), passingInspector("unused"))
			aliasFactoryCalls := 0
			alias.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
				aliasFactoryCalls++
				return nil, nil, errors.New("help must not initialize Chatwork")
			}
			if code := runCLI(alias, test.alias); code != ExitOK {
				t.Fatalf("alias exit=%d stderr=%q", code, aliasErr.String())
			}

			canonicalOut, canonicalErr := &bytes.Buffer{}, &bytes.Buffer{}
			canonical := newCLI(strings.NewReader(""), canonicalOut, canonicalErr, DefaultCatalog(), passingInspector("unused"))
			if code := runCLI(canonical, test.canonical); code != ExitOK {
				t.Fatalf("canonical exit=%d stderr=%q", code, canonicalErr.String())
			}
			if aliasOut.String() != canonicalOut.String() || aliasErr.String() != canonicalErr.String() {
				t.Fatalf("alias output=%q/%q canonical=%q/%q", aliasOut.String(), aliasErr.String(), canonicalOut.String(), canonicalErr.String())
			}
			if aliasFactoryCalls != 0 {
				t.Fatalf("alias help initialized Chatwork %d times", aliasFactoryCalls)
			}
		})
	}
}

func TestHelpSelectorsNormalizeDeclaredCommandAliasesAndIgnoreRedundantHelpFlags(t *testing.T) {
	canonicalOut, canonicalErr := &bytes.Buffer{}, &bytes.Buffer{}
	canonical := newCLI(strings.NewReader(""), canonicalOut, canonicalErr, DefaultCatalog(), passingInspector("unused"))
	if code := runCLI(canonical, []string{"help", "messages", "show"}); code != ExitOK {
		t.Fatalf("canonical help exit=%d stderr=%q", code, canonicalErr.String())
	}

	for _, args := range [][]string{
		{"help", "messages", "get"},
		{"help", "messages.get"},
		{"help", "messages", "show", "--help"},
		{"help", "--help", "messages", "show"},
	} {
		var stdout, stderr bytes.Buffer
		command := newCLI(strings.NewReader(""), &stdout, &stderr, DefaultCatalog(), passingInspector("unused"))
		if code := runCLI(command, args); code != ExitOK {
			t.Fatalf("help %v exit=%d stderr=%q", args, code, stderr.String())
		}
		if stdout.String() != canonicalOut.String() || stderr.Len() != 0 {
			t.Fatalf("help %v output=%q/%q canonical=%q", args, stdout.String(), stderr.String(), canonicalOut.String())
		}
	}
}

func TestStaticCommandAliasesDispatchOnlyTheirCanonicalHandlers(t *testing.T) {
	tests := []struct {
		alias  []string
		target string
	}{
		{alias: []string{"messages", "get", "marker"}, target: "messages show"},
		{alias: []string{"rooms", "members", "find", "marker"}, target: "members find"},
		{alias: []string{"rooms", "members", "list", "marker"}, target: "members list"},
		{alias: []string{"rooms", "members", "replace", "marker"}, target: "members replace"},
	}
	for _, test := range tests {
		t.Run(strings.Join(test.alias[:len(test.alias)-1], " "), func(t *testing.T) {
			commands := DefaultCatalog().Commands()
			calls := 0
			for index := range commands {
				if commands[index].Path != test.target {
					continue
				}
				commands[index].handler = func(_ context.Context, _ *CLI, command CommandSpec, intent operation.Intent, args []string) int {
					calls++
					if command.Path != test.target || intent.Command != test.target || !reflect.DeepEqual(args, []string{"marker"}) {
						t.Errorf("handler command=%q intent=%q args=%v", command.Path, intent.Command, args)
					}
					return ExitOK
				}
			}
			var stdout, stderr bytes.Buffer
			command := newCLI(strings.NewReader(""), &stdout, &stderr, NewCatalog(commands...), passingInspector("unused"))
			if code := runCLI(command, test.alias); code != ExitOK {
				t.Fatalf("alias exit=%d stderr=%q", code, stderr.String())
			}
			if calls != 1 || stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatalf("calls=%d stdout=%q stderr=%q", calls, stdout.String(), stderr.String())
			}
		})
	}
}

func TestHelpSelectorsAcceptDotsAsWordSeparators(t *testing.T) {
	for _, command := range DefaultCatalog().Commands() {
		dotted := strings.ReplaceAll(command.Path, " ", ".")
		_, selector, err := parseHelpArgs([]string{dotted})
		if err != nil || selector != command.Path {
			t.Errorf("parseHelpArgs(%q) selector=%q err=%v want=%q", dotted, selector, err, command.Path)
		}
	}

	var dottedOut, dottedErr bytes.Buffer
	dottedCLI := newCLI(strings.NewReader(""), &dottedOut, &dottedErr, DefaultCatalog(), passingInspector("unused"))
	if code := runCLI(dottedCLI, []string{"help", "messages.list"}); code != ExitOK {
		t.Fatalf("dotted help exit=%d stderr=%q", code, dottedErr.String())
	}
	var spacedOut, spacedErr bytes.Buffer
	spacedCLI := newCLI(strings.NewReader(""), &spacedOut, &spacedErr, DefaultCatalog(), passingInspector("unused"))
	if code := runCLI(spacedCLI, []string{"help", "messages", "list"}); code != ExitOK {
		t.Fatalf("spaced help exit=%d stderr=%q", code, spacedErr.String())
	}
	if dottedOut.String() != spacedOut.String() || dottedErr.String() != spacedErr.String() {
		t.Fatalf("dotted=%q/%q spaced=%q/%q", dottedOut.String(), dottedErr.String(), spacedOut.String(), spacedErr.String())
	}
}

func TestHelpFlagAnywhereAfterKnownExactCommandIgnoresInvocationInputs(t *testing.T) {
	var canonicalOut, canonicalErr bytes.Buffer
	canonical := newCLI(strings.NewReader(""), &canonicalOut, &canonicalErr, DefaultCatalog(), passingInspector("unused"))
	if code := runCLI(canonical, []string{"help", "files", "list"}); code != ExitOK {
		t.Fatalf("canonical help exit=%d stderr=%q", code, canonicalErr.String())
	}

	for _, args := range [][]string{
		{"files", "list", "--room", "123", "--help"},
		{"files", "list", "--help", "--room", "123"},
		{"files", "list", "--room", "--help"},
		{"files", "list", "--unknown", "value", "-h", "--room", "invalid"},
	} {
		var stdout, stderr bytes.Buffer
		command := newCLI(strings.NewReader(""), &stdout, &stderr, DefaultCatalog(), passingInspector("unused"))
		factoryCalls := 0
		command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
			factoryCalls++
			return nil, nil, errors.New("help must not initialize Chatwork")
		}
		if code := runCLI(command, args); code != ExitOK {
			t.Errorf("Run(%v) exit=%d stderr=%q", args, code, stderr.String())
			continue
		}
		if stdout.String() != canonicalOut.String() || stderr.String() != canonicalErr.String() {
			t.Errorf("Run(%v)=%q/%q canonical=%q/%q", args, stdout.String(), stderr.String(), canonicalOut.String(), canonicalErr.String())
		}
		if factoryCalls != 0 {
			t.Errorf("Run(%v) initialized Chatwork %d times", args, factoryCalls)
		}
	}
}

func TestStaticCommandAliasDoesNotBypassFirstRunSelection(t *testing.T) {
	for _, args := range [][]string{
		{"messages", "get", "marker"},
		{"help", "messages", "get"},
		{"help", "messages.get"},
		{"help", "messages", "show", "--help"},
		{"help", "messages", "get", "--help"},
		{"files", "list", "--room", "123", "--help"},
	} {
		h := newCommandSelectionHarness(t, strings.NewReader(""))
		factoryCalls := 0
		h.command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
			factoryCalls++
			return nil, nil, errors.New("selection rejection must precede Chatwork")
		}
		if code := runCLI(h.command, args); code != ExitRejected {
			t.Fatalf("alias %v exit=%d stderr=%q", args, code, h.stderr.String())
		}
		if h.stdout.Len() != 0 || !strings.Contains(h.stderr.String(), "code: command_selection_required") || factoryCalls != 0 {
			t.Fatalf("args=%v stdout=%q stderr=%q factory_calls=%d", args, h.stdout.String(), h.stderr.String(), factoryCalls)
		}
	}
}

type failingCommandSelectionStore struct {
	err error
}

func (s failingCommandSelectionStore) Load(context.Context) (commandselection.Profile, bool, error) {
	return commandselection.Profile{}, false, s.err
}

func (s failingCommandSelectionStore) Save(context.Context, commandselection.Profile) error {
	return errors.New("must not save while rendering help")
}

func TestStaticHelpUsesFullCatalogWhenCommandSelectionCannotBeLoaded(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{name: "invalid", err: fault.New(fault.KindInvalidInput, "command_selection_invalid", "invalid", false), code: "command_selection_invalid"},
		{name: "unsafe", err: fault.New(fault.KindUnavailable, "command_selection_unsafe", "unsafe", false), code: "command_selection_unsafe"},
		{name: "unavailable", err: fault.New(fault.KindUnavailable, "command_selection_unavailable", "unavailable", true), code: "command_selection_unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			command := newCLI(strings.NewReader(""), &stdout, &stderr, DefaultCatalog(), passingInspector("unused"))
			command.commandSelection = configcmd.New(failingCommandSelectionStore{err: test.err})
			factoryCalls := 0
			command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
				factoryCalls++
				return nil, nil, errors.New("must not initialize Chatwork")
			}

			helpCases := []struct {
				args       []string
				want       string
				wantNotice bool
			}{
				{args: []string{"--help"}, want: "Chatwork CLI\n", wantNotice: true},
				{args: []string{"help", "messages"}, want: "名前空間: messages", wantNotice: true},
				{args: []string{"help", "messages.list"}, want: "cwk messages list"},
				{args: []string{"files", "list", "--room", "123", "--help"}, want: "cwk files list"},
			}
			for _, helpCase := range helpCases {
				stdout.Reset()
				stderr.Reset()
				if code := runCLI(command, helpCase.args); code != ExitOK {
					t.Fatalf("help %v exit=%d stderr=%q", helpCase.args, code, stderr.String())
				}
				if !strings.Contains(stdout.String(), helpCase.want) || stderr.Len() != 0 {
					t.Errorf("help %v stdout=%q stderr=%q", helpCase.args, stdout.String(), stderr.String())
				}
				hasNotice := strings.Contains(stdout.String(), "静的カタログ") && strings.Contains(stdout.String(), "cwk doctor")
				if hasNotice != helpCase.wantNotice {
					t.Errorf("help %v static notice=%t want=%t output=%q", helpCase.args, hasNotice, helpCase.wantNotice, stdout.String())
				}
			}

			for _, ordinary := range [][]string{{"rooms", "list"}, {"messages", "get", "marker"}} {
				stdout.Reset()
				stderr.Reset()
				if code := runCLI(command, ordinary); code == ExitOK {
					t.Fatalf("ordinary configurable task %v succeeded with broken selection state", ordinary)
				}
				if stdout.Len() != 0 || !strings.Contains(stderr.String(), "code: "+test.code) {
					t.Errorf("ordinary task %v stdout=%q stderr=%q", ordinary, stdout.String(), stderr.String())
				}
			}
			if factoryCalls != 0 {
				t.Fatalf("help or rejected task initialized Chatwork %d times", factoryCalls)
			}
		})
	}
}

func TestStaticHelpFallbackDoesNotIgnoreCancellation(t *testing.T) {
	if commandSelectionHelpFallbackAllowed(fault.New(fault.KindCanceled, "operation_canceled", "canceled", true)) {
		t.Fatal("canceled selection load was admitted as a static-help fallback")
	}
	if commandSelectionHelpFallbackAllowed(fault.New(fault.KindInternal, "command_selection_invalid", "wrong kind", false)) {
		t.Fatal("wrong-kind selection fault was admitted as a static-help fallback")
	}
}
