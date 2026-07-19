// Package commandconfig persists the non-secret command-selection profile in
// cwk's platform configuration directory.
package commandconfig

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
	"unicode/utf8"

	"github.com/tasuku43/cwk/internal/domain/commandselection"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

const (
	SchemaVersion = 1

	applicationDirectory = "cwk"
	configurationFile    = "command-selection.json"
	maxConfigBytes       = 512 * 1024
)

var (
	// ErrInvalid identifies content that violates the serialized
	// command-configuration contract and may be repaired by an
	// explicit replacement. Public errors contain only stable text.
	ErrInvalid = errors.New("command selection configuration is invalid")
	// ErrUnsafe identifies a filesystem object or mode that this adapter will
	// not read or replace. It requires external filesystem repair rather than
	// an in-tool content replacement.
	ErrUnsafe = errors.New("command selection configuration storage is unsafe")
	// ErrUnavailable identifies a configuration directory or filesystem that
	// could not be accessed.
	ErrUnavailable = errors.New("command selection configuration is unavailable")
)

// FileStore persists one explicit command-selection profile. The file stores
// no credentials and cannot represent provider or user data.
type FileStore struct {
	userConfigDir func() (string, error)
}

// NewFileStore constructs the production store. Unix-like systems, including
// macOS, use XDG semantics; Windows uses the roaming AppData directory.
func NewFileStore() *FileStore {
	return &FileStore{userConfigDir: resolveUserConfigDir}
}

// NewFileStoreAt constructs an isolated store below an already-resolved
// absolute user configuration directory.
func NewFileStoreAt(userConfigDir string) *FileStore {
	return &FileStore{userConfigDir: func() (string, error) { return userConfigDir, nil }}
}

// Load returns a profile and whether a file was present. A missing base,
// application directory, or file is the unconfigured state, not an error.
func (s *FileStore) Load(ctx context.Context) (commandselection.Profile, bool, error) {
	if err := contextFault(ctx); err != nil {
		return commandselection.Profile{}, false, err
	}
	base, err := s.baseDirectory()
	if err != nil {
		return commandselection.Profile{}, false, err
	}
	baseInfo, present, err := inspectDirectory(base, false)
	if err != nil || !present {
		return commandselection.Profile{}, false, err
	}
	_ = baseInfo

	appDirectory := filepath.Join(base, applicationDirectory)
	appInfo, present, err := inspectDirectory(appDirectory, true)
	if err != nil || !present {
		return commandselection.Profile{}, false, err
	}
	root, err := openVerifiedRoot(appDirectory, appInfo)
	if err != nil {
		return commandselection.Profile{}, false, err
	}
	defer func() { _ = root.Close() }()

	storedInfo, err := root.Lstat(configurationFile)
	if errors.Is(err, fs.ErrNotExist) {
		return commandselection.Profile{}, false, nil
	}
	if err != nil {
		return commandselection.Profile{}, false, unavailable("configuration metadata")
	}
	if !validConfigurationFile(storedInfo) {
		return commandselection.Profile{}, false, unsafeStorage("configuration file contract")
	}
	if storedInfo.Size() < 1 || storedInfo.Size() > maxConfigBytes {
		return commandselection.Profile{}, false, invalid("configuration size")
	}

	file, err := root.Open(configurationFile)
	if err != nil {
		return commandselection.Profile{}, false, unavailable("configuration read")
	}
	openedInfo, statErr := file.Stat()
	if statErr != nil {
		_ = file.Close()
		return commandselection.Profile{}, false, unavailable("configuration metadata")
	}
	if !validConfigurationFile(openedInfo) || !os.SameFile(storedInfo, openedInfo) {
		_ = file.Close()
		return commandselection.Profile{}, false, unsafeStorage("configuration file changed during read")
	}
	contents, readErr := io.ReadAll(io.LimitReader(file, maxConfigBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return commandselection.Profile{}, false, unavailable("configuration read")
	}
	if err := contextFault(ctx); err != nil {
		return commandselection.Profile{}, false, err
	}
	if len(contents) > maxConfigBytes {
		return commandselection.Profile{}, false, invalid("configuration size")
	}
	profile, err := decode(contents)
	if err != nil {
		return commandselection.Profile{}, false, err
	}
	return profile, true, nil
}

// Save validates and replaces the explicit profile from a same-directory
// temporary file. Unix uses rename plus directory sync; Windows requests
// replace-existing but the portable API does not guarantee atomicity or
// durability there. All failures before replacement are structured and safe
// to present. A replacement or durability error is intentionally returned raw:
// once replacement is attempted, only execution.Invoker may classify the
// outcome without incorrectly promising that the previous profile remains
// active.
func (s *FileStore) Save(ctx context.Context, profile commandselection.Profile) error {
	if err := contextFault(ctx); err != nil {
		return err
	}
	if err := profile.Validate(); err != nil {
		return invalidWithCause("profile", err)
	}
	contents, err := encode(profile)
	if err != nil {
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
	appInfo, err := ensureApplicationDirectory(appDirectory)
	if err != nil {
		return err
	}
	root, err := openVerifiedRoot(appDirectory, appInfo)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	if err := validateReplaceTarget(root); err != nil {
		return err
	}

	temporary, err := os.CreateTemp(appDirectory, ".command-selection.json.tmp-*")
	if err != nil {
		return unavailable("temporary configuration create")
	}
	temporaryName := filepath.Base(temporary.Name())
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = root.Remove(temporaryName)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return unavailable("temporary configuration permissions")
	}
	temporaryInfo, statErr := temporary.Stat()
	rootTemporaryInfo, rootStatErr := root.Lstat(temporaryName)
	if statErr != nil || rootStatErr != nil || !validConfigurationFile(temporaryInfo) ||
		!validConfigurationFile(rootTemporaryInfo) || !os.SameFile(temporaryInfo, rootTemporaryInfo) {
		return unsafeStorage("temporary configuration contract")
	}
	if written, err := temporary.Write(contents); err != nil || written != len(contents) {
		return unavailable("temporary configuration write")
	}
	if err := temporary.Sync(); err != nil {
		return unavailable("temporary configuration sync")
	}
	if err := temporary.Close(); err != nil {
		return unavailable("temporary configuration close")
	}
	if err := contextFault(ctx); err != nil {
		return err
	}

	// Verify both directory identity and target shape again immediately before
	// replacement. Root.Rename remains confined to the opened directory even if
	// its path is renamed after this check.
	currentAppInfo, present, err := inspectDirectory(appDirectory, true)
	if err != nil {
		return err
	}
	if !present || !os.SameFile(appInfo, currentAppInfo) {
		return unsafeStorage("application configuration directory changed during save")
	}
	if err := validateReplaceTarget(root); err != nil {
		return err
	}
	if err := root.Rename(temporaryName, configurationFile); err != nil {
		return err
	}
	committed = true
	if err := syncDirectory(root); err != nil {
		return err
	}
	return nil
}

func encode(profile commandselection.Profile) ([]byte, error) {
	value := struct {
		SchemaVersion   int      `json:"schema_version"`
		EnabledCommands []string `json:"enabled_commands"`
	}{
		SchemaVersion:   SchemaVersion,
		EnabledCommands: profile.EnabledCommands(),
	}
	if value.EnabledCommands == nil {
		value.EnabledCommands = []string{}
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

func decode(contents []byte) (commandselection.Profile, error) {
	if !utf8.Valid(contents) {
		return commandselection.Profile{}, invalid("configuration encoding")
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return commandselection.Profile{}, invalid("configuration JSON")
	}
	seen := make(map[string]struct{}, 2)
	schemaVersion := 0
	var enabled []string
	for decoder.More() {
		token, err := decoder.Token()
		name, ok := token.(string)
		if err != nil || !ok {
			return commandselection.Profile{}, invalid("configuration JSON")
		}
		if _, duplicate := seen[name]; duplicate {
			return commandselection.Profile{}, invalid("duplicate configuration field")
		}
		seen[name] = struct{}{}
		switch name {
		case "schema_version":
			if err := decoder.Decode(&schemaVersion); err != nil {
				return commandselection.Profile{}, invalid("configuration field type")
			}
		case "enabled_commands":
			arrayStart, err := decoder.Token()
			if err != nil || arrayStart != json.Delim('[') {
				return commandselection.Profile{}, invalid("configuration field type")
			}
			for decoder.More() {
				var path string
				if err := decoder.Decode(&path); err != nil {
					return commandselection.Profile{}, invalid("configuration field type")
				}
				enabled = append(enabled, path)
				if len(enabled) > commandselection.MaxEnabledCommands {
					return commandselection.Profile{}, invalid("enabled command count")
				}
			}
			arrayEnd, err := decoder.Token()
			if err != nil || arrayEnd != json.Delim(']') {
				return commandselection.Profile{}, invalid("configuration field type")
			}
		default:
			return commandselection.Profile{}, invalid("unknown configuration field")
		}
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return commandselection.Profile{}, invalid("configuration JSON")
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return commandselection.Profile{}, invalid("trailing configuration data")
	}
	for _, required := range []string{"schema_version", "enabled_commands"} {
		if _, ok := seen[required]; !ok {
			return commandselection.Profile{}, invalid("missing configuration field")
		}
	}
	if schemaVersion != SchemaVersion {
		return commandselection.Profile{}, invalid("configuration schema version")
	}
	profile, err := commandselection.New(enabled)
	if err != nil {
		return commandselection.Profile{}, invalidWithCause("enabled commands", err)
	}
	return profile, nil
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

func inspectDirectory(path string, restricted bool) (fs.FileInfo, bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, unavailable("configuration directory metadata")
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, false, unsafeStorage("configuration directory contract")
	}
	if restricted && !validApplicationDirectory(info) {
		return nil, false, unsafeStorage("application configuration directory permissions")
	}
	return info, true, nil
}

func ensureBaseDirectory(path string) error {
	_, present, err := inspectDirectory(path, false)
	if err != nil {
		return err
	}
	if present {
		return nil
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return unavailable("configuration base create")
	}
	_, present, err = inspectDirectory(path, false)
	if err != nil {
		return err
	}
	if !present {
		return unavailable("configuration base create")
	}
	return nil
}

func ensureApplicationDirectory(path string) (fs.FileInfo, error) {
	info, present, err := inspectDirectory(path, true)
	if err != nil {
		return nil, err
	}
	if present {
		return info, nil
	}
	if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, unavailable("application configuration directory create")
	}
	info, present, err = inspectDirectory(path, true)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, unavailable("application configuration directory create")
	}
	return info, nil
}

func openVerifiedRoot(path string, expected fs.FileInfo) (*os.Root, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, unavailable("application configuration directory open")
	}
	openedInfo, statErr := root.Stat(".")
	if statErr != nil || !openedInfo.IsDir() || !os.SameFile(expected, openedInfo) {
		_ = root.Close()
		return nil, unsafeStorage("application configuration directory changed during open")
	}
	return root, nil
}

func validateReplaceTarget(root *os.Root) error {
	info, err := root.Lstat(configurationFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return unavailable("configuration target metadata")
	}
	if !validConfigurationFile(info) {
		return unsafeStorage("configuration target contract")
	}
	return nil
}

func validApplicationDirectory(info fs.FileInfo) bool {
	return runtime.GOOS == "windows" || info.Mode().Perm() == 0o700
}

func validConfigurationFile(info fs.FileInfo) bool {
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false
	}
	return runtime.GOOS == "windows" || info.Mode().Perm() == 0o600
}

func contextFault(ctx context.Context) error {
	if ctx == nil {
		return fault.New(fault.KindContract, "missing_context", "コマンド選択コンテキストが設定されていません", false)
	}
	if err := ctx.Err(); err != nil {
		return fault.Wrap(fault.KindCanceled, "operation_canceled", "コマンド選択処理がキャンセルされました", true, err)
	}
	return nil
}

func invalid(reason string) error {
	return invalidWithCause(reason, nil)
}

func invalidWithCause(reason string, cause error) error {
	if cause == nil {
		cause = fmt.Errorf("%w: %s", ErrInvalid, reason)
	} else {
		cause = fmt.Errorf("%w: %s: %v", ErrInvalid, reason, cause)
	}
	return fault.Wrap(
		fault.KindInvalidInput,
		"command_selection_invalid",
		"コマンド選択は無効です",
		false,
		cause,
	)
}

func unsafeStorage(reason string) error {
	return fault.Wrap(
		fault.KindUnavailable,
		"command_selection_unsafe",
		"コマンド選択の保存先は安全ではありません",
		false,
		fmt.Errorf("%w: %s", ErrUnsafe, reason),
	)
}

func unavailable(reason string) error {
	return fault.Wrap(
		fault.KindUnavailable,
		"command_selection_unavailable",
		"コマンド選択を利用できません",
		true,
		fmt.Errorf("%w: %s", ErrUnavailable, reason),
	)
}
