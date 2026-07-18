package chatworkconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	SchemaVersion    = 1
	AuthMethodOAuth2 = "oauth2"
	FixedRedirectURI = "cwk://oauth/callback"

	applicationDirectory = "cwk"
	configurationFile    = "config.json"
	maxConfigBytes       = 4096
	maxClientIDBytes     = 512
)

var (
	// ErrConfigNotFound means no public OAuth configuration has been saved.
	ErrConfigNotFound = errors.New("Chatwork public configuration not found")
	// ErrConfigInvalid means configuration storage or content violated its
	// fixed, secret-free schema or filesystem safety contract.
	ErrConfigInvalid = errors.New("Chatwork public configuration is invalid")
	// ErrConfigUnavailable means the operating-system configuration directory
	// or its filesystem could not be accessed.
	ErrConfigUnavailable = errors.New("Chatwork public configuration is unavailable")
)

// PublicConfig is the complete non-secret, single-account authentication
// selection. It intentionally cannot represent credentials or token material.
type PublicConfig struct {
	AuthMethod  string
	ClientID    string
	RedirectURI string
}

// NewOAuthPublicConfig constructs the only supported persisted authentication
// selection. The fixed redirect is part of the executable contract rather than
// a caller-provided setting.
func NewOAuthPublicConfig(clientID string) (PublicConfig, error) {
	config := PublicConfig{
		AuthMethod:  AuthMethodOAuth2,
		ClientID:    clientID,
		RedirectURI: FixedRedirectURI,
	}
	if err := validate(config); err != nil {
		return PublicConfig{}, err
	}
	return config, nil
}

// FileStore persists public configuration below cwk's platform configuration
// directory. It does not read or write OAuth credentials.
type FileStore struct {
	userConfigDir func() (string, error)
}

// NewFileStore constructs the production store. Unix-like systems use the XDG
// configuration directory contract, including macOS; Windows uses AppData.
func NewFileStore() *FileStore {
	return &FileStore{userConfigDir: resolveUserConfigDir}
}

// NewFileStoreAt constructs a store rooted at a supplied user configuration
// directory. It exists for isolated tests and callers with an already-resolved
// platform directory.
func NewFileStoreAt(userConfigDir string) *FileStore {
	return &FileStore{userConfigDir: func() (string, error) { return userConfigDir, nil }}
}

// Load returns a validated public configuration. Callers can distinguish an
// unset configuration from invalid data and an unavailable filesystem with
// errors.Is and the exported sentinel errors.
func (s *FileStore) Load(ctx context.Context) (PublicConfig, error) {
	if err := contextError(ctx); err != nil {
		return PublicConfig{}, err
	}
	base, err := s.baseDirectory()
	if err != nil {
		return PublicConfig{}, err
	}
	appDirectory := filepath.Join(base, applicationDirectory)
	if _, err := validateDirectoryForLoad(base, false); err != nil {
		return PublicConfig{}, err
	}
	appInfo, err := validateDirectoryForLoad(appDirectory, true)
	if err != nil {
		return PublicConfig{}, err
	}
	root, err := os.OpenRoot(appDirectory)
	if err != nil {
		return PublicConfig{}, unavailable("application configuration directory open")
	}
	defer func() { _ = root.Close() }()
	openedDirectoryInfo, err := root.Stat(".")
	if err != nil {
		return PublicConfig{}, unavailable("application configuration directory metadata")
	}
	if !openedDirectoryInfo.IsDir() || !os.SameFile(appInfo, openedDirectoryInfo) {
		return PublicConfig{}, invalid("application configuration directory changed during read")
	}
	info, err := root.Lstat(configurationFile)
	if errors.Is(err, fs.ErrNotExist) {
		return PublicConfig{}, ErrConfigNotFound
	}
	if err != nil {
		return PublicConfig{}, unavailable("configuration metadata")
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || !restrictedPermissions(info.Mode()) {
		return PublicConfig{}, invalid("configuration file contract")
	}
	if info.Size() < 1 || info.Size() > maxConfigBytes {
		return PublicConfig{}, invalid("configuration size")
	}
	file, err := root.Open(configurationFile)
	if err != nil {
		return PublicConfig{}, unavailable("configuration read")
	}
	openedInfo, statErr := file.Stat()
	if statErr != nil {
		_ = file.Close()
		return PublicConfig{}, unavailable("configuration metadata")
	}
	if !openedInfo.Mode().IsRegular() || !restrictedPermissions(openedInfo.Mode()) || !os.SameFile(info, openedInfo) {
		_ = file.Close()
		return PublicConfig{}, invalid("configuration file changed during read")
	}
	contents, readErr := io.ReadAll(io.LimitReader(file, maxConfigBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return PublicConfig{}, unavailable("configuration read")
	}
	if err := contextError(ctx); err != nil {
		return PublicConfig{}, err
	}
	if len(contents) > maxConfigBytes {
		return PublicConfig{}, invalid("configuration size")
	}
	return decode(contents)
}

// Save validates and atomically replaces the public configuration. It never
// runs implicitly. First-login orchestration stores this non-secret selection
// before requesting a credential so a token is never usable without its exact
// public configuration; a failed consent leaves an inspectable unconfigured
// state that a later login can reuse.
func (s *FileStore) Save(ctx context.Context, config PublicConfig) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := validate(config); err != nil {
		return err
	}
	base, err := s.baseDirectory()
	if err != nil {
		return err
	}
	if err := ensureBaseDirectory(base); err != nil {
		return err
	}
	appDirectory := filepath.Join(base, applicationDirectory)
	if err := ensureApplicationDirectory(appDirectory); err != nil {
		return err
	}
	path := filepath.Join(appDirectory, configurationFile)
	if err := validateReplaceTarget(path); err != nil {
		return err
	}
	contents, err := encode(config)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(appDirectory, ".config.json.tmp-*")
	if err != nil {
		return unavailable("temporary configuration create")
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return unavailable("temporary configuration permissions")
	}
	if _, err := temporary.Write(contents); err != nil {
		return unavailable("temporary configuration write")
	}
	if err := temporary.Sync(); err != nil {
		return unavailable("temporary configuration sync")
	}
	if err := temporary.Close(); err != nil {
		return unavailable("temporary configuration close")
	}
	if err := contextError(ctx); err != nil {
		return err
	}
	// Repeat the target check immediately before replacement so a link or
	// special file observed during the write window is never accepted.
	if err := validateReplaceTarget(path); err != nil {
		return err
	}
	if err := atomicReplace(temporaryPath, path); err != nil {
		return unavailable("configuration replace")
	}
	committed = true
	return nil
}

type diskConfig struct {
	SchemaVersion int
	AuthMethod    string
	ClientID      string
	RedirectURI   string
}

func encode(config PublicConfig) ([]byte, error) {
	value := struct {
		SchemaVersion int    `json:"schema_version"`
		AuthMethod    string `json:"auth_method"`
		ClientID      string `json:"client_id"`
		RedirectURI   string `json:"redirect_uri"`
	}{
		SchemaVersion: SchemaVersion,
		AuthMethod:    config.AuthMethod,
		ClientID:      config.ClientID,
		RedirectURI:   config.RedirectURI,
	}
	contents, err := json.Marshal(value)
	if err != nil {
		return nil, invalid("configuration encoding")
	}
	contents = append(contents, '\n')
	if len(contents) > maxConfigBytes {
		return nil, invalid("configuration size")
	}
	return contents, nil
}

func decode(contents []byte) (PublicConfig, error) {
	decoder := json.NewDecoder(bytes.NewReader(contents))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return PublicConfig{}, invalid("configuration JSON")
	}
	seen := make(map[string]struct{}, 4)
	var stored diskConfig
	for decoder.More() {
		token, err := decoder.Token()
		name, ok := token.(string)
		if err != nil || !ok {
			return PublicConfig{}, invalid("configuration JSON")
		}
		if _, duplicate := seen[name]; duplicate {
			return PublicConfig{}, invalid("duplicate configuration field")
		}
		seen[name] = struct{}{}
		switch name {
		case "schema_version":
			err = decoder.Decode(&stored.SchemaVersion)
		case "auth_method":
			err = decoder.Decode(&stored.AuthMethod)
		case "client_id":
			err = decoder.Decode(&stored.ClientID)
		case "redirect_uri":
			err = decoder.Decode(&stored.RedirectURI)
		default:
			return PublicConfig{}, invalid("unknown configuration field")
		}
		if err != nil {
			return PublicConfig{}, invalid("configuration field type")
		}
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return PublicConfig{}, invalid("configuration JSON")
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return PublicConfig{}, invalid("trailing configuration data")
	}
	for _, required := range []string{"schema_version", "auth_method", "client_id", "redirect_uri"} {
		if _, ok := seen[required]; !ok {
			return PublicConfig{}, invalid("missing configuration field")
		}
	}
	if stored.SchemaVersion != SchemaVersion {
		return PublicConfig{}, invalid("configuration schema version")
	}
	config := PublicConfig{AuthMethod: stored.AuthMethod, ClientID: stored.ClientID, RedirectURI: stored.RedirectURI}
	if err := validate(config); err != nil {
		return PublicConfig{}, err
	}
	return config, nil
}

func validate(config PublicConfig) error {
	if config.AuthMethod != AuthMethodOAuth2 {
		return invalid("authentication method")
	}
	if config.RedirectURI != FixedRedirectURI {
		return invalid("OAuth redirect URI")
	}
	if config.ClientID == "" || len(config.ClientID) > maxClientIDBytes || unsafeText(config.ClientID) {
		return invalid("OAuth client ID")
	}
	return nil
}

func unsafeText(value string) bool {
	if !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return true
	}
	for _, character := range value {
		if unicode.Is(unicode.C, character) || character == '\u2028' || character == '\u2029' {
			return true
		}
	}
	return false
}

func (s *FileStore) baseDirectory() (string, error) {
	if s == nil || s.userConfigDir == nil {
		return "", unavailable("configuration directory resolver")
	}
	base, err := s.userConfigDir()
	if err != nil || base == "" || !filepath.IsAbs(base) {
		return "", unavailable("configuration directory resolution")
	}
	return filepath.Clean(base), nil
}

func validateDirectoryForLoad(path string, application bool) (fs.FileInfo, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrConfigNotFound
	}
	if err != nil {
		return nil, unavailable("configuration directory metadata")
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, invalid("configuration directory contract")
	}
	if application && !restrictedPermissions(info.Mode()) {
		return nil, invalid("configuration directory permissions")
	}
	return info, nil
}

func ensureBaseDirectory(path string) error {
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if err := os.MkdirAll(path, 0o700); err != nil {
			return unavailable("configuration base create")
		}
		info, err = os.Lstat(path)
	case err != nil:
		return unavailable("configuration base metadata")
	}
	if err != nil {
		return unavailable("configuration base metadata")
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return invalid("configuration base contract")
	}
	return nil
}

func ensureApplicationDirectory(path string) error {
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if err := os.Mkdir(path, 0o700); err != nil {
			return unavailable("application configuration directory create")
		}
		info, err = os.Lstat(path)
	case err != nil:
		return unavailable("application configuration directory metadata")
	}
	if err != nil {
		return unavailable("application configuration directory metadata")
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return invalid("application configuration directory contract")
	}
	if runtime.GOOS != "windows" {
		// #nosec G302 -- path is the application configuration directory;
		// directory mode 0700 is the least-privilege searchable mode.
		if err := os.Chmod(path, 0o700); err != nil {
			return unavailable("application configuration directory permissions")
		}
	}
	return nil
}

func validateReplaceTarget(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return unavailable("configuration target metadata")
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return invalid("configuration target contract")
	}
	return nil
}

func restrictedPermissions(mode fs.FileMode) bool {
	return runtime.GOOS == "windows" || mode.Perm()&0o077 == 0
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return context.Canceled
	}
	return ctx.Err()
}

func invalid(reason string) error {
	return fmt.Errorf("%w: %s", ErrConfigInvalid, reason)
}

func unavailable(reason string) error {
	return fmt.Errorf("%w: %s", ErrConfigUnavailable, reason)
}
