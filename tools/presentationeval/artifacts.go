package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/tasuku43/cwk/internal/cli/capsule"
)

type renderMeasurement struct {
	Schema      string `json:"schema"`
	Candidate   string `json:"candidate"`
	SituationID string `json:"situation_id"`
	CommandPath string `json:"command_path"`
	Iteration   int    `json:"iteration"`
	Bytes       int    `json:"bytes"`
	SHA256      string `json:"sha256"`
	LatencyNS   int64  `json:"latency_ns"`
}

type renderSummary struct {
	SituationID string   `json:"situation_id"`
	CommandPath string   `json:"command_path"`
	Bytes       int      `json:"bytes"`
	SHA256      string   `json:"sha256"`
	MedianNS    int64    `json:"median_latency_ns"`
	P95NS       int64    `json:"p95_latency_ns"`
	Determinism bool     `json:"deterministic_bytes"`
	StaticPass  bool     `json:"static_render_gate"`
	Failures    []string `json:"failures"`
}

func prepareArtifacts(outputDirectory, candidate string, iterations int) error {
	if err := validateCandidate(candidate); err != nil {
		return err
	}
	if err := createEmptyDirectory(outputDirectory); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(outputDirectory, "rendered", candidate), 0o755); err != nil {
		return err
	}

	metricsFile, err := os.OpenFile(filepath.Join(outputDirectory, "render-metrics.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer metricsFile.Close()
	metrics := bufio.NewWriter(metricsFile)

	var summaries []renderSummary
	for _, scenario := range situations() {
		paths := make([]string, 0, len(scenario.Operations))
		for path := range scenario.Operations {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			operation := scenario.Operations[path]
			if operation.Failure != nil {
				continue
			}
			summary, rawMetrics, output, renderErr := measureRender(candidate, scenario.ID, operation, iterations)
			if renderErr != nil {
				return renderErr
			}
			filename := artifactName(scenario.ID, path) + ".txt"
			if err := os.WriteFile(filepath.Join(outputDirectory, "rendered", candidate, filename), output, 0o644); err != nil {
				return err
			}
			for _, measurement := range rawMetrics {
				if err := writeJSON(metrics, measurement); err != nil {
					return err
				}
			}
			summaries = append(summaries, summary)
		}
	}
	if err := metrics.Flush(); err != nil {
		return err
	}

	manifest := map[string]any{
		"schema":                 toolSchema,
		"candidate":              candidate,
		"candidate_schema":       candidateSchema(candidate),
		"fixture_source":         "synthetic typed chatwork.Result values",
		"network_allowed":        false,
		"credential_input":       false,
		"runner_invocation":      "go run ./tools/presentationeval run --candidate <label> --scenario <id> --codex <absolute-path> --model <id> --out <runs.jsonl>",
		"situation_count":        len(situations()),
		"render_iterations":      iterations,
		"go_version":             runtime.Version(),
		"goos":                   runtime.GOOS,
		"goarch":                 runtime.GOARCH,
		"system_prompt_sha256":   hashBytes([]byte(agentSystemPrompt)),
		"natural_language_unit":  true,
		"agent_chooses_commands": true,
	}
	prompts := map[string]any{"schema": toolSchema, "system": agentSystemPrompt, "situations": publicSituations()}
	keys := make([]map[string]any, 0, len(situations()))
	for _, scenario := range situations() {
		var answer any
		if err := json.Unmarshal(scenario.AnswerKey, &answer); err != nil {
			return fmt.Errorf("answer key %s: %w", scenario.ID, err)
		}
		keys = append(keys, map[string]any{
			"id": scenario.ID, "answer": answer, "critical_paths": scenario.CriticalPaths,
			"required_paths": scenario.RequiredPaths, "forbidden_paths": scenario.ForbiddenPaths,
			"max_commands": scenario.MaxCommands,
		})
	}
	staticPass := true
	for _, summary := range summaries {
		staticPass = staticPass && summary.Determinism && summary.StaticPass
	}
	summaryInput := map[string]any{
		"schema": toolSchema, "candidate": candidate, "render_count": len(summaries),
		"static_render_gates_pass": staticPass, "renders": summaries,
	}

	for name, value := range map[string]any{
		"manifest.json": manifest, "prompts.json": prompts,
		"answer-keys.json":   map[string]any{"schema": toolSchema, "situations": keys},
		"summary-input.json": summaryInput,
	} {
		if err := writeJSONFile(filepath.Join(outputDirectory, name), value); err != nil {
			return err
		}
	}
	return nil
}

func measureRender(candidate, situationID string, operation fixtureOperation, iterations int) (renderSummary, []renderMeasurement, []byte, error) {
	latencies := make([]int64, 0, iterations)
	metrics := make([]renderMeasurement, 0, iterations)
	var first []byte
	deterministic := true
	for iteration := 1; iteration <= iterations; iteration++ {
		started := time.Now()
		text, err := capsule.Render(operation.Result)
		latency := time.Since(started).Nanoseconds()
		if err != nil {
			return renderSummary{}, nil, nil, fmt.Errorf("render %s %s: %w", situationID, operation.Path, err)
		}
		output := []byte(text)
		if first == nil {
			first = append([]byte{}, output...)
		} else if string(first) != string(output) {
			deterministic = false
		}
		latencies = append(latencies, latency)
		metrics = append(metrics, renderMeasurement{
			Schema: toolSchema, Candidate: candidate, SituationID: situationID, CommandPath: operation.Path,
			Iteration: iteration, Bytes: len(output), SHA256: hashBytes(output), LatencyNS: latency,
		})
	}

	failures := renderGateFailures(first)
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	return renderSummary{
		SituationID: situationID, CommandPath: operation.Path, Bytes: len(first), SHA256: hashBytes(first),
		MedianNS: quantile(latencies, 0.50), P95NS: quantile(latencies, 0.95), Determinism: deterministic,
		StaticPass: len(failures) == 0, Failures: failures,
	}, metrics, first, nil
}

func renderGateFailures(output []byte) []string {
	var failures []string
	text := string(output)
	for _, r := range text {
		if r == '\n' {
			continue
		}
		if unicode.Is(unicode.C, r) || r == '\u2028' || r == '\u2029' {
			failures = append(failures, fmt.Sprintf("raw structural rune U+%04X", r))
			break
		}
	}
	return failures
}

func quantile(sorted []int64, q float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1) * q)
	return sorted[index]
}

func artifactName(situationID, path string) string {
	replacer := strings.NewReplacer(".", "-", " ", "-", "/", "-")
	return replacer.Replace(situationID + "__" + path)
}

func createEmptyDirectory(path string) error {
	info, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		return os.MkdirAll(path, 0o755)
	case err != nil:
		return err
	case !info.IsDir():
		return fmt.Errorf("output path must be a directory")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("output directory must be empty")
	}
	return nil
}

func hashBytes(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func writeJSONFile(path string, value any) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeJSON(file, value)
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}
