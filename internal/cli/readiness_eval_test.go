//go:build readiness_eval

package cli

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	appauthn "github.com/tasuku43/cwk/internal/app/authn"
	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/app/configcmd"
	domainauthn "github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/commandselection"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/infra/readinessio"
)

const (
	readinessEvalAuditFlag        = "--cwk-readiness-audit"
	readinessEvalRoomRef          = "8173"
	readinessEvalMessage          = "921004"
	readinessEvalFileRef          = "44501"
	readinessEvalTaskRef          = "66301"
	readinessEvalCallLimit        = 7
	readinessEvalJournalByteLimit = 1 << 20
	readinessEvalOutputByteLimit  = 128 << 10
)

// readinessReservationTestHook is set only by a local test subprocess.
var readinessReservationTestHook func()

// TestReadinessHelper is selected only by the disposable wrapper built for a
// readiness run. Its fixture, selection profile, authenticator, and task port
// live in this test binary; production constructors and CLI input surfaces are
// unchanged.
func TestReadinessHelper(t *testing.T) {
	auditPath, publicArgs, ok := readinessHelperArguments(os.Args)
	if !ok {
		t.Skip("readiness helper is invoked only by the disposable cwk wrapper")
	}
	journal, err := readinessio.LockJournal(auditPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "readiness journal lock is unavailable")
		os.Exit(2)
	}
	defer journal.Close()
	priorCalls, err := countReadinessJournalReader(journal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "readiness journal is invalid: %v\n", err)
		os.Exit(2)
	}
	if priorCalls >= readinessEvalCallLimit {
		fmt.Fprintf(os.Stderr, "readiness CLI invocation budget exceeded: limit=%d\n", readinessEvalCallLimit)
		os.Exit(2)
	}

	if err := appendLockedReadinessEvent(journal, readinessInvocationEvent{Type: "cwk_invocation_started", Argv: publicArgs}); err != nil {
		fmt.Fprintln(os.Stderr, "cannot reserve readiness invocation")
		os.Exit(2)
	}

	if readinessReservationTestHook != nil {
		readinessReservationTestHook()
	}

	stdout, stderr := &readinessEvalBuffer{}, &readinessEvalBuffer{}
	fixture := newReadinessFixture()
	cli := newCLI(strings.NewReader(""), stdout, stderr, DefaultCatalog(), passingInspector("synthetic readiness evaluation"))
	profile, err := commandselection.New([]string{
		"rooms list",
		"messages list",
		"messages resources",
		"room-tasks show",
		"files show",
	})
	if err != nil {
		t.Fatalf("readiness profile: %v", err)
	}
	cli.commandSelection = configcmd.New(readinessSelectionStore{profile: profile})
	cli.chatwork = chatworkcmd.New(fixture)
	binding, err := domainauthn.NewBindingID("readiness-eval-binding")
	if err != nil {
		t.Fatalf("readiness binding: %v", err)
	}
	authenticator := &readinessAuthenticator{binding: binding}
	cli.chatworkAuth = appauthn.New(authenticator)
	cli.Version = "readiness-eval"

	started := time.Now().UTC()
	exitCode := cli.RunContext(context.Background(), publicArgs)
	ended := time.Now().UTC()
	if stdout.exceeded || stderr.exceeded {
		fmt.Fprintln(os.Stderr, "readiness fixture output byte limit exceeded")
		os.Exit(2)
	}
	event := readinessInvocationEvent{
		Type:           "cwk_invocation",
		Argv:           append([]string(nil), publicArgs...),
		ExitCode:       exitCode,
		Stdout:         stdout.String(),
		Stderr:         stderr.String(),
		DurationMillis: ended.Sub(started).Milliseconds(),
		FixtureSHA256:  readinessFixtureDigest(),
		Authentication: readinessAuthenticationEvent{Calls: authenticator.calls},
		ProviderCalls:  append([]readinessProviderCall(nil), fixture.calls...),
		Writes:         fixture.writes,
	}
	if err := appendLockedReadinessEvent(journal, event); err != nil {
		t.Fatalf("write readiness trace: %v", err)
	}
	_, _ = os.Stdout.Write(stdout.Bytes())
	_, _ = os.Stderr.Write(stderr.Bytes())
	os.Exit(exitCode)
}

type readinessEvalBuffer struct {
	buffer   bytes.Buffer
	exceeded bool
}

func (buffer *readinessEvalBuffer) Write(p []byte) (int, error) {
	if len(p) > readinessEvalOutputByteLimit-buffer.buffer.Len() {
		buffer.exceeded = true
		return 0, errors.New("readiness fixture output byte limit exceeded")
	}
	return buffer.buffer.Write(p)
}

func (buffer *readinessEvalBuffer) String() string { return buffer.buffer.String() }
func (buffer *readinessEvalBuffer) Bytes() []byte  { return buffer.buffer.Bytes() }

// TestReadinessFixtureContract pins the task inputs, decoys, and expected
// internal call budget without invoking production I/O or an external agent.
func TestReadinessFixtureContract(t *testing.T) {
	fixture := newReadinessFixture()
	room := readinessEvalReference(t, chatwork.ReferenceRoom, readinessEvalRoomRef)
	message := readinessEvalReference(t, chatwork.ReferenceMessage, readinessEvalMessage)
	file := readinessEvalReference(t, chatwork.ReferenceFile, readinessEvalFileRef)
	task := readinessEvalReference(t, chatwork.ReferenceTask, readinessEvalTaskRef)
	if fixture.room.Ref != room || fixture.message != message || fixture.files[1].Ref != file || fixture.tasks[1].Ref != task {
		t.Fatal("readiness fixture changed an exact canonical reference")
	}
	if fixture.files[0].Message == message || fixture.files[0].Room != room || fixture.tasks[0].Message == message || fixture.tasks[0].Room.Ref != room {
		t.Fatal("readiness fixture no longer contains same-room, wrong-message decoys")
	}
	if fixture.files[1].Message != message || fixture.files[1].Room != room || fixture.tasks[1].Message != message || fixture.tasks[1].Room.Ref != room {
		t.Fatal("readiness fixture lacks exact message matches")
	}

	binding, err := domainauthn.NewBindingID("readiness-fixture-contract")
	if err != nil {
		t.Fatal(err)
	}
	service := chatworkcmd.New(fixture)
	resourceRequest := chatwork.Request{Task: chatwork.TaskMessagesResources, Room: room, Message: message, MessageResourcesKind: chatwork.MessageResourceAll}
	resources, err := service.Execute(context.Background(), binding, resourceRequest)
	if err != nil {
		t.Fatalf("messages resources fixture: %v", err)
	}
	if resources.MessageResources.Files.SourceCount != 2 || len(resources.MessageResources.Files.Matches) != 1 || resources.MessageResources.Files.Matches[0].Ref != file {
		t.Fatalf("file exact-match projection = %+v", resources.MessageResources.Files)
	}
	if resources.MessageResources.Tasks.SourceCount != 2 || len(resources.MessageResources.Tasks.Matches) != 1 || resources.MessageResources.Tasks.Matches[0].Ref != task {
		t.Fatalf("task exact-match projection = %+v", resources.MessageResources.Tasks)
	}
	fileResult, err := service.Execute(context.Background(), binding, chatwork.Request{Task: chatwork.TaskFilesShow, Room: room, File: file})
	if err != nil || len(fileResult.Files) != 1 || fileResult.Files[0].Name != "公開前確認表.pdf" {
		t.Fatalf("files show exact reference result=%+v err=%v", fileResult, err)
	}
	taskResult, err := service.Execute(context.Background(), binding, chatwork.Request{Task: chatwork.TaskRoomTasksShow, Room: room, TaskRef: task})
	if err != nil || len(taskResult.Tasks) != 1 || taskResult.Tasks[0].Body != "staging 確認後にリリース候補を公開する" {
		t.Fatalf("room-tasks show exact reference result=%+v err=%v", taskResult, err)
	}
	keyBytes, err := os.ReadFile(filepath.Join("..", "..", "tools", "readinessrunner", "testdata", "semantic_answer.json"))
	if err != nil {
		t.Fatal(err)
	}
	var key map[string]any
	if err := json.Unmarshal(keyBytes, &key); err != nil {
		t.Fatal(err)
	}
	files, tasks := resources.MessageResources.Files, resources.MessageResources.Tasks
	observed := map[string]any{
		"room_ref": room.Value, "message_ref": message.Value, "kind": string(resources.MessageResources.Kind),
		"files": map[string]any{"source_count": files.SourceCount, "limit": files.Coverage.Limit, "complete": files.Coverage.Complete, "absence": string(files.Absence), "matched": len(files.Matches), "ref": fileResult.Files[0].Ref.Value, "name": fileResult.Files[0].Name},
		"tasks": map[string]any{"source_count": tasks.SourceCount, "limit": tasks.Coverage.Limit, "complete": tasks.Coverage.Complete, "absence": string(tasks.Absence), "matched": len(tasks.Matches), "ref": taskResult.Tasks[0].Ref.Value, "body": taskResult.Tasks[0].Body, "status": taskResult.Tasks[0].Status},
	}
	expectedJSON, err := json.Marshal(key)
	if err != nil {
		t.Fatal(err)
	}
	observedJSON, err := json.Marshal(observed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(expectedJSON, observedJSON) {
		t.Fatal("semantic answer key does not match production-CLI fixture results")
	}
	wantCalls := []string{"files.list", "room-tasks.list", "files.show", "room-tasks.show"}
	if len(fixture.calls) != len(wantCalls) {
		t.Fatalf("fixture provider calls = %d, want %d", len(fixture.calls), len(wantCalls))
	}
	for index, want := range wantCalls {
		if fixture.calls[index].Task != want {
			t.Fatalf("provider call %d task = %q, want %q", index, fixture.calls[index].Task, want)
		}
	}
}

func TestReadinessHelperArgumentsPreservesPublicArgvAfterAuditPreamble(t *testing.T) {
	auditFile := filepath.Join(t.TempDir(), "cwk-readiness-events.jsonl")
	path, argv, ok := readinessHelperArguments([]string{
		"cwk-eval", "-test.run", "^TestReadinessHelper$", "--",
		readinessEvalAuditFlag, auditFile, "messages", "resources", "--room", readinessEvalRoomRef,
	})
	if !ok {
		t.Fatal("readiness test-binary trailer was not recognized")
	}
	if path != auditFile || strings.Join(argv, " ") != "messages resources --room 8173" {
		t.Fatalf("helper parsed audit path/argv = %q/%q", path, argv)
	}
	if _, _, ok := readinessHelperArguments([]string{"cwk-eval", "-test.run", "^TestReadinessHelper$", "--", "--help"}); ok {
		t.Fatal("ordinary public argv was mistaken for an audit-internal invocation")
	}
}

func TestReadinessJournalCountRejectsUnknownEvents(t *testing.T) {
	path := t.TempDir() + "/events.jsonl"
	if err := os.WriteFile(path, []byte("{\"type\":\"foreign\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := countReadinessJournalEvents(path); err == nil {
		t.Fatal("unknown readiness journal entry was counted")
	}
}

func TestReadinessJournalCountRejectsOversizedInput(t *testing.T) {
	path := t.TempDir() + "/events.jsonl"
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), readinessEvalJournalByteLimit+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := countReadinessJournalEvents(path); err == nil || !strings.Contains(err.Error(), "byte limit") {
		t.Fatalf("oversized journal error = %v", err)
	}
}

func TestReadinessEvalBufferRejectsOversizedWrites(t *testing.T) {
	var output readinessEvalBuffer
	if n, err := output.Write(bytes.Repeat([]byte("a"), readinessEvalOutputByteLimit)); err != nil || n != readinessEvalOutputByteLimit {
		t.Fatalf("exact limit: n=%d err=%v", n, err)
	}
	if _, err := io.WriteString(&output, "x"); err == nil || !output.exceeded || len(output.Bytes()) != readinessEvalOutputByteLimit {
		t.Fatalf("excess output: err=%v exceeded=%v bytes=%d", err, output.exceeded, len(output.Bytes()))
	}
}

func TestReadinessJournalAppendRejectsOversizedEvent(t *testing.T) {
	path := t.TempDir() + "/events.jsonl"
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(readinessEvalJournalByteLimit - 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := appendReadinessEvent(path, readinessInvocationEvent{Type: "cwk_invocation"}); err == nil {
		t.Fatal("oversized append was accepted")
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() != readinessEvalJournalByteLimit-1 {
		t.Fatalf("journal size changed after refusal: %v, %v", info, err)
	}
}

type readinessSelectionStore struct {
	profile commandselection.Profile
}

func (s readinessSelectionStore) Load(context.Context) (commandselection.Profile, bool, error) {
	return s.profile, true, nil
}

func (readinessSelectionStore) Save(context.Context, commandselection.Profile) error { return nil }

type readinessAuthenticator struct {
	binding domainauthn.BindingID
	calls   int
}

func (a *readinessAuthenticator) Authenticate(_ context.Context, requirement domainauthn.Requirement) (domainauthn.Session, error) {
	a.calls++
	method := domainauthn.MethodPAT
	if len(requirement.Methods) > 0 {
		method = requirement.Methods[0]
	}
	return domainauthn.Session{
		Method:              method,
		Authority:           requirement.Authority,
		Audience:            requirement.Audience,
		SubjectID:           "synthetic-readiness-user",
		AccountID:           requirement.AccountID,
		BindingID:           a.binding,
		GrantedCapabilities: append([]string(nil), requirement.RequiredCapabilities...),
	}, nil
}

type readinessFixturePort struct {
	room    chatwork.Room
	message chatwork.Reference
	files   []chatwork.File
	tasks   []chatwork.WorkTask
	calls   []readinessProviderCall
	writes  int
}

func newReadinessFixture() *readinessFixturePort {
	room := mustReadinessReference(chatwork.ReferenceRoom, readinessEvalRoomRef)
	message := mustReadinessReference(chatwork.ReferenceMessage, readinessEvalMessage)
	otherMessage := mustReadinessReference(chatwork.ReferenceMessage, "921005")
	fileRef := mustReadinessReference(chatwork.ReferenceFile, readinessEvalFileRef)
	decoyFileRef := mustReadinessReference(chatwork.ReferenceFile, "44502")
	taskRef := mustReadinessReference(chatwork.ReferenceTask, readinessEvalTaskRef)
	decoyTaskRef := mustReadinessReference(chatwork.ReferenceTask, "66302")
	uploaderRef := mustReadinessReference(chatwork.ReferenceAccount, "7101")
	assigneeRef := mustReadinessReference(chatwork.ReferenceAccount, "7102")
	assignerRef := mustReadinessReference(chatwork.ReferenceAccount, "7103")
	roomValue := chatwork.Room{Ref: room, Name: "暫定リリース計画"}
	return &readinessFixturePort{
		room:    roomValue,
		message: message,
		files: []chatwork.File{
			{Ref: decoyFileRef, Room: room, Message: otherMessage, Account: chatwork.Account{Ref: uploaderRef, Name: "合成アップロード担当"}, Name: "別メッセージの資料.txt", Size: 128},
			{Ref: fileRef, Room: room, Message: message, Account: chatwork.Account{Ref: uploaderRef, Name: "合成アップロード担当"}, Name: "公開前確認表.pdf", Size: 2048},
		},
		tasks: []chatwork.WorkTask{
			{Ref: decoyTaskRef, Room: roomValue, Account: chatwork.Account{Ref: assigneeRef, Name: "合成担当者"}, AssignedBy: chatwork.Account{Ref: assignerRef, Name: "合成依頼者"}, Message: otherMessage, Body: "別のメッセージのタスク", Status: "open"},
			{Ref: taskRef, Room: roomValue, Account: chatwork.Account{Ref: assigneeRef, Name: "合成担当者"}, AssignedBy: chatwork.Account{Ref: assignerRef, Name: "合成依頼者"}, Message: message, Body: "staging 確認後にリリース候補を公開する", Status: "open"},
		},
	}
}

func (p *readinessFixturePort) Execute(_ context.Context, _ domainauthn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	operation := readinessProviderOperation(request.Task)
	if operation == "write" {
		p.writes++
	}
	call := readinessProviderCall{
		Operation:  operation,
		Task:       string(request.Task),
		RoomRef:    request.Room.Value,
		MessageRef: request.Message.Value,
		FileRef:    request.File.Value,
		TaskRef:    request.TaskRef.Value,
		Kind:       string(request.MessageResourcesKind),
	}
	p.calls = append(p.calls, call)
	if operation != "read" {
		return chatwork.Result{}, fault.New(fault.KindUnsupported, "readiness_fixture_task_unavailable", "この合成評価では書き込みまたは未登録の要求を提供していません", false)
	}
	coverage := chatwork.Coverage{Kind: "provider_window", Limit: chatwork.MaxMessageResourceSourceItems, Complete: false, Description: "synthetic bounded source"}
	switch request.Task {
	case chatwork.TaskFilesList:
		return chatwork.Result{Task: request.Task, Coverage: coverage, Files: append([]chatwork.File(nil), p.files...)}, nil
	case chatwork.TaskRoomTasksList:
		return chatwork.Result{Task: request.Task, Coverage: coverage, Tasks: append([]chatwork.WorkTask(nil), p.tasks...)}, nil
	case chatwork.TaskFilesShow:
		for _, file := range p.files {
			if file.Ref == request.File && file.Room == request.Room {
				return chatwork.Result{Task: request.Task, Files: []chatwork.File{file}}, nil
			}
		}
		return chatwork.Result{}, readinessMissingTarget("file")
	case chatwork.TaskRoomTasksShow:
		for _, task := range p.tasks {
			if task.Ref == request.TaskRef && task.Room.Ref == request.Room {
				return chatwork.Result{Task: request.Task, Tasks: []chatwork.WorkTask{task}}, nil
			}
		}
		return chatwork.Result{}, readinessMissingTarget("room task")
	case chatwork.TaskRoomsList:
		return chatwork.Result{Task: request.Task, Coverage: chatwork.Coverage{Kind: "complete", Complete: true}, Rooms: []chatwork.Room{p.room}}, nil
	default:
		return chatwork.Result{}, fault.New(fault.KindUnsupported, "readiness_fixture_task_unavailable", "この合成評価では要求された読み取りタスクを提供していません", false)
	}
}

func readinessProviderOperation(task chatwork.Task) string {
	switch task {
	case chatwork.TaskRoomsList,
		chatwork.TaskMessagesList,
		chatwork.TaskFilesList,
		chatwork.TaskFilesShow,
		chatwork.TaskRoomTasksList,
		chatwork.TaskRoomTasksShow:
		return "read"
	default:
		// Unknown tasks fail closed in the fixture journal so the evaluator
		// cannot report a zero-write run from an unclassified request.
		return "write"
	}
}

func readinessMissingTarget(label string) error {
	return fault.New(fault.KindInvalidInput, "readiness_fixture_target_missing", label+" の合成参照は存在しません", false)
}

type readinessInvocationEvent struct {
	Type           string                       `json:"type"`
	Argv           []string                     `json:"argv"`
	ExitCode       int                          `json:"exit_code"`
	Stdout         string                       `json:"stdout"`
	Stderr         string                       `json:"stderr"`
	DurationMillis int64                        `json:"duration_ms"`
	FixtureSHA256  string                       `json:"fixture_sha256"`
	Authentication readinessAuthenticationEvent `json:"authentication"`
	ProviderCalls  []readinessProviderCall      `json:"provider_calls"`
	Writes         int                          `json:"writes"`
}

type readinessAuthenticationEvent struct {
	Calls int `json:"calls"`
}

type readinessProviderCall struct {
	Operation  string `json:"operation"`
	Task       string `json:"task"`
	RoomRef    string `json:"room_ref,omitempty"`
	MessageRef string `json:"message_ref,omitempty"`
	FileRef    string `json:"file_ref,omitempty"`
	TaskRef    string `json:"task_ref,omitempty"`
	Kind       string `json:"kind,omitempty"`
}

func readinessHelperArguments(args []string) (string, []string, bool) {
	for index, argument := range args {
		if argument != "--" {
			continue
		}
		remaining := args[index+1:]
		if len(remaining) < 1 || remaining[0] != readinessEvalAuditFlag {
			return "", nil, false
		}
		if len(remaining) < 3 || !filepath.IsAbs(remaining[1]) {
			return "", nil, false
		}
		return remaining[1], append([]string(nil), remaining[2:]...), true
	}
	return "", nil, false
}

func appendReadinessEvent(path string, value readinessInvocationEvent) error {
	if path == "" {
		return errors.New("audit path is empty")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return readinessio.AppendBounded(path, append(encoded, '\n'), readinessEvalJournalByteLimit)
}

func appendLockedReadinessEvent(file *os.File, value readinessInvocationEvent) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return readinessio.AppendLocked(file, append(encoded, '\n'), readinessEvalJournalByteLimit)
}

func countReadinessJournalEvents(path string) (int, error) {
	file, err := readinessio.OpenRegular(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return countReadinessJournalReader(file)
}

func countReadinessJournalReader(reader io.Reader) (int, error) {
	contents, err := io.ReadAll(io.LimitReader(reader, readinessEvalJournalByteLimit+1))
	if err != nil {
		return 0, fmt.Errorf("read invocation journal: %w", err)
	}
	if len(contents) > readinessEvalJournalByteLimit {
		return 0, errors.New("invocation journal byte limit exceeded")
	}
	scanner := bufio.NewScanner(bytes.NewReader(contents))
	scanner.Buffer(make([]byte, 16*1024), readinessEvalJournalByteLimit)
	count := 0
	var started *readinessInvocationEvent
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			return 0, errors.New("blank invocation journal line")
		}
		var event readinessInvocationEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return 0, fmt.Errorf("decode invocation journal: %w", err)
		}
		switch event.Type {
		case "cwk_invocation_started":
			if started != nil {
				return 0, errors.New("nested invocation reservation")
			}
			started = &event
		case "cwk_invocation":
			if started == nil || !slices.Equal(started.Argv, event.Argv) {
				return 0, errors.New("unmatched invocation completion")
			}
			count++
			started = nil
		default:
			return 0, errors.New("unexpected journal event type")
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("read invocation journal: %w", err)
	}
	if started != nil {
		return 0, errors.New("incomplete invocation reservation")
	}
	return count, nil
}

func mustReadinessReference(kind chatwork.ReferenceKind, value string) chatwork.Reference {
	reference, err := chatwork.NewReference(kind, value)
	if err != nil {
		panic(fmt.Sprintf("readiness fixture reference %s/%s: %v", kind, value, err))
	}
	return reference
}

func readinessEvalReference(t *testing.T, kind chatwork.ReferenceKind, value string) chatwork.Reference {
	t.Helper()
	reference, err := chatwork.NewReference(kind, value)
	if err != nil {
		t.Fatalf("NewReference(%s, %q): %v", kind, value, err)
	}
	return reference
}

// readinessFixtureDigest identifies the exact in-code synthetic corpus for a
// run report. It is intentionally computed from semantic fixture facts, not
// host-specific process data.
func readinessFixtureDigest() string {
	fixture := newReadinessFixture()
	canonical := struct {
		RoomRef    string              `json:"room_ref"`
		MessageRef string              `json:"message_ref"`
		Files      []chatwork.File     `json:"files"`
		Tasks      []chatwork.WorkTask `json:"tasks"`
	}{
		RoomRef: fixture.room.Ref.Value, MessageRef: fixture.message.Value,
		Files: fixture.files, Tasks: fixture.tasks,
	}
	encoded, _ := json.Marshal(canonical)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
