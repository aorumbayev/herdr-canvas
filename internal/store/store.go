package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"herdr-canvas/internal/canvas"
)

// Dir returns the central store directory, honoring XDG_DATA_HOME and
// falling back to ~/.local/share.
func Dir() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "herdr-canvas")
}

// Store reads and writes diagrams in a base directory (the central store by
// default).
type Store struct {
	Base string
}

// Path returns the on-disk path for a named diagram.
func (s *Store) Path(name string) string {
	if s.Base == "" {
		return filepath.Join(Dir(), name+".json")
	}
	return filepath.Join(s.Base, name+".json")
}

// Save writes a diagram as <name>.json, creating the directory if needed.
func (s *Store) Save(d *canvas.Diagram) error {
	if err := validateName(d.Name); err != nil {
		return err
	}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.Path(d.Name)), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.Path(d.Name), append(b, '\n'), 0o644)
}

// Load reads a named diagram from the store.
func (s *Store) Load(name string) (*canvas.Diagram, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(s.Path(name))
	if err != nil {
		return nil, err
	}
	var d canvas.Diagram
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", name, err)
	}
	return &d, nil
}

// validateName rejects names that could escape the store directory.
func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("diagram name is empty")
	}
	if name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("invalid diagram name %q", name)
	}
	return nil
}
