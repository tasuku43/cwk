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
	profileKind := chatworkauth.OAuthProfileReferenceKind
	profileInput := CommandInput{
		Name: "--profile", Source: InputSourceFlag, Required: true,
		Description: "Pass profile_ref from auth profiles unchanged.", AllowedValues: []string{}, ReferenceKind: profileKind,
	}
	profileOutput := OutputField{
		Name: "profile_ref", Type: OutputFieldTypeString,
		Description: "Exact OAuth profile reference accepted unchanged by login, status, and logout.", ReferenceKind: profileKind,
	}

	return []CommandSpec{
		{
			Path: "auth profiles", Summary: "Discover configured Chatwork authentication profiles", Effect: operation.EffectRead, Role: RoleDiscover,
			Agent: AgentContract{
				CapabilityID: chatworkAuthCapability,
				Outcome:      "Discover the one exact public-client OAuth profile reference and the deterministic PAT or OAuth2 task-selection rule",
				Inputs:       []CommandInput{},
				Output: authOutput(
					profileOutput,
					OutputField{Name: "method", Type: OutputFieldTypeString, Description: "Authentication method represented by this profile: oauth2."},
					OutputField{Name: "api_selector", Type: OutputFieldTypeString, Description: "Required API-task selector name: CWK_AUTH_METHOD."},
					OutputField{Name: "allowed_api_methods", Type: OutputFieldTypeArray, Description: "Exact supported selector values: pat and oauth2; no fallback is attempted."},
					OutputField{Name: "callback_model", Type: OutputFieldTypeString, Description: "Public-client authorization code with PKCE S256 and manual full-callback paste."},
					OutputField{Name: "credential_storage", Type: OutputFieldTypeString, Description: "Operating-system credential store; never a plaintext project file."},
				),
				Prerequisites: []string{},
				Errors:        authProfilesErrors(),
			},
		},
		{
			Path: "auth login", Summary: "Authorize and store the Chatwork OAuth profile", Args: "--profile <oauth-profile-ref>", Effect: operation.EffectCreate, Role: RoleAct,
			Agent: AgentContract{
				CapabilityID: chatworkAuthCapability,
				Outcome:      "Authorize the exact public-client profile with state and PKCE S256, then store only its tokens in the operating-system credential store",
				Inputs: []CommandInput{
					profileInput,
					{Name: "CWK_OAUTH_CLIENT_ID", Source: InputSourceEnvironment, Required: true, Description: "Registered public OAuth client ID; non-secret configuration.", AllowedValues: []string{}},
					{Name: "CWK_OAUTH_REDIRECT_URI", Source: InputSourceEnvironment, Required: true, Description: "Registered non-HTTP custom redirect URI; non-secret configuration.", AllowedValues: []string{}},
					{Name: "callback_url", Source: InputSourceStdin, Required: true, Description: "Paste the complete redirect URL once; authorization code and state never enter argv or output.", AllowedValues: []string{}},
				},
				Output: authOutput(
					profileOutput,
					OutputField{Name: "method", Type: OutputFieldTypeString, Description: "Established authentication method: oauth2."},
					OutputField{Name: "status", Type: OutputFieldTypeString, Description: "Credential state after validated exchange and storage: ready."},
					OutputField{Name: "expires_at", Type: OutputFieldTypeInteger, Description: "Provider-advertised access-token expiry as a Unix timestamp; never token material."},
				),
				Prerequisites: []string{
					"Register the exact non-HTTP redirect URI for the public Chatwork OAuth client.",
					"The command writes one transient authorization URL to stderr before reading one complete callback URL from stdin.",
				},
				Errors: authLoginErrors(),
				Mutation: &MutationContract{
					TargetKind: "chatwork-oauth-credential", TargetInputs: []string{"--profile"}, ParentInput: "--profile",
					Impact: operation.Impact{Cardinality: operation.CardinalityOne, Notification: operation.DeclarationNo, AccessChange: operation.DeclarationYes, Destructive: operation.DeclarationNo},
				},
			},
		},
		{
			Path: "auth status", Summary: "Inspect the stored Chatwork OAuth profile", Args: "--profile <oauth-profile-ref>", Effect: operation.EffectRead, Role: RoleAct,
			Agent: AgentContract{
				CapabilityID: chatworkAuthCapability,
				Outcome:      "Inspect one exact OAuth profile without revealing, refreshing, or sending its credential",
				Inputs:       []CommandInput{profileInput},
				Output: authOutput(
					profileOutput,
					OutputField{Name: "method", Type: OutputFieldTypeString, Description: "Profile method: oauth2."},
					OutputField{Name: "status", Type: OutputFieldTypeString, Description: "Secret-free state: unconfigured, ready, or expired."},
					OutputField{Name: "expires_at", Type: OutputFieldTypeInteger, Description: "Provider-advertised expiry as a Unix timestamp, or zero when unavailable."},
				),
				Prerequisites: []string{},
				Errors:        authStatusErrors(),
			},
		},
		{
			Path: "auth logout", Summary: "Remove the stored Chatwork OAuth credential", Args: "--profile <oauth-profile-ref>", Effect: operation.EffectWrite, Role: RoleAct,
			Agent: AgentContract{
				CapabilityID: chatworkAuthCapability,
				Outcome:      "Remove only the credential stored for one exact OAuth profile without claiming remote token revocation",
				Inputs:       []CommandInput{profileInput},
				Output: authOutput(
					profileOutput,
					OutputField{Name: "acknowledged", Type: OutputFieldTypeBoolean, Description: "Whether the operating-system credential store confirmed removal."},
					OutputField{Name: "remote_revocation", Type: OutputFieldTypeBoolean, Description: "Always false; Chatwork remote revocation is not claimed by this command."},
				),
				Prerequisites: []string{},
				Errors:        authLogoutErrors(),
				Mutation: &MutationContract{
					TargetKind: profileKind, TargetInputs: []string{"--profile"}, TargetIDInput: "--profile",
					Impact: operation.Impact{Cardinality: operation.CardinalityOne, Notification: operation.DeclarationNo, AccessChange: operation.DeclarationYes, Destructive: operation.DeclarationYes},
				},
			},
		},
	}
}

func authOutput(output ...OutputField) CommandOutput {
	return CommandOutput{Formats: []OutputFormat{OutputFormatText}, DefaultFormat: OutputFormatText, Fields: output, Completeness: OutputCompletenessComplete}
}

func authProfilesErrors() []CommandError {
	return authBaseErrors("auth profiles", false, false, false, false)
}

func authStatusErrors() []CommandError {
	return authBaseErrors("auth status", false, true, true, true)
}

func authLoginErrors() []CommandError {
	path := "auth login"
	help := "help " + path
	errors := authBaseErrors(path, true, true, true, true)
	errors = append(errors,
		declaredCommandError(fault.KindInvalidInput, "oauth_client_configuration_missing", false, help, "Set the documented non-secret client ID and redirect URI configuration."),
		declaredCommandError(fault.KindInvalidInput, "oauth_client_configuration_invalid", false, help, "Correct the documented public-client ID and redirect URI configuration."),
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
		declaredCommandError(fault.KindAuthentication, "authentication_expired", false, "auth status", "Inspect and then re-establish the exact OAuth profile."),
		declaredCommandError(fault.KindContract, "oauth_credential_too_large", false, "help auth status", "Review provider token growth against the fixed store bound."),
		declaredCommandError(fault.KindRejected, "oauth_credential_already_present", false, "auth status", "Inspect or explicitly log out the existing profile before replacement."),
		declaredCommandError(fault.KindContract, "oauth_identity_request_invalid", false, "help auth status", "Repair the fixed OAuth identity-verification request contract."),
		declaredCommandError(fault.KindUnavailable, "oauth_identity_verification_unavailable", true, "auth status", "Inspect the profile after Chatwork identity verification becomes available."),
		declaredCommandError(fault.KindContract, "oauth_identity_response_invalid", false, "help auth status", "Review Chatwork identity schema drift before using the credential."),
		declaredCommandError(fault.KindAuthentication, "oauth_identity_verification_failed", false, "auth status", "Inspect and then re-establish the exact OAuth profile."),
	)
	return appendAuthMutationErrors(errors, path, "auth status")
}

func authLogoutErrors() []CommandError {
	return appendAuthMutationErrors(authBaseErrors("auth logout", true, true, false, true), "auth logout", "auth status")
}

func authBaseErrors(path string, mutation, profile, session, store bool) []CommandError {
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
	if profile {
		errors = append(errors, declaredCommandError(fault.KindInvalidInput, "oauth_profile_reference_invalid", false, "auth profiles", "Discover and reuse the exact supported profile reference."))
	}
	if session {
		errors = append(errors, declaredCommandError(fault.KindAuthentication, "invalid_authentication_session", false, "auth status", "Inspect and then re-establish the exact OAuth profile."))
	}
	if store {
		errors = append(errors, declaredCommandError(fault.KindUnavailable, "oauth_credential_store_unavailable", true, "auth status", "Inspect the profile after operating-system credential access is restored."))
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
		declaredCommandError(fault.KindContract, "unclassified_mutation_outcome", false, reconcile, "Inspect the stored profile before another authentication mutation."),
	)
	return errors
}
