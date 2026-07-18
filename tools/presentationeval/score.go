package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

func scoreRuns(runsPath, outputDirectory string) error {
	if err := createEmptyDirectory(outputDirectory); err != nil {
		return err
	}
	input, err := os.Open(runsPath)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.OpenFile(filepath.Join(outputDirectory, "runs.scored.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer output.Close()
	writer := bufio.NewWriter(output)

	var scored []scoredRun
	seen := make(map[string]struct{})
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 4*1024*1024)
	for line := 1; scanner.Scan(); line++ {
		if strings.TrimSpace(scanner.Text()) == "" {
			return fmt.Errorf("runs line %d is empty", line)
		}
		var submission runSubmission
		decoder := json.NewDecoder(strings.NewReader(scanner.Text()))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&submission); err != nil {
			return fmt.Errorf("runs line %d: %w", line, err)
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			return fmt.Errorf("runs line %d has trailing JSON", line)
		}
		if _, exists := seen[submission.RunID]; exists {
			return fmt.Errorf("runs line %d duplicates run_id %q", line, submission.RunID)
		}
		seen[submission.RunID] = struct{}{}
		value, err := scoreSubmission(submission)
		if err != nil {
			return fmt.Errorf("runs line %d: %w", line, err)
		}
		if err := writeJSON(writer, value); err != nil {
			return err
		}
		scored = append(scored, value)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(scored) == 0 {
		return fmt.Errorf("runs file contains no submissions")
	}
	if err := writer.Flush(); err != nil {
		return err
	}

	summary := summarizeScores(scored)
	return writeJSONFile(filepath.Join(outputDirectory, "score-summary.json"), summary)
}

func scoreSubmission(submission runSubmission) (scoredRun, error) {
	if submission.Schema != runSchema {
		return scoredRun{}, fmt.Errorf("schema must be %q", runSchema)
	}
	if submission.RunID == "" || submission.Agent == "" || submission.Model == "" || len(submission.Commit) != 40 || submission.WallTimeMS < 0 {
		return scoredRun{}, fmt.Errorf("run_id, agent, model, full commit, and non-negative wall time are required")
	}
	if err := validateCandidate(submission.Candidate); err != nil {
		return scoredRun{}, err
	}
	if submission.Repetition < 1 {
		return scoredRun{}, fmt.Errorf("repetition must be positive")
	}
	if submission.FailureCode != "" && submission.FailureCode != "agent_run_failed" && submission.FailureCode != "transcript_replay_failed" {
		return scoredRun{}, fmt.Errorf("unknown failure_code %q", submission.FailureCode)
	}
	scenario, found := situationByID(submission.SituationID)
	if !found {
		return scoredRun{}, fmt.Errorf("unknown situation %q", submission.SituationID)
	}

	value := scoredRun{
		Schema: scoreSchema, RunID: submission.RunID, Candidate: submission.Candidate,
		SituationID: submission.SituationID, Repetition: submission.Repetition, FailureCode: submission.FailureCode,
		CommandCount: len(submission.Steps), Tokens: submission.Usage,
		PresentationProbe: submission.PresentationProbe,
	}
	paths := make([]string, 0, len(submission.Steps))
	transcriptPass := submission.FailureCode == ""
	if submission.FailureCode != "" {
		value.Violations = append(value.Violations, "recorded runner failure: "+submission.FailureCode)
	}
	for index, step := range submission.Steps {
		parsedArgv, parseErr := parseCWKCommand(step.Command)
		if step.EventID == "" || parseErr != nil || !reflect.DeepEqual(parsedArgv, step.Argv) || len(step.Argv) == 0 {
			value.Violations = append(value.Violations, fmt.Sprintf("step %d has no cwk argv", index+1))
			transcriptPass = false
			continue
		}
		result, err := simulate(scenario.ID, step.Argv)
		if err != nil {
			return scoredRun{}, err
		}
		paths = append(paths, result.Path)
		if strings.HasPrefix(result.Path, "help") {
			value.DiscoveryCalls++
		}
		combined := append(append([]byte{}, result.Stdout...), result.Stderr...)
		if step.ExitCode != result.ExitCode || step.ObservedOutputSHA256 != hashBytes(combined) || step.StdoutSHA256 != hashBytes(result.Stdout) || step.StderrSHA256 != hashBytes(result.Stderr) {
			value.Violations = append(value.Violations, fmt.Sprintf("step %d transcript does not match deterministic simulator replay", index+1))
			transcriptPass = false
		}
	}
	value.TranscriptPass = transcriptPass

	value.WorkflowPass = orderedPaths(paths, scenario.RequiredPaths) && len(submission.Steps) <= scenario.MaxCommands
	if !value.WorkflowPass {
		value.Violations = append(value.Violations, "required cwk workflow was missing, out of order, or exceeded the command budget")
	}
	for _, forbidden := range scenario.ForbiddenPaths {
		if containsPath(paths, forbidden) {
			value.WorkflowPass = false
			value.Violations = append(value.Violations, "forbidden command executed: "+forbidden)
		}
	}
	value.ExtraExploration = extraExploration(paths, scenario.RequiredPaths)

	value.ReferenceReusePass = validateReferenceFlows(submission.Steps, paths, scenario.ReferenceFlows)
	if !value.ReferenceReusePass {
		value.Violations = append(value.Violations, "canonical reference was not reused unchanged from producer to consumer")
	}

	value.UsagePass = validateUsage(submission, &value.Violations)
	if submission.FailureCode != "" {
		value.UsagePass = false
	}
	var expected, actual any
	if err := json.Unmarshal(scenario.AnswerKey, &expected); err != nil {
		return scoredRun{}, err
	}
	if len(submission.Answer) == 0 || !json.Valid(submission.Answer) {
		value.Violations = append(value.Violations, "answer is missing or malformed JSON")
	} else if err := json.Unmarshal(submission.Answer, &actual); err != nil {
		value.Violations = append(value.Violations, "answer is malformed JSON")
	}
	value.ExactAnswer = reflect.DeepEqual(expected, actual)
	value.FieldCorrect, value.FieldTotal = countCorrectLeaves(expected, actual)
	value.CriticalPass = criticalPathsPass(expected, actual, scenario.CriticalPaths) && value.WorkflowPass && value.ReferenceReusePass && value.UsagePass && value.TranscriptPass
	if !value.ExactAnswer {
		value.Violations = append(value.Violations, "answer does not exactly match the presentation-independent key")
	}
	sort.Strings(value.Violations)
	return value, nil
}

func validateUsage(submission runSubmission, violations *[]string) bool {
	pass := true
	if !reflect.DeepEqual(submission.AllowedTools, []string{"cwk"}) || len(submission.ForbiddenTools) != 0 {
		*violations = append(*violations, "allowed/forbidden tool record is inconsistent")
		pass = false
	}
	if len(submission.NonCWKTools) != 0 {
		*violations = append(*violations, "non-cwk tool use: "+strings.Join(submission.NonCWKTools, ","))
		pass = false
	}
	if len(submission.ExternalProcessing) != 0 {
		*violations = append(*violations, "external processing: "+strings.Join(submission.ExternalProcessing, ","))
		pass = false
	}
	u := submission.Usage
	if u.Prompt < 0 || u.Cached < 0 || u.CacheWrite < 0 || u.Completion < 0 || u.Reasoning < 0 || u.Total < 0 || u.Cached > u.Prompt || u.Total != u.Prompt+u.Completion {
		*violations = append(*violations, "end-to-end token usage is inconsistent")
		pass = false
	}
	p := submission.PresentationProbe
	if p.ProbeID == "" || p.CandidateInputTokens < 0 || p.ControlInputTokens < 0 || p.InputTokenDelta != p.CandidateInputTokens-p.ControlInputTokens {
		*violations = append(*violations, "paired model-authoritative presentation token probe is missing or inconsistent")
		pass = false
	}
	return pass
}

func orderedPaths(got, required []string) bool {
	index := 0
	for _, path := range got {
		if index < len(required) && path == required[index] {
			index++
		}
	}
	return index == len(required)
}

func containsPath(paths []string, wanted string) bool {
	for _, path := range paths {
		if path == wanted {
			return true
		}
	}
	return false
}

func extraExploration(paths, required []string) int {
	requiredCounts := make(map[string]int)
	for _, path := range required {
		requiredCounts[path]++
	}
	extra, help := 0, 0
	for _, path := range paths {
		if strings.HasPrefix(path, "help") {
			help++
			continue
		}
		if requiredCounts[path] > 0 {
			requiredCounts[path]--
		} else {
			extra++
		}
	}
	if help > 2 {
		extra += help - 2
	}
	return extra
}

func validateReferenceFlows(steps []runStep, paths []string, flows []referenceFlow) bool {
	for _, flow := range flows {
		producer := -1
		for index, path := range paths {
			if producer < 0 && path == flow.ProducerPath {
				producer = index
				continue
			}
			if producer >= 0 && path == flow.ConsumerPath {
				if flagValue(steps[index].Argv, flow.InputFlag) == flow.Value {
					producer = -2
					break
				}
			}
		}
		if producer != -2 {
			return false
		}
	}
	return true
}

func flagValue(argv []string, name string) string {
	for index, value := range argv {
		if strings.HasPrefix(value, name+"=") {
			return strings.TrimPrefix(value, name+"=")
		}
		if value == name && index+1 < len(argv) {
			return argv[index+1]
		}
	}
	return ""
}

func countCorrectLeaves(expected, actual any) (int, int) {
	switch value := expected.(type) {
	case map[string]any:
		actualMap, _ := actual.(map[string]any)
		correct, total := 0, 0
		for key, child := range value {
			childCorrect, childTotal := countCorrectLeaves(child, actualMap[key])
			correct += childCorrect
			total += childTotal
		}
		return correct, total
	case []any:
		actualSlice, _ := actual.([]any)
		correct, total := 0, 0
		for index, child := range value {
			var actualChild any
			if index < len(actualSlice) {
				actualChild = actualSlice[index]
			}
			childCorrect, childTotal := countCorrectLeaves(child, actualChild)
			correct += childCorrect
			total += childTotal
		}
		return correct, total
	default:
		if reflect.DeepEqual(expected, actual) {
			return 1, 1
		}
		return 0, 1
	}
}

func criticalPathsPass(expected, actual any, paths []string) bool {
	for _, path := range paths {
		expectedValue, expectedOK := pointerValue(expected, path)
		actualValue, actualOK := pointerValue(actual, path)
		if !expectedOK || !actualOK || !reflect.DeepEqual(expectedValue, actualValue) {
			return false
		}
	}
	return true
}

func pointerValue(value any, pointer string) (any, bool) {
	if pointer == "" {
		return value, true
	}
	current := value
	for _, token := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		switch typed := current.(type) {
		case map[string]any:
			var found bool
			current, found = typed[token]
			if !found {
				return nil, false
			}
		case []any:
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, false
			}
			current = typed[index]
		default:
			return nil, false
		}
	}
	return current, true
}

func summarizeScores(values []scoredRun) scoreSummary {
	summary := scoreSummary{Schema: scoreSchema, RunCount: len(values), Eligible: true}
	deltas := make([]int64, 0, len(values))
	for _, value := range values {
		if value.ExactAnswer {
			summary.ExactAnswerRuns++
		}
		if value.CriticalPass {
			summary.CriticalPassRuns++
		}
		if value.WorkflowPass {
			summary.WorkflowPassRuns++
		}
		if value.UsagePass {
			summary.UsagePassRuns++
		}
		summary.TotalTokens += value.Tokens.Total
		deltas = append(deltas, value.PresentationProbe.InputTokenDelta)
	}
	sort.Slice(deltas, func(i, j int) bool { return deltas[i] < deltas[j] })
	summary.MedianPresentationInputTokenDelta = quantile(deltas, 0.5)
	if summary.ExactAnswerRuns != summary.RunCount {
		summary.Reasons = append(summary.Reasons, "not every run had an exact semantic answer")
	}
	if summary.CriticalPassRuns != summary.RunCount {
		summary.Reasons = append(summary.Reasons, "not every run passed critical semantic and workflow gates")
	}
	if summary.UsagePassRuns != summary.RunCount {
		summary.Reasons = append(summary.Reasons, "tool or token-usage violations were present")
	}
	summary.Eligible = len(summary.Reasons) == 0
	return summary
}
