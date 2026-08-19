package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// keybindingBlock is appended to the herdr config to bind the canvas hotkey.
// prefix+c is herdr's default new-tab binding, so we use prefix+d.
const keybindingBlock = `
[[keys.command]]
key = "prefix+d"
type = "plugin_action"
command = "herdr-canvas.open"
description = "open the herdr-canvas diagram in a new tab"
`

const keybindingMarker = "herdr-canvas.open"

func setupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Install the herdr hotkey binding",
		RunE: func(cmd *cobra.Command, args []string) error {
			return installKeybinding()
		},
	}
}

func herdrConfigPath() string {
	if p := os.Getenv("HERDR_CONFIG_PATH"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "herdr", "config.toml")
}

// installKeybinding appends the canvas hotkey to the herdr config if absent.
// ponytail: append-or-skip, no config-check validation or backup.
func installKeybinding() error {
	path := herdrConfigPath()
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if strings.Contains(string(data), keybindingMarker) {
		fmt.Println("herdr-canvas keybinding already present; skipping")
		return nil
	}
	content := string(data)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += keybindingBlock
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
