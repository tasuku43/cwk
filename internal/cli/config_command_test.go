package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	"github.com/tasuku43/cwk/internal/infra/terminalui"
)

type fakeConfigTerminalOpener struct {
	openErr    error
	width      int
	height     int
	sizes      []fakeConfigTerminalSize
	sizeErr    error
	readErr    error
	closeErr   error
	opens      int
	last       *fakeConfigTerminalSession
	beforeOpen func()
}

func (o *fakeConfigTerminalOpener) Open(input io.Reader, _ io.Writer) (terminalui.Session, error) {
	o.opens++
	if o.beforeOpen != nil {
		o.beforeOpen()
	}
	if o.openErr != nil {
		return nil, o.openErr
	}
	width, height := o.width, o.height
	if width == 0 {
		width = 120
	}
	if height == 0 {
		height = 12
	}
	o.last = &fakeConfigTerminalSession{
		reader: input, width: width, height: height, sizes: append([]fakeConfigTerminalSize(nil), o.sizes...),
		sizeErr: o.sizeErr, readErr: o.readErr, closeErr: o.closeErr,
	}
	return o.last, nil
}

type fakeConfigTerminalSize struct {
	width, height int
	err           error
}

type fakeConfigTerminalSession struct {
	reader        io.Reader
	width, height int
	sizes         []fakeConfigTerminalSize
	sizeCalls     int
	sizeErr       error
	readErr       error
	closeErr      error
	closed        bool
	closes        int
}

func (s *fakeConfigTerminalSession) Size() (int, int, error) {
	if len(s.sizes) != 0 {
		index := s.sizeCalls
		if index >= len(s.sizes) {
			index = len(s.sizes) - 1
		}
		s.sizeCalls++
		return s.sizes[index].width, s.sizes[index].height, s.sizes[index].err
	}
	return s.width, s.height, s.sizeErr
}

func (s *fakeConfigTerminalSession) Read(ctx context.Context, buffer []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if s.readErr != nil {
		return 0, s.readErr
	}
	if reader, ok := s.reader.(interface {
		ReadContext(context.Context, []byte) (int, error)
	}); ok {
		return reader.ReadContext(ctx, buffer)
	}
	return s.reader.Read(buffer)
}

func (s *fakeConfigTerminalSession) Close() error {
	s.closes++
	s.closed = true
	return s.closeErr
}

type commandSelectionHarness struct {
	base     string
	store    *commandconfig.FileStore
	command  *CLI
	terminal *fakeConfigTerminalOpener
	stdout   *bytes.Buffer
	stderr   *bytes.Buffer
}

func newCommandSelectionHarness(t *testing.T, input io.Reader) *commandSelectionHarness {
	t.Helper()
	base := t.TempDir()
	store := commandconfig.NewFileStoreAt(base)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := newCLI(input, stdout, stderr, DefaultCatalog(), passingInspector("unused"))
	terminal := &fakeConfigTerminalOpener{}
	command.terminal = terminal
	command.commandSelection = configcmd.New(store)
	return &commandSelectionHarness{
		base: base, store: store, command: command, terminal: terminal, stdout: stdout, stderr: stderr,
	}
}

func (h *commandSelectionHarness) reset(input io.Reader) {
	h.command.In = input
	h.stdout.Reset()
	h.stderr.Reset()
	h.terminal = &fakeConfigTerminalOpener{}
	h.command.terminal = h.terminal
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

func configKeysForSelection(catalog Catalog, initial, target []string) string {
	initialSet := stringSet(initial)
	targetSet := stringSet(target)
	var keys strings.Builder
	cursor := 0
	for index, command := range catalog.ConfigurableCommands() {
		if initialSet[command.Path] == targetSet[command.Path] {
			continue
		}
		for cursor < index {
			keys.WriteString("\x1b[B")
			cursor++
		}
		keys.WriteByte(' ')
	}
	keys.WriteByte('\r')
	return keys.String()
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func withoutPrefix(values []string, prefix string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !strings.HasPrefix(value, prefix) {
			result = append(result, value)
		}
	}
	return result
}

func TestConfigIsOneAlwaysOnInteractiveCommandAndLegacyPathsAreGone(t *testing.T) {
	catalog := DefaultCatalog()
	if _, found := catalog.Lookup("config"); !found {
		t.Fatal("exact config command is missing")
	}
	for _, old := range []string{"config show", "config edit"} {
		if _, found := catalog.Lookup(old); found {
			t.Fatalf("legacy command %q remains in catalog", old)
		}
	}
	wantAlways := []string{"doctor", "help", "version", "config"}
	if got := catalogPaths(NewCatalog(catalog.AlwaysCommands()...)); !reflect.DeepEqual(got, wantAlways) {
		t.Fatalf("always-on paths = %v, want %v", got, wantAlways)
	}
	for _, command := range catalog.ConfigurableCommands() {
		if command.chatwork == nil {
			t.Fatalf("non-Chatwork command %q is selectable", command.Path)
		}
	}

	h := newCommandSelectionHarness(t, strings.NewReader("\r"))
	h.command.terminal = &fakeConfigTerminalOpener{openErr: terminalui.ErrNotTerminal}
	if code := runCLI(h.command, []string{"config"}); code != ExitUnavailable {
		t.Fatalf("non-TTY config exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}
	if h.stdout.Len() != 0 || !strings.Contains(h.stderr.String(), "code: interactive_terminal_required") || !strings.Contains(h.stderr.String(), "cwk help config") {
		t.Fatalf("non-TTY fault = stdout=%q stderr=%q", h.stdout.String(), h.stderr.String())
	}
}

func TestProfileFailureRecoveryIslandDerivesFromCatalogMetadata(t *testing.T) {
	catalog := DefaultCatalog()
	for _, command := range catalog.Commands() {
		scopedHelp := append([]string{"help"}, strings.Fields(command.Path)...)
		if got := commandViewAlwaysInvocation(catalog, scopedHelp); got != !command.Configurable {
			t.Fatalf("scoped help for %q allowed=%v want=%v", command.Path, got, !command.Configurable)
		}
		if command.Path != "help" {
			if got := commandViewAlwaysInvocation(catalog, strings.Fields(command.Path)); got != !command.Configurable {
				t.Fatalf("direct %q allowed=%v want=%v", command.Path, got, !command.Configurable)
			}
		}
	}
	if commandViewAlwaysInvocation(catalog, []string{"help"}) {
		t.Fatal("bare root help must not present an always-only view after profile failure")
	}

	commands := catalog.Commands()
	for index := range commands {
		switch commands[index].Path {
		case "version":
			commands[index].Configurable = true
		case "rooms list":
			commands[index].Configurable = false
		}
	}
	mutated := NewCatalog(commands...)
	if commandViewAlwaysInvocation(mutated, []string{"version"}) {
		t.Fatal("retired always-on path remained hard-coded in the recovery island")
	}
	if !commandViewAlwaysInvocation(mutated, []string{"rooms", "list"}) ||
		!commandViewAlwaysInvocation(mutated, []string{"help", "rooms", "list"}) {
		t.Fatal("new catalog-declared always-on path was not admitted to direct and scoped recovery")
	}
}

func TestConfigTUITogglesInCatalogOrderShowsEffectsAndSavesAfterRestore(t *testing.T) {
	catalog := DefaultCatalog()
	initial := configurableCommandPaths(catalog)
	target := withoutPrefix(initial, "contact-requests ")
	h := newCommandSelectionHarness(t, strings.NewReader(configKeysForSelection(catalog, initial, target)))
	if code := runCLI(h.command, []string{"config"}); code != ExitOK {
		t.Fatalf("config exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}
	if got := loadCommandSelection(t, h.store); !reflect.DeepEqual(got, target) {
		t.Fatalf("saved paths=%v want=%v", got, target)
	}
	if h.terminal.last == nil || !h.terminal.last.closed || h.terminal.last.closes != 1 {
		t.Fatalf("terminal was not restored exactly once: %+v", h.terminal.last)
	}
	output := h.stdout.String()
	for _, want := range []string{"[read]", "[create]", "[write]", "config saved enabled=", "fingerprint=sha256:"} {
		if !strings.Contains(output, want) {
			t.Errorf("output lacks %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{"purpose=", "security-boundary=", "source=", "Selectable commands:", "config>"} {
		if strings.Contains(output, forbidden) {
			t.Errorf("legacy presentation %q remains:\n%s", forbidden, output)
		}
	}
}

func TestConfigTUIFullWidthSpaceTogglesCurrentItem(t *testing.T) {
	catalog := DefaultCatalog()
	want := configurableCommandPaths(catalog)
	want = append([]string(nil), want[1:]...)
	h := newCommandSelectionHarness(t, strings.NewReader("　\r"))
	if code := runCLI(h.command, []string{"config"}); code != ExitOK {
		t.Fatalf("config exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}
	if got := loadCommandSelection(t, h.store); !reflect.DeepEqual(got, want) {
		t.Fatalf("saved paths=%v want=%v", got, want)
	}
	if h.terminal.last == nil || !h.terminal.last.closed {
		t.Fatalf("terminal was not restored: %+v", h.terminal.last)
	}
}

func TestBatchedMoveToggleAndSaveRepaintsEachActionableSelection(t *testing.T) {
	catalog := DefaultCatalog()
	choices := catalog.ConfigurableCommands()
	if len(choices) < 2 {
		t.Fatal("selector needs at least two configurable commands")
	}

	// Height four leaves exactly one item row. strings.Reader returns this
	// complete Down, Space, Enter sequence in one terminal read, so the second
	// command cannot be reviewed unless dispatch repaints between decoded keys.
	h := newCommandSelectionHarness(t, strings.NewReader("\x1b[B \r"))
	h.terminal.height = 4
	if code := runCLI(h.command, []string{"config"}); code != ExitOK {
		t.Fatalf("config exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}

	want := configurableCommandPaths(catalog)
	want = append([]string(nil), want...)
	want = append(want[:1], want[2:]...)
	if got := loadCommandSelection(t, h.store); !reflect.DeepEqual(got, want) {
		t.Fatalf("saved paths=%v want=%v", got, want)
	}
	output := h.stdout.String()
	identity := choices[1].Path
	identityIndex := strings.Index(output, identity)
	savedIndex := strings.Index(output, "config saved")
	if identityIndex < 0 || savedIndex < 0 || identityIndex >= savedIndex {
		t.Fatalf("batched target %q was not displayed before save:\n%s", identity, output)
	}
}

func TestConfigQuitEOFAndInterruptPreserveTheLastSavedProfile(t *testing.T) {
	before := []string{"rooms list"}
	for _, test := range []struct {
		name     string
		input    string
		wantCode int
		wantText string
	}{
		{name: "q", input: "q", wantCode: ExitOK, wantText: "config unchanged"},
		{name: "escape", input: "\x1b", wantCode: ExitOK, wantText: "config unchanged"},
		{name: "EOF", input: "", wantCode: ExitOK, wantText: "config unchanged"},
		{name: "Ctrl-C", input: string([]byte{0x03}), wantCode: ExitCanceled, wantText: "code: operation_canceled"},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := newCommandSelectionHarness(t, strings.NewReader(test.input))
			saveCommandSelection(t, h.store, before)
			if code := runCLI(h.command, []string{"config"}); code != test.wantCode {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
			}
			if got := loadCommandSelection(t, h.store); !reflect.DeepEqual(got, before) {
				t.Fatalf("profile changed: got=%v want=%v", got, before)
			}
			if !strings.Contains(h.stdout.String()+h.stderr.String(), test.wantText) {
				t.Fatalf("result lacks %q: stdout=%q stderr=%q", test.wantText, h.stdout.String(), h.stderr.String())
			}
			if h.terminal.last == nil || !h.terminal.last.closed {
				t.Fatal("terminal was not restored")
			}
		})
	}
}

type dataAndEOFReader struct {
	data []byte
}

func (r *dataAndEOFReader) Read(buffer []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	count := copy(buffer, r.data)
	r.data = r.data[count:]
	return count, io.EOF
}

func TestConfigProcessesTerminalBytesReturnedWithEOF(t *testing.T) {
	h := newCommandSelectionHarness(t, &dataAndEOFReader{data: []byte{'\r'}})
	if code := runCLI(h.command, []string{"config"}); code != ExitOK {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}
	if _, configured, err := h.store.Load(context.Background()); err != nil || !configured {
		t.Fatalf("Enter returned with EOF was not saved: configured=%v err=%v", configured, err)
	}
	if !strings.Contains(h.stdout.String(), "config saved") {
		t.Fatalf("result=%q", h.stdout.String())
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

func (r *blockingCommandConfigReader) ReadContext(ctx context.Context, _ []byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-r.release:
		return 0, io.EOF
	}
}

func TestConfigContextCancellationRestoresTerminalAndWritesNothing(t *testing.T) {
	reader := newBlockingCommandConfigReader()
	h := newCommandSelectionHarness(t, reader)
	before := []string{"rooms list"}
	saveCommandSelection(t, h.store, before)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan int, 1)
	go func() { done <- h.command.RunContext(ctx, []string{"config"}) }()
	select {
	case <-reader.started:
	case <-time.After(2 * time.Second):
		close(reader.release)
		t.Fatal("config did not begin reading")
	}
	cancel()
	select {
	case code := <-done:
		if code != ExitCanceled {
			t.Errorf("exit=%d stderr=%q", code, h.stderr.String())
		}
	case <-time.After(2 * time.Second):
		close(reader.release)
		t.Fatal("config did not honor context cancellation")
	}
	close(reader.release)
	if got := loadCommandSelection(t, h.store); !reflect.DeepEqual(got, before) {
		t.Fatalf("profile changed: got=%v want=%v", got, before)
	}
	if h.terminal.last == nil || !h.terminal.last.closed {
		t.Fatal("terminal was not restored")
	}
}

func TestMalformedProfileFailsTasksButAlwaysOnHelpAndExplicitConfigRepairRemainAvailable(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader("q"))
	writeMalformedCommandSelection(t, h.base, []byte("{\n"))
	factoryCalls := 0
	h.command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
		factoryCalls++
		return nil, nil, errors.New("must not resolve authentication")
	}
	if code := runCLI(h.command, []string{"rooms", "list"}); code != ExitUsage {
		t.Fatalf("task exit=%d stderr=%q", code, h.stderr.String())
	}
	if factoryCalls != 0 || !strings.Contains(h.stderr.String(), "code: command_selection_invalid") {
		t.Fatalf("malformed profile reached PAT or changed fault: calls=%d stderr=%q", factoryCalls, h.stderr.String())
	}

	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"help"}); code != ExitUsage {
		t.Fatalf("root help under invalid profile exit=%d stderr=%q", code, h.stderr.String())
	}
	if h.stdout.Len() != 0 || !strings.Contains(h.stderr.String(), "code: command_selection_invalid") {
		t.Fatalf("root help masqueraded as a valid always-only view: stdout=%q stderr=%q", h.stdout.String(), h.stderr.String())
	}
	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"help", "config"}); code != ExitOK {
		t.Fatalf("config help under invalid profile exit=%d stderr=%q", code, h.stderr.String())
	}
	if !strings.Contains(h.stdout.String(), "cwk config") || strings.Contains(h.stdout.String(), "rooms") {
		t.Fatalf("config help under invalid profile is not isolated:\n%s", h.stdout.String())
	}
	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"version"}); code != ExitOK {
		t.Fatalf("version under invalid profile exit=%d stderr=%q", code, h.stderr.String())
	}
	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"doctor"}); code != ExitRejected {
		t.Fatalf("doctor under invalid profile exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}
	if !strings.Contains(h.stdout.String(), "command-selection\tfail\tstate=invalid") {
		t.Fatalf("doctor lacks invalid selection diagnostic:\n%s", h.stdout.String())
	}
	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"help", "rooms"}); code != ExitUsage {
		t.Fatalf("scoped task help under invalid profile exit=%d stderr=%q", code, h.stderr.String())
	}
	if !strings.Contains(h.stderr.String(), "code: command_selection_invalid") {
		t.Fatalf("scoped task help masqueraded as unknown: %q", h.stderr.String())
	}

	h.reset(strings.NewReader("q"))
	if code := runCLI(h.command, []string{"config"}); code != ExitOK {
		t.Fatalf("repair quit exit=%d stderr=%q", code, h.stderr.String())
	}
	contents, err := os.ReadFile(filepath.Join(h.base, "cwk", "command-selection.json"))
	if err != nil || string(contents) != "{\n" {
		t.Fatalf("quit changed malformed profile: contents=%q err=%v", contents, err)
	}

	h.reset(strings.NewReader("\r"))
	if code := runCLI(h.command, []string{"config"}); code != ExitOK {
		t.Fatalf("repair save exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}
	if got, want := loadCommandSelection(t, h.store), configurableCommandPaths(DefaultCatalog()); !reflect.DeepEqual(got, want) {
		t.Fatalf("repaired profile=%v want=%v", got, want)
	}
}

func TestUnsafeProfileKeepsAlwaysOnDiagnosticsAvailableAndRefusesSelector(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic-link creation requires platform-specific privileges on Windows")
	}
	h := newCommandSelectionHarness(t, strings.NewReader("\r"))
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
		t.Fatalf("unsafe root help exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}
	if h.stdout.Len() != 0 || !strings.Contains(h.stderr.String(), "code: command_selection_unsafe") {
		t.Fatalf("unsafe root help masqueraded as valid: stdout=%q stderr=%q", h.stdout.String(), h.stderr.String())
	}
	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"help", "config"}); code != ExitOK {
		t.Fatalf("unsafe scoped config help exit=%d stderr=%q", code, h.stderr.String())
	}
	h.reset(strings.NewReader("\r"))
	if code := runCLI(h.command, []string{"config"}); code != ExitUnavailable {
		t.Fatalf("unsafe config exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}
	if h.terminal.opens != 0 || !strings.Contains(h.stderr.String(), "code: command_selection_unsafe") || !strings.Contains(h.stderr.String(), "cwk doctor") {
		t.Fatalf("unsafe storage entered selector: opens=%d stderr=%q", h.terminal.opens, h.stderr.String())
	}
	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"doctor"}); code != ExitRejected {
		t.Fatalf("unsafe doctor exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}
	if !strings.Contains(h.stdout.String(), "command-selection\tfail\tstate=unsafe") {
		t.Fatalf("doctor lacks safe storage state:\n%s", h.stdout.String())
	}
	contents, err := os.ReadFile(target)
	if err != nil || string(contents) != "{}\n" {
		t.Fatalf("unsafe target changed: contents=%q err=%v", contents, err)
	}
}

type unavailableCommandSelectionStore struct{}

func (unavailableCommandSelectionStore) Load(context.Context) (commandselection.Profile, bool, error) {
	return commandselection.Profile{}, false, fault.New(fault.KindUnavailable, "command_selection_unavailable", "command selection is unavailable", true)
}

func (unavailableCommandSelectionStore) Save(context.Context, commandselection.Profile) error {
	return errors.New("must not save unavailable configuration")
}

func TestUnavailableProfileRoutesConfigToDoctorAndDoctorRemainsReadOnly(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader("\r"), &stdout, &stderr, DefaultCatalog(), passingInspector("runtime"))
	command.commandSelection = configcmd.New(unavailableCommandSelectionStore{})
	command.terminal = &fakeConfigTerminalOpener{}

	if code := command.RunContext(context.Background(), []string{"config"}); code != ExitUnavailable {
		t.Fatalf("config exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "cwk doctor") {
		t.Fatalf("config unavailable recovery=%q", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := command.RunContext(context.Background(), []string{"doctor"}); code != ExitRejected {
		t.Fatalf("doctor exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "state=unavailable") {
		t.Fatalf("doctor lacks unavailable diagnostic:\n%s", stdout.String())
	}
}

type canceledCommandSelectionStore struct{}

func (canceledCommandSelectionStore) Load(context.Context) (commandselection.Profile, bool, error) {
	return commandselection.Profile{}, false, fault.New(fault.KindCanceled, "operation_canceled", "command selection operation was canceled", true)
}

func (canceledCommandSelectionStore) Save(context.Context, commandselection.Profile) error {
	return errors.New("must not save canceled configuration")
}

func TestCanceledProfileLoadRetainsNormalRetryRecovery(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader(""), &stdout, &stderr, DefaultCatalog(), passingInspector("unused"))
	command.commandSelection = configcmd.New(canceledCommandSelectionStore{})
	if code := command.RunContext(context.Background(), []string{"rooms", "list"}); code != ExitCanceled {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "code: operation_canceled") || !strings.Contains(stderr.String(), "cwk help") {
		t.Fatalf("canceled load recovery: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

type cancelAfterConfigSaveStore struct {
	cancel        context.CancelFunc
	saved         []string
	saves         int
	sessionClosed func() bool
}

func (s *cancelAfterConfigSaveStore) Load(context.Context) (commandselection.Profile, bool, error) {
	return commandselection.Profile{}, false, nil
}

func (s *cancelAfterConfigSaveStore) Save(_ context.Context, profile commandselection.Profile) error {
	if s.sessionClosed != nil && !s.sessionClosed() {
		return fault.New(fault.KindContract, "save_before_terminal_restore", "terminal must be restored before save", false)
	}
	s.saves++
	s.saved = profile.EnabledCommands()
	s.cancel()
	return nil
}

func TestConfigRestoresBeforeSaveAndDoesNotOverwriteSuccessWithLateCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var stdout, stderr bytes.Buffer
	terminal := &fakeConfigTerminalOpener{}
	store := &cancelAfterConfigSaveStore{cancel: cancel, sessionClosed: func() bool { return terminal.last != nil && terminal.last.closed }}
	command := newCLI(strings.NewReader("\r"), &stdout, &stderr, DefaultCatalog(), passingInspector("unused"))
	command.commandSelection = configcmd.New(store)
	command.terminal = terminal
	if code := command.RunContext(ctx, []string{"config"}); code != ExitOK {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if ctx.Err() == nil || store.saves != 1 || len(store.saved) != len(DefaultCatalog().ConfigurableCommands()) || !strings.Contains(stdout.String(), "config saved enabled=") || stderr.Len() != 0 {
		t.Fatalf("save result: canceled=%v saves=%d saved=%d stdout=%q stderr=%q", ctx.Err(), store.saves, len(store.saved), stdout.String(), stderr.String())
	}
}

func TestTerminalRestoreFailurePreventsSave(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader("\r"))
	h.terminal.closeErr = errors.New("restore failed")
	if code := runCLI(h.command, []string{"config"}); code != ExitInternal {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}
	if !strings.Contains(h.stderr.String(), "code: terminal_restore_failed") {
		t.Fatalf("restore fault=%q", h.stderr.String())
	}
	if _, configured, err := h.store.Load(context.Background()); err != nil || configured {
		t.Fatalf("restore failure saved profile: configured=%v err=%v", configured, err)
	}
}

func TestTerminalSetupRollbackFailureUsesNonRetryableRestoreRecovery(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader(""))
	h.command.terminal = &fakeConfigTerminalOpener{openErr: errors.Join(errors.New("setup failed"), terminalui.ErrRestoreFailed)}
	if code := runCLI(h.command, []string{"config"}); code != ExitInternal {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}
	if !strings.Contains(h.stderr.String(), "code: terminal_restore_failed") || !strings.Contains(h.stderr.String(), "cwk doctor") {
		t.Fatalf("setup rollback fault=%q", h.stderr.String())
	}
	if strings.Contains(h.stderr.String(), "cwk config\n") {
		t.Fatalf("uncertain terminal state was presented as directly retryable: %q", h.stderr.String())
	}
	if _, configured, err := h.store.Load(context.Background()); err != nil || configured {
		t.Fatalf("setup rollback failure saved profile: configured=%v err=%v", configured, err)
	}
}

func TestTerminalSizeAndInputFailuresRestoreAndSaveNothing(t *testing.T) {
	for _, test := range []struct {
		name     string
		prepare  func(*fakeConfigTerminalOpener)
		wantCode string
	}{
		{name: "size", prepare: func(terminal *fakeConfigTerminalOpener) { terminal.sizeErr = errors.New("size failed") }, wantCode: "terminal_setup_failed"},
		{name: "input", prepare: func(terminal *fakeConfigTerminalOpener) { terminal.readErr = errors.New("read failed") }, wantCode: "configuration_input_failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := newCommandSelectionHarness(t, strings.NewReader("\r"))
			test.prepare(h.terminal)
			if code := runCLI(h.command, []string{"config"}); code != ExitInternal {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
			}
			if !strings.Contains(h.stderr.String(), "code: "+test.wantCode) {
				t.Fatalf("fault lacks %s: %q", test.wantCode, h.stderr.String())
			}
			if h.terminal.last == nil || !h.terminal.last.closed || h.terminal.last.closes != 1 {
				t.Fatalf("terminal was not restored exactly once: %+v", h.terminal.last)
			}
			if _, configured, err := h.store.Load(context.Background()); err != nil || configured {
				t.Fatalf("failure saved profile: configured=%v err=%v", configured, err)
			}
		})
	}
}

func TestHiddenSelectionCannotBeToggledOrSavedInAnUnusableTerminal(t *testing.T) {
	for _, test := range []struct {
		name          string
		width, height int
	}{
		{name: "exact command does not fit", width: 16, height: 12},
		{name: "no item row fits", width: 120, height: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := newCommandSelectionHarness(t, strings.NewReader(" \rq"))
			h.terminal.width = test.width
			h.terminal.height = test.height
			if code := runCLI(h.command, []string{"config"}); code != ExitOK {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
			}
			if !strings.Contains(h.stdout.String(), "Resize terminal") || !strings.Contains(h.stdout.String(), "config unchanged") || strings.Contains(h.stdout.String(), "config saved") {
				t.Fatalf("unusable terminal result:\n%s", h.stdout.String())
			}
			if _, configured, err := h.store.Load(context.Background()); err != nil || configured {
				t.Fatalf("hidden selection was persisted: configured=%v err=%v", configured, err)
			}
		})
	}
}

func TestFirstActionAfterResizeOnlyFrameOnlyRedrawsTheSelection(t *testing.T) {
	catalog := DefaultCatalog()
	all := configurableCommandPaths(catalog)
	withoutFirst := append([]string(nil), all[1:]...)
	tests := []struct {
		name           string
		input          string
		wantConfigured bool
		wantPaths      []string
	}{
		{name: "first Enter only redraws", input: "\rq"},
		{name: "first Space only redraws before Enter saves", input: " \r", wantConfigured: true, wantPaths: all},
		{name: "second Enter may save", input: "\r\r", wantConfigured: true, wantPaths: all},
		{name: "second Space may toggle", input: "  \r", wantConfigured: true, wantPaths: withoutFirst},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newCommandSelectionHarness(t, strings.NewReader(test.input))
			h.terminal.sizes = []fakeConfigTerminalSize{
				{width: 16, height: 12},
				{width: 120, height: 12},
			}
			if code := runCLI(h.command, []string{"config"}); code != ExitOK {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
			}
			profile, configured, err := h.store.Load(context.Background())
			if err != nil || configured != test.wantConfigured {
				t.Fatalf("configured=%v err=%v, want configured=%v", configured, err, test.wantConfigured)
			}
			if configured && !reflect.DeepEqual(profile.EnabledCommands(), test.wantPaths) {
				t.Fatalf("saved paths=%v want=%v", profile.EnabledCommands(), test.wantPaths)
			}
			if !strings.Contains(h.stdout.String(), "Resize terminal") || !strings.Contains(h.stdout.String(), all[0]) {
				t.Fatalf("resize transition did not repaint the exact current identity:\n%s", h.stdout.String())
			}
		})
	}
}

func TestInvalidViewNoticeMustFitCompletelyBeforeFurtherMutation(t *testing.T) {
	catalog := DefaultCatalog()
	before := configurableCommandPaths(catalog)
	target := []string{"messages mark-read"}
	for _, test := range []struct {
		name          string
		width, height int
	}{
		{name: "notice has no height", width: 120, height: 4},
		{name: "notice does not fit width", width: 48, height: 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := configKeysForSelection(catalog, before, target) + "\rq"
			h := newCommandSelectionHarness(t, strings.NewReader(input))
			h.terminal.width = test.width
			h.terminal.height = test.height
			saveCommandSelection(t, h.store, before)
			if code := runCLI(h.command, []string{"config"}); code != ExitOK {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
			}
			if got := loadCommandSelection(t, h.store); !reflect.DeepEqual(got, before) {
				t.Fatalf("incomplete notice permitted a save: got=%v want=%v", got, before)
			}
			output := h.stdout.String()
			if !strings.Contains(output, "Resize terminal") || strings.Contains(output, "config saved") {
				t.Fatalf("incomplete notice did not fail closed:\n%s", output)
			}
		})
	}
}

func TestLegacyDoctorAndVersionSelectionsLoadAndNormalizeOnEnter(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader("\r"))
	saveCommandSelection(t, h.store, []string{"doctor", "rooms list", "version"})
	if code := runCLI(h.command, []string{"config"}); code != ExitOK {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}
	if got, want := loadCommandSelection(t, h.store), []string{"rooms list"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized paths=%v want=%v", got, want)
	}
	if !strings.Contains(h.stdout.String(), "legacy-removed=2") {
		t.Fatalf("save lacks migration evidence:\n%s", h.stdout.String())
	}
}

func TestActiveCommandViewHidesDisabledPathsFromEveryDiscoveryAndRoute(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader(""))
	enabled := withoutPrefix(configurableCommandPaths(DefaultCatalog()), "contact-requests ")
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
			t.Fatalf("hidden invocation %v succeeded: stdout=%q", args, h.stdout.String())
		}
		if strings.Contains(h.stdout.String(), "contact-requests") {
			t.Fatalf("hidden invocation %v leaked path: %q", args, h.stdout.String())
		}
	}

	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"help", "--format", "agent"}); code != ExitOK {
		t.Fatalf("root agent help exit=%d stderr=%q", code, h.stderr.String())
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
			t.Fatalf("scoped help %v exit=%d stderr=%q", args, code, h.stderr.String())
		}
		if strings.Contains(h.stdout.String(), "contact-requests") {
			t.Fatalf("scoped help %q leaked disabled path:\n%s", namespace, h.stdout.String())
		}
	}
}

func TestDisabledRouteIsUnknownBeforePATAndSameCLICanReenableIt(t *testing.T) {
	h := newCommandSelectionHarness(t, strings.NewReader(""))
	saveCommandSelection(t, h.store, []string{})
	factoryCalls := 0
	h.command.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
		factoryCalls++
		return nil, nil, fault.New(fault.KindAuthentication, "chatwork_token_missing", "A Chatwork API token is required.", false)
	}
	if code := runCLI(h.command, []string{"rooms", "list"}); code != ExitUsage {
		t.Fatalf("disabled route exit=%d stderr=%q", code, h.stderr.String())
	}
	if factoryCalls != 0 || !strings.Contains(h.stderr.String(), "code: unknown_command") {
		t.Fatalf("disabled route reached PAT: calls=%d stderr=%q", factoryCalls, h.stderr.String())
	}

	all := configurableCommandPaths(DefaultCatalog())
	h.reset(strings.NewReader(configKeysForSelection(DefaultCatalog(), []string{}, all)))
	if code := runCLI(h.command, []string{"config"}); code != ExitOK {
		t.Fatalf("reenable config exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
	}
	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"rooms", "list"}); code != ExitAuthentication {
		t.Fatalf("reenabled route exit=%d stderr=%q", code, h.stderr.String())
	}
	if factoryCalls != 1 {
		t.Fatalf("PAT calls=%d want=1", factoryCalls)
	}

	h.reset(strings.NewReader(""))
	if code := runCLI(h.command, []string{"contact-requests", "accept", "--request", "123"}); code != ExitRejected {
		t.Fatalf("unconfirmed mutation exit=%d stderr=%q", code, h.stderr.String())
	}
	if factoryCalls != 1 || !strings.Contains(h.stderr.String(), "code: mutation_rejected") {
		t.Fatalf("reenable bypassed confirmation: calls=%d stderr=%q", factoryCalls, h.stderr.String())
	}
}

func TestInvalidDependencyOrRecoverySelectionStaysInTUIAndDoesNotOverwrite(t *testing.T) {
	for _, test := range []struct {
		name   string
		target []string
		want   string
	}{
		{name: "missing producer", target: []string{"messages mark-read"}, want: "enable one producer:"},
		{name: "hidden recovery", target: []string{"rooms list", "messages send"}, want: "messages list"},
	} {
		t.Run(test.name, func(t *testing.T) {
			catalog := DefaultCatalog()
			before := configurableCommandPaths(catalog)
			keys := configKeysForSelection(catalog, before, test.target)
			h := newCommandSelectionHarness(t, strings.NewReader(keys))
			saveCommandSelection(t, h.store, before)
			if code := runCLI(h.command, []string{"config"}); code != ExitOK {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, h.stdout.String(), h.stderr.String())
			}
			if !strings.Contains(h.stdout.String(), test.want) {
				t.Errorf("diagnostic lacks %q:\n%s", test.want, h.stdout.String())
			}
			if got := loadCommandSelection(t, h.store); !reflect.DeepEqual(got, before) {
				t.Fatalf("invalid view overwrote profile: got=%v want=%v", got, before)
			}
		})
	}
}

type uncertainCommandSelectionStore struct {
	profile commandselection.Profile
	saved   bool
	persist bool
}

func (s *uncertainCommandSelectionStore) Load(context.Context) (commandselection.Profile, bool, error) {
	return s.profile, s.saved, nil
}

func (s *uncertainCommandSelectionStore) Save(_ context.Context, profile commandselection.Profile) error {
	if s.persist {
		s.profile = profile
		s.saved = true
	}
	return errors.New("durability result unavailable")
}

func TestUnclassifiedSaveOutcomePointsToDoctorFingerprintReconciliation(t *testing.T) {
	store := &uncertainCommandSelectionStore{persist: true}
	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader("\r"), &stdout, &stderr, DefaultCatalog(), passingInspector("runtime"))
	command.commandSelection = configcmd.New(store)
	command.terminal = &fakeConfigTerminalOpener{}
	if code := command.RunContext(context.Background(), []string{"config"}); code != ExitContract {
		t.Fatalf("config exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "code: unclassified_mutation_outcome") || !strings.Contains(stderr.String(), "cwk doctor") || !strings.Contains(stderr.String(), "expected-source=saved") {
		t.Fatalf("uncertain recovery=%q", stderr.String())
	}
	wantFingerprint := commandSelectionFingerprint(configurableCommandPaths(DefaultCatalog()))
	if !strings.Contains(stderr.String(), wantFingerprint) {
		t.Fatalf("uncertain fault lacks candidate fingerprint %s: %q", wantFingerprint, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := command.RunContext(context.Background(), []string{"doctor"}); code != ExitOK {
		t.Fatalf("doctor exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), wantFingerprint) || !strings.Contains(stdout.String(), "state=valid source=saved") {
		t.Fatalf("doctor lacks reconciled fingerprint %s:\n%s", wantFingerprint, stdout.String())
	}
}

func TestUnclassifiedSaveOutcomeDoesNotMistakeDefaultForPersistedCandidate(t *testing.T) {
	store := &uncertainCommandSelectionStore{persist: false}
	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader("\r"), &stdout, &stderr, DefaultCatalog(), passingInspector("runtime"))
	command.commandSelection = configcmd.New(store)
	command.terminal = &fakeConfigTerminalOpener{}
	if code := command.RunContext(context.Background(), []string{"config"}); code != ExitContract {
		t.Fatalf("config exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	wantFingerprint := commandSelectionFingerprint(configurableCommandPaths(DefaultCatalog()))
	for _, want := range []string{"expected-source=saved", "candidate-fingerprint=" + wantFingerprint} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("uncertain fault lacks %q: %q", want, stderr.String())
		}
	}

	stdout.Reset()
	stderr.Reset()
	if code := command.RunContext(context.Background(), []string{"doctor"}); code != ExitOK {
		t.Fatalf("doctor exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "state=valid source=default") || !strings.Contains(stdout.String(), wantFingerprint) {
		t.Fatalf("doctor did not expose the same-selection but non-persisted state:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "source=saved") {
		t.Fatalf("default selection falsely reconciled as the saved candidate:\n%s", stdout.String())
	}
}

func TestUnclassifiedSaveOutcomePublishesAndFollowsItsMessageGrammar(t *testing.T) {
	helpHarness := newCommandSelectionHarness(t, strings.NewReader(""))
	if code := runCLI(helpHarness.command, []string{"help", "config", "--format=agent"}); code != ExitOK {
		t.Fatalf("help exit=%d stdout=%q stderr=%q", code, helpHarness.stdout.String(), helpHarness.stderr.String())
	}
	var helpDocument agentDocument
	if err := json.Unmarshal(helpHarness.stdout.Bytes(), &helpDocument); err != nil {
		t.Fatalf("config agent help: %v\n%s", err, helpHarness.stdout.String())
	}
	if len(helpDocument.Commands) != 1 || helpDocument.Commands[0].Path != "config" {
		t.Fatalf("config help commands=%+v", helpDocument.Commands)
	}
	var declared *CommandError
	for index := range helpDocument.Commands[0].Contract.Errors {
		candidate := &helpDocument.Commands[0].Contract.Errors[index]
		if candidate.Code == "unclassified_mutation_outcome" {
			declared = candidate
			break
		}
	}
	if declared == nil || declared.MessageGrammar != commandSelectionUncertainMessageGrammar {
		t.Fatalf("uncertain message grammar = %+v", declared)
	}

	store := &uncertainCommandSelectionStore{persist: true}
	var stdout, stderr bytes.Buffer
	command := newCLI(strings.NewReader("\r"), &stdout, &stderr, DefaultCatalog(), passingInspector("runtime"))
	command.commandSelection = configcmd.New(store)
	command.terminal = &fakeConfigTerminalOpener{}
	if code := command.RunContext(context.Background(), []string{"--error-format=json", "config"}); code != ExitContract {
		t.Fatalf("config exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var runtimeDocument errorDocument
	if err := json.Unmarshal(stderr.Bytes(), &runtimeDocument); err != nil {
		t.Fatalf("config JSON error: %v\n%s", err, stderr.String())
	}
	wantFingerprint := commandSelectionFingerprint(configurableCommandPaths(DefaultCatalog()))
	if got, want := runtimeDocument.Error.Message, commandSelectionUncertainMessage(wantFingerprint); got != want {
		t.Fatalf("uncertain message=%q want=%q", got, want)
	}
	if len(runtimeDocument.Error.NextActions) != 1 || runtimeDocument.Error.NextActions[0].Command != "doctor" {
		t.Fatalf("uncertain next actions=%+v", runtimeDocument.Error.NextActions)
	}
}

func TestCatalogRejectsUnsafeErrorMessageGrammar(t *testing.T) {
	commands := DefaultCatalog().Commands()
	for commandIndex := range commands {
		if commands[commandIndex].Path != "config" {
			continue
		}
		for errorIndex := range commands[commandIndex].Agent.Errors {
			if commands[commandIndex].Agent.Errors[errorIndex].Code == "unclassified_mutation_outcome" {
				commands[commandIndex].Agent.Errors[errorIndex].MessageGrammar = "unsafe\ngrammar"
			}
		}
	}
	if err := NewCatalog(commands...).Validate(); err == nil || !strings.Contains(err.Error(), "error message grammar") {
		t.Fatalf("unsafe message grammar validation error=%v", err)
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
