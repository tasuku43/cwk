package cli

import (
	"fmt"
	"strconv"

	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/cli/capsule"
	"github.com/tasuku43/cwk/internal/domain/chatwork"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

// chatworkPresentationOptions contains invocation-local projection controls.
// These values are removed before the provider-independent request is built.
type chatworkPresentationOptions struct {
	MaxBytes int
}

func splitChatworkPresentationArguments(command CommandSpec, arguments chatworkArguments) (chatworkArguments, chatworkPresentationOptions, error) {
	requestArguments := make(chatworkArguments, len(arguments))
	for name, values := range arguments {
		requestArguments[name] = append([]string(nil), values...)
	}
	values := requestArguments["--max-bytes"]
	if len(values) == 0 {
		return requestArguments, chatworkPresentationOptions{}, nil
	}
	if command.chatwork == nil || command.chatwork.Task != chatwork.TaskMessagesList || len(values) != 1 {
		return nil, chatworkPresentationOptions{}, fmt.Errorf("--max-bytes は messages list で1回だけ指定できます")
	}
	maxBytes, err := strconv.Atoi(values[0])
	if err != nil || maxBytes < 1 || maxBytes > maxChatworkOutputBytes {
		return nil, chatworkPresentationOptions{}, fmt.Errorf("--max-bytes は 1 から %d までの整数で指定してください", maxChatworkOutputBytes)
	}
	delete(requestArguments, "--max-bytes")
	return requestArguments, chatworkPresentationOptions{MaxBytes: maxBytes}, nil
}

// renderChatworkResult applies a byte budget to the already assembled result.
// It performs no provider I/O and includes the final delivery line and newline
// in every budget measurement.
func renderChatworkResult(result chatwork.Result, options chatworkPresentationOptions) (string, error) {
	if options.MaxBytes == 0 {
		return capsule.Render(result)
	}
	if options.MaxBytes < 0 || options.MaxBytes > maxChatworkOutputBytes {
		return "", fmt.Errorf("Chatwork 出力予算は宣言済み上限内で指定する必要があります")
	}
	totalItems, err := chatworkcmd.MessageDeliveryAnchorCount(result)
	if err != nil {
		return "", err
	}
	renderItems := func(items int) (string, error) {
		delivery, planErr := chatworkcmd.PlanMessageDelivery(result, items)
		if planErr != nil {
			return "", planErr
		}
		return capsule.RenderWithOptions(result, capsule.Options{
			MaxBytes:        options.MaxBytes,
			MessageDelivery: &delivery,
		})
	}

	minimumItems := 0
	if totalItems > 0 {
		minimumItems = 1
	}
	minimumOutput, err := renderItems(minimumItems)
	if err != nil {
		return "", err
	}
	if len(minimumOutput) > options.MaxBytes {
		return "", outputBudgetTooSmallFault(options.MaxBytes, len(minimumOutput), totalItems > 0)
	}
	best := minimumOutput
	for items := minimumItems + 1; items <= totalItems; items++ {
		candidate, renderErr := renderItems(items)
		if renderErr != nil {
			return "", renderErr
		}
		if len(candidate) <= options.MaxBytes {
			best = candidate
		}
	}
	return best, nil
}

func outputBudgetTooSmallFault(maxBytes, minimumBytes int, includesItem bool) error {
	requirement := "固定メタデータ"
	if includesItem {
		requirement = "固定メタデータと最初の原子的なメッセージ単位"
	}
	return fault.New(
		fault.KindInvalidInput,
		"output_budget_too_small",
		fmt.Sprintf("--max-bytes %d では%sを収められません。最低 %d バイトが必要です。", maxBytes, requirement, minimumBytes),
		false,
		fault.NextAction{
			Command: "help messages list",
			Reason:  "--max-bytes を必要バイト数以上へ増やすか、送信者・期間・順位で対象を絞ってください。",
		},
	)
}
