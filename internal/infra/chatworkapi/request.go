package chatworkapi

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/tasuku43/cwk/internal/domain/chatwork"
)

func (c *Client) buildRequest(input chatwork.Request) (requestSpec, error) {
	if err := input.Validate(); err != nil {
		return requestSpec{}, invalidRequest("Chatwork タスク入力がドメイン検証に失敗しました")
	}
	roomPath := func(suffix string) (string, error) {
		room, err := decimal(input.Room, chatwork.ReferenceRoom)
		if err != nil {
			return "", err
		}
		return "/rooms/" + room + suffix, nil
	}
	messagePath := func() (string, error) {
		path, err := roomPath("/messages/")
		if err != nil {
			return "", err
		}
		message, err := decimal(input.Message, chatwork.ReferenceMessage)
		return path + message, err
	}
	taskPath := func() (string, error) {
		path, err := roomPath("/tasks/")
		if err != nil {
			return "", err
		}
		task, err := decimal(input.TaskRef, chatwork.ReferenceTask)
		return path + task, err
	}

	switch input.Task {
	case chatwork.TaskAccountShow:
		return noBodyRequest(http.MethodGet, "/me", nil), nil
	case chatwork.TaskAccountStatus:
		return noBodyRequest(http.MethodGet, "/my/status", nil), nil
	case chatwork.TaskPersonalTasksList:
		query := url.Values{}
		if input.AssignedBy.Value != "" {
			value, err := decimal(input.AssignedBy, chatwork.ReferenceAccount)
			if err != nil {
				return requestSpec{}, err
			}
			query.Set("assigned_by_account_id", value)
		}
		setOptional(query, "status", input.Status)
		return noBodyRequest(http.MethodGet, "/my/tasks", query), nil
	case chatwork.TaskContactsList:
		return noBodyRequest(http.MethodGet, "/contacts", nil), nil
	case chatwork.TaskRoomsList:
		return noBodyRequest(http.MethodGet, "/rooms", nil), nil
	case chatwork.TaskRoomsCreate:
		if input.Name == "" || len(input.Admins) == 0 {
			return requestSpec{}, invalidRequest("room creation requires a name and at least one administrator")
		}
		form := url.Values{"name": {input.Name}, "members_admin_ids": {joinRefs(input.Admins)}}
		setOptional(form, "members_member_ids", joinRefs(input.Members))
		setOptional(form, "members_readonly_ids", joinRefs(input.ReadonlyMembers))
		setOptional(form, "description", input.Description)
		setOptional(form, "icon_preset", input.Icon)
		if input.InviteEnabled {
			form.Set("link", "1")
			setOptional(form, "link_code", input.InviteCode)
			if input.InviteApprovalSet {
				form.Set("link_need_acceptance", bool01(input.InviteNeedsApproval))
			}
		}
		return formRequest(http.MethodPost, "/rooms", form), nil
	case chatwork.TaskRoomsShow:
		path, err := roomPath("")
		return noBodyRequest(http.MethodGet, path, nil), err
	case chatwork.TaskRoomsUpdate:
		path, err := roomPath("")
		if err != nil {
			return requestSpec{}, err
		}
		form := url.Values{}
		setOptional(form, "name", input.Name)
		setOptional(form, "description", input.Description)
		setOptional(form, "icon_preset", input.Icon)
		if len(form) == 0 {
			return requestSpec{}, invalidRequest("room update requires at least one changed field")
		}
		return formRequest(http.MethodPut, path, form), nil
	case chatwork.TaskRoomsLeave, chatwork.TaskRoomsDelete:
		path, err := roomPath("")
		if err != nil {
			return requestSpec{}, err
		}
		action := "leave"
		if input.Task == chatwork.TaskRoomsDelete {
			action = "delete"
		}
		return formRequest(http.MethodDelete, path, url.Values{"action_type": {action}}), nil
	case chatwork.TaskMembersList:
		path, err := roomPath("/members")
		return noBodyRequest(http.MethodGet, path, nil), err
	case chatwork.TaskMembersReplace:
		path, err := roomPath("/members")
		if err != nil {
			return requestSpec{}, err
		}
		if len(input.Admins) == 0 {
			return requestSpec{}, invalidRequest("member replacement requires at least one administrator")
		}
		form := url.Values{"members_admin_ids": {joinRefs(input.Admins)}}
		setOptional(form, "members_member_ids", joinRefs(input.Members))
		setOptional(form, "members_readonly_ids", joinRefs(input.ReadonlyMembers))
		return formRequest(http.MethodPut, path, form), nil
	case chatwork.TaskMessagesList:
		if len(input.MessageFilter.Senders) != 0 || input.MessageFilter.Period != (chatwork.MessagePeriod{}) || input.MessageFilter.Context != "" || input.MessageFilter.StartIndex != 0 || input.MessageFilter.Count != 0 {
			return requestSpec{}, invalidRequest("message selection must be applied before the Chatwork request boundary")
		}
		path, err := roomPath("/messages")
		if err != nil {
			return requestSpec{}, err
		}
		query := url.Values{}
		if input.ForceRecent {
			query.Set("force", "1")
		}
		return noBodyRequest(http.MethodGet, path, query), nil
	case chatwork.TaskMessagesSend:
		path, err := roomPath("/messages")
		if err != nil {
			return requestSpec{}, err
		}
		if input.Body == "" {
			return requestSpec{}, invalidRequest("message creation requires a body")
		}
		form := url.Values{"body": {input.Body}}
		if input.SelfUnread {
			form.Set("self_unread", "1")
		}
		return formRequest(http.MethodPost, path, form), nil
	case chatwork.TaskMessagesMarkRead, chatwork.TaskMessagesMarkUnread:
		suffix := "/messages/read"
		if input.Task == chatwork.TaskMessagesMarkUnread {
			suffix = "/messages/unread"
		}
		path, err := roomPath(suffix)
		if err != nil {
			return requestSpec{}, err
		}
		form := url.Values{}
		if input.Message.Value != "" {
			message, err := decimal(input.Message, chatwork.ReferenceMessage)
			if err != nil {
				return requestSpec{}, err
			}
			form.Set("message_id", message)
		} else if input.Task == chatwork.TaskMessagesMarkUnread {
			return requestSpec{}, invalidRequest("marking messages unread requires an exact message reference")
		}
		return formRequest(http.MethodPut, path, form), nil
	case chatwork.TaskMessagesShow:
		path, err := messagePath()
		return noBodyRequest(http.MethodGet, path, nil), err
	case chatwork.TaskMessagesUpdate:
		path, err := messagePath()
		if err != nil {
			return requestSpec{}, err
		}
		if input.Body == "" {
			return requestSpec{}, invalidRequest("message update requires a body")
		}
		return formRequest(http.MethodPut, path, url.Values{"body": {input.Body}}), nil
	case chatwork.TaskMessagesDelete:
		path, err := messagePath()
		return noBodyRequest(http.MethodDelete, path, nil), err
	case chatwork.TaskRoomTasksList:
		path, err := roomPath("/tasks")
		if err != nil {
			return requestSpec{}, err
		}
		query := url.Values{}
		for name, ref := range map[string]chatwork.Reference{"account_id": input.Account, "assigned_by_account_id": input.AssignedBy} {
			if ref.Value != "" {
				value, err := decimal(ref, chatwork.ReferenceAccount)
				if err != nil {
					return requestSpec{}, err
				}
				query.Set(name, value)
			}
		}
		setOptional(query, "status", input.Status)
		return noBodyRequest(http.MethodGet, path, query), nil
	case chatwork.TaskRoomTasksCreate:
		path, err := roomPath("/tasks")
		if err != nil {
			return requestSpec{}, err
		}
		if input.Body == "" || len(input.Assignees) == 0 {
			return requestSpec{}, invalidRequest("task creation requires a body and at least one assignee")
		}
		form := url.Values{"body": {input.Body}, "to_ids": {joinRefs(input.Assignees)}}
		if input.Limit != 0 {
			form.Set("limit", strconv.FormatInt(input.Limit, 10))
		}
		setOptional(form, "limit_type", input.LimitType)
		return formRequest(http.MethodPost, path, form), nil
	case chatwork.TaskRoomTasksShow:
		path, err := taskPath()
		return noBodyRequest(http.MethodGet, path, nil), err
	case chatwork.TaskRoomTasksSetStatus:
		path, err := taskPath()
		if err != nil {
			return requestSpec{}, err
		}
		if input.Status == "" {
			return requestSpec{}, invalidRequest("task status update requires a status")
		}
		return formRequest(http.MethodPut, path+"/status", url.Values{"body": {input.Status}}), nil
	case chatwork.TaskFilesList:
		path, err := roomPath("/files")
		if err != nil {
			return requestSpec{}, err
		}
		query := url.Values{}
		if input.Account.Value != "" {
			value, err := decimal(input.Account, chatwork.ReferenceAccount)
			if err != nil {
				return requestSpec{}, err
			}
			query.Set("account_id", value)
		}
		return noBodyRequest(http.MethodGet, path, query), nil
	case chatwork.TaskFilesUpload:
		path, err := roomPath("/files")
		if err != nil {
			return requestSpec{}, err
		}
		return c.multipartRequest(path, input)
	case chatwork.TaskFilesShow:
		path, err := roomPath("/files/")
		if err != nil {
			return requestSpec{}, err
		}
		file, err := decimal(input.File, chatwork.ReferenceFile)
		if err != nil {
			return requestSpec{}, err
		}
		query := url.Values{}
		if input.CreateDownloadURL {
			query.Set("create_download_url", "1")
		}
		return noBodyRequest(http.MethodGet, path+file, query), nil
	case chatwork.TaskInviteLinkShow, chatwork.TaskInviteLinkCreate:
		path, err := roomPath("/link")
		if err != nil {
			return requestSpec{}, err
		}
		if input.Task == chatwork.TaskInviteLinkShow {
			return noBodyRequest(http.MethodGet, path, nil), nil
		}
		return formRequest(http.MethodPost, path, inviteForm(input)), nil
	case chatwork.TaskInviteLinkUpdate, chatwork.TaskInviteLinkDelete:
		invite, err := decimal(input.Invite, chatwork.ReferenceInvite)
		if err != nil {
			return requestSpec{}, err
		}
		path := "/rooms/" + invite + "/link"
		if input.Task == chatwork.TaskInviteLinkDelete {
			return noBodyRequest(http.MethodDelete, path, nil), nil
		}
		form, err := inviteUpdateForm(input)
		if err != nil {
			return requestSpec{}, err
		}
		return formRequest(http.MethodPut, path, form), nil
	case chatwork.TaskContactRequestsList:
		return noBodyRequest(http.MethodGet, "/incoming_requests", nil), nil
	case chatwork.TaskContactRequestsAccept, chatwork.TaskContactRequestsReject:
		requestID, err := decimal(input.Request, chatwork.ReferenceRequest)
		if err != nil {
			return requestSpec{}, err
		}
		method := http.MethodPut
		if input.Task == chatwork.TaskContactRequestsReject {
			method = http.MethodDelete
		}
		return noBodyRequest(method, "/incoming_requests/"+requestID, nil), nil
	default:
		return requestSpec{}, invalidRequest("Chatwork タスクが公式の処理にマッピングされていません")
	}
}

func inviteForm(input chatwork.Request) url.Values {
	values := url.Values{}
	setOptional(values, "code", input.InviteCode)
	if input.DescriptionSet {
		values.Set("description", input.Description)
	}
	if input.InviteApprovalSet {
		values.Set("need_acceptance", bool01(input.InviteNeedsApproval))
	}
	return values
}

func inviteUpdateForm(input chatwork.Request) (url.Values, error) {
	if !input.InviteApprovalSet || !input.DescriptionSet || input.Description == "" {
		return nil, invalidRequest("招待リンク更新には明示した承認要件と空でない説明が必要です")
	}
	if (input.InviteCode == "") == !input.InviteRegenerateCode {
		return nil, invalidRequest("招待リンク更新には明示コードまたはコード再生成のどちらか一方が必要です")
	}
	values := url.Values{
		"description":     {input.Description},
		"need_acceptance": {bool01(input.InviteNeedsApproval)},
	}
	if !input.InviteRegenerateCode {
		values.Set("code", input.InviteCode)
	}
	return values, nil
}

func joinRefs(refs []chatwork.Reference) string {
	values := make([]string, 0, len(refs))
	for _, ref := range refs {
		values = append(values, ref.Value)
	}
	return strings.Join(values, ",")
}

func setOptional(values url.Values, name, value string) {
	if value != "" {
		values.Set(name, value)
	}
}

func bool01(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func mutating(task chatwork.Task) bool {
	switch task {
	case chatwork.TaskRoomsCreate, chatwork.TaskRoomsUpdate, chatwork.TaskRoomsLeave,
		chatwork.TaskRoomsDelete, chatwork.TaskMembersReplace, chatwork.TaskMessagesSend,
		chatwork.TaskMessagesMarkRead, chatwork.TaskMessagesMarkUnread,
		chatwork.TaskMessagesUpdate, chatwork.TaskMessagesDelete,
		chatwork.TaskRoomTasksCreate, chatwork.TaskRoomTasksSetStatus,
		chatwork.TaskFilesUpload, chatwork.TaskInviteLinkCreate,
		chatwork.TaskInviteLinkUpdate, chatwork.TaskInviteLinkDelete,
		chatwork.TaskContactRequestsAccept, chatwork.TaskContactRequestsReject:
		return true
	default:
		return false
	}
}
