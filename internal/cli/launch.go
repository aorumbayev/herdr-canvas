package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"herdr-canvas/internal/herdr"
)

// binaryName is the program the canvas pane runs. findPanes reads it from the
// foreground process group of each pane.
const binaryName = "herdr-canvas"

func launchCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "launch",
		Short:  "Open the canvas beside the active pane, or close it when it is open",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return toggle(herdr.New(), herdr.Workspace())
		},
	}
}

// paneHost is the part of the herdr client that toggle uses. The interface
// lets a test drive toggle without a herdr server.
type paneHost interface {
	PaneIDs(workspace string) ([]string, error)
	RunsProgram(paneID, prog string) (bool, error)
	ClosePane(paneID string) error
	OpenSplit(cwd string) error
}

// toggle closes every canvas pane of the workspace. If the workspace has no
// canvas pane, toggle opens one beside the active pane.
func toggle(c paneHost, workspace string) error {
	if workspace == "" {
		return fmt.Errorf("no workspace; run this inside herdr")
	}
	panes, err := c.PaneIDs(workspace)
	if err != nil {
		return err
	}
	var open []string
	for _, id := range panes {
		is, err := c.RunsProgram(id, binaryName)
		if err != nil {
			// A failed read must not read as "no canvas pane". That would
			// stack a second canvas on top of the first one.
			return err
		}
		if is {
			open = append(open, id)
		}
	}
	if len(open) > 0 {
		for _, id := range open {
			if err := c.ClosePane(id); err != nil {
				return err
			}
		}
		return nil
	}

	cwd := os.Getenv("HERDR_ACTIVE_PANE_CWD")
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	return c.OpenSplit(cwd)
}
