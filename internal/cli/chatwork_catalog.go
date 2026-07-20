package cli

import (
	"github.com/tasuku43/cwk/internal/domain/authn"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

type chatworkCommandDefinition struct {
	Task         chatwork.Task
	Confirmation string
	Reconcile    string
}

const (
	confirmAccessChange = "access-change"
	confirmDestructive  = "destructive"
)

var chatworkAuthentication = &authn.Requirement{
	Methods:              []authn.Method{authn.MethodPAT},
	Authority:            chatwork.AuthenticationAuthority,
	Audience:             chatwork.AuthenticationAudience,
	RequiredCapabilities: []string{chatwork.AuthenticationCapability},
}

func chatworkCommandSpecs() []CommandSpec {
	room := string(chatwork.ReferenceRoom)
	account := string(chatwork.ReferenceAccount)
	message := string(chatwork.ReferenceMessage)
	task := string(chatwork.ReferenceTask)
	file := string(chatwork.ReferenceFile)
	invite := string(chatwork.ReferenceInvite)
	request := string(chatwork.ReferenceRequest)

	return []CommandSpec{
		chatworkRead("account show", "認証済みのChatworkアカウントを表示する", "", RoleDiscover,
			"chatwork.account.inspect", "設定済みのChatworkトークンに結び付いたアカウントを正確に確認する",
			nil, fields(refField("account_ref", account, "ルーム作成とアカウント絞り込みで使用できる正規のアカウント参照。"), textField("name", "アカウントの表示名。"), textField("organization", "存在する場合の、空でない人間可読な組織名と部署情報。")), chatwork.TaskAccountShow),
		chatworkRead("account status", "未読、メンション、タスクの件数を表示する", "", RoleUtility,
			"chatwork.account.inspect", "認証済みアカウントのChatwork集計ステータスを確認する",
			nil, fields(integerField("unread", "未読メッセージの合計件数。"), integerField("mentions", "未読メンションの合計件数。"), integerField("tasks", "未完了タスクの合計件数。")), chatwork.TaskAccountStatus),
		chatworkRead("personal-tasks list", "認証済みアカウントに割り当てられたタスクを一覧表示する", "[--assigned-by <account-ref>] [--status open|done]", RoleDiscover,
			"chatwork.personal-tasks.inspect", "上限付きの個人タスク一覧を、正規のタスク、ルーム、アカウント、メッセージ参照を含む固定位置スキーマで取得する",
			[]CommandInput{refFlag("--assigned-by", false, account, "割り当て元を一つの完全一致アカウント参照で絞り込みます。"), enumFlag("--status", false, "タスクのステータスで絞り込みます。", "open", "done")},
			fields(refField("task_ref", task, "1番目にある正規のタスク参照。"), refField("room_ref", room, "2番目にある正規のルーム参照。"), refField("assigned_by_ref", account, "3番目にある正規の割り当て元アカウント参照。"), refField("message_ref", message, "4番目にある正規のタスクメッセージ参照。"), textField("body", "5番目にある、ターミナルで安全な引用済みタスク本文。"), textField("status", "6番目にあるタスクのステータス。"), limitField(), completeField()), chatwork.TaskPersonalTasksList),
		chatworkRead("contacts list", "Chatworkコンタクトを検索する", "", RoleDiscover,
			"chatwork.contacts.discover", "コンタクトを、完全一致のアカウント参照とダイレクトチャット参照を含む固定位置スキーマで一覧表示する",
			nil, fields(refField("account_ref", account, "1番目にある正規のコンタクトアカウント参照。"), refField("room_ref", room, "2番目にある正規のダイレクトチャット参照。"), textField("name", "3番目にある、ターミナルで安全な引用済みコンタクト表示名。"), textField("organization", "空でない組織名または部署情報がある場合に末尾へ付く任意項目。"), completeField()), chatwork.TaskContactsList),
		chatworkRead("rooms list", "参加中のChatworkルームを検索する", "", RoleDiscover,
			"chatwork.rooms.manage", "参加中のルームを、完全一致のルーム参照とタスクに必要なステータスを含む固定位置スキーマで一覧表示する",
			nil, roomFields(room, true), chatwork.TaskRoomsList),
		chatworkMutation("rooms create", "メンバーを正確に指定してグループチャットを作成する", "--account <account-ref> --name <text> --admin <account-ref> [--member <account-ref>] [--readonly <account-ref>] [--description <text>] [--icon meeting|group|check|document|event|project|business|study|security|star|idea|heart|magcup|beer|music|sports|travel] [--invite-code <code>] [--invite-approval required|not-required] --confirm=access-change", RoleAct,
			"chatwork.rooms.manage", "検証済みの認証アカウント範囲で、メンバー構成とアクセスへの影響を明示してグループチャットを一つ作成する",
			[]CommandInput{refFlag("--account", true, account, "ルーム作成を、実行前に検証する認証アカウントの完全一致参照に結び付けます。"), textFlag("--name", true, "1〜255文字のルーム名。"), repeatedRefFlag("--admin", true, account, "管理者を一人追加します。複数追加する場合は繰り返し指定します。"), repeatedRefFlag("--member", false, account, "メンバーを一人追加します。複数追加する場合は繰り返し指定します。"), repeatedRefFlag("--readonly", false, account, "閲覧のみのメンバーを一人追加します。複数追加する場合は繰り返し指定します。"), textFlag("--description", false, "ルームの説明。"), enumFlag("--icon", false, "Chatwork公式のルームアイコンプリセット。", chatwork.RoomIconPresetValues()...), textFlag("--invite-code", false, "1〜50文字の英数字、アンダースコア、ハイフンからなる任意コードの招待リンクを同時に作成します。"), enumFlag("--invite-approval", false, "この承認要件を持つ招待リンクを作成します。", "required", "not-required"), confirmFlag(confirmAccessChange)},
			fields(refField("room_ref", room, "作成したルームの正規参照。")), chatwork.TaskRoomsCreate, confirmAccessChange, "rooms list",
			createMutation("chatwork-room", "--account", operation.CardinalityMany, yes, yes, no)),
		chatworkRead("rooms show", "完全一致するルームを一つ表示する", "--room <room-ref>", RoleAct,
			"chatwork.rooms.manage", "表示名で再検索せず、完全一致するルームを一つ確認する",
			[]CommandInput{refFlag("--room", true, room, "ルーム検索で得た room_ref を変更せずに渡します。")}, roomFields(room, false), chatwork.TaskRoomsShow),
		chatworkMutation("rooms update", "完全一致するルームの説明情報を更新する", "--room <room-ref> [--name <text>] [--description <text>] [--icon <preset>]", RoleAct,
			"chatwork.rooms.manage", "メンバー構成を変えずに、選択したルームの名前、説明、アイコンを更新する",
			[]CommandInput{refFlag("--room", true, room, "更新する完全一致のルーム。"), textFlag("--name", false, "変更後のルーム名。"), textFlag("--description", false, "変更後のルーム説明。"), textFlag("--icon", false, "変更後の定義済みアイコンプリセット。")},
			fields(refField("room_ref", room, "更新したルームの正規参照。")), chatwork.TaskRoomsUpdate, "", "rooms show",
			writeMutation("chatwork-room", "--room", "", operation.CardinalityOne, no, no, no)),
		chatworkMutation("rooms leave", "完全一致するグループチャットから退席する", "--room <room-ref> --confirm=destructive", RoleAct,
			"chatwork.rooms.manage", "破壊的影響とアクセスへの影響を明示して、選択したグループチャットから退席する",
			[]CommandInput{refFlag("--room", true, room, "退席する完全一致のルーム。"), confirmFlag(confirmDestructive)}, fields(refField("room_ref", room, "退席したルームの正規参照。")), chatwork.TaskRoomsLeave, confirmDestructive, "rooms show",
			writeMutation("chatwork-room", "--room", "", operation.CardinalityMany, no, yes, yes)),
		chatworkMutation("rooms delete", "完全一致するグループチャットを完全に削除する", "--room <room-ref> --confirm=destructive", RoleAct,
			"chatwork.rooms.manage", "選択したグループチャットとその中のデータを完全に削除する",
			[]CommandInput{refFlag("--room", true, room, "削除する完全一致のルーム。"), confirmFlag(confirmDestructive)}, fields(refField("room_ref", room, "削除したルームの正規参照。")), chatwork.TaskRoomsDelete, confirmDestructive, "rooms show",
			writeMutation("chatwork-room", "--room", "", operation.CardinalityUnbounded, no, yes, yes)),
		chatworkRead("members list", "完全一致するルームのメンバーを一覧表示する", "--room <room-ref>", RoleAct,
			"chatwork.members.manage", "完全一致する一つのルームについて、メンバーの識別情報と権限を固定位置スキーマで一覧表示する",
			[]CommandInput{refFlag("--room", true, room, "メンバー構成を確認する完全一致のルーム。")}, fields(refField("account_ref", account, "1番目にあるメンバーアカウントの正規参照。"), textField("name", "2番目にある、ターミナルで安全な引用済みメンバー表示名。"), textField("role", "3番目にあるメンバーの権限。"), completeField()), chatwork.TaskMembersList),
		chatworkMutation("members replace", "ルームのメンバー構成全体を置き換える", "--room <room-ref> --admin <account-ref> [--member <account-ref>] [--readonly <account-ref>] --confirm=access-change", RoleAct,
			"chatwork.members.manage", "アクセスへの影響を明示して、選択したルームの権限別メンバー構成全体を置き換える",
			[]CommandInput{refFlag("--room", true, room, "メンバー構成を置き換える完全一致のルーム。"), repeatedRefFlag("--admin", true, account, "管理者アカウント。複数指定する場合は繰り返します。"), repeatedRefFlag("--member", false, account, "メンバーアカウント。複数指定する場合は繰り返します。"), repeatedRefFlag("--readonly", false, account, "閲覧のみのアカウント。複数指定する場合は繰り返します。"), confirmFlag(confirmAccessChange)},
			fields(integerField("administrators", "置き換え後の管理者数。"), integerField("members", "置き換え後のメンバー数。"), integerField("readonly", "置き換え後の閲覧のみメンバー数。")), chatwork.TaskMembersReplace, confirmAccessChange, "members list",
			writeMutation(room, "--room", "", operation.CardinalityMany, yes, yes, yes)),
		chatworkRead("messages list", "選択可能な上限付きメッセージ範囲を取得する", "--room <room-ref> [--window recent|changes] [--since <RFC3339>] [--until <RFC3339>] [--on <day>] [--start-index <index>] [--count <count>] [--sender <account-ref>] [--context none|replies] [--resolve-relations <count>]", RoleAct,
			"chatwork.messages.manage", "閲覧制限・期間の到達不能・解析不能な関係を空や不存在と区別し、明示した追加取得予算内で返信元を自己完結させる上限付きメッセージ範囲を取得する",
			[]CommandInput{
				refFlag("--room", true, room, "メッセージを取得する完全一致のルーム。"),
				enumFlag("--window", false, "最新の上限付き範囲（recent、既定値）またはプロバイダーの差分範囲（changes）を選択します。", "recent", "changes"),
				textFlag("--since", false, "上限付き取得範囲内で、この時刻を含む秒精度・明示オフセット付き RFC3339 送信時刻以降を主要候補にします。--on とは同時に指定できません。"),
				textFlag("--until", false, "上限付き取得範囲内で、この時刻を含まない秒精度・明示オフセット付き RFC3339 送信時刻より前を主要候補にします。--on とは同時に指定できません。"),
				textFlag("--on", false, "Asia/Tokyo の一暦日を YYYY-MM-DD、today、yesterday のいずれかで選びます。相対日はコマンド開始時に一度だけ具体化し、--since/--until とは同時に指定できません。"),
				integerFlag("--start-index", false, "任意の送信者・期間照合後に、型付き送信時刻が新しい順で選択を始める1始まりの順位（1〜100）です。省略時は1です。"),
				integerFlag("--count", false, "--start-index から返す主要メッセージの最大件数（1〜100）です。終了順位ではありません。--start-index 11 --count 20 は順位11〜30を選びます。直接の返信コンテキストにより、表示件数はこの値を超えることがあります。"),
				repeatedRefFlag("--sender", false, account, "プロバイダーの上限付き範囲内で、完全一致の送信者アカウントに絞り込みます。列挙した送信者のいずれかに一致させる（OR）には繰り返し指定し、完全一致参照は最大100件です。"),
				enumFlag("--context", false, "送信者、期間、順位のいずれかの選択とともに使用し、関連レコードを含めない（none、既定値）か、上限付き範囲内にある明示的な返信元・返信先を1ホップだけ含める（replies）かを選択します。返信コンテキストは主要期間外を含む場合があります。", "none", "replies"),
				integerFlag("--resolve-relations", false, "表示対象と補足メッセージが参照する同一ルームの未解決返信元を、正規 message_ref による追加の一件取得で再帰的に補う最大件数（0〜100）です。既定値は5、0は追加取得を無効化します。元の100件内にある返信元と重複IDは枠を消費せず、循環は一度だけ扱います。"),
			}, messageFields(room, message, account, true), chatwork.TaskMessagesList),
		chatworkMutation("messages send", "完全一致するルームへメッセージを送信する", "--room <room-ref> --body <text> [--self-unread]", RoleAct,
			"chatwork.messages.manage", "選択したルームへ、指定したメッセージ本文を一件送信する",
			[]CommandInput{refFlag("--room", true, room, "送信先の完全一致ルーム。"), textFlag("--body", true, "メッセージ本文。意図して使用する場合は、確認済みのChatwork記法を含められます。"), boolFlag("--self-unread", "送信したメッセージを、認証済みアカウントで未読のままにします。")},
			fields(refField("message_ref", message, "作成したメッセージの正規参照。"), refField("room_ref", room, "親ルームの正規参照。")), chatwork.TaskMessagesSend, "", "messages list",
			createMutation("chatwork-message", "--room", operation.CardinalityOne, yes, no, no)),
		chatworkMutation("messages mark-read", "完全一致するメッセージまでを既読にする", "--room <room-ref> --message <message-ref>", RoleAct,
			"chatwork.messages.manage", "一つのルームで、選択した完全一致メッセージまでを既読にする",
			[]CommandInput{refFlag("--room", true, room, "完全一致する親ルーム。"), refFlag("--message", true, message, "既読にする範囲の境界となる完全一致メッセージ。")}, unreadFields(), chatwork.TaskMessagesMarkRead, "", "messages show",
			writeMutation(message, "--message", "--room", operation.CardinalityMany, no, no, no)),
		chatworkMutation("messages mark-unread", "完全一致するメッセージからを未読にする", "--room <room-ref> --message <message-ref>", RoleAct,
			"chatwork.messages.manage", "一つのルームで、選択した完全一致メッセージからを未読にする",
			[]CommandInput{refFlag("--room", true, room, "完全一致する親ルーム。"), refFlag("--message", true, message, "未読にする範囲の境界となる完全一致メッセージ。")}, unreadFields(), chatwork.TaskMessagesMarkUnread, "", "messages show",
			writeMutation(message, "--message", "--room", operation.CardinalityMany, no, no, no)),
		chatworkRead("messages show", "完全一致するメッセージを一件表示する", "--room <room-ref> --message <message-ref>", RoleAct,
			"chatwork.messages.manage", "完全一致するメッセージを安全な本文と型付き関係で確認し、閲覧制限を不存在と区別する",
			[]CommandInput{refFlag("--room", true, room, "完全一致する親ルーム。"), refFlag("--message", true, message, "確認する完全一致メッセージ。")}, messageFields(room, message, account, false), chatwork.TaskMessagesShow),
		chatworkMutation("messages update", "完全一致するメッセージを更新する", "--room <room-ref> --message <message-ref> --body <text>", RoleAct,
			"chatwork.messages.manage", "完全一致するメッセージ一件の本文を置き換える",
			[]CommandInput{refFlag("--room", true, room, "完全一致する親ルーム。"), refFlag("--message", true, message, "更新する完全一致メッセージ。"), textFlag("--body", true, "変更後のメッセージ本文。")}, fields(refField("message_ref", message, "更新したメッセージの正規参照。")), chatwork.TaskMessagesUpdate, "", "messages show",
			writeMutation("chatwork-message", "--message", "--room", operation.CardinalityOne, no, no, no)),
		chatworkMutation("messages delete", "完全一致するメッセージを削除する", "--room <room-ref> --message <message-ref> --confirm=destructive", RoleAct,
			"chatwork.messages.manage", "破壊的影響を明示して、選択した完全一致メッセージを削除する",
			[]CommandInput{refFlag("--room", true, room, "完全一致する親ルーム。"), refFlag("--message", true, message, "削除する完全一致メッセージ。"), confirmFlag(confirmDestructive)}, fields(refField("message_ref", message, "削除したメッセージの正規参照。")), chatwork.TaskMessagesDelete, confirmDestructive, "messages show",
			writeMutation("chatwork-message", "--message", "--room", operation.CardinalityOne, no, no, yes)),
		chatworkRead("room-tasks list", "完全一致するルームのタスクを一覧表示する", "--room <room-ref> [--account <account-ref>] [--assigned-by <account-ref>] [--status open|done]", RoleAct,
			"chatwork.room-tasks.manage", "上限付きのルームタスク一覧を、完全一致のタスク参照とアカウント参照を含む固定位置スキーマで取得する",
			[]CommandInput{refFlag("--room", true, room, "タスクを取得する完全一致のルーム。"), refFlag("--account", false, account, "完全一致する担当者アカウントで絞り込みます。"), refFlag("--assigned-by", false, account, "完全一致する割り当て元アカウントで絞り込みます。"), enumFlag("--status", false, "タスクのステータスで絞り込みます。", "open", "done")}, taskFields(room, task, account, message, true), chatwork.TaskRoomTasksList),
		chatworkMutation("room-tasks create", "ルーム内の完全一致する担当者にタスクを作成する", "--room <room-ref> --body <text> --assignee <account-ref> [--limit <unix-time>] [--limit-type date|time]", RoleAct,
			"chatwork.room-tasks.manage", "一つのルームで、指定したタスク本文を完全一致する担当者アカウントに割り当てる",
			[]CommandInput{refFlag("--room", true, room, "完全一致する親ルーム。"), textFlag("--body", true, "タスク本文。"), repeatedRefFlag("--assignee", true, account, "完全一致する担当者。複数追加する場合は繰り返し指定します。"), integerFlag("--limit", false, "任意のUnix期限。"), enumFlag("--limit-type", false, "期限を日付または時刻として解釈します。", "date", "time")}, fields(refField("task_ref", task, "作成したタスクの正規参照。"), refField("room_ref", room, "親ルームの正規参照。")), chatwork.TaskRoomTasksCreate, "", "room-tasks list",
			createMutation("chatwork-task", "--room", operation.CardinalityMany, yes, no, no)),
		chatworkRead("room-tasks show", "完全一致するルームタスクを一件表示する", "--room <room-ref> --task <task-ref>", RoleAct,
			"chatwork.room-tasks.manage", "再検索せず、完全一致するタスクを一件確認する",
			[]CommandInput{refFlag("--room", true, room, "完全一致する親ルーム。"), refFlag("--task", true, task, "確認する完全一致タスク。")}, taskFields(room, task, account, message, false), chatwork.TaskRoomTasksShow),
		chatworkMutation("room-tasks set-status", "完全一致するタスクの完了状態を設定する", "--room <room-ref> --task <task-ref> --status open|done", RoleAct,
			"chatwork.room-tasks.manage", "選択した完全一致タスクを open または done に設定する",
			[]CommandInput{refFlag("--room", true, room, "完全一致する親ルーム。"), refFlag("--task", true, task, "変更する完全一致タスク。"), enumFlag("--status", true, "変更後のタスクステータス。", "open", "done")}, fields(refField("task_ref", task, "更新したタスクの正規参照。")), chatwork.TaskRoomTasksSetStatus, "", "room-tasks show",
			writeMutation("chatwork-task", "--task", "--room", operation.CardinalityOne, yes, no, no)),
		chatworkRead("files list", "完全一致するルームのファイルを一覧表示する", "--room <room-ref> [--account <account-ref>]", RoleAct,
			"chatwork.files.manage", "上限付きのルームファイル一覧を固定位置スキーマで取得する。1番目と2番目の値は変更せず files show に渡す",
			[]CommandInput{refFlag("--room", true, room, "ファイルを取得する完全一致のルーム。"), refFlag("--account", false, account, "完全一致するアップロード者アカウントで絞り込みます。")}, fileFields(room, file, account, message, true), chatwork.TaskFilesList),
		chatworkMutation("files upload", "上限付きファイルを完全一致するルームへアップロードする", "--room <room-ref> --path <file> [--message <text>]", RoleAct,
			"chatwork.files.manage", "選択したルームへ、5 MiB以下のローカルファイルを一つアップロードする",
			[]CommandInput{refFlag("--room", true, room, "アップロード先の完全一致ルーム。"), textFlag("--path", true, "ローカルファイルのパス。アップロード前に上限と妥当性を検証します。"), textFlag("--message", false, "ファイルに添付する任意のメッセージ。")}, fields(refField("file_ref", file, "アップロードしたファイルの正規参照。"), refField("room_ref", room, "親ルームの正規参照。")), chatwork.TaskFilesUpload, "", "files list",
			createMutation("chatwork-file", "--room", operation.CardinalityOne, yes, no, no)),
		chatworkRead("files show", "完全一致するルームファイルを一件表示する", "--room <room-ref> --file <file-ref> [--create-download-url]", RoleAct,
			"chatwork.files.manage", "完全一致するファイルを一件確認し、任意で上限付きのプロバイダーダウンロードURLを要求する",
			[]CommandInput{refFlag("--room", true, room, "完全一致する親ルーム。"), refFlag("--file", true, file, "確認する完全一致ファイル。"), boolFlag("--create-download-url", "結果にプロバイダーのダウンロードURLを要求します。")}, fileFields(room, file, account, message, false), chatwork.TaskFilesShow),
		chatworkRead("invite-link show", "ルームの招待リンク状態を表示する", "--room <room-ref>", RoleAct,
			"chatwork.invite-links.manage", "完全一致するルーム一つの招待リンク状態を確認する",
			[]CommandInput{refFlag("--room", true, room, "招待リンク状態を確認する完全一致のルーム。")}, inviteFields(invite), chatwork.TaskInviteLinkShow),
		chatworkMutation("invite-link create", "ルームの招待リンクを作成する", "--room <room-ref> [--code <code>] [--approval required|not-required] [--description <text>] --confirm=access-change", RoleAct,
			"chatwork.invite-links.manage", "アクセスへの影響を明示して、ルームの招待リンクを作成する",
			[]CommandInput{refFlag("--room", true, room, "完全一致する親ルーム。"), textFlag("--code", false, "1〜50文字の英数字、アンダースコア、ハイフンからなる任意の招待リンクコード。"), enumFlag("--approval", false, "参加に管理者の承認が必要かどうか。", "required", "not-required"), textFlag("--description", false, "招待リンクの説明。"), confirmFlag(confirmAccessChange)}, inviteFields(invite), chatwork.TaskInviteLinkCreate, confirmAccessChange, "invite-link show",
			createMutation("chatwork-invite-link", "--room", operation.CardinalityOne, no, yes, no)),
		chatworkMutation("invite-link update", "完全一致する招待リンクを全項目置換する", "--invite <invite-ref> [--code <code>] [--regenerate-code] --approval required|not-required --description <text> --confirm=access-change", RoleAct,
			"chatwork.invite-links.manage", "コードの明示値または再生成、承認要件、非空の説明をすべて指定し、選択した招待リンクを置き換える",
			[]CommandInput{refFlag("--invite", true, invite, "invite-link show または create が出力した完全一致の招待リンク参照。"), textFlag("--code", false, "変更後の1〜50文字の英数字、アンダースコア、ハイフンからなる招待リンクコード。--regenerate-codeとは同時に指定できません。"), boolFlag("--regenerate-code", "コードをランダム再生成する意図を明示します。--codeとは同時に指定できません。"), enumFlag("--approval", true, "変更後の承認要件。", "required", "not-required"), textFlag("--description", true, "変更後の空でない招待リンク説明。"), confirmFlag(confirmAccessChange)}, inviteFields(invite), chatwork.TaskInviteLinkUpdate, confirmAccessChange, "invite-link show",
			writeMutation("chatwork-invite-link", "--invite", "", operation.CardinalityOne, no, yes, no)),
		chatworkMutation("invite-link delete", "完全一致する招待リンクを無効にする", "--invite <invite-ref> --confirm=destructive", RoleAct,
			"chatwork.invite-links.manage", "アクセスへの破壊的影響を明示して、選択した招待リンクを無効にする",
			[]CommandInput{refFlag("--invite", true, invite, "完全一致する招待リンク参照。"), confirmFlag(confirmDestructive)}, inviteFields(invite), chatwork.TaskInviteLinkDelete, confirmDestructive, "invite-link show",
			writeMutation("chatwork-invite-link", "--invite", "", operation.CardinalityOne, no, yes, yes)),
		chatworkRead("contact-requests list", "受信したコンタクト承認依頼を検索する", "", RoleDiscover,
			"chatwork.contact-requests.manage", "受信したコンタクト承認依頼を、完全一致の承認依頼参照とアカウント参照を含む固定位置スキーマで一覧表示する",
			nil, fields(refField("request_ref", request, "1番目にある受信済み承認依頼の正規参照。"), refField("account_ref", account, "2番目にある申請元アカウントの正規参照。"), textField("name", "3番目にある、ターミナルで安全な引用済み申請元アカウント名。"), textField("message", "末尾に任意で付く、ターミナルで安全な引用済み申請メッセージ。"), limitField(), completeField()), chatwork.TaskContactRequestsList),
		chatworkMutation("contact-requests accept", "完全一致するコンタクト承認依頼を承認する", "--request <request-ref> --confirm=access-change", RoleAct,
			"chatwork.contact-requests.manage", "選択した受信済みコンタクト承認依頼を承認する",
			[]CommandInput{refFlag("--request", true, request, "受信済み承認依頼の完全一致参照。"), confirmFlag(confirmAccessChange)}, fields(refField("account_ref", account, "承認したコンタクトアカウントの正規参照。"), refField("room_ref", room, "ダイレクトチャットの正規参照。")), chatwork.TaskContactRequestsAccept, confirmAccessChange, "contact-requests list",
			writeMutation("chatwork-contact-request", "--request", "", operation.CardinalityOne, yes, yes, no)),
		chatworkMutation("contact-requests reject", "完全一致するコンタクト承認依頼を拒否する", "--request <request-ref> --confirm=destructive", RoleAct,
			"chatwork.contact-requests.manage", "選択した受信済みコンタクト承認依頼を拒否する",
			[]CommandInput{refFlag("--request", true, request, "受信済み承認依頼の完全一致参照。"), confirmFlag(confirmDestructive)}, fields(refField("request_ref", request, "拒否したコンタクト承認依頼の正規参照。")), chatwork.TaskContactRequestsReject, confirmDestructive, "contact-requests list",
			writeMutation("chatwork-contact-request", "--request", "", operation.CardinalityOne, no, yes, yes)),
	}
}

const (
	no  = operation.DeclarationNo
	yes = operation.DeclarationYes
)

func chatworkRead(path, summary, args string, role CommandRole, capability, outcome string, inputs []CommandInput, output []OutputField, task chatwork.Task) CommandSpec {
	if inputs == nil {
		inputs = []CommandInput{}
	}
	return chatworkBase(path, summary, args, operation.EffectRead, role, capability, outcome, inputs, output, task, "", "", nil)
}

func chatworkMutation(path, summary, args string, role CommandRole, capability, outcome string, inputs []CommandInput, output []OutputField, task chatwork.Task, confirmation, reconcile string, mutation MutationContract) CommandSpec {
	effect := operation.EffectWrite
	if mutation.TargetIDInput == "" {
		effect = operation.EffectCreate
	}
	return chatworkBase(path, summary, args, effect, role, capability, outcome, inputs, output, task, confirmation, reconcile, &mutation)
}

func chatworkBase(path, summary, args string, effect operation.Effect, role CommandRole, capability, outcome string, inputs []CommandInput, output []OutputField, task chatwork.Task, confirmation, reconcile string, mutation *MutationContract) CommandSpec {
	return CommandSpec{
		Path: path, Summary: summary, Args: args, Effect: effect, Role: role, Configurable: true,
		Agent: AgentContract{
			CapabilityID: capability, Outcome: outcome, Inputs: inputs,
			Output:         CommandOutput{Formats: []OutputFormat{OutputFormatText}, DefaultFormat: OutputFormatText, Fields: output, Completeness: OutputCompletenessComplete},
			Prerequisites:  []string{"CWK_API_TOKEN はコマンドのプロセスにだけ設定し、argvやプロジェクトファイルにはトークンを渡さないでください。"},
			Authentication: chatworkAuthentication,
			Errors:         chatworkCommandErrors(path, task, reconcile, mutation != nil), Mutation: mutation,
		},
		handler:  runChatwork,
		chatwork: &chatworkCommandDefinition{Task: task, Confirmation: confirmation, Reconcile: reconcile},
	}
}

func chatworkCommandErrors(path string, task chatwork.Task, reconcile string, mutation bool) []CommandError {
	help := "help " + path
	retry := path
	rateLimit := declaredCommandError(
		fault.KindRateLimited,
		"chatwork_rate_limited",
		true,
		path,
		"待機時間が表示された場合だけその時間を待ち、不明な場合は解除時刻を推測せずに同じ読み取りの再試行時期を判断してください。",
	)
	if mutation {
		// Mutation recovery never suggests replaying a write. Even failures that
		// are retryable at the fault level route through scoped help; uncertain
		// outcomes use the exact read-only reconciliation task below.
		retry = help
		rateLimit = declaredCommandError(
			fault.KindRateLimited,
			"chatwork_mutation_rate_limited",
			false,
			help,
			"待機情報は変更の再実行許可ではありません。自動再試行せず、このコマンドの変更契約を確認してください。",
		)
	}
	errors := []CommandError{
		declaredCommandError(fault.KindInvalidInput, "invalid_arguments", false, help, "宣言されたコマンド入力を修正してください。"),
		declaredCommandError(fault.KindInvalidInput, "invalid_chatwork_task", false, help, "型付きChatworkタスクの入力を修正してください。"),
		declaredCommandError(fault.KindInvalidInput, "invalid_chatwork_request", false, help, "型付きChatworkアダプター要求を修正してください。"),
		declaredCommandError(fault.KindContract, "missing_context", false, help, "コンテキスト対応のコマンド呼び出しを修正してください。"),
		declaredCommandError(fault.KindContract, "missing_chatwork_port", false, help, "Chatworkアダプターの構成を修正してください。"),
		declaredCommandError(fault.KindContract, "chatwork_result_mismatch", false, help, "型付きChatworkアダプター結果の契約を修正してください。"),
		declaredCommandError(fault.KindContract, "chatwork_result_invalid", false, help, "タスク固有の型付きChatwork結果契約を修正してください。"),
		declaredCommandError(fault.KindAuthentication, "invalid_authentication_binding", false, help, "設定済みのChatwork認証を確立し直してください。"),
		declaredCommandError(fault.KindInternal, "unclassified_chatwork_error", false, help, "Chatworkアダプターのエラー分類を確認してください。"),
		declaredCommandError(fault.KindInvalidInput, "chatwork_invalid_request", false, help, "Chatworkが受け付けるタスク入力に修正してください。"),
		declaredCommandError(fault.KindAuthentication, "chatwork_token_missing", false, help, "このコマンドのプロセスに CWK_API_TOKEN を設定してください。"),
		declaredCommandError(fault.KindAuthentication, "chatwork_token_invalid", false, help, "CWK_API_TOKEN を有効なプロセスローカルのトークンに置き換えてください。"),
		declaredCommandError(fault.KindAuthentication, "chatwork_authentication_failed", false, help, "設定済みのChatworkトークンを置き換えてください。"),
		declaredCommandError(fault.KindPermission, "chatwork_permission_denied", false, help, "このタスクを実行できるアカウントを使用してください。"),
		declaredCommandError(fault.KindNotFound, "chatwork_not_found", false, help, "現在有効な正規参照を再検索してください。"),
		rateLimit,
		declaredCommandError(fault.KindContract, "chatwork_response_too_large", false, help, "タスクの範囲を狭めるか、固定された応答上限を確認してください。"),
		declaredCommandError(fault.KindContract, "chatwork_response_invalid", false, help, "再試行する前に、プロバイダースキーマの変更を確認してください。"),
		declaredCommandError(fault.KindContract, "chatwork_response_malformed", false, help, "再試行する前に、プロバイダースキーマの変更を確認してください。"),
		declaredCommandError(fault.KindContract, "chatwork_response_unmapped", false, help, "型付き応答のマッピングを修正してください。"),
		declaredCommandError(fault.KindUnavailable, "chatwork_response_unavailable", true, retry, "上限付き応答の失敗を確認してから再試行してください。"),
		declaredCommandError(fault.KindContract, "chatwork_request_contract_invalid", false, help, "型付き要求のマッピングを修正してください。"),
		declaredCommandError(fault.KindContract, "chatwork_transport_missing", false, help, "Chatworkトランスポートの構成を修正してください。"),
		declaredCommandError(fault.KindContract, "chatwork_unexpected_response", false, help, "再試行する前に、文書化されていないプロバイダー動作を確認してください。"),
		declaredCommandError(fault.KindContract, "output_contract_exceeded", false, help, "結果の範囲を狭めるか、固定された出力上限を確認してください。"),
		declaredCommandError(fault.KindContract, "output_encoding_failed", false, help, "タスク結果の変換を修正してください。"),
		declaredCommandError(fault.KindInternal, "output_write_failed", true, retry, "書き込み可能な出力ストリームで再試行してください。"),
		declaredCommandError(fault.KindCanceled, "operation_canceled", true, retry, "呼び出し元の準備ができたら再試行してください。"),
	}
	if !mutation {
		errors = append(errors, declaredCommandError(fault.KindUnavailable, "chatwork_unavailable", true, path, "Chatworkが利用可能になってから再試行してください。"))
	}
	if task == chatwork.TaskMessagesList || task == chatwork.TaskMessagesShow {
		errors = append(errors, declaredCommandError(fault.KindContract, "chatwork_message_limitation_invalid", false, help, "閲覧制限を推測せず、Chatworkの応答契約変更を確認してください。"))
	}
	if task == chatwork.TaskMessagesShow {
		errors = append(errors, declaredCommandError(fault.KindPermission, "chatwork_message_restricted", false, help, "閲覧できるアカウントまたはプランで対象を確認してください。"))
	}
	if task == chatwork.TaskRoomsCreate {
		errors = append(errors, declaredCommandError(fault.KindUnavailable, "chatwork_account_verification_failed", false, help, "変更を再実行せず、認証アカウントを確認できる状態かをこのコマンドの契約で確認してください。"))
	}
	if task == chatwork.TaskFilesUpload {
		errors = append(errors,
			declaredCommandError(fault.KindInvalidInput, "chatwork_file_name_invalid", false, help, "有効なベース名を持つファイルを選択してください。"),
			declaredCommandError(fault.KindInvalidInput, "chatwork_file_unreadable", false, help, "読み取り可能なローカルファイルを選択してください。"),
			declaredCommandError(fault.KindInvalidInput, "chatwork_file_too_large", false, help, "5 MiB以下のファイルを選択してください。"),
			declaredCommandError(fault.KindContract, "chatwork_upload_contract_invalid", false, help, "上限付きmultipart要求のマッピングを修正してください。"),
		)
	}
	for _, required := range []struct {
		kind fault.Kind
		code string
	}{
		{fault.KindContract, "missing_authentication_context"}, {fault.KindContract, "missing_authenticated_action"},
		{fault.KindContract, "invalid_authentication_requirement"}, {fault.KindAuthentication, "missing_authenticator"},
		{fault.KindContract, "missing_authentication_clock"}, {fault.KindAuthentication, "invalid_authentication_session"},
		{fault.KindContract, "authentication_evaluation_failed"}, {fault.KindPermission, "insufficient_authentication_capability"},
		{fault.KindAuthentication, "authentication_expired"}, {fault.KindAuthentication, "authentication_context_mismatch"},
		{fault.KindAuthentication, "authentication_failed"}, {fault.KindCanceled, "authentication_canceled"},
		{fault.KindInternal, "unclassified_authenticated_action_error"},
	} {
		errors = append(errors, declaredCommandError(required.kind, required.code, false, help, "宣言されたChatwork認証コンテキストを修正するか、確立し直してください。"))
	}
	if mutation {
		errors = append(errors,
			declaredCommandError(fault.KindContract, "invalid_mutation_contract", false, help, "変更対象と影響の宣言を修正してください。"),
			declaredCommandError(fault.KindContract, "missing_mutation_action", false, help, "変更処理の構成を修正してください。"),
			declaredCommandError(fault.KindRejected, "missing_mutation_policy", false, help, "宣言されたChatwork変更ポリシーを設定してください。"),
			declaredCommandError(fault.KindRejected, "mutation_rejected", false, help, "対象を変更せず、必要な明示的確認を指定してください。"),
			declaredCommandError(fault.KindContract, "unclassified_mutation_outcome", false, reconcile, "再度変更する前に、この読み取り専用タスクで状態を照合してください。"),
			declaredCommandError(fault.KindContract, "chatwork_mutation_outcome_unknown", false, reconcile, "再度変更する前に、この読み取り専用タスクで状態を照合してください。"),
		)
	}
	return errors
}

func createMutation(kind, parent string, cardinality operation.Cardinality, notification, access, destructive operation.Declaration) MutationContract {
	return MutationContract{TargetKind: kind, TargetInputs: []string{parent}, ParentInput: parent, Impact: operation.Impact{Cardinality: cardinality, Notification: notification, AccessChange: access, Destructive: destructive}}
}

func writeMutation(kind, target, parent string, cardinality operation.Cardinality, notification, access, destructive operation.Declaration) MutationContract {
	targets := []string{target}
	if parent != "" {
		targets = append(targets, parent)
	}
	return MutationContract{TargetKind: kind, TargetInputs: targets, ParentInput: parent, TargetIDInput: target, Impact: operation.Impact{Cardinality: cardinality, Notification: notification, AccessChange: access, Destructive: destructive}}
}

func refFlag(name string, required bool, kind, description string) CommandInput {
	return CommandInput{Name: name, Source: InputSourceFlag, Required: required, Description: description, AllowedValues: []string{}, ReferenceKind: kind}
}
func repeatedRefFlag(name string, required bool, kind, description string) CommandInput {
	input := refFlag(name, required, kind, description)
	input.Repeatable = true
	return input
}
func textFlag(name string, required bool, description string) CommandInput {
	return CommandInput{Name: name, Source: InputSourceFlag, Required: required, Description: description, AllowedValues: []string{}}
}
func integerFlag(name string, required bool, description string) CommandInput {
	return textFlag(name, required, description)
}
func boolFlag(name, description string) CommandInput {
	return CommandInput{Name: name, Source: InputSourceFlag, Required: false, Description: description, AllowedValues: []string{}}
}
func enumFlag(name string, required bool, description string, values ...string) CommandInput {
	return CommandInput{Name: name, Source: InputSourceFlag, Required: required, Description: description, AllowedValues: values}
}
func confirmFlag(value string) CommandInput {
	return enumFlag("--confirm", true, "宣言された高影響の変更区分を明示的に確認します。", value)
}

func fields(values ...OutputField) []OutputField { return values }
func refField(name, kind, description string) OutputField {
	return OutputField{Name: name, Type: OutputFieldTypeString, Description: description, ReferenceKind: kind}
}
func textField(name, description string) OutputField {
	return OutputField{Name: name, Type: OutputFieldTypeString, Description: description}
}
func integerField(name, description string) OutputField {
	return OutputField{Name: name, Type: OutputFieldTypeInteger, Description: description}
}
func booleanField(name, description string) OutputField {
	return OutputField{Name: name, Type: OutputFieldTypeBoolean, Description: description}
}
func limitField() OutputField {
	return integerField("limit", "この結果範囲に含まれるプロバイダー項目の上限。")
}
func completeField() OutputField {
	return booleanField("complete", "この出力が、操作について文書化された完全な一覧かどうか。")
}
func roomFields(room string, collection bool) []OutputField {
	roomDescription := "ルーム操作に変更せず渡せる正規のルーム参照。"
	nameDescription := "信頼されていない外部テキストとしてのルーム名。"
	if collection {
		roomDescription = "1番目にある正規のルーム参照。変更せずルーム操作に渡します。"
		nameDescription = "2番目にある、ターミナルで安全な引用済みルーム名。"
	}
	result := fields(refField("room_ref", room, roomDescription), textField("name", nameDescription), textField("type", "ルームの種類。"), textField("role", "認証済みアカウントの権限。"), integerField("unread", "未読メッセージ数。"), integerField("mentions", "未読メンション数。"), integerField("tasks", "未完了タスク数。"))
	if collection {
		result = append(result, completeField())
	}
	return result
}
func messageFields(room, message, account string, collection bool) []OutputField {
	messageDescription := "正規のメッセージ参照。"
	bodyDescription := "構造を安全に区切った、信頼されていないテキストとしてのメッセージ本文。"
	sendDescription := "Unix送信時刻。"
	if collection {
		messageDescription = "位置レコードの2番目にある正規のメッセージ参照。変更せずメッセージ操作に渡します。"
		sendDescription = "位置レコードの4番目にあるUnix送信時刻。"
		bodyDescription = "位置レコードの末尾にある、ターミナルで安全な引用済みメッセージ本文。"
	}
	result := fields(refField("message_ref", message, messageDescription), refField("room_ref", room, "親ルームの正規参照。"), refField("sender_ref", account, "送信者アカウントの正規参照。"), textField("sender_name", "構造を安全に区切った、信頼されていないテキストとしての送信者表示名。"), textField("body", bodyDescription), integerField("send_time", sendDescription), textField("relation_state", "不完全な公式記法により関係集合を証明できない場合だけ unknown。省略時は reviewed relation set が完全です。"), OutputField{Name: "relations", Type: OutputFieldTypeArray, Description: "relation_state が unknown の場合は省略する、解決済みまたは未解決の型付きTo・返信・引用関係。"})
	if collection {
		result = append(result,
			integerField("sequence", "プロバイダーが返した元のメッセージ範囲内での1始まりの位置。選択後の出力では連番にならない場合があります。"),
			textField("actor_alias", "文書内だけで決定論的に使う送信者別名。コマンドの識別子にはなりません。"),
			textField("window", "要求した recent または差分メッセージ範囲。"),
			integerField("source_limit", "ローカルでメッセージを選択する前の、プロバイダー取得範囲の上限。"), completeField(),
			textField("access_limitation", "当該取得範囲の閲覧制限。none は制限ヘッダーなし、partial は一部制限、all は全件制限です。"),
			integerField("unresolved_relations", "正規の対象を解決できなかった型付き関係の数。"),
			integerField("unknown_relation_sets", "不完全な公式記法により関係集合が unknown となったメッセージ数。"),
			refField("oldest_reachable_message_ref", message, "recent・閲覧制限なし・非空の取得範囲で、messages list が到達できた最古の正規メッセージ参照。条件を証明できない場合は存在しません。"),
			integerField("oldest_reachable_send_time", "oldest_reachable_message_ref のUnix送信時刻。境界を証明できない場合は存在しません。"),
			textField("period_reachability", "期間指定が最古到達境界内、境界を一部超過、全体が境界外、または判定不能のいずれか。期間を指定しなかった場合は存在しません。"),
			integerField("relation_fetch_limit", "返信連鎖を補う追加の一件取得上限。既定値は5で、--resolve-relations 0 は無効化します。"),
			integerField("relation_fetch_attempts", "実際に行った追加の一件取得回数。元の取得範囲内での解決は含みません。"),
			OutputField{Name: "relation_resolution_targets", Type: OutputFieldTypeArray, Description: "表示対象から再帰的に到達した同一ルーム返信元ごとの正規 message_ref、source・fetched・not-found・restricted・budget-exhausted 状態、および解決できた補足メッセージ。"},
			integerField("source_count", "有効なローカル選択を適用する前にプロバイダーが返したメッセージ数。選択を要求しなかった場合は存在しません。"),
			integerField("candidate_count", "任意の送信者・期間照合後かつ start-index/count 適用前にある、上限付き取得範囲内の主要メッセージ候補数。選択を要求しなかった場合は存在しません。"),
			integerField("filter_since", "主要メッセージに含める最初のUnix秒。下限を指定しなかった場合は存在しません。"),
			integerField("filter_until", "主要メッセージに含めない最初のUnix秒。上限を指定しなかった場合は存在しません。"),
			textField("filter_on", "--on を具体化した YYYY-MM-DD。--on を指定しなかった場合は存在しません。"),
			textField("filter_time_zone", "--on の暦日境界に使用した Asia/Tokyo。--on を指定しなかった場合は存在しません。"),
			integerField("selection_start_index", "適用した1始まりの主要メッセージ開始順位。"),
			integerField("selection_count", "要求した主要メッセージの最大件数。--count を指定しなかった場合は存在しません。"),
			integerField("items_per_page", "この選択で実際に返した主要メッセージ数。返信コンテキストは含みません。"),
			integerField("next_start_index", "同じ条件と変化していない上限付き範囲で、次の未選択主要メッセージから続けるための --start-index 値。候補が残る場合だけ存在します。"),
			OutputField{Name: "filter_senders", Type: OutputFieldTypeArray, Description: "OR条件の基点として使用した、完全一致する正規の送信者アカウント参照。絞り込みを要求しなかった場合は存在しません。"},
			textField("filter_context", "実際に適用した none または replies のコンテキストポリシー。既定の none で索引選択だけを行った場合は省略されます。"),
			OutputField{Name: "anchor_sequences", Type: OutputFieldTypeArray, Description: "主要メッセージとして選択した、プロバイダー元のシーケンス。表示される基点以外のシーケンスは返信コンテキストです。"},
		)
	}
	return result
}
func taskFields(room, task, account, message string, collection bool) []OutputField {
	taskDescription := "正規のタスク参照。"
	roomDescription := "親ルームの正規参照。"
	accountDescription := "担当者アカウントの正規参照。"
	messageDescription := "タスクメッセージの正規参照。"
	bodyDescription := "信頼されていない外部テキストとしてのタスク本文。"
	if collection {
		taskDescription = "1番目にある正規のタスク参照。"
		roomDescription = "2番目にある親ルームの正規参照。"
		accountDescription = "3番目にある担当者アカウントの正規参照。"
		messageDescription = "4番目にあるタスクメッセージの正規参照。"
		bodyDescription = "5番目にある、ターミナルで安全な引用済みタスク本文。"
	}
	result := fields(refField("task_ref", task, taskDescription), refField("room_ref", room, roomDescription), refField("account_ref", account, accountDescription), refField("message_ref", message, messageDescription), textField("body", bodyDescription), textField("status", "タスクの完了ステータス。"), integerField("limit_time", "Unix期限。期限なしの場合は0。"))
	if collection {
		result = append(result, limitField(), completeField())
	}
	return result
}
func fileFields(room, file, account, message string, collection bool) []OutputField {
	fileDescription := "正規のファイル参照。"
	roomDescription := "親ルームの正規参照。"
	accountDescription := "アップロード者アカウントの正規参照。"
	messageDescription := "ファイルメッセージの正規参照。"
	nameDescription := "信頼されていない外部テキストとしてのファイル名。"
	if collection {
		fileDescription = "1番目にある正規のファイル参照。変更せず files show --file に渡します。"
		roomDescription = "2番目にある親ルームの正規参照。変更せず files show --room に渡します。"
		accountDescription = "3番目にあるアップロード者アカウントの正規参照。"
		messageDescription = "4番目は、存在する場合は正規のファイルメッセージ参照、存在しない場合はリテラル absent です。absent を参照として渡してはいけません。"
		nameDescription = "5番目にある、ターミナルで安全な引用済みファイル名。"
	}
	result := fields(refField("file_ref", file, fileDescription), refField("room_ref", room, roomDescription), refField("account_ref", account, accountDescription), refField("message_ref", message, messageDescription), textField("name", nameDescription), integerField("size", "ファイルサイズ（バイト）。"))
	if collection {
		return append(result, limitField(), completeField())
	}
	return append(result, textField("download_url", "プロバイダーが返した場合の、要求済みダウンロードURL。"))
}
func inviteFields(invite string) []OutputField {
	return fields(refField("invite_ref", invite, "update/delete に変更せず渡せる正規の招待リンク参照。"), booleanField("public", "招待リンクが有効かどうか。"), textField("url", "有効かつ空でない場合の招待URL。"), booleanField("needs_approval", "参加に管理者の承認が必要かどうか。有効な場合に出力されます。"), textField("description", "有効な場合にプロバイダーが返す、空でない招待の説明。"))
}
func unreadFields() []OutputField {
	return fields(integerField("unread", "変更後のルーム未読件数。"), integerField("mentions", "変更後のルーム未読メンション件数。"))
}
