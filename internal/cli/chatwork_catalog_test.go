package cli

import "testing"

func TestChatworkCatalogSpecsValidateWithPublicCatalog(t *testing.T) {
	if err := DefaultCatalog().Validate(); err != nil {
		t.Fatalf("Chatwork catalog validation failed: %v", err)
	}
}

func TestChatworkCatalogContainsEveryTypedTaskOnce(t *testing.T) {
	seen := make(map[string]string)
	for _, command := range chatworkCommandSpecs() {
		if command.chatwork == nil {
			t.Fatalf("command %q has no Chatwork task binding", command.Path)
		}
		task := string(command.chatwork.Task)
		if previous, exists := seen[task]; exists {
			t.Fatalf("task %q is bound by %q and %q", task, previous, command.Path)
		}
		seen[task] = command.Path
	}
	if len(seen) != 33 {
		t.Fatalf("typed Chatwork task bindings = %d, want 33", len(seen))
	}
}
