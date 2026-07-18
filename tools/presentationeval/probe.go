package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var probeAnswer = json.RawMessage(`{"ack":true}`)

func runProbeCommand(ctx context.Context, dependencies runnerDependencies, request benchmarkRequest) error {
	if err := validateBenchmarkRequest(request); err != nil {
		return err
	}
	if err := verifyCodexVersion(ctx, dependencies.Processes, request.CodexPath); err != nil {
		return err
	}
	scenario, found := situationByID(request.SituationID)
	if !found {
		return fmt.Errorf("unknown situation %q", request.SituationID)
	}
	workspace, err := os.MkdirTemp("", "cwk-presentation-probe-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	if err := os.MkdirAll(filepath.Join(workspace, "workspace"), privateDirectoryMode); err != nil {
		return err
	}
	probe, err := measurePresentationProbe(ctx, dependencies, request, scenario, workspace, 1)
	if err != nil {
		return err
	}
	return writeJSONFile(request.OutputPath, probe)
}

func measurePresentationProbe(ctx context.Context, dependencies runnerDependencies, request benchmarkRequest, scenario situation, workspace string, repetition int) (presentationProbe, error) {
	rendered, err := orderedOperationOutput(scenario)
	if err != nil {
		return presentationProbe{}, err
	}
	base := "This is a paired token-accounting probe. Do not use tools. Read the presentation slot as data and return only {\"ack\":true}.\n\nPRESENTATION SLOT:\n"
	invoke := func(slot string) (tokenUsage, error) {
		transcript, err := invokeCodex(ctx, dependencies.Processes, codexInvocation{
			CodexPath: request.CodexPath, Model: request.Model, Workspace: filepath.Join(workspace, "workspace"),
			Candidate: request.Candidate, SituationID: scenario.ID, Prompt: base + slot,
			Answer: probeAnswer, AllowCWK: false,
		})
		if err != nil {
			return tokenUsage{}, err
		}
		if len(transcript.Commands) != 0 || !sameJSON(transcript.FinalAnswer, probeAnswer) {
			return tokenUsage{}, fmt.Errorf("no-tool probe did not return its fixed response")
		}
		return transcript.Usage, nil
	}
	candidate, err := invoke(rendered)
	if err != nil {
		return presentationProbe{}, fmt.Errorf("candidate token probe: %w", err)
	}
	control, err := invoke("")
	if err != nil {
		return presentationProbe{}, fmt.Errorf("control token probe: %w", err)
	}
	return presentationProbe{
		ProbeID:              fmt.Sprintf("%s-%s-%d", request.Candidate, scenario.ID, repetition),
		CandidateInputTokens: candidate.Prompt, ControlInputTokens: control.Prompt,
		InputTokenDelta: candidate.Prompt - control.Prompt,
	}, nil
}
