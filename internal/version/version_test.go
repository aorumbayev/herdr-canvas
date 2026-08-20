package version

import "testing"

func TestDefaultVersionIsDev(t *testing.T) {
	if Version != "dev" {
		t.Fatalf("Version = %q, want dev", Version)
	}
}

func TestIsRelease(t *testing.T) {
	t.Cleanup(func() { Version = "dev" })
	cases := []struct {
		v    string
		want bool
	}{
		{"0.1.0", true},
		{"0.0.0", true},
		{"0.12.34", true},
		{"dev", false},
		{"", false},
		{"v0.1.0", false},
		{"1.0.0", false},
		{"0.1", false},
		{"0.1.0-rc.1", false},
	}
	for _, c := range cases {
		Version = c.v
		if got := IsRelease(); got != c.want {
			t.Errorf("IsRelease(%q) = %v, want %v", c.v, got, c.want)
		}
	}
}
