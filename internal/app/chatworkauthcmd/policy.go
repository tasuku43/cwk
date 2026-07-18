package chatworkauthcmd

import (
	"context"
	"fmt"

	"github.com/tasuku43/cwk/internal/domain/chatworkauth"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

// ExactTargetPolicy accepts only the catalog-declared single-account local
// target. It grants no reusable or broader OAuth authority.
type ExactTargetPolicy struct{}

func (ExactTargetPolicy) Check(ctx context.Context, intent operation.Intent) error {
	if ctx == nil || ctx.Err() != nil {
		return fmt.Errorf("authentication mutation context is unavailable")
	}
	if intent.Target.Kind != chatworkauth.TargetKind {
		return fmt.Errorf("authentication target kind is invalid")
	}
	switch intent.Effect {
	case operation.EffectCreate:
		if intent.Target.ParentID != chatworkauth.TargetStableID || intent.Target.ID != "" {
			return fmt.Errorf("authentication create scope does not match the fixed target")
		}
	case operation.EffectWrite:
		if intent.Target.ID != chatworkauth.TargetStableID || intent.Target.ParentID != "" {
			return fmt.Errorf("authentication write target does not match the fixed target")
		}
	default:
		return fmt.Errorf("authentication mutation effect is invalid")
	}
	return nil
}
