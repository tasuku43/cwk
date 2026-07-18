package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/tasuku43/cwk/internal/app/chatworkauthcmd"
	"github.com/tasuku43/cwk/internal/infra/chatworkoauth"
)

var _ chatworkauthcmd.ManagerPort = (*chatworkoauth.Manager)(nil)

type authManagerStub struct {
	loginStatus chatworkauthcmd.CredentialStatus
	status      chatworkauthcmd.CredentialStatus
	loginErr    error
	statusErr   error
	logoutErr   error
	loginCalls  int
	statusCalls int
	logoutCalls int
	callback    string
	consentURL  string
}

func (m *authManagerStub) Login(ctx context.Context, receive chatworkauthcmd.RedirectReceiver) (chatworkauthcmd.CredentialStatus, error) {
	m.loginCalls++
	if m.loginErr != nil {
		return chatworkauthcmd.CredentialStatus{}, m.loginErr
	}
	consent := m.consentURL
	if consent == "" {
		consent = "https://www.chatwork.example/consent?opaque=transient"
	}
	callback, err := receive(ctx, consent)
	m.callback = callback
	if err != nil {
		return chatworkauthcmd.CredentialStatus{}, err
	}
	return m.loginStatus, nil
}

func (m *authManagerStub) Status(context.Context) (chatworkauthcmd.CredentialStatus, error) {
	m.statusCalls++
	return m.status, m.statusErr
}

func (m *authManagerStub) Logout(context.Context) error {
	m.logoutCalls++
	return m.logoutErr
}

func newAuthCLI(input string, manager *authManagerStub) (*CLI, *bytes.Buffer, *bytes.Buffer) {
	service := chatworkauthcmd.New(manager)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := newCLI(strings.NewReader(input), stdout, stderr, DefaultCatalog(), passingInspector("ready"))
	command.chatworkOAuth = service
	return command, stdout, stderr
}

func TestAuthHandlersAreAttachedFromCatalogSpecs(t *testing.T) {
	for _, spec := range withChatworkAuthHandlers(chatworkAuthCommandSpecs(), chatworkauthcmd.New(&authManagerStub{})) {
		if spec.handler == nil {
			t.Errorf("%s has no handler", spec.Path)
		}
	}
}

func TestAuthProfilesRendersDeterministicSecretFreeDiscovery(t *testing.T) {
	command, stdout, stderr := newAuthCLI("", &authManagerStub{})
	if code := command.RunContext(context.Background(), []string{"auth", "profiles"}); code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	want := "cwk-auth-profiles/1\n" +
		"profile_ref: cwk_chatwork_oauth_public_v1\n" +
		"method: oauth2\n" +
		"api_selector: CWK_AUTH_METHOD\n" +
		"allowed_api_methods: pat,oauth2\n" +
		"callback_model: authorization_code_pkce_s256_manual_callback\n" +
		"credential_storage: operating_system\n"
	if stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("stdout/stderr = %q / %q", stdout.String(), stderr.String())
	}
}

func TestAuthLoginUsesTransientConsentAndNeverRendersCallback(t *testing.T) {
	const callback = "cwk://oauth/callback?code=authorization-code-canary&state=state-canary"
	expires := time.Unix(1_800_000_000, 0).UTC()
	manager := &authManagerStub{loginStatus: chatworkauthcmd.CredentialStatus{Authenticated: true, ExpiresAt: expires}}
	command, stdout, stderr := newAuthCLI(callback+"\n", manager)
	args := []string{"auth", "login", "--profile", "cwk_chatwork_oauth_public_v1"}
	if code := command.RunContext(context.Background(), args); code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	if manager.loginCalls != 1 || manager.callback != callback {
		t.Fatalf("login calls/callback = %d/%q", manager.loginCalls, manager.callback)
	}
	want := "cwk-auth-profile/1\nprofile_ref: cwk_chatwork_oauth_public_v1\nmethod: oauth2\nstatus: ready\nexpires_at: 1800000000\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "authorization-code-canary") || strings.Contains(stderr.String(), "authorization-code-canary") || strings.Contains(stdout.String(), "state-canary") || strings.Contains(stderr.String(), "state-canary") {
		t.Fatal("callback code or state was rendered")
	}
	if !strings.Contains(stderr.String(), "authorization_url: https://www.chatwork.example/consent?opaque=transient\ncallback_url: ") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestAuthStatusAndLogoutRenderBoundedFacts(t *testing.T) {
	expires := time.Unix(1_800_000_000, 0).UTC()
	manager := &authManagerStub{status: chatworkauthcmd.CredentialStatus{ExpiresAt: expires}}
	command, stdout, stderr := newAuthCLI("", manager)
	profile := "cwk_chatwork_oauth_public_v1"
	if code := command.RunContext(context.Background(), []string{"auth", "status", "--profile=" + profile}); code != ExitOK {
		t.Fatalf("status exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "status: expired\nexpires_at: 1800000000\n") || manager.statusCalls != 1 {
		t.Fatalf("status stdout/calls = %q/%d", stdout.String(), manager.statusCalls)
	}

	command, stdout, stderr = newAuthCLI("", manager)
	if code := command.RunContext(context.Background(), []string{"auth", "logout", "--profile", profile}); code != ExitOK {
		t.Fatalf("logout exit = %d, stderr = %q", code, stderr.String())
	}
	want := "cwk-auth-logout/1\nprofile_ref: cwk_chatwork_oauth_public_v1\nacknowledged: true\nremote_revocation: false\n"
	if stdout.String() != want || manager.logoutCalls != 1 {
		t.Fatalf("logout stdout/calls = %q/%d", stdout.String(), manager.logoutCalls)
	}
}

func TestAuthInvalidReferenceMakesZeroManagerCalls(t *testing.T) {
	manager := &authManagerStub{}
	command, stdout, stderr := newAuthCLI("", manager)
	code := command.RunContext(context.Background(), []string{"auth", "logout", "--profile", "oauth2"})
	if code != ExitUsage || stdout.Len() != 0 || manager.logoutCalls != 0 {
		t.Fatalf("exit/stdout/calls = %d/%q/%d", code, stdout.String(), manager.logoutCalls)
	}
	if !strings.Contains(stderr.String(), "code: invalid_arguments") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestAuthLoginBoundsCallbackBeforeManagerCanUseIt(t *testing.T) {
	manager := &authManagerStub{loginStatus: chatworkauthcmd.CredentialStatus{Authenticated: true, ExpiresAt: time.Unix(1_800_000_000, 0)}}
	command, stdout, stderr := newAuthCLI(strings.Repeat("x", maxOAuthCallbackBytes+1)+"\n", manager)
	code := command.RunContext(context.Background(), []string{"auth", "login", "--profile", "cwk_chatwork_oauth_public_v1"})
	if code != ExitAuthentication || stdout.Len() != 0 || manager.loginCalls != 1 || manager.callback != "" {
		t.Fatalf("exit/stdout/calls/callback = %d/%q/%d/%q", code, stdout.String(), manager.loginCalls, manager.callback)
	}
	if strings.Contains(stderr.String(), strings.Repeat("x", 32)) || !strings.Contains(stderr.String(), "code: oauth_redirect_receive_failed") {
		t.Fatalf("stderr leaked callback or missed fault: %q", stderr.String())
	}
}

func TestAuthRawMutationFailureBecomesReadOnlyReconciliation(t *testing.T) {
	manager := &authManagerStub{loginErr: errors.New("token-and-code-canary")}
	command, stdout, stderr := newAuthCLI("", manager)
	code := command.RunContext(context.Background(), []string{"auth", "login", "--profile", "cwk_chatwork_oauth_public_v1"})
	if code != ExitContract || stdout.Len() != 0 || manager.loginCalls != 1 {
		t.Fatalf("exit/stdout/calls = %d/%q/%d", code, stdout.String(), manager.loginCalls)
	}
	if strings.Contains(stderr.String(), "token-and-code-canary") || !strings.Contains(stderr.String(), "code: unclassified_mutation_outcome") || !strings.Contains(stderr.String(), "cwk auth status") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestReadBoundedLineStopsAtOneLineWithoutReadingFollowingData(t *testing.T) {
	reader := strings.NewReader("first\nsecond\n")
	line, err := readBoundedLine(reader, 16)
	if err != nil || line != "first" {
		t.Fatalf("line/err = %q/%v", line, err)
	}
	rest, _ := io.ReadAll(reader)
	if string(rest) != "second\n" {
		t.Fatalf("reader consumed additional input: %q", rest)
	}
}
