package cli

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func launchCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "launch",
		Short:  "Open the canvas in a new herdr tab",
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
			open := exec.Command("herdr", "plugin", "pane", "open",
				"--plugin", "herdr-canvas",
				"--entrypoint", "canvas",
				"--placement", "tab",
				"--cwd", cwd,
			)
			open.Stdin = os.Stdin
			open.Stdout = os.Stdout
			open.Stderr = os.Stderr
			return open.Run()
		},
	}
}
