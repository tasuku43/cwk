package chatwork

import "testing"

func TestMessageResourcesResultBindsExactSelectedSourcesAndPreservesDuplicates(t *testing.T) {
	room := messageResourcesReference(t, ReferenceRoom, "42")
	otherRoom := messageResourcesReference(t, ReferenceRoom, "43")
	message := messageResourcesReference(t, ReferenceMessage, "100")
	otherMessage := messageResourcesReference(t, ReferenceMessage, "101")
	account := Account{Ref: messageResourcesReference(t, ReferenceAccount, "7")}
	file := File{Ref: messageResourcesReference(t, ReferenceFile, "501"), Room: room, Account: account, Message: message}
	task := WorkTask{
		Ref: messageResourcesReference(t, ReferenceTask, "601"), Room: Room{Ref: room}, Account: account,
		AssignedBy: Account{Ref: messageResourcesReference(t, ReferenceAccount, "8")}, Message: message,
	}
	request := Request{Task: TaskMessagesResources, Room: room, Message: message, MessageResourcesKind: MessageResourceAll}
	result := Result{Task: TaskMessagesResources, MessageResources: &MessageResources{
		Room: room, Message: message, Kind: MessageResourceAll,
		Files: &FileMessageResourceSource{
			SourceCount: 2, Coverage: messageResourcesDomainCoverage(), Absence: MessageResourceAbsenceUnknown, Matches: []File{file, file},
		},
		Tasks: &RoomTaskMessageResourceSource{
			SourceCount: 1, Coverage: messageResourcesDomainCoverage(), Absence: MessageResourceAbsenceUnknown, Matches: []WorkTask{task},
		},
	}}
	if err := result.ValidateFor(request); err != nil {
		t.Fatalf("valid duplicate-preserving message resources result rejected: %v", err)
	}

	wrongRoom := result
	wrongRoom.MessageResources = cloneMessageResources(result.MessageResources)
	wrongRoom.MessageResources.Files.Matches[0].Room = otherRoom
	if err := wrongRoom.ValidateFor(request); err == nil {
		t.Fatal("cross-room file match passed validation")
	}
	wrongMessage := result
	wrongMessage.MessageResources = cloneMessageResources(result.MessageResources)
	wrongMessage.MessageResources.Tasks.Matches[0].Message = otherMessage
	if err := wrongMessage.ValidateFor(request); err == nil {
		t.Fatal("different-message task match passed validation")
	}
	wrongKind := result
	wrongKind.MessageResources = cloneMessageResources(result.MessageResources)
	wrongKind.MessageResources.Kind = MessageResourceFiles
	if err := wrongKind.ValidateFor(request); err == nil {
		t.Fatal("kind-mismatched result passed validation")
	}
	missingSelectedSource := result
	missingSelectedSource.MessageResources = cloneMessageResources(result.MessageResources)
	missingSelectedSource.MessageResources.Tasks = nil
	if err := missingSelectedSource.ValidateFor(request); err == nil {
		t.Fatal("result missing selected source passed validation")
	}
	overLimit := result
	overLimit.MessageResources = cloneMessageResources(result.MessageResources)
	overLimit.MessageResources.Files.SourceCount = MaxMessageResourceSourceItems + 1
	if err := overLimit.ValidateFor(request); err == nil {
		t.Fatal("over-limit source count passed validation")
	}
	completeSource := result
	completeSource.MessageResources = cloneMessageResources(result.MessageResources)
	completeSource.MessageResources.Tasks.Coverage.Complete = true
	if err := completeSource.ValidateFor(request); err == nil {
		t.Fatal("bounded source claimed completeness")
	}
}

func TestMessageResourcesRequestRequiresExactReferencesAndOnlyDeclaredInputs(t *testing.T) {
	room := messageResourcesReference(t, ReferenceRoom, "42")
	message := messageResourcesReference(t, ReferenceMessage, "100")
	valid := Request{Task: TaskMessagesResources, Room: room, Message: message, MessageResourcesKind: MessageResourceFiles}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid message resources request rejected: %v", err)
	}
	for _, invalid := range []Request{
		{Task: TaskMessagesResources, Room: room, Message: message},
		{Task: TaskMessagesResources, Room: room, Message: message, MessageResourcesKind: "download"},
		{Task: TaskMessagesResources, Room: room, Message: message, MessageResourcesKind: MessageResourceAll, ForceRecent: true},
		{Task: TaskMessagesResources, Room: room, Message: Reference{Kind: ReferenceMessage, Value: "0100"}, MessageResourcesKind: MessageResourceFiles},
		{Task: TaskMessagesList, Room: room, Message: message, MessageResourcesKind: MessageResourceFiles},
	} {
		if err := invalid.Validate(); err == nil {
			t.Errorf("invalid request passed: %+v", invalid)
		}
	}
}

func messageResourcesReference(t *testing.T, kind ReferenceKind, value string) Reference {
	t.Helper()
	reference, err := NewReference(kind, value)
	if err != nil {
		t.Fatal(err)
	}
	return reference
}

func messageResourcesDomainCoverage() Coverage {
	return Coverage{Kind: "provider_window", Limit: MaxMessageResourceSourceItems, Complete: false}
}

func cloneMessageResources(resources *MessageResources) *MessageResources {
	clone := *resources
	if resources.Files != nil {
		files := *resources.Files
		files.Matches = append([]File{}, resources.Files.Matches...)
		clone.Files = &files
	}
	if resources.Tasks != nil {
		tasks := *resources.Tasks
		tasks.Matches = append([]WorkTask{}, resources.Tasks.Matches...)
		clone.Tasks = &tasks
	}
	return &clone
}
