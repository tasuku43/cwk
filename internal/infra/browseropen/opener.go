// Package browseropen owns the bounded platform handoff to the user's default
// browser. It accepts only Chatwork's fixed authorization destination.
package browseropen

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"unicode"
)

const (
	maxAuthorizationURLBytes = 8192
	fixedRedirectURI         = "cwk://oauth/callback"
	fixedScope               = "users.all:read rooms.all:read_write contacts.all:read_write"
)

var (
	ErrInvalidURL  = errors.New("browser authorization URL is invalid")
	ErrUnavailable = errors.New("platform browser opener is unavailable")
)

type launchFunc func(context.Context, string) error

// Opener validates the fixed destination before crossing the platform process
// or URI-activation boundary.
type Opener struct {
	launch launchFunc
}

func New() *Opener {
	return &Opener{launch: platformLaunch}
}

func newWithLauncher(launch launchFunc) *Opener {
	return &Opener{launch: launch}
}

func (o *Opener) Open(ctx context.Context, raw string) error {
	if ctx == nil {
		return context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateAuthorizationURL(raw); err != nil {
		return ErrInvalidURL
	}
	if o == nil || o.launch == nil {
		return ErrUnavailable
	}
	if err := o.launch(ctx, raw); err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return contextErr
		}
		return ErrUnavailable
	}
	return nil
}

func validateAuthorizationURL(raw string) error {
	if raw == "" || len(raw) > maxAuthorizationURLBytes {
		return ErrInvalidURL
	}
	for _, character := range raw {
		if character <= 0x20 || unicode.Is(unicode.C, character) || character == '\u2028' || character == '\u2029' {
			return ErrInvalidURL
		}
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "www.chatwork.com" ||
		parsed.Path != "/packages/oauth2/login.php" || parsed.RawQuery == "" ||
		parsed.RawPath != "" || parsed.Fragment != "" || parsed.RawFragment != "" || parsed.User != nil || parsed.Opaque != "" ||
		strings.Contains(raw, "#") {
		return ErrInvalidURL
	}
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil || len(query) != 7 || query.Encode() != parsed.RawQuery {
		return ErrInvalidURL
	}
	allowed := map[string]struct{}{
		"response_type": {}, "client_id": {}, "redirect_uri": {}, "scope": {},
		"state": {}, "code_challenge": {}, "code_challenge_method": {},
	}
	for _, component := range strings.Split(parsed.RawQuery, "&") {
		key, _, found := strings.Cut(component, "=")
		if !found {
			return ErrInvalidURL
		}
		if _, ok := allowed[key]; !ok {
			return ErrInvalidURL
		}
	}
	responseType, ok := exactlyOne(query, "response_type")
	if !ok || responseType != "code" {
		return ErrInvalidURL
	}
	clientID, ok := exactlyOne(query, "client_id")
	if !ok || !validASCIIValue(clientID, 1, 512, isClientIDCharacter) {
		return ErrInvalidURL
	}
	redirect, ok := exactlyOne(query, "redirect_uri")
	if !ok || !validPublicRedirect(redirect) {
		return ErrInvalidURL
	}
	scope, ok := exactlyOne(query, "scope")
	if !ok || !validScope(scope) {
		return ErrInvalidURL
	}
	state, ok := exactlyOne(query, "state")
	if !ok || !validASCIIValue(state, 32, 512, isUnreserved) {
		return ErrInvalidURL
	}
	challenge, ok := exactlyOne(query, "code_challenge")
	if !ok || !validASCIIValue(challenge, 43, 128, isUnreserved) {
		return ErrInvalidURL
	}
	method, ok := exactlyOne(query, "code_challenge_method")
	if !ok || method != "S256" {
		return ErrInvalidURL
	}
	return nil
}

func exactlyOne(query url.Values, key string) (string, bool) {
	values, ok := query[key]
	returnValue := ""
	if ok && len(values) == 1 {
		returnValue = values[0]
	}
	return returnValue, ok && len(values) == 1
}

func validPublicRedirect(raw string) bool {
	if raw != fixedRedirectURI || !safeDecodedText(raw) {
		return false
	}
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" &&
		parsed.RawQuery == "" && parsed.Fragment == "" && parsed.User == nil && parsed.Opaque == ""
}

func safeDecodedText(raw string) bool {
	for _, character := range raw {
		if unicode.IsSpace(character) || unicode.Is(unicode.C, character) || character == '\u2028' || character == '\u2029' {
			return false
		}
	}
	return true
}

func validScope(raw string) bool {
	if raw != fixedScope {
		return false
	}
	parts := strings.Fields(raw)
	if len(parts) == 0 || len(parts) > 64 || strings.Join(parts, " ") != raw {
		return false
	}
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		if part == "offline_access" || !validASCIIValue(part, 1, 128, isScopeCharacter) {
			return false
		}
		if _, exists := seen[part]; exists {
			return false
		}
		seen[part] = struct{}{}
	}
	return true
}

func validASCIIValue(raw string, minimum, maximum int, allowed func(byte) bool) bool {
	if len(raw) < minimum || len(raw) > maximum {
		return false
	}
	for index := 0; index < len(raw); index++ {
		if !allowed(raw[index]) {
			return false
		}
	}
	return true
}

func isUnreserved(character byte) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9' || character == '-' || character == '.' || character == '_' || character == '~'
}

func isClientIDCharacter(character byte) bool {
	return isUnreserved(character)
}

func isScopeCharacter(character byte) bool {
	return character >= 'a' && character <= 'z' || character >= '0' && character <= '9' ||
		character == '.' || character == '_' || character == ':'
}
