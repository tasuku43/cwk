//go:build !windows

package chatworkconfig

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUserConfigDirUsesAbsoluteXDGOnUnixIncludingDarwin(t *testing.T) {
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home-must-not-be-used"))

	got, err := resolveUserConfigDir()
	if err != nil {
		t.Fatalf("resolveUserConfigDir: %v", err)
	}
	if got != xdg {
		t.Fatalf("resolveUserConfigDir() = %q, want %q", got, xdg)
	}

	config, err := NewOAuthPublicConfig("public-client")
	if err != nil {
		t.Fatal(err)
	}
	if err := NewFileStore().Save(context.Background(), config); err != nil {
		t.Fatalf("Save with production resolver: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(xdg, applicationDirectory, configurationFile)); err != nil {
		t.Fatalf("XDG configuration was not created: %v", err)
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
	if err != nil {
		t.Fatalf("resolveUserConfigDir: %v", err)
	}
	want := filepath.Join(home, ".config")
	if got != want {
		t.Fatalf("resolveUserConfigDir() = %q, want %q", got, want)
	}
}

func TestResolveUserConfigDirRejectsRelativeXDGWithoutHomeFallback(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home-must-not-be-used")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join("relative", "config"))
	t.Setenv("HOME", home)

	if _, err := resolveUserConfigDir(); err == nil {
		t.Fatal("relative XDG_CONFIG_HOME was accepted")
	}
	_, err := NewFileStore().Load(context.Background())
	if !errors.Is(err, ErrConfigUnavailable) || errors.Is(err, ErrConfigNotFound) || errors.Is(err, ErrConfigInvalid) {
		t.Fatalf("Load relative XDG error = %v", err)
	}
	config, configErr := NewOAuthPublicConfig("public-client")
	if configErr != nil {
		t.Fatal(configErr)
	}
	if err := NewFileStore().Save(context.Background(), config); !errors.Is(err, ErrConfigUnavailable) {
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
