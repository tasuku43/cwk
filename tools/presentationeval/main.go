// Command presentationeval prepares and scores the offline presentation
// competition. It never contacts Chatwork or reads a credential.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const usage = `usage:
  presentationeval prepare --candidate <c0|p|l|r|j> --out <empty-directory> [--iterations <n>]
  presentationeval cwk --scenario <id> -- <cwk-arguments...>
  presentationeval run --candidate <c0|p|l|r|j> --scenario <id> --codex <absolute-path> --model <id> --out <runs.jsonl> [--repetitions <1..8>]
  presentationeval run-suite --candidate <c0|p|l|r|j> --codex <absolute-path> --model <id> --out <new-runs.jsonl>
  presentationeval token-probe --candidate <c0|p|l|r|j> --scenario <id> --codex <absolute-path> --model <id> --out <probe.json>
  presentationeval score --runs <runs.jsonl> --out <empty-directory>
  presentationeval situations
`

func main() {
	if filepath.Base(os.Args[0]) == "cwk" {
		os.Exit(runFixtureBinary(os.Args[1:]))
	}
	os.Exit(runWithDeps(os.Args[1:], runnerDependencies{Processes: osProcessRunner{}, Now: time.Now}))
}

func run(args []string) int {
	return runWithDeps(args, runnerDependencies{Processes: osProcessRunner{}, Now: time.Now})
}

func runWithDeps(args []string, dependencies runnerDependencies) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}

	var err error
	switch args[0] {
	case "prepare":
		set := flag.NewFlagSet("prepare", flag.ContinueOnError)
		set.SetOutput(os.Stderr)
		out := set.String("out", "", "empty artifact directory")
		candidate := set.String("candidate", baselineName, "candidate label: c0, p, l, r, or j")
		iterations := set.Int("iterations", 20, "render repetitions used for latency evidence")
		if parseErr := set.Parse(args[1:]); parseErr != nil {
			return 2
		}
		if set.NArg() != 0 || *out == "" || *iterations < 1 || *iterations > 10000 {
			fmt.Fprint(os.Stderr, usage)
			return 2
		}
		err = prepareArtifacts(*out, *candidate, *iterations)
	case "cwk":
		set := flag.NewFlagSet("cwk", flag.ContinueOnError)
		set.SetOutput(os.Stderr)
		scenario := set.String("scenario", "", "benchmark situation ID")
		if parseErr := set.Parse(args[1:]); parseErr != nil {
			return 2
		}
		if *scenario == "" || set.NArg() == 0 {
			fmt.Fprint(os.Stderr, usage)
			return 2
		}
		result, simulateErr := simulate(*scenario, set.Args())
		if simulateErr != nil {
			fmt.Fprintf(os.Stderr, "presentationeval: %v\n", simulateErr)
			return 2
		}
		_, _ = os.Stdout.Write(result.Stdout)
		_, _ = os.Stderr.Write(result.Stderr)
		return result.ExitCode
	case "run":
		set := flag.NewFlagSet("run", flag.ContinueOnError)
		set.SetOutput(os.Stderr)
		candidate := set.String("candidate", "", "candidate label")
		scenario := set.String("scenario", "", "benchmark situation ID")
		codex := set.String("codex", "", "absolute pinned codex executable path")
		model := set.String("model", "", "pinned model identifier")
		out := set.String("out", "", "JSONL run output")
		repetitions := set.Int("repetitions", 1, "finite repetition count")
		if parseErr := set.Parse(args[1:]); parseErr != nil {
			return 2
		}
		if set.NArg() != 0 || *candidate == "" || *scenario == "" || *codex == "" || *model == "" || *out == "" || *repetitions < 1 || *repetitions > 8 {
			fmt.Fprint(os.Stderr, usage)
			return 2
		}
		err = runBenchmark(context.Background(), dependencies, benchmarkRequest{Candidate: *candidate, SituationID: *scenario, CodexPath: *codex, Model: *model, OutputPath: *out, Repetitions: *repetitions})
	case "run-suite":
		set := flag.NewFlagSet("run-suite", flag.ContinueOnError)
		set.SetOutput(os.Stderr)
		candidate := set.String("candidate", "", "candidate label")
		codex := set.String("codex", "", "absolute pinned codex executable path")
		model := set.String("model", "", "pinned model identifier")
		out := set.String("out", "", "new JSONL suite output")
		if parseErr := set.Parse(args[1:]); parseErr != nil {
			return 2
		}
		if set.NArg() != 0 || *candidate == "" || *codex == "" || *model == "" || *out == "" {
			fmt.Fprint(os.Stderr, usage)
			return 2
		}
		err = runSuite(context.Background(), dependencies, benchmarkRequest{Candidate: *candidate, CodexPath: *codex, Model: *model, OutputPath: *out})
	case "token-probe":
		set := flag.NewFlagSet("token-probe", flag.ContinueOnError)
		set.SetOutput(os.Stderr)
		candidate := set.String("candidate", "", "candidate label")
		scenario := set.String("scenario", "", "benchmark situation ID")
		codex := set.String("codex", "", "absolute pinned codex executable path")
		model := set.String("model", "", "pinned model identifier")
		out := set.String("out", "", "probe JSON output")
		if parseErr := set.Parse(args[1:]); parseErr != nil {
			return 2
		}
		if set.NArg() != 0 || *candidate == "" || *scenario == "" || *codex == "" || *model == "" || *out == "" {
			fmt.Fprint(os.Stderr, usage)
			return 2
		}
		err = runProbeCommand(context.Background(), dependencies, benchmarkRequest{Candidate: *candidate, SituationID: *scenario, CodexPath: *codex, Model: *model, OutputPath: *out, Repetitions: 1})
	case "score":
		set := flag.NewFlagSet("score", flag.ContinueOnError)
		set.SetOutput(os.Stderr)
		runs := set.String("runs", "", "strict JSONL run submissions")
		out := set.String("out", "", "empty score directory")
		if parseErr := set.Parse(args[1:]); parseErr != nil {
			return 2
		}
		if set.NArg() != 0 || *runs == "" || *out == "" {
			fmt.Fprint(os.Stderr, usage)
			return 2
		}
		err = scoreRuns(*runs, *out)
	case "situations":
		if len(args) != 1 {
			fmt.Fprint(os.Stderr, usage)
			return 2
		}
		err = writeJSON(os.Stdout, publicSituations())
	default:
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "presentationeval: %v\n", err)
		return 1
	}
	return 0
}
