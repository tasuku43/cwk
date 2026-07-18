package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

const (
	privateDirectoryMode = 0o750
	privateArtifactMode  = 0o600
)

func createEmptyRoot(path string) (*os.Root, error) {
	info, err := os.Lstat(path)
	switch {
	case os.IsNotExist(err):
		if err := os.MkdirAll(path, privateDirectoryMode); err != nil {
			return nil, err
		}
		info, err = os.Lstat(path)
	case err != nil:
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("output path must be a real directory")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	if len(entries) != 0 {
		return nil, fmt.Errorf("output directory must be empty")
	}
	return os.OpenRoot(path)
}

func openRootedPath(path string, flags int) (*os.File, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(filepath.Dir(absolute))
	if err != nil {
		return nil, err
	}
	defer root.Close()
	name := filepath.Base(absolute)
	if info, statErr := root.Lstat(name); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("path must identify a regular file")
		}
	} else if !os.IsNotExist(statErr) {
		return nil, statErr
	}
	file, err := root.OpenFile(name, flags, privateArtifactMode)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = file.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("path must identify a regular file")
	}
	return file, nil
}

func writeJSONFile(path string, value any) error {
	file, err := openRootedPath(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return err
	}
	if err := writeJSON(file, value); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func writeRootFile(root *os.Root, name string, value []byte) error {
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, privateArtifactMode)
	if err != nil {
		return err
	}
	if _, err := file.Write(value); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func writeRootJSON(root *os.Root, name string, value any) error {
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, privateArtifactMode)
	if err != nil {
		return err
	}
	if err := writeJSON(file, value); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

type processPurpose uint8

const (
	processGitStatus processPurpose = iota + 1
	processGitRevision
	processGoBuild
	processCodexVersion
	processCodexRun
)

func validateProcessRequest(request processRequest) (string, error) {
	var expectedPath string
	switch request.Purpose {
	case processGitStatus:
		expectedPath = "git"
		if !slices.Equal(request.Args, []string{"status", "--porcelain", "--untracked-files=all"}) {
			return "", fmt.Errorf("invalid git status arguments")
		}
	case processGitRevision:
		expectedPath = "git"
		if !slices.Equal(request.Args, []string{"rev-parse", "HEAD"}) {
			return "", fmt.Errorf("invalid git revision arguments")
		}
	case processGoBuild:
		expectedPath = "go"
		if len(request.Args) != 5 || !slices.Equal(request.Args[:3], []string{"build", "-trimpath", "-o"}) || request.Args[4] != "./tools/presentationeval" || !pathWithin(request.TrustedRoot, request.Args[3]) {
			return "", fmt.Errorf("invalid fixture build arguments")
		}
	case processCodexVersion:
		if !filepath.IsAbs(request.Path) || !slices.Equal(request.Args, []string{"--version"}) {
			return "", fmt.Errorf("invalid Codex version invocation")
		}
		expectedPath = request.Path
	case processCodexRun:
		if !filepath.IsAbs(request.Path) || !validateCodexRunArguments(request.Args, request.TrustedRoot) {
			return "", fmt.Errorf("invalid pinned Codex invocation")
		}
		expectedPath = request.Path
	default:
		return "", fmt.Errorf("unknown subprocess purpose")
	}
	if request.Path != expectedPath {
		return "", fmt.Errorf("subprocess executable does not match its fixed purpose")
	}
	resolved, err := filepath.Abs(expectedPath)
	if expectedPath == "git" || expectedPath == "go" {
		resolved, err = exec.LookPath(expectedPath)
		if err == nil {
			resolved, err = filepath.Abs(resolved)
		}
	}
	if err != nil {
		return "", err
	}
	resolved, err = filepath.EvalSymlinks(resolved)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || (runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0) {
		return "", fmt.Errorf("subprocess executable must be a regular executable file")
	}
	return resolved, nil
}

func validateCodexRunArguments(args []string, trustedRoot string) bool {
	fixed := map[int]string{
		0: "--ask-for-approval", 1: "never", 2: "exec", 3: "--sandbox", 4: "workspace-write",
		5: "--ignore-user-config", 6: "--ignore-rules", 7: "--ephemeral", 8: "--skip-git-repo-check",
		9: "--strict-config", 10: "-C", 12: "--model", 14: "--json", 15: "--output-schema",
		17: "--output-last-message", 19: "-c", 20: "sandbox_workspace_write.network_access=false",
		21: "-c", 22: "shell_environment_policy.inherit=none", 23: "-c", 24: `model_reasoning_effort="medium"`,
		25: "-c", 27: "-c", 29: "-c",
	}
	minimum := 31 + 2*len(disabledCodexFeatures) + 1
	if len(args) != minimum && len(args) != minimum+2 {
		return false
	}
	for index, value := range fixed {
		if args[index] != value {
			return false
		}
	}
	if args[13] == "" || strings.HasPrefix(args[13], "-") || !pathWithin(trustedRoot, args[11]) || !pathWithin(trustedRoot, args[16]) || !pathWithin(trustedRoot, args[18]) {
		return false
	}
	fixtureBin, ok := quotedConfig(args[26], "shell_environment_policy.set.PATH")
	if !ok || (fixtureBin != "" && !pathWithin(trustedRoot, fixtureBin)) {
		return false
	}
	scenario, ok := quotedConfig(args[28], "shell_environment_policy.set."+fixtureScenarioEnvironment)
	if !ok {
		return false
	}
	if _, found := situationByID(scenario); !found {
		return false
	}
	candidate, ok := quotedConfig(args[30], "shell_environment_policy.set."+fixtureCandidateEnvironment)
	if !ok || validateCandidate(candidate) != nil {
		return false
	}
	index := 31
	for _, feature := range disabledCodexFeatures {
		if args[index] != "--disable" || args[index+1] != feature {
			return false
		}
		index += 2
	}
	if len(args) == minimum+2 {
		if args[index] != "--disable" || args[index+1] != "shell_tool" {
			return false
		}
		index += 2
	}
	return args[index] == "-"
}

func quotedConfig(value, key string) (string, bool) {
	prefix := key + "="
	if !strings.HasPrefix(value, prefix) {
		return "", false
	}
	decoded, err := strconv.Unquote(strings.TrimPrefix(value, prefix))
	return decoded, err == nil
}

func pathWithin(root, path string) bool {
	if root == "" || !filepath.IsAbs(root) || !filepath.IsAbs(path) {
		return false
	}
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
