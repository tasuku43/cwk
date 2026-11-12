package cli

import (
	"context"
	"fmt"

	"github.com/tasuku43/cwk/internal/app/linkcmd"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

func messageLinkCommandSpec() CommandSpec {
	const path = "messages resolve-link"
	return CommandSpec{
		Path: path, Summary: "メッセージリンクから参照を取り出す", Args: "--url <message-link>",
		Effect: operation.EffectRead, Role: RoleDiscover, Configurable: true,
		Agent: AgentContract{
			CapabilityID: "chatwork.messages.manage", Outcome: "検証されていないリンク文字列からルームとメッセージの正規参照を取り出す",
			Inputs: []CommandInput{textFlag("--url", true, "https://www.chatwork.com/#!rid<room-id>-<message-id>（各1〜32桁、先頭ゼロ不可）。存在やアクセスは未確認。")},
			Output: CommandOutput{Formats: []OutputFormat{OutputFormatText}, DefaultFormat: OutputFormatText, Completeness: OutputCompletenessComplete, Fields: []OutputField{
				refField("room_ref", string(chatwork.ReferenceRoom), "リンクから抽出した正規ルーム参照。"),
				refField("message_ref", string(chatwork.ReferenceMessage), "リンクから抽出した正規メッセージ参照。"),
				textField("existence", "対象の存在は確認していないことを示します。"),
				textField("access", "対象へのアクセスは確認していないことを示します。"),
			}},
			Prerequisites: []string{}, Errors: []CommandError{
				declaredCommandError(fault.KindInvalidInput, "invalid_arguments", false, "help "+path, "メッセージリンクの形式を確認してください。"),
				declaredCommandError(fault.KindInternal, "output_write_failed", true, path, "書き込み可能な出力先で再試行してください。"),
				declaredCommandError(fault.KindCanceled, "operation_canceled", true, path, "呼び出し元の準備ができたら再試行してください。"),
			},
		},
		handler: runMessageLink,
	}
}

func runMessageLink(ctx context.Context, c *CLI, command CommandSpec, _ operation.Intent, args []string) int {
	parsed, _, err := parseChatworkInvocationArgumentsInView(command, args, c.catalog)
	if err != nil {
		return c.failUsage(ctx, "invalid_arguments", "メッセージリンクの指定を確認してください。", "help "+command.Path, "形式と必須入力を確認してください。")
	}
	result, err := linkcmd.Resolve(ctx, parsed.first("--url"))
	if err != nil {
		if ctx.Err() != nil {
			return c.fail(ctx, ctx.Err())
		}
		return c.failUsage(ctx, "invalid_arguments", "メッセージリンクの形式が正しくありません。", "help "+command.Path, "形式と必須入力を確認してください。")
	}
	return c.emit(ctx, []byte(fmt.Sprintf("room-ref=%s message-ref=%s existence=not-checked access=not-checked\n", result.Room.Value, result.Message.Value)))
}
