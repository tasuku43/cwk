package cli

import (
	"testing"

	"github.com/tasuku43/cwk/internal/domain/chatworkauth"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

func TestChatworkAuthCatalogValidatesWithTaskCatalog(t *testing.T) {
	if err := DefaultCatalog().Validate(); err != nil {
		t.Fatalf("Chatwork auth catalog validation failed: %v", err)
	}
}

func TestChatworkAuthCatalogHasExactFixedTargetAndEffects(t *testing.T) {
	specs := chatworkAuthCommandSpecs()
	if len(specs) != 3 {
		t.Fatalf("auth command count = %d, want 3", len(specs))
	}
	want := map[string]struct {
		role   CommandRole
		effect operation.Effect
	}{
		"auth login":  {RoleAct, operation.EffectCreate},
		"auth status": {RoleAct, operation.EffectRead},
		"auth logout": {RoleAct, operation.EffectWrite},
	}
	for _, spec := range specs {
		expected, exists := want[spec.Path]
		if !exists {
			t.Fatalf("unexpected auth command %q", spec.Path)
		}
		if spec.Role != expected.role || spec.Effect != expected.effect {
			t.Errorf("%s = (%s, %s), want (%s, %s)", spec.Path, spec.Role, spec.Effect, expected.role, expected.effect)
		}
		if spec.Agent.Authentication != nil {
			t.Errorf("%s incorrectly requires an already-established auth session", spec.Path)
		}
		if spec.Agent.FixedTarget == nil || spec.Agent.FixedTarget.Scope != FixedTargetScopeToolLocal || spec.Agent.FixedTarget.Kind != chatworkauth.TargetKind || spec.Agent.FixedTarget.StableID != chatworkauth.TargetStableID {
			t.Errorf("%s fixed target = %+v", spec.Path, spec.Agent.FixedTarget)
		}
		if len(spec.ProducedRefs()) != 0 || len(spec.ConsumedRefs()) != 0 {
			t.Errorf("%s unexpectedly exposes a target reference", spec.Path)
		}
	}
}

func TestChatworkOAuthSecretsNeverUseArgumentOrFlagInputs(t *testing.T) {
	login := chatworkAuthCommandSpecs()[0]
	sources := make(map[string]InputSource, len(login.Agent.Inputs))
	for _, input := range login.Agent.Inputs {
		sources[input.Name] = input.Source
	}
	if sources["--client-id"] != InputSourceFlag {
		t.Fatal("first-login public client ID must use the documented optional flag")
	}
	if sources["callback_url"] != InputSourceStdin {
		t.Fatal("the full OAuth callback must enter only through stdin")
	}
	for _, forbidden := range []string{"token", "access_token", "refresh_token", "client_secret", "authorization_code", "pkce_verifier", "state"} {
		if _, exists := sources[forbidden]; exists {
			t.Errorf("secret-bearing public input %q exists", forbidden)
		}
	}
}

func TestChatworkAuthMutationsBindExactTargets(t *testing.T) {
	specs := chatworkAuthCommandSpecs()
	login := specs[0]
	if login.Agent.Mutation == nil || login.Agent.Mutation.ParentInput != "" || login.Agent.Mutation.TargetIDInput != "" || login.Agent.Mutation.TargetInputs == nil || len(login.Agent.Mutation.TargetInputs) != 0 || login.Agent.Mutation.TargetKind != chatworkauth.TargetKind {
		t.Fatalf("login mutation = %+v", login.Agent.Mutation)
	}
	logout := specs[2]
	if logout.Agent.Mutation == nil || logout.Agent.Mutation.TargetIDInput != "" || logout.Agent.Mutation.TargetInputs == nil || len(logout.Agent.Mutation.TargetInputs) != 0 || logout.Agent.Mutation.TargetKind != chatworkauth.TargetKind {
		t.Fatalf("logout mutation = %+v", logout.Agent.Mutation)
	}
	if logout.Agent.Mutation.Impact.AccessChange != operation.DeclarationYes || logout.Agent.Mutation.Impact.Destructive != operation.DeclarationYes {
		t.Fatalf("logout impact = %+v", logout.Agent.Mutation.Impact)
	}
}
