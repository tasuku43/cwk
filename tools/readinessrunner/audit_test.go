//go:build readiness_eval

package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

type readinessTestCall struct {
	command  string
	output   string
	auth     int
	provider []readinessProviderCall
	writes   int
}

func TestAuditReadinessRunAcceptsTheBoundedSyntheticTranscript(t *testing.T) {
	trace, journal := readinessTestRun()
	report, err := auditReadinessTrace(trace, journal)
	if err != nil {
		t.Fatalf("auditReadinessRun(): %v (facts=%+v)", err, report.AnswerFacts)
	}
	if report.HelpCalls != 4 || report.TaskCalls != 3 || report.ProviderWriteCalls != 0 || report.AnswerScore != 13 || report.AnswerPossible != 13 {
		t.Fatalf("audit report does not reflect the pinned budgets: %+v", report)
	}
	if _, err := auditReadinessRun(trace, journal); err == nil || !strings.Contains(err.Error(), "inconclusive") {
		t.Fatalf("unverified semantic and workspace evidence must be inconclusive: %v", err)
	}
}

func TestAuditRejectsContradictoryAnswerEvenWithAllKeywords(t *testing.T) {
	trace, journal := readinessTestRun()
	trace = []byte(strings.Replace(string(trace), "本文は読み直していません", "本文は読み直していません。実際には本文を読み直しました", 1))
	report, err := auditReadinessTrace(trace, journal)
	if err != nil || report.AnswerScore != report.AnswerPossible {
		t.Fatalf("contradiction should demonstrate the limits of keyword scoring: score=%d/%d err=%v", report.AnswerScore, report.AnswerPossible, err)
	}
	if _, err := auditReadinessRun(trace, journal); err == nil {
		t.Fatal("contradictory answer was accepted")
	}
}

func TestAuditReadinessRunRejectsUnsafeOrIncompleteEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(trace, journal []byte) ([]byte, []byte)
	}{
		{
			name: "shell composition",
			mutate: func(trace, journal []byte) ([]byte, []byte) {
				return []byte(strings.ReplaceAll(string(trace), "cwk --help", "cwk --help | cat")), journal
			},
		},
		{
			name: "wrong canonical ref in argv",
			mutate: func(trace, journal []byte) ([]byte, []byte) {
				return []byte(strings.ReplaceAll(string(trace), "--file 44501", "--file 044501")), journal
			},
		},
		{
			name: "provider write",
			mutate: func(trace, journal []byte) ([]byte, []byte) {
				return trace, []byte(strings.ReplaceAll(string(journal), `"writes":0`, `"writes":1`))
			},
		},
		{
			name: "provider ref drift",
			mutate: func(trace, journal []byte) ([]byte, []byte) {
				events, err := parseReadinessJournal(journal)
				if err != nil {
					t.Fatalf("decode test journal: %v", err)
				}
				for index := range events {
					for providerIndex := range events[index].ProviderCalls {
						if events[index].ProviderCalls[providerIndex].Task == "files.show" {
							events[index].ProviderCalls[providerIndex].FileRef = "044501"
						}
					}
				}
				encoded := make([]byte, 0, len(journal))
				for _, event := range events {
					line, err := json.Marshal(event)
					if err != nil {
						t.Fatalf("encode test journal: %v", err)
					}
					start, err := json.Marshal(readinessInvocationEvent{Type: "cwk_invocation_started", Argv: event.Argv})
					if err != nil {
						t.Fatal(err)
					}
					encoded = append(encoded, start...)
					encoded = append(encoded, '\n')
					encoded = append(encoded, line...)
					encoded = append(encoded, '\n')
				}
				return trace, encoded
			},
		},
		{
			name: "unknown event",
			mutate: func(trace, journal []byte) ([]byte, []byte) {
				return []byte(strings.Replace(string(trace), `"type":"turn.completed"`, `"type":"custom.event"`, 1)), journal
			},
		},
		{
			name: "unknown item type",
			mutate: func(trace, journal []byte) ([]byte, []byte) {
				line := `{"type":"item.completed","item":{"id":"foreign-1","type":"file_change"}}`
				return []byte(strings.Replace(string(trace), `{"type":"turn.completed"`, line+"\n"+`{"type":"turn.completed"}`, 1)), journal
			},
		},
		{
			name: "cancelled command",
			mutate: func(trace, journal []byte) ([]byte, []byte) {
				return []byte(strings.Replace(string(trace), `"status":"completed"`, `"status":"cancelled"`, 1)), journal
			},
		},
		{
			name: "answer omits required facts",
			mutate: func(trace, journal []byte) ([]byte, []byte) {
				return []byte(strings.ReplaceAll(string(trace), "本文は読み直していません", "本文についての記載なし")), journal
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			trace, journal := readinessTestRun()
			trace, journal = test.mutate(trace, journal)
			if _, err := auditReadinessTrace(trace, journal); err == nil {
				t.Fatal("unsafe or incomplete evidence unexpectedly passed")
			}
		})
	}
}

func TestParseReadinessShellCommandRejectsNonCWKAndComposition(t *testing.T) {
	for _, raw := range []string{
		"/bin/zsh -c 'jq .'",
		"/bin/zsh -c 'cat /etc/passwd'",
		"/bin/zsh -c 'cwk --help; cat /etc/passwd'",
		"/bin/zsh -c 'cwk --help | jq .'",
		"/bin/zsh -c 'cwk config show'",
		"/bin/bash -c 'cwk --help'",
	} {
		if _, err := parseReadinessShellCommand(raw); err == nil {
			t.Errorf("parseReadinessShellCommand(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestParseReadinessTraceRejectsOverlappingCommands(t *testing.T) {
	trace := strings.Join([]string{
		`{"type":"thread.started","thread_id":"synthetic-thread"}`,
		`{"type":"turn.started"}`,
		`{"type":"item.started","item":{"id":"cmd-1","type":"command_execution","command":"/bin/zsh -c 'cwk --help'"}}`,
		`{"type":"item.started","item":{"id":"cmd-2","type":"command_execution","command":"/bin/zsh -c 'cwk --help'"}}`,
		`{"type":"turn.completed","usage":{"input_tokens":10}}`,
	}, "\n") + "\n"
	if _, err := parseReadinessTrace([]byte(trace)); err == nil {
		t.Fatal("overlapping command executions unexpectedly succeeded")
	}
}

func TestReadinessJournalRejectsUnknownEvents(t *testing.T) {
	if _, err := parseReadinessJournal([]byte("{\"type\":\"filesystem_write\"}\n")); err == nil {
		t.Fatal("unknown journal event unexpectedly succeeded")
	}
}

func TestAuditReadinessRunRejectsInvocationBudgetOverflow(t *testing.T) {
	trace, journal := readinessTestRun()
	traceLines := strings.Split(strings.TrimSuffix(string(trace), "\n"), "\n")
	duplicateCommand := strings.ReplaceAll(strings.Join(traceLines[2:4], "\n"), "cmd-1", "cmd-8")
	traceLines = append(append(traceLines[:len(traceLines)-2], duplicateCommand), traceLines[len(traceLines)-2:]...)
	journalLines := strings.Split(strings.TrimSuffix(string(journal), "\n"), "\n")
	journalLines = append(journalLines, journalLines[0])
	if _, err := auditReadinessRun([]byte(strings.Join(traceLines, "\n")+"\n"), []byte(strings.Join(journalLines, "\n")+"\n")); err == nil {
		t.Fatal("eighth CLI invocation unexpectedly passed")
	}
}

func TestParseReadinessTraceRejectsMalformedJSON(t *testing.T) {
	if _, err := parseReadinessTrace([]byte("not-json\n")); err == nil {
		t.Fatal("malformed Codex event unexpectedly parsed")
	}
}

func TestCodexReadinessArgsPinModelAndIsolateShellEnvironment(t *testing.T) {
	args := codexReadinessArgs("/tmp/workspace", "/tmp/workspace/home", "/tmp/fixture-bin")
	joined := strings.Join(args, " ")
	for _, required := range []string{
		"--ephemeral",
		"--ignore-user-config",
		"--ignore-rules",
		"--sandbox workspace-write",
		"--model " + readinessModel,
		`model_reasoning_effort="max"`,
		"sandbox_workspace_write.network_access=false",
		`shell_environment_policy.inherit="none"`,
		`shell_environment_policy.include_only=["HOME","PATH"]`,
		"/tmp/fixture-bin",
	} {
		if !strings.Contains(joined, required) {
			t.Errorf("Codex args omit %q: %s", required, joined)
		}
	}
	if args[len(args)-1] != "-" {
		t.Fatalf("prompt should be read from stdin, final arg=%q", args[len(args)-1])
	}
}

func TestReadinessTreeHashDetectsWorkingTreeChanges(t *testing.T) {
	root := t.TempDir()
	if output, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (%s)", err, output)
	}
	path := filepath.Join(root, "fixture.go")
	if err := os.WriteFile(path, []byte("package fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := readinessTreeSHA256(root)
	if err != nil {
		t.Fatalf("hash first tree: %v", err)
	}
	if err := os.WriteFile(path, []byte("package fixture\n// changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := readinessTreeSHA256(root)
	if err != nil {
		t.Fatalf("hash changed tree: %v", err)
	}
	if first == second {
		t.Fatal("working tree content change did not change the tree hash")
	}
}

func TestReadinessTreeCopyPreservesIgnoredFilesAndSeparatesChanges(t *testing.T) {
	source := t.TempDir()
	if output, err := exec.Command("git", "init", source).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (%s)", err, output)
	}
	if err := os.WriteFile(filepath.Join(source, ".gitignore"), []byte("ignored/\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(source, "ignored"), 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(source, "ignored", "fixture.go")
	if err := os.WriteFile(path, []byte("package fixture\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	destination := readinessPrivateDirectory(t)
	snapshot, err := os.OpenRoot(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	want, err := copyAndHashReadinessTree(source, snapshot, readinessTreeEntryLimit, readinessTreeByteLimit)
	if err != nil {
		t.Fatal(err)
	}
	got, err := readinessTreeSHA256(destination)
	if err != nil || got != want {
		t.Fatalf("copy hash = %s, want %s: %v", got, want, err)
	}
	if _, err := os.Lstat(filepath.Join(destination, ".git")); !os.IsNotExist(err) {
		t.Fatalf("Git metadata unexpectedly copied: %v", err)
	}
	if err := os.WriteFile(path, []byte("package changed\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(destination, "ignored", "fixture.go"))
	if err != nil || string(contents) != "package fixture\n" {
		t.Fatalf("copy changed with source: %q, %v", contents, err)
	}
	after, err := readinessTreeSHA256(destination)
	if err != nil || after != want {
		t.Fatalf("copy hash changed: %s, %v", after, err)
	}
}

func TestReadinessSnapshotBuildUsesCopyAndCleansUp(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "go.mod"), []byte("module example.invalid/fixture\n\ngo 1.26.10\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	packageDir := filepath.Join(source, "internal", "cli")
	if err := os.MkdirAll(packageDir, 0o700); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(packageDir, "fixture_test.go")
	if err := os.WriteFile(fixture, []byte("package cli\nimport \"testing\"\nfunc TestSnapshotValue(t *testing.T) {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	evidence := readinessPrivateDirectory(t)
	snapshot, err := createReadinessSnapshot(source, evidence)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = snapshot.Close() })
	if err := os.WriteFile(fixture, []byte("this is not Go source"), 0o600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(evidence, "fixture.test")
	log := filepath.Join(evidence, "build.log")
	output, err := runCommandOutput(snapshot.Path, log, "go", "test", "-buildvcs=false", "-c", "-o", binary, "./internal/cli")
	if err != nil {
		t.Fatalf("build from source copy: %v (%s)", err, output)
	}
	if output, err := exec.Command(binary, "-test.run=^TestSnapshotValue$").CombinedOutput(); err != nil {
		t.Fatalf("run copied fixture: %v (%s)", err, output)
	}
	if err := snapshot.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(snapshot.Path); !os.IsNotExist(err) {
		t.Fatalf("snapshot remains after cleanup: %v", err)
	}
}

func TestReadinessSnapshotRejectsExistingDirectoryWithoutChangingIt(t *testing.T) {
	source := t.TempDir()
	evidence := readinessPrivateDirectory(t)
	existing := filepath.Join(evidence, "source-tree")
	if err := os.Mkdir(existing, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(existing, "marker")
	if err := os.WriteFile(marker, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := createReadinessSnapshot(source, evidence); err == nil {
		t.Fatal("accepted existing snapshot directory")
	}
	if contents, err := os.ReadFile(marker); err != nil || string(contents) != "keep" {
		t.Fatalf("existing snapshot changed: %q, %v", contents, err)
	}
}

func TestReadinessSnapshotCleansPartialCopyAfterFailure(t *testing.T) {
	source := t.TempDir()
	evidence := readinessPrivateDirectory(t)
	if err := os.WriteFile(filepath.Join(source, "a-valid"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(source, "z-link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := createReadinessSnapshot(source, evidence); err == nil {
		t.Fatal("accepted source symlink")
	}
	if _, err := os.Lstat(filepath.Join(evidence, "source-tree")); !os.IsNotExist(err) {
		t.Fatalf("partial snapshot remains: %v", err)
	}
	if contents, err := os.ReadFile(filepath.Join(source, "a-valid")); err != nil || string(contents) != "fixture" {
		t.Fatalf("source changed after rejection: %q, %v", contents, err)
	}
}

func TestReadinessSnapshotCleansReadOnlyDirectories(t *testing.T) {
	source := t.TempDir()
	dir := filepath.Join(source, "readonly")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	snapshot, err := createReadinessSnapshot(source, readinessPrivateDirectory(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := snapshot.Close(); err != nil {
		t.Fatalf("cleanup read-only directory: %v", err)
	}
	if _, err := os.Lstat(snapshot.Path); !os.IsNotExist(err) {
		t.Fatalf("snapshot remains: %v", err)
	}
}

func TestReadinessSnapshotCleanupDoesNotDeleteReplacement(t *testing.T) {
	source, evidence := t.TempDir(), readinessPrivateDirectory(t)
	if err := os.WriteFile(filepath.Join(source, "file"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := createReadinessSnapshot(source, evidence)
	if err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(evidence, "moved-snapshot")
	if err := os.Rename(snapshot.Path, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(snapshot.Path, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(snapshot.Path, "marker")
	if err := os.WriteFile(marker, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.Close(); err == nil {
		t.Fatal("accepted a replaced snapshot directory")
	}
	if contents, err := os.ReadFile(marker); err != nil || string(contents) != "preserve" {
		t.Fatalf("replacement was changed: %q, %v", contents, err)
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(snapshot.Path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(moved, snapshot.Path); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.Close(); err != nil {
		t.Fatalf("retry cleanup: %v", err)
	}
}

func TestReadinessSnapshotWithoutRootDoesNotRemoveReplacement(t *testing.T) {
	evidence := readinessPrivateDirectory(t)
	marker := filepath.Join(evidence, "source-tree")
	if err := os.WriteFile(marker, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	parent, err := os.OpenRoot(evidence)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := &readinessSnapshot{Path: marker, parent: parent}
	if err := snapshot.Close(); err == nil {
		t.Fatal("cleanup accepted unknown identity")
	}
	if snapshot.parent != nil {
		t.Fatal("unusable parent handle was retained")
	}
	if _, err := parent.Stat("."); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("parent handle remains open: %v", err)
	}
	if contents, err := os.ReadFile(marker); err != nil || string(contents) != "preserve" {
		t.Fatalf("replacement changed: %q, %v", contents, err)
	}
}

func TestReadinessSnapshotCreationFailureWithoutIdentityIsNotRetryable(t *testing.T) {
	evidence := readinessPrivateDirectory(t)
	parent, err := os.OpenRoot(evidence)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := &readinessSnapshot{Path: filepath.Join(evidence, "source-tree"), parent: parent}
	cause := errors.New("synthetic open failure")
	retained, err := cleanupReadinessSnapshotAfterCreateFailure(snapshot, cause)
	var creationErr *readinessSnapshotCreationError
	if retained != nil || !errors.As(err, &creationErr) || !errors.Is(err, cause) || !strings.Contains(creationErr.cleanup.Error(), "cannot be retried") {
		t.Fatalf("retained=%v err=%v", retained != nil, err)
	}
	if _, err := parent.Stat("."); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("parent handle remains open: %v", err)
	}
}

func TestReadinessSnapshotReleasesBothHandlesAfterRemovalEvenOnCloseError(t *testing.T) {
	parent, err := os.OpenRoot(readinessPrivateDirectory(t))
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(readinessPrivateDirectory(t))
	if err != nil {
		parent.Close()
		t.Fatal(err)
	}
	snapshot := &readinessSnapshot{parent: parent, root: root}
	rootCloseError := errors.New("synthetic root close failure")
	err = finishReadinessSnapshotHandles(snapshot, func() error {
		_ = root.Close()
		return rootCloseError
	}, parent.Close)
	if !errors.Is(err, rootCloseError) || snapshot.root != nil || snapshot.parent != nil {
		t.Fatalf("close error=%v snapshot=%+v", err, snapshot)
	}
	if _, err := parent.Stat("."); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("parent handle remains open: %v", err)
	}
}

func TestReadinessSnapshotCreationFailureKeepsCleanupRetry(t *testing.T) {
	source, evidence := t.TempDir(), readinessPrivateDirectory(t)
	if err := os.WriteFile(filepath.Join(source, "file"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := createReadinessSnapshot(source, evidence)
	if err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(evidence, "moved-snapshot")
	if err := os.Rename(snapshot.Path, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(snapshot.Path, 0o700); err != nil {
		t.Fatal(err)
	}
	cause := errors.New("synthetic copy failure")
	retained, err := cleanupReadinessSnapshotAfterCreateFailure(snapshot, cause)
	var creationErr *readinessSnapshotCreationError
	if retained != snapshot || !errors.As(err, &creationErr) || !errors.Is(err, cause) || creationErr.cleanup == nil {
		t.Fatalf("retained=%v err=%v", retained != nil, err)
	}
	if err := os.Remove(snapshot.Path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(moved, snapshot.Path); err != nil {
		t.Fatal(err)
	}
	if err := retained.Close(); err != nil {
		t.Fatalf("retry cleanup: %v", err)
	}
}

func TestReadinessSnapshotCreateReturnsCleanupFailureAndHandle(t *testing.T) {
	source, evidence := t.TempDir(), readinessPrivateDirectory(t)
	path := filepath.Join(evidence, "source-tree")
	moved := filepath.Join(evidence, "moved-snapshot")
	cause := errors.New("synthetic copy failure")
	snapshot, err := createReadinessSnapshotWithCopier(source, evidence, func(_ string, _ *os.Root) (string, error) {
		if err := os.Rename(path, moved); err != nil {
			return "", err
		}
		if err := os.Mkdir(path, 0o700); err != nil {
			return "", err
		}
		return "", cause
	})
	var creationErr *readinessSnapshotCreationError
	if snapshot == nil || !errors.As(err, &creationErr) || !errors.Is(err, cause) || creationErr.cleanup == nil {
		t.Fatalf("snapshot retained=%v err=%v", snapshot != nil, err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(moved, path); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.Close(); err != nil {
		t.Fatalf("retry cleanup: %v", err)
	}
}

func TestReadinessSnapshotRejectsNestedGitMetadata(t *testing.T) {
	source, evidence := t.TempDir(), readinessPrivateDirectory(t)
	metadata := filepath.Join(source, "nested", ".git")
	if err := os.MkdirAll(metadata, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metadata, "config"), []byte("private fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := createReadinessSnapshot(source, evidence); err == nil || !strings.Contains(err.Error(), "nested Git") {
		t.Fatalf("nested Git error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(evidence, "source-tree")); !os.IsNotExist(err) {
		t.Fatalf("partial snapshot remains: %v", err)
	}
}

func TestReadinessTreeCopyEnforcesLimits(t *testing.T) {
	for _, tc := range []struct {
		name       string
		entryLimit int
		byteLimit  int64
		want       string
	}{
		{name: "entries", entryLimit: 1, byteLimit: 100, want: "entry limit"},
		{name: "bytes", entryLimit: 10, byteLimit: 1, want: "byte limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := t.TempDir()
			for _, name := range []string{"first", "second"} {
				if err := os.WriteFile(filepath.Join(source, name), []byte("data"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			snapshot, err := os.OpenRoot(readinessPrivateDirectory(t))
			if err != nil {
				t.Fatal(err)
			}
			defer snapshot.Close()
			if _, err := copyAndHashReadinessTree(source, snapshot, tc.entryLimit, tc.byteLimit); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("copy error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestReadinessTreeCopyRejectsUnsafeDestinations(t *testing.T) {
	source := t.TempDir()
	stale := readinessPrivateDirectory(t)
	if err := os.WriteFile(filepath.Join(stale, "stale"), []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(source, "nested")
	if err := os.Mkdir(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "nonempty", path: stale},
		{name: "same", path: source},
		{name: "nested", path: nested},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot, err := os.OpenRoot(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			defer snapshot.Close()
			if _, err := copyAndHashReadinessTree(source, snapshot, readinessTreeEntryLimit, readinessTreeByteLimit); err == nil {
				t.Fatalf("accepted %s destination", tc.name)
			}
		})
	}
	contents, err := os.ReadFile(filepath.Join(stale, "stale"))
	if err != nil || string(contents) != "preserve" {
		t.Fatalf("existing destination changed: %q, %v", contents, err)
	}
}

func TestReadinessTreeCopyRejectsNonprivateDestination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits do not validate Windows ACLs")
	}
	destination := t.TempDir()
	if err := os.Chmod(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	snapshot, err := os.OpenRoot(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	if _, err := copyAndHashReadinessTree(t.TempDir(), snapshot, readinessTreeEntryLimit, readinessTreeByteLimit); err == nil {
		t.Fatal("accepted nonprivate destination")
	}
}

func TestReadinessTreeCopyRejectsUnsupportedEntries(t *testing.T) {
	for _, kind := range []string{"link", "socket", "special-mode"} {
		t.Run(kind, func(t *testing.T) {
			source := t.TempDir()
			if kind == "socket" {
				source = readinessShortSocketDirectory(t)
			}
			path := filepath.Join(source, "entry")
			switch kind {
			case "link":
				if err := os.Symlink(t.TempDir(), path); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			case "socket":
				listener, err := net.Listen("unix", path)
				if err != nil {
					t.Fatalf("create Unix socket: %v", err)
				}
				defer listener.Close()
			case "special-mode":
				if err := os.WriteFile(path, nil, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(path, 0o600|os.ModeSetuid); err != nil {
					t.Skipf("setuid mode unavailable: %v", err)
				}
				info, err := os.Stat(path)
				if err != nil || info.Mode()&os.ModeSetuid == 0 {
					t.Skip("filesystem does not retain setuid mode")
				}
			}
			snapshot, err := os.OpenRoot(readinessPrivateDirectory(t))
			if err != nil {
				t.Fatal(err)
			}
			defer snapshot.Close()
			if _, err := copyAndHashReadinessTree(source, snapshot, readinessTreeEntryLimit, readinessTreeByteLimit); err == nil {
				t.Fatalf("copy accepted %s", kind)
			}
		})
	}
}

func readinessPrivateDirectory(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("private snapshot ACL verification is unavailable on Windows")
	}
	path := t.TempDir()
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadinessTreeHashIncludesIgnoredFilesAndDirectories(t *testing.T) {
	root := t.TempDir()
	if output, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (%s)", err, output)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("ignored/\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ignored := filepath.Join(root, "ignored")
	if err := os.Mkdir(ignored, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(ignored, "fixture.go")
	if err := os.WriteFile(path, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	check := exec.Command("git", "check-ignore", "ignored/fixture.go")
	check.Dir = root
	if output, err := check.CombinedOutput(); err != nil {
		t.Fatalf("fixture must be git-ignored: %v (%s)", err, output)
	}
	first, err := readinessTreeSHA256(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("other"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := readinessTreeSHA256(root)
	if err != nil || first == second {
		t.Fatalf("ignored content change not detected: first=%s second=%s err=%v", first, second, err)
	}
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o700); err != nil {
		t.Fatal(err)
	}
	third, err := readinessTreeSHA256(root)
	if err != nil || second == third {
		t.Fatalf("directory addition not detected: second=%s third=%s err=%v", second, third, err)
	}
}

func TestReadinessTreeHashRejectsSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked-directory")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := readinessTreeSHA256(root); err == nil {
		t.Fatal("accepted a symlinked parent directory")
	}
}

func TestReadinessTreeHashRejectsFileSymlink(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(outside, []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "file-link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := readinessTreeSHA256(root); err == nil {
		t.Fatal("accepted a file symlink")
	}
}

func TestReadinessTreeHashRejectsSocket(t *testing.T) {
	root := readinessShortSocketDirectory(t)
	listener, err := net.Listen("unix", filepath.Join(root, "socket"))
	if err != nil {
		t.Fatalf("create Unix socket: %v", err)
	}
	defer listener.Close()
	if _, err := readinessTreeSHA256(root); err == nil {
		t.Fatal("accepted a socket")
	}
}

func readinessShortSocketDirectory(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Unix-domain socket fixture requires a Unix temporary directory")
	}
	root, err := os.MkdirTemp("/tmp", "cwks-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Errorf("remove socket fixture: %v", err)
		}
	})
	return root
}

type readinessCountingDirectory struct {
	entry fs.DirEntry
	calls int
}

func (d *readinessCountingDirectory) ReadDir(count int) ([]fs.DirEntry, error) {
	d.calls++
	batch := make([]fs.DirEntry, count)
	for i := range batch {
		batch[i] = d.entry
	}
	return batch, nil
}

func TestReadinessDirectoryStopsReadingWhenEntryLimitExceeded(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "entry"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	reader := &readinessCountingDirectory{entry: entries[0]}
	count := 0
	if _, err := readinessDirectoryNames(reader, false, &count, 2); err == nil {
		t.Fatal("unbounded directory was accepted")
	}
	if reader.calls != 1 || count != 3 {
		t.Fatalf("read beyond entry budget: calls=%d count=%d", reader.calls, count)
	}
}

func TestReadinessTreeHashRejectsSymlinkedGitMetadata(t *testing.T) {
	root := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(root, ".git")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := readinessTreeSHA256(root); err == nil {
		t.Fatal("accepted symlinked .git entry")
	}
}

func TestReadinessTreeHashEnforcesEntryLimit(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"one", "two", "three"} {
		if err := os.WriteFile(filepath.Join(root, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := readinessTreeSHA256Bounded(root, 2, 1024); err == nil || !strings.Contains(err.Error(), "entry limit") {
		t.Fatalf("entry limit error = %v", err)
	}
}

func TestReadinessTreeHashRejectsOversizedInput(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "oversized")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(readinessTreeByteLimit + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := readinessTreeSHA256(root); err == nil || !strings.Contains(err.Error(), "byte limit") {
		t.Fatalf("oversized input error = %v", err)
	}
}

func TestPathInsideUsesFilesystemBoundaries(t *testing.T) {
	if !pathInside("/repo", "/repo/subdir") || !pathInside("/repo", "/repo") {
		t.Fatal("repository path was not recognized as inside")
	}
	if pathInside("/repo", "/repo-backup") || pathInside("/repo", "/tmp/evidence") {
		t.Fatal("outside path was treated as inside the repository")
	}
}

func readinessTestRun() ([]byte, []byte) {
	resourceOutput := "room-ref=8173 message-ref=921004 kind=all\nfiles source-count=2 limit=100 complete=false absence=unknown matched=1 file-ref=44501\nroom-tasks source-count=2 limit=100 complete=false absence=unknown matched=1 task-ref=66301\n"
	fileOutput := "file-ref=44501 room-ref=8173 message-ref=921004 name=公開前確認表.pdf\n"
	taskOutput := "task-ref=66301 room-ref=8173 message-ref=921004 body=staging 確認後にリリース候補を公開する status=\"open\"\n"
	answer := "ルーム参照 8173、メッセージ参照 921004 の資料とルームタスクを `--kind all` で確認。取得件数 2、上限 100、該当数 1。complete=false、absence=unknown。参照 44501 は公開前確認表.pdf。参照 66301 は staging 確認後にリリース候補を公開する、状態 open。本文は読み直していません。"
	rootHelp := "Usage:\ncwk messages resources --help\ncwk files show --help\ncwk room-tasks show --help\n"
	calls := []readinessTestCall{
		{command: "cwk --help", output: rootHelp},
		{command: "cwk messages resources --help", output: "usage: messages resources\n"},
		{command: "cwk files show --help", output: "usage: files show\n"},
		{command: "cwk room-tasks show --help", output: "usage: room-tasks show\n"},
		{command: "cwk messages resources --room 8173 --message 921004 --kind all", output: resourceOutput, auth: 1, provider: []readinessProviderCall{
			{Operation: "read", Task: "files.list", RoomRef: "8173"},
			{Operation: "read", Task: "room-tasks.list", RoomRef: "8173"},
		}},
		{command: "cwk files show --room 8173 --file 44501", output: fileOutput, auth: 1, provider: []readinessProviderCall{
			{Operation: "read", Task: "files.show", RoomRef: "8173", FileRef: "44501"},
		}},
		{command: "cwk room-tasks show --room 8173 --task 66301", output: taskOutput, auth: 1, provider: []readinessProviderCall{
			{Operation: "read", Task: "room-tasks.show", RoomRef: "8173", TaskRef: "66301"},
		}},
	}
	var traceLines, journalLines []string
	traceLines = append(traceLines, `{"type":"thread.started","thread_id":"synthetic-thread"}`, `{"type":"turn.started"}`)
	for index, call := range calls {
		itemID := "cmd-" + strconv.Itoa(index+1)
		shellCommand := "/bin/zsh -c '" + call.command + "'"
		traceLines = append(traceLines,
			`{"type":"item.started","item":{"id":"`+itemID+`","type":"command_execution","command":"`+shellCommand+`"}}`,
			`{"type":"item.completed","item":{"id":"`+itemID+`","type":"command_execution","command":"`+shellCommand+`","aggregated_output":`+mustJSON(call.output)+`,"exit_code":0,"status":"completed"}}`,
		)
		journalEvent := readinessInvocationEvent{Type: "cwk_invocation", Argv: strings.Fields(strings.TrimPrefix(call.command, "cwk ")), ExitCode: 0, Stdout: call.output, DurationMillis: 10, FixtureSHA256: readinessFixtureSHA256, ProviderCalls: call.provider, Writes: call.writes}
		journalEvent.Authentication.Calls = call.auth
		encoded, err := json.Marshal(journalEvent)
		if err != nil {
			panic(err)
		}
		started, err := json.Marshal(readinessInvocationEvent{Type: "cwk_invocation_started", Argv: journalEvent.Argv})
		if err != nil {
			panic(err)
		}
		journalLines = append(journalLines, string(started), string(encoded))
	}
	traceLines = append(traceLines,
		`{"type":"item.completed","item":{"id":"answer-1","type":"agent_message","text":`+mustJSON(answer)+`}}`,
		`{"type":"turn.completed","usage":{"input_tokens":1000,"cached_input_tokens":500,"output_tokens":100,"reasoning_output_tokens":20}}`,
	)
	return []byte(strings.Join(traceLines, "\n") + "\n"), []byte(strings.Join(journalLines, "\n") + "\n")
}

func mustJSON(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

func TestReadinessJournalRequiresMatchingReservation(t *testing.T) {
	start := `{"type":"cwk_invocation_started","argv":["--help"]}`
	finish := `{"type":"cwk_invocation","argv":["--help"]}`
	for name, journal := range map[string]string{
		"missing start":  finish + "\n",
		"incomplete":     start + "\n",
		"nested":         start + "\n" + start + "\n",
		"different argv": start + "\n" + `{"type":"cwk_invocation","argv":["version"]}` + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseReadinessJournal([]byte(journal)); err == nil {
				t.Fatal("accepted unmatched reservation")
			}
		})
	}
	if events, err := parseReadinessJournal([]byte(start + "\n" + finish + "\n")); err != nil || len(events) != 1 {
		t.Fatalf("valid pair events=%v err=%v", events, err)
	}
}
