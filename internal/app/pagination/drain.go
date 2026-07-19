// Package pagination provides bounded, all-or-nothing page traversal.
package pagination

import (
	"context"

	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/page"
)

// Budget prevents a CLI from silently traversing an unbounded remote result.
type Budget struct {
	PageSize int
	MaxPages int
	MaxItems int
}

// Validate rejects omitted limits; each derived command must choose explicit
// bounds in its product contract.
func (b Budget) Validate() error {
	if b.PageSize <= 0 || b.MaxPages <= 0 || b.MaxItems <= 0 {
		return fault.New(
			fault.KindContract,
			"invalid_pagination_budget",
			"ページサイズ、最大ページ数、最大項目数は正の値である必要があります",
			false,
		)
	}
	return nil
}

// Fetch retrieves one page for the exact opaque request token.
type Fetch[T any] func(context.Context, page.Request) (page.Result[T], error)

// Drain returns a complete result or no result. It detects cursor loops,
// enforces page/item budgets, and never converts an incomplete traversal into
// successful partial output.
func Drain[T any](ctx context.Context, budget Budget, fetch Fetch[T]) ([]T, error) {
	if err := budget.Validate(); err != nil {
		return nil, err
	}
	if ctx == nil {
		return nil, fault.New(fault.KindContract, "missing_context", "ページ処理コンテキストが設定されていません", false)
	}
	if fetch == nil {
		return nil, fault.New(fault.KindContract, "missing_page_fetcher", "ページ取得処理が設定されていません", false)
	}
	if err := ctx.Err(); err != nil {
		return nil, fault.Wrap(fault.KindCanceled, "operation_canceled", "ページ処理がキャンセルされました", true, err)
	}

	items := make([]T, 0)
	seenTokens := map[string]struct{}{"": {}}
	token := ""
	for pageNumber := 1; ; pageNumber++ {
		if pageNumber > budget.MaxPages {
			return nil, fault.New(
				fault.KindContract,
				"pagination_page_limit",
				"ページ処理が宣言済みのページ数上限内に完了しませんでした",
				false,
			)
		}
		if err := ctx.Err(); err != nil {
			return nil, fault.Wrap(fault.KindCanceled, "operation_canceled", "ページ処理がキャンセルされました", true, err)
		}
		request := page.Request{Token: token, Size: budget.PageSize}
		result, err := fetch(ctx, request)
		if contextErr := ctx.Err(); contextErr != nil {
			return nil, fault.Wrap(fault.KindCanceled, "operation_canceled", "ページ処理がキャンセルされました", true, contextErr)
		}
		if err != nil {
			if structured, ok := fault.PublicCopy(err); ok {
				return nil, structured
			}
			return nil, fault.Wrap(
				fault.KindUnavailable,
				"page_fetch_failed",
				"ページを取得できなかったため、部分的な結果は出力されませんでした",
				true,
				err,
			)
		}
		if err := result.Validate(); err != nil {
			return nil, fault.Wrap(fault.KindContract, "invalid_page_contract", "ページレスポンス契約は無効です", false, err)
		}
		if len(result.Items) > request.Size {
			return nil, fault.New(
				fault.KindContract,
				"invalid_page_contract",
				"ページレスポンスが要求したページサイズを超えたため、部分的な結果は出力されませんでした",
				false,
			)
		}
		if len(result.Items) > budget.MaxItems-len(items) {
			return nil, fault.New(
				fault.KindContract,
				"pagination_item_limit",
				"ページ処理が宣言済みの項目数上限内に完了しませんでした",
				false,
			)
		}
		items = append(items, result.Items...)
		if result.NextToken == "" {
			if contextErr := ctx.Err(); contextErr != nil {
				return nil, fault.Wrap(fault.KindCanceled, "operation_canceled", "完了前にページ処理がキャンセルされました", true, contextErr)
			}
			return items, nil
		}
		if _, exists := seenTokens[result.NextToken]; exists {
			return nil, fault.New(
				fault.KindContract,
				"pagination_cursor_loop",
				"ページ処理が同じカーソルを再度返したため、部分的な結果は出力されませんでした",
				false,
			)
		}
		seenTokens[result.NextToken] = struct{}{}
		token = result.NextToken
	}
}
