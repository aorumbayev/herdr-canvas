package cli

import (
	"encoding/json"
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

// paneCwd returns the directory of the pane the person invoked the hotkey
// from. The canvas names its diagram after that directory, so the wrong answer
// opens the wrong diagram.
//
// The working directory of this process is the plugin root, which is a
// detached checkout of this repository. Naming a diagram after it produces
// repo@<commit> and gives a different diagram for every installed version, so
// paneCwd refuses it.
func paneCwd() (string, error) {
	if p := os.Getenv("HERDR_ACTIVE_PANE_CWD"); p != "" {
		return p, nil
	}
	if raw := os.Getenv("HERDR_PLUGIN_CONTEXT_JSON"); raw != "" {
		var ctx struct {
			FocusedPaneCwd string `json:"focused_pane_cwd"`
			WorkspaceCwd   string `json:"workspace_cwd"`
		}
		if err := json.Unmarshal([]byte(raw), &ctx); err == nil {
			if ctx.FocusedPaneCwd != "" {
				return ctx.FocusedPaneCwd, nil
			}
			if ctx.WorkspaceCwd != "" {
				return ctx.WorkspaceCwd, nil
			}
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if root := os.Getenv("HERDR_PLUGIN_ROOT"); root != "" && wd == root {
		return "", fmt.Errorf("cannot tell which pane this is; herdr set no pane directory")
	}
	return wd, nil
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

	cwd, err := paneCwd()
	if err != nil {
		return err
	}
	return c.OpenSplit(cwd)
}
