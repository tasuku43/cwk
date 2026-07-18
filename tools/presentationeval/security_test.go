package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPrivateArtifactHelpersRejectSymlinksAndBroadModes(t *testing.T) {
	base := t.TempDir()
	directory := filepath.Join(base, "artifacts")
	root, err := createEmptyRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(directory)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&^privateDirectoryMode != 0 {
		t.Fatalf("directory mode = %o", info.Mode().Perm())
	}

	artifact := filepath.Join(base, "result.json")
	if err := writeJSONFile(artifact, map[string]bool{"ok": true}); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != privateArtifactMode {
		t.Fatalf("artifact mode = %o", info.Mode().Perm())
	}

	target := filepath.Join(base, "target.json")
	if err := os.WriteFile(target, nil, privateArtifactMode); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := openRootedPath(link, os.O_RDONLY); err == nil {
		t.Fatal("symlink input accepted")
	}
}

func TestProcessValidatorRejectsArgumentExpansion(t *testing.T) {
	request := processRequest{
		Purpose: processGoBuild,
		Path:    "go",
		Args:    []string{"build", "-trimpath", "-o", filepath.Join(t.TempDir(), "cwk"), "./tools/presentationeval", "./unexpected"},
	}
	if _, err := validateProcessRequest(request); err == nil {
		t.Fatal("expanded go build invocation accepted")
	}
}

func TestCodexArgumentContractAcceptsOnlyTheFixedInvocation(t *testing.T) {
	root := t.TempDir()
	invocation := codexInvocation{
		Model: "gpt-test", Workspace: filepath.Join(root, "workspace"), FixtureBin: filepath.Join(root, "bin"),
		Candidate: "c0", SituationID: "attention.rooms", AllowCWK: true,
	}
	args := codexArguments(invocation, filepath.Join(root, "call", "answer.schema.json"), filepath.Join(root, "call", "last-message.json"))
	if !validateCodexRunArguments(args, root) {
		t.Fatal("fixed Codex invocation was rejected")
	}
	args = append(args[:2], append([]string{"--dangerously-bypass-approvals-and-sandbox"}, args[2:]...)...)
	if validateCodexRunArguments(args, root) {
		t.Fatal("expanded Codex invocation was accepted")
	}
}
