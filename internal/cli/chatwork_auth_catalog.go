package cli

import (
	"github.com/tasuku43/cwk/internal/domain/chatworkauth"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
)

// Authentication lifecycle is outside the fixed 32-operation Chatwork API
// snapshot, so its capability deliberately does not claim an upstream REST
// operation mapping.
const chatworkAuthCapability = "authentication.manage"

// chatworkAuthCommandSpecs declares the OAuth lifecycle independently of the
// provider-task catalog. These commands establish or inspect authentication;
// requiring the authentication gate here would make login recovery circular.
func chatworkAuthCommandSpecs() []CommandSpec {
	return []CommandSpec{
		{
			Path: "auth login", Summary: "Authorize the single Chatwork account", Args: "[--client-id <public-client-id>]", Effect: operation.EffectCreate, Role: RoleAct,
			Agent: AgentContract{
				CapabilityID: chatworkAuthCapability,
				Outcome:      "Configure and authorize the single Chatwork account with state and PKCE S256, then store public selection separately from OS-protected tokens",
				Inputs: []CommandInput{
					{Name: "--client-id", Source: InputSourceFlag, Required: false, Description: "Registered public OAuth client ID; required only while local public configuration is absent.", AllowedValues: []string{}},
					{Name: "callback_url", Source: InputSourceStdin, Required: true, Description: "Paste the complete redirect URL once; authorization code and state never enter argv or output.", AllowedValues: []string{}},
				},
				Output: authOutput(
					OutputField{Name: "method", Type: OutputFieldTypeString, Description: "Established authentication method: oauth2."},
					OutputField{Name: "status", Type: OutputFieldTypeString, Description: "Credential state after validated exchange and storage: ready."},
					OutputField{Name: "expires_at", Type: OutputFieldTypeInteger, Description: "Provider-advertised access-token expiry as a Unix timestamp; never token material."},
				),
				Prerequisites: []string{
					"Register the exact cwk://oauth/callback redirect URI for the public Chatwork OAuth client.",
					"The command opens the consent URL in the default browser when available, otherwise writes it to stderr, then reads one complete callback URL from stdin.",
				},
				Errors:      authLoginErrors(),
				FixedTarget: authFixedTarget(),
				Mutation: &MutationContract{
					TargetKind: chatworkauth.TargetKind, TargetInputs: []string{},
					Impact: operation.Impact{Cardinality: operation.CardinalityOne, Notification: operation.DeclarationNo, AccessChange: operation.DeclarationYes, Destructive: operation.DeclarationNo},
				},
			},
		},
		{
			Path: "auth status", Summary: "Inspect single-account Chatwork authentication", Effect: operation.EffectRead, Role: RoleAct,
			Agent: AgentContract{
				CapabilityID: chatworkAuthCapability,
				Outcome:      "Inspect the fixed local authentication target without revealing, refreshing, or sending its credential",
				Inputs:       []CommandInput{},
				Output: authOutput(
					OutputField{Name: "method", Type: OutputFieldTypeString, Description: "Authentication method: oauth2."},
					OutputField{Name: "status", Type: OutputFieldTypeString, Description: "Secret-free state: unconfigured, ready, or expired."},
					OutputField{Name: "expires_at", Type: OutputFieldTypeInteger, Description: "Provider-advertised expiry as a Unix timestamp, or zero when unavailable."},
				),
				Prerequisites: []string{},
				Errors:        authStatusErrors(),
				FixedTarget:   authFixedTarget(),
			},
		},
		{
			Path: "auth logout", Summary: "Remove the stored Chatwork OAuth credential", Effect: operation.EffectWrite, Role: RoleAct,
			Agent: AgentContract{
				CapabilityID: chatworkAuthCapability,
				Outcome:      "Remove only the credential stored for the fixed local authentication target without claiming remote token revocation",
				Inputs:       []CommandInput{},
				Output: authOutput(
					OutputField{Name: "acknowledged", Type: OutputFieldTypeBoolean, Description: "Whether the operating-system credential store confirmed removal."},
					OutputField{Name: "remote_revocation", Type: OutputFieldTypeBoolean, Description: "Always false; Chatwork remote revocation is not claimed by this command."},
				),
				Prerequisites: []string{},
				Errors:        authLogoutErrors(),
				FixedTarget:   authFixedTarget(),
				Mutation: &MutationContract{
					TargetKind: chatworkauth.TargetKind, TargetInputs: []string{},
					Impact: operation.Impact{Cardinality: operation.CardinalityOne, Notification: operation.DeclarationNo, AccessChange: operation.DeclarationYes, Destructive: operation.DeclarationYes},
				},
			},
		},
	}
}

func authFixedTarget() *FixedTargetContract {
	return &FixedTargetContract{
		Scope: FixedTargetScopeToolLocal, Kind: chatworkauth.TargetKind,
		StableID:    chatworkauth.TargetStableID,
		Description: "The one Chatwork authentication state owned by this cwk installation.",
	}
}

func authOutput(output ...OutputField) CommandOutput {
	return CommandOutput{Formats: []OutputFormat{OutputFormatText}, DefaultFormat: OutputFormatText, Fields: output, Completeness: OutputCompletenessComplete}
}

func authStatusErrors() []CommandError {
	return authBaseErrors("auth status", false, true, true)
}

func authLoginErrors() []CommandError {
	path := "auth login"
	help := "help " + path
	errors := authBaseErrors(path, true, true, true)
	errors = append(errors,
		declaredCommandError(fault.KindInvalidInput, "oauth_client_configuration_missing", false, help, "Pass --client-id on first login."),
		declaredCommandError(fault.KindInvalidInput, "oauth_client_configuration_invalid", false, help, "Correct or remove the stored public-client configuration."),
		declaredCommandError(fault.KindUnavailable, "oauth_public_configuration_unavailable", true, help, "Restore access to the platform user configuration and inspect auth status."),
		declaredCommandError(fault.KindInvalidInput, "oauth_configuration_invalid", false, help, "Correct the fixed public-client OAuth configuration."),
		declaredCommandError(fault.KindInvalidInput, "oauth_callback_missing", false, help, "Paste one complete callback URL through stdin."),
		declaredCommandError(fault.KindInvalidInput, "oauth_redirect_invalid", false, help, "Paste the complete redirect URL without editing it."),
		declaredCommandError(fault.KindAuthentication, "oauth_redirect_mismatch", false, help, "Start a new login and use only its exact registered callback."),
		declaredCommandError(fault.KindAuthentication, "oauth_redirect_receive_failed", false, help, "Start a new login and paste one complete callback through stdin."),
		declaredCommandError(fault.KindAuthentication, "oauth_state_mismatch", false, help, "Start a new login; do not reuse the rejected callback."),
		declaredCommandError(fault.KindRejected, "oauth_authorization_denied", false, help, "Start a new login only when authorization is intended."),
		declaredCommandError(fault.KindAuthentication, "oauth_code_exchange_failed", false, help, "Start a new login with a new state and PKCE verifier."),
		declaredCommandError(fault.KindContract, "oauth_login_receiver_missing", false, help, "Repair the stdin callback receiver composition."),
		declaredCommandError(fault.KindInternal, "oauth_state_generation_failed", false, help, "Retry only after the secure random source is available."),
		declaredCommandError(fault.KindPermission, "insufficient_authentication_capability", false, help, "Authorize the documented Chatwork scopes."),
		declaredCommandError(fault.KindAuthentication, "authentication_expired", false, "auth status", "Inspect and then re-establish OAuth authentication."),
		declaredCommandError(fault.KindContract, "oauth_credential_too_large", false, "help auth status", "Review provider token growth against the fixed store bound."),
		declaredCommandError(fault.KindRejected, "oauth_credential_already_present", false, "auth status", "Inspect or explicitly log out the existing credential before replacement."),
		declaredCommandError(fault.KindContract, "oauth_identity_request_invalid", false, "help auth status", "Repair the fixed OAuth identity-verification request contract."),
		declaredCommandError(fault.KindUnavailable, "oauth_identity_verification_unavailable", true, "auth status", "Inspect authentication after Chatwork identity verification becomes available."),
		declaredCommandError(fault.KindContract, "oauth_identity_response_invalid", false, "help auth status", "Review Chatwork identity schema drift before using the credential."),
		declaredCommandError(fault.KindAuthentication, "oauth_identity_verification_failed", false, "auth status", "Inspect and then re-establish OAuth authentication."),
	)
	return appendAuthMutationErrors(errors, path, "auth status")
}

func authLogoutErrors() []CommandError {
	return appendAuthMutationErrors(authBaseErrors("auth logout", true, false, true), "auth logout", "auth status")
}

func authBaseErrors(path string, mutation, session, store bool) []CommandError {
	help := "help " + path
	retry := path
	if mutation {
		retry = help
	}
	errors := []CommandError{
		declaredCommandError(fault.KindInvalidInput, "invalid_arguments", false, help, "Correct the declared authentication task inputs."),
		declaredCommandError(fault.KindContract, "missing_authentication_context", false, help, "Repair the context-aware authentication invocation."),
		declaredCommandError(fault.KindCanceled, "authentication_canceled", false, retry, "Start a new authentication task when the caller is ready."),
		declaredCommandError(fault.KindContract, "output_contract_exceeded", false, help, "Repair the bounded authentication projection."),
		declaredCommandError(fault.KindContract, "output_encoding_failed", false, help, "Repair the authentication output projection."),
		declaredCommandError(fault.KindInternal, "output_write_failed", true, retry, "Retry with a writable output stream."),
		declaredCommandError(fault.KindCanceled, "operation_canceled", true, retry, "Retry when the caller is ready."),
	}
	if session {
		errors = append(errors, declaredCommandError(fault.KindAuthentication, "invalid_authentication_session", false, "auth status", "Inspect and then re-establish OAuth authentication."))
	}
	if store {
		errors = append(errors, declaredCommandError(fault.KindUnavailable, "oauth_credential_store_unavailable", true, "auth status", "Inspect authentication after operating-system credential access is restored."))
	}
	return errors
}

func appendAuthMutationErrors(errors []CommandError, path, reconcile string) []CommandError {
	help := "help " + path
	errors = append(errors,
		declaredCommandError(fault.KindContract, "invalid_mutation_contract", false, help, "Repair the authentication mutation target and impact declaration."),
		declaredCommandError(fault.KindContract, "missing_mutation_action", false, help, "Repair authentication mutation composition."),
		declaredCommandError(fault.KindRejected, "missing_mutation_policy", false, help, "Configure the reviewed authentication mutation policy."),
		declaredCommandError(fault.KindRejected, "mutation_rejected", false, help, "Revalidate the exact authentication target and policy."),
		declaredCommandError(fault.KindContract, "unclassified_mutation_outcome", false, reconcile, "Inspect stored authentication before another mutation."),
	)
	return errors
}
