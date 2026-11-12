package chatwork

import "fmt"

const MaxMessageResourceSourceItems = 100

type MessageResourceAbsenceStatus string

const MessageResourceAbsenceUnknown MessageResourceAbsenceStatus = "unknown"

// MessageResourceKind selects which bounded room collections are inspected
// for items linked to one exact message.
type MessageResourceKind string

const (
	MessageResourceFiles MessageResourceKind = "files"
	MessageResourceTasks MessageResourceKind = "tasks"
	MessageResourceAll   MessageResourceKind = "all"
)

// FileMessageResourceSource records the bounded file source before the exact
// message-reference filter. Matches retain provider order and duplicates.
type FileMessageResourceSource struct {
	SourceCount int
	Coverage    Coverage
	Absence     MessageResourceAbsenceStatus
	Matches     []File
}

// RoomTaskMessageResourceSource records the bounded room-task source before
// the exact message-reference filter. Matches retain provider order and
// duplicates.
type RoomTaskMessageResourceSource struct {
	SourceCount int
	Coverage    Coverage
	Absence     MessageResourceAbsenceStatus
	Matches     []WorkTask
}

// MessageResources is the typed result of the message-to-resource task. It
// contains only explicit file/task message references in the exact room and
// does not imply that the message itself exists or that the source is complete.
type MessageResources struct {
	Room    Reference
	Message Reference
	Kind    MessageResourceKind
	Files   *FileMessageResourceSource
	Tasks   *RoomTaskMessageResourceSource
}

func validMessageResourceKind(kind MessageResourceKind) bool {
	switch kind {
	case MessageResourceFiles, MessageResourceTasks, MessageResourceAll:
		return true
	default:
		return false
	}
}

func validateMessageResourcesRequestExtras(request Request) error {
	filter := request.MessageFilter
	if request.TaskRef != (Reference{}) || request.File != (Reference{}) || request.Invite != (Reference{}) || request.Request != (Reference{}) ||
		request.Account != (Reference{}) || request.AssignedBy != (Reference{}) || len(request.MessageRecipients) != 0 || request.MessageRecipients != nil || request.MessageReplyTarget != nil ||
		len(request.Admins) != 0 || request.Admins != nil || len(request.Members) != 0 || request.Members != nil || len(request.ReadonlyMembers) != 0 || request.ReadonlyMembers != nil || len(request.Assignees) != 0 || request.Assignees != nil ||
		request.Name != "" || request.Description != "" || request.Icon != "" || request.Body != "" || request.BodyFile != "" || request.Status != "" || request.RoomAction != "" || request.Limit != 0 || request.LimitType != "" ||
		request.ForceRecent || request.SelfUnread || request.CreateDownloadURL || request.InviteCode != "" || request.InviteRegenerateCode || request.InviteEnabled || request.InviteNeedsApproval || request.InviteApprovalSet || request.DescriptionSet ||
		request.FilePath != "" || request.FileMessage != "" || request.MemberQuery != "" || request.RoomQuery != "" ||
		filter.Senders != nil || filter.Contains != nil || filter.Period != (MessagePeriod{}) || filter.Context != "" || filter.StartIndex != 0 || filter.Count != 0 ||
		request.MessageRelationFetchLimit != 0 || request.MessageAround != nil {
		return fmt.Errorf("messages.resources request contains fields outside its exact room, message, and kind inputs")
	}
	return nil
}

func validateMessageResources(resources MessageResources) error {
	if !validMessageResourceKind(resources.Kind) {
		return fmt.Errorf("message resources kind is missing or invalid")
	}
	if err := validateResultReference("message resources room", resources.Room, ReferenceRoom, false); err != nil {
		return err
	}
	if err := validateResultReference("message resources message", resources.Message, ReferenceMessage, false); err != nil {
		return err
	}
	wantFiles := resources.Kind == MessageResourceFiles || resources.Kind == MessageResourceAll
	wantTasks := resources.Kind == MessageResourceTasks || resources.Kind == MessageResourceAll
	if (resources.Files != nil) != wantFiles || (resources.Tasks != nil) != wantTasks {
		return fmt.Errorf("message resources sources do not match the selected kind")
	}
	if resources.Files != nil {
		if err := validateFileMessageResourceSource(resources.Room, resources.Message, *resources.Files); err != nil {
			return err
		}
	}
	if resources.Tasks != nil {
		if err := validateRoomTaskMessageResourceSource(resources.Room, resources.Message, *resources.Tasks); err != nil {
			return err
		}
	}
	return nil
}

func validateFileMessageResourceSource(room, message Reference, source FileMessageResourceSource) error {
	if source.Matches == nil {
		return fmt.Errorf("message resources file matches must be an explicit collection")
	}
	if err := validateMessageResourceCoverage("file", source.SourceCount, len(source.Matches), source.Coverage, source.Absence); err != nil {
		return err
	}
	for index, file := range source.Matches {
		if err := validateResultFile(fmt.Sprintf("message resources file[%d]", index), file); err != nil {
			return err
		}
		if file.Room != room || file.Message != message {
			return fmt.Errorf("message resources file[%d] does not match the exact room and message", index)
		}
	}
	return nil
}

func validateRoomTaskMessageResourceSource(room, message Reference, source RoomTaskMessageResourceSource) error {
	if source.Matches == nil {
		return fmt.Errorf("message resources task matches must be an explicit collection")
	}
	if err := validateMessageResourceCoverage("room task", source.SourceCount, len(source.Matches), source.Coverage, source.Absence); err != nil {
		return err
	}
	for index, task := range source.Matches {
		if err := validateResultWorkTask(fmt.Sprintf("message resources task[%d]", index), task, false); err != nil {
			return err
		}
		if task.Room.Ref != room || task.Message != message {
			return fmt.Errorf("message resources task[%d] does not match the exact room and message", index)
		}
	}
	return nil
}

func validateMessageResourceCoverage(label string, sourceCount, matchCount int, coverage Coverage, absence MessageResourceAbsenceStatus) error {
	if sourceCount < 0 || sourceCount > MaxMessageResourceSourceItems {
		return fmt.Errorf("message resources %s source count is outside the provider bound", label)
	}
	if matchCount > sourceCount {
		return fmt.Errorf("message resources %s match count exceeds its source count", label)
	}
	if coverage.Kind != "provider_window" || coverage.Limit != MaxMessageResourceSourceItems || coverage.Complete {
		return fmt.Errorf("message resources %s source must retain the bounded incomplete provider window", label)
	}
	if absence != MessageResourceAbsenceUnknown {
		return fmt.Errorf("message resources %s source must keep absence unknown", label)
	}
	return nil
}
