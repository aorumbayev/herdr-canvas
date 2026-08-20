// Package herdr calls the herdr command line. The canvas uses it to find its
// own panes, to close them, and to send a diagram to an agent.
package herdr

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Agent is one agent pane in a workspace. TabLabel and WorkspaceLabel do not
// come from the agent list. Agents fills them from the tab and workspace
// lists, because a pane id alone does not tell a person which agent this is.
type Agent struct {
	PaneID    string `json:"pane_id"`
	Agent     string `json:"agent"`
	Status    string `json:"agent_status"`
	Workspace string `json:"workspace_id"`
	TabID     string `json:"tab_id"`
	Title     string `json:"terminal_title_stripped"`

	TabLabel       string `json:"-"`
	WorkspaceLabel string `json:"-"`
}

// Client runs the herdr command line.
type Client struct {
	bin string
}

// New returns a client. herdr runs a plugin command with a minimal PATH, so
// the client prefers HERDR_BIN_PATH over the name "herdr".
func New() *Client {
	bin := os.Getenv("HERDR_BIN_PATH")
	if bin == "" {
		bin = "herdr"
	}
	return &Client{bin: bin}
}

// Workspace returns the workspace of the running pane. The value is empty
// outside herdr.
func Workspace() string { return os.Getenv("HERDR_WORKSPACE_ID") }

func (c *Client) run(args ...string) ([]byte, error) {
	out, err := exec.Command(c.bin, args...).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("herdr %s: %s", args[0], strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("herdr %s: %w", args[0], err)
	}
	return out, nil
}

// PaneIDs returns the panes of a workspace.
func (c *Client) PaneIDs(workspace string) ([]string, error) {
	out, err := c.run("pane", "list", "--workspace", workspace)
	if err != nil {
		return nil, err
	}
	var env struct {
		Result struct {
			Panes []struct {
				PaneID string `json:"pane_id"`
			} `json:"panes"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		return nil, fmt.Errorf("pane list: %w", err)
	}
	ids := make([]string, 0, len(env.Result.Panes))
	for _, p := range env.Result.Panes {
		ids = append(ids, p.PaneID)
	}
	return ids, nil
}

// RunsProgram reports whether a pane runs prog in its foreground process
// group. The check reads argv0, because a process can rewrite its name.
func (c *Client) RunsProgram(paneID, prog string) (bool, error) {
	out, err := c.run("pane", "process-info", "--pane", paneID)
	if err != nil {
		// A pane that closed between the list and this read is not a match.
		if strings.Contains(err.Error(), "pane_not_found") {
			return false, nil
		}
		return false, err
	}
	var env struct {
		Result struct {
			ProcessInfo struct {
				Foreground []struct {
					Argv0 string   `json:"argv0"`
					Argv  []string `json:"argv"`
				} `json:"foreground_processes"`
			} `json:"process_info"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		return false, fmt.Errorf("process-info: %w", err)
	}
	for _, p := range env.Result.ProcessInfo.Foreground {
		if filepath.Base(p.Argv0) == prog {
			return true, nil
		}
		if len(p.Argv) > 0 && filepath.Base(p.Argv[0]) == prog {
			return true, nil
		}
	}
	return false, nil
}

// ClosePane closes one pane.
func (c *Client) ClosePane(paneID string) error {
	_, err := c.run("pane", "close", paneID)
	return err
}

// OpenSplit opens the canvas pane to the right of the active pane.
func (c *Client) OpenSplit(cwd string) error {
	_, err := c.run("plugin", "pane", "open",
		"--plugin", "herdr-canvas",
		"--entrypoint", "canvas",
		"--placement", "split",
		"--direction", "right",
		"--focus",
		"--cwd", cwd,
	)
	return err
}

// Agents returns the agent panes of a workspace. An empty workspace returns
// the agents of every workspace.
func (c *Client) Agents(workspace string) ([]Agent, error) {
	out, err := c.run("agent", "list")
	if err != nil {
		return nil, err
	}
	var env struct {
		Result struct {
			Agents []Agent `json:"agents"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		return nil, fmt.Errorf("agent list: %w", err)
	}
	in := env.Result.Agents
	if workspace != "" {
		in = nil
		for _, a := range env.Result.Agents {
			if a.Workspace == workspace {
				in = append(in, a)
			}
		}
	}
	// A label read that fails leaves the ids in place. A picker with weaker
	// names is better than no picker.
	tabs, err := c.tabLabels()
	if err != nil {
		return in, nil
	}
	spaces, err := c.workspaceLabels()
	if err != nil {
		spaces = map[string]string{}
	}
	for i := range in {
		in[i].TabLabel = tabs[in[i].TabID]
		in[i].WorkspaceLabel = spaces[in[i].Workspace]
	}
	return in, nil
}

// tabLabels maps a tab id to the label herdr shows in the tab bar.
func (c *Client) tabLabels() (map[string]string, error) {
	out, err := c.run("tab", "list")
	if err != nil {
		return nil, err
	}
	var env struct {
		Result struct {
			Tabs []struct {
				TabID string `json:"tab_id"`
				Label string `json:"label"`
			} `json:"tabs"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		return nil, fmt.Errorf("tab list: %w", err)
	}
	m := make(map[string]string, len(env.Result.Tabs))
	for _, tb := range env.Result.Tabs {
		m[tb.TabID] = tb.Label
	}
	return m, nil
}

// workspaceLabels maps a workspace id to the name herdr shows for it.
func (c *Client) workspaceLabels() (map[string]string, error) {
	out, err := c.run("workspace", "list")
	if err != nil {
		return nil, err
	}
	var env struct {
		Result struct {
			Workspaces []struct {
				ID    string `json:"workspace_id"`
				Label string `json:"label"`
			} `json:"workspaces"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		return nil, fmt.Errorf("workspace list: %w", err)
	}
	m := make(map[string]string, len(env.Result.Workspaces))
	for _, w := range env.Result.Workspaces {
		m[w.ID] = w.Label
	}
	return m, nil
}

// Prompt submits text to an agent pane.
func (c *Client) Prompt(paneID, text string) error {
	_, err := c.run("agent", "prompt", paneID, text)
	return err
}
