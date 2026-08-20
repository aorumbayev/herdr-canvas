package cli

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func launchCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "launch",
		Short:  "Open the canvas beside the active herdr pane",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd := os.Getenv("HERDR_ACTIVE_PANE_CWD")
			if cwd == "" {
				var err error
				cwd, err = os.Getwd()
				if err != nil {
					return err
				}
			}
			// herdr runs a plugin command with a minimal PATH, so the name
			// "herdr" does not always resolve.
			bin := os.Getenv("HERDR_BIN_PATH")
			if bin == "" {
				bin = "herdr"
			}
			open := exec.Command(bin, "plugin", "pane", "open",
				"--plugin", "herdr-canvas",
				"--entrypoint", "canvas",
				"--placement", "split",
				"--direction", "right",
				"--focus",
				"--cwd", cwd,
			)
			open.Stdin = os.Stdin
			open.Stdout = os.Stdout
			open.Stderr = os.Stderr
			return open.Run()
		},
	}
}
