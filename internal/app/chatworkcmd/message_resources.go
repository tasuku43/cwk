package chatworkcmd

import (
	"context"

	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

// executeMessageResources performs only the selected bounded collection
// reads. It deliberately never reads the message body or interprets provider
// notation; explicit typed message references on files/tasks own the join.
func (s *Service) executeMessageResources(ctx context.Context, binding authn.BindingID, request chatwork.Request) (chatwork.Result, error) {
	resources := chatwork.MessageResources{
		Room:    request.Room,
		Message: request.Message,
		Kind:    request.MessageResourcesKind,
	}
	if request.MessageResourcesKind == chatwork.MessageResourceFiles || request.MessageResourcesKind == chatwork.MessageResourceAll {
		providerRequest := chatwork.Request{Task: chatwork.TaskFilesList, Room: request.Room}
		source, err := s.executeProvider(ctx, binding, providerRequest)
		if err != nil {
			return chatwork.Result{}, err
		}
		if err := validateResourceSourceBound("files", len(source.Files), source.Coverage); err != nil {
			return chatwork.Result{}, err
		}
		matches := make([]chatwork.File, 0, len(source.Files))
		for _, file := range source.Files {
			if file.Room == request.Room && file.Message == request.Message {
				matches = append(matches, file)
			}
		}
		resources.Files = &chatwork.FileMessageResourceSource{
			SourceCount: len(source.Files),
			Coverage:    source.Coverage,
			Absence:     chatwork.MessageResourceAbsenceUnknown,
			Matches:     matches,
		}
	}
	if request.MessageResourcesKind == chatwork.MessageResourceTasks || request.MessageResourcesKind == chatwork.MessageResourceAll {
		providerRequest := chatwork.Request{Task: chatwork.TaskRoomTasksList, Room: request.Room}
		source, err := s.executeProvider(ctx, binding, providerRequest)
		if err != nil {
			return chatwork.Result{}, err
		}
		if err := validateResourceSourceBound("room tasks", len(source.Tasks), source.Coverage); err != nil {
			return chatwork.Result{}, err
		}
		matches := make([]chatwork.WorkTask, 0, len(source.Tasks))
		for _, task := range source.Tasks {
			if task.Room.Ref == request.Room && task.Message == request.Message {
				matches = append(matches, task)
			}
		}
		resources.Tasks = &chatwork.RoomTaskMessageResourceSource{
			SourceCount: len(source.Tasks),
			Coverage:    source.Coverage,
			Absence:     chatwork.MessageResourceAbsenceUnknown,
			Matches:     matches,
		}
	}
	result := chatwork.Result{
		Task:             chatwork.TaskMessagesResources,
		MessageResources: &resources,
	}
	if err := result.ValidateFor(request); err != nil {
		return chatwork.Result{}, fault.Wrap(fault.KindContract, "chatwork_result_invalid", "Chatwork タスクアダプターが無効な型付き結果を返しました", false, err)
	}
	return result, nil
}

func validateResourceSourceBound(name string, sourceCount int, coverage chatwork.Coverage) error {
	if sourceCount > chatwork.MaxMessageResourceSourceItems || sourceCount < 0 ||
		coverage.Kind != "provider_window" || coverage.Limit != chatwork.MaxMessageResourceSourceItems || coverage.Complete {
		return fault.New(fault.KindContract, "chatwork_result_invalid", "メッセージ関連リソースの取得範囲が固定上限と一致しません", false,
			fault.NextAction{Command: "help messages resources", Reason: name + " の取得範囲が宣言済みの上限と一致するよう、型付きアダプター契約を修正してください。"})
	}
	return nil
}
