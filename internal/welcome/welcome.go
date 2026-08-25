// Package welcome persists whether the first-run walkthrough has been seen.
package welcome

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type state struct {
	Seen bool `json:"seen"`
}

// Client holds injectable seams so tests point at a temp state dir.
type Client struct {
	LookupEnv func(string) string
	UserHome  func() (string, error)
	TempDir   func() string
}

// Default returns production seams.
func Default() *Client { return &Client{} }

// Seen reports whether the walkthrough has been dismissed for good.
func Seen() (bool, error) { return Default().Seen() }

// Mark records that the walkthrough should not auto-open again.
func Mark() error { return Default().Mark() }

// StatePath is the walkthrough flag file under XDG state.
func StatePath() string { return Default().statePath() }

func (c *Client) Seen() (bool, error) {
	b, err := os.ReadFile(c.statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	var st state
	if err := json.Unmarshal(b, &st); err != nil {
		return false, err
	}
	return st.Seen, nil
}

func (c *Client) Mark() error {
	path := c.statePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(state{Seen: true})
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".welcome.*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
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
	return filepath.Join(base, "herdr-canvas", "welcome.json")
}

func (c *Client) lookup(key string) string {
	if c.LookupEnv != nil {
		return c.LookupEnv(key)
	}
	return os.Getenv(key)
}

func (c *Client) home() (string, error) {
	if c.UserHome != nil {
		return c.UserHome()
	}
	return os.UserHomeDir()
}

func (c *Client) tempDir() string {
	if c.TempDir != nil {
		return c.TempDir()
	}
	return os.TempDir()
}
