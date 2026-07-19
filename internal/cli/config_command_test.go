package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	appauthn "github.com/tasuku43/cwk/internal/app/authn"
	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/app/configcmd"
	"github.com/tasuku43/cwk/internal/domain/commandselection"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/infra/commandconfig"
)

type commandSelectionHarness struct {
	base    string
	store   *commandconfig.FileStore
	command *CLI
	stdout  *bytes.Buffer
	stderr  *bytes.Buffer
}

func newCommandSelectionHarness(t *testing.T, input io.Reader) *commandSelectionHarness {
	t.Helper()
	base := t.TempDir()
	store := commandconfig.NewFileStoreAt(base)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := newCLI(input, stdout, stderr, DefaultCatalog(), passingInspector("unused"))
	command.commandSelection = configcmd.New(store)
	return &commandSelectionHarness{
		base: base, store: store, command: command, stdout: stdout, stderr: stderr,
	}
}

func (h *commandSelectionHarness) reset(input io.Reader) {
	h.command.In = input
	h.stdout.Reset()
	h.stderr.Reset()
}

func saveCommandSelection(t *testing.T, store *commandconfig.FileStore, paths []string) {
	t.Helper()
	profile, err := commandselection.New(paths)
	if err != nil {
		t.Fatalf("commandselection.New(%v): %v", paths, err)
	}
	if err := store.Save(context.Background(), profile); err != nil {
		t.Fatalf("store.Save(%v): %v", paths, err)
	}
}

func loadCommandSelection(t *testing.T, store *commandconfig.FileStore) []string {
	t.Helper()
	profile, configured, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("store.Load(): %v", err)
	}
	if !configured {
		t.Fatal("store.Load() reports an absent profile")
	}
	return profile.EnabledCommands()
}

func configurableCommandPaths(catalog Catalog) []string {
	commands := catalog.ConfigurableCommands()
	paths := make([]string, len(commands))
	for index, command := range commands {
		paths[index] = command.Path
	}
	return paths
}

func commandNumber(t *testing.T, catalog Catalog, path string) int {
	t.Helper()
	for index, command := range catalog.ConfigurableCommands() {
		if command.Path == path {
			return index + 1
		}
	}
	t.Fatalf("configurable command %q is missing", path)
	return 0
}

func TestConfigShowDefaultIsCatalogOrderedAndExplicitlyNotSecurity(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader(""))
	if code := runCLI(h.command, []string{"config", "show"}); code != ExitOK {
		t.Fatalf("config show exit = %d, stderr = %q", code, h.stderr.String())
	}

	var want strings.Builder
	want.WriteString("config purpose=attention-only security-boundary=false source=default\n")
	always := DefaultCatalog().AlwaysCommands()
	fmt.Fprintf(&want, "always-on count=%d\n", len(always))
	for _, command := range always {
		fmt.Fprintf(&want, "  %s\n", command.Path)
	}
	configurable := DefaultCatalog().ConfigurableCommands()
	fmt.Fprintf(&want, "enabled count=%d\n", len(configurable))
	for _, command := range configurable {
		fmt.Fprintf(&want, "  %s\n", command.Path)
	}
	want.WriteString("disabled count=0\n")
	want.WriteString("stale count=0\n")
	if got := h.stdout.String(); got != want.String() {
		t.Fatalf("config show output =\n%s\nwant =\n%s", got, want.String())
	}
	if h.stderr.Len() != 0 {
		t.Fatalf("config show stderr = %q", h.stderr.String())
	}
}

func TestConfigShowSeparatesSavedEnabledDisabledAndStalePaths(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader(""))
	saveCommandSelection(t, h.store, []string{"doctor", "retired command"})
	if code := runCLI(h.command, []string{"config", "show"}); code != ExitOK {
		t.Fatalf("config show exit = %d, stderr = %q", code, h.stderr.String())
	}
	var want strings.Builder
	want.WriteString("config purpose=attention-only security-boundary=false source=saved\n")
	always := DefaultCatalog().AlwaysCommands()
	fmt.Fprintf(&want, "always-on count=%d\n", len(always))
	for _, command := range always {
		fmt.Fprintf(&want, "  %s\n", command.Path)
	}
	want.WriteString("enabled count=1\n  doctor\n")
	configurable := DefaultCatalog().ConfigurableCommands()
	fmt.Fprintf(&want, "disabled count=%d\n", len(configurable)-1)
	for _, command := range configurable {
		if command.Path != "doctor" {
			fmt.Fprintf(&want, "  %s\n", command.Path)
		}
	}
	want.WriteString("stale count=1\n  retired command\n")
	if got := h.stdout.String(); got != want.String() {
		t.Fatalf("saved config show output =\n%s\nwant =\n%s", got, want.String())
	}
}

func TestConfigEditMenuUsesStableCatalogNumbersAndAtomicToggles(t *testing.T) {
	catalog := DefaultCatalog()
	doctorNumber := commandNumber(t, catalog, "doctor")
	versionNumber := commandNumber(t, catalog, "version")
	input := fmt.Sprintf("%d,bad,%d\n%d %d\nsave\n", doctorNumber, versionNumber, doctorNumber, versionNumber)
	h := newCommandSelectionHarness(t, strings.NewReader(input))
	if code := runCLI(h.command, []string{"config", "edit"}); code != ExitOK {
		t.Fatalf("config edit exit = %d, stdout = %q, stderr = %q", code, h.stdout.String(), h.stderr.String())
	}
	output := h.stdout.String()
	if !strings.HasPrefix(output, "Command selection (attention only; not an authorization or security boundary)\n") {
		t.Fatalf("config edit omitted its non-security framing:\n%s", output)
	}
	for _, want := range []string{
		"Always enabled:\n  help\n  config show\n  config edit\n",
		"Enter numbers to toggle, or one of: all, none, save, cancel\nconfig> ",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("config edit menu lacks %q:\n%s", want, output)
		}
	}
	last := -1
	for index, command := range catalog.ConfigurableCommands() {
		line := fmt.Sprintf("%d [x] %s", index+1, command.Path)
		offset := strings.Index(output, line)
		if offset <= last {
			t.Errorf("initial menu line %q is absent or out of catalog order:\n%s", line, output)
		}
		last = offset
	}
	if !strings.Contains(output, "invalid selection") {
		t.Errorf("invalid mixed line was not diagnosed:\n%s", output)
	}

	paths := loadCommandSelection(t, h.store)
	if containsString(paths, "doctor") || containsString(paths, "version") {
		t.Fatalf("valid toggle did not disable doctor/version: %v", paths)
	}
	if got, want := len(paths), len(catalog.ConfigurableCommands())-2; got != want {
		t.Fatalf("saved enabled count = %d, want %d (%v)", got, want, paths)
	}
	if !strings.Contains(output, fmt.Sprintf("config saved enabled=%d disabled=2 changed=2 stale-removed=0", len(paths))) {
		t.Errorf("save summary lacks exact counts:\n%s", output)
	}
}

func TestConfigEditAllNoneAndStaleRemovalAreExplicit(t *testing.T) {
	tests := []struct {
		name        string
		initial     []string
		input       string
		wantEnabled []string
		wantStale   int
		wantChanged int
	}{
		{name: "none", initial: []string{"doctor"}, input: "none\nsave\n", wantEnabled: []string{}, wantStale: 0, wantChanged: 1},
		{name: "all removes stale", initial: []string{"doctor", "retired command"}, input: "all\nsave\n", wantEnabled: configurableCommandPaths(DefaultCatalog()), wantStale: 1, wantChanged: len(DefaultCatalog().ConfigurableCommands()) - 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newCommandSelectionHarness(t, strings.NewReader(test.input))
			saveCommandSelection(t, h.store, test.initial)
			if code := runCLI(h.command, []string{"config", "edit"}); code != ExitOK {
				t.Fatalf("config edit exit = %d, stdout = %q, stderr = %q", code, h.stdout.String(), h.stderr.String())
			}
			if got := loadCommandSelection(t, h.store); !equalCommandPaths(got, test.wantEnabled) {
				t.Fatalf("saved paths = %v, want %v", got, test.wantEnabled)
			}
			if !strings.Contains(h.stdout.String(), fmt.Sprintf("stale-removed=%d", test.wantStale)) {
				t.Errorf("save summary lacks stale-removed=%d:\n%s", test.wantStale, h.stdout.String())
			}
			if !strings.Contains(h.stdout.String(), fmt.Sprintf("changed=%d", test.wantChanged)) {
				t.Errorf("save summary lacks changed=%d:\n%s", test.wantChanged, h.stdout.String())
			}
		})
	}
}

func TestConfigEditCancelAndEOFPreservePreviousProfile(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
	}{
		{name: "cancel", input: "none\ncancel\n"},
		{name: "EOF", input: "none\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := newCommandSelectionHarness(t, strings.NewReader(test.input))
			before := []string{"doctor"}
			saveCommandSelection(t, h.store, before)
			if code := runCLI(h.command, []string{"config", "edit"}); code != ExitCanceled {
				t.Fatalf("config edit exit = %d, stdout = %q, stderr = %q", code, h.stdout.String(), h.stderr.String())
			}
			if got := loadCommandSelection(t, h.store); !reflect.DeepEqual(got, before) {
				t.Fatalf("profile after %s = %v, want unchanged %v", test.name, got, before)
			}
			if !strings.Contains(h.stderr.String(), "code: configuration_canceled") {
				t.Errorf("%s stderr lacks stable cancellation: %q", test.name, h.stderr.String())
			}
		})
	}
}

func TestConfigEditJSONCancellationKeepsStructuredErrorStreamPure(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader("cancel\n"))
	if code := runCLI(h.command, []string{"--error-format=json", "config", "edit"}); code != ExitCanceled {
		t.Fatalf("config edit JSON cancel exit = %d, stderr = %q", code, h.stderr.String())
	}
	if !strings.HasPrefix(h.stdout.String(), "Command selection (attention only;") {
		t.Fatalf("selector transcript is missing from stdout: %q", h.stdout.String())
	}
	var document errorDocument
	if err := json.Unmarshal(h.stderr.Bytes(), &document); err != nil {
		t.Fatalf("stderr is not one pure JSON fault: %v\n%s", err, h.stderr.String())
	}
	if document.Error.Code != "configuration_canceled" || document.Error.Kind != fault.KindCanceled {
		t.Fatalf("JSON cancellation = %+v", document.Error)
	}
}

func TestConfigEditRejectsOversizedInputWithoutWriting(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader(strings.Repeat("1", maxConfigSelectionLineBytes+1)+"\n"))
	before := []string{"doctor"}
	saveCommandSelection(t, h.store, before)
	if code := runCLI(h.command, []string{"config", "edit"}); code != ExitInternal {
		t.Fatalf("oversized config input exit = %d, stderr = %q", code, h.stderr.String())
	}
	if !strings.Contains(h.stderr.String(), "code: configuration_input_failed") {
		t.Fatalf("oversized input fault = %q", h.stderr.String())
	}
	if got := loadCommandSelection(t, h.store); !reflect.DeepEqual(got, before) {
		t.Fatalf("oversized input changed profile: got %v want %v", got, before)
	}
}

type blockingCommandConfigReader struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func newBlockingCommandConfigReader() *blockingCommandConfigReader {
	return &blockingCommandConfigReader{started: make(chan struct{}), release: make(chan struct{})}
}

func (r *blockingCommandConfigReader) Read([]byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	<-r.release
	return 0, io.EOF
}

func TestConfigEditContextCancellationInterruptsBlockedInputWithoutWriting(t *testing.T) {
	reader := newBlockingCommandConfigReader()
	h := newCommandSelectionHarness(t, reader)
	before := []string{"doctor"}
	saveCommandSelection(t, h.store, before)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan int, 1)
	go func() { done <- h.command.RunContext(ctx, []string{"config", "edit"}) }()
	select {
	case <-reader.started:
	case <-time.After(2 * time.Second):
		close(reader.release)
		t.Fatal("config edit did not begin reading stdin")
	}
	cancel()
	select {
	case code := <-done:
		if code != ExitCanceled {
			t.Errorf("canceled config edit exit = %d, stderr = %q", code, h.stderr.String())
		}
	case <-time.After(2 * time.Second):
		close(reader.release)
		t.Fatal("config edit did not honor context cancellation while stdin was blocked")
	}
	close(reader.release)
	if got := loadCommandSelection(t, h.store); !reflect.DeepEqual(got, before) {
		t.Fatalf("profile after context cancellation = %v, want unchanged %v", got, before)
	}
}

func TestMalformedProfileFailsClosedAndConfigEditRepairsOnlyOnSave(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader(""))
	writeMalformedCommandSelection(t, h.base, []byte("{\n"))
	if code := runCLI(h.command, []string{"config", "show"}); code != ExitUsage {
		t.Fatalf("config show malformed exit = %d, stderr = %q", code, h.stderr.String())
	}
	if !strings.Contains(h.stderr.String(), "code: command_selection_invalid") {
		t.Fatalf("config show malformed stderr = %q", h.stderr.String())
	}

	// A normal task fails on the preference before lazy PAT construction.
	factoryCalls := 0
	h.command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
		factoryCalls++
		return nil, nil, errors.New("must not resolve authentication")
	}
	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"rooms", "list"}); code != ExitUsage {
		t.Fatalf("normal command with malformed profile exit = %d, stderr = %q", code, h.stderr.String())
	}
	if factoryCalls != 0 || !strings.Contains(h.stderr.String(), "code: command_selection_invalid") {
		t.Fatalf("malformed profile reached PAT or changed fault: calls=%d stderr=%q", factoryCalls, h.stderr.String())
	}

	// Root help must not silently masquerade as a deliberately empty view.
	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"help"}); code != ExitUsage {
		t.Fatalf("help with malformed profile exit = %d, stderr = %q", code, h.stderr.String())
	}
	if !strings.Contains(h.stderr.String(), "code: command_selection_invalid") || !strings.Contains(h.stderr.String(), "cwk config edit") {
		t.Fatalf("malformed-state root help did not expose the repair fault:\n%s", h.stderr.String())
	}

	// Config-scoped help remains available without presenting a false normal
	// command view.
	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"config", "--help"}); code != ExitOK {
		t.Fatalf("config help with malformed profile exit = %d, stderr = %q", code, h.stderr.String())
	}
	if !strings.Contains(h.stdout.String(), "  show  Show") || !strings.Contains(h.stdout.String(), "  edit  Select") || strings.Contains(h.stdout.String(), "rooms") {
		t.Fatalf("malformed-state config help did not isolate the repair commands:\n%s", h.stdout.String())
	}

	// Merely entering and canceling the repair path must retain the bad bytes.
	h.reset(strings.NewReader("cancel\n"))
	if code := runCLI(h.command, []string{"config", "edit"}); code != ExitCanceled {
		t.Fatalf("config edit cancel malformed exit = %d, stderr = %q", code, h.stderr.String())
	}
	contents, err := os.ReadFile(filepath.Join(h.base, "cwk", "command-selection.json"))
	if err != nil || string(contents) != "{\n" {
		t.Fatalf("cancel changed malformed profile: contents=%q err=%v", contents, err)
	}

	h.reset(strings.NewReader("save\n"))
	if code := runCLI(h.command, []string{"config", "edit"}); code != ExitOK {
		t.Fatalf("config edit repair exit = %d, stdout = %q, stderr = %q", code, h.stdout.String(), h.stderr.String())
	}
	if got, want := loadCommandSelection(t, h.store), configurableCommandPaths(DefaultCatalog()); !reflect.DeepEqual(got, want) {
		t.Fatalf("repaired profile = %v, want all-enabled baseline %v", got, want)
	}
}

func TestUnsafeProfileCannotEnterAnUnrepairableEditLoop(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic-link creation requires platform-specific privileges on Windows")
	}
	h := newCommandSelectionHarness(t, strings.NewReader("save\n"))
	app := filepath.Join(h.base, "cwk")
	if err := os.Mkdir(app, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(h.base, "target.json")
	if err := os.WriteFile(target, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(app, "command-selection.json")); err != nil {
		t.Fatal(err)
	}

	if code := runCLI(h.command, []string{"help"}); code != ExitUnavailable {
		t.Fatalf("unsafe root help exit = %d, stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}
	if h.stdout.Len() != 0 || !strings.Contains(h.stderr.String(), "code: command_selection_unsafe") || !strings.Contains(h.stderr.String(), "cwk config show") {
		t.Fatalf("unsafe root help presented a false command view: stdout=%q stderr=%q", h.stdout.String(), h.stderr.String())
	}
	h.reset(strings.NewReader("save\n"))
	if code := runCLI(h.command, []string{"config", "edit"}); code != ExitUnavailable {
		t.Fatalf("unsafe config edit exit = %d, stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}
	if h.stdout.Len() != 0 || !strings.Contains(h.stderr.String(), "code: command_selection_unsafe") || !strings.Contains(h.stderr.String(), "cwk config show") {
		t.Fatalf("unsafe storage entered repair selector or lacked external-repair guidance: stdout=%q stderr=%q", h.stdout.String(), h.stderr.String())
	}
	contents, err := os.ReadFile(target)
	if err != nil || string(contents) != "{}\n" {
		t.Fatalf("unsafe target changed: contents=%q err=%v", contents, err)
	}
}

type unavailableCommandSelectionStore struct{}

func (unavailableCommandSelectionStore) Load(context.Context) (commandselection.Profile, bool, error) {
	return commandselection.Profile{}, false, fault.New(
		fault.KindUnavailable,
		"command_selection_unavailable",
		"command selection is unavailable",
		true,
	)
}

func (unavailableCommandSelectionStore) Save(context.Context, commandselection.Profile) error {
	return errors.New("must not save unavailable configuration")
}

func TestUnavailableProfileRoutesToInspectionAfterExternalRepair(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader("save\n"), &stdout, &stderr, DefaultCatalog(), passingInspector("unused"))
	command.commandSelection = configcmd.New(unavailableCommandSelectionStore{})

	for _, args := range [][]string{{"help"}, {"config", "edit"}} {
		stdout.Reset()
		stderr.Reset()
		if code := command.RunContext(context.Background(), args); code != ExitUnavailable {
			t.Fatalf("unavailable invocation %v exit=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
		if stdout.Len() != 0 || !strings.Contains(stderr.String(), "code: command_selection_unavailable") ||
			!strings.Contains(stderr.String(), "cwk config show") || strings.Contains(stderr.String(), "cwk config edit") {
			t.Fatalf("unavailable invocation %v has unusable recovery: stdout=%q stderr=%q", args, stdout.String(), stderr.String())
		}
	}
}

type cancelAfterConfigSaveStore struct {
	cancel context.CancelFunc
	saved  []string
	saves  int
}

func (s *cancelAfterConfigSaveStore) Load(context.Context) (commandselection.Profile, bool, error) {
	return commandselection.Profile{}, false, nil
}

func (s *cancelAfterConfigSaveStore) Save(_ context.Context, profile commandselection.Profile) error {
	s.saves++
	s.saved = profile.EnabledCommands()
	s.cancel()
	return nil
}

func TestConfigEditDoesNotOverwriteConfirmedSaveWithLateCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store := &cancelAfterConfigSaveStore{cancel: cancel}
	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader("none\nsave\n"), &stdout, &stderr, DefaultCatalog(), passingInspector("unused"))
	command.commandSelection = configcmd.New(store)

	if code := command.RunContext(ctx, []string{"config", "edit"}); code != ExitOK {
		t.Fatalf("late-canceled save exit = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if ctx.Err() == nil || store.saves != 1 || len(store.saved) != 0 || !strings.Contains(stdout.String(), "config saved enabled=0") || stderr.Len() != 0 {
		t.Fatalf("confirmed save was not reported exactly once: canceled=%v saves=%d saved=%v stdout=%q stderr=%q", ctx.Err(), store.saves, store.saved, stdout.String(), stderr.String())
	}
}

func TestActiveCommandViewHidesDisabledPathsFromEveryDiscoveryAndRoute(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader(""))
	enabled := make([]string, 0)
	for _, path := range configurableCommandPaths(DefaultCatalog()) {
		if !strings.HasPrefix(path, "contact-requests ") {
			enabled = append(enabled, path)
		}
	}
	saveCommandSelection(t, h.store, enabled)

	for _, args := range [][]string{
		{"contact-requests", "list"},
		{"contact-requests", "list", "--help"},
		{"contact-requests", "--help"},
		{"help", "contact-requests"},
		{"help", "contact-requests", "list", "--format", "agent"},
	} {
		h.reset(strings.NewReader(""))
		if code := runCLI(h.command, args); code == ExitOK {
			t.Fatalf("hidden invocation %v unexpectedly succeeded: stdout=%q", args, h.stdout.String())
		}
		if strings.Contains(h.stdout.String(), "contact-requests") {
			t.Fatalf("hidden invocation %v leaked its path on stdout: %q", args, h.stdout.String())
		}
	}

	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"help"}); code != ExitOK {
		t.Fatalf("root help exit = %d, stderr = %q", code, h.stderr.String())
	}
	if strings.Contains(h.stdout.String(), "contact-requests") {
		t.Fatalf("root help leaked disabled namespace:\n%s", h.stdout.String())
	}

	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"help", "--format", "agent"}); code != ExitOK {
		t.Fatalf("root agent help exit = %d, stderr = %q", code, h.stderr.String())
	}
	var index agentIndexDocument
	if err := json.Unmarshal(h.stdout.Bytes(), &index); err != nil {
		t.Fatalf("root agent help JSON: %v\n%s", err, h.stdout.String())
	}
	for _, command := range index.Commands {
		if strings.HasPrefix(command.Path, "contact-requests ") {
			t.Fatalf("root agent help leaked disabled command %+v", command)
		}
	}

	// Scan every remaining scoped agent document: recovery actions and
	// reference workflows must be projected from the same active catalog.
	namespaces := make(map[string]struct{})
	for _, command := range h.command.catalog.Commands() {
		namespaces[commandNamespace(command.Path)] = struct{}{}
	}
	ordered := make([]string, 0, len(namespaces))
	for namespace := range namespaces {
		ordered = append(ordered, namespace)
	}
	sort.Strings(ordered)
	for _, namespace := range ordered {
		h.reset(strings.NewReader(""))
		args := []string{"help", namespace, "--format", "agent"}
		if code := runCLI(h.command, args); code != ExitOK {
			t.Fatalf("scoped agent help %v exit = %d, stderr = %q", args, code, h.stderr.String())
		}
		if strings.Contains(h.stdout.String(), "contact-requests") {
			t.Fatalf("scoped agent help %q leaked disabled recovery/workflow:\n%s", namespace, h.stdout.String())
		}
	}
}

func TestDisabledRouteIsUnknownBeforePATResolutionAndSameCLIReenablesIt(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader(""))
	saveCommandSelection(t, h.store, []string{"doctor"})
	factoryCalls := 0
	h.command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
		factoryCalls++
		return nil, nil, fault.New(fault.KindAuthentication, "chatwork_token_missing", "A Chatwork API token is required.", false)
	}
	if code := runCLI(h.command, []string{"rooms", "list"}); code != ExitUsage {
		t.Fatalf("disabled rooms list exit = %d, stderr = %q", code, h.stderr.String())
	}
	if factoryCalls != 0 || !strings.Contains(h.stderr.String(), "code: unknown_command") {
		t.Fatalf("disabled route resolved PAT or changed taxonomy: calls=%d stderr=%q", factoryCalls, h.stderr.String())
	}

	h.reset(strings.NewReader("all\nsave\n"))
	if code := runCLI(h.command, []string{"config", "edit"}); code != ExitOK {
		t.Fatalf("config edit all exit = %d, stdout = %q, stderr = %q", code, h.stdout.String(), h.stderr.String())
	}
	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"rooms", "list"}); code != ExitAuthentication {
		t.Fatalf("re-enabled rooms list exit = %d, stderr = %q", code, h.stderr.String())
	}
	if factoryCalls != 1 || strings.Contains(h.stderr.String(), "code: unknown_command") {
		t.Fatalf("same CLI did not re-enable route under its original auth boundary: calls=%d stderr=%q", factoryCalls, h.stderr.String())
	}

	// Re-enabling an access-changing mutation restores routing only; its
	// invocation-local confirmation still rejects before authentication.
	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"contact-requests", "accept", "--request", "123"}); code != ExitRejected {
		t.Fatalf("re-enabled unconfirmed mutation exit = %d, stderr = %q", code, h.stderr.String())
	}
	if factoryCalls != 1 || !strings.Contains(h.stderr.String(), "code: mutation_rejected") {
		t.Fatalf("re-enable bypassed confirmation or reached PAT: calls=%d stderr=%q", factoryCalls, h.stderr.String())
	}
}

func TestConfigEditRejectsInvalidDependencyAndRecoveryViewsWithoutOverwriting(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{name: "missing producer", paths: []string{"messages mark-read"}, want: `enable one producer:`},
		{name: "hidden recovery", paths: []string{"rooms list", "messages send"}, want: "messages list"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog := DefaultCatalog()
			var numbers []string
			for _, path := range test.paths {
				numbers = append(numbers, fmt.Sprint(commandNumber(t, catalog, path)))
			}
			input := "none\n" + strings.Join(numbers, " ") + "\nsave\ncancel\n"
			h := newCommandSelectionHarness(t, strings.NewReader(input))
			before := configurableCommandPaths(catalog)
			saveCommandSelection(t, h.store, before)
			if code := runCLI(h.command, []string{"config", "edit"}); code != ExitCanceled {
				t.Fatalf("invalid view edit exit = %d, stdout = %q, stderr = %q", code, h.stdout.String(), h.stderr.String())
			}
			if !strings.Contains(h.stdout.String(), test.want) {
				t.Errorf("invalid view diagnostic lacks %q:\n%s", test.want, h.stdout.String())
			}
			if got := loadCommandSelection(t, h.store); !reflect.DeepEqual(got, before) {
				t.Fatalf("invalid view overwrote profile: got %v, want %v", got, before)
			}
		})
	}
}

func writeMalformedCommandSelection(t *testing.T, base string, contents []byte) {
	t.Helper()
	app := filepath.Join(base, "cwk")
	if err := os.Mkdir(app, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "command-selection.json"), contents, 0o600); err != nil {
		t.Fatal(err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func equalCommandPaths(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
