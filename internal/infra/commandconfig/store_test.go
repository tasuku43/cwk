package commandconfig

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/commandselection"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

func TestFileStoreRoundTripAtomicReplacementAndModes(t *testing.T) {
	base := t.TempDir()
	store := NewFileStoreAt(base)
	first, _ := commandselection.New([]string{"messages list", "rooms list", "retired command"})
	if err := store.Save(context.Background(), first); err != nil {
		t.Fatalf("Save(first): %v", err)
	}
	path := filepath.Join(base, applicationDirectory, configurationFile)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\"schema_version\":1,\"enabled_commands\":[\"messages list\",\"rooms list\",\"retired command\"]}\n"
	if string(contents) != want {
		t.Fatalf("stored contents = %q, want %q", contents, want)
	}
	for _, forbidden := range []string{"CWK_API_TOKEN", "access_token", "refresh_token", "task.teckac"} {
		if strings.Contains(string(contents), forbidden) {
			t.Fatalf("forbidden field entered command configuration: %q", forbidden)
		}
	}
	assertExactMode(t, filepath.Join(base, applicationDirectory), 0o700, true)
	assertExactMode(t, path, 0o600, false)

	second, _ := commandselection.New([]string{"account show"})
	if err := store.Save(context.Background(), second); err != nil {
		t.Fatalf("Save(second): %v", err)
	}
	loaded, configured, err := store.Load(context.Background())
	if err != nil || !configured {
		t.Fatalf("Load: configured=%t err=%v", configured, err)
	}
	if got, want := fmt.Sprint(loaded.EnabledCommands()), "[account show]"; got != want {
		t.Fatalf("Load enabled = %s, want %s", got, want)
	}
	entries, err := os.ReadDir(filepath.Join(base, applicationDirectory))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != configurationFile {
		t.Fatalf("configuration entries = %v", entryNames(entries))
	}
}

func TestFileStoreDistinguishesMissingAndSavedEmptyProfile(t *testing.T) {
	base := t.TempDir()
	store := NewFileStoreAt(base)
	profile, configured, err := store.Load(context.Background())
	if err != nil || configured || len(profile.EnabledCommands()) != 0 {
		t.Fatalf("missing Load = %#v, configured=%t err=%v", profile, configured, err)
	}
	if err := store.Save(context.Background(), commandselection.Profile{}); err != nil {
		t.Fatalf("Save(empty): %v", err)
	}
	profile, configured, err = store.Load(context.Background())
	if err != nil || !configured || len(profile.EnabledCommands()) != 0 {
		t.Fatalf("saved-empty Load = %#v, configured=%t err=%v", profile, configured, err)
	}
	contents, err := os.ReadFile(filepath.Join(base, applicationDirectory, configurationFile))
	if err != nil || string(contents) != "{\"schema_version\":1,\"enabled_commands\":[]}\n" {
		t.Fatalf("saved-empty contents=%q err=%v", contents, err)
	}
}

func TestFileStoreRejectsInvalidSerializedConfiguration(t *testing.T) {
	tooMany := make([]string, commandselection.MaxEnabledCommands+1)
	for index := range tooMany {
		tooMany[index] = fmt.Sprintf("command-%d", index)
	}
	tooManyJSON := `{"schema_version":1,"enabled_commands":["` + strings.Join(tooMany, `","`) + `"]}`
	tests := map[string][]byte{
		"empty":              {},
		"malformed":          []byte("{"),
		"array root":         []byte("[]"),
		"unknown schema":     []byte(`{"schema_version":2,"enabled_commands":[]}`),
		"missing field":      []byte(`{"schema_version":1}`),
		"wrong version type": []byte(`{"schema_version":"1","enabled_commands":[]}`),
		"wrong list type":    []byte(`{"schema_version":1,"enabled_commands":{}}`),
		"null list":          []byte(`{"schema_version":1,"enabled_commands":null}`),
		"wrong item type":    []byte(`{"schema_version":1,"enabled_commands":[1]}`),
		"unknown field":      []byte(`{"schema_version":1,"enabled_commands":[],"unexpected":true}`),
		"duplicate field":    []byte(`{"schema_version":1,"enabled_commands":[],"enabled_commands":[]}`),
		"trailing value":     []byte(`{"schema_version":1,"enabled_commands":[]}{}`),
		"duplicate command":  []byte(`{"schema_version":1,"enabled_commands":["rooms list","rooms list"]}`),
		"invalid command":    []byte(`{"schema_version":1,"enabled_commands":["Rooms list"]}`),
		"too many commands":  []byte(tooManyJSON),
		"invalid utf8":       {'{', '"', 0xff, '"', '}'},
	}
	for name, contents := range tests {
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			writeConfigFixture(t, base, contents, 0o600, 0o700)
			_, configured, err := NewFileStoreAt(base).Load(context.Background())
			if configured || !errors.Is(err, ErrInvalid) || errors.Is(err, ErrUnavailable) {
				t.Fatalf("Load error=%#v configured=%t", err, configured)
			}
			assertSafeConfigFault(t, err, fault.KindInvalidInput, "command_selection_invalid")
		})
	}
}

func TestFileStoreRejectsOversizedConfiguration(t *testing.T) {
	base := t.TempDir()
	writeConfigFixture(t, base, []byte(strings.Repeat("x", maxConfigBytes+1)), 0o600, 0o700)
	_, configured, err := NewFileStoreAt(base).Load(context.Background())
	if configured || !errors.Is(err, ErrInvalid) {
		t.Fatalf("Load oversized: configured=%t err=%v", configured, err)
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
		_, _, err := NewFileStoreAt(linkBase).Load(context.Background())
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("Load base symlink error = %v", err)
		}
	})
	t.Run("application symlink", func(t *testing.T) {
		base := t.TempDir()
		target := t.TempDir()
		if err := os.Symlink(target, filepath.Join(base, applicationDirectory)); err != nil {
			t.Fatal(err)
		}
		_, _, err := NewFileStoreAt(base).Load(context.Background())
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("Load application symlink error = %v", err)
		}
	})
	t.Run("file symlink is not replaced", func(t *testing.T) {
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
		err := NewFileStoreAt(base).Save(context.Background(), commandselection.Profile{})
		if !errors.Is(err, ErrInvalid) {
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
		_, _, err := NewFileStoreAt(base).Load(context.Background())
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("Load non-regular error = %v", err)
		}
	})
}

func TestFileStoreRejectsPermissiveApplicationObjects(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ACLs are not represented by Unix mode bits")
	}
	t.Run("directory on load and save", func(t *testing.T) {
		base := t.TempDir()
		writeConfigFixture(t, base, []byte(`{"schema_version":1,"enabled_commands":[]}`), 0o600, 0o755)
		store := NewFileStoreAt(base)
		if _, _, err := store.Load(context.Background()); !errors.Is(err, ErrInvalid) {
			t.Fatalf("Load permissive directory error = %v", err)
		}
		if err := store.Save(context.Background(), commandselection.Profile{}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("Save permissive directory error = %v", err)
		}
	})
	t.Run("file on load and save", func(t *testing.T) {
		base := t.TempDir()
		writeConfigFixture(t, base, []byte(`{"schema_version":1,"enabled_commands":[]}`), 0o644, 0o700)
		store := NewFileStoreAt(base)
		if _, _, err := store.Load(context.Background()); !errors.Is(err, ErrInvalid) {
			t.Fatalf("Load permissive file error = %v", err)
		}
		if err := store.Save(context.Background(), commandselection.Profile{}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("Save permissive file error = %v", err)
		}
	})
}

func TestFileStoreUnavailableCanceledAndNilContextAreStructured(t *testing.T) {
	store := NewFileStoreAt(string([]byte{'/', 0, 'x'}))
	_, _, err := store.Load(context.Background())
	if !errors.Is(err, ErrUnavailable) || errors.Is(err, ErrInvalid) {
		t.Fatalf("Load unavailable error = %v", err)
	}
	assertSafeConfigFault(t, err, fault.KindUnavailable, "command_selection_unavailable")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = NewFileStoreAt(t.TempDir()).Load(ctx)
	assertSafeConfigFault(t, err, fault.KindCanceled, "operation_canceled")
	if err := NewFileStoreAt(t.TempDir()).Save(ctx, commandselection.Profile{}); err == nil {
		t.Fatal("canceled Save succeeded")
	} else {
		assertSafeConfigFault(t, err, fault.KindCanceled, "operation_canceled")
	}

	_, _, err = NewFileStoreAt(t.TempDir()).Load(nil)
	assertSafeConfigFault(t, err, fault.KindContract, "missing_context")
	if err := NewFileStoreAt(t.TempDir()).Save(nil, commandselection.Profile{}); err == nil {
		t.Fatal("nil-context Save succeeded")
	} else {
		assertSafeConfigFault(t, err, fault.KindContract, "missing_context")
	}
}

func writeConfigFixture(t *testing.T, base string, contents []byte, fileMode, directoryMode os.FileMode) {
	t.Helper()
	app := filepath.Join(base, applicationDirectory)
	if err := os.Mkdir(app, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, configurationFile), contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(filepath.Join(app, configurationFile), fileMode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(app, directoryMode); err != nil {
			t.Fatal(err)
		}
	}
}

func assertExactMode(t *testing.T, path string, mode os.FileMode, directory bool) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.IsDir() != directory || info.Mode().Perm() != mode {
		t.Fatalf("%s mode=%o dir=%t", path, info.Mode().Perm(), info.IsDir())
	}
}

func assertSafeConfigFault(t *testing.T, err error, kind fault.Kind, code string) {
	t.Helper()
	var structured *fault.Error
	if !errors.As(err, &structured) || structured.Validate() != nil || structured.Kind != kind || structured.Code != code {
		t.Fatalf("error=%#v, want valid %s/%s fault", err, kind, code)
	}
	if strings.Contains(err.Error(), string(filepath.Separator)) || strings.Contains(err.Error(), "private") {
		t.Fatalf("public fault leaked path or private content: %q", err.Error())
	}
}

func entryNames(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}
