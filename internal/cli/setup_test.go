package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallKeybindingAppendsAndIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("HERDR_CONFIG_PATH", path)

	if err := installKeybinding(); err != nil {
		t.Fatalf("first install: %v", err)
	}
	first, _ := os.ReadFile(path)
	if !strings.Contains(string(first), `command = "herdr-canvas.open"`) {
		t.Fatalf("keybinding not written:\n%s", first)
	}

	if err := installKeybinding(); err != nil {
		t.Fatalf("second install: %v", err)
	}
	second, _ := os.ReadFile(path)
	if string(second) != string(first) {
		t.Fatalf("reinstall mutated config:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestInstallKeybindingPreservesExistingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("HERDR_CONFIG_PATH", path)
	existing := "onboarding = false\n[ui]\nshow_agent_labels_on_pane_borders = true\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installKeybinding(); err != nil {
		t.Fatalf("install: %v", err)
	}
	got, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(got), existing) {
		t.Fatalf("existing config not preserved:\n%s", got)
	}
	if !strings.Contains(string(got), `key = "prefix+d"`) {
		t.Fatalf("prefix+d binding missing:\n%s", got)
	}
}
