package chatworkconfig

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFileStoreRoundTripAndAtomicReplacement(t *testing.T) {
	base := t.TempDir()
	store := NewFileStoreAt(base)
	first, err := NewOAuthPublicConfig("public-client-one")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), first); err != nil {
		t.Fatalf("Save(first): %v", err)
	}
	path := filepath.Join(base, applicationDirectory, configurationFile)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\"schema_version\":1,\"auth_method\":\"oauth2\",\"client_id\":\"public-client-one\",\"redirect_uri\":\"cwk://oauth/callback\"}\n"
	if string(contents) != want {
		t.Fatalf("stored contents = %q, want %q", contents, want)
	}
	if strings.Contains(string(contents), "access_token") || strings.Contains(string(contents), "refresh_token") {
		t.Fatalf("credential field entered public config: %s", contents)
	}
	assertRestrictedMode(t, filepath.Join(base, applicationDirectory), true)
	assertRestrictedMode(t, path, false)

	second, err := NewOAuthPublicConfig("public-client-two")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), second); err != nil {
		t.Fatalf("Save(second): %v", err)
	}
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != second {
		t.Fatalf("Load() = %#v, want %#v", loaded, second)
	}
	entries, err := os.ReadDir(filepath.Join(base, applicationDirectory))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != configurationFile {
		t.Fatalf("configuration entries = %v", entryNames(entries))
	}
}

func TestFileStoreMissingIsDistinct(t *testing.T) {
	_, err := NewFileStoreAt(t.TempDir()).Load(context.Background())
	if !errors.Is(err, ErrConfigNotFound) || errors.Is(err, ErrConfigInvalid) || errors.Is(err, ErrConfigUnavailable) {
		t.Fatalf("Load missing error = %v", err)
	}
}

func TestFileStoreRejectsInvalidSerializedConfiguration(t *testing.T) {
	tests := map[string]string{
		"empty":            "",
		"malformed":        "{",
		"array":            "[]",
		"unknown schema":   `{"schema_version":2,"auth_method":"oauth2","client_id":"client","redirect_uri":"cwk://oauth/callback"}`,
		"wrong method":     `{"schema_version":1,"auth_method":"pat","client_id":"client","redirect_uri":"cwk://oauth/callback"}`,
		"wrong redirect":   `{"schema_version":1,"auth_method":"oauth2","client_id":"client","redirect_uri":"other://oauth/callback"}`,
		"missing field":    `{"schema_version":1,"auth_method":"oauth2","client_id":"client"}`,
		"wrong field type": `{"schema_version":"1","auth_method":"oauth2","client_id":"client","redirect_uri":"cwk://oauth/callback"}`,
		"unknown field":    `{"schema_version":1,"auth_method":"oauth2","client_id":"client","redirect_uri":"cwk://oauth/callback","unexpected":"value"}`,
		"duplicate field":  `{"schema_version":1,"auth_method":"oauth2","client_id":"client","client_id":"replacement","redirect_uri":"cwk://oauth/callback"}`,
		"trailing value":   `{"schema_version":1,"auth_method":"oauth2","client_id":"client","redirect_uri":"cwk://oauth/callback"}{}`,
		"blank client":     `{"schema_version":1,"auth_method":"oauth2","client_id":"","redirect_uri":"cwk://oauth/callback"}`,
		"unsafe client":    "{\"schema_version\":1,\"auth_method\":\"oauth2\",\"client_id\":\"client\\nvalue\",\"redirect_uri\":\"cwk://oauth/callback\"}",
	}
	for name, contents := range tests {
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			writeConfigFixture(t, base, []byte(contents))
			_, err := NewFileStoreAt(base).Load(context.Background())
			if !errors.Is(err, ErrConfigInvalid) || errors.Is(err, ErrConfigNotFound) || errors.Is(err, ErrConfigUnavailable) {
				t.Fatalf("Load error = %v", err)
			}
		})
	}
}

func TestFileStoreRejectsOversizedConfiguration(t *testing.T) {
	base := t.TempDir()
	writeConfigFixture(t, base, []byte(strings.Repeat("x", maxConfigBytes+1)))
	_, err := NewFileStoreAt(base).Load(context.Background())
	if !errors.Is(err, ErrConfigInvalid) {
		t.Fatalf("Load oversized error = %v", err)
	}
}

func TestFileStoreRejectsUnsafeFilesystemObjects(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic-link creation requires platform-specific privileges on Windows")
	}
	t.Run("base symlink", func(t *testing.T) {
		parent := t.TempDir()
		realBase := filepath.Join(parent, "real")
		if err := os.Mkdir(realBase, 0o700); err != nil {
			t.Fatal(err)
		}
		linkBase := filepath.Join(parent, "link")
		if err := os.Symlink(realBase, linkBase); err != nil {
			t.Fatal(err)
		}
		_, err := NewFileStoreAt(linkBase).Load(context.Background())
		if !errors.Is(err, ErrConfigInvalid) {
			t.Fatalf("Load base symlink error = %v", err)
		}
	})
	t.Run("application symlink", func(t *testing.T) {
		base := t.TempDir()
		target := t.TempDir()
		if err := os.Symlink(target, filepath.Join(base, applicationDirectory)); err != nil {
			t.Fatal(err)
		}
		_, err := NewFileStoreAt(base).Load(context.Background())
		if !errors.Is(err, ErrConfigInvalid) {
			t.Fatalf("Load application symlink error = %v", err)
		}
	})
	t.Run("file symlink", func(t *testing.T) {
		base := t.TempDir()
		app := filepath.Join(base, applicationDirectory)
		if err := os.Mkdir(app, 0o700); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(base, "target")
		if err := os.WriteFile(target, []byte("do not replace"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(app, configurationFile)); err != nil {
			t.Fatal(err)
		}
		config, _ := NewOAuthPublicConfig("client")
		err := NewFileStoreAt(base).Save(context.Background(), config)
		if !errors.Is(err, ErrConfigInvalid) {
			t.Fatalf("Save file symlink error = %v", err)
		}
		contents, readErr := os.ReadFile(target)
		if readErr != nil || string(contents) != "do not replace" {
			t.Fatalf("symlink target changed: contents=%q err=%v", contents, readErr)
		}
	})
	t.Run("file directory", func(t *testing.T) {
		base := t.TempDir()
		app := filepath.Join(base, applicationDirectory)
		if err := os.Mkdir(app, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(app, configurationFile), 0o700); err != nil {
			t.Fatal(err)
		}
		_, err := NewFileStoreAt(base).Load(context.Background())
		if !errors.Is(err, ErrConfigInvalid) {
			t.Fatalf("Load non-regular error = %v", err)
		}
	})
}

func TestFileStoreRejectsPermissiveObjectsWhereModesAreSupported(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ACLs are not represented by Unix mode bits")
	}
	t.Run("directory", func(t *testing.T) {
		base := t.TempDir()
		writeConfigFixture(t, base, []byte(`{"schema_version":1,"auth_method":"oauth2","client_id":"client","redirect_uri":"cwk://oauth/callback"}`))
		if err := os.Chmod(filepath.Join(base, applicationDirectory), 0o755); err != nil {
			t.Fatal(err)
		}
		_, err := NewFileStoreAt(base).Load(context.Background())
		if !errors.Is(err, ErrConfigInvalid) {
			t.Fatalf("Load permissive directory error = %v", err)
		}
	})
	t.Run("file", func(t *testing.T) {
		base := t.TempDir()
		writeConfigFixture(t, base, []byte(`{"schema_version":1,"auth_method":"oauth2","client_id":"client","redirect_uri":"cwk://oauth/callback"}`))
		if err := os.Chmod(filepath.Join(base, applicationDirectory, configurationFile), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := NewFileStoreAt(base).Load(context.Background())
		if !errors.Is(err, ErrConfigInvalid) {
			t.Fatalf("Load permissive file error = %v", err)
		}
	})
}

func TestFileStoreSaveRejectsInvalidInputBeforeFilesystemWrite(t *testing.T) {
	base := t.TempDir()
	store := NewFileStoreAt(base)
	invalidConfigs := []PublicConfig{
		{},
		{AuthMethod: "pat", ClientID: "client", RedirectURI: FixedRedirectURI},
		{AuthMethod: AuthMethodOAuth2, ClientID: "client", RedirectURI: "https://example.com/callback"},
		{AuthMethod: AuthMethodOAuth2, ClientID: " client", RedirectURI: FixedRedirectURI},
		{AuthMethod: AuthMethodOAuth2, ClientID: strings.Repeat("x", maxClientIDBytes+1), RedirectURI: FixedRedirectURI},
	}
	for _, config := range invalidConfigs {
		if err := store.Save(context.Background(), config); !errors.Is(err, ErrConfigInvalid) {
			t.Fatalf("Save(%#v) error = %v", config, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(base, applicationDirectory)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid Save created filesystem state: %v", err)
	}
}

func TestFileStoreUnavailableAndCanceledAreDistinct(t *testing.T) {
	store := NewFileStoreAt(string([]byte{'/', 0, 'x'}))
	_, err := store.Load(context.Background())
	if !errors.Is(err, ErrConfigUnavailable) || errors.Is(err, ErrConfigInvalid) || errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("Load unavailable error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = NewFileStoreAt(t.TempDir()).Load(ctx)
	if !errors.Is(err, context.Canceled) || errors.Is(err, ErrConfigUnavailable) {
		t.Fatalf("Load canceled error = %v", err)
	}
	config, _ := NewOAuthPublicConfig("client")
	err = NewFileStoreAt(t.TempDir()).Save(ctx, config)
	if !errors.Is(err, context.Canceled) || errors.Is(err, ErrConfigUnavailable) {
		t.Fatalf("Save canceled error = %v", err)
	}
}

func TestNewOAuthPublicConfigPinsPublicFields(t *testing.T) {
	config, err := NewOAuthPublicConfig("public-client")
	if err != nil {
		t.Fatal(err)
	}
	if config.AuthMethod != AuthMethodOAuth2 || config.ClientID != "public-client" || config.RedirectURI != FixedRedirectURI {
		t.Fatalf("NewOAuthPublicConfig() = %#v", config)
	}
	for _, clientID := range []string{"", " client", "client\nvalue", string([]byte{0xff}), strings.Repeat("x", maxClientIDBytes+1)} {
		if _, err := NewOAuthPublicConfig(clientID); !errors.Is(err, ErrConfigInvalid) {
			t.Fatalf("NewOAuthPublicConfig(%q) error = %v", clientID, err)
		}
	}
}

func writeConfigFixture(t *testing.T, base string, contents []byte) {
	t.Helper()
	app := filepath.Join(base, applicationDirectory)
	if err := os.Mkdir(app, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, configurationFile), contents, 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertRestrictedMode(t *testing.T, path string, directory bool) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if directory != info.IsDir() {
		t.Fatalf("%s directory = %t", path, info.IsDir())
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("%s permissions = %o", path, info.Mode().Perm())
	}
}

func entryNames(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}
