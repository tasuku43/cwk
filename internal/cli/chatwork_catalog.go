package cli

import (
	"strings"

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

	commands := []CommandSpec{
		chatworkRead("account show", "認証済みのChatworkアカウントを表示する", "", RoleDiscover,
			"chatwork.account.inspect", "設定済みのChatworkトークンに結び付いたアカウントを正確に確認する",
			nil, fields(refField("account_ref", account, "ルーム作成とアカウント絞り込みで使用できる正規のアカウント参照。"), textField("name", "アカウントの表示名。"), textField("organization", "存在する場合の、空でない人間可読な組織名と部署情報。")), chatwork.TaskAccountShow),
		chatworkRead("account status", "未読、メンション、タスクの件数を表示する", "", RoleUtility,
			"chatwork.account.inspect", "認証済みアカウントのChatwork集計ステータスを確認する",
			nil, fields(integerField("unread", "未読メッセージの合計件数。"), integerField("mentions", "未読メンションの合計件数。"), integerField("tasks", "未完了タスクの合計件数。")), chatwork.TaskAccountStatus),
		chatworkRead("personal-tasks list", "認証済みアカウントに割り当てられた最大100件のタスクを一覧表示する", "[--assigned-by <account-ref>] [--status open|done]", RoleDiscover,
			"chatwork.personal-tasks.inspect", "個人タスクを最大100件、ルームや担当者などの参照と共に一覧表示する",
			[]CommandInput{refFlag("--assigned-by", false, account, "割り当て元を一つの完全一致アカウント参照で絞り込みます。"), enumFlag("--status", false, "タスクのステータスで絞り込みます。", "open", "done")},
			fields(refField("task_ref", task, "1番目にある正規のタスク参照。"), refField("room_ref", room, "2番目にある正規のルーム参照。"), refField("assigned_by_ref", account, "3番目にある正規の割り当て元アカウント参照。"), refField("message_ref", message, "4番目にある正規のタスクメッセージ参照。"), textField("body", "5番目にある、ターミナルで安全な引用済みタスク本文。"), textField("status", "6番目にあるタスクのステータス。"), limitField(), completeField()), chatwork.TaskPersonalTasksList),
		chatworkRead("contacts list", "Chatworkコンタクトを検索する", "", RoleDiscover,
			"chatwork.contacts.discover", "コンタクトをアカウントとダイレクトチャットの参照と共に一覧表示する",
			nil, fields(refField("account_ref", account, "1番目にある正規のコンタクトアカウント参照。"), refField("room_ref", room, "2番目にある正規のダイレクトチャット参照。"), textField("name", "3番目にある、ターミナルで安全な引用済みコンタクト表示名。"), textField("organization", "空でない組織名または部署情報がある場合に末尾へ付く任意項目。"), completeField()), chatwork.TaskContactsList),
		chatworkRead("rooms list", "参加中のChatworkルームを検索する", "[--query <text>]", RoleDiscover,
			"chatwork.rooms.manage", "参加中のルームを、名前の大文字小文字を区別しない部分一致で絞り、--room にそのまま使える room_ref として返す",
			[]CommandInput{textFlag("--query", false, "1〜255 UTF-8バイトのルーム名部分文字列。大文字小文字は区別せず、複数候補を自動選択しません。")}, roomDiscoveryFields(room), chatwork.TaskRoomsList),
		chatworkMutation("rooms create", "メンバーを正確に指定してグループチャットを作成する", "--account <account-ref> --name <text> --admin <account-ref> [--member <account-ref>] [--readonly <account-ref>] [--description <text>] [--icon "+roomIconPresetListForHelp()+"] [--invite-code <code>] [--invite-approval required|not-required] --confirm=access-change", RoleAct,
			"chatwork.rooms.manage", "検証済みの認証アカウント範囲で、メンバー構成とアクセスへの影響を明示してグループチャットを一つ作成する",
			[]CommandInput{refFlag("--account", true, account, "ルーム作成を、実行前に検証する認証アカウントの完全一致参照に結び付けます。"), textFlag("--name", true, "1〜255文字のルーム名。"), repeatedRefFlag("--admin", true, account, "管理者を一人追加します。複数追加する場合は繰り返し指定します。"), repeatedRefFlag("--member", false, account, "メンバーを一人追加します。複数追加する場合は繰り返し指定します。"), repeatedRefFlag("--readonly", false, account, "閲覧のみのメンバーを一人追加します。複数追加する場合は繰り返し指定します。"), textFlag("--description", false, "ルームの説明。"), enumFlag("--icon", false, "Chatwork公式のルームアイコンプリセット。", chatwork.RoomIconPresetValues()...), textFlag("--invite-code", false, "1〜50文字の英数字、アンダースコア、ハイフンからなる任意コードの招待リンクを同時に作成します。"), enumFlag("--invite-approval", false, "この承認要件を持つ招待リンクを作成します。", "required", "not-required"), confirmFlag(confirmAccessChange)},
			fields(refField("room_ref", room, "作成したルームの正規参照。")), chatwork.TaskRoomsCreate, confirmAccessChange, "rooms list",
			createMutation("chatwork-room", "--account", operation.CardinalityMany, yes, yes, no)),
		chatworkRead("rooms show", "指定したルームを一つ表示する", "--room <room-id>", RoleAct,
			"chatwork.rooms.manage", "名前で探し直さず、指定したルームを確認する",
			[]CommandInput{refFlag("--room", true, room, "確認するルームです。")}, roomFields(room, false), chatwork.TaskRoomsShow),
		chatworkMutation("rooms update", "指定したルームの説明情報を更新する", "--room <room-id> [--name <text>] [--description <text>] [--icon <preset>]", RoleAct,
			"chatwork.rooms.manage", "メンバー構成を変えずに、選択したルームの名前、説明、アイコンを更新する",
			[]CommandInput{refFlag("--room", true, room, "更新するルームです。"), textFlag("--name", false, "変更後のルーム名。"), textFlag("--description", false, "変更後のルーム説明。"), textFlag("--icon", false, "変更後の定義済みアイコンプリセット。Chatwork公式の一覧: "+roomIconPresetListForHelp()+"。")},
			fields(refField("room_ref", room, "更新したルームの正規参照。")), chatwork.TaskRoomsUpdate, "", "rooms show",
			writeMutation("chatwork-room", "--room", "", operation.CardinalityOne, no, no, no)),
		chatworkMutation("rooms leave", "指定したグループチャットから退席する", "--room <room-id> --confirm=destructive", RoleAct,
			"chatwork.rooms.manage", "破壊的影響とアクセスへの影響を明示して、選択したグループチャットから退席する",
			[]CommandInput{refFlag("--room", true, room, "退席するルームです。"), confirmFlag(confirmDestructive)}, fields(refField("room_ref", room, "退席したルームの正規参照。")), chatwork.TaskRoomsLeave, confirmDestructive, "rooms show",
			writeMutation("chatwork-room", "--room", "", operation.CardinalityMany, no, yes, yes)),
		chatworkMutation("rooms delete", "指定したグループチャットを完全に削除する", "--room <room-id> --confirm=destructive", RoleAct,
			"chatwork.rooms.manage", "選択したグループチャットとその中のデータを完全に削除する",
			[]CommandInput{refFlag("--room", true, room, "削除するルームです。"), confirmFlag(confirmDestructive)}, fields(refField("room_ref", room, "削除したルームの正規参照。")), chatwork.TaskRoomsDelete, confirmDestructive, "rooms show",
			writeMutation("chatwork-room", "--room", "", operation.CardinalityUnbounded, no, yes, yes)),
		chatworkRead("members find", "ルーム内の表示名からメンバー候補を探す", "--room <room-id> --query <text>", RoleDiscover,
			"chatwork.members.manage", "指定したルームの全メンバーから表示名の部分一致候補を返し、候補から一人を自動選択しない",
			[]CommandInput{
				refFlag("--room", true, room, "メンバー候補を探すルームです。"),
				textFlag("--query", true, "表示名に含まれる1〜255 UTF-8バイトの文字列。大文字小文字を区別した部分一致で照合し、空白やUnicode文字の形は変換しません。複数候補を自動選択しません。"),
			}, fields(textField("query", "照合に使った引用済み文字列。"), integerField("source_count", "照合した完全なルームメンバー数。"), integerField("candidate_count", "表示名が部分一致した候補数。"), refField("account_ref", account, "候補の正規アカウント参照。選択後は変更せず messages list --sender へ渡します。"), textField("name", "ターミナルで安全な引用済み候補表示名。"), textField("role", "候補のルーム権限。"), completeField()), chatwork.TaskMembersFind),
		chatworkRead("members list", "指定したルームのメンバーを一覧表示する", "--room <room-id>", RoleAct,
			"chatwork.members.manage", "指定した一つのルームについて、メンバーの名前と権限を一覧表示する",
			[]CommandInput{refFlag("--room", true, room, "メンバー構成を確認するルームです。")}, fields(refField("account_ref", account, "1番目にあるメンバーアカウントの正規参照。"), textField("name", "2番目にある、ターミナルで安全な引用済みメンバー表示名。"), textField("role", "3番目にあるメンバーの権限。"), completeField()), chatwork.TaskMembersList),
		chatworkMutation("members replace", "ルームのメンバー構成全体を置き換える", "--room <room-id> --admin <account-ref> [--member <account-ref>] [--readonly <account-ref>] --confirm=access-change", RoleAct,
			"chatwork.members.manage", "アクセスへの影響を明示して、選択したルームの権限別メンバー構成全体を置き換える",
			[]CommandInput{refFlag("--room", true, room, "メンバー構成を置き換えるルームです。"), repeatedRefFlag("--admin", true, account, "管理者アカウント。複数指定する場合は繰り返します。"), repeatedRefFlag("--member", false, account, "メンバーアカウント。複数指定する場合は繰り返します。"), repeatedRefFlag("--readonly", false, account, "閲覧のみのアカウント。複数指定する場合は繰り返します。"), confirmFlag(confirmAccessChange)},
			fields(integerField("administrators", "置き換え後の管理者数。"), integerField("members", "置き換え後のメンバー数。"), integerField("readonly", "置き換え後の閲覧のみメンバー数。")), chatwork.TaskMembersReplace, confirmAccessChange, "members list",
			writeMutation(room, "--room", "", operation.CardinalityMany, yes, yes, yes)),
		chatworkRead("messages list", "最新または差分の最大100件から、条件に合うメッセージを読む", "--room <room-id> [--window recent|changes] [--since <RFC3339>] [--until <RFC3339>] [--on <day>] [--start-index <index>] [--count <count>] [--sender <account-ref>] [--contains <text>] [--context none|replies] [--resolve-relations <count>] [--max-body-bytes <bytes>] [--max-bytes <bytes>] [--jst]", RoleAct,
			"chatwork.messages.manage", "取得できる最新または差分の最大100件から条件に合うメッセージを選びます。閲覧制限や届かない期間を空の結果と混同せず、指定件数まで明示された返信元を追加で確認します。",
			[]CommandInput{
				refFlag("--room", true, room, ""),
				enumFlag("--window", false, "recent=最新（既定）、changes=差分。", "recent", "changes"),
				textFlag("--since", false, "開始含む秒精度RFC3339、オフセット必須。"),
				textFlag("--until", false, "終了含まない秒精度RFC3339、オフセット必須。"),
				textFlag("--on", false, "固定Asia/Tokyo日付。YYYY-MM-DD/today/yesterday（相対日は開始時確定）。"),
				integerFlag("--start-index", false, "新しい順の1始まり順位（1〜100、既定1）。"),
				deprecatedIntegerFlag("--count", false, "主要アンカー数1〜100。--start-index起点、返信文脈は件数外。", "--limit"),
				repeatedRefFlag("--sender", false, account, "members find の account-ref をそのまま。完全一致、最大100回、反復OR。"),
				repeatedTextFlag("--contains", false, "リテラル部分一致・大小文字区別（1〜1024 UTF-8バイト、最大100語）。語句OR、送信者/期間AND。trim/Unicode正規化なし。"),
				enumFlag("--context", false, "replies（既定）は範囲内の型付き同室返信元/先を1ホップ追加。noneは除外。", "none", "replies"),
				integerFlag("--resolve-relations", false, "返信元を再帰取得（0〜100、既定5、0=無効）。"),
				integerFlag("--max-body-bytes", false, "生UTF-8本文0〜65535バイトに制限。文字境界で切り、body-truncatedと元バイト数を表示。"),
				integerFlag("--max-bytes", false, "標準出力合計1〜16777216バイト。本文分割なし。"),
				boolFlag("--jst", "+09:00日時を併記。"),
			}, messageFields(room, message, account, true), chatwork.TaskMessagesList),
		chatworkMutation("messages send", "指定したルームへメッセージを送信する", "--room <room-id> [--to <account-ref>] [--reply-to <message-ref>] [--body <text>] [--body-file <path>] [--self-unread]", RoleAct,
			"chatwork.messages.manage", "選択したルームへ、指定した宛先と返信先を保ったメッセージ本文を一件送信する",
			[]CommandInput{
				refFlag("--room", true, room, "送信先のルームです。"),
				repeatedRefFlag("--to", false, account, "宛先アカウント参照を指定します。最大100回、指定順にTo通知します。重複参照は拒否します。"),
				refFlag("--reply-to", false, message, "同じルームの指定したメッセージに返信します。送信前にその一件を同じ認証で読み、投稿者を確認します。"),
				textFlag("--body", false, "UTF-8本文（展開後1〜65535バイト）。- は標準入力から読みます。"),
				textFlag("--body-file", false, "本文を読み取るローカルファイル。展開後UTF-8で1〜65535バイトです。"),
				boolFlag("--self-unread", "送信したメッセージを、認証済みアカウントで未読のままにします。"),
			},
			fields(refField("message_ref", message, "作成したメッセージの正規参照。"), refField("room_ref", room, "親ルームの正規参照。")), chatwork.TaskMessagesSend, "", "messages reconcile-send",
			createMutationWithPreflight("chatwork-message", "--room", []string{"--to", "--reply-to"}, operation.CardinalityOne, yes, no, no)),
		chatworkRead("messages reconcile-send", "送信結果が不明なとき、本文が完全一致する直近候補を探す", "--room <room-id> [--to <account-ref>] [--reply-to <message-ref>] [--body <text>] [--body-file <path>]", RoleDiscover,
			"chatwork.messages.manage", "認証アカウントの投稿から、送信と同じ本文展開の完全一致候補を直近100件で探す。候補の存在も不在もこの呼び出しの送信結果を証明しない",
			[]CommandInput{
				refFlag("--room", true, room, "照合するルームです。送信時と同じ参照を使います。"),
				repeatedRefFlag("--to", false, account, "送信時に指定した宛先アカウントを同じ順序で指定します。最大100件です。"),
				refFlag("--reply-to", false, message, "送信時に指定した同室の返信先を指定します。照合時に投稿者を同じ認証で読み直します。"),
				textFlag("--body", false, "送信時と同じ本文を指定します。- は標準入力から読みます。本文は出力しません。"),
				textFlag("--body-file", false, "送信時と同じ本文ファイルを指定します。本文は最大65,535バイトで、超過確認のためファイルは最大65,536バイトまで読み取ります。パスと本文は表示しません。"),
			}, messageSendReconciliationFields(room, message, account), chatwork.TaskMessagesReconcileSend),
		chatworkMutation("messages mark-read", "指定したメッセージまでを既読にする", "--room <room-id> --message <message-ref>", RoleAct,
			"chatwork.messages.manage", "指定したメッセージまでを既読にする",
			[]CommandInput{refFlag("--room", true, room, "対象のルームです。"), refFlag("--message", true, message, "既読にする範囲の境界となるメッセージです。")}, unreadFields(), chatwork.TaskMessagesMarkRead, "", "messages show",
			writeMutation(message, "--message", "--room", operation.CardinalityMany, no, no, no)),
		chatworkMutation("messages mark-unread", "指定したメッセージから未読にする", "--room <room-id> --message <message-ref>", RoleAct,
			"chatwork.messages.manage", "指定したメッセージから未読にする",
			[]CommandInput{refFlag("--room", true, room, "対象のルームです。"), refFlag("--message", true, message, "未読にする範囲の境界となるメッセージです。")}, unreadFields(), chatwork.TaskMessagesMarkUnread, "", "messages show",
			writeMutation(message, "--message", "--room", operation.CardinalityMany, no, no, no)),
		chatworkRead("messages show", "指定したメッセージと周辺文脈を表示する", "--room <room-id> --message <message-ref> [--context none|replies] [--around <count>] [--resolve-relations <count>] [--jst]", RoleAct,
			"chatwork.messages.manage", "指定したメッセージを中心に、安全な本文、明示的な直接返信、最新100件の範囲内の位置付き周辺メッセージ、および指定件数までの返信元連鎖を確認する",
			[]CommandInput{
				refFlag("--room", true, room, "メッセージがあるルームです。"),
				refFlag("--message", true, message, "確認するメッセージです。"),
				enumFlag("--context", false, "none は一件だけ、replies は最新100件を一度読み、型付き同一ルーム返信辺がアンカーを指す直接の子だけを追加します。省略時は一件だけです。", "none", "replies"),
				integerFlag("--around", false, "指定したメッセージの前後を最新範囲から各最大N件まで表示します（0〜100）。指定時は最新範囲を一度読みます。範囲外はoutside-window、制限中の不明な位置はunknownと示し、隣接だけから関係を推定しません。"),
				integerFlag("--resolve-relations", false, "返信元の追加取得件数（0〜100、値を省略すると5）。--context 省略時は replies、0でも replies の固定最新範囲を読みます。"),
				boolFlag("--jst", "+09:00日時を併記。"),
			}, messageFields(room, message, account, false), chatwork.TaskMessagesShow),
		chatworkRead("messages resources", "指定したメッセージに関連するファイルとルームタスクを探す", "--room <room-id> --message <message-ref> --kind files|tasks|all", RoleAct,
			"chatwork.message-resources.inspect", "最大100件のファイルまたはルームタスクから、指定したルームとメッセージの参照が完全一致する全件を返す。本文記法からイベント種別やメッセージの存在は推測しない",
			[]CommandInput{
				refFlag("--room", true, room, "リソースを探すルームです。"),
				refFlag("--message", true, message, "リソースとの関係を照合する完全一致メッセージ参照。本文記法や表示名は使いません。"),
				enumFlag("--kind", true, "各選択ソースを一度、最大100件読み取ります。all は files、room-tasks の順です。", "files", "tasks", "all"),
			}, messageResourceFields(room, message, file, task), chatwork.TaskMessagesResources),
		chatworkMutation("messages update", "指定したメッセージを更新する", "--room <room-id> --message <message-ref> [--body <text>] [--body-file <path>]", RoleAct,
			"chatwork.messages.manage", "指定したメッセージ一件の本文を置き換える",
			[]CommandInput{refFlag("--room", true, room, "対象のルームです。"), refFlag("--message", true, message, "更新するメッセージです。"), textFlag("--body", false, "変更後のUTF-8本文（1〜65535バイト）。- は標準入力から読みます。"), textFlag("--body-file", false, "変更後の本文を読み取るローカルファイル。UTF-8、1〜65535バイトです。")}, fields(refField("message_ref", message, "更新したメッセージの正規参照。")), chatwork.TaskMessagesUpdate, "", "messages show",
			writeMutation("chatwork-message", "--message", "--room", operation.CardinalityOne, no, no, no)),
		chatworkMutation("messages delete", "指定したメッセージを削除する", "--room <room-id> --message <message-ref> --confirm=destructive", RoleAct,
			"chatwork.messages.manage", "破壊的影響を明示して、指定したメッセージを削除する",
			[]CommandInput{refFlag("--room", true, room, "対象のルームです。"), refFlag("--message", true, message, "削除するメッセージです。"), confirmFlag(confirmDestructive)}, fields(refField("message_ref", message, "削除したメッセージの正規参照。")), chatwork.TaskMessagesDelete, confirmDestructive, "messages show",
			writeMutation("chatwork-message", "--message", "--room", operation.CardinalityOne, no, no, yes)),
		chatworkRead("room-tasks list", "指定したルームのタスクを最大100件一覧表示する", "--room <room-id> [--account <account-ref>] [--assigned-by <account-ref>] [--status open|done]", RoleAct,
			"chatwork.room-tasks.manage", "指定したルームのタスクを最大100件、タスクと担当者の参照を含めて一覧表示する",
			[]CommandInput{refFlag("--room", true, room, "タスクを探すルームです。"), refFlag("--account", false, account, "指定した担当者アカウントで絞り込みます。"), refFlag("--assigned-by", false, account, "指定した割り当て元アカウントで絞り込みます。"), enumFlag("--status", false, "タスクのステータスで絞り込みます。", "open", "done")}, taskFields(room, task, account, message, true), chatwork.TaskRoomTasksList),
		chatworkMutation("room-tasks create", "指定した担当者にルームタスクを作成する", "--room <room-id> --body <text> --assignee <account-ref> [--limit <unix-seconds|RFC3339>] [--limit-type date|time]", RoleAct,
			"chatwork.room-tasks.manage", "一つのルームで、指定したタスク本文を指定した担当者に割り当てる",
			[]CommandInput{refFlag("--room", true, room, "タスクを作るルームです。"), textFlag("--body", true, "タスク本文。"), repeatedRefFlag("--assignee", true, account, "担当者アカウント。複数追加する場合は繰り返し指定します。"), textFlag("--limit", false, "正のUnix秒、または秒精度RFC3339（明示タイムゾーン付き）。"), enumFlag("--limit-type", false, "期限を日付または時刻として解釈します。", "date", "time")}, fields(refField("task_ref", task, "作成したタスクの正規参照。"), refField("room_ref", room, "親ルームの正規参照.")), chatwork.TaskRoomTasksCreate, "", "room-tasks list",
			createMutation("chatwork-task", "--room", operation.CardinalityMany, yes, no, no)),
		chatworkRead("room-tasks show", "指定したルームタスクを一件表示する", "--room <room-id> --task <task-ref>", RoleAct,
			"chatwork.room-tasks.manage", "再検索せず、指定したタスクを一件確認する",
			[]CommandInput{refFlag("--room", true, room, "タスクがあるルームです。"), refFlag("--task", true, task, "確認するタスクです。")}, taskFields(room, task, account, message, false), chatwork.TaskRoomTasksShow),
		chatworkMutation("room-tasks set-status", "指定したタスクの完了状態を設定する", "--room <room-id> --task <task-ref> --status open|done", RoleAct,
			"chatwork.room-tasks.manage", "指定したタスクを open または done に設定する",
			[]CommandInput{refFlag("--room", true, room, "対象のルームです。"), refFlag("--task", true, task, "変更するタスクです。"), enumFlag("--status", true, "変更後のタスクステータス。", "open", "done")}, fields(refField("task_ref", task, "更新したタスクの正規参照。")), chatwork.TaskRoomTasksSetStatus, "", "room-tasks show",
			writeMutation("chatwork-task", "--task", "--room", operation.CardinalityOne, yes, no, no)),
		chatworkRead("files list", "指定したルームのファイルを最大100件一覧表示する", "--room <room-id> [--account <account-ref>]", RoleAct,
			"chatwork.files.manage", "指定したルームのファイルを最大100件一覧表示する。1番目と2番目の値は変更せず files show に渡す",
			[]CommandInput{refFlag("--room", true, room, "ファイルを探すルームです。"), refFlag("--account", false, account, "指定したアップロード者アカウントで絞り込みます。")}, fileFields(room, file, account, message, true), chatwork.TaskFilesList),
		chatworkMutation("files upload", "最大5 MiBのファイルを指定したルームへアップロードする", "--room <room-id> --path <file> [--message <text>]", RoleAct,
			"chatwork.files.manage", "選択したルームへ、5 MiB以下のローカルファイルを一つアップロードする",
			[]CommandInput{refFlag("--room", true, room, "アップロード先のルームです。"), textFlag("--path", true, "ローカルファイルのパス。通常ファイルで、最大5 MiBです。アップロード前に確認します。"), textFlag("--message", false, "ファイルに添付する任意のメッセージ。")}, fields(refField("file_ref", file, "アップロードしたファイルの正規参照。"), refField("room_ref", room, "親ルームの正規参照。")), chatwork.TaskFilesUpload, "", "files list",
			createMutation("chatwork-file", "--room", operation.CardinalityOne, yes, no, no)),
		chatworkRead("files show", "指定したルームファイルを一件表示する", "--room <room-id> --file <file-ref> [--create-download-url]", RoleAct,
			"chatwork.files.manage", "指定したファイルを一件確認し、必要な場合にダウンロードURLを要求する",
			[]CommandInput{refFlag("--room", true, room, "ファイルがあるルームです。"), refFlag("--file", true, file, "確認するファイルです。"), boolFlag("--create-download-url", "結果にプロバイダーのダウンロードURLを要求します。")}, fileFields(room, file, account, message, false), chatwork.TaskFilesShow),
		chatworkRead("invite-link show", "ルームの招待リンク状態を表示する", "--room <room-id>", RoleAct,
			"chatwork.invite-links.manage", "指定した一つのルームの招待リンク状態を確認する",
			[]CommandInput{refFlag("--room", true, room, "招待リンク状態を確認するルームです。")}, inviteFields(invite), chatwork.TaskInviteLinkShow),
		chatworkMutation("invite-link create", "ルームの招待リンクを作成する", "--room <room-id> [--code <code>] [--approval required|not-required] [--description <text>] --confirm=access-change", RoleAct,
			"chatwork.invite-links.manage", "アクセスへの影響を明示して、ルームの招待リンクを作成する",
			[]CommandInput{refFlag("--room", true, room, "招待リンクを作るルームです。"), textFlag("--code", false, "1〜50文字の英数字、アンダースコア、ハイフンからなる任意の招待リンクコード。"), enumFlag("--approval", false, "参加に管理者の承認が必要かどうか。", "required", "not-required"), textFlag("--description", false, "招待リンクの説明。"), confirmFlag(confirmAccessChange)}, inviteFields(invite), chatwork.TaskInviteLinkCreate, confirmAccessChange, "invite-link show",
			createMutation("chatwork-invite-link", "--room", operation.CardinalityOne, no, yes, no)),
		chatworkMutation("invite-link update", "指定した招待リンクを全項目置換する", "--invite <invite-ref> [--code <code>] [--regenerate-code] --approval required|not-required --description <text> --confirm=access-change", RoleAct,
			"chatwork.invite-links.manage", "コードの明示値または再生成、承認要件、非空の説明をすべて指定し、選択した招待リンクを置き換える",
			[]CommandInput{refFlag("--invite", true, invite, "invite-link show または create が返す参照をそのまま指定します。"), textFlag("--code", false, "変更後の1〜50文字の英数字、アンダースコア、ハイフンからなる招待リンクコード。--regenerate-codeとは同時に指定できません。"), boolFlag("--regenerate-code", "コードをランダム再生成する意図を明示します。--codeとは同時に指定できません。"), enumFlag("--approval", true, "変更後の承認要件。", "required", "not-required"), textFlag("--description", true, "変更後の空でない招待リンク説明。"), confirmFlag(confirmAccessChange)}, inviteFields(invite), chatwork.TaskInviteLinkUpdate, confirmAccessChange, "invite-link show",
			writeMutation("chatwork-invite-link", "--invite", "", operation.CardinalityOne, no, yes, no)),
		chatworkMutation("invite-link delete", "指定した招待リンクを無効にする", "--invite <invite-ref> --confirm=destructive", RoleAct,
			"chatwork.invite-links.manage", "アクセスへの破壊的影響を明示して、選択した招待リンクを無効にする",
			[]CommandInput{refFlag("--invite", true, invite, "招待リンク参照を指定します。"), confirmFlag(confirmDestructive)}, inviteFields(invite), chatwork.TaskInviteLinkDelete, confirmDestructive, "invite-link show",
			writeMutation("chatwork-invite-link", "--invite", "", operation.CardinalityOne, no, yes, yes)),
		chatworkRead("contact-requests list", "受信したコンタクト承認依頼を検索する", "", RoleDiscover,
			"chatwork.contact-requests.manage", "受信したコンタクト承認依頼を、完全一致の承認依頼参照とアカウント参照を含む固定位置スキーマで一覧表示する",
			nil, fields(refField("request_ref", request, "1番目にある受信済み承認依頼の正規参照。"), refField("account_ref", account, "2番目にある申請元アカウントの正規参照。"), textField("name", "3番目にある、ターミナルで安全な引用済み申請元アカウント名。"), textField("message", "末尾に任意で付く、ターミナルで安全な引用済み申請メッセージ。"), limitField(), completeField()), chatwork.TaskContactRequestsList),
		chatworkMutation("contact-requests accept", "指定したコンタクト承認依頼を承認する", "--request <request-ref> --confirm=access-change", RoleAct,
			"chatwork.contact-requests.manage", "選択した受信済みコンタクト承認依頼を承認する",
			[]CommandInput{refFlag("--request", true, request, "受信済み承認依頼の参照を指定します。"), confirmFlag(confirmAccessChange)}, fields(refField("account_ref", account, "承認したコンタクトアカウントの正規参照。"), refField("room_ref", room, "ダイレクトチャットの正規参照。")), chatwork.TaskContactRequestsAccept, confirmAccessChange, "contact-requests list",
			writeMutation("chatwork-contact-request", "--request", "", operation.CardinalityOne, yes, yes, no)),
		chatworkMutation("contact-requests reject", "指定したコンタクト承認依頼を拒否する", "--request <request-ref> --confirm=destructive", RoleAct,
			"chatwork.contact-requests.manage", "選択した受信済みコンタクト承認依頼を拒否する",
			[]CommandInput{refFlag("--request", true, request, "受信済み承認依頼の参照を指定します。"), confirmFlag(confirmDestructive)}, fields(refField("request_ref", request, "拒否したコンタクト承認依頼の正規参照。")), chatwork.TaskContactRequestsReject, confirmDestructive, "contact-requests list",
			writeMutation("chatwork-contact-request", "--request", "", operation.CardinalityOne, no, yes, yes)),
	}
	for index := range commands {
		switch commands[index].Path {
		case "members find":
			commands[index].DeprecatedAliases = []string{"rooms members find"}
		case "members list":
			commands[index].DeprecatedAliases = []string{"rooms members list"}
		case "members replace":
			commands[index].DeprecatedAliases = []string{"rooms members replace"}
		case "rooms list":
			commands[index].HumanRecipes = []HumanHelpRecipe{{
				Title: "ルーム名しか分からない状態から入る",
				Steps: []HumanHelpRecipeStep{{Path: "rooms list", Args: "--query <room-name>"}},
			}}
		case "messages list":
			commands[index].HelpConstraints = []string{
				"--onと--since/--untilは排他。since<until。",
				"--containsは取得済み範囲でローカル照合（API検索なし）。",
				"output-next-start-indexで続行。output-next-countを--countへ。",
			}
			commands[index].HumanRecipes = []HumanHelpRecipe{
				{
					Title: "人物の発言を期間で追う",
					Steps: []HumanHelpRecipeStep{
						{Path: "members find", Args: "--room <room-id> --query <name>"},
						{Path: "messages list", Args: "--room <room-id> --sender <account-ref> --since 2026-08-01T00:00:00+09:00 --until 2026-08-08T00:00:00+09:00"},
					},
					Note: "候補が複数なら account-ref を勝手に選ばず、対象を確認してから2段目へ進みます。",
				},
				{
					Title: "今日または昨日の会話を確認する",
					Steps: []HumanHelpRecipeStep{
						{Path: "messages list", Args: "--room <room-id> --on today"},
					},
					Note: "昨日は today を yesterday に置き換えます。",
				},
			}
		case "messages show":
			commands[index].DeprecatedAliases = []string{"messages get"}
			commands[index].HelpConstraints = []string{
				"--context none は正の --resolve-relations と併用できません。replies は最新100件内の型付き直接子だけを追加し、範囲外・閲覧制限を明示します。",
				"--around はアンカー前後を最新範囲から各N件まで選びます。replies と併用しても最新範囲を一度だけ読みます。アンカー不在時は outside-window または unknown とし、隣接から関係を推定しません。",
			}
		case "messages send", "messages reconcile-send", "messages update":
			commands[index].ExclusiveInputGroups = [][]string{{"--body", "--body-file"}}
			commands[index].HelpConstraints = []string{
				"--body と --body-file はどちらか一方だけを指定します。--body - は標準入力を読みます。",
			}
			if commands[index].Path == "messages reconcile-send" {
				commands[index].HelpConstraints = append(commands[index].HelpConstraints,
					"--to と --reply-to は送信時と同じ値・順序を指定し、送信時と同じ本文展開全体を完全一致で照合します。",
					"候補はこの送信呼び出しの証明ではありません。候補なしも未送信の証明ではありません。消費済み標準入力は同じ本文を再指定してください。",
				)
			}
			if commands[index].Path == "messages send" {
				commands[index].HelpConstraints = append(commands[index].HelpConstraints,
					"--to は最大100回、指定順に評価し、同じアカウント参照の重複は拒否します。",
					"--reply-to は同じルームで対象メッセージを一度読み、宛先と返信記法を本文の前に加えます。既存本文の記法は解析・変更しないため、指定と重複する場合があります。",
					"展開後の本文全体が1〜65535 UTF-8バイトである必要があります。",
				)
			}
		case "invite-link update":
			commands[index].ExclusiveInputGroups = [][]string{{"--code", "--regenerate-code"}}
			commands[index].HelpConstraints = []string{"--code と --regenerate-code はどちらか一方だけを指定します。"}
		}
	}
	return commands
}

func roomIconPresetListForHelp() string {
	return strings.Join(chatwork.RoomIconPresetValues(), "|")
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
		// Mutation recovery routes uncertain outcomes through the exact read-only
		// reconciliation task. Only the infrastructure's pre-dispatch fault can
		// safely recommend retrying the same invocation.
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
		declaredCommandError(fault.KindAuthentication, "chatwork_token_missing", false, "doctor", "CWK_API_TOKEN、設定ファイル、キーチェーンの探索結果を確認し、このコマンドのプロセスにトークンを設定してください。"),
		declaredCommandError(fault.KindAuthentication, "chatwork_token_invalid", false, "doctor", "CWK_API_TOKEN、設定ファイル、キーチェーンの探索結果を確認し、有効なプロセスローカルのトークンに置き換えてください。"),
		declaredCommandError(fault.KindAuthentication, "chatwork_authentication_failed", false, "doctor", "Chatwork 認証の診断結果を確認してから、設定済みトークンを置き換えてください。"),
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
	if mutation {
		errors = append(errors, declaredCommandError(
			fault.KindCanceled,
			"chatwork_mutation_not_dispatched",
			true,
			path,
			"送信前のキャンセルが確認されたため、同じ入力で変更を再実行できます。",
		))
	}
	if task == chatwork.TaskMessagesSend {
		errors = append(errors,
			declaredCommandError(fault.KindRateLimited, "chatwork_rate_limited", true, path, "返信先の読み取りがレート上限に達しました。返信先を確認し、メッセージが未送信であることを保ったまま再実行できます。"),
			declaredCommandError(fault.KindUnavailable, "chatwork_unavailable", true, path, "返信先の読み取りが利用可能になってから同じ送信入力を再実行してください。"),
			declaredCommandError(fault.KindCanceled, "chatwork_send_preflight_not_completed", true, path, "返信先の確認は送信前に停止しました。同じ入力で再実行できます。"),
			declaredCommandError(fault.KindCanceled, "chatwork_send_body_preflight_not_completed", true, path, "本文ファイルの送信前確認は返信先を読む前に停止しました。同じ入力で再実行できます。"),
			declaredCommandError(fault.KindUnavailable, "chatwork_reply_preflight_failed", true, path, "返信先を送信前に確認できませんでした。同じ入力で再実行できます。"),
			declaredCommandError(fault.KindContract, "chatwork_send_body_preparation_failed", false, "help "+path, "本文ファイルを返信先確認前に検証できませんでした。本文ファイルの読み取り契約を確認してください。"),
			declaredCommandError(fault.KindContract, "chatwork_reply_preflight_invalid", false, "help messages send", "返信先の完全一致結果が入力と一致しません。返信先の参照を確認してください。メッセージは送信されていません。"),
			declaredCommandError(fault.KindContract, "chatwork_message_limitation_invalid", false, "help messages send", "閲覧制限を推測せず、返信先を再確認してください。"),
			declaredCommandError(fault.KindPermission, "chatwork_message_restricted", false, "help messages send", "閲覧できるアカウントまたはプランで返信先を確認してください。"),
		)
	}
	if task == chatwork.TaskMessagesReconcileSend {
		errors = append(errors,
			declaredCommandError(fault.KindContract, "chatwork_reconciliation_body_failed", false, help, "照合本文を安全に準備できませんでした。読み取り専用照合の入力を確認してください。"),
			declaredCommandError(fault.KindContract, "chatwork_reconciliation_reply_invalid", false, help, "返信先が指定ルーム・メッセージ・投稿者に一致しないため照合できませんでした。"),
			declaredCommandError(fault.KindContract, "chatwork_message_limitation_invalid", false, help, "閲覧制限を推測せず、Chatworkの応答契約変更を確認してください。"),
			declaredCommandError(fault.KindPermission, "chatwork_message_restricted", false, help, "指定した返信先を現在のアカウントで読めないため照合できませんでした。"),
		)
	}
	if !mutation {
		errors = append(errors, declaredCommandError(fault.KindUnavailable, "chatwork_unavailable", true, path, "Chatworkが利用可能になってから再試行してください。"))
	}
	if task == chatwork.TaskMessagesList || task == chatwork.TaskMessagesShow {
		errors = append(errors, declaredCommandError(fault.KindContract, "chatwork_message_limitation_invalid", false, help, "閲覧制限を推測せず、Chatworkの応答契約変更を確認してください。"))
	}
	if task == chatwork.TaskMessagesList {
		errors = append(errors, declaredCommandError(fault.KindInvalidInput, "output_budget_too_small", false, help, "--max-bytes を必要量まで増やすか、送信者・期間・順位で対象を絞ってください。"))
	}
	if task == chatwork.TaskMessagesShow {
		errors = append(errors, declaredCommandError(fault.KindPermission, "chatwork_message_restricted", false, help, "閲覧できるアカウントまたはプランで対象を確認してください。"))
	}
	if task == chatwork.TaskMessagesSend || task == chatwork.TaskMessagesReconcileSend || task == chatwork.TaskMessagesUpdate {
		errors = append(errors,
			declaredCommandError(fault.KindInvalidInput, "chatwork_message_body_stdin_unreadable", false, help, "読み取り可能な標準入力を使うか、--body-file を指定してください。"),
			declaredCommandError(fault.KindInvalidInput, "chatwork_message_body_file_unreadable", false, help, "読み取り可能なローカルファイルを指定してください。"),
			declaredCommandError(fault.KindInvalidInput, "chatwork_message_body_too_large", false, help, "本文を65535バイト以下にしてください。"),
			declaredCommandError(fault.KindInvalidInput, "chatwork_message_body_invalid", false, help, "空でない有効なUTF-8本文を指定してください。"),
		)
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

func createMutationWithPreflight(kind, parent string, preflight []string, cardinality operation.Cardinality, notification, access, destructive operation.Declaration) MutationContract {
	mutation := createMutation(kind, parent, cardinality, notification, access, destructive)
	mutation.PreflightInputs = append([]string(nil), preflight...)
	return mutation
}

func writeMutation(kind, target, parent string, cardinality operation.Cardinality, notification, access, destructive operation.Declaration) MutationContract {
	targets := []string{target}
	if parent != "" {
		targets = append(targets, parent)
	}
	return MutationContract{TargetKind: kind, TargetInputs: targets, ParentInput: parent, TargetIDInput: target, Impact: operation.Impact{Cardinality: cardinality, Notification: notification, AccessChange: access, Destructive: destructive}}
}

func refFlag(name string, required bool, kind, description string) CommandInput {
	if name == "--room" && kind == string(chatwork.ReferenceRoom) {
		roomReferenceDescription := "rooms list の room-ref をそのまま（正の10進数、1〜32桁、先頭0なし）。"
		if description != "" {
			description = roomReferenceDescription + " " + description
		} else {
			description = roomReferenceDescription
		}
	}
	return CommandInput{Name: name, Source: InputSourceFlag, Required: required, Description: description, AllowedValues: []string{}, ReferenceKind: kind}
}
func repeatedRefFlag(name string, required bool, kind, description string) CommandInput {
	input := refFlag(name, required, kind, description)
	input.Repeatable = true
	return input
}
func repeatedTextFlag(name string, required bool, description string) CommandInput {
	input := textFlag(name, required, description)
	input.Repeatable = true
	return input
}
func textFlag(name string, required bool, description string) CommandInput {
	return CommandInput{Name: name, Source: InputSourceFlag, Required: required, Description: description, AllowedValues: []string{}}
}
func integerFlag(name string, required bool, description string) CommandInput {
	return textFlag(name, required, description)
}
func deprecatedIntegerFlag(name string, required bool, description string, aliases ...string) CommandInput {
	input := integerFlag(name, required, description)
	input.DeprecatedAliases = append([]string(nil), aliases...)
	return input
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
func refArrayField(name, kind, description string) OutputField {
	return OutputField{Name: name, Type: OutputFieldTypeArray, Description: description, ReferenceKind: kind}
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
func roomDiscoveryFields(room string) []OutputField {
	selection := fields(
		textField("query", "指定時だけ存在する、照合に使った引用済み文字列。"),
		integerField("source_count", "--query の照合対象となった参加中ルーム数。"),
		integerField("candidate_count", "--query に部分一致した候補ルーム数。"),
	)
	return append(selection, roomFields(room, true)...)
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
	result := fields(refField("message_ref", message, messageDescription), refField("room_ref", room, "親ルームの正規参照。"), refField("sender_ref", account, "送信者アカウントの正規参照。"), textField("sender_name", "構造を安全に区切った、信頼されていないテキストとしての送信者表示名。"), textField("body", bodyDescription), integerField("send_time", sendDescription), textField("send_time_jst", "--jst指定時だけ存在する固定+09:00の4桁RFC3339送信日時。年が0000〜9999外なら値は unavailable。"), textField("relation_state", "不完全な公式記法により関係集合を証明できない場合だけ unknown。完全な複数返信は unknown ではなく、省略時は reviewed relation set が完全です。"), OutputField{Name: "relations", Type: OutputFieldTypeArray, Description: "relation_state が unknown の場合は省略する、解決済みまたは未解決の型付きTo・返信・引用関係。返信は公式記法の順序を保ち、1件は reply=#N、複数は reply=[#N,#M] のように表します。各値は既存の解決済み・未解決文法を使います。"}, OutputField{Name: "quote_candidates", Type: OutputFieldTypeArray, Description: "引用ごとの元の上限付き取得範囲での照合結果。候補は同じルーム、正確な引用者アカウント参照、引用元とメッセージの Unix 送信時刻全体が一致する場合だけ表示します。message_ref は正規参照で、元の範囲内の位置は現在も表示対象である場合だけ付きます。状態は unique-window-candidate、ambiguous-window-candidates、no-window-candidate、missing-timestamp、unknown のいずれかです。scope は source-window、identity は常に unproven です。候補は引用を解決せず、返信辺・追加取得先・全履歴で一意なメッセージを証明しません。"})
	if !collection {
		return append(result,
			OutputField{Name: "reply_child_refs", Type: OutputFieldTypeArray, Description: "--context replies で、最新の上限付き範囲から型付き同一ルーム返信辺により証明した直接の子メッセージ正規参照。プロバイダー順です。", ReferenceKind: message},
			OutputField{Name: "reply_child_source_sequences", Type: OutputFieldTypeArray, Description: "各直接子の最新範囲内での1始まりのプロバイダー位置。"},
			integerField("reply_context_source_count", "--context replies の固定最新範囲で見えたメッセージ数。"),
			integerField("reply_context_source_limit", "直接子を探した固定最新範囲の上限100。"),
			booleanField("reply_context_complete", "直接子の探索範囲が完全か。false は範囲外の子が存在し得ることを示します。"),
			textField("reply_context_access_limitation", "直接子の探索範囲に対する none・partial・all の閲覧制限。"),
			textField("around_anchor_state", "--around 指定時のアンカー位置。in-window、outside-window、unknown のいずれかです。"),
			integerField("around_anchor_source_sequence", "in-window アンカーの最新範囲内の1始まり位置。位置を証明できない場合は存在しません。"),
			integerField("around_requested", "各側で要求した周辺メッセージ数（0〜100）。"),
			OutputField{Name: "around_before_refs", Type: OutputFieldTypeArray, Description: "アンカーより前に選んだ周辺メッセージの正規参照。プロバイダー順です。", ReferenceKind: message},
			OutputField{Name: "around_before_source_sequences", Type: OutputFieldTypeArray, Description: "各 before メッセージの最新範囲内での1始まり位置。"},
			integerField("around_actual_before", "実際に選んだアンカー前の周辺メッセージ数。"),
			OutputField{Name: "around_after_refs", Type: OutputFieldTypeArray, Description: "アンカーより後に選んだ周辺メッセージの正規参照。プロバイダー順です。", ReferenceKind: message},
			OutputField{Name: "around_after_source_sequences", Type: OutputFieldTypeArray, Description: "各 after メッセージの最新範囲内での1始まり位置。"},
			integerField("around_actual_after", "実際に選んだアンカー後の周辺メッセージ数。"),
			integerField("around_source_count", "周辺メッセージを調べた最新範囲で見えたメッセージ数。"),
			integerField("around_source_limit", "周辺位置を調べた最新範囲の上限100。"),
			booleanField("around_complete", "周辺位置を調べた最新範囲が完全か。"),
			textField("around_access_limitation", "周辺位置を調べた最新範囲に対する none・partial・all の閲覧制限。"),
			integerField("relation_fetch_limit", "返信関係を補う追加の一件取得上限。指定時だけ存在します。"),
			integerField("relation_fetch_attempts", "実際に行った追加の一件取得回数。指定時だけ存在します。"),
			OutputField{Name: "relation_resolution_targets", Type: OutputFieldTypeArray, Description: "明示的な同一ルーム返信先ごとの正規 message_ref、fetched・not-found・restricted・budget-exhausted 状態、および取得できた補足メッセージ。"},
		)
	}
	if collection {
		result = append(result,
			booleanField("body_truncated", "--max-body-bytes を指定し、元の本文が上限を超えたレコードにだけ body-truncated=true として存在します。"),
			integerField("original_body_bytes", "body_truncated=true の本文について、表示投影前の元UTF-8本文バイト数。"),
			integerField("sequence", "プロバイダーが返した元のメッセージ範囲内での1始まりの位置。選択後の出力では連番にならない場合があります。"),
			textField("actor_alias", "文書内だけで決定論的に使う送信者別名。コマンドの識別子にはなりません。"),
			textField("window", "要求した recent または差分メッセージ範囲。"),
			integerField("source_limit", "ローカルでメッセージを選択する前の、プロバイダー取得範囲の上限。"), completeField(),
			textField("access_limitation", "当該取得範囲の閲覧制限。none は制限ヘッダーなし、partial は一部制限、all は全件制限です。"),
			integerField("unresolved_replies", "正規の返信先を解決できない型付き返信関係の数。"),
			integerField("unresolved_quotes", "quote_candidates があっても解決扱いにならない、未解決の型付き引用関係の数。"),
			integerField("unknown_relation_sets", "不完全な公式記法により関係集合が unknown となったメッセージ数。"),
			refField("oldest_reachable_message_ref", message, "recent・閲覧制限なし・非空の取得範囲で、messages list が到達できた最古の正規メッセージ参照。条件を証明できない場合は存在しません。"),
			integerField("oldest_reachable_send_time", "oldest_reachable_message_ref のUnix送信時刻。境界を証明できない場合は存在しません。"),
			textField("period_reachability", "期間指定が最古到達境界内、境界を一部超過、全体が境界外、または判定不能のいずれか。期間を指定しなかった場合は存在しません。"),
			integerField("relation_fetch_limit", "返信連鎖を補う追加の一件取得上限。既定値は5で、--resolve-relations 0 は無効化します。"),
			integerField("relation_fetch_attempts", "実際に行った追加の一件取得回数。元の取得範囲内での解決は含みません。"),
			OutputField{Name: "relation_resolution_targets", Type: OutputFieldTypeArray, Description: "表示対象のすべての返信辺から再帰的に到達した同一ルーム返信先ごとの正規 message_ref、source・fetched・not-found・restricted・budget-exhausted 状態、および解決できた補足メッセージ。"},
			integerField("source_count", "有効なローカル選択を適用する前にプロバイダーが返したメッセージ数。選択を要求しなかった場合は存在しません。"),
			integerField("candidate_count", "任意の送信者・本文語句・期間照合後かつ start-index/count 適用前にある、上限付き取得範囲内の主要メッセージ候補数。選択を要求しなかった場合は存在しません。"),
			integerField("filter_since", "主要メッセージに含める最初のUnix秒。下限を指定しなかった場合は存在しません。"),
			integerField("filter_until", "主要メッセージに含めない最初のUnix秒。上限を指定しなかった場合は存在しません。"),
			textField("filter_on", "--on を具体化した YYYY-MM-DD。--on を指定しなかった場合は存在しません。"),
			textField("filter_time_zone", "--on の暦日境界に使用した Asia/Tokyo。--on を指定しなかった場合は存在しません。"),
			integerField("selection_start_index", "適用した1始まりの主要メッセージ開始順位。"),
			integerField("selection_count", "要求した主要メッセージの最大件数。--count を指定しなかった場合は存在しません。"),
			integerField("items_per_page", "この選択で実際に返した主要メッセージ数。返信コンテキストは含みません。"),
			integerField("next_start_index", "同じ条件と変化していない上限付き範囲で、次の未選択主要メッセージから続けるための --start-index 値。候補が残る場合だけ存在します。"),
			OutputField{Name: "filter_senders", Type: OutputFieldTypeArray, Description: "OR条件の基点として使用した、完全一致する正規の送信者アカウント参照。絞り込みを要求しなかった場合は存在しません。"},
			OutputField{Name: "filter_contains", Type: OutputFieldTypeArray, Description: "照合に使った順序と重複を保つ本文語句。各候補は少なくとも一語を大文字小文字を区別して部分一致し、他の選択条件とはANDです。未指定時は存在しません。"},
			textField("filter_context", "選択時に実際に適用した none または replies のコンテキストポリシー。省略時の既定値は replies です。"),
			OutputField{Name: "anchor_sequences", Type: OutputFieldTypeArray, Description: "主要メッセージとして選択した、プロバイダー元のシーケンス。表示される基点以外のシーケンスは返信コンテキストです。"},
			integerField("max_bytes", "--max-bytes 指定時だけ最終配送行にある、末尾改行を含む標準出力予算。"),
			booleanField("output_complete", "--max-bytes 指定時だけ存在し、この呼び出しの主要アンカーをすべて配送したかを示します。プロバイダーの complete とは別です。"),
			integerField("output_items", "--max-bytes 指定時だけ存在する、配送した主要アンカー数。返信コンテキストと関係補足は数えません。"),
			integerField("output_total_items", "--max-bytes 指定時だけ存在する、この呼び出しで配送対象だった主要アンカー総数。"),
			integerField("output_next_start_index", "予算で未配送の主要アンカーがある場合だけ存在する、同じ条件で次のnewest-first順位から続ける --start-index 値。"),
			integerField("output_next_count", "元の --count 範囲が予算で未配送の場合だけ存在する、続きで --count を置き換える残り主要アンカー数。"),
		)
	}
	return result
}

func messageSendReconciliationFields(room, message, account string) []OutputField {
	return fields(
		refField("room_ref", room, "送信結果を照合したルームの正規参照。"),
		refField("account_ref", account, "照合に使用した認証アカウントの正規参照。候補がなくても表示します。"),
		textField("candidate_status", "候補をすべて見つけたか、直近の可視範囲に一致がないか。"),
		textField("outcome", "常にunknown。完全一致候補は、この送信呼び出しが作成した証明ではありません。"),
		integerField("candidate_count", "直近の可視範囲で本人の本文全体が完全一致したメッセージ数。"),
		integerField("source_count", "候補を照合した直近の取得範囲で見えたメッセージ数。"),
		integerField("source_limit", "Chatworkの直近メッセージ取得上限100。"),
		booleanField("complete", "ルーム全履歴に対する完全性。上限付きの直近範囲ではfalseです。"),
		textField("access_limitation", "候補を照合した取得範囲に対するnone・partial・allの閲覧制限。"),
		textField("uncertainty", "過去の同文、送信時と異なるアカウント、遅延、窓外、閲覧制限、編集・削除、変更された本文ファイル、消費済み標準入力の再指定、変化した返信先投稿者により、候補は送信証明にならず候補なしも未送信の証明にはなりません。"),
		refField("candidate_message_ref", message, "完全一致した候補メッセージの正規参照。表示された参照は変更せず messages show に渡せます。"),
		refField("sender_ref", account, "候補メッセージの送信者の正規参照。認証アカウントの参照と一致します。"),
		textField("sender_name", "候補の送信者名。信頼されていない外部テキストとして安全に引用します。"),
		integerField("send_time", "候補メッセージのUnix送信時刻。"),
	)
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
func messageResourceFields(room, message, file, task string) []OutputField {
	return fields(
		textField("kind", "選択したソース（files、tasks、all）。"),
		refField("room_ref", room, "選択に使った完全一致ルーム参照。変更せず files show または room-tasks show に渡します。"),
		refField("message_ref", message, "照合に使った完全一致メッセージ参照。メッセージの存在は主張しません。"),
		integerField("files_source_count", "files を選択した場合に照合した上限付きルームファイルソースの件数。"),
		integerField("files_limit", "files を選択した場合の固定ソース上限100。"),
		booleanField("files_complete", "files のルーム全体に対する完全性。上限付きソースではfalse。"),
		textField("files_absence", "files の一致が0件でも不在は確認できないため、選択時の値は常に unknown。"),
		integerField("files_matched", "files の上限付きソース内で完全一致するルームとメッセージ参照を持つ件数。"),
		refArrayField("file_ref", file, "files を選択した場合に返す全一致ファイルの正規参照。各値を変更せず、room_ref と共に files show --file に渡します。"),
		integerField("tasks_source_count", "tasks を選択した場合に照合した上限付きルームタスクソースの件数。"),
		integerField("tasks_limit", "tasks を選択した場合の固定ソース上限100。"),
		booleanField("tasks_complete", "tasks のルーム全体に対する完全性。上限付きソースではfalse。"),
		textField("tasks_absence", "tasks の一致が0件でも不在は確認できないため、選択時の値は常に unknown。"),
		integerField("tasks_matched", "tasks の上限付きソース内で完全一致するルームとメッセージ参照を持つ件数。"),
		refArrayField("task_ref", task, "tasks を選択した場合に返す全一致ルームタスクの正規参照。各値を変更せず、room_ref と共に room-tasks show --task に渡します。"),
	)
}
func inviteFields(invite string) []OutputField {
	return fields(refField("invite_ref", invite, "update/delete に変更せず渡せる正規の招待リンク参照。"), booleanField("public", "招待リンクが有効かどうか。"), textField("url", "有効かつ空でない場合の招待URL。"), booleanField("needs_approval", "参加に管理者の承認が必要かどうか。有効な場合に出力されます。"), textField("description", "有効な場合にプロバイダーが返す、空でない招待の説明。"))
}
func unreadFields() []OutputField {
	return fields(integerField("unread", "変更後のルーム未読件数。"), integerField("mentions", "変更後のルーム未読メンション件数。"))
}
