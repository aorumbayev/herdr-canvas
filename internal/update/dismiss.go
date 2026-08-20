package update

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type dismissState struct {
	Dismissed string `json:"dismissed"`
}

// StatePath is XDG state, never store.Dir().
func StatePath() string {
	return Default().statePath()
}

func (c *Client) statePath() string {
	base := c.lookup("XDG_STATE_HOME")
	if base == "" {
		home, err := c.home()
		if err != nil || home == "" {
			home = c.tempDir()
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "herdr-canvas", "update.json")
}

// DismissedVersion is the last 0.x.y the person hid. Empty means none.
func DismissedVersion() (string, error) {
	return Default().DismissedVersion()
}

func (c *Client) DismissedVersion() (string, error) {
	b, err := os.ReadFile(c.statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var st dismissState
	if err := json.Unmarshal(b, &st); err != nil {
		return "", err
	}
	return st.Dismissed, nil
}

// Dismiss records latest so the TUI hides that tag until a newer one appears.
func Dismiss(latest string) error {
	return Default().Dismiss(latest)
}

func (c *Client) Dismiss(latest string) error {
	path := c.statePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(dismissState{Dismissed: latest})
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".update.*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// Hidden reports whether latest should stay off the TUI notice.
func Hidden(latest string) (bool, error) {
	return Default().Hidden(latest)
}

func (c *Client) Hidden(latest string) (bool, error) {
	d, err := c.DismissedVersion()
	if err != nil {
		return false, err
	}
	return d != "" && d == latest, nil
}
