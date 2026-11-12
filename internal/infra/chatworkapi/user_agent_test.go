package chatworkapi

import (
	"strings"
	"testing"
)

func TestChatworkUserAgentAcceptsOnlyDevelopmentOrReleaseCompatibleVersions(t *testing.T) {
	maxLengthVersion := "1.2.3-" + strings.Repeat("a", maxUserAgentVersionBytes-len("1.2.3-"))
	tooLongVersion := maxLengthVersion + "a"
	valid := []string{"dev", "0.0.0", "1.2.3", "0.3.0-rc.1", "1.2.3-alpha-beta.4", "1.2.3-01alpha", maxLengthVersion}
	for _, version := range valid {
		t.Run(version, func(t *testing.T) {
			if got := chatworkUserAgent(version); got != "cwk/"+version {
				t.Fatalf("chatworkUserAgent(%q) = %q", version, got)
			}
		})
	}

	invalid := []string{
		"", "v1.2.3", "1.2", "1.2.3.4", "01.2.3", "1.02.3", "1.2.03",
		"1.2.3+build", "1.2.3-", "1.2.3-01", "1.2.3-alpha..1", "1.2.3-alpha.",
		"1.2.3/rc", "1.2.3-rc/1", "1.2.3-β", "1.2.3\nforged: value", "1.2.3\r\nX-Test: forged",
		"1.2.3\x00", "1.2.3\x1b", tooLongVersion,
	}
	for index, version := range invalid {
		t.Run("invalid_"+string(rune('a'+index)), func(t *testing.T) {
			if got := chatworkUserAgent(version); got != "cwk/dev" {
				t.Fatalf("chatworkUserAgent(%q) = %q, want safe fallback", version, got)
			}
		})
	}
}

func TestNewFromEnvironmentUsesValidatedVersionWithoutEchoingIt(t *testing.T) {
	t.Setenv(TokenEnvironment, syntheticToken)
	for _, test := range []struct {
		version string
		want    string
		invalid bool
	}{
		{version: "dev", want: "cwk/dev"},
		{version: "1.2.3-rc.1", want: "cwk/1.2.3-rc.1"},
		{version: "1.2.3\r\nX-Leak: value", want: "cwk/dev", invalid: true},
	} {
		client, err := NewFromEnvironment(test.version)
		if err != nil {
			t.Fatal(err)
		}
		if client.userAgent != test.want {
			t.Fatalf("user agent for %q = %q, want %q", test.version, client.userAgent, test.want)
		}
		if test.invalid && strings.Contains(client.userAgent, test.version) {
			t.Fatalf("invalid version leaked into user agent: %q", client.userAgent)
		}
	}
}
