package tui

import (
	"strings"
	"testing"

	"herdr-canvas/internal/canvas"
)

func TestViewportCanvasPointAtEachZoom(t *testing.T) {
	v := viewport{zoom: 10, origin: [2]int{5, 7}}
	cases := []struct {
		zoom       int
		x, y, w, h int
		want       [2]int
		ok         bool
	}{
		{10, 0, 0, 40, 12, [2]int{}, false},
		{10, 0, 1, 40, 12, [2]int{5, 7}, true},
		{10, 3, 4, 40, 12, [2]int{8, 10}, true},
		{20, 0, 1, 40, 12, [2]int{5, 7}, true},
		{20, 2, 3, 40, 12, [2]int{6, 8}, true},
		{5, 0, 1, 40, 12, [2]int{5, 7}, true},
		{5, 1, 2, 40, 12, [2]int{7, 9}, true},
		{15, 0, 1, 40, 12, [2]int{5, 7}, true},
		{15, 1, 1, 40, 12, [2]int{5, 7}, true},
		{15, 2, 1, 40, 12, [2]int{6, 7}, true},
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
	v := viewport{zoom: 10, origin: [2]int{1, 1}}
	v.pan(-3, -3)
	if v.origin != [2]int{0, 0} {
		t.Errorf("origin = %v, want [0 0]", v.origin)
	}
}

func TestViewportIgnoresBadZoom(t *testing.T) {
	v := viewport{zoom: 10}
	v.setZoom(3)
	if v.zoom != 10 {
		t.Errorf("zoom = %d", v.zoom)
	}
	v.setZoom(20)
	if v.zoom != 20 {
		t.Errorf("zoom = %d, want 20", v.zoom)
	}
}

func TestViewportZoomInOutStopsAtEnds(t *testing.T) {
	v := viewport{zoom: 10}
	v.zoomOut()
	if v.zoom != 5 {
		t.Errorf("zoom = %d, want 5", v.zoom)
	}
	v.zoomOut()
	if v.zoom != 5 {
		t.Errorf("zoom-out wrapped: %d", v.zoom)
	}
	v.zoomIn()
	v.zoomIn()
	v.zoomIn()
	if v.zoom != 20 {
		t.Errorf("zoom = %d, want 20", v.zoom)
	}
	v.zoomIn()
	if v.zoom != 20 {
		t.Errorf("zoom-in wrapped: %d", v.zoom)
	}
}

func TestViewportRecenterCentersSmallBox(t *testing.T) {
	d := &canvas.Diagram{}
	if err := d.Apply(canvas.BoxCmd{X1: 2, Y1: 2, X2: 4, Y2: 3}); err != nil {
		t.Fatal(err)
	}
	v := viewport{zoom: 10, origin: [2]int{9, 9}}
	v.recenter(d.Elements, 10, 6)
	if v.zoom != 10 {
		t.Errorf("recenter changed zoom: %d", v.zoom)
	}
	if v.origin[0] != 0 || v.origin[1] != 0 {
		t.Errorf("origin = %v, want [0 0]", v.origin)
	}
}

func TestViewportRecenterLargeBoxUsesMin(t *testing.T) {
	d := &canvas.Diagram{}
	if err := d.Apply(canvas.BoxCmd{X1: 80, Y1: 40, X2: 90, Y2: 50}); err != nil {
		t.Fatal(err)
	}
	v := viewport{zoom: 20, origin: [2]int{0, 0}}
	v.recenter(d.Elements, 40, 10)
	if v.zoom != 20 {
		t.Errorf("zoom = %d, want 20 (keep zoom)", v.zoom)
	}
	if v.origin[0] != 80 || v.origin[1] != 40 {
		t.Errorf("origin = %v, want bbox min", v.origin)
	}
}

func TestViewportRecenterEmpty(t *testing.T) {
	v := viewport{zoom: 15, origin: [2]int{3, 3}}
	v.recenter(nil, 40, 10)
	if v.zoom != 15 || v.origin != [2]int{0, 0} {
		t.Errorf("empty recenter zoom=%d origin=%v", v.zoom, v.origin)
	}
}

func TestViewportPaint05PicksNonSpace(t *testing.T) {
	d := &canvas.Diagram{}
	if err := d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatal(err)
	}
	v := viewport{zoom: 5}
	got := v.paint(d.Render(), 4, 3)
	if !strings.Contains(got, "┌") {
		t.Errorf("0.5× paint missing box glyph: %q", got)
	}
}

func TestViewportPaint2xDoublesGlyphs(t *testing.T) {
	d := &canvas.Diagram{}
	if err := d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 1}); err != nil {
		t.Fatal(err)
	}
	v := viewport{zoom: 20}
	got := v.paint(d.Render(), 8, 4)
	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Fatalf("lines = %d, want 4", len(lines))
	}
	if !strings.Contains(lines[0], "┌") || !strings.Contains(lines[0], "──") {
		t.Errorf("row0 = %q, want doubled ─ after ┌", lines[0])
	}
}
