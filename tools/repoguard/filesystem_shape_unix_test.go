//go:build darwin || linux

package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/tasuku43/cwk/tools/internal/projectconfig"
)

func TestCheckFilesystemShapeRejectsSpecialFilesInsideUntrackedClaudeDirectory(t *testing.T) {
	root := t.TempDir()
	initGitRepositoryFixture(t, root)
	localClaude := filepath.Join(root, ".claude")
	if err := os.Mkdir(localClaude, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(localClaude, "events"), 0o600); err != nil {
		t.Fatal(err)
	}

	issues, err := checkFilesystemShape(root, projectconfig.Config{Project: projectconfig.Project{BinaryName: "cwk"}})
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.ToSlash(filepath.Join(".claude", "events"))
	if len(issues) != 1 || issues[0].Path != wantPath || !strings.Contains(issues[0].Message, "special files") {
		t.Fatalf("local Claude special-file issues = %#v", issues)
	}
}
