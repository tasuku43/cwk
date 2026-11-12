//go:build readiness_eval

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/tasuku43/cwk/internal/infra/readinessio"
)

const (
	readinessCodexVersion     = "codex-cli 0.164.0"
	readinessModel            = "gpt-6-luna"
	readinessEffort           = "max"
	readinessTimeout          = 3 * time.Minute
	readinessTraceByteLimit   = 16 << 20
	readinessJournalByteLimit = 1 << 20
	readinessStderrByteLimit  = 1 << 20
	readinessHelperAuditFlag  = "--cwk-readiness-audit"
	readinessTreeEntryLimit   = 100000
	readinessTreeByteLimit    = 512 << 20
	readinessCommandByteLimit = 1 << 20
	readinessProcessWaitDelay = time.Second
)

const readinessPrompt = "使えるコマンドは cwk だけです。最初に cwk --help で利用できる操作を調べてください。ルーム 8173 のメッセージ 921004 に関連する資料とルームタスクを確認し、それぞれ該当する項目をまとめてください。資料名とタスクの内容・状態に加えて、資料とタスクごとの取得件数、上限、該当数、完全性、該当なしの状態を示してください。必要な項目の詳細も確認し、メッセージ本文は読み直さないでください。"

type readinessRunManifest struct {
	Status                    string   `json:"status"`
	EvidenceDir               string   `json:"evidence_dir"`
	StartedAt                 string   `json:"started_at"`
	FinishedAt                string   `json:"finished_at"`
	DurationMillis            int64    `json:"duration_ms"`
	CodexDurationMillis       int64    `json:"codex_duration_ms"`
	GitHead                   string   `json:"git_head"`
	TreeSHA256BeforeBuild     string   `json:"tree_sha256_before_build"`
	TreeSHA256BeforeRun       string   `json:"tree_sha256_before_run"`
	TreeSHA256AfterRun        string   `json:"tree_sha256_after_run"`
	TreeHashesMatch           bool     `json:"tree_hashes_match"`
	SnapshotSHA256AfterBuild  string   `json:"snapshot_sha256_after_build"`
	SnapshotSHA256AfterRun    string   `json:"snapshot_sha256_after_run"`
	GoVersion                 string   `json:"go_version"`
	CodexPath                 string   `json:"codex_path"`
	CodexBinarySHA256         string   `json:"codex_binary_sha256"`
	CodexVersion              string   `json:"codex_version"`
	CodexVersionPinned        bool     `json:"codex_version_pinned"`
	Model                     string   `json:"model"`
	ReasoningEffort           string   `json:"reasoning_effort"`
	ModelAliasLimitation      string   `json:"model_alias_limitation"`
	Sandbox                   string   `json:"sandbox"`
	WorkspaceNetworkAccess    bool     `json:"workspace_network_access"`
	SessionEphemeral          bool     `json:"session_ephemeral"`
	UserConfigIgnored         bool     `json:"user_config_ignored"`
	ExecRulesIgnored          bool     `json:"exec_rules_ignored"`
	ShellEnvironmentInherited string   `json:"shell_environment_inherited"`
	ShellEnvironmentAllowlist []string `json:"shell_environment_allowlist"`
	ShellPATHEntries          int      `json:"shell_path_entries"`
	CredentialBoundary        string   `json:"credential_boundary"`
	FixtureSHA256             string   `json:"fixture_sha256"`
	PromptSHA256              string   `json:"prompt_sha256"`
	HelperBinarySHA256        string   `json:"helper_binary_sha256"`
	TraceSHA256               string   `json:"trace_sha256"`
	TraceBytes                int64    `json:"trace_bytes"`
	CodexStderrSHA256         string   `json:"codex_stderr_sha256"`
	CodexStderrBytes          int64    `json:"codex_stderr_bytes"`
	JournalSHA256             string   `json:"journal_sha256"`
	JournalBytes              int64    `json:"journal_bytes"`
	AuditSHA256               string   `json:"audit_sha256"`
	AuditError                string   `json:"audit_error,omitempty"`
	CodexExitCode             int      `json:"codex_exit_code"`
	CodexRunError             string   `json:"codex_run_error,omitempty"`
	Failure                   string   `json:"failure,omitempty"`
	CleanupError              string   `json:"cleanup_error,omitempty"`
}

func runCodexReadiness([]string) error {
	return errors.New("readiness execution is disabled until the agent isolation and evidence boundaries are verified")
}

// The draft is retained for the isolation redesign; it must not be exposed by
// the command entry point until its external execution boundaries are verified.
func runCodexReadinessDraft(args []string) (runErr error) {
	flags := flag.NewFlagSet("readinessrunner run", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoRootFlag := flags.String("repo-root", ".", "repository root used to build the synthetic CLI helper")
	evidenceDirFlag := flags.String("evidence-dir", "", "private evidence directory outside the repository; defaults to a temporary directory")
	codexPathFlag := flags.String("codex", "", "Codex CLI executable; defaults to PATH lookup")
	if err := flags.Parse(args); err != nil {
		return errors.New(boundedReadinessDiagnostic(err.Error(), 1024))
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}

	repoRoot, err := filepath.Abs(*repoRootFlag)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	repoRoot, err = filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return fmt.Errorf("resolve repository root symlinks: %w", err)
	}
	gitRoot, err := commandText(repoRoot, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("resolve Git repository root: %w", err)
	}
	gitRoot, err = filepath.EvalSymlinks(gitRoot)
	if err != nil || gitRoot != repoRoot {
		return fmt.Errorf("repo-root must be the Git working-tree root: %s", repoRoot)
	}
	evidenceDir, err := createEvidenceDirectory(repoRoot, *evidenceDirFlag)
	if err != nil {
		return err
	}

	startedAt := time.Now().UTC()
	manifest := readinessRunManifest{
		Status:                    "inconclusive",
		EvidenceDir:               evidenceDir,
		StartedAt:                 startedAt.Format(time.RFC3339Nano),
		Model:                     readinessModel,
		ReasoningEffort:           readinessEffort,
		ModelAliasLimitation:      "model alias pinned; underlying hosted model build is not exposed by the Codex CLI trace",
		Sandbox:                   "workspace-write",
		WorkspaceNetworkAccess:    false,
		SessionEphemeral:          true,
		UserConfigIgnored:         true,
		ExecRulesIgnored:          true,
		ShellEnvironmentInherited: "none",
		ShellEnvironmentAllowlist: []string{"HOME", "PATH"},
		ShellPATHEntries:          1,
		CredentialBoundary:        "synthetic test authenticator; agent shell inherits no host environment and receives only temporary HOME and fixture-only PATH",
		FixtureSHA256:             readinessFixtureSHA256,
		PromptSHA256:              sha256Text(readinessPrompt),
	}
	defer func() {
		manifest.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		manifest.DurationMillis = time.Since(startedAt).Milliseconds()
		if err := writeRunSummary(evidenceDir, manifest, os.Stdout); err != nil {
			runErr = errors.Join(runErr, err)
		}
	}()

	if err := os.WriteFile(filepath.Join(evidenceDir, "prompt.txt"), []byte(readinessPrompt+"\n"), 0o600); err != nil {
		manifest.Failure = fmt.Sprintf("save prompt: %v", err)
		return err
	}
	manifest.GoVersion, err = commandText(repoRoot, "go", "version")
	if err != nil {
		manifest.Failure = fmt.Sprintf("read Go version: %v", err)
		return err
	}
	manifest.GitHead, err = commandText(repoRoot, "git", "rev-parse", "HEAD")
	if err != nil {
		manifest.Failure = fmt.Sprintf("read git HEAD: %v", err)
		return err
	}
	manifest.CodexPath = *codexPathFlag
	if manifest.CodexPath == "" {
		manifest.CodexPath, err = exec.LookPath("codex")
		if err != nil {
			manifest.Failure = "Codex CLI not found on PATH"
			return err
		}
	}
	manifest.CodexPath, err = filepath.Abs(manifest.CodexPath)
	if err != nil {
		manifest.Failure = fmt.Sprintf("resolve Codex CLI path: %v", err)
		return err
	}
	manifest.CodexBinarySHA256, err = fileSHA256(manifest.CodexPath)
	if err != nil {
		manifest.Failure = fmt.Sprintf("hash Codex CLI binary: %v", err)
		return err
	}
	manifest.CodexVersion, err = commandText(repoRoot, manifest.CodexPath, "--version")
	if err != nil {
		manifest.Failure = fmt.Sprintf("read Codex version: %v", err)
		return err
	}
	manifest.CodexVersionPinned = strings.TrimSpace(manifest.CodexVersion) == readinessCodexVersion
	if !manifest.CodexVersionPinned {
		manifest.Failure = fmt.Sprintf("Codex version %q does not match reviewed pin %q", strings.TrimSpace(manifest.CodexVersion), readinessCodexVersion)
		return errors.New(manifest.Failure)
	}
	snapshot, err := createReadinessSnapshot(repoRoot, evidenceDir)
	if err != nil {
		var creationErr *readinessSnapshotCreationError
		if errors.As(err, &creationErr) {
			manifest.Failure = fmt.Sprintf("create source snapshot: %v", creationErr.cause)
			manifest.CleanupError = fmt.Sprintf("remove private source snapshot: %v", creationErr.cleanup)
			if snapshot != nil {
				defer func() {
					if retryErr := snapshot.Close(); retryErr != nil {
						manifest.CleanupError = fmt.Sprintf("%s; retry: %v", manifest.CleanupError, retryErr)
						runErr = errors.Join(runErr, retryErr)
					}
				}()
			}
		} else {
			manifest.Failure = fmt.Sprintf("create source snapshot: %v", err)
		}
		return err
	}
	defer func() {
		if err := snapshot.Close(); err != nil {
			runErr = recordReadinessCleanupFailure(&manifest, runErr, err)
		}
	}()
	manifest.TreeSHA256BeforeBuild = snapshot.Digest
	sourceAfterCopy, err := readinessTreeSHA256(repoRoot)
	if err != nil || sourceAfterCopy != snapshot.Digest {
		manifest.Failure = "source tree changed while creating snapshot; model was not started"
		return errors.Join(errors.New(manifest.Failure), err)
	}
	binDir := filepath.Join(evidenceDir, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		manifest.Failure = fmt.Sprintf("create helper directory: %v", err)
		return err
	}
	helperPath := filepath.Join(binDir, "cwk-readiness.test")
	buildLog, err := runCommandOutput(snapshot.Path, filepath.Join(evidenceDir, "helper-build.log"), "go", "test", "-buildvcs=false", "-tags", "readiness_eval", "-c", "-o", helperPath, "./internal/cli")
	if err != nil {
		manifest.Failure = fmt.Sprintf("build helper: %v; %s", err, strings.TrimSpace(string(buildLog)))
		return errors.New(manifest.Failure)
	}
	manifest.HelperBinarySHA256, err = fileSHA256(helperPath)
	if err != nil {
		manifest.Failure = fmt.Sprintf("hash helper binary: %v", err)
		return err
	}
	manifest.SnapshotSHA256AfterBuild, err = readinessTreeSHA256(snapshot.Path)
	if err != nil || manifest.SnapshotSHA256AfterBuild != snapshot.Digest {
		manifest.Failure = "source snapshot changed during helper build; model was not started"
		return errors.Join(errors.New(manifest.Failure), err)
	}
	manifest.TreeSHA256BeforeRun, err = readinessTreeSHA256(repoRoot)
	if err != nil {
		manifest.Failure = fmt.Sprintf("hash target tree before model run: %v", err)
		return err
	}
	if manifest.TreeSHA256BeforeBuild != manifest.TreeSHA256BeforeRun {
		manifest.Failure = "target tree changed between helper build and model run; model was not started"
		return errors.New(manifest.Failure)
	}

	workspace := filepath.Join(evidenceDir, "workspace")
	helperBin := filepath.Join(evidenceDir, "fixture-bin")
	for _, dir := range []string{workspace, filepath.Join(workspace, "home"), helperBin} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			manifest.Failure = fmt.Sprintf("create isolated run directory: %v", err)
			return err
		}
	}
	journalPath := filepath.Join(workspace, "cwk-invocations.jsonl")
	if err := os.WriteFile(journalPath, nil, 0o600); err != nil {
		manifest.Failure = fmt.Sprintf("initialize invocation journal: %v", err)
		return err
	}
	wrapperPath := filepath.Join(helperBin, "cwk")
	wrapper := "#!/bin/sh\nexec " + shellQuote(helperPath) + " -test.run '^TestReadinessHelper$' -- " + readinessHelperAuditFlag + " " + shellQuote(journalPath) + " \"$@\"\n"
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o700); err != nil {
		manifest.Failure = fmt.Sprintf("write cwk fixture wrapper: %v", err)
		return err
	}
	tracePath := filepath.Join(evidenceDir, "codex-trace.jsonl")
	stderrPath := filepath.Join(evidenceDir, "codex-stderr.txt")
	traceFile, err := os.OpenFile(tracePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		manifest.Failure = fmt.Sprintf("create Codex trace: %v", err)
		return err
	}
	stderrFile, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		traceFile.Close()
		manifest.Failure = fmt.Sprintf("create Codex stderr log: %v", err)
		return err
	}
	modelContext, cancel := context.WithTimeout(context.Background(), readinessTimeout)
	defer cancel()
	codexArgs := codexReadinessArgs(workspace, filepath.Join(workspace, "home"), helperBin)
	codexCmd := exec.CommandContext(modelContext, manifest.CodexPath, codexArgs...)
	codexCmd.WaitDelay = readinessProcessWaitDelay
	codexCmd.Dir = workspace
	codexCmd.Stdin = strings.NewReader(readinessPrompt)
	setReadinessStreams(codexCmd, traceFile, stderrFile, readinessTraceByteLimit, readinessStderrByteLimit, cancel)
	modelStarted := time.Now()
	err = codexCmd.Run()
	manifest.CodexDurationMillis = time.Since(modelStarted).Milliseconds()
	_ = traceFile.Close()
	_ = stderrFile.Close()
	if codexCmd.ProcessState != nil {
		manifest.CodexExitCode = codexCmd.ProcessState.ExitCode()
	} else {
		manifest.CodexExitCode = -1
	}
	if err != nil {
		manifest.CodexRunError = err.Error()
		if errors.Is(modelContext.Err(), context.DeadlineExceeded) {
			manifest.CodexRunError = "time budget exceeded"
		}
	}
	manifest.TreeSHA256AfterRun, err = readinessTreeSHA256(repoRoot)
	if err != nil {
		manifest.Failure = fmt.Sprintf("hash target tree after model run: %v", err)
		return err
	}
	manifest.SnapshotSHA256AfterRun, err = readinessTreeSHA256(snapshot.Path)
	if err != nil {
		manifest.Failure = fmt.Sprintf("hash source snapshot after model run: %v", err)
		return err
	}
	manifest.TreeHashesMatch = manifest.TreeSHA256BeforeBuild == manifest.TreeSHA256BeforeRun && manifest.TreeSHA256BeforeRun == manifest.TreeSHA256AfterRun && manifest.SnapshotSHA256AfterBuild == manifest.TreeSHA256BeforeBuild && manifest.SnapshotSHA256AfterRun == manifest.TreeSHA256BeforeBuild
	trace, traceReadErr := readReadinessArtifact(tracePath, readinessTraceByteLimit)
	stderr, stderrReadErr := readReadinessArtifact(stderrPath, readinessStderrByteLimit)
	journal, journalReadErr := readReadinessArtifact(journalPath, readinessJournalByteLimit)
	if traceReadErr != nil || stderrReadErr != nil || journalReadErr != nil {
		manifest.AuditError = fmt.Sprintf("read run artifacts: trace=%v stderr=%v journal=%v", traceReadErr, stderrReadErr, journalReadErr)
	} else {
		manifest.TraceSHA256 = sha256Bytes(trace)
		manifest.TraceBytes = int64(len(trace))
		manifest.CodexStderrSHA256 = sha256Bytes(stderr)
		manifest.CodexStderrBytes = int64(len(stderr))
		manifest.JournalSHA256 = sha256Bytes(journal)
		manifest.JournalBytes = int64(len(journal))
		report, auditErr := auditReadinessRun(trace, journal)
		if err := writeJSONFile(filepath.Join(evidenceDir, "audit.json"), report); err != nil {
			manifest.AuditError = fmt.Sprintf("write audit report: %v", err)
		} else {
			manifest.AuditSHA256, err = fileSHA256(filepath.Join(evidenceDir, "audit.json"))
			if err != nil {
				manifest.AuditError = fmt.Sprintf("hash audit report: %v", err)
			}
			if auditErr != nil {
				manifest.AuditError = auditErr.Error()
			}
		}
	}

	if !manifest.TreeHashesMatch {
		manifest.Failure = "target tree changed during the evaluation; evidence is inconclusive"
		return errors.New(manifest.Failure)
	}
	if manifest.CodexRunError != "" || manifest.CodexExitCode != 0 {
		manifest.Failure = "Codex run did not complete successfully; evidence is inconclusive"
		return errors.New(manifest.Failure)
	}
	if manifest.AuditError != "" {
		manifest.Failure = "trace audit failed; evidence is inconclusive"
		return errors.New(manifest.Failure)
	}
	manifest.Failure = "matching tree hashes do not prove a frozen evaluation snapshot; evidence is inconclusive"
	return errors.New(manifest.Failure)
}

func recordReadinessCleanupFailure(manifest *readinessRunManifest, prior, cleanup error) error {
	manifest.CleanupError = fmt.Sprintf("remove private source snapshot: %v", cleanup)
	return errors.Join(prior, cleanup)
}

type readinessBoundedWriter struct {
	output    io.Writer
	remaining int64
	cancel    context.CancelFunc
}

func setReadinessStreams(cmd *exec.Cmd, stdout, stderr io.Writer, stdoutLimit, stderrLimit int64, cancel context.CancelFunc) {
	cmd.Stdout = &readinessBoundedWriter{output: stdout, remaining: stdoutLimit, cancel: cancel}
	cmd.Stderr = &readinessBoundedWriter{output: stderr, remaining: stderrLimit, cancel: cancel}
}

func (w *readinessBoundedWriter) Write(p []byte) (int, error) {
	tooLarge := int64(len(p)) > w.remaining
	if tooLarge {
		p = p[:int(w.remaining)]
	}
	n, err := w.output.Write(p)
	w.remaining -= int64(n)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	if tooLarge && err == nil {
		err = errors.New("readiness artifact byte limit exceeded")
	}
	if err != nil {
		w.cancel()
	}
	return n, err
}

func readReadinessArtifact(path string, limit int64) ([]byte, error) {
	file, err := readinessio.OpenRegular(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	contents, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(contents)) > limit {
		return nil, errors.New("readiness artifact byte limit exceeded")
	}
	return contents, nil
}

func sha256Bytes(contents []byte) string {
	sum := sha256.Sum256(contents)
	return hex.EncodeToString(sum[:])
}

type readinessSnapshot struct {
	Path   string
	Digest string
	parent *os.Root
	root   *os.Root
}

func createReadinessSnapshot(source, evidence string) (*readinessSnapshot, error) {
	return createReadinessSnapshotWithCopier(source, evidence, func(source string, destination *os.Root) (string, error) {
		return copyAndHashReadinessTree(source, destination, readinessTreeEntryLimit, readinessTreeByteLimit)
	})
}

func createReadinessSnapshotWithCopier(source, evidence string, copyTree func(string, *os.Root) (string, error)) (result *readinessSnapshot, resultErr error) {
	if runtime.GOOS == "windows" {
		return nil, errors.New("private snapshot ACL verification is unavailable on Windows")
	}
	resolvedSource, err := filepath.EvalSymlinks(source)
	if err != nil {
		return nil, fmt.Errorf("resolve source: %w", err)
	}
	resolvedEvidence, err := filepath.EvalSymlinks(evidence)
	if err != nil {
		return nil, fmt.Errorf("resolve evidence directory: %w", err)
	}
	if pathInside(resolvedSource, resolvedEvidence) {
		return nil, errors.New("snapshot parent must be outside source tree")
	}
	parent, err := os.OpenRoot(resolvedEvidence)
	if err != nil {
		return nil, err
	}
	info, err := parent.Stat(".")
	if err != nil || !info.IsDir() || (runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0) {
		parent.Close()
		return nil, errors.New("snapshot parent must be a private directory")
	}
	const name = "source-tree"
	if err := parent.Mkdir(name, 0o700); err != nil {
		parent.Close()
		return nil, fmt.Errorf("create private source snapshot: %w", err)
	}
	value := &readinessSnapshot{Path: filepath.Join(resolvedEvidence, name), parent: parent}
	defer func() {
		if resultErr != nil {
			result, resultErr = cleanupReadinessSnapshotAfterCreateFailure(value, resultErr)
		}
	}()
	value.root, err = parent.OpenRoot(name)
	if err != nil {
		return nil, fmt.Errorf("open source snapshot: %w", err)
	}
	value.Digest, err = copyTree(resolvedSource, value.root)
	if err != nil {
		return nil, err
	}
	copyDigest, err := readinessTreeSHA256(value.Path)
	if err != nil || copyDigest != value.Digest {
		return nil, errors.Join(errors.New("source snapshot digest mismatch"), err)
	}
	return value, nil
}

type readinessSnapshotCreationError struct {
	cause   error
	cleanup error
}

func (err *readinessSnapshotCreationError) Error() string {
	return fmt.Sprintf("%v; snapshot cleanup failed: %v", err.cause, err.cleanup)
}

func (err *readinessSnapshotCreationError) Unwrap() []error {
	return []error{err.cause, err.cleanup}
}

func cleanupReadinessSnapshotAfterCreateFailure(snapshot *readinessSnapshot, cause error) (*readinessSnapshot, error) {
	if cleanupErr := snapshot.Close(); cleanupErr != nil {
		if snapshot.parent == nil {
			snapshot = nil
		}
		return snapshot, &readinessSnapshotCreationError{cause: cause, cleanup: cleanupErr}
	}
	return nil, cause
}

func (snapshot *readinessSnapshot) Close() error {
	if snapshot.parent == nil {
		return nil
	}
	if snapshot.root == nil {
		closeErr := snapshot.parent.Close()
		snapshot.parent = nil
		return errors.Join(errors.New("snapshot identity unavailable; entry left in place and cannot be retried safely"), closeErr)
	} else {
		if err := removeReadinessSnapshotContents(snapshot.root); err != nil {
			return err
		}
		original, err := snapshot.root.Stat(".")
		if err != nil {
			return err
		}
		current, err := snapshot.parent.Lstat("source-tree")
		if err != nil || !current.IsDir() || !os.SameFile(original, current) {
			return errors.New("snapshot entry changed before cleanup")
		}
		if err := snapshot.parent.Remove("source-tree"); err != nil {
			return err
		}
		return finishReadinessSnapshotHandles(snapshot, snapshot.root.Close, snapshot.parent.Close)
	}
}

func finishReadinessSnapshotHandles(snapshot *readinessSnapshot, closeRoot, closeParent func() error) error {
	rootErr := closeRoot()
	parentErr := closeParent()
	snapshot.root = nil
	snapshot.parent = nil
	return errors.Join(rootErr, parentErr)
}

func removeReadinessSnapshotContents(root *os.Root) error {
	var entries int
	var prepare func(string, int) error
	prepare = func(rel string, depth int) error {
		if depth > 128 {
			return errors.New("snapshot cleanup depth limit exceeded")
		}
		info, err := root.Lstat(rel)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		dir, err := root.Open(rel)
		if err != nil {
			return err
		}
		opened, err := dir.Stat()
		if err != nil || !opened.IsDir() || !os.SameFile(info, opened) {
			dir.Close()
			return errors.New("snapshot directory changed during cleanup")
		}
		if err := dir.Chmod(opened.Mode().Perm() | 0o700); err != nil {
			dir.Close()
			return err
		}
		names, readErr := readinessDirectoryNames(dir, false, &entries, readinessTreeEntryLimit)
		closeErr := dir.Close()
		if readErr != nil || closeErr != nil {
			return errors.Join(readErr, closeErr)
		}
		for _, name := range names {
			if err := prepare(filepath.Join(rel, name), depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := prepare(".", 0); err != nil {
		return err
	}
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	count := 0
	names, readErr := readinessDirectoryNames(dir, false, &count, readinessTreeEntryLimit)
	closeErr := dir.Close()
	if readErr != nil || closeErr != nil {
		return errors.Join(readErr, closeErr)
	}
	for _, name := range names {
		if err := root.RemoveAll(name); err != nil {
			return err
		}
	}
	return nil
}

func createEvidenceDirectory(repoRoot, requested string) (string, error) {
	parent := os.TempDir()
	name := ""
	if requested != "" {
		absolute, err := filepath.Abs(requested)
		if err != nil {
			return "", fmt.Errorf("resolve evidence directory: %w", err)
		}
		parent, name = filepath.Dir(absolute), filepath.Base(absolute)
	}
	parent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", fmt.Errorf("resolve evidence parent: %w", err)
	}
	if pathInside(repoRoot, parent) {
		return "", errors.New("evidence directory must be outside the repository")
	}
	before, err := os.Stat(parent)
	if err != nil {
		return "", fmt.Errorf("inspect evidence parent: %w", err)
	}
	parentRoot, err := os.OpenRoot(parent)
	if err != nil {
		return "", fmt.Errorf("open evidence parent: %w", err)
	}
	defer parentRoot.Close()
	opened, err := parentRoot.Stat(".")
	if err != nil || !os.SameFile(before, opened) {
		return "", errors.New("evidence parent changed while opening")
	}
	if name == "" {
		var suffix [16]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return "", fmt.Errorf("generate evidence directory name: %w", err)
		}
		name = "cwk-readiness-evidence-" + hex.EncodeToString(suffix[:])
	}
	directory := filepath.Join(parent, name)
	if pathInside(repoRoot, directory) {
		return "", errors.New("evidence directory must be outside the repository")
	}
	if err := parentRoot.Mkdir(name, 0o700); err != nil {
		return "", fmt.Errorf("create new evidence directory: %w", err)
	}
	current, err := os.Stat(parent)
	if err != nil || !os.SameFile(opened, current) {
		if removeErr := parentRoot.Remove(name); removeErr != nil {
			return "", errors.Join(errors.New("evidence parent changed after creation"), removeErr)
		}
		return "", errors.New("evidence parent changed after creation")
	}
	return directory, nil
}

func writeRunSummary(evidenceDir string, manifest readinessRunManifest, output io.Writer) error {
	if err := writeJSONFile(filepath.Join(evidenceDir, "run-manifest.json"), manifest); err != nil {
		return fmt.Errorf("write run manifest: %w", err)
	}
	return json.NewEncoder(output).Encode(struct {
		Status      string `json:"status"`
		EvidenceDir string `json:"evidence_dir"`
	}{Status: manifest.Status, EvidenceDir: evidenceDir})
}

func codexReadinessArgs(workspace, home, helperBin string) []string {
	return []string{
		"exec",
		"--json",
		"--color", "never",
		"--ephemeral",
		"--ignore-user-config",
		"--ignore-rules",
		"--strict-config",
		"--skip-git-repo-check",
		"--sandbox", "workspace-write",
		"--model", readinessModel,
		"--cd", workspace,
		"-c", `model_reasoning_effort="max"`,
		"-c", "sandbox_workspace_write.network_access=false",
		"-c", `shell_environment_policy.inherit="none"`,
		"-c", `shell_environment_policy.include_only=["HOME","PATH"]`,
		"-c", "shell_environment_policy.set={HOME=" + strconvQuote(home) + ",PATH=" + strconvQuote(helperBin) + "}",
		"-",
	}
}

func readinessTreeSHA256(root string) (string, error) {
	return readinessTreeSHA256Bounded(root, readinessTreeEntryLimit, readinessTreeByteLimit)
}

func readinessTreeSHA256Bounded(root string, entryLimit int, byteLimit int64) (string, error) {
	return copyAndHashReadinessTree(root, nil, entryLimit, byteLimit)
}

// copyAndHashReadinessTree optionally copies the exact bytes it hashes into an
// empty, privately owned directory. It does not make that directory immutable.
func copyAndHashReadinessTree(root string, snapshot *os.Root, entryLimit int, byteLimit int64) (string, error) {
	anchored, err := os.OpenRoot(root)
	if err != nil {
		return "", fmt.Errorf("open target tree: %w", err)
	}
	defer anchored.Close()
	if snapshot != nil {
		if err := validateReadinessSnapshotRoots(anchored, snapshot); err != nil {
			return "", err
		}
	}
	hash := sha256.New()
	var entries int
	var totalBytes int64
	var walk func(string, int) error
	walk = func(rel string, depth int) error {
		info, err := anchored.Lstat(rel)
		if err != nil {
			return fmt.Errorf("inspect %q: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			return fmt.Errorf("target tree contains unsupported entry %q", rel)
		}
		if snapshot != nil && info.Mode() & ^(os.ModeDir|os.ModePerm) != 0 {
			return fmt.Errorf("target tree contains unsupported mode at %q", rel)
		}
		if rel != "." {
			writeHashString(hash, rel)
			writeHashString(hash, info.Mode().String())
		}
		if info.IsDir() {
			if depth > 128 {
				return errors.New("target tree depth limit exceeded")
			}
			if snapshot != nil && rel != "." {
				if err := snapshot.Mkdir(rel, 0o700); err != nil {
					return fmt.Errorf("create snapshot directory %q: %w", rel, err)
				}
			}
			dir, err := anchored.Open(rel)
			if err != nil {
				return fmt.Errorf("open directory %q: %w", rel, err)
			}
			opened, err := dir.Stat()
			if err != nil || !os.SameFile(info, opened) || !opened.IsDir() {
				dir.Close()
				return fmt.Errorf("target tree directory %q changed before reading", rel)
			}
			names, readErr := readinessDirectoryNames(dir, rel == ".", &entries, entryLimit)
			if err := dir.Close(); err != nil {
				return fmt.Errorf("close directory %q: %w", rel, err)
			}
			if readErr != nil {
				return fmt.Errorf("read directory %q: %w", rel, readErr)
			}
			sort.Strings(names)
			for _, name := range names {
				if rel == "." && name == ".git" {
					info, err := anchored.Lstat(name)
					if err != nil || (info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular())) {
						return errors.New("unsupported root .git entry")
					}
					continue
				}
				if name == ".git" && snapshot != nil {
					return errors.New("nested Git metadata is not copied into readiness snapshots")
				}
				if err := walk(filepath.Join(rel, name), depth+1); err != nil {
					return err
				}
			}
			if snapshot != nil && rel != "." {
				if err := snapshot.Chmod(rel, info.Mode().Perm()); err != nil {
					return fmt.Errorf("set snapshot directory mode %q: %w", rel, err)
				}
			}
			return nil
		}
		if info.Size() < 0 || info.Size() > byteLimit-totalBytes {
			return errors.New("target tree byte limit exceeded")
		}
		totalBytes += info.Size()
		_ = binary.Write(hash, binary.BigEndian, uint64(info.Size()))
		file, err := anchored.Open(rel)
		if err != nil {
			return fmt.Errorf("open %q: %w", rel, err)
		}
		opened, err := file.Stat()
		if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
			file.Close()
			return fmt.Errorf("target tree entry %q changed before reading", rel)
		}
		var snapshotFile *os.File
		var target io.Writer = hash
		if snapshot != nil {
			snapshotFile, err = snapshot.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
			if err != nil {
				file.Close()
				return fmt.Errorf("create snapshot file %q: %w", rel, err)
			}
			target = io.MultiWriter(hash, snapshotFile)
		}
		count, copyErr := io.Copy(target, io.LimitReader(file, info.Size()+1))
		after, statErr := file.Stat()
		closeErr := file.Close()
		if snapshotFile != nil {
			if err := snapshotFile.Chmod(info.Mode().Perm()); err != nil && copyErr == nil {
				copyErr = err
			}
			if err := snapshotFile.Close(); err != nil && copyErr == nil {
				copyErr = err
			}
		}
		if copyErr != nil {
			return fmt.Errorf("hash %q: %w", rel, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close %q: %w", rel, closeErr)
		}
		if statErr != nil || count != info.Size() || after.Size() != info.Size() || after.Mode() != info.Mode() || !after.ModTime().Equal(info.ModTime()) {
			return fmt.Errorf("target tree entry %q changed while reading", rel)
		}
		writeHashString(hash, "\x00END\x00")
		return nil
	}
	err = walk(".", 0)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func validateReadinessSnapshotRoots(source, snapshot *os.Root) error {
	if runtime.GOOS == "windows" {
		return errors.New("private snapshot ACL verification is unavailable on Windows")
	}
	sourceInfo, err := source.Stat(".")
	if err != nil {
		return fmt.Errorf("inspect source root: %w", err)
	}
	snapshotInfo, err := snapshot.Stat(".")
	if err != nil {
		return fmt.Errorf("inspect snapshot root: %w", err)
	}
	if os.SameFile(sourceInfo, snapshotInfo) {
		return errors.New("snapshot root must differ from source root")
	}
	if runtime.GOOS != "windows" && snapshotInfo.Mode().Perm()&0o077 != 0 {
		return errors.New("snapshot root must be private")
	}
	sourcePath, err := filepath.EvalSymlinks(source.Name())
	if err != nil {
		return fmt.Errorf("resolve source root: %w", err)
	}
	snapshotPath, err := filepath.EvalSymlinks(snapshot.Name())
	if err != nil {
		return fmt.Errorf("resolve snapshot root: %w", err)
	}
	sourcePath, err = filepath.Abs(sourcePath)
	if err != nil {
		return err
	}
	snapshotPath, err = filepath.Abs(snapshotPath)
	if err != nil || pathInside(sourcePath, snapshotPath) {
		return errors.New("snapshot root must be outside source root")
	}
	directory, err := snapshot.Open(".")
	if err != nil {
		return fmt.Errorf("open snapshot root: %w", err)
	}
	entries, readErr := directory.ReadDir(1)
	closeErr := directory.Close()
	if readErr != nil && readErr != io.EOF {
		return fmt.Errorf("inspect snapshot contents: %w", readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close snapshot root: %w", closeErr)
	}
	if len(entries) != 0 {
		return errors.New("snapshot root must be empty")
	}
	return nil
}

type readinessDirectoryReader interface {
	ReadDir(int) ([]fs.DirEntry, error)
}

func readinessDirectoryNames(dir readinessDirectoryReader, root bool, entries *int, limit int) ([]string, error) {
	var names []string
	for {
		batch, err := dir.ReadDir(256)
		for _, child := range batch {
			if !(root && child.Name() == ".git") {
				*entries++
				if *entries > limit {
					return nil, errors.New("target tree entry limit exceeded")
				}
			}
			names = append(names, child.Name())
		}
		if err == io.EOF {
			return names, nil
		}
		if err != nil {
			return nil, err
		}
	}
}

func pathInside(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return true
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func writeHashString(hash io.Writer, value string) {
	_ = binary.Write(hash, binary.BigEndian, uint64(len(value)))
	_, _ = io.WriteString(hash, value)
}

func commandText(dir, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = readinessProcessWaitDelay
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	setReadinessStreams(cmd, &stdout, &stderr, readinessCommandByteLimit, readinessCommandByteLimit, cancel)
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func runCommandOutput(dir, logPath, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), readinessTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = readinessProcessWaitDelay
	cmd.Dir = dir
	if name == "go" {
		cmd.Env = append(os.Environ(), "GOENV=off", "GOFLAGS=", "GOWORK=off", "GOTOOLCHAIN=auto", "GOPROXY=off")
	}
	var buffer bytes.Buffer
	writer := &readinessBoundedWriter{output: &buffer, remaining: readinessCommandByteLimit, cancel: cancel}
	cmd.Stdout, cmd.Stderr = writer, writer
	err := cmd.Run()
	output := buffer.Bytes()
	if writeErr := os.WriteFile(logPath, output, 0o600); writeErr != nil {
		return output, writeErr
	}
	return output, err
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func sha256Text(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func writeJSONFile(path string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(append(encoded, '\n'))
	return errors.Join(writeErr, file.Close())
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func strconvQuote(value string) string {
	// TOML basic strings and Go string literals share the escapes needed by
	// the temporary HOME and PATH values used here.
	var buffer bytes.Buffer
	buffer.WriteByte('"')
	for _, r := range value {
		switch r {
		case '\\', '"':
			buffer.WriteByte('\\')
			buffer.WriteRune(r)
		case '\n':
			buffer.WriteString("\\n")
		case '\r':
			buffer.WriteString("\\r")
		case '\t':
			buffer.WriteString("\\t")
		default:
			buffer.WriteRune(r)
		}
	}
	buffer.WriteByte('"')
	return buffer.String()
}
