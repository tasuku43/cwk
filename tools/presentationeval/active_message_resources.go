package main

import (
	"encoding/json"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

// activeMessageResourcesFixture keeps the provider facts, joined task result,
// and exact show results independent from the task projection.
type activeMessageResourcesFixture struct {
	Room        chatwork.Reference
	Message     chatwork.Reference
	FilesSource chatwork.Result
	TasksSource chatwork.Result
	Resources   chatwork.Result
	FileShow    chatwork.Result
	TaskShow    chatwork.Result
	AnswerKey   json.RawMessage
}

// messageResourcesReadinessScenario records the intended fixture prompt,
// sequence, and limits for a future agent run. The sequence-level contract
// test does not observe an agent, so its budgets are constraints rather than
// measured tool or write counts.
type messageResourcesReadinessScenario struct {
	ID                       string
	UserPrompt               string
	ExpectedCommandArgv      [][]string
	ProviderCallBudget       int
	ExternalProcessingBudget int
	WriteCallBudget          int
	AnswerShape              string
	AnswerKey                json.RawMessage
	Operations               map[string]fixtureOperation
}

func messageResourcesFixture() activeMessageResourcesFixture {
	room := ref(chatwork.ReferenceRoom, "4201")
	otherRoom := ref(chatwork.ReferenceRoom, "4202")
	messageRef := ref(chatwork.ReferenceMessage, "9101")
	otherMessage := ref(chatwork.ReferenceMessage, "9102")
	uploader := account("3201", "Aki")
	assignee := account("3202", "Beni")
	assigner := account("3203", "Chika")

	files := []chatwork.File{
		{Ref: ref(chatwork.ReferenceFile, "7711"), Room: room, Account: uploader, Message: messageRef, Name: "release-notes.txt", Size: 128},
		{Ref: ref(chatwork.ReferenceFile, "7712"), Room: room, Account: uploader, Message: otherMessage, Name: "same-room-different-message.txt", Size: 256},
		{Ref: ref(chatwork.ReferenceFile, "7713"), Room: otherRoom, Account: uploader, Message: messageRef, Name: "same-message-different-room.txt", Size: 512},
	}
	tasks := []chatwork.WorkTask{
		{Ref: ref(chatwork.ReferenceTask, "8811"), Room: chatwork.Room{Ref: room}, Account: assignee, AssignedBy: assigner, Message: messageRef, Body: "Review the linked release notes.", Status: "open"},
		{Ref: ref(chatwork.ReferenceTask, "8812"), Room: chatwork.Room{Ref: room}, Account: assignee, AssignedBy: assigner, Message: otherMessage, Body: "Different message task.", Status: "open"},
		{Ref: ref(chatwork.ReferenceTask, "8813"), Room: chatwork.Room{Ref: otherRoom}, Account: assignee, AssignedBy: assigner, Message: messageRef, Body: "Same message in another room.", Status: "open"},
	}
	coverage := chatwork.Coverage{Kind: "provider_window", Limit: 100, Complete: false, Description: "synthetic bounded resource source"}
	filesSource := chatwork.Result{Task: chatwork.TaskFilesList, Coverage: coverage, Files: files}
	tasksSource := chatwork.Result{Task: chatwork.TaskRoomTasksList, Coverage: coverage, Tasks: tasks}
	resources := chatwork.Result{
		Task: chatwork.TaskMessagesResources,
		MessageResources: &chatwork.MessageResources{
			Room: room, Message: messageRef, Kind: chatwork.MessageResourceAll,
			Files: &chatwork.FileMessageResourceSource{
				SourceCount: len(files), Coverage: coverage, Absence: chatwork.MessageResourceAbsenceUnknown,
				Matches: []chatwork.File{files[0]},
			},
			Tasks: &chatwork.RoomTaskMessageResourceSource{
				SourceCount: len(tasks), Coverage: coverage, Absence: chatwork.MessageResourceAbsenceUnknown,
				Matches: []chatwork.WorkTask{tasks[0]},
			},
		},
	}
	fileShow := chatwork.Result{
		Task: chatwork.TaskFilesShow, Coverage: chatwork.Coverage{Kind: "single_operation", Complete: true},
		Files: []chatwork.File{files[0]},
	}
	taskShow := chatwork.Result{
		Task: chatwork.TaskRoomTasksShow, Coverage: chatwork.Coverage{Kind: "single_operation", Complete: true},
		Tasks: []chatwork.WorkTask{tasks[0]},
	}
	return activeMessageResourcesFixture{
		Room: room, Message: messageRef, FilesSource: filesSource, TasksSource: tasksSource,
		Resources: resources, FileShow: fileShow, TaskShow: taskShow,
		AnswerKey: raw(`{"room_ref":"4201","message_ref":"9101","kind":"all","files":{"source_count":3,"limit":100,"complete":false,"absence":"unknown","matched":1,"file_refs":["7711"]},"tasks":{"source_count":3,"limit":100,"complete":false,"absence":"unknown","matched":1,"task_refs":["8811"]},"verified":{"file_show":{"room_ref":"4201","file_ref":"7711","message_ref":"9101","name":"release-notes.txt"},"room_task_show":{"room_ref":"4201","task_ref":"8811","message_ref":"9101","body":"Review the linked release notes.","status":"open"}}}`),
	}
}

func activeMessageResourcesScenario() messageResourcesReadinessScenario {
	fixture := messageResourcesFixture()
	return messageResourcesReadinessScenario{
		ID:         "active.message-resources",
		UserPrompt: "In room 4201, find the files and room tasks linked to message 9101. Verify each returned canonical reference by reading that exact file or task, then report the source counts, bounded coverage, uncertainty, and confirmed details. Use only cwk; do not use jq, grep, an external parser, source inspection, another messages read, or a caller-side join. Do not infer resource events from message notation or claim the message exists.",
		ExpectedCommandArgv: [][]string{
			{"messages", "resources", "--room", "4201", "--message", "9101", "--kind", "all"},
			{"files", "show", "--room", "4201", "--file", "7711"},
			{"room-tasks", "show", "--room", "4201", "--task", "8811"},
		},
		ProviderCallBudget:       4,
		ExternalProcessingBudget: 0,
		WriteCallBudget:          0,
		AnswerShape:              `{"room_ref":"<room-ref>","message_ref":"<message-ref>","kind":"files|tasks|all","files":{"source_count":0,"limit":100,"complete":false,"absence":"unknown","matched":0,"file_refs":["<file-ref>"]},"tasks":{"source_count":0,"limit":100,"complete":false,"absence":"unknown","matched":0,"task_refs":["<task-ref>"]},"verified":{"file_show":{"room_ref":"<room-ref>","file_ref":"<file-ref>","message_ref":"<message-ref>","name":"<name>"},"room_task_show":{"room_ref":"<room-ref>","task_ref":"<task-ref>","message_ref":"<message-ref>","body":"<body>","status":"open|done"}}}`,
		AnswerKey:                fixture.AnswerKey,
		Operations: map[string]fixtureOperation{
			"messages resources": operation("messages resources", fixture.Resources, map[string]string{"--room": fixture.Room.Value, "--message": fixture.Message.Value, "--kind": "all"}),
			"files show":         operation("files show", fixture.FileShow, map[string]string{"--room": fixture.Room.Value, "--file": fixture.Resources.MessageResources.Files.Matches[0].Ref.Value}),
			"room-tasks show":    operation("room-tasks show", fixture.TaskShow, map[string]string{"--room": fixture.Room.Value, "--task": fixture.Resources.MessageResources.Tasks.Matches[0].Ref.Value}),
		},
	}
}
