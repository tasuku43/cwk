// Command presentationeval prepares and scores the offline presentation
// competition. It never contacts Chatwork or reads a credential.
package main

import (
	"flag"
	"fmt"
	"os"
)

const usage = `usage:
  presentationeval prepare --out <empty-directory> [--iterations <n>]
  presentationeval cwk --scenario <id> -- <cwk-arguments...>
  presentationeval score --runs <runs.jsonl> --out <empty-directory>
  presentationeval situations
`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
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
		iterations := set.Int("iterations", 20, "render repetitions used for latency evidence")
		if parseErr := set.Parse(args[1:]); parseErr != nil {
			return 2
		}
		if set.NArg() != 0 || *out == "" || *iterations < 1 || *iterations > 10000 {
			fmt.Fprint(os.Stderr, usage)
			return 2
		}
		err = prepareArtifacts(*out, *iterations)
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
