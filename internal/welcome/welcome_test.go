package welcome

import (
	"os"
	"path/filepath"
	"testing"
)

func tempClient(t *testing.T) (*Client, string) {
	t.Helper()
	dir := t.TempDir()
	return &Client{LookupEnv: func(k string) string {
		if k == "XDG_STATE_HOME" {
			return dir
		}
		return ""
	}}, dir
}

func TestSeenFalseWhenAbsent(t *testing.T) {
	c, _ := tempClient(t)
	seen, err := c.Seen()
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}
	if seen {
		t.Fatal("want not seen for a fresh state dir")
	}
}

func TestMarkThenSeen(t *testing.T) {
	c, dir := tempClient(t)
	if err := c.Mark(); err != nil {
		t.Fatalf("Mark: %v", err)
	}
	seen, err := c.Seen()
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}
	if !seen {
		t.Fatal("want seen after Mark")
	}
	if _, err := os.Stat(filepath.Join(dir, "herdr-canvas", "welcome.json")); err != nil {
		t.Fatalf("state file: %v", err)
	}
}

func TestSeenErrorsOnCorruptFile(t *testing.T) {
	c, dir := tempClient(t)
	path := filepath.Join(dir, "herdr-canvas", "welcome.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Seen(); err == nil {
		t.Fatal("want error for corrupt state file")
	}
}
