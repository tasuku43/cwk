package chatworkauthcmd

import (
	"context"
	"fmt"

	"github.com/tasuku43/cwk/internal/domain/chatworkauth"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

// ExactProfilePolicy accepts only the fixed profile that the caller already
// discovered and passed unchanged. It grants no reusable or broader OAuth
// authority.
type ExactProfilePolicy struct {
	Profile chatworkauth.ProfileReference
}

func (p ExactProfilePolicy) Check(ctx context.Context, intent operation.Intent) error {
	if ctx == nil || ctx.Err() != nil {
		return fmt.Errorf("authentication mutation context is unavailable")
	}
	if !p.Profile.Valid() {
		return fmt.Errorf("authentication profile is invalid")
	}
	switch intent.Effect {
	case operation.EffectCreate:
		if intent.Target.ParentID != p.Profile.Value() || intent.Target.ID != "" {
			return fmt.Errorf("authentication create scope does not match the exact profile")
		}
	case operation.EffectWrite:
		if intent.Target.ID != p.Profile.Value() || intent.Target.ParentID != "" {
			return fmt.Errorf("authentication write target does not match the exact profile")
		}
	default:
		return fmt.Errorf("authentication mutation effect is invalid")
	}
	return nil
}
