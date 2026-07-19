package cli

import (
	"reflect"
	"strings"
	"testing"
)

func configurablePaths(catalog Catalog) []string {
	commands := catalog.ConfigurableCommands()
	paths := make([]string, len(commands))
	for index, command := range commands {
		paths[index] = command.Path
	}
	return paths
}

func catalogPaths(catalog Catalog) []string {
	commands := catalog.Commands()
	paths := make([]string, len(commands))
	for index, command := range commands {
		paths[index] = command.Path
	}
	return paths
}

func TestDefaultCatalogDeclaresSelectableLeavesAndExactControlPlane(t *testing.T) {
	catalog := DefaultCatalog()

	for _, path := range []string{"doctor", "version"} {
		command, found := catalog.Lookup(path)
		if !found || !command.Configurable {
			t.Fatalf("%q must be configurable: found=%v command=%+v", path, found, command)
		}
	}
	for _, command := range chatworkCommandSpecs() {
		if !command.Configurable {
			t.Fatalf("Chatwork leaf %q must be configurable", command.Path)
		}
	}
	always := catalog.AlwaysCommands()
	if got, want := catalogPaths(NewCatalog(always...)), []string{"help", "config show", "config edit"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("always-on command paths = %v, want %v", got, want)
	}
}

func TestActiveViewPreservesCatalogOrderNotSelectionOrder(t *testing.T) {
	catalog := DefaultCatalog()
	view, stale, err := catalog.ActiveView([]string{"version", "doctor"})
	if err != nil {
		t.Fatalf("ActiveView returned an error: %v", err)
	}
	if len(stale) != 0 {
		t.Fatalf("stale = %v, want none", stale)
	}
	want := make([]string, 0)
	for _, command := range catalog.Commands() {
		if !command.Configurable || command.Path == "doctor" || command.Path == "version" {
			want = append(want, command.Path)
		}
	}
	if got := catalogPaths(view); !reflect.DeepEqual(got, want) {
		t.Fatalf("active paths = %v, want catalog order %v", got, want)
	}
}

func TestActiveViewWithEveryConfigurableCommandReconstructsFullCatalog(t *testing.T) {
	catalog := DefaultCatalog()
	view, stale, err := catalog.ActiveView(configurablePaths(catalog))
	if err != nil {
		t.Fatalf("ActiveView returned an error: %v", err)
	}
	if len(stale) != 0 {
		t.Fatalf("stale = %v, want none", stale)
	}
	if got, want := catalogPaths(view), catalogPaths(catalog); !reflect.DeepEqual(got, want) {
		t.Fatalf("active paths = %v, want full catalog %v", got, want)
	}
}

func TestActiveViewLeavesNewConfigurableCommandOff(t *testing.T) {
	base := DefaultCatalog()
	enabled := configurablePaths(base)
	commands := base.Commands()
	future := utilitySpec("future")
	future.Configurable = true
	extended := NewCatalog(append(commands, future)...)

	view, stale, err := extended.ActiveView(enabled)
	if err != nil {
		t.Fatalf("ActiveView returned an error: %v", err)
	}
	if len(stale) != 0 {
		t.Fatalf("stale = %v, want none", stale)
	}
	if _, found := view.Lookup("future"); found {
		t.Fatal("new command unexpectedly entered a previously saved allowlist")
	}
}

func TestActiveViewReportsUnknownCanonicalSelectionsAsStale(t *testing.T) {
	view, stale, err := DefaultCatalog().ActiveView([]string{"retired command"})
	if err != nil {
		t.Fatalf("ActiveView returned an error: %v", err)
	}
	if want := []string{"retired command"}; !reflect.DeepEqual(stale, want) {
		t.Fatalf("stale = %v, want %v", stale, want)
	}
	if _, found := view.Lookup("retired command"); found {
		t.Fatal("stale command unexpectedly entered the active view")
	}
}

func TestActiveViewAllowsControlPlaneOnly(t *testing.T) {
	catalog := DefaultCatalog()
	view, stale, err := catalog.ActiveView([]string{})
	if err != nil {
		t.Fatalf("ActiveView returned an error: %v", err)
	}
	if len(stale) != 0 {
		t.Fatalf("stale = %v, want none", stale)
	}
	for _, command := range view.Commands() {
		if command.Configurable {
			t.Fatalf("configurable command %q remained in an empty selection", command.Path)
		}
	}
	for _, command := range catalog.AlwaysCommands() {
		if _, found := view.Lookup(command.Path); !found {
			t.Fatalf("always command %q is missing", command.Path)
		}
	}
}

func TestActiveViewRejectsInvalidSelectionDocuments(t *testing.T) {
	tests := []struct {
		name    string
		enabled []string
		want    string
	}{
		{name: "always command", enabled: []string{"help"}, want: "always enabled"},
		{name: "duplicate", enabled: []string{"doctor", "doctor"}, want: "duplicated"},
		{name: "noncanonical", enabled: []string{"Rooms List"}, want: "command path is missing or invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := DefaultCatalog().ActiveView(test.enabled)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ActiveView error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestActiveViewRejectsVisibleConsumerWithoutProducer(t *testing.T) {
	_, _, err := DefaultCatalog().ActiveView([]string{"messages mark-read"})
	if err == nil || !strings.Contains(err.Error(), "has no visible producer") {
		t.Fatalf("ActiveView error = %v, want missing visible producer", err)
	}
}

func TestActiveViewRejectsClosedRequiredReferenceCycle(t *testing.T) {
	_, _, err := DefaultCatalog().ActiveView([]string{"messages show"})
	if err == nil || !strings.Contains(err.Error(), "closed required-reference cycle") {
		t.Fatalf("ActiveView error = %v, want closed required-reference cycle", err)
	}
}

func TestActiveViewRejectsRecoveryToDisabledCommand(t *testing.T) {
	_, _, err := DefaultCatalog().ActiveView([]string{"rooms list", "messages send"})
	if err == nil || !strings.Contains(err.Error(), `next command "messages list" is not an exact catalog path`) {
		t.Fatalf("ActiveView error = %v, want disabled recovery rejection", err)
	}
}

func TestActiveViewAllowsTerminalProducer(t *testing.T) {
	view, _, err := DefaultCatalog().ActiveView([]string{"rooms list"})
	if err != nil {
		t.Fatalf("terminal producer was rejected: %v", err)
	}
	if _, found := view.Lookup("rooms list"); !found {
		t.Fatal("terminal producer is missing from active view")
	}
}

func TestActiveViewDoesNotAliasFullCatalog(t *testing.T) {
	full := DefaultCatalog()
	view, _, err := full.ActiveView([]string{"doctor"})
	if err != nil {
		t.Fatalf("ActiveView returned an error: %v", err)
	}
	for index := range view.commands {
		if view.commands[index].Path != "doctor" {
			continue
		}
		view.commands[index].Agent.Inputs[0].Description = "mutated"
		view.commands[index].Agent.Errors[0].NextActions[0].Reason = "mutated"
	}
	doctor, found := full.Lookup("doctor")
	if !found {
		t.Fatal("full catalog lost doctor")
	}
	if doctor.Agent.Inputs[0].Description == "mutated" || doctor.Agent.Errors[0].NextActions[0].Reason == "mutated" {
		t.Fatal("active view aliases nested full-catalog contract data")
	}
}
