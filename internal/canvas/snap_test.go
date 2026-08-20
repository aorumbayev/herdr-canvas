package canvas

import "testing"

func TestJunctionCrossAndTee(t *testing.T) {
	cases := []struct {
		existing, inc rune
		cross         bool
		sdcol, sdrow  int
		want          rune
	}{
		{'│', '─', true, 0, 0, '┼'},
		{'─', '│', true, 0, 0, '┼'},
		{'│', '─', false, -1, 0, '┤'}, // horizontal ends on vertical, stub left
		{'│', '─', false, 1, 0, '├'},  // horizontal starts on vertical, stub right
		{'─', '│', false, 0, -1, '┴'}, // vertical ends on horizontal, stub up
		{'─', '│', false, 0, 1, '┬'},  // vertical starts on horizontal, stub down
	}
	for _, c := range cases {
		if got := junction(c.existing, c.inc, c.cross, c.sdcol, c.sdrow); got != c.want {
			t.Errorf("junction(%q,%q,cross=%v,sd=%d,%d) = %q, want %q",
				c.existing, c.inc, c.cross, c.sdcol, c.sdrow, got, c.want)
		}
	}
}

func TestJunctionKeepsBoxCorner(t *testing.T) {
	for _, existing := range []rune{'┌', '┐', '└', '┘'} {
		if got := junction(existing, '─', true, 0, 0); got != existing {
			t.Errorf("junction(%q,'─') = %q, want %q", existing, got, existing)
		}
		if got := junction(existing, '│', false, 0, 1); got != existing {
			t.Errorf("junction(%q,'│') = %q, want %q", existing, got, existing)
		}
	}
}
