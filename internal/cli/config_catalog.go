package cli

import (
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

const (
	commandSelectionCapability              = "cli.command-selection"
	commandSelectionUncertainMessageGrammar = "コマンド選択の保存結果を確定できません。expected-source=saved candidate-fingerprint=<sha256:64-lowercase-hex>。"
)

func commandSelectionUncertainMessage(fingerprint string) string {
	return "コマンド選択の保存結果を確定できません。expected-source=saved candidate-fingerprint=" + fingerprint + "。"
}

func configCommandSpecs() []CommandSpec {
	return []CommandSpec{{
		Path:    "config",
		Summary: "エージェントに表示するコマンドを選択する",
		Effect:  operation.EffectWrite,
		Role:    RoleAct,
		Agent: AgentContract{
			CapabilityID: commandSelectionCapability,
			Outcome:      "権限を変更せず、対話型ターミナルの選択画面で整理したコマンド表示を保存する",
			Inputs: []CommandInput{
				{Name: "selection", Source: InputSourceStdin, Required: true, Description: "対話型ターミナルで、上下キーで移動、Spaceで切り替え、Enterで保存します。qを押すと、前回保存した選択を変更せずに終了します。", AllowedValues: []string{}},
			},
			Output: CommandOutput{
				Formats:       []OutputFormat{OutputFormatText},
				DefaultFormat: OutputFormatText,
				Fields: []OutputField{
					{Name: "status", Type: OutputFieldTypeString, Description: "選択内容を置き換えて保存したか、以前のプロファイルを変更しなかったか。"},
					{Name: "visible", Type: OutputFieldTypeInteger, Description: "保存した場合だけ示す、確定後にエージェントのコマンド表示へ含めるChatworkコマンド数。"},
					{Name: "hidden", Type: OutputFieldTypeInteger, Description: "保存した場合だけ示す、確定後にエージェントのコマンド表示から除くChatworkコマンド数。"},
					{Name: "changed", Type: OutputFieldTypeInteger, Description: "保存した場合だけ示す、読み込んだ選択から表示設定が変わった現在のカタログ項目数。古い設定の整理件数は含めません。"},
					{Name: "cleaned", Type: OutputFieldTypeInteger, Description: "保存時に整理した、現在は存在しない設定と旧形式の設定の合計件数。0件の場合、成功出力では省略します。"},
				},
				Completeness: OutputCompletenessComplete,
			},
			FixedTarget: &FixedTargetContract{
				Scope:       FixedTargetScopeToolLocal,
				Kind:        "command-selection",
				StableID:    "default",
				Description: "エージェントに提示する、ユーザー単位で一つだけ存在するcwkコマンド完全一致パスの集合。",
			},
			Prerequisites: []string{"対話可能な標準入力・標準出力ターミナルと、書き込み可能なユーザー設定ディレクトリが必要です。この表示対象フィルターは、認可またはセキュリティ境界ではありません。"},
			Errors: []CommandError{
				declaredCommandError(fault.KindInvalidInput, "invalid_arguments", false, "help config", "コマンド引数を付けずに選択画面を実行してください。"),
				declaredCommandError(fault.KindInvalidInput, "command_selection_invalid", false, "config", "無効な保存済みコマンド選択を明示的に置き換えてください。"),
				declaredCommandError(fault.KindUnavailable, "command_selection_unsafe", false, "doctor", "ローカル設定ファイルまたはディレクトリを修復し、コマンド選択の診断結果を確認してください。"),
				declaredCommandError(fault.KindUnavailable, "command_selection_unavailable", true, "doctor", "ユーザー設定ディレクトリへのアクセスを復旧し、ローカル診断結果を確認してください。"),
				declaredCommandError(fault.KindUnavailable, "interactive_terminal_required", false, "help config", "対話可能な標準入力・標準出力ターミナルで選択画面を実行してください。"),
				declaredCommandError(fault.KindInternal, "terminal_setup_failed", true, "config", "ターミナルを使用可能な状態に戻してから、選択画面を再試行してください。"),
				declaredCommandError(fault.KindInternal, "terminal_restore_failed", false, "doctor", "再試行する前に、ローカルターミナルとコマンド選択の状態を確認してください。"),
				declaredCommandError(fault.KindInternal, "configuration_input_failed", false, "config", "読み取り可能な対話型ターミナルで再試行してください。"),
				declaredCommandError(fault.KindContract, "invalid_mutation_contract", false, "help config", "固定されたコマンド選択対象と影響宣言を修正してください。"),
				declaredCommandError(fault.KindContract, "missing_mutation_action", false, "help config", "コマンド選択の変更処理の構成を修正してください。"),
				declaredCommandError(fault.KindRejected, "missing_mutation_policy", false, "help config", "Enterによる明示的な変更ポリシーを復元してください。"),
				declaredCommandError(fault.KindRejected, "mutation_rejected", false, "config", "選択内容を正確に確認してからEnterを押してください。"),
				declaredCommandErrorWithMessageGrammar(fault.KindContract, "unclassified_mutation_outcome", false, commandSelectionUncertainMessageGrammar, "doctor", "再度変更する前に source=saved と候補選択のフィンガープリントが一致することを確認してください。"),
				declaredCommandError(fault.KindInternal, "output_write_failed", true, "doctor", "書き込み可能な出力ストリームを復旧した後、保存済みの選択内容を確認してください。"),
				declaredCommandError(fault.KindCanceled, "operation_canceled", true, "config", "呼び出し元の準備ができたら再試行してください。"),
			},
			Mutation: &MutationContract{
				TargetKind:   "command-selection",
				TargetInputs: []string{},
				Impact: operation.Impact{
					Cardinality:  operation.CardinalityOne,
					Notification: operation.DeclarationNo,
					AccessChange: operation.DeclarationNo,
					Destructive:  operation.DeclarationNo,
				},
			},
		},
		handler: runConfig,
	}}
}
