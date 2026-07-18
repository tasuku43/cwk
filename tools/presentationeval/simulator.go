package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/tasuku43/cwk/internal/cli"
	"github.com/tasuku43/cwk/internal/cli/capsule"
)

func simulate(situationID string, argv []string) (commandResult, error) {
	scenario, found := situationByID(situationID)
	if !found {
		return commandResult{}, fmt.Errorf("unknown situation %q", situationID)
	}
	if len(argv) == 0 {
		return commandResult{}, fmt.Errorf("cwk arguments are required")
	}

	if isHelpInvocation(argv) {
		var stdout, stderr bytes.Buffer
		command := cli.New(strings.NewReader(""), &stdout, &stderr)
		exitCode := command.RunContext(context.Background(), argv)
		return commandResult{Path: helpPath(argv), ExitCode: exitCode, Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, nil
	}

	jsonErrors, commandArgs, err := stripErrorFormat(argv)
	if err != nil {
		return invalidSimulation("", err.Error(), false), nil
	}
	path, rest, operation, found := matchOperation(scenario, commandArgs)
	if !found {
		return invalidSimulation(strings.Join(commandArgs, " "), "The situation does not provide that synthetic cwk operation.", jsonErrors), nil
	}
	flags, err := parseOperationFlags(rest)
	if err != nil {
		return invalidSimulation(path, err.Error(), jsonErrors), nil
	}
	if err := validateRequiredArgs(operation.RequiredArgs, flags); err != nil {
		return invalidSimulation(path, err.Error(), jsonErrors), nil
	}

	if operation.Failure != nil {
		return renderSimulatedFailure(path, *operation.Failure, jsonErrors), nil
	}
	output, err := capsule.Render(operation.Result)
	if err != nil {
		return commandResult{}, fmt.Errorf("render %s: %w", path, err)
	}
	return commandResult{Path: path, ExitCode: 0, Stdout: []byte(output)}, nil
}

func isHelpInvocation(argv []string) bool {
	_, values, err := stripErrorFormat(argv)
	return err == nil && len(values) > 0 && (values[0] == "help" || values[0] == "--help" || values[0] == "-h")
}

func helpPath(argv []string) string {
	_, values, _ := stripErrorFormat(argv)
	if len(values) == 0 || values[0] != "help" {
		return "help"
	}
	parts := []string{"help"}
	for _, value := range values[1:] {
		if strings.HasPrefix(value, "--") {
			continue
		}
		parts = append(parts, value)
	}
	return strings.Join(parts, " ")
}

func stripErrorFormat(argv []string) (bool, []string, error) {
	values := append([]string{}, argv...)
	jsonErrors := false
	for len(values) > 0 {
		switch {
		case values[0] == "--error-format":
			if len(values) < 2 || (values[1] != "text" && values[1] != "json") {
				return false, nil, fmt.Errorf("--error-format requires text or json")
			}
			jsonErrors = values[1] == "json"
			values = values[2:]
		case strings.HasPrefix(values[0], "--error-format="):
			value := strings.TrimPrefix(values[0], "--error-format=")
			if value != "text" && value != "json" {
				return false, nil, fmt.Errorf("--error-format requires text or json")
			}
			jsonErrors = value == "json"
			values = values[1:]
		default:
			return jsonErrors, values, nil
		}
	}
	return jsonErrors, values, nil
}

func matchOperation(scenario situation, argv []string) (string, []string, fixtureOperation, bool) {
	paths := make([]string, 0, len(scenario.Operations))
	for path := range scenario.Operations {
		paths = append(paths, path)
	}
	sort.Slice(paths, func(i, j int) bool { return len(paths[i]) > len(paths[j]) })
	for _, path := range paths {
		words := strings.Fields(path)
		if len(argv) < len(words) {
			continue
		}
		matched := true
		for index := range words {
			if argv[index] != words[index] {
				matched = false
				break
			}
		}
		if matched {
			return path, argv[len(words):], scenario.Operations[path], true
		}
	}
	return "", nil, fixtureOperation{}, false
}

func parseOperationFlags(argv []string) (map[string]string, error) {
	values := make(map[string]string)
	for index := 0; index < len(argv); index++ {
		argument := argv[index]
		if !strings.HasPrefix(argument, "--") {
			return nil, fmt.Errorf("synthetic cwk accepts only declared flags after the task path")
		}
		name, value, found := strings.Cut(argument, "=")
		if !found {
			if index+1 >= len(argv) || strings.HasPrefix(argv[index+1], "--") {
				return nil, fmt.Errorf("%s requires a value", name)
			}
			index++
			value = argv[index]
		}
		if value == "" {
			return nil, fmt.Errorf("%s requires a non-empty value", name)
		}
		if _, duplicate := values[name]; duplicate {
			return nil, fmt.Errorf("%s may be supplied only once", name)
		}
		values[name] = value
	}
	return values, nil
}

func validateRequiredArgs(required, got map[string]string) error {
	if len(required) != len(got) {
		return fmt.Errorf("the synthetic operation requires exactly its reviewed flags")
	}
	for name, value := range required {
		if got[name] != value {
			return fmt.Errorf("%s must preserve the exact synthetic value %q", name, value)
		}
	}
	return nil
}

func invalidSimulation(path, message string, jsonOutput bool) commandResult {
	failure := simulatedFailure{Kind: "invalid_input", Code: "invalid_arguments", Message: message, NextCommand: "help", NextReason: "Choose a declared synthetic cwk task.", ExitCode: 2}
	return renderSimulatedFailure(path, failure, jsonOutput)
}

func renderSimulatedFailure(path string, failure simulatedFailure, jsonOutput bool) commandResult {
	type action struct {
		Command string `json:"command"`
		Reason  string `json:"reason"`
	}
	payload := struct {
		SchemaVersion int `json:"schema_version"`
		Error         struct {
			Kind        string   `json:"kind"`
			Code        string   `json:"code"`
			Message     string   `json:"message"`
			Retryable   bool     `json:"retryable"`
			RetryAfter  *string  `json:"retry_after"`
			NextActions []action `json:"next_actions"`
		} `json:"error"`
	}{SchemaVersion: 1}
	payload.Error.Kind = failure.Kind
	payload.Error.Code = failure.Code
	payload.Error.Message = failure.Message
	payload.Error.Retryable = failure.Retryable
	payload.Error.NextActions = []action{{Command: failure.NextCommand, Reason: failure.NextReason}}

	var stderr []byte
	if jsonOutput {
		stderr, _ = json.Marshal(payload)
		stderr = append(stderr, '\n')
	} else {
		stderr = []byte(fmt.Sprintf("error: %s\nkind: %s\ncode: %s\nretryable: %t\nretry_after: none\nnext_action: cwk %s — %s\n", failure.Message, failure.Kind, failure.Code, failure.Retryable, failure.NextCommand, failure.NextReason))
	}
	return commandResult{Path: path, ExitCode: failure.ExitCode, Stderr: stderr}
}
