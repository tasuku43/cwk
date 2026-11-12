//go:build readiness_eval

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "run" {
		if err := runCodexReadiness(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "readiness runner inconclusive:", err)
			os.Exit(1)
		}
		return
	}
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "audit" {
		args = args[1:]
	}
	flags := flag.NewFlagSet("readinessrunner", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	tracePath := flags.String("trace", "", "Codex JSONL trace path")
	journalPath := flags.String("journal", "", "synthetic cwk invocation journal path")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "readiness audit arguments:", boundedReadinessDiagnostic(err.Error(), 1024))
		os.Exit(2)
	}
	if *tracePath == "" || *journalPath == "" || flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: go run -tags readiness_eval ./tools/readinessrunner audit --trace PATH --journal PATH")
		fmt.Fprintln(os.Stderr, "   or: go run -tags readiness_eval ./tools/readinessrunner run --repo-root PATH --evidence-dir PATH")
		os.Exit(2)
	}
	traceJSONL, err := readReadinessArtifact(*tracePath, readinessTraceByteLimit)
	if err != nil {
		fatal(err)
	}
	journalJSONL, err := readReadinessArtifact(*journalPath, readinessJournalByteLimit)
	if err != nil {
		fatal(err)
	}
	report, err := auditReadinessRun(traceJSONL, journalJSONL)
	if err != nil {
		fatal(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "readiness audit inconclusive:", boundedReadinessDiagnostic(err.Error(), 1024))
	os.Exit(1)
}

func boundedReadinessDiagnostic(value string, limit int) string {
	var result strings.Builder
	for _, r := range value {
		if !unicode.IsPrint(r) {
			r = '?'
		}
		if result.Len()+utf8.RuneLen(r) > limit {
			break
		}
		result.WriteRune(r)
	}
	return result.String()
}
