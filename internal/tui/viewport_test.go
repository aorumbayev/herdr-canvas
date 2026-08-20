package tui

import (
	"strings"
	"testing"

	"herdr-canvas/internal/canvas"
)

func TestViewportCanvasPointAt1xAnd2x(t *testing.T) {
	v := viewport{zoom: 1, origin: [2]int{5, 7}}
	cases := []struct {
		zoom       int
		x, y, w, h int
		want       [2]int
		ok         bool
	}{
		{1, 0, 0, 40, 12, [2]int{}, false},
		{1, 0, 1, 40, 12, [2]int{5, 7}, true},
		{1, 3, 4, 40, 12, [2]int{8, 10}, true},
		{1, 0, 10, 40, 12, [2]int{5, 16}, true},
		{1, 0, 11, 40, 12, [2]int{}, false},
		{1, 40, 1, 40, 12, [2]int{}, false},
		{2, 0, 1, 40, 12, [2]int{5, 7}, true},
		{2, 2, 3, 40, 12, [2]int{6, 8}, true},
		{2, 1, 1, 40, 12, [2]int{5, 7}, true},
	}
	for _, c := range cases {
		v.zoom = c.zoom
		got, ok := v.canvasPoint(c.x, c.y, c.w, c.h)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("zoom=%d canvasPoint(%d,%d) = %v,%v, want %v,%v",
				c.zoom, c.x, c.y, got, ok, c.want, c.ok)
		}
	}
}

func TestViewportPanClampsToZero(t *testing.T) {
	v := viewport{zoom: 1, origin: [2]int{1, 1}}
	v.pan(-3, -3)
	if v.origin != [2]int{0, 0} {
		t.Errorf("origin = %v, want [0 0]", v.origin)
	}
}

func TestViewportFitEmptyAndOffscreen(t *testing.T) {
	v := viewport{zoom: 2, origin: [2]int{9, 9}}
	v.fit(nil, 40, 10)
	if v.zoom != 1 || v.origin != [2]int{0, 0} {
		t.Errorf("empty fit zoom=%d origin=%v", v.zoom, v.origin)
	}
	d := &canvas.Diagram{}
	if err := d.Apply(canvas.BoxCmd{X1: 80, Y1: 40, X2: 90, Y2: 50}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	v.fit(d.Elements, 40, 10)
	if v.zoom != 1 {
		t.Errorf("zoom = %d, want 1", v.zoom)
	}
	if v.origin[0] != 80 || v.origin[1] != 40 {
		t.Errorf("origin = %v, want [80 40] (bbox min; box larger than pane)", v.origin)
	}
}

func TestViewportIgnoresBadZoom(t *testing.T) {
	v := viewport{zoom: 1}
	v.setZoom(3)
	if v.zoom != 1 {
		t.Errorf("zoom = %d", v.zoom)
	}
	v.setZoom(2)
	if v.zoom != 2 {
		t.Errorf("zoom = %d, want 2", v.zoom)
	}
}

func TestViewportPaint2xDoublesGlyphs(t *testing.T) {
	d := &canvas.Diagram{}
	if err := d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 1}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	v := viewport{zoom: 2}
	got := v.paint(d.Render(), 8, 4)
	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Fatalf("lines = %d, want 4", len(lines))
	}
	if !strings.Contains(lines[0], "┌") || !strings.Contains(lines[0], "──") {
		t.Errorf("row0 = %q, want doubled ─ after ┌", lines[0])
	}
}
