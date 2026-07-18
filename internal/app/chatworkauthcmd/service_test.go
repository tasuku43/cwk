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
}

func (m *managerStub) Login(ctx context.Context, receive RedirectReceiver) (CredentialStatus, error) {
	m.loginCalls++
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

func (*typedNilManager) Login(context.Context, RedirectReceiver) (CredentialStatus, error) {
	panic("typed nil manager called")
}
func (*typedNilManager) Status(context.Context) (CredentialStatus, error) {
	panic("typed nil manager called")
}
func (*typedNilManager) Logout(context.Context) error { panic("typed nil manager called") }

func exactRef(t *testing.T) chatworkauth.ProfileReference {
	t.Helper()
	ref, err := chatworkauth.NewProfileReference(chatworkauth.PublicClientProfileReference)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func assertFault(t *testing.T, err error, kind fault.Kind, code string) {
	t.Helper()
	public, ok := fault.PublicCopy(err)
	if !ok || public.Kind != kind || public.Code != code {
		t.Fatalf("fault = (%v, %v), want %s/%s", public, ok, kind, code)
	}
}

func TestProfilesIsDeterministicAndDoesNotTouchManager(t *testing.T) {
	manager := &managerStub{}
	profiles, err := New(manager).Profiles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || profiles[0].Ref.Value() != chatworkauth.PublicClientProfileReference || profiles[0].Method != "oauth2" {
		t.Fatalf("profiles = %+v", profiles)
	}
	if manager.loginCalls+manager.statusCalls+manager.logoutCalls != 0 {
		t.Fatal("profile discovery accessed the private manager")
	}
}

func TestLoginPassesReceiverAndReturnsSecretFreeReadyProfile(t *testing.T) {
	expires := time.Unix(1_800_000_000, 0).UTC()
	manager := &managerStub{loginStatus: CredentialStatus{Authenticated: true, ExpiresAt: expires}}
	profile, err := New(manager).Login(context.Background(), exactRef(t), func(_ context.Context, url string) (string, error) {
		manager.receivedURL = url
		return "cwk://oauth/callback?opaque=private", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if manager.loginCalls != 1 || manager.receivedURL == "" || manager.callbackSeen == "" {
		t.Fatalf("manager calls/url/callback = %d/%q/%q", manager.loginCalls, manager.receivedURL, manager.callbackSeen)
	}
	if profile.State != chatworkauth.ProfileStateReady || profile.ExpiresAt != expires.Unix() || profile.Method != "oauth2" {
		t.Fatalf("profile = %+v", profile)
	}
}

func TestStatusDistinguishesUnconfiguredExpiredAndReady(t *testing.T) {
	expires := time.Unix(1_800_000_000, 0).UTC()
	for _, test := range []struct {
		name   string
		status CredentialStatus
		state  chatworkauth.ProfileState
		expiry int64
	}{
		{name: "unconfigured", state: chatworkauth.ProfileStateUnconfigured},
		{name: "expired", status: CredentialStatus{ExpiresAt: expires}, state: chatworkauth.ProfileStateExpired, expiry: expires.Unix()},
		{name: "ready", status: CredentialStatus{Authenticated: true, ExpiresAt: expires}, state: chatworkauth.ProfileStateReady, expiry: expires.Unix()},
	} {
		t.Run(test.name, func(t *testing.T) {
			manager := &managerStub{status: test.status}
			profile, err := New(manager).Status(context.Background(), exactRef(t))
			if err != nil {
				t.Fatal(err)
			}
			if profile.State != test.state || profile.ExpiresAt != test.expiry {
				t.Fatalf("profile = %+v", profile)
			}
		})
	}
}

func TestLogoutAcknowledgesOnlyLocalRemoval(t *testing.T) {
	manager := &managerStub{}
	result, err := New(manager).Logout(context.Background(), exactRef(t))
	if err != nil {
		t.Fatal(err)
	}
	if manager.logoutCalls != 1 || !result.Acknowledged || result.RemoteRevocation {
		t.Fatalf("logout = calls %d, result %+v", manager.logoutCalls, result)
	}
}

func TestInvalidInputsAndMissingPortsMakeZeroManagerCalls(t *testing.T) {
	manager := &managerStub{}
	service := New(manager)
	if _, err := service.Status(context.Background(), chatworkauth.ProfileReference{}); err == nil {
		t.Fatal("invalid reference succeeded")
	} else {
		assertFault(t, err, fault.KindInvalidInput, "oauth_profile_reference_invalid")
	}
	if manager.statusCalls != 0 {
		t.Fatal("invalid reference reached manager")
	}

	var nilManager *typedNilManager
	typedNilService := New(nilManager)
	if _, err := typedNilService.Status(context.Background(), exactRef(t)); err == nil {
		t.Fatal("typed nil manager succeeded")
	} else {
		assertFault(t, err, fault.KindUnavailable, "oauth_credential_store_unavailable")
	}
	if _, err := service.Login(context.Background(), exactRef(t), nil); err == nil {
		t.Fatal("nil receiver succeeded")
	} else {
		assertFault(t, err, fault.KindContract, "oauth_login_receiver_missing")
	}
}

func TestManagerFaultIsPreservedWithoutItsPrivateCause(t *testing.T) {
	private := errors.New("access-token-canary")
	manager := &managerStub{statusErr: fault.Wrap(fault.KindUnavailable, "oauth_credential_store_unavailable", "The OAuth credential store is unavailable.", true, private)}
	_, err := New(manager).Status(context.Background(), exactRef(t))
	assertFault(t, err, fault.KindUnavailable, "oauth_credential_store_unavailable")
}

func TestCanceledContextMakesZeroManagerCalls(t *testing.T) {
	manager := &managerStub{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := New(manager).Status(ctx, exactRef(t))
	assertFault(t, err, fault.KindCanceled, "authentication_canceled")
	if manager.statusCalls != 0 {
		t.Fatal("canceled status reached manager")
	}
}
