package tui

import (
	"strings"
	"testing"

	"herdr-canvas/internal/canvas"
)

func TestViewportCanvasPoint(t *testing.T) {
	v := viewport{origin: [2]int{5, 7}}
	cases := []struct {
		x, y, w, h int
		want       [2]int
		ok         bool
	}{
		{0, 0, 40, 12, [2]int{}, false},
		{0, 1, 40, 12, [2]int{5, 7}, true},
		{3, 4, 40, 12, [2]int{8, 10}, true},
	}
	for _, c := range cases {
		got, ok := v.canvasPoint(c.x, c.y, c.w, c.h)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("canvasPoint(%d,%d) = %v,%v, want %v,%v",
				c.x, c.y, got, ok, c.want, c.ok)
		}
	}
}

func TestViewportPanClampsToZero(t *testing.T) {
	v := viewport{origin: [2]int{1, 1}}
	v.pan(-3, -3)
	if v.origin != [2]int{0, 0} {
		t.Errorf("origin = %v, want [0 0]", v.origin)
	}
}

func TestViewportRecenterCentersSmallBox(t *testing.T) {
	d := &canvas.Diagram{}
	if err := d.Apply(canvas.BoxCmd{X1: 2, Y1: 2, X2: 4, Y2: 3}); err != nil {
		t.Fatal(err)
	}
	v := viewport{origin: [2]int{9, 9}}
	v.recenter(d.Elements, 10, 6)
	if v.origin[0] != 0 || v.origin[1] != 0 {
		t.Errorf("origin = %v, want [0 0]", v.origin)
	}
}

func TestViewportRecenterLargeBoxUsesMin(t *testing.T) {
	d := &canvas.Diagram{}
	if err := d.Apply(canvas.BoxCmd{X1: 80, Y1: 40, X2: 90, Y2: 50}); err != nil {
		t.Fatal(err)
	}
	v := viewport{origin: [2]int{0, 0}}
	v.recenter(d.Elements, 40, 10)
	if v.origin[0] != 80 || v.origin[1] != 40 {
		t.Errorf("origin = %v, want bbox min", v.origin)
	}
}

func TestViewportRecenterEmpty(t *testing.T) {
	v := viewport{origin: [2]int{3, 3}}
	v.recenter(nil, 40, 10)
	if v.origin != [2]int{0, 0} {
		t.Errorf("empty recenter origin=%v", v.origin)
	}
}

func TestViewportPaintIsOneToOne(t *testing.T) {
	d := &canvas.Diagram{}
	if err := d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatal(err)
	}
	got := (viewport{}).paint(d.Render(), 4, 3)
	if !strings.Contains(got, "┌") {
		t.Errorf("paint missing box: %q", got)
	}
}
