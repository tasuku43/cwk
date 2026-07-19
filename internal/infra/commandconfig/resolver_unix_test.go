//go:build !windows

package commandconfig

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tasuku43/cwk/internal/domain/commandselection"
)

func TestResolveUserConfigDirUsesAbsoluteXDGOnUnixIncludingDarwin(t *testing.T) {
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home-must-not-be-used"))

	got, err := resolveUserConfigDir()
	if err != nil || got != xdg {
		t.Fatalf("resolveUserConfigDir() = %q, %v; want %q", got, err, xdg)
	}
	if err := NewFileStore().Save(context.Background(), commandselection.Profile{}); err != nil {
		t.Fatalf("Save with production resolver: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(xdg, applicationDirectory, configurationFile)); err != nil {
		t.Fatalf("XDG configuration was not created: %v", err)
	}
}

func TestProductionStoreResolvesSymlinkedXDGConfigHome(t *testing.T) {
	parent := t.TempDir()
	realBase := filepath.Join(parent, "dotfiles-config")
	if err := os.Mkdir(realBase, 0o700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(parent, "xdg")
	if err := os.Symlink(filepath.Base(realBase), alias); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", alias)
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home-must-not-be-used"))

	profile, _ := commandselection.New([]string{"rooms list"})
	store := NewFileStore()
	if err := store.Save(context.Background(), profile); err != nil {
		t.Fatalf("Save with symlinked XDG configuration home: %v", err)
	}
	loaded, configured, err := store.Load(context.Background())
	if err != nil || !configured || len(loaded.EnabledCommands()) != 1 || loaded.EnabledCommands()[0] != "rooms list" {
		t.Fatalf("Load with symlinked XDG configuration home = %#v, configured=%t err=%v", loaded, configured, err)
	}
	if _, err := os.Lstat(filepath.Join(realBase, applicationDirectory, configurationFile)); err != nil {
		t.Fatalf("resolved XDG target does not contain configuration: %v", err)
	}
}

func TestResolveUserConfigDirFallsBackToHomeDotConfig(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	if err := os.Mkdir(home, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	got, err := resolveUserConfigDir()
	want := filepath.Join(home, ".config")
	if err != nil || got != want {
		t.Fatalf("resolveUserConfigDir() = %q, %v; want %q", got, err, want)
	}
}

func TestResolveUserConfigDirRejectsRelativeXDGWithoutHomeFallback(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home-must-not-be-used")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join("relative", "config"))
	t.Setenv("HOME", home)

	if _, err := resolveUserConfigDir(); err == nil {
		t.Fatal("relative XDG_CONFIG_HOME was accepted")
	}
	_, _, err := NewFileStore().Load(context.Background())
	if !errors.Is(err, ErrUnavailable) || errors.Is(err, ErrInvalid) {
		t.Fatalf("Load relative XDG error = %v", err)
	}
	if err := NewFileStore().Save(context.Background(), commandselection.Profile{}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Save relative XDG error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(home, ".config")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("relative XDG unexpectedly fell back to HOME: %v", err)
	}
}

func TestResolveUserConfigDirRejectsUnavailableOrRelativeHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	for _, home := range []string{"", filepath.Join("relative", "home")} {
		t.Run(home, func(t *testing.T) {
			t.Setenv("HOME", home)
			if _, err := resolveUserConfigDir(); err == nil {
				t.Fatalf("HOME %q was accepted", home)
			}
		})
	}
}
