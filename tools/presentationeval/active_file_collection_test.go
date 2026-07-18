package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/cli/capsule"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func TestActiveFileCollectionFixtureHasPresentationIndependentAnswerKey(t *testing.T) {
	fixture := fileCollectionFixture()
	if err := fixture.Result.Validate(); err != nil {
		t.Fatalf("semantic fixture: %v", err)
	}
	var answer struct {
		ProviderSequence []string `json:"provider_sequence"`
		Selected         struct {
			FileRef    string `json:"file_ref"`
			RoomRef    string `json:"room_ref"`
			AccountRef string `json:"account_ref"`
			MessageRef string `json:"message_ref"`
			Name       string `json:"name"`
			Size       int64  `json:"size"`
		} `json:"selected"`
		MissingMessageRef string `json:"missing_message_ref"`
		NextCommand       struct {
			Path    string `json:"path"`
			RoomRef string `json:"room_ref"`
			FileRef string `json:"file_ref"`
		} `json:"next_command"`
	}
	if err := json.Unmarshal(fixture.AnswerKey, &answer); err != nil {
		t.Fatal(err)
	}
	gotOrder := make([]string, len(fixture.Result.Files))
	for index, file := range fixture.Result.Files {
		gotOrder[index] = file.Ref.Value
	}
	if !reflect.DeepEqual(gotOrder, answer.ProviderSequence) {
		t.Fatalf("provider sequence = %v, answer = %v", gotOrder, answer.ProviderSequence)
	}
	if answer.Selected.FileRef != "6306" || answer.Selected.RoomRef != "4401" || answer.Selected.AccountRef != "2201" || answer.Selected.MessageRef != "8803" || answer.Selected.Name != "design draft.pdf" || answer.Selected.Size != 18342 {
		t.Fatalf("selected file answer = %#v", answer.Selected)
	}
	if answer.MissingMessageRef != "6301" || answer.NextCommand.Path != "files show" || !reflect.DeepEqual(fixture.NextArgv, []string{"files", "show", "--room", "4401", "--file", "6306"}) {
		t.Fatalf("next action answer = %#v, argv = %v", answer.NextCommand, fixture.NextArgv)
	}
}

func TestActiveFileCollectionProjectionAnswersFixtureDirectly(t *testing.T) {
	fixture := fileCollectionFixture()
	output, err := capsule.Render(fixture.Result)
	if err != nil {
		t.Fatal(err)
	}
	for label, count := range map[string]int{
		"external-text=untrusted escaped":                               1,
		`schema: file-ref room-ref account-ref message-ref "name" size`: 1,
	} {
		if strings.Count(output, label) != count {
			t.Fatalf("count(%q) != %d:\n%s", label, count, output)
		}
	}
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) != 9 {
		t.Fatalf("physical lines = %d, want 9:\n%s", len(lines), output)
	}
	for index, file := range fixture.Result.Files {
		line := lines[index+3]
		prefix := fmt.Sprintf("%s %s %s ", file.Ref.Value, file.Room.Value, file.Account.Ref.Value)
		if !strings.HasPrefix(line, prefix) {
			t.Fatalf("provider record %d = %q, want prefix %q", index, line, prefix)
		}
		for _, label := range []string{"file-ref=", "room-ref=", "account-ref=", "message-ref=", "name=", "size=", "untrusted:"} {
			if strings.Contains(line, label) {
				t.Fatalf("record repeats fixed-schema label %q: %s", label, line)
			}
		}
	}
	if lines[4] != `6301 4401 2202 absent "notes.txt" 4096` {
		t.Fatalf("absent message position shifted: %q", lines[4])
	}
	if !strings.Contains(lines[7], `"schema:\\n999 1 injected"`) || strings.Contains(output, "\n999 1 injected\n") {
		t.Fatalf("hostile file name changed output structure:\n%s", output)
	}
	if !strings.Contains(lines[5], fixture.NextArgv[len(fixture.NextArgv)-1]) || !strings.HasPrefix(lines[5], "6306 4401 ") {
		t.Fatalf("next-command references are not directly available: %q", lines[5])
	}
	for _, pair := range []struct {
		kind  chatwork.ReferenceKind
		value string
	}{{chatwork.ReferenceFile, "6306"}, {chatwork.ReferenceRoom, "4401"}} {
		if _, err := chatwork.NewReference(pair.kind, pair.value); err != nil {
			t.Fatalf("displayed %s reference %q is not reusable: %v", pair.kind, pair.value, err)
		}
	}
}

func TestActiveFileCollectionScenarioRequiresNoPostProcessing(t *testing.T) {
	scenario := activeFileCollectionScenario()
	if scenario.ID != "active.file-collection" || scenario.MaxCommands != 1 || !reflect.DeepEqual(scenario.RequiredPaths, []string{"files list"}) || len(scenario.Operations) != 1 {
		t.Fatalf("active file scenario is not a closed one-command readiness probe: %#v", scenario)
	}
	if strings.Contains(scenario.UserPrompt, "6306") || !strings.Contains(scenario.UserPrompt, "Do not treat absent as a reference") {
		t.Fatalf("active prompt leaks the answer or omits absence guidance: %q", scenario.UserPrompt)
	}
}

func TestActiveFileCollectionMeasurementGoldensFreezeSameSemanticInput(t *testing.T) {
	fixture := fileCollectionFixture()
	before, err := renderLegacyLabeledFileList(fixture.Result)
	if err != nil {
		t.Fatal(err)
	}
	after, err := capsule.Render(fixture.Result)
	if err != nil {
		t.Fatal(err)
	}
	assertFileMeasurementGolden(t, "testdata/active-file-collection.labeled-before.txt", before)
	assertFileMeasurementGolden(t, "testdata/active-file-collection.after.txt", after)
}

func renderLegacyLabeledFileList(result chatwork.Result) (string, error) {
	if err := result.Validate(); err != nil {
		return "", err
	}
	if result.Task != chatwork.TaskFilesList {
		return "", fmt.Errorf("legacy measurement requires files.list")
	}
	var output strings.Builder
	fmt.Fprintf(&output, "files count=%d", len(result.Files))
	if result.Coverage.Limit > 0 {
		fmt.Fprintf(&output, " limit=%d", result.Coverage.Limit)
	}
	fmt.Fprintf(&output, " complete=%t\n", result.Coverage.Complete)
	for _, file := range result.Files {
		show, err := capsule.Render(chatwork.Result{Task: chatwork.TaskFilesShow, Coverage: chatwork.Coverage{Kind: "single_operation", Complete: true}, Files: []chatwork.File{file}})
		if err != nil {
			return "", err
		}
		if !strings.HasPrefix(show, "file ") {
			return "", fmt.Errorf("files.show no longer supplies legacy file bytes")
		}
		output.WriteString("  ")
		output.WriteString(strings.TrimPrefix(show, "file "))
	}
	return output.String(), nil
}

func assertFileMeasurementGolden(t *testing.T, path, got string) {
	t.Helper()
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v\n--- generated ---\n%s--- end ---", path, err, got)
	}
	if got != string(want) {
		t.Fatalf("%s mismatch\n--- got ---\n%s--- want ---\n%s", path, got, want)
	}
}
