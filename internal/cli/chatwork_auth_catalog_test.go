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

func TestChatworkAuthCatalogHasExactReferenceChainAndEffects(t *testing.T) {
	specs := chatworkAuthCommandSpecs()
	if len(specs) != 4 {
		t.Fatalf("auth command count = %d, want 4", len(specs))
	}
	want := map[string]struct {
		role   CommandRole
		effect operation.Effect
	}{
		"auth profiles": {RoleDiscover, operation.EffectRead},
		"auth login":    {RoleAct, operation.EffectCreate},
		"auth status":   {RoleAct, operation.EffectRead},
		"auth logout":   {RoleAct, operation.EffectWrite},
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
	}

	profiles := specs[0]
	if len(profiles.Agent.Output.Fields) == 0 || profiles.Agent.Output.Fields[0].ReferenceKind != chatworkauth.OAuthProfileReferenceKind {
		t.Fatal("auth profiles does not produce the OAuth profile reference")
	}
	for _, spec := range specs[1:] {
		if len(spec.Agent.Inputs) == 0 || spec.Agent.Inputs[0].Name != "--profile" || !spec.Agent.Inputs[0].Required || spec.Agent.Inputs[0].ReferenceKind != chatworkauth.OAuthProfileReferenceKind {
			t.Errorf("%s does not consume the exact required OAuth profile reference", spec.Path)
		}
	}
}

func TestChatworkOAuthSecretsNeverUseArgumentOrFlagInputs(t *testing.T) {
	login := chatworkAuthCommandSpecs()[1]
	sources := make(map[string]InputSource, len(login.Agent.Inputs))
	for _, input := range login.Agent.Inputs {
		sources[input.Name] = input.Source
	}
	if sources["CWK_OAUTH_CLIENT_ID"] != InputSourceEnvironment || sources["CWK_OAUTH_REDIRECT_URI"] != InputSourceEnvironment {
		t.Fatal("non-secret OAuth registration must use documented environment configuration")
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
	login := specs[1]
	if login.Agent.Mutation == nil || login.Agent.Mutation.ParentInput != "--profile" || login.Agent.Mutation.TargetIDInput != "" || len(login.Agent.Mutation.TargetInputs) != 1 {
		t.Fatalf("login mutation = %+v", login.Agent.Mutation)
	}
	logout := specs[3]
	if logout.Agent.Mutation == nil || logout.Agent.Mutation.TargetIDInput != "--profile" || logout.Agent.Mutation.TargetKind != chatworkauth.OAuthProfileReferenceKind {
		t.Fatalf("logout mutation = %+v", logout.Agent.Mutation)
	}
	if logout.Agent.Mutation.Impact.AccessChange != operation.DeclarationYes || logout.Agent.Mutation.Impact.Destructive != operation.DeclarationYes {
		t.Fatalf("logout impact = %+v", logout.Agent.Mutation.Impact)
	}
}
