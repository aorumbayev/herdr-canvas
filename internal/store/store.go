package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"herdr-canvas/internal/canvas"
)

// New returns a Store rooted at the central store directory.
func New() *Store { return &Store{} }

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

// dir returns the directory this Store reads and writes.
func (s *Store) dir() string {
	if s.Base == "" {
		return Dir()
	}
	return s.Base
}

// Path returns the on-disk path for a named diagram.
func (s *Store) Path(name string) string {
	return filepath.Join(s.dir(), name+".json")
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
	path := s.Path(d.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+d.Name+".*.tmp")
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

// ModTime returns the modification time of a named diagram. A diagram that is
// not in the store yields the zero time.
func (s *Store) ModTime(name string) (time.Time, error) {
	if err := validateName(name); err != nil {
		return time.Time{}, err
	}
	info, err := os.Stat(s.Path(name))
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return info.ModTime(), nil
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

// List returns the names of all diagrams in the store, sorted.
func (s *Store) List() ([]string, error) {
	dir := s.dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if name := strings.TrimSuffix(e.Name(), ".json"); name != e.Name() {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}
