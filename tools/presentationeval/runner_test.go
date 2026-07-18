package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fakeCodexProcesses struct {
	t        *testing.T
	requests []processRequest
}

func (f *fakeCodexProcesses) Run(_ context.Context, request processRequest) (processResponse, error) {
	f.requests = append(f.requests, request)
	switch request.Path {
	case "/test/codex":
		if reflect.DeepEqual(request.Args, []string{"--version"}) {
			return processResponse{Stdout: []byte(pinnedCodexCLI + "\n")}, nil
		}
		last := argumentAfter(request.Args, "--output-last-message")
		if last == "" {
			f.t.Fatal("Codex request has no last-message path")
		}
		if strings.Contains(string(request.Stdin), "paired token-accounting probe") {
			answer := []byte(`{"ack":true}`)
			if err := os.WriteFile(last, answer, 0o600); err != nil {
				f.t.Fatal(err)
			}
			input := int64(100)
			if strings.Contains(string(request.Stdin), "TASK ") {
				input = 150
			}
			return processResponse{Stdout: fakeCodexJSONL(f.t, "", "", answer, input)}, nil
		}
		result, err := simulate("attention.rooms", []string{"rooms", "list"})
		if err != nil {
			f.t.Fatal(err)
		}
		answer := []byte(`{"attention_room_refs":["4101","4102","4104"]}`)
		if err := os.WriteFile(last, answer, 0o600); err != nil {
			f.t.Fatal(err)
		}
		return processResponse{Stdout: fakeCodexJSONL(f.t, "cwk rooms list", string(result.Stdout), answer, 200)}, nil
	case "git":
		if reflect.DeepEqual(request.Args, []string{"status", "--porcelain", "--untracked-files=all"}) {
			return processResponse{Stdout: []byte{}}, nil
		}
		return processResponse{Stdout: []byte(strings.Repeat("a", 40) + "\n")}, nil
	case "go":
		output := argumentAfter(request.Args, "-o")
		if err := os.WriteFile(output, []byte("fixture"), 0o755); err != nil {
			f.t.Fatal(err)
		}
		return processResponse{}, nil
	default:
		f.t.Fatalf("unexpected process %q", request.Path)
		return processResponse{}, nil
	}
}

func fakeCodexJSONL(t *testing.T, command, output string, answer []byte, input int64) []byte {
	t.Helper()
	var buffer bytes.Buffer
	write := func(value any) {
		if err := writeJSON(&buffer, value); err != nil {
			t.Fatal(err)
		}
	}
	write(map[string]any{"type": "thread.started", "thread_id": "thread-1"})
	write(map[string]any{"type": "turn.started"})
	if command != "" {
		write(map[string]any{"type": "item.started", "item": map[string]any{"id": "cmd-1", "type": "command_execution", "command": command, "status": "in_progress"}})
		write(map[string]any{"type": "item.completed", "item": map[string]any{"id": "cmd-1", "type": "command_execution", "command": command, "aggregated_output": output, "exit_code": 0, "status": "completed"}})
	}
	write(map[string]any{"type": "item.completed", "item": map[string]any{"id": "msg-1", "type": "agent_message", "text": string(answer)}})
	write(map[string]any{"type": "turn.completed", "usage": map[string]any{"input_tokens": input, "cached_input_tokens": 0, "output_tokens": 10}})
	return buffer.Bytes()
}

func TestRunBenchmarkBuildsFixtureAndRecordsFailClosedCodexRun(t *testing.T) {
	processes := &fakeCodexProcesses{t: t}
	now := time.Unix(1700000000, 0)
	output := filepath.Join(t.TempDir(), "p.jsonl")
	err := runBenchmark(context.Background(), runnerDependencies{Processes: processes, Now: func() time.Time { now = now.Add(time.Second); return now }}, benchmarkRequest{
		Candidate: "p", SituationID: "attention.rooms", CodexPath: "/test/codex", Model: "gpt-test-pinned", OutputPath: output, Repetitions: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var submission runSubmission
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&submission); err != nil {
		t.Fatal(err)
	}
	if submission.Candidate != "p" || submission.Commit != strings.Repeat("a", 40) || submission.PresentationProbe.InputTokenDelta != 50 || submission.WallTimeMS != 1000 {
		t.Fatalf("submission = %#v", submission)
	}
	scored, err := scoreSubmission(submission)
	if err != nil || !scored.ExactAnswer || !scored.TranscriptPass || !scored.UsagePass {
		t.Fatalf("scored = %#v, err = %v", scored, err)
	}

	var e2e processRequest
	for _, request := range processes.requests {
		if request.Path == "/test/codex" && strings.Contains(string(request.Stdin), "User situation:") {
			e2e = request
		}
	}
	joined := strings.Join(e2e.Args, " ")
	for _, required := range []string{"--ask-for-approval never exec", "--sandbox workspace-write", "--ignore-user-config", "--ignore-rules", "sandbox_workspace_write.network_access=false", "shell_environment_policy.inherit=none", `model_reasoning_effort="medium"`, "--disable apps", "--disable plugins", "--disable standalone_web_search"} {
		if !strings.Contains(joined, required) {
			t.Errorf("Codex args missing %q: %s", required, joined)
		}
	}
	if strings.Contains(joined, "--search") {
		t.Fatalf("Codex args enabled search: %s", joined)
	}
}

func TestCodexJSONLRejectsUnknownEventsAndShellOperators(t *testing.T) {
	if _, err := parseCodexJSONL([]byte(`{"type":"future.event"}`+"\n"), []byte(`{}`), true); err == nil {
		t.Fatal("unknown event accepted")
	}
	if _, err := parseCWKCommand("cwk rooms list | grep x"); err == nil {
		t.Fatal("shell pipeline accepted")
	}
	got, err := parseCWKCommand(`/bin/zsh -lc 'cwk rooms list'`)
	if err != nil || !reflect.DeepEqual(got, []string{"rooms", "list"}) {
		t.Fatalf("single Codex shell wrapper = %v, %v", got, err)
	}
	if _, err := parseCWKCommand(`/bin/zsh -lc 'cwk rooms list; id'`); err == nil {
		t.Fatal("shell wrapper with a second command accepted")
	}
}

func TestRecordedAgentFailureIsPreservedAsIneligible(t *testing.T) {
	submission := runSubmission{
		Schema: runSchema, RunID: "failed-1", Candidate: "p", SituationID: "attention.rooms", Repetition: 1,
		Agent: pinnedCodexCLI, Model: "gpt-test", Commit: strings.Repeat("a", 40), WallTimeMS: 10,
		FailureCode: "agent_run_failed", Steps: []runStep{}, Answer: json.RawMessage(`{}`),
		Usage: tokenUsage{}, PresentationProbe: presentationProbe{ProbeID: "p-attention.rooms-1"},
		AllowedTools: []string{"cwk"}, ForbiddenTools: []string{}, NonCWKTools: []string{}, ExternalProcessing: []string{},
	}
	scored, err := scoreSubmission(submission)
	if err != nil {
		t.Fatal(err)
	}
	if scored.FailureCode != "agent_run_failed" || scored.ExactAnswer || scored.CriticalPass || scored.TranscriptPass || scored.UsagePass {
		t.Fatalf("recorded failure was not scored as ineligible: %#v", scored)
	}
}

func TestFrozenSuiteSchedulesTenScoredRunsAndEightProbePairs(t *testing.T) {
	runs := 0
	probes := 0
	for _, scenario := range situations() {
		runs += suiteRepetitions(scenario)
		probes++
	}
	if runs != 10 || probes != 8 {
		t.Fatalf("suite schedule = %d runs, %d probes; want 10 and 8", runs, probes)
	}
}

func TestCommandCompletionAcceptsIntentionalNonZeroExitOnlyAsFailed(t *testing.T) {
	for _, test := range []struct {
		status string
		exit   int
		want   bool
	}{{"completed", 0, true}, {"completed", 6, true}, {"failed", 6, true}, {"failed", 0, false}, {"in_progress", 0, false}} {
		if got := validCommandCompletion(test.status, test.exit); got != test.want {
			t.Errorf("validCommandCompletion(%q, %d) = %t, want %t", test.status, test.exit, got, test.want)
		}
	}
}

func TestCandidateLabelsAreFixedAndAnswerSchemaDoesNotLeakValues(t *testing.T) {
	for _, candidate := range []string{"c0", "p", "l", "r", "j"} {
		if err := validateCandidate(candidate); err != nil {
			t.Errorf("candidate %q rejected: %v", candidate, err)
		}
	}
	if err := validateCandidate("candidate-c"); err == nil {
		t.Fatal("unreviewed candidate label accepted")
	}
	schema, err := answerSchema(raw(`{"secret_ref":"4101","ok":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(schema, []byte("4101")) || bytes.Contains(schema, []byte(`"const"`)) {
		t.Fatalf("answer schema leaked answer values: %s", schema)
	}
}

func argumentAfter(values []string, name string) string {
	for index := range values {
		if values[index] == name && index+1 < len(values) {
			return values[index+1]
		}
	}
	return ""
}
