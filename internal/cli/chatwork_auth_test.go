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
)

var _ chatworkauthcmd.ManagerPort = (*chatworkOAuthManager)(nil)

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
	clientID    string
}

type authBrowserStub struct {
	err   error
	calls int
	url   string
}

type releasableAuthReader struct {
	started  chan struct{}
	release  chan struct{}
	finished chan struct{}
}

func newReleasableAuthReader() *releasableAuthReader {
	return &releasableAuthReader{
		started: make(chan struct{}), release: make(chan struct{}), finished: make(chan struct{}),
	}
}

func (r *releasableAuthReader) Read([]byte) (int, error) {
	close(r.started)
	<-r.release
	close(r.finished)
	return 0, io.EOF
}

func (s *authBrowserStub) Open(_ context.Context, raw string) error {
	s.calls++
	s.url = raw
	return s.err
}

func (m *authManagerStub) Login(ctx context.Context, clientID string, receive chatworkauthcmd.RedirectReceiver) (chatworkauthcmd.CredentialStatus, error) {
	m.loginCalls++
	m.clientID = clientID
	if m.loginErr != nil {
		return chatworkauthcmd.CredentialStatus{}, m.loginErr
	}
	consent := m.consentURL
	if consent == "" {
		consent = "https://www.chatwork.com/packages/oauth2/login.php?opaque=transient"
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

func TestAuthLoginUsesTransientConsentAndNeverRendersCallback(t *testing.T) {
	const callback = "cwk://oauth/callback?code=authorization-code-canary&state=state-canary"
	expires := time.Unix(1_800_000_000, 0).UTC()
	manager := &authManagerStub{loginStatus: chatworkauthcmd.CredentialStatus{Authenticated: true, ExpiresAt: expires}}
	command, stdout, stderr := newAuthCLI(callback+"\n", manager)
	args := []string{"auth", "login", "--client-id", "public-client"}
	if code := command.RunContext(context.Background(), args); code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	if manager.loginCalls != 1 || manager.clientID != "public-client" || manager.callback != callback {
		t.Fatalf("login calls/client/callback = %d/%q/%q", manager.loginCalls, manager.clientID, manager.callback)
	}
	want := "cwk-auth/1\nmethod: oauth2\nstatus: ready\nexpires_at: 1800000000\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "authorization-code-canary") || strings.Contains(stderr.String(), "authorization-code-canary") || strings.Contains(stdout.String(), "state-canary") || strings.Contains(stderr.String(), "state-canary") {
		t.Fatal("callback code or state was rendered")
	}
	if !strings.Contains(stderr.String(), "authorization_url: https://www.chatwork.com/packages/oauth2/login.php?opaque=transient\ncallback_url: ") || strings.Contains(stderr.String(), "browser_opened") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestAuthLoginOpensBrowserAndDoesNotRequireAuthorizationURLCopy(t *testing.T) {
	callback := "cwk://oauth/callback?code=synthetic&state=synthetic"
	manager := &authManagerStub{loginStatus: chatworkauthcmd.CredentialStatus{Authenticated: true, ExpiresAt: time.Unix(1_800_000_000, 0)}}
	command, _, stderr := newAuthCLI(callback+"\n", manager)
	opener := &authBrowserStub{}
	command.authBrowser = opener
	if code := command.RunContext(context.Background(), []string{"auth", "login", "--client-id", "public-client"}); code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	if opener.calls != 1 || opener.url != manager.consentURL && !strings.Contains(opener.url, "www.chatwork.com/packages/oauth2/login.php") {
		t.Fatalf("opener calls/url = %d/%q", opener.calls, opener.url)
	}
	if strings.Contains(stderr.String(), "authorization_url:") || strings.Contains(stderr.String(), "browser_opened") || !strings.Contains(stderr.String(), "callback_url: ") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestAuthStatusAndLogoutRenderBoundedFacts(t *testing.T) {
	expires := time.Unix(1_800_000_000, 0).UTC()
	manager := &authManagerStub{status: chatworkauthcmd.CredentialStatus{ExpiresAt: expires}}
	command, stdout, stderr := newAuthCLI("", manager)
	if code := command.RunContext(context.Background(), []string{"auth", "status"}); code != ExitOK {
		t.Fatalf("status exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "status: expired\nexpires_at: 1800000000\n") || manager.statusCalls != 1 {
		t.Fatalf("status stdout/calls = %q/%d", stdout.String(), manager.statusCalls)
	}

	command, stdout, stderr = newAuthCLI("", manager)
	if code := command.RunContext(context.Background(), []string{"auth", "logout"}); code != ExitOK {
		t.Fatalf("logout exit = %d, stderr = %q", code, stderr.String())
	}
	want := "cwk-auth-logout/1\nacknowledged: true\nremote_revocation: false\n"
	if stdout.String() != want || manager.logoutCalls != 1 {
		t.Fatalf("logout stdout/calls = %q/%d", stdout.String(), manager.logoutCalls)
	}
}

func TestAuthInvalidClientIDMakesZeroManagerCalls(t *testing.T) {
	manager := &authManagerStub{}
	command, stdout, stderr := newAuthCLI("", manager)
	code := command.RunContext(context.Background(), []string{"auth", "login", "--client-id", " invalid"})
	if code != ExitUsage || stdout.Len() != 0 || manager.loginCalls != 0 {
		t.Fatalf("exit/stdout/calls = %d/%q/%d", code, stdout.String(), manager.loginCalls)
	}
	if !strings.Contains(stderr.String(), "code: invalid_arguments") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestAuthLoginBoundsCallbackBeforeManagerCanUseIt(t *testing.T) {
	manager := &authManagerStub{loginStatus: chatworkauthcmd.CredentialStatus{Authenticated: true, ExpiresAt: time.Unix(1_800_000_000, 0)}}
	command, stdout, stderr := newAuthCLI(strings.Repeat("x", maxOAuthCallbackBytes+1)+"\n", manager)
	code := command.RunContext(context.Background(), []string{"auth", "login", "--client-id", "public-client"})
	if code != ExitAuthentication || stdout.Len() != 0 || manager.loginCalls != 1 || manager.callback != "" {
		t.Fatalf("exit/stdout/calls/callback = %d/%q/%d/%q", code, stdout.String(), manager.loginCalls, manager.callback)
	}
	if strings.Contains(stderr.String(), strings.Repeat("x", 32)) || !strings.Contains(stderr.String(), "code: oauth_redirect_receive_failed") {
		t.Fatalf("stderr leaked callback or missed fault: %q", stderr.String())
	}
}

func TestAuthLoginCancellationUnblocksCallbackRead(t *testing.T) {
	reader := newReleasableAuthReader()
	manager := &authManagerStub{loginStatus: chatworkauthcmd.CredentialStatus{Authenticated: true, ExpiresAt: time.Unix(1_800_000_000, 0)}}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := newCLI(reader, stdout, stderr, DefaultCatalog(), passingInspector("ready"))
	command.chatworkOAuth = chatworkauthcmd.New(manager)
	command.authBrowser = &authBrowserStub{}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan int, 1)
	go func() {
		result <- command.RunContext(ctx, []string{"auth", "login", "--client-id", "public-client"})
	}()
	select {
	case <-reader.started:
	case <-time.After(time.Second):
		t.Fatal("callback read did not start")
	}
	cancel()
	select {
	case code := <-result:
		if code != ExitCanceled {
			t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not release the callback wait")
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "code: authentication_canceled") || strings.Contains(stderr.String(), "browser_opened") {
		t.Fatalf("stdout/stderr = %q/%q", stdout.String(), stderr.String())
	}
	close(reader.release)
	select {
	case <-reader.finished:
	case <-time.After(time.Second):
		t.Fatal("released callback reader worker did not finish")
	}
}

func TestAuthRawMutationFailureBecomesReadOnlyReconciliation(t *testing.T) {
	manager := &authManagerStub{loginErr: errors.New("token-and-code-canary")}
	command, stdout, stderr := newAuthCLI("", manager)
	code := command.RunContext(context.Background(), []string{"auth", "login", "--client-id", "public-client"})
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
