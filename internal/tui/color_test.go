package tui

import (
	"strings"
	"testing"

	"herdr-canvas/internal/canvas"
)

func TestBrushAppliedOnBoxCommit(t *testing.T) {
	m := editor(t)
	m.brushColor = "red"
	m.brushFill = true
	m = send(t, m, leftDown(5, 3), leftUp(8, 5))
	if len(m.d.Elements) != 1 {
		t.Fatalf("elements = %d", len(m.d.Elements))
	}
	e := m.d.Elements[0]
	if e.Color != "red" || !e.Fill {
		t.Errorf("element = %+v, want red fill", e)
	}
}

func TestPaletteSetsBrushColor(t *testing.T) {
	m := editor(t)
	m = send(t, m, key("c"), key("1"))
	if m.brushColor != "red" || m.phase != phaseEdit {
		t.Errorf("brush = %q phase = %v, want red edit", m.brushColor, m.phase)
	}
}

func TestPaletteEscLeavesBrushUnchanged(t *testing.T) {
	m := editor(t)
	m.brushColor = "blue"
	m = send(t, m, key("c"), key("esc"))
	if m.brushColor != "blue" || m.phase != phaseEdit {
		t.Errorf("brush = %q phase = %v", m.brushColor, m.phase)
	}
}

func TestPaletteOverlayKeepsCanvasAndClickPicks(t *testing.T) {
	m := editor(t)
	m.apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2})
	m = send(t, m, key("c"))
	if m.phase != phasePalette {
		t.Fatalf("phase = %v, want palette", m.phase)
	}
	view := screen(m)
	if !strings.Contains(view, "0 default") {
		t.Fatalf("view missing palette: %q", view)
	}
	if !strings.Contains(view, "┌") {
		t.Fatalf("view dropped the canvas: %q", view)
	}
	m = send(t, m, leftDown(0, 1), leftUp(0, 1))
	if m.brushColor != "" {
		t.Errorf("click default: brush = %q", m.brushColor)
	}
	m = send(t, m, key("c"))
	line := layoutPaletteLine(m.width)
	idx := strings.Index(line, "1 red")
	if idx < 0 {
		t.Fatal("palette line has no 1 red")
	}
	m = send(t, m, leftDown(idx, 1), leftUp(idx, 1))
	if m.brushColor != "red" || m.phase != phaseEdit {
		t.Errorf("click red: brush = %q phase = %v", m.brushColor, m.phase)
	}
}

func TestFillKeyTogglesBrushWithoutSelection(t *testing.T) {
	m := editor(t)
	m = send(t, m, key("f"))
	if !m.brushFill {
		t.Fatal("brushFill should be true after f")
	}
	m = send(t, m, key("f"))
	if m.brushFill {
		t.Fatal("brushFill should be false after second f")
	}
}

func TestFillKeyFillsSelectedBoxes(t *testing.T) {
	m := editor(t)
	m.apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2})
	m.apply(canvas.BoxCmd{X1: 5, Y1: 0, X2: 7, Y2: 2})
	m.selected = map[string]bool{"b1": true, "b2": true}
	m = send(t, m, key("f"))
	if !m.d.Elements[0].Fill || !m.d.Elements[1].Fill {
		t.Errorf("boxes = %+v %+v, want filled", m.d.Elements[0], m.d.Elements[1])
	}
}

func TestChromeHeaderShowsBrush(t *testing.T) {
	ch := layoutChrome(80, "demo", [2]int{4, 2}, "red", true, toolBox, true, true, "")
	if !strings.Contains(ch.header, "red fill") {
		t.Errorf("header = %q, want red fill", ch.header)
	}
}
