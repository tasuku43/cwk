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
	MaxBytes           int
	MaxBodyBytes       int
	LimitMessageBodies bool
	IncludeSendTimeJST bool
}

func splitChatworkPresentationArguments(command CommandSpec, arguments chatworkArguments) (chatworkArguments, chatworkPresentationOptions, error) {
	requestArguments := make(chatworkArguments, len(arguments))
	for name, values := range arguments {
		requestArguments[name] = append([]string(nil), values...)
	}
	options := chatworkPresentationOptions{}
	if values := requestArguments["--jst"]; len(values) > 0 {
		if command.chatwork == nil || (command.chatwork.Task != chatwork.TaskMessagesList && command.chatwork.Task != chatwork.TaskMessagesShow) || len(values) != 1 || values[0] != "true" {
			return nil, chatworkPresentationOptions{}, fmt.Errorf("--jst は messages list または messages show で値なしのフラグとして指定できます")
		}
		options.IncludeSendTimeJST = true
		delete(requestArguments, "--jst")
	}
	if values := requestArguments["--max-bytes"]; len(values) > 0 {
		if command.chatwork == nil || command.chatwork.Task != chatwork.TaskMessagesList || len(values) != 1 {
			return nil, chatworkPresentationOptions{}, fmt.Errorf("--max-bytes は messages list で1回だけ指定できます")
		}
		maxBytes, err := strconv.Atoi(values[0])
		if err != nil || maxBytes < 1 || maxBytes > maxChatworkOutputBytes {
			return nil, chatworkPresentationOptions{}, fmt.Errorf("--max-bytes は 1 から %d までの整数で指定してください", maxChatworkOutputBytes)
		}
		options.MaxBytes = maxBytes
		delete(requestArguments, "--max-bytes")
	}
	if values := requestArguments["--max-body-bytes"]; len(values) > 0 {
		if command.chatwork == nil || command.chatwork.Task != chatwork.TaskMessagesList || len(values) != 1 {
			return nil, chatworkPresentationOptions{}, fmt.Errorf("--max-body-bytes は messages list で1回だけ指定できます")
		}
		maxBodyBytes, err := parseNonNegativeDecimal(values[0])
		if err != nil || maxBodyBytes > chatwork.MaxMessageBodyBytes {
			return nil, chatworkPresentationOptions{}, fmt.Errorf("--max-body-bytes は 0 から %d までの十進整数で指定してください", chatwork.MaxMessageBodyBytes)
		}
		options.MaxBodyBytes = maxBodyBytes
		options.LimitMessageBodies = true
		delete(requestArguments, "--max-body-bytes")
	}
	return requestArguments, options, nil
}

func parseNonNegativeDecimal(value string) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("empty decimal value")
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return 0, fmt.Errorf("value is not an ASCII decimal integer")
		}
	}
	return strconv.Atoi(value)
}

// renderChatworkResult applies a byte budget to the already assembled result.
// It performs no provider I/O and includes the final delivery line and newline
// in every budget measurement.
func renderChatworkResult(result chatwork.Result, options chatworkPresentationOptions) (string, error) {
	if options.MaxBytes == 0 {
		if options.LimitMessageBodies || options.IncludeSendTimeJST {
			return capsule.RenderWithOptions(result, capsule.Options{
				MaxBodyBytes:       options.MaxBodyBytes,
				LimitMessageBodies: options.LimitMessageBodies,
				IncludeSendTimeJST: options.IncludeSendTimeJST,
			})
		}
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
			MaxBytes:           options.MaxBytes,
			MaxBodyBytes:       options.MaxBodyBytes,
			LimitMessageBodies: options.LimitMessageBodies,
			IncludeSendTimeJST: options.IncludeSendTimeJST,
			MessageDelivery:    &delivery,
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
