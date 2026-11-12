//go:build readiness_eval

package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

const readinessFixtureSHA256 = "cbb62f210aea42633731853bdabcbc52d888a24b34dc2f16d0f46ee15a3c482a"

var readinessAllowedCommands = map[string]string{
	"cwk --help":                    "help",
	"cwk messages resources --help": "help",
	"cwk files show --help":         "help",
	"cwk room-tasks show --help":    "help",
	"cwk messages resources --room 8173 --message 921004 --kind all": "task",
	"cwk files show --room 8173 --file 44501":                        "task",
	"cwk room-tasks show --room 8173 --task 66301":                   "task",
}

type readinessAudit struct {
	EventCounts        map[string]int          `json:"event_counts"`
	ThreadID           string                  `json:"thread_id"`
	ToolCalls          []readinessToolCall     `json:"tool_calls"`
	HelpCalls          int                     `json:"help_calls"`
	TaskCalls          int                     `json:"task_calls"`
	ProviderCalls      []readinessProviderCall `json:"provider_calls"`
	ProviderWriteCalls int                     `json:"provider_write_calls"`
	NonCWKCalls        int                     `json:"non_cwk_calls"`
	Usage              readinessTokenUsage     `json:"usage"`
	AnswerFacts        []readinessAnswerFact   `json:"answer_facts"`
	AnswerScore        int                     `json:"answer_score"`
	AnswerPossible     int                     `json:"answer_possible"`
	FinalAnswer        string                  `json:"final_answer"`
}

type readinessToolCall struct {
	Command        string   `json:"command"`
	Argv           []string `json:"argv"`
	ExitCode       int      `json:"exit_code"`
	DurationMillis int64    `json:"duration_ms"`
	StdoutBytes    int      `json:"stdout_bytes"`
	StderrBytes    int      `json:"stderr_bytes"`
	StdoutSHA256   string   `json:"stdout_sha256"`
	StderrSHA256   string   `json:"stderr_sha256"`
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

type readinessAnswerFact struct {
	Name    string `json:"name"`
	Present bool   `json:"present"`
}

type readinessTokenUsage struct {
	InputTokens       int64 `json:"input_tokens"`
	CachedInputTokens int64 `json:"cached_input_tokens"`
	OutputTokens      int64 `json:"output_tokens"`
	ReasoningTokens   int64 `json:"reasoning_output_tokens"`
}

type readinessCodexEvent struct {
	Type     string               `json:"type"`
	ThreadID string               `json:"thread_id"`
	Item     *readinessCodexItem  `json:"item"`
	Usage    *readinessTokenUsage `json:"usage"`
}

type readinessCodexItem struct {
	ID               string `json:"id"`
	Type             string `json:"type"`
	Command          string `json:"command"`
	AggregatedOutput string `json:"aggregated_output"`
	ExitCode         *int   `json:"exit_code"`
	Status           string `json:"status"`
	Text             string `json:"text"`
}

type readinessInvocationEvent struct {
	Type           string   `json:"type"`
	Argv           []string `json:"argv"`
	ExitCode       int      `json:"exit_code"`
	Stdout         string   `json:"stdout"`
	Stderr         string   `json:"stderr"`
	DurationMillis int64    `json:"duration_ms"`
	FixtureSHA256  string   `json:"fixture_sha256"`
	Authentication struct {
		Calls int `json:"calls"`
	} `json:"authentication"`
	ProviderCalls []readinessProviderCall `json:"provider_calls"`
	Writes        int                     `json:"writes"`
}

type readinessCompletedCommand struct {
	command string
	output  string
	exit    int
}

// auditReadinessRun fails closed on unrecognized Codex event/item types,
// non-cwk commands, unexpected CLI argv, missing journal entries, or any
// mutation/provider call that falls outside this synthetic read scenario.
// The trace does not establish whether workspace writes occurred, and keyword
// matches do not verify the meaning of the final answer.
func auditReadinessRun(traceJSONL, journalJSONL []byte) (readinessAudit, error) {
	result, err := auditReadinessTrace(traceJSONL, journalJSONL)
	if err != nil {
		return result, err
	}
	return result, fmt.Errorf("semantic answer and workspace writes are not independently verified; evidence is inconclusive")
}

func auditReadinessTrace(traceJSONL, journalJSONL []byte) (readinessAudit, error) {
	trace, err := parseReadinessTrace(traceJSONL)
	if err != nil {
		return readinessAudit{}, err
	}
	journal, err := parseReadinessJournal(journalJSONL)
	if err != nil {
		return readinessAudit{}, err
	}
	if len(trace.commands) != len(journal) {
		return readinessAudit{}, fmt.Errorf("Codex command count %d differs from fixture journal %d", len(trace.commands), len(journal))
	}
	if len(trace.commands) > 7 || len(journal) > 7 {
		return readinessAudit{}, fmt.Errorf("total invocation budget exceeded: got %d, limit 7", len(trace.commands))
	}
	if len(trace.messages) == 0 {
		return readinessAudit{}, fmt.Errorf("Codex trace has no agent answer")
	}
	if len(trace.commands) == 0 || trace.commands[0].command != "cwk --help" {
		return readinessAudit{}, fmt.Errorf("first CLI command must be cwk --help")
	}

	result := readinessAudit{
		EventCounts: trace.eventCounts,
		ThreadID:    trace.threadID,
		Usage:       trace.usage,
		FinalAnswer: trace.messages[len(trace.messages)-1],
	}
	if result.ThreadID == "" {
		return readinessAudit{}, fmt.Errorf("Codex trace has no thread ID")
	}
	if result.Usage.InputTokens <= 0 || result.Usage.CachedInputTokens < 0 || result.Usage.OutputTokens < 0 || result.Usage.ReasoningTokens < 0 {
		return readinessAudit{}, fmt.Errorf("Codex trace has invalid token usage: %+v", result.Usage)
	}

	helpSeen := make(map[string]bool)
	taskSeen := make(map[string]bool)
	commandIndex := make(map[string]int, len(trace.commands))
	for index, command := range trace.commands {
		commandIndex[command.command] = index
		kind, allowed := readinessAllowedCommands[command.command]
		if !allowed {
			return readinessAudit{}, fmt.Errorf("undeclared cwk command: %q", command.command)
		}
		args := strings.Fields(command.command)
		if len(args) == 0 || args[0] != "cwk" {
			result.NonCWKCalls++
			return readinessAudit{}, fmt.Errorf("non-cwk shell command: %q", command.command)
		}
		if kind == "help" {
			result.HelpCalls++
			helpSeen[command.command] = true
		} else {
			result.TaskCalls++
			taskSeen[command.command] = true
		}
		call := journal[index]
		if strings.Join(append([]string{"cwk"}, call.Argv...), " ") != command.command {
			return readinessAudit{}, fmt.Errorf("argv mismatch at call %d: trace=%q journal=%v", index+1, command.command, call.Argv)
		}
		if command.output != call.Stdout {
			return readinessAudit{}, fmt.Errorf("stdout mismatch at call %d: %q", index+1, command.command)
		}
		if command.exit != call.ExitCode || call.ExitCode != 0 || call.Stderr != "" {
			return readinessAudit{}, fmt.Errorf("CLI call failed or wrote stderr: %q", command.command)
		}
		if call.FixtureSHA256 != readinessFixtureSHA256 {
			return readinessAudit{}, fmt.Errorf("fixture digest mismatch at call %d: %q", index+1, call.FixtureSHA256)
		}
		if call.Writes != 0 {
			return readinessAudit{}, fmt.Errorf("fixture journal recorded a write at call %d", index+1)
		}
		if kind == "help" {
			if call.Authentication.Calls != 0 || len(call.ProviderCalls) != 0 {
				return readinessAudit{}, fmt.Errorf("help invocation reached authentication or provider: %q", command.command)
			}
			if command.command != "cwk --help" && len(call.Stdout) > 2048 {
				return readinessAudit{}, fmt.Errorf("exact help exceeded 2048 bytes: %q", command.command)
			}
		} else {
			if call.Authentication.Calls != 1 {
				return readinessAudit{}, fmt.Errorf("task invocation authenticated %d times: %q", call.Authentication.Calls, command.command)
			}
		}
		stdoutHash := sha256.Sum256([]byte(call.Stdout))
		stderrHash := sha256.Sum256([]byte(call.Stderr))
		result.ToolCalls = append(result.ToolCalls, readinessToolCall{
			Command:        command.command,
			Argv:           append([]string(nil), call.Argv...),
			ExitCode:       call.ExitCode,
			DurationMillis: call.DurationMillis,
			StdoutBytes:    len([]byte(call.Stdout)),
			StderrBytes:    len([]byte(call.Stderr)),
			StdoutSHA256:   hex.EncodeToString(stdoutHash[:]),
			StderrSHA256:   hex.EncodeToString(stderrHash[:]),
		})
		result.ProviderCalls = append(result.ProviderCalls, call.ProviderCalls...)
		result.ProviderWriteCalls += call.Writes
	}
	if result.HelpCalls != 4 || result.TaskCalls != 3 {
		return readinessAudit{}, fmt.Errorf("help/task budgets require four help and three task calls; got %d/%d", result.HelpCalls, result.TaskCalls)
	}
	for _, command := range []string{
		"cwk messages resources --help",
		"cwk files show --help",
		"cwk room-tasks show --help",
	} {
		if !helpSeen[command] {
			return readinessAudit{}, fmt.Errorf("required command discovery help is missing: %q", command)
		}
	}
	for _, command := range []string{
		"cwk messages resources --room 8173 --message 921004 --kind all",
		"cwk files show --room 8173 --file 44501",
		"cwk room-tasks show --room 8173 --task 66301",
	} {
		if !taskSeen[command] {
			return readinessAudit{}, fmt.Errorf("required exact task invocation is missing: %q", command)
		}
	}
	for help, task := range map[string]string{
		"cwk messages resources --help": "cwk messages resources --room 8173 --message 921004 --kind all",
		"cwk files show --help":         "cwk files show --room 8173 --file 44501",
		"cwk room-tasks show --help":    "cwk room-tasks show --room 8173 --task 66301",
	} {
		if commandIndex[help] >= commandIndex[task] {
			return readinessAudit{}, fmt.Errorf("command discovery help %q did not precede task %q", help, task)
		}
	}
	if !strings.Contains(trace.commands[0].output, "cwk messages resources --help") ||
		!strings.Contains(trace.commands[0].output, "cwk files show --help") ||
		!strings.Contains(trace.commands[0].output, "cwk room-tasks show --help") {
		return readinessAudit{}, fmt.Errorf("root help did not expose all task paths")
	}
	if result.ProviderWriteCalls != 0 || len(result.ProviderCalls) != 4 {
		return readinessAudit{}, fmt.Errorf("provider budget mismatch: reads=%d writes=%d", len(result.ProviderCalls), result.ProviderWriteCalls)
	}
	if result.ProviderCalls[0].Task != "files.list" || result.ProviderCalls[1].Task != "room-tasks.list" {
		return readinessAudit{}, fmt.Errorf("resource reads were not files-before-room-tasks: %+v", result.ProviderCalls[:2])
	}
	providerTasks := make(map[string]int)
	for _, call := range result.ProviderCalls {
		if call.Operation != "read" {
			return readinessAudit{}, fmt.Errorf("non-read provider operation observed: %+v", call)
		}
		providerTasks[call.Task]++
	}
	for task, count := range map[string]int{"files.list": 1, "room-tasks.list": 1, "files.show": 1, "room-tasks.show": 1} {
		if providerTasks[task] != count {
			return readinessAudit{}, fmt.Errorf("provider task %s count=%d want=%d", task, providerTasks[task], count)
		}
	}
	resourceCall := journalCall(journal, "cwk messages resources --room 8173 --message 921004 --kind all")
	fileCall := journalCall(journal, "cwk files show --room 8173 --file 44501")
	taskCall := journalCall(journal, "cwk room-tasks show --room 8173 --task 66301")
	if resourceCall == nil || fileCall == nil || taskCall == nil {
		return readinessAudit{}, fmt.Errorf("required invocation journal entry is absent")
	}
	if !providerCallsMatch(resourceCall.ProviderCalls,
		readinessProviderCall{Operation: "read", Task: "files.list", RoomRef: "8173"},
		readinessProviderCall{Operation: "read", Task: "room-tasks.list", RoomRef: "8173"},
	) {
		return readinessAudit{}, fmt.Errorf("resource invocation provider calls do not match the bounded source reads: %+v", resourceCall.ProviderCalls)
	}
	if !providerCallsMatch(fileCall.ProviderCalls,
		readinessProviderCall{Operation: "read", Task: "files.show", RoomRef: "8173", FileRef: "44501"},
	) {
		return readinessAudit{}, fmt.Errorf("file show did not use the exact discovered canonical reference: %+v", fileCall.ProviderCalls)
	}
	if !providerCallsMatch(taskCall.ProviderCalls,
		readinessProviderCall{Operation: "read", Task: "room-tasks.show", RoomRef: "8173", TaskRef: "66301"},
	) {
		return readinessAudit{}, fmt.Errorf("room-task show did not use the exact discovered canonical reference: %+v", taskCall.ProviderCalls)
	}
	for _, want := range []string{
		"room-ref=8173 message-ref=921004 kind=all",
		"files source-count=2 limit=100 complete=false absence=unknown matched=1",
		"file-ref=44501",
		"room-tasks source-count=2 limit=100 complete=false absence=unknown matched=1",
		"task-ref=66301",
	} {
		if !strings.Contains(resourceCall.Stdout, want) {
			return readinessAudit{}, fmt.Errorf("resource result omitted %q", want)
		}
	}
	for _, want := range []string{"file-ref=44501", "room-ref=8173", "message-ref=921004", "公開前確認表.pdf"} {
		if !strings.Contains(fileCall.Stdout, want) {
			return readinessAudit{}, fmt.Errorf("file show result omitted %q", want)
		}
	}
	for _, want := range []string{"task-ref=66301", "room-ref=8173", "message-ref=921004", "staging 確認後にリリース候補を公開する", "status=\"open\""} {
		if !strings.Contains(taskCall.Stdout, want) {
			return readinessAudit{}, fmt.Errorf("room-task show result omitted %q", want)
		}
	}
	result.AnswerFacts = scoreReadinessAnswer(result.FinalAnswer)
	for _, fact := range result.AnswerFacts {
		if fact.Present {
			result.AnswerScore++
		}
	}
	result.AnswerPossible = len(result.AnswerFacts)
	if result.AnswerScore != result.AnswerPossible {
		return result, fmt.Errorf("semantic answer score %d/%d", result.AnswerScore, result.AnswerPossible)
	}
	return result, nil
}

func providerCallsMatch(actual []readinessProviderCall, expected ...readinessProviderCall) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index, want := range expected {
		got := actual[index]
		if got.Operation != want.Operation || got.Task != want.Task || got.RoomRef != want.RoomRef || got.MessageRef != want.MessageRef || got.FileRef != want.FileRef || got.TaskRef != want.TaskRef || got.Kind != want.Kind {
			return false
		}
	}
	return true
}

type parsedReadinessTrace struct {
	eventCounts map[string]int
	threadID    string
	commands    []readinessCompletedCommand
	messages    []string
	usage       readinessTokenUsage
}

func parseReadinessTrace(data []byte) (parsedReadinessTrace, error) {
	parsed := parsedReadinessTrace{eventCounts: make(map[string]int)}
	started := make(map[string]string)
	completed := make(map[string]bool)
	itemIDs := make(map[string]bool)
	threadCount, turnStartedCount, turnCompletedCount := 0, 0, 0
	threadStarted := false
	turnOpen := false
	openCommandID := ""
	lastCompletedItemType := ""
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			return parsedReadinessTrace{}, fmt.Errorf("blank Codex JSONL line %d", lineNumber)
		}
		var event readinessCodexEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return parsedReadinessTrace{}, fmt.Errorf("invalid Codex JSONL line %d: %w", lineNumber, err)
		}
		if event.ThreadID != "" && parsed.threadID != "" && event.ThreadID != parsed.threadID {
			return parsedReadinessTrace{}, fmt.Errorf("Codex thread changed at line %d", lineNumber)
		}
		parsed.eventCounts[event.Type]++
		switch event.Type {
		case "thread.started":
			threadCount++
			if threadStarted || lineNumber != 1 || event.ThreadID == "" {
				return parsedReadinessTrace{}, fmt.Errorf("thread.started lacks thread_id")
			}
			threadStarted = true
			parsed.threadID = event.ThreadID
		case "turn.started":
			if !threadStarted || turnOpen || turnCompletedCount != 0 {
				return parsedReadinessTrace{}, fmt.Errorf("unexpected turn.started at line %d", lineNumber)
			}
			turnStartedCount++
			turnOpen = true
		case "turn.completed":
			if !turnOpen || openCommandID != "" || lastCompletedItemType != "agent_message" {
				return parsedReadinessTrace{}, fmt.Errorf("unexpected turn.completed at line %d", lineNumber)
			}
			turnCompletedCount++
			if event.Usage == nil {
				return parsedReadinessTrace{}, fmt.Errorf("turn.completed lacks token usage")
			}
			parsed.usage = *event.Usage
			turnOpen = false
		case "item.started", "item.completed":
			if !turnOpen {
				return parsedReadinessTrace{}, fmt.Errorf("item event outside the active turn at line %d", lineNumber)
			}
			if event.Item == nil {
				return parsedReadinessTrace{}, fmt.Errorf("%s lacks item at line %d", event.Type, lineNumber)
			}
			switch event.Item.Type {
			case "command_execution":
				if event.Item.ID == "" || event.Item.Command == "" {
					return parsedReadinessTrace{}, fmt.Errorf("command_execution lacks identity at line %d", lineNumber)
				}
				if event.Type == "item.started" {
					if openCommandID != "" {
						return parsedReadinessTrace{}, fmt.Errorf("overlapping command executions %q and %q", openCommandID, event.Item.ID)
					}
					if itemIDs[event.Item.ID] {
						return parsedReadinessTrace{}, fmt.Errorf("duplicate command start %q", event.Item.ID)
					}
					itemIDs[event.Item.ID] = true
					started[event.Item.ID] = event.Item.Command
					openCommandID = event.Item.ID
					continue
				}
				if openCommandID != event.Item.ID || completed[event.Item.ID] || started[event.Item.ID] != event.Item.Command {
					return parsedReadinessTrace{}, fmt.Errorf("command lifecycle mismatch %q", event.Item.ID)
				}
				if event.Item.ExitCode == nil || *event.Item.ExitCode != 0 || event.Item.Status != "completed" {
					return parsedReadinessTrace{}, fmt.Errorf("command did not complete successfully: %q", event.Item.Command)
				}
				command, err := parseReadinessShellCommand(event.Item.Command)
				if err != nil {
					return parsedReadinessTrace{}, err
				}
				completed[event.Item.ID] = true
				parsed.commands = append(parsed.commands, readinessCompletedCommand{command: command, output: event.Item.AggregatedOutput, exit: *event.Item.ExitCode})
				openCommandID = ""
				lastCompletedItemType = "command_execution"
			case "agent_message":
				if event.Type != "item.completed" || openCommandID != "" || event.Item.ID == "" || event.Item.Text == "" || itemIDs[event.Item.ID] {
					return parsedReadinessTrace{}, fmt.Errorf("unexpected agent_message lifecycle event at line %d", lineNumber)
				}
				itemIDs[event.Item.ID] = true
				parsed.messages = append(parsed.messages, event.Item.Text)
				lastCompletedItemType = "agent_message"
			default:
				return parsedReadinessTrace{}, fmt.Errorf("unknown Codex item type %q at line %d", event.Item.Type, lineNumber)
			}
		default:
			return parsedReadinessTrace{}, fmt.Errorf("unknown Codex event type %q at line %d", event.Type, lineNumber)
		}
	}
	if err := scanner.Err(); err != nil {
		return parsedReadinessTrace{}, fmt.Errorf("read Codex JSONL: %w", err)
	}
	if threadCount != 1 || turnStartedCount != 1 || turnCompletedCount != 1 || turnOpen {
		return parsedReadinessTrace{}, fmt.Errorf("Codex lifecycle counts are thread=%d turn-started=%d turn-completed=%d", threadCount, turnStartedCount, turnCompletedCount)
	}
	if len(started) != len(completed) || len(started) != len(parsed.commands) {
		return parsedReadinessTrace{}, fmt.Errorf("Codex trace has incomplete command lifecycle: started=%d completed=%d", len(started), len(completed))
	}
	return parsed, nil
}

func parseReadinessShellCommand(raw string) (string, error) {
	const prefix = "/bin/zsh -c '"
	if !strings.HasPrefix(raw, prefix) || !strings.HasSuffix(raw, "'") {
		return "", fmt.Errorf("unexpected Codex shell wrapper: %q", raw)
	}
	command := strings.TrimSuffix(strings.TrimPrefix(raw, prefix), "'")
	if strings.ContainsAny(command, "\r\n;&|<>`") || strings.Contains(command, "$(") {
		return "", fmt.Errorf("shell composition detected in %q", command)
	}
	if _, allowed := readinessAllowedCommands[command]; !allowed {
		return "", fmt.Errorf("undeclared shell command: %q", command)
	}
	return command, nil
}

func parseReadinessJournal(data []byte) ([]readinessInvocationEvent, error) {
	var events []readinessInvocationEvent
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 16*1024), 8*1024*1024)
	lineNumber := 0
	var started *readinessInvocationEvent
	for scanner.Scan() {
		lineNumber++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			return nil, fmt.Errorf("blank fixture journal line %d", lineNumber)
		}
		var event readinessInvocationEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return nil, fmt.Errorf("invalid fixture journal line %d: %w", lineNumber, err)
		}
		switch event.Type {
		case "cwk_invocation_started":
			if started != nil {
				return nil, errors.New("nested fixture invocation reservation")
			}
			started = &event
		case "cwk_invocation":
			if started == nil || !slices.Equal(started.Argv, event.Argv) {
				return nil, errors.New("unmatched fixture invocation completion")
			}
			events = append(events, event)
			started = nil
		default:
			return nil, fmt.Errorf("unknown fixture journal event %q", event.Type)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read fixture journal: %w", err)
	}
	if started != nil || len(events) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	return events, nil
}

func journalCall(events []readinessInvocationEvent, command string) *readinessInvocationEvent {
	for index := range events {
		if strings.Join(append([]string{"cwk"}, events[index].Argv...), " ") == command {
			return &events[index]
		}
	}
	return nil
}

func scoreReadinessAnswer(answer string) []readinessAnswerFact {
	return []readinessAnswerFact{
		{Name: "room_ref_8173", Present: strings.Contains(answer, "8173")},
		{Name: "message_ref_921004", Present: strings.Contains(answer, "921004")},
		{Name: "both_resource_categories", Present: strings.Contains(answer, "資料") && strings.Contains(answer, "ルームタスク")},
		{Name: "all_resource_kind", Present: strings.Contains(answer, "--kind all") || strings.Contains(answer, "kind=all")},
		{Name: "separate_source_and_match_counts", Present: strings.Contains(answer, "取得件数") && strings.Contains(answer, "該当数")},
		{Name: "source_limit", Present: strings.Contains(answer, "100") && strings.Contains(answer, "上限")},
		{Name: "incomplete_and_unknown", Present: strings.Contains(answer, "false") && strings.Contains(answer, "unknown")},
		{Name: "file_canonical_ref", Present: strings.Contains(answer, "44501")},
		{Name: "file_name", Present: strings.Contains(answer, "公開前確認表.pdf")},
		{Name: "task_canonical_ref", Present: strings.Contains(answer, "66301")},
		{Name: "task_body", Present: strings.Contains(answer, "staging 確認後にリリース候補を公開する")},
		{Name: "task_status", Present: strings.Contains(answer, "open")},
		{Name: "message_body_not_re_read", Present: strings.Contains(answer, "本文は読み直していません")},
	}
}
