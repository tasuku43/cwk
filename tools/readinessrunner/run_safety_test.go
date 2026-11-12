//go:build readiness_eval

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestReadinessBoundedWriterStopsAndCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var output bytes.Buffer
	writer := &readinessBoundedWriter{output: &output, remaining: 5, cancel: cancel}
	if n, err := writer.Write([]byte("abc")); n != 3 || err != nil {
		t.Fatalf("first write = %d, %v", n, err)
	}
	if n, err := writer.Write([]byte("def")); n != 2 || err == nil {
		t.Fatalf("excess write = %d, %v", n, err)
	}
	if output.String() != "abcde" || !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("output = %q, cancellation = %v", output.String(), ctx.Err())
	}
}

func TestReadinessBoundedWriterCancelsChildProcess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var output bytes.Buffer
	cmd := readinessOutputProcess(ctx, "stdout")
	cmd.Stdout = &readinessBoundedWriter{output: &output, remaining: 1024, cancel: cancel}
	err := cmd.Run()
	if err == nil || !errors.Is(ctx.Err(), context.Canceled) || output.Len() != 1024 {
		t.Fatalf("child error=%v context=%v bytes=%d", err, ctx.Err(), output.Len())
	}
}

func TestReadinessBoundedWriterPropagatesShortWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writer := &readinessBoundedWriter{output: shortReadinessWriter{}, remaining: 10, cancel: cancel}
	if _, err := writer.Write([]byte("abc")); !errors.Is(err, io.ErrShortWrite) || !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("short write error=%v context=%v", err, ctx.Err())
	}
}

type shortReadinessWriter struct{}

func (shortReadinessWriter) Write(p []byte) (int, error) { return len(p) / 2, nil }

func TestReadReadinessArtifactEnforcesLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact")
	if err := os.WriteFile(path, []byte("abcde"), 0o600); err != nil {
		t.Fatal(err)
	}
	contents, err := readReadinessArtifact(path, 5)
	if err != nil || string(contents) != "abcde" {
		t.Fatalf("exact limit = %q, %v", contents, err)
	}
	if contents, err := readReadinessArtifact(path, 4); err == nil || contents != nil {
		t.Fatalf("oversized artifact = %q, %v", contents, err)
	}
}

func TestReadinessAuditRejectsOversizedInputs(t *testing.T) {
	if os.Getenv("CWK_READINESS_AUDIT_HELPER") == "1" {
		os.Args = []string{"readinessrunner", "audit", "--trace", os.Getenv("CWK_READINESS_TRACE"), "--journal", os.Getenv("CWK_READINESS_JOURNAL")}
		main()
		return
	}
	for _, tc := range []struct {
		name  string
		limit int64
	}{
		{name: "trace", limit: readinessTraceByteLimit},
		{name: "journal", limit: readinessJournalByteLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			trace := filepath.Join(root, "trace")
			journal := filepath.Join(root, "journal")
			for _, path := range []string{trace, journal} {
				if err := os.WriteFile(path, nil, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			path := trace
			if tc.name == "journal" {
				path = journal
			}
			if err := os.Truncate(path, tc.limit+1); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestReadinessAuditRejectsOversizedInputs$")
			cmd.Env = append(os.Environ(), "CWK_READINESS_AUDIT_HELPER=1", "CWK_READINESS_TRACE="+trace, "CWK_READINESS_JOURNAL="+journal)
			output, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(output), "byte limit exceeded") {
				t.Fatalf("audit output=%q err=%v", output, err)
			}
		})
	}
}

func TestReadinessAuditBoundsDiagnostic(t *testing.T) {
	root := t.TempDir()
	trace := filepath.Join(root, "trace")
	journal := filepath.Join(root, "journal")
	line := `{"type":"` + strings.Repeat("x", 100000) + `"}` + "\n"
	if err := os.WriteFile(trace, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(journal, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestReadinessAuditRejectsOversizedInputs$")
	cmd.Env = append(os.Environ(), "CWK_READINESS_AUDIT_HELPER=1", "CWK_READINESS_TRACE="+trace, "CWK_READINESS_JOURNAL="+journal)
	output, err := cmd.CombinedOutput()
	if err == nil || len(output) > 1200 || !strings.Contains(string(output), "unknown Codex event type") {
		t.Fatalf("audit error=%v output bytes=%d", err, len(output))
	}
}

func TestReadinessAuditBoundsFlagErrors(t *testing.T) {
	if os.Getenv("CWK_READINESS_FLAG_HELPER") == "1" {
		os.Args = []string{"readinessrunner", "audit", "--\x1b[31m" + strings.Repeat("x", 16000)}
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestReadinessAuditBoundsFlagErrors$")
	cmd.Env = append(os.Environ(), "CWK_READINESS_FLAG_HELPER=1")
	output, err := cmd.CombinedOutput()
	if err == nil || len(output) > 1200 || bytes.Contains(output, []byte{0x1b}) || !strings.Contains(string(output), "flag provided but not defined") {
		t.Fatalf("flag error=%v output bytes=%d", err, len(output))
	}
}

func TestReadinessDiagnosticNeverExceedsBudget(t *testing.T) {
	for _, limit := range []int{0, 1, 2, 3, 1024} {
		got := boundedReadinessDiagnostic(strings.Repeat("あ", 500)+"\x1b", limit)
		if len(got) > limit || strings.Contains(got, "\x1b") {
			t.Fatalf("limit=%d output bytes=%d", limit, len(got))
		}
	}
}

func TestReadinessDraftBoundsArgumentErrors(t *testing.T) {
	err := runCodexReadinessDraft([]string{"--\x1b[31m" + strings.Repeat("x", 16000)})
	if err == nil || len(err.Error()) > 1024 || strings.Contains(err.Error(), "\x1b") {
		t.Fatalf("draft argument error = %v", err)
	}
}

func TestReadinessAuxiliaryCommandsBoundOutput(t *testing.T) {
	t.Setenv("CWK_READINESS_OUTPUT_HELPER", "stdout")
	start := time.Now()
	if output, err := commandText(t.TempDir(), os.Args[0], "-test.run=^TestReadinessSyntheticOutputHelper$"); err == nil || output != "" {
		t.Fatalf("commandText error=%v output bytes=%d", err, len(output))
	}
	log := filepath.Join(t.TempDir(), "build.log")
	output, err := runCommandOutput(t.TempDir(), log, os.Args[0], "-test.run=^TestReadinessSyntheticOutputHelper$")
	if err == nil || len(output) > readinessCommandByteLimit {
		t.Fatalf("runCommandOutput error=%v output bytes=%d", err, len(output))
	}
	info, statErr := os.Stat(log)
	if statErr != nil || info.Size() > readinessCommandByteLimit {
		t.Fatalf("bounded log: info=%v err=%v", info, statErr)
	}
	if time.Since(start) > 15*time.Second {
		t.Fatal("auxiliary commands did not stop promptly")
	}
}

func TestReadinessAuxiliaryCommandStopsWaitingForInheritedPipe(t *testing.T) {
	pidPath := filepath.Join(t.TempDir(), "child.pid")
	t.Setenv("CWK_READINESS_OUTPUT_HELPER", "descendant-parent")
	t.Setenv("CWK_READINESS_DESCENDANT_PID", pidPath)
	defer func() {
		data, err := os.ReadFile(pidPath)
		if err != nil {
			return
		}
		pid, err := strconv.Atoi(string(data))
		if err == nil {
			if process, err := os.FindProcess(pid); err == nil {
				_ = process.Kill()
			}
		}
	}()
	start := time.Now()
	_, err := commandText(t.TempDir(), os.Args[0], "-test.run=^TestReadinessSyntheticOutputHelper$")
	if !errors.Is(err, exec.ErrWaitDelay) || time.Since(start) > 8*time.Second {
		t.Fatalf("inherited pipe: error=%v duration=%v", err, time.Since(start))
	}
}

func TestReadinessBoundedWriterCancelsChildOnOutputError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := readinessOutputProcess(ctx, "stdout")
	cmd.Stdout = &readinessBoundedWriter{output: failingSummaryWriter{}, remaining: 1024, cancel: cancel}
	if err := cmd.Run(); err == nil || !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("child error=%v context=%v", err, ctx.Err())
	}
}

func TestReadinessStreamsCapAndCancelStderr(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var stdout, stderr bytes.Buffer
	cmd := readinessOutputProcess(ctx, "stderr")
	setReadinessStreams(cmd, &stdout, &stderr, 1024, 100, cancel)
	if err := cmd.Run(); err == nil || !errors.Is(ctx.Err(), context.Canceled) || stdout.Len() != 0 || stderr.Len() != 100 {
		t.Fatalf("child error=%v context=%v stdout=%d stderr=%d", err, ctx.Err(), stdout.Len(), stderr.Len())
	}
}

func readinessOutputProcess(ctx context.Context, stream string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestReadinessSyntheticOutputHelper$")
	cmd.Env = append(os.Environ(), "CWK_READINESS_OUTPUT_HELPER="+stream)
	return cmd
}

func TestReadinessSyntheticOutputHelper(t *testing.T) {
	stream := os.Getenv("CWK_READINESS_OUTPUT_HELPER")
	if stream == "" {
		return
	}
	if stream == "descendant-parent" {
		child := exec.Command(os.Args[0], "-test.run=^TestReadinessSyntheticOutputHelper$")
		child.Env = append(os.Environ(), "CWK_READINESS_OUTPUT_HELPER=descendant-holder")
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err := child.Start(); err != nil {
			os.Exit(3)
		}
		if err := os.WriteFile(os.Getenv("CWK_READINESS_DESCENDANT_PID"), []byte(strconv.Itoa(child.Process.Pid)), 0o600); err != nil {
			_ = child.Process.Kill()
			os.Exit(3)
		}
		os.Exit(0)
	}
	if stream == "descendant-holder" {
		time.Sleep(10 * time.Second)
		os.Exit(0)
	}
	output := os.Stdout
	if stream == "stderr" {
		output = os.Stderr
	}
	for {
		if _, err := output.Write([]byte("0123456789")); err != nil {
			os.Exit(2)
		}
	}
}

func TestReadinessPublicRunIsDisabledBeforeExternalAction(t *testing.T) {
	if os.Getenv("CWK_READINESS_RUN_HELPER") == "1" {
		os.Args = []string{"readinessrunner", "run", "--repo-root", os.Getenv("CWK_READINESS_ROOT"), "--evidence-dir", os.Getenv("CWK_READINESS_EVIDENCE"), "--codex", os.Getenv("CWK_READINESS_CODEX")}
		main()
		return
	}
	root := t.TempDir()
	marker := filepath.Join(root, "executed")
	for _, name := range []string{"git", "go", "codex"} {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf executed > "+shellQuote(marker)+"\n"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	evidence := filepath.Join(root, "evidence")
	cmd := exec.Command(os.Args[0], "-test.run=^TestReadinessPublicRunIsDisabledBeforeExternalAction$")
	cmd.Env = append(os.Environ(), "CWK_READINESS_RUN_HELPER=1", "CWK_READINESS_ROOT="+root, "CWK_READINESS_EVIDENCE="+evidence, "CWK_READINESS_CODEX="+filepath.Join(root, "codex"), "PATH="+root)
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "disabled") {
		t.Fatalf("public run = %v, output %q; want disabled failure", err, output)
	}
	for _, path := range []string{marker, evidence} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("public run changed %q: %v", path, err)
		}
	}
}

type failingSummaryWriter struct{}

func (failingSummaryWriter) Write([]byte) (int, error) {
	return 0, errors.New("synthetic summary writer failure")
}

type partialSummaryWriter struct {
	bytes.Buffer
}

func TestReadinessCleanupFailurePreservesPrimaryFailure(t *testing.T) {
	manifest := readinessRunManifest{Failure: "helper build failed"}
	buildErr := errors.New("synthetic build failure")
	cleanupErr := errors.New("synthetic cleanup failure")
	err := recordReadinessCleanupFailure(&manifest, buildErr, cleanupErr)
	if manifest.Failure != "helper build failed" || !strings.Contains(manifest.CleanupError, "synthetic cleanup failure") || !errors.Is(err, buildErr) || !errors.Is(err, cleanupErr) {
		t.Fatalf("manifest=%+v error=%v", manifest, err)
	}
}

func (w *partialSummaryWriter) Write(p []byte) (int, error) {
	n, _ := w.Buffer.Write(p[:len(p)/2])
	return n, errors.New("synthetic partial output failure")
}

func TestReadinessRunIsDisabledBeforeAnyExternalAction(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "executed")
	binary := filepath.Join(root, "fake-codex")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\ntouch "+shellQuote(marker)+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	evidence := filepath.Join(root, "evidence")
	err := runCodexReadiness([]string{"--repo-root", root, "--evidence-dir", evidence, "--codex", binary})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("readiness run error = %v, want disabled", err)
	}
	for _, path := range []string{marker, evidence} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("disabled run changed %q: %v", path, err)
		}
	}
}

func TestCreateEvidenceDirectoryRejectsRepositoryPathsBeforeWriting(t *testing.T) {
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(root, "existing")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(parent)
	if err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(parent, "evidence")
	if _, err := createEvidenceDirectory(root, inside); err == nil {
		t.Fatal("accepted an evidence path inside the repository")
	}
	if _, err := os.Lstat(inside); !os.IsNotExist(err) {
		t.Fatalf("rejected path was created or cannot be inspected: %v", err)
	}
	after, err := os.Stat(parent)
	if err != nil || after.Mode().Perm() != before.Mode().Perm() {
		t.Fatalf("existing parent permissions changed: before=%v after=%v err=%v", before.Mode(), after, err)
	}
	if runtime.GOOS != "windows" {
		outside := t.TempDir()
		alias := filepath.Join(outside, "repository-alias")
		if err := os.Symlink(root, alias); err != nil {
			t.Fatal(err)
		}
		if _, err := createEvidenceDirectory(root, filepath.Join(alias, "evidence")); err == nil {
			t.Fatal("accepted a symlinked evidence parent inside the repository")
		}
		if _, err := os.Lstat(filepath.Join(root, "evidence")); !os.IsNotExist(err) {
			t.Fatalf("symlinked rejected path was created: %v", err)
		}
	}
}

func TestCreateEvidenceDirectoryRejectsRepositoryTempDir(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", root)
	if _, err := createEvidenceDirectory(root, ""); err == nil {
		t.Fatal("accepted a default temporary directory inside the repository")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("rejected default parent changed: %v, %v", entries, err)
	}
}

func TestCreateEvidenceDirectoryDoesNotChangeExistingDirectory(t *testing.T) {
	root := t.TempDir()
	parent := t.TempDir()
	existing := filepath.Join(parent, "existing")
	if err := os.Mkdir(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(existing)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := createEvidenceDirectory(root, existing); err == nil {
		t.Fatal("accepted an existing evidence directory")
	}
	after, err := os.Stat(existing)
	if err != nil || after.Mode().Perm() != before.Mode().Perm() {
		t.Fatalf("existing directory permissions changed: before=%v after=%v err=%v", before.Mode(), after, err)
	}
	directory, err := createEvidenceDirectory(root, filepath.Join(parent, "new"))
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(directory)
	if err != nil || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("new evidence directory is not private: %v, %v", info, err)
	}
}

func TestWriteRunSummaryDoesNotReportPassWithoutManifest(t *testing.T) {
	directory := t.TempDir()
	manifestPath := filepath.Join(directory, "run-manifest.json")
	if err := os.WriteFile(manifestPath, []byte("existing evidence"), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err := writeRunSummary(directory, readinessRunManifest{Status: "qualifying_pass"}, &output)
	if err == nil || output.Len() != 0 {
		t.Fatalf("manifest failure = %v, output = %q; want an error and no pass output", err, output.String())
	}
	contents, err := os.ReadFile(manifestPath)
	if err != nil || string(contents) != "existing evidence" {
		t.Fatalf("existing manifest changed: %q, %v", contents, err)
	}
	newDirectory := t.TempDir()
	if err := writeRunSummary(newDirectory, readinessRunManifest{Status: "inconclusive"}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"status":"inconclusive"`) {
		t.Fatalf("summary output missing status: %q", output.String())
	}
}

func TestWriteRunSummaryReturnsOutputFailureAfterSavingManifest(t *testing.T) {
	directory := t.TempDir()
	err := writeRunSummary(directory, readinessRunManifest{Status: "inconclusive"}, failingSummaryWriter{})
	if err == nil || !strings.Contains(err.Error(), "synthetic summary writer failure") {
		t.Fatalf("summary writer error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "run-manifest.json")); err != nil {
		t.Fatalf("manifest was not saved before the output failure: %v", err)
	}
}

func TestWriteRunSummaryReturnsPartialOutputFailure(t *testing.T) {
	directory := t.TempDir()
	var output partialSummaryWriter
	err := writeRunSummary(directory, readinessRunManifest{Status: "qualifying_pass"}, &output)
	if err == nil || !strings.Contains(err.Error(), "synthetic partial output failure") {
		t.Fatalf("partial output error = %v", err)
	}
	if output.Len() == 0 {
		t.Fatal("test writer did not exercise a partial write")
	}
	if _, err := os.Stat(filepath.Join(directory, "run-manifest.json")); err != nil {
		t.Fatalf("manifest was not saved: %v", err)
	}
}
