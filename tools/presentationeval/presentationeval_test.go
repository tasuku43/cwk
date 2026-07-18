package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSituationsAreNaturalLanguageWorkflowUnits(t *testing.T) {
	values := situations()
	wantIDs := []string{
		"attention.rooms", "failure.recover-not-found", "file.verify-created-parent",
		"mark-read.explicit-zero", "message.hostile-untrusted", "reply.choose-target",
		"rooms.large-attention", "thread.relationships",
	}
	gotIDs := make([]string, len(values))
	for index, value := range values {
		gotIDs[index] = value.ID
		if value.UserPrompt == "" || value.AnswerShape == "" || len(value.RequiredPaths) == 0 || value.MaxCommands < len(value.RequiredPaths) {
			t.Errorf("situation %s is not an executable workflow unit: %#v", value.ID, value)
		}
		if !json.Valid(value.AnswerKey) || len(value.CriticalPaths) == 0 {
			t.Errorf("situation %s has no exact semantic oracle", value.ID)
		}
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("situation IDs = %v, want %v", gotIDs, wantIDs)
	}
}

func TestSimulatorUsesPublicHelpAndExactFixtureReferences(t *testing.T) {
	help, err := simulate("thread.relationships", []string{"help", "messages", "list", "--format", "agent"})
	if err != nil || help.ExitCode != 0 || !bytes.Contains(help.Stdout, []byte(`"path":"messages list"`)) {
		t.Fatalf("help = %#v, err = %v", help, err)
	}

	rooms, err := simulate("thread.relationships", []string{"rooms", "list"})
	if err != nil || rooms.ExitCode != 0 || !bytes.Contains(rooms.Stdout, []byte("4101")) {
		t.Fatalf("rooms = %#v, err = %v", rooms, err)
	}
	messages, err := simulate("thread.relationships", []string{"messages", "list", "--room", "4101", "--window=recent"})
	if err != nil || messages.ExitCode != 0 || !bytes.Contains(messages.Stdout, []byte("reply")) || !bytes.Contains(messages.Stdout, []byte("resolved")) {
		t.Fatalf("messages = %#v, err = %v", messages, err)
	}

	wrong, err := simulate("thread.relationships", []string{"messages", "list", "--room", "04101", "--window=recent"})
	if err != nil || wrong.ExitCode != 2 || !bytes.Contains(wrong.Stderr, []byte("preserve the exact synthetic value")) {
		t.Fatalf("wrong exact reference = %#v, err = %v", wrong, err)
	}
}

func TestSimulatorReturnsStructuredReadOnlyRecovery(t *testing.T) {
	result, err := simulate("failure.recover-not-found", []string{"--error-format=json", "messages", "show", "--room", "4101", "--message", "9999"})
	if err != nil || result.ExitCode != 6 || len(result.Stdout) != 0 {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
	var document struct {
		Error struct {
			Code        string `json:"code"`
			NextActions []struct {
				Command string `json:"command"`
			} `json:"next_actions"`
		} `json:"error"`
	}
	if err := json.Unmarshal(result.Stderr, &document); err != nil {
		t.Fatal(err)
	}
	if document.Error.Code != "chatwork_not_found" || len(document.Error.NextActions) != 1 || document.Error.NextActions[0].Command != "messages list" {
		t.Fatalf("recovery document = %#v", document)
	}
}

func TestPrepareRecordsCandidateNeutralStaticGates(t *testing.T) {
	output := filepath.Join(t.TempDir(), "evidence")
	if err := prepareArtifacts(output, baselineName, 3); err != nil {
		t.Fatal(err)
	}
	var summary struct {
		Schema  string          `json:"schema"`
		Static  bool            `json:"static_render_gates_pass"`
		Renders []renderSummary `json:"renders"`
	}
	readJSONFile(t, filepath.Join(output, "summary-input.json"), &summary)
	if summary.Schema != toolSchema || !summary.Static {
		t.Fatalf("summary = %#v, want linked renderer to pass candidate-neutral static gates", summary)
	}
	for _, render := range summary.Renders {
		if !render.Determinism || !render.StaticPass || len(render.Failures) != 0 {
			t.Errorf("render gate failed after shared contract repair: %#v", render)
		}
	}
	metrics, err := os.ReadFile(filepath.Join(output, "render-metrics.jsonl"))
	if err != nil || !bytes.Contains(metrics, []byte(`"latency_ns":`)) || !bytes.Contains(metrics, []byte(`"sha256":`)) {
		t.Fatalf("render metrics missing latency/hash: %s, %v", metrics, err)
	}
}

func TestScoreReplaysWorkflowAndSeparatesPresentationTokenDelta(t *testing.T) {
	result, err := simulate("attention.rooms", []string{"rooms", "list"})
	if err != nil {
		t.Fatal(err)
	}
	submission := runSubmission{
		Schema: runSchema, RunID: "run-1", Candidate: baselineName, SituationID: "attention.rooms", Repetition: 1,
		Agent: "synthetic-agent", Model: "synthetic-model", Commit: strings.Repeat("a", 40), WallTimeMS: 1,
		Steps:             []runStep{{EventID: "item-1", Command: "cwk rooms list", Argv: []string{"rooms", "list"}, ExitCode: result.ExitCode, ObservedOutputSHA256: hashBytes(result.Stdout), StdoutSHA256: hashBytes(result.Stdout), StderrSHA256: hashBytes(result.Stderr)}},
		Answer:            raw(`{"attention_room_refs":["4101","4102","4104"]}`),
		Usage:             tokenUsage{Prompt: 12000, Cached: 10000, Completion: 20, Total: 12020},
		PresentationProbe: presentationProbe{ProbeID: "paired-1", CandidateInputTokens: 420, ControlInputTokens: 100, InputTokenDelta: 320},
		AllowedTools:      []string{"cwk"}, ForbiddenTools: []string{},
		NonCWKTools: []string{}, ExternalProcessing: []string{},
	}
	runs := filepath.Join(t.TempDir(), "runs.jsonl")
	file, err := os.Create(runs)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(file, submission); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()

	output := filepath.Join(t.TempDir(), "scores")
	if err := scoreRuns(runs, output); err != nil {
		t.Fatal(err)
	}
	var summary scoreSummary
	readJSONFile(t, filepath.Join(output, "score-summary.json"), &summary)
	if !summary.Eligible || summary.ExactAnswerRuns != 1 || summary.TotalTokens != 12020 || summary.MedianPresentationInputTokenDelta != 320 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestCommittedProtocolAndPromptMatchScaffold(t *testing.T) {
	base := filepath.Join("..", "..", "docs", "work", "presentation-competition-1")
	prompt, err := os.ReadFile(filepath.Join(base, "agent-system-prompt.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(prompt)) != agentSystemPrompt {
		t.Fatal("committed agent system prompt differs from evaluator prompt")
	}
	var protocol struct {
		Schema string `json:"schema"`
		Status string `json:"status"`
		Unit   string `json:"unit"`
	}
	readJSONFile(t, filepath.Join(base, "benchmark-protocol.json"), &protocol)
	if protocol.Schema != "cwk-presentation-benchmark-protocol/1" || protocol.Status != "frozen" || protocol.Unit != "natural-language-agent-workflow" {
		t.Fatalf("protocol = %#v", protocol)
	}
	for _, name := range []string{"benchmark-protocol.schema.json", "run.schema.json"} {
		var document map[string]any
		readJSONFile(t, filepath.Join(base, name), &document)
		if document["$schema"] == nil || document["type"] != "object" {
			t.Fatalf("%s is not a JSON object schema", name)
		}
	}
}

func readJSONFile(t *testing.T, path string, target any) {
	t.Helper()
	value, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(value, target); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}
