package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	fixtureScenarioEnvironment  = "CWK_PRESENTATION_EVAL_SCENARIO"
	fixtureCandidateEnvironment = "CWK_PRESENTATION_EVAL_CANDIDATE"
	maxCodexOutputBytes         = 16 << 20
)

var candidateNames = map[string]struct{}{"c0": {}, "p": {}, "l": {}, "r": {}, "j": {}}

type benchmarkRequest struct {
	Candidate   string
	SituationID string
	CodexPath   string
	Model       string
	OutputPath  string
	Repetitions int
}

type runnerDependencies struct {
	Processes processRunner
	Now       func() time.Time
}

type processRequest struct {
	Path        string
	Args        []string
	Dir         string
	Env         []string
	Stdin       []byte
	StdoutLimit int
	StderrLimit int
}

type processResponse struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

type processRunner interface {
	Run(context.Context, processRequest) (processResponse, error)
}

type osProcessRunner struct{}

func (osProcessRunner) Run(ctx context.Context, request processRequest) (processResponse, error) {
	command := exec.CommandContext(ctx, request.Path, request.Args...)
	command.Dir = request.Dir
	command.Env = request.Env
	command.Stdin = bytes.NewReader(request.Stdin)
	var stdout, stderr boundedBuffer
	stdout.Limit = request.StdoutLimit
	stderr.Limit = request.StderrLimit
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	response := processResponse{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), ExitCode: 0}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		response.ExitCode = exitError.ExitCode()
		err = nil
	}
	if stdout.Exceeded || stderr.Exceeded {
		return response, fmt.Errorf("subprocess output exceeded its fixed limit")
	}
	return response, err
}

type boundedBuffer struct {
	bytes.Buffer
	Limit    int
	Exceeded bool
}

func (b *boundedBuffer) Write(value []byte) (int, error) {
	if b.Limit <= 0 {
		return b.Buffer.Write(value)
	}
	remaining := b.Limit - b.Buffer.Len()
	if remaining <= 0 {
		b.Exceeded = true
		return len(value), nil
	}
	if len(value) > remaining {
		b.Exceeded = true
		_, _ = b.Buffer.Write(value[:remaining])
		return len(value), nil
	}
	return b.Buffer.Write(value)
}

func validateCandidate(candidate string) error {
	if _, found := candidateNames[candidate]; !found {
		return fmt.Errorf("candidate must be one of c0, p, l, r, or j")
	}
	return nil
}

func candidateSchema(candidate string) string {
	if candidate == baselineName {
		return baselineVersion
	}
	return "candidate-" + candidate + "-linked-renderer"
}

func runFixtureBinary(argv []string) int {
	scenario := os.Getenv(fixtureScenarioEnvironment)
	candidate := os.Getenv(fixtureCandidateEnvironment)
	if scenario == "" || validateCandidate(candidate) != nil {
		fmt.Fprintln(os.Stderr, "cwk: this fixture binary requires the trusted presentation runner environment")
		return 2
	}
	result, err := simulate(scenario, argv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cwk: %v\n", err)
		return 2
	}
	_, _ = os.Stdout.Write(result.Stdout)
	_, _ = os.Stderr.Write(result.Stderr)
	return result.ExitCode
}

func runBenchmark(ctx context.Context, dependencies runnerDependencies, request benchmarkRequest) error {
	if err := validateBenchmarkRequest(request); err != nil {
		return err
	}
	if dependencies.Processes == nil || dependencies.Now == nil {
		return fmt.Errorf("runner dependencies are required")
	}
	scenario, found := situationByID(request.SituationID)
	if !found {
		return fmt.Errorf("unknown situation %q", request.SituationID)
	}
	repository, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := verifyCodexVersion(ctx, dependencies.Processes, request.CodexPath); err != nil {
		return err
	}
	commit, err := repositoryCommit(ctx, dependencies.Processes, repository)
	if err != nil {
		return err
	}
	if err := requireCleanRepository(ctx, dependencies.Processes, repository); err != nil {
		return err
	}
	workspace, err := os.MkdirTemp("", "cwk-presentation-run-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	fixtureBin := filepath.Join(workspace, "bin")
	work := filepath.Join(workspace, "workspace")
	if err := os.MkdirAll(fixtureBin, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		return err
	}
	if err := buildFixture(ctx, dependencies.Processes, repository, filepath.Join(fixtureBin, "cwk")); err != nil {
		return err
	}
	file, err := os.OpenFile(request.OutputPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	probe, probeErr := measurePresentationProbe(ctx, dependencies, request, scenario, workspace, 1)
	if probeErr != nil {
		return probeErr
	}
	for repetition := 1; repetition <= request.Repetitions; repetition++ {
		started := dependencies.Now()
		submission := runSubmission{
			Schema: runSchema, RunID: fmt.Sprintf("%s-%s-%d-%d", request.Candidate, scenario.ID, started.UnixNano(), repetition),
			Candidate: request.Candidate, SituationID: scenario.ID, Repetition: repetition,
			Agent: pinnedCodexCLI, Model: request.Model, Commit: commit,
			PresentationProbe: probe, AllowedTools: []string{"cwk"}, ForbiddenTools: []string{},
			NonCWKTools: []string{}, ExternalProcessing: []string{},
		}
		transcript, runErr := invokeCodex(ctx, dependencies.Processes, codexInvocation{
			CodexPath: request.CodexPath, Model: request.Model, Workspace: work,
			FixtureBin: fixtureBin, Candidate: request.Candidate, SituationID: scenario.ID,
			Prompt: benchmarkPrompt(scenario), Answer: scenario.AnswerKey, AllowCWK: true,
		})
		wall := dependencies.Now().Sub(started).Milliseconds()
		submission.WallTimeMS = wall
		if runErr != nil {
			submission.FailureCode = "agent_run_failed"
			submission.Steps = []runStep{}
			submission.Answer = json.RawMessage(`{}`)
			if err := writeJSON(file, submission); err != nil {
				return err
			}
			continue
		}
		steps, err := replaySteps(scenario.ID, transcript.Commands)
		if err != nil {
			submission.FailureCode = "transcript_replay_failed"
			submission.Steps = []runStep{}
			submission.Answer = transcript.FinalAnswer
			submission.Usage = transcript.Usage
			if err := writeJSON(file, submission); err != nil {
				return err
			}
			continue
		}
		submission.Steps = steps
		submission.Answer = transcript.FinalAnswer
		submission.Usage = transcript.Usage
		submission.ForbiddenTools = nonNilStrings(transcript.ForbiddenTools)
		if err := writeJSON(file, submission); err != nil {
			return err
		}
	}
	return nil
}

func runSuite(ctx context.Context, dependencies runnerDependencies, request benchmarkRequest) error {
	if err := validateCandidate(request.Candidate); err != nil {
		return err
	}
	if request.Model == "" || request.OutputPath == "" || !filepath.IsAbs(request.CodexPath) {
		return fmt.Errorf("candidate, absolute codex path, model, and output path are required")
	}
	if info, err := os.Stat(request.OutputPath); err == nil {
		if info.IsDir() || info.Size() != 0 {
			return fmt.Errorf("suite output must be a new path or an empty file")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, scenario := range situations() {
		item := request
		item.SituationID = scenario.ID
		item.Repetitions = suiteRepetitions(scenario)
		if err := runBenchmark(ctx, dependencies, item); err != nil {
			return fmt.Errorf("suite scenario %s: %w", scenario.ID, err)
		}
	}
	return nil
}

func suiteRepetitions(scenario situation) int {
	switch scenario.ID {
	case "rooms.large-attention", "thread.relationships":
		return 2
	default:
		return 1
	}
}

func requireCleanRepository(ctx context.Context, runner processRunner, repository string) error {
	response, err := runner.Run(ctx, processRequest{Path: "git", Args: []string{"status", "--porcelain", "--untracked-files=all"}, Dir: repository, Env: os.Environ(), StdoutLimit: 1 << 20, StderrLimit: 1 << 20})
	if err != nil || response.ExitCode != 0 {
		return fmt.Errorf("inspect candidate worktree: %w", err)
	}
	if len(bytes.TrimSpace(response.Stdout)) != 0 {
		return fmt.Errorf("candidate worktree must be clean before a scored run")
	}
	return nil
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func validateBenchmarkRequest(request benchmarkRequest) error {
	if err := validateCandidate(request.Candidate); err != nil {
		return err
	}
	if request.SituationID == "" || request.Model == "" || request.OutputPath == "" {
		return fmt.Errorf("scenario, model, and output path are required")
	}
	if !filepath.IsAbs(request.CodexPath) {
		return fmt.Errorf("codex path must be absolute")
	}
	if request.Repetitions < 1 || request.Repetitions > 8 {
		return fmt.Errorf("repetitions must be between 1 and 8")
	}
	return nil
}

func verifyCodexVersion(ctx context.Context, runner processRunner, path string) error {
	response, err := runner.Run(ctx, processRequest{Path: path, Args: []string{"--version"}, Env: benchmarkProcessEnvironment(), StdoutLimit: 4096, StderrLimit: 4096})
	if err != nil {
		return fmt.Errorf("read codex version: %w", err)
	}
	if response.ExitCode != 0 || strings.TrimSpace(string(response.Stdout)) != pinnedCodexCLI {
		return fmt.Errorf("codex version must be exactly %q", pinnedCodexCLI)
	}
	return nil
}

func repositoryCommit(ctx context.Context, runner processRunner, repository string) (string, error) {
	response, err := runner.Run(ctx, processRequest{Path: "git", Args: []string{"rev-parse", "HEAD"}, Dir: repository, Env: os.Environ(), StdoutLimit: 4096, StderrLimit: 4096})
	if err != nil {
		return "", fmt.Errorf("resolve candidate commit: %w", err)
	}
	if response.ExitCode != 0 {
		return "", fmt.Errorf("resolve candidate commit: git exited %d", response.ExitCode)
	}
	commit := strings.TrimSpace(string(response.Stdout))
	if len(commit) != 40 {
		return "", fmt.Errorf("candidate commit is not a full SHA-1")
	}
	return commit, nil
}

func buildFixture(ctx context.Context, runner processRunner, repository, output string) error {
	response, err := runner.Run(ctx, processRequest{Path: "go", Args: []string{"build", "-trimpath", "-o", output, "./tools/presentationeval"}, Dir: repository, Env: os.Environ(), StdoutLimit: 1 << 20, StderrLimit: 1 << 20})
	if err != nil {
		return fmt.Errorf("build fixture cwk: %w", err)
	}
	if response.ExitCode != 0 {
		return fmt.Errorf("build fixture cwk failed: %s", strings.TrimSpace(string(response.Stderr)))
	}
	return nil
}

func replaySteps(scenarioID string, commands []observedCommand) ([]runStep, error) {
	steps := make([]runStep, 0, len(commands))
	for _, command := range commands {
		result, err := simulate(scenarioID, command.Argv)
		if err != nil {
			return nil, err
		}
		combined := append(append([]byte{}, result.Stdout...), result.Stderr...)
		if command.ExitCode != result.ExitCode || hashBytes([]byte(command.Output)) != hashBytes(combined) {
			return nil, fmt.Errorf("command event %s did not match deterministic fixture replay", command.EventID)
		}
		steps = append(steps, runStep{EventID: command.EventID, Command: command.Command, Argv: command.Argv, ExitCode: command.ExitCode, ObservedOutputSHA256: hashBytes([]byte(command.Output)), StdoutSHA256: hashBytes(result.Stdout), StderrSHA256: hashBytes(result.Stderr)})
	}
	return steps, nil
}

func benchmarkPrompt(scenario situation) string {
	return agentSystemPrompt + "\n\nUser situation:\n" + scenario.UserPrompt + "\n\nRequired final JSON shape:\n" + scenario.AnswerShape + "\n\nInvoke commands only as a direct simple `cwk ...` command with no shell operators."
}

func benchmarkProcessEnvironment() []string {
	values := os.Environ()
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.HasPrefix(value, "CWK_API_TOKEN=") || strings.HasPrefix(value, "CWK_AUTH_METHOD=") {
			continue
		}
		result = append(result, value)
	}
	return result
}

func configString(key, value string) string { return key + "=" + strconv.Quote(value) }

func orderedOperationOutput(scenario situation) (string, error) {
	paths := make([]string, 0, len(scenario.Operations))
	for path, operation := range scenario.Operations {
		if operation.Failure == nil {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	var output strings.Builder
	for _, path := range paths {
		operation := scenario.Operations[path]
		args := strings.Fields(path)
		names := make([]string, 0, len(operation.RequiredArgs))
		for name := range operation.RequiredArgs {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			args = append(args, name, operation.RequiredArgs[name])
		}
		result, err := simulate(scenario.ID, args)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&output, "TASK %s\n%s", path, result.Stdout)
	}
	return output.String(), nil
}
