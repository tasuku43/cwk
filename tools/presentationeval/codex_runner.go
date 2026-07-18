package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

type codexInvocation struct {
	CodexPath   string
	Model       string
	Workspace   string
	FixtureBin  string
	Candidate   string
	SituationID string
	Prompt      string
	Answer      json.RawMessage
	AllowCWK    bool
}

type codexTranscript struct {
	Commands       []observedCommand
	FinalAnswer    json.RawMessage
	Usage          tokenUsage
	ForbiddenTools []string
}

type observedCommand struct {
	EventID  string
	Command  string
	Argv     []string
	ExitCode int
	Output   string
}

var disabledCodexFeatures = []string{
	"apps", "browser_use", "browser_use_external", "computer_use", "image_generation",
	"in_app_browser", "multi_agent", "multi_agent_v2", "plugins", "standalone_web_search",
	"tool_suggest", "workspace_dependencies",
}

func invokeCodex(ctx context.Context, runner processRunner, invocation codexInvocation) (codexTranscript, error) {
	schema, err := answerSchema(invocation.Answer)
	if err != nil {
		return codexTranscript{}, err
	}
	temp, err := os.MkdirTemp(filepath.Dir(invocation.Workspace), "codex-call-")
	if err != nil {
		return codexTranscript{}, err
	}
	defer os.RemoveAll(temp)
	schemaPath := filepath.Join(temp, "answer.schema.json")
	lastPath := filepath.Join(temp, "last-message.json")
	if err := os.WriteFile(schemaPath, schema, 0o600); err != nil {
		return codexTranscript{}, err
	}
	args := []string{"--ask-for-approval", "never", "exec", "--sandbox", "workspace-write", "--ignore-user-config", "--ignore-rules", "--ephemeral", "--skip-git-repo-check", "--strict-config", "-C", invocation.Workspace, "--model", invocation.Model, "--json", "--output-schema", schemaPath, "--output-last-message", lastPath,
		"-c", "sandbox_workspace_write.network_access=false", "-c", "shell_environment_policy.inherit=none",
		"-c", `model_reasoning_effort="medium"`,
		"-c", configString("shell_environment_policy.set.PATH", invocation.FixtureBin),
		"-c", configString("shell_environment_policy.set."+fixtureScenarioEnvironment, invocation.SituationID),
		"-c", configString("shell_environment_policy.set."+fixtureCandidateEnvironment, invocation.Candidate),
	}
	for _, feature := range disabledCodexFeatures {
		args = append(args, "--disable", feature)
	}
	if !invocation.AllowCWK {
		args = append(args, "--disable", "shell_tool")
	}
	args = append(args, "-")
	callContext, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	response, err := runner.Run(callContext, processRequest{Path: invocation.CodexPath, Args: args, Env: benchmarkProcessEnvironment(), Stdin: []byte(invocation.Prompt), StdoutLimit: maxCodexOutputBytes, StderrLimit: 1 << 20})
	if err != nil {
		return codexTranscript{}, fmt.Errorf("invoke codex: %w", err)
	}
	if response.ExitCode != 0 {
		return codexTranscript{}, fmt.Errorf("codex exited %d: %s", response.ExitCode, strings.TrimSpace(string(response.Stderr)))
	}
	last, err := os.ReadFile(lastPath)
	if err != nil {
		return codexTranscript{}, fmt.Errorf("read final answer: %w", err)
	}
	return parseCodexJSONL(response.Stdout, last, invocation.AllowCWK)
}

func parseCodexJSONL(input, last []byte, allowCWK bool) (codexTranscript, error) {
	var transcript codexTranscript
	started := make(map[string]string)
	var lastAgent string
	seenThread, seenTurn, seenCompleted := false, false, false
	scanner := bufio.NewScanner(bytes.NewReader(input))
	scanner.Buffer(make([]byte, 4096), maxCodexOutputBytes)
	for line := 1; scanner.Scan(); line++ {
		if strings.TrimSpace(scanner.Text()) == "" {
			return codexTranscript{}, fmt.Errorf("codex JSONL line %d is empty", line)
		}
		var envelope map[string]json.RawMessage
		decoder := json.NewDecoder(strings.NewReader(scanner.Text()))
		if err := decoder.Decode(&envelope); err != nil {
			return codexTranscript{}, fmt.Errorf("codex JSONL line %d: %w", line, err)
		}
		if err := requireEOF(decoder); err != nil {
			return codexTranscript{}, fmt.Errorf("codex JSONL line %d has trailing JSON", line)
		}
		var eventType string
		if err := json.Unmarshal(envelope["type"], &eventType); err != nil || eventType == "" {
			return codexTranscript{}, fmt.Errorf("codex JSONL line %d has no event type", line)
		}
		switch eventType {
		case "thread.started":
			if err := exactKeys(envelope, "type", "thread_id"); err != nil {
				return codexTranscript{}, fmt.Errorf("codex JSONL line %d: %w", line, err)
			}
			if seenThread {
				return codexTranscript{}, fmt.Errorf("duplicate thread.started")
			}
			seenThread = true
		case "turn.started":
			if err := exactKeys(envelope, "type"); err != nil {
				return codexTranscript{}, fmt.Errorf("codex JSONL line %d: %w", line, err)
			}
			if !seenThread || seenTurn {
				return codexTranscript{}, fmt.Errorf("invalid turn.started order")
			}
			seenTurn = true
		case "item.started", "item.completed":
			if err := exactKeys(envelope, "type", "item"); err != nil {
				return codexTranscript{}, fmt.Errorf("codex JSONL line %d: %w", line, err)
			}
			if !seenTurn || seenCompleted {
				return codexTranscript{}, fmt.Errorf("item event outside active turn")
			}
			var item struct {
				ID               string `json:"id"`
				Type             string `json:"type"`
				Text             string `json:"text"`
				Command          string `json:"command"`
				AggregatedOutput string `json:"aggregated_output"`
				ExitCode         *int   `json:"exit_code"`
				Status           string `json:"status"`
			}
			if err := decodeStrict(envelope["item"], &item); err != nil {
				return codexTranscript{}, fmt.Errorf("codex JSONL line %d item: %w", line, err)
			}
			switch item.Type {
			case "reasoning":
			case "agent_message":
				if eventType == "item.completed" {
					lastAgent = item.Text
				}
			case "command_execution":
				if !allowCWK {
					return codexTranscript{}, fmt.Errorf("no-tool probe emitted a command event")
				}
				if eventType == "item.started" {
					if item.ID == "" || item.Command == "" {
						return codexTranscript{}, fmt.Errorf("incomplete command start event")
					}
					started[item.ID] = item.Command
					continue
				}
				if started[item.ID] != item.Command || item.ExitCode == nil || !validCommandCompletion(item.Status, *item.ExitCode) {
					return codexTranscript{}, fmt.Errorf("unmatched or incomplete command event %q", item.ID)
				}
				argv, err := parseCWKCommand(item.Command)
				if err != nil {
					return codexTranscript{}, err
				}
				delete(started, item.ID)
				transcript.Commands = append(transcript.Commands, observedCommand{EventID: item.ID, Command: item.Command, Argv: argv, ExitCode: *item.ExitCode, Output: item.AggregatedOutput})
			default:
				transcript.ForbiddenTools = append(transcript.ForbiddenTools, item.Type)
				return codexTranscript{}, fmt.Errorf("forbidden Codex item type %q", item.Type)
			}
		case "turn.completed":
			if err := exactKeys(envelope, "type", "usage"); err != nil {
				return codexTranscript{}, fmt.Errorf("codex JSONL line %d: %w", line, err)
			}
			if !seenTurn || seenCompleted || len(started) != 0 {
				return codexTranscript{}, fmt.Errorf("invalid turn.completed order")
			}
			var usage struct {
				Input      int64 `json:"input_tokens"`
				Cached     int64 `json:"cached_input_tokens"`
				CacheWrite int64 `json:"cache_write_input_tokens"`
				Output     int64 `json:"output_tokens"`
				Reasoning  int64 `json:"reasoning_output_tokens"`
			}
			if err := decodeStrict(envelope["usage"], &usage); err != nil {
				return codexTranscript{}, fmt.Errorf("usage: %w", err)
			}
			transcript.Usage = tokenUsage{Prompt: usage.Input, Cached: usage.Cached, CacheWrite: usage.CacheWrite, Completion: usage.Output, Reasoning: usage.Reasoning, Total: usage.Input + usage.Output}
			seenCompleted = true
		case "error", "turn.failed":
			return codexTranscript{}, fmt.Errorf("Codex emitted %s", eventType)
		default:
			return codexTranscript{}, fmt.Errorf("unknown Codex JSONL event %q", eventType)
		}
	}
	if err := scanner.Err(); err != nil {
		return codexTranscript{}, err
	}
	if !seenThread || !seenTurn || !seenCompleted || transcript.Usage.Total <= 0 {
		return codexTranscript{}, fmt.Errorf("Codex JSONL lifecycle or usage is incomplete")
	}
	last = bytes.TrimSpace(last)
	if !json.Valid(last) {
		return codexTranscript{}, fmt.Errorf("final answer is not JSON")
	}
	if strings.TrimSpace(lastAgent) != string(last) {
		return codexTranscript{}, fmt.Errorf("final answer file and agent event differ")
	}
	transcript.FinalAnswer = append(json.RawMessage(nil), last...)
	return transcript, nil
}

func validCommandCompletion(status string, exitCode int) bool {
	return status == "completed" || (status == "failed" && exitCode != 0)
}

func exactKeys(values map[string]json.RawMessage, allowed ...string) error {
	wanted := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		wanted[key] = struct{}{}
	}
	if len(values) != len(wanted) {
		return fmt.Errorf("event fields do not match the pinned Codex schema")
	}
	for key := range values {
		if _, found := wanted[key]; !found {
			return fmt.Errorf("unknown event field %q", key)
		}
	}
	return nil
}

func requireEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("trailing value")
	}
	return nil
}

func decodeStrict(value json.RawMessage, target any) error {
	if len(value) == 0 {
		return fmt.Errorf("missing value")
	}
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return requireEOF(decoder)
}

func parseCWKCommand(command string) ([]string, error) {
	if strings.ContainsAny(command, "|&;<>()$`\n\r") {
		return nil, fmt.Errorf("command event contains a forbidden shell operator")
	}
	var argv []string
	var word strings.Builder
	quote := rune(0)
	escaped, active := false, false
	flush := func() {
		if active {
			argv = append(argv, word.String())
			word.Reset()
			active = false
		}
	}
	for _, r := range command {
		if escaped {
			word.WriteRune(r)
			active = true
			escaped = false
			continue
		}
		if r == '\\' && quote != '\'' {
			escaped = true
			active = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				word.WriteRune(r)
				active = true
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			active = true
			continue
		}
		if r == ' ' || r == '\t' {
			flush()
			continue
		}
		word.WriteRune(r)
		active = true
	}
	if escaped || quote != 0 {
		return nil, fmt.Errorf("command event has incomplete quoting")
	}
	flush()
	if len(argv) >= 2 && argv[0] == "cwk" {
		return argv[1:], nil
	}
	if len(argv) == 3 && argv[1] == "-lc" && allowedShellWrapper(argv[0]) {
		return parseCWKCommand(argv[2])
	}
	return nil, fmt.Errorf("command event must contain only one cwk invocation")
}

func allowedShellWrapper(value string) bool {
	switch value {
	case "sh", "bash", "zsh", "/bin/sh", "/bin/bash", "/bin/zsh":
		return true
	default:
		return false
	}
}

func answerSchema(answer json.RawMessage) ([]byte, error) {
	var value any
	if err := json.Unmarshal(answer, &value); err != nil {
		return nil, fmt.Errorf("answer key: %w", err)
	}
	schema := schemaForValue(value)
	root, ok := schema.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("answer key must be an object")
	}
	root["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	return json.Marshal(root)
}

func schemaForValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		properties := make(map[string]any, len(typed))
		required := make([]string, 0, len(typed))
		for key, child := range typed {
			properties[key] = schemaForValue(child)
			required = append(required, key)
		}
		sortStrings(required)
		return map[string]any{"type": "object", "additionalProperties": false, "required": required, "properties": properties}
	case []any:
		var items any = map[string]any{}
		if len(typed) != 0 {
			items = schemaForValue(typed[0])
		}
		return map[string]any{"type": "array", "items": items}
	case string:
		return map[string]any{"type": "string"}
	case bool:
		return map[string]any{"type": "boolean"}
	case float64:
		return map[string]any{"type": "number"}
	case nil:
		return map[string]any{"type": "null"}
	default:
		panic(fmt.Sprintf("unsupported JSON type %T", value))
	}
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func sameJSON(left, right []byte) bool {
	var a, b any
	return json.Unmarshal(left, &a) == nil && json.Unmarshal(right, &b) == nil && reflect.DeepEqual(a, b)
}
