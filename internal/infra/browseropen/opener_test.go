package browseropen

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

var syntheticAuthorizationURL = authorizationURL(nil)

func authorizationURL(change func(url.Values)) string {
	query := url.Values{
		"response_type":         {"code"},
		"client_id":             {"public-client"},
		"redirect_uri":          {"cwk://oauth/callback"},
		"scope":                 {"users.all:read rooms.all:read_write contacts.all:read_write"},
		"state":                 {strings.Repeat("s", 43)},
		"code_challenge":        {strings.Repeat("c", 43)},
		"code_challenge_method": {"S256"},
	}
	if change != nil {
		change(query)
	}
	return "https://www.chatwork.com/packages/oauth2/login.php?" + query.Encode()
}

func TestOpenPassesOnlyValidatedChatworkAuthorizationURL(t *testing.T) {
	var calls int
	var received string
	opener := newWithLauncher(func(_ context.Context, raw string) error {
		calls++
		received = raw
		return nil
	})
	if err := opener.Open(context.Background(), syntheticAuthorizationURL); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || received != syntheticAuthorizationURL {
		t.Fatalf("calls/url = %d/%q", calls, received)
	}
}

func TestValidatorAcceptsOAuthLibraryAuthorizationURL(t *testing.T) {
	config := oauth2.Config{
		ClientID: "public-client", RedirectURL: "cwk://oauth/callback",
		Scopes:   []string{"users.all:read", "rooms.all:read_write", "contacts.all:read_write"},
		Endpoint: oauth2.Endpoint{AuthURL: "https://www.chatwork.com/packages/oauth2/login.php"},
	}
	verifier := strings.Repeat("v", 43)
	raw := config.AuthCodeURL(strings.Repeat("s", 43), oauth2.S256ChallengeOption(verifier))
	if err := validateAuthorizationURL(raw); err != nil {
		t.Fatalf("oauth2.Config authorization URL rejected: %v", err)
	}
}

func TestInvalidAuthorizationURLMakesZeroLaunchCalls(t *testing.T) {
	var calls int
	opener := newWithLauncher(func(context.Context, string) error {
		calls++
		return nil
	})
	invalid := []string{
		"", "http://www.chatwork.com/packages/oauth2/login.php?state=x",
		strings.Replace(syntheticAuthorizationURL, "www.chatwork.com", "example.com", 1),
		strings.Replace(syntheticAuthorizationURL, "/packages/oauth2/login.php", "/other", 1),
		strings.Replace(syntheticAuthorizationURL, "https://", "https://user@", 1),
		syntheticAuthorizationURL + "#fragment",
		strings.Replace(syntheticAuthorizationURL, "login.php", "%6cogin.php", 1),
		strings.Replace(syntheticAuthorizationURL, "login.php?", "login.php\n?", 1),
		strings.Replace(syntheticAuthorizationURL, "public-client", "public client", 1),
		strings.Replace(syntheticAuthorizationURL, strings.Repeat("s", 43), strings.Repeat("s", 31), 1),
		strings.Replace(syntheticAuthorizationURL, strings.Repeat("c", 43), "short", 1),
		authorizationURL(func(query url.Values) { query.Set("response_type", "token") }),
		authorizationURL(func(query url.Values) { query.Set("redirect_uri", "http://localhost/callback") }),
		authorizationURL(func(query url.Values) { query.Set("redirect_uri", "other://oauth/callback") }),
		authorizationURL(func(query url.Values) { query.Set("scope", "offline_access") }),
		authorizationURL(func(query url.Values) { query.Set("scope", "users.all:read") }),
		authorizationURL(func(query url.Values) { query.Set("code_challenge_method", "plain") }),
		authorizationURL(func(query url.Values) { query.Del("state") }),
		authorizationURL(func(query url.Values) { query.Add("state", strings.Repeat("x", 43)) }),
		authorizationURL(func(query url.Values) { query.Set("unexpected", "value") }),
		strings.Replace(syntheticAuthorizationURL, "state=", "%73tate=", 1),
		strings.Replace(syntheticAuthorizationURL, "state=", "state=%ZZ", 1),
		strings.Replace(syntheticAuthorizationURL, "state=", "state=%E2%80%8B", 1),
		strings.Repeat("x", maxAuthorizationURLBytes+1),
	}
	for _, raw := range invalid {
		if err := opener.Open(context.Background(), raw); !errors.Is(err, ErrInvalidURL) {
			t.Errorf("Open(%q) error = %v", raw, err)
		}
	}
	if calls != 0 {
		t.Fatalf("invalid URLs launched %d times", calls)
	}
}

func TestOpenCancellationAndFailureAreRedacted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := newWithLauncher(func(context.Context, string) error { return nil }).Open(ctx, syntheticAuthorizationURL); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled error = %v", err)
	}
	deadlineCtx, deadlineCancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer deadlineCancel()
	if err := newWithLauncher(func(context.Context, string) error { return nil }).Open(deadlineCtx, syntheticAuthorizationURL); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline error = %v", err)
	}
	if err := newWithLauncher(func(context.Context, string) error { return nil }).Open(nil, syntheticAuthorizationURL); !errors.Is(err, context.Canceled) {
		t.Fatalf("nil context error = %v", err)
	}
	opener := newWithLauncher(func(context.Context, string) error {
		return errors.New("state-canary provider-url-canary")
	})
	err := opener.Open(context.Background(), syntheticAuthorizationURL)
	if !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), "state-canary") || strings.Contains(err.Error(), "provider-url-canary") {
		t.Fatalf("unavailable error = %v", err)
	}
}

func TestConfirmedLaunchIsNotOverwrittenByLaterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	opener := newWithLauncher(func(context.Context, string) error {
		cancel()
		return nil
	})
	if err := opener.Open(ctx, syntheticAuthorizationURL); err != nil {
		t.Fatalf("confirmed browser handoff was overwritten: %v", err)
	}
}
