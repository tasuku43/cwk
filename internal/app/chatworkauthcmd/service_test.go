package chatworkauthcmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tasuku43/cwk/internal/domain/chatworkauth"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

type managerStub struct {
	loginStatus  CredentialStatus
	status       CredentialStatus
	loginErr     error
	statusErr    error
	logoutErr    error
	loginCalls   int
	statusCalls  int
	logoutCalls  int
	receivedURL  string
	callbackSeen string
	clientID     string
}

func (m *managerStub) Login(ctx context.Context, clientID string, receive RedirectReceiver) (CredentialStatus, error) {
	m.loginCalls++
	m.clientID = clientID
	if m.loginErr != nil {
		return CredentialStatus{}, m.loginErr
	}
	callback, err := receive(ctx, "https://www.chatwork.example/consent?request=transient")
	m.callbackSeen = callback
	if err != nil {
		return CredentialStatus{}, err
	}
	return m.loginStatus, nil
}

func (m *managerStub) Status(context.Context) (CredentialStatus, error) {
	m.statusCalls++
	return m.status, m.statusErr
}

func (m *managerStub) Logout(context.Context) error {
	m.logoutCalls++
	return m.logoutErr
}

type typedNilManager struct{}

func (*typedNilManager) Login(context.Context, string, RedirectReceiver) (CredentialStatus, error) {
	panic("typed nil manager called")
}
func (*typedNilManager) Status(context.Context) (CredentialStatus, error) {
	panic("typed nil manager called")
}
func (*typedNilManager) Logout(context.Context) error { panic("typed nil manager called") }

func assertFault(t *testing.T, err error, kind fault.Kind, code string) {
	t.Helper()
	public, ok := fault.PublicCopy(err)
	if !ok || public.Kind != kind || public.Code != code {
		t.Fatalf("fault = (%v, %v), want %s/%s", public, ok, kind, code)
	}
}

func TestLoginPassesPublicClientIDAndReceiverAndReturnsSecretFreeReadySummary(t *testing.T) {
	expires := time.Unix(1_800_000_000, 0).UTC()
	manager := &managerStub{loginStatus: CredentialStatus{Authenticated: true, ExpiresAt: expires}}
	summary, err := New(manager).Login(context.Background(), "public-client", func(_ context.Context, url string) (string, error) {
		manager.receivedURL = url
		return "cwk://oauth/callback?opaque=private", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if manager.loginCalls != 1 || manager.clientID != "public-client" || manager.receivedURL == "" || manager.callbackSeen == "" {
		t.Fatalf("manager calls/client/url/callback = %d/%q/%q/%q", manager.loginCalls, manager.clientID, manager.receivedURL, manager.callbackSeen)
	}
	if summary.State != chatworkauth.StateReady || summary.ExpiresAt != expires.Unix() || summary.Method != "oauth2" {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestStatusDistinguishesUnconfiguredExpiredAndReady(t *testing.T) {
	expires := time.Unix(1_800_000_000, 0).UTC()
	for _, test := range []struct {
		name   string
		status CredentialStatus
		state  chatworkauth.State
		expiry int64
	}{
		{name: "unconfigured", state: chatworkauth.StateUnconfigured},
		{name: "expired", status: CredentialStatus{ExpiresAt: expires}, state: chatworkauth.StateExpired, expiry: expires.Unix()},
		{name: "ready", status: CredentialStatus{Authenticated: true, ExpiresAt: expires}, state: chatworkauth.StateReady, expiry: expires.Unix()},
	} {
		t.Run(test.name, func(t *testing.T) {
			manager := &managerStub{status: test.status}
			summary, err := New(manager).Status(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if summary.State != test.state || summary.ExpiresAt != test.expiry {
				t.Fatalf("summary = %+v", summary)
			}
		})
	}
}

func TestLogoutAcknowledgesOnlyLocalRemoval(t *testing.T) {
	manager := &managerStub{}
	result, err := New(manager).Logout(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if manager.logoutCalls != 1 || !result.Acknowledged || result.RemoteRevocation {
		t.Fatalf("logout = calls %d, result %+v", manager.logoutCalls, result)
	}
}

func TestMissingPortsMakeZeroManagerCalls(t *testing.T) {
	manager := &managerStub{}
	service := New(manager)
	var nilManager *typedNilManager
	typedNilService := New(nilManager)
	if _, err := typedNilService.Status(context.Background()); err == nil {
		t.Fatal("typed nil manager succeeded")
	} else {
		assertFault(t, err, fault.KindUnavailable, "oauth_credential_store_unavailable")
	}
	if _, err := service.Login(context.Background(), "public-client", nil); err == nil {
		t.Fatal("nil receiver succeeded")
	} else {
		assertFault(t, err, fault.KindContract, "oauth_login_receiver_missing")
	}
}

func TestManagerFaultIsPreservedWithoutItsPrivateCause(t *testing.T) {
	private := errors.New("access-token-canary")
	manager := &managerStub{statusErr: fault.Wrap(fault.KindUnavailable, "oauth_credential_store_unavailable", "The OAuth credential store is unavailable.", true, private)}
	_, err := New(manager).Status(context.Background())
	assertFault(t, err, fault.KindUnavailable, "oauth_credential_store_unavailable")
}

func TestCanceledContextMakesZeroManagerCalls(t *testing.T) {
	manager := &managerStub{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := New(manager).Status(ctx)
	assertFault(t, err, fault.KindCanceled, "authentication_canceled")
	if manager.statusCalls != 0 {
		t.Fatal("canceled status reached manager")
	}
}
