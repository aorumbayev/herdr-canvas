package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// keybindingBlock is the text that installKeybinding appends to the herdr
// config file. herdr uses prefix+c for a new tab. This plugin uses prefix+d.
const keybindingBlock = `
[[keys.command]]
key = "prefix+d"
type = "plugin_action"
command = "herdr-canvas.open"
description = "toggle the herdr-canvas diagram beside this pane"
`

const keybindingMarker = "herdr-canvas.open"

func setupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Install the herdr hotkey binding",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := installKeybinding(); err != nil {
				return err
			}
			return linkBinary()
		},
	}
}

// linkBinary puts herdr-canvas on the PATH of the person and of every agent
// they run. An agent that gets the diagram must be able to run the command it
// is told to run. Without the link the agent searches the disk for the binary
// before it can draw.
//
// The link points at HERDR_PLUGIN_ROOT, which herdr sets for a build step. The
// function replaces a link and never a real file, and a failure here never
// fails the install.
func linkBinary() error {
	root := os.Getenv("HERDR_PLUGIN_ROOT")
	if root == "" {
		return nil
	}
	target := filepath.Join(root, "bin", "herdr-canvas")
	if _, err := os.Stat(target); err != nil {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dir := filepath.Join(home, ".local", "bin")
	if _, err := os.Stat(dir); err != nil {
		return nil
	}
	link := filepath.Join(dir, "herdr-canvas")
	if fi, err := os.Lstat(link); err == nil && fi.Mode()&os.ModeSymlink == 0 {
		fmt.Printf("%s is not a link; leaving it alone\n", link)
		return nil
	}
	_ = os.Remove(link)
	if err := os.Symlink(target, link); err != nil {
		fmt.Printf("could not link %s: %v\n", link, err)
		return nil
	}
	fmt.Printf("linked %s -> %s\n", link, target)
	return nil
}

func herdrConfigPath() string {
	if p := os.Getenv("HERDR_CONFIG_PATH"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "herdr", "config.toml")
}

// installKeybinding appends the canvas hotkey to the herdr config file.
// installKeybinding makes no change if the hotkey is already in the file.
// This function does not validate the config file. It does not make a backup.
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
