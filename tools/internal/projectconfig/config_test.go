package projectconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRejectsUnknownFields(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".harness"), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `{"schema_version":1,"profile":"template","project":{},"public_guard":{},"unknown":true}`
	if err := os.WriteFile(filepath.Join(root, ".harness", "project.json"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("error = %v", err)
	}
}

func TestReadyProblemsRequireProjectSpecificIdentity(t *testing.T) {
	if got := ReadyProblems(Defaults); len(got) != 7 {
		t.Fatalf("problems = %v", got)
	}
	project := Defaults
	project.Name = "Example Tool"
	project.BinaryName = "example-tool"
	project.GoModule = "github.com/acme/example-tool"
	project.GitHubOwner = "acme"
	project.GitHubRepository = "example-tool"
	project.Description = "An example command-line tool."
	project.FormulaClass = "ExampleTool"
	project.SecurityContact = "security@acme.example"
	if got := ReadyProblems(project); len(got) != 0 {
		t.Fatalf("problems = %v", got)
	}

	project.GitHubOwner = Defaults.GitHubOwner
	if got := ReadyProblems(project); len(got) != 0 {
		t.Fatalf("same-owner derived project problems = %v", got)
	}
}

func TestConfigRejectsWindowsReservedBinaryNames(t *testing.T) {
	config := Config{
		SchemaVersion: SchemaVersion,
		Profile:       "template",
		Project:       Defaults,
		PublicGuard:   PublicGuard{DenylistFile: ".harness/denylist.txt"},
	}
	for _, name := range []string{"con", "aux", "prn", "nul", "com1", "com9", "lpt1", "lpt9", "Con", "cOm1", "LpT9"} {
		config.Project.BinaryName = name
		if err := config.Validate(); err == nil || !strings.Contains(err.Error(), "reserved Windows device") {
			t.Fatalf("Validate() accepted binary_name %q: %v", name, err)
		}
	}
	for _, name := range []string{"console", "auxiliary", "null", "com0", "com10", "lpt0", "lpt10"} {
		config.Project.BinaryName = name
		if err := config.Validate(); err != nil {
			t.Fatalf("Validate() rejected binary_name %q: %v", name, err)
		}
	}
	for _, name := range []string{"Con", "cOm1", "LpT9"} {
		if !isWindowsReservedBaseName(name) {
			t.Fatalf("isWindowsReservedBaseName(%q) = false", name)
		}
	}
}
