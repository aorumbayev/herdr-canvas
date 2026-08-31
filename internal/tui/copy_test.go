package tui

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"

	"herdr-canvas/internal/canvas"
)

func ctrlC() tea.KeyPressMsg { return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl} }

func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestCtrlCDoesNotQuitFromAnyPhase(t *testing.T) {
	for _, p := range []phase{phaseEdit, phasePick, phasePalette, phaseHelp} {
		m := editor(t)
		m.phase = p
		_, cmd := m.Update(ctrlC())
		if quits(cmd) {
			t.Errorf("ctrl+c quit from phase %v", p)
		}
	}
}

func TestQStillQuitsFromEveryPhase(t *testing.T) {
	for _, p := range []phase{phaseEdit, phasePick, phasePalette, phaseHelp} {
		m := editor(t)
		m.phase = p
		_, cmd := m.Update(key("q"))
		if !quits(cmd) {
			t.Errorf("q did not quit from phase %v", p)
		}
	}
}

func TestCopyPasteRoundTripKeepsShapeColorAndFill(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 2, Y1: 1, X2: 6, Y2: 4, Label: "hi", Color: "red", Fill: true}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	m.tool = toolSelect
	m.selected = map[string]bool{"b1": true}
	m.cursor = [2]int{10, 5}

	m = send(t, m, ctrlC(), ctrlV())

	if len(m.d.Elements) != 2 {
		t.Fatalf("elements = %d, want 2", len(m.d.Elements))
	}
	got := m.d.Elements[1]
	want := canvas.Element{
		ID: "b2", Type: canvas.Box,
		X1: 10, Y1: 5, X2: 14, Y2: 8,
		Label: "hi", Color: "red", Fill: true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pasted = %+v\nwant   = %+v", got, want)
	}
}

func TestPasteRelinksAnEdgeBetweenTheCopiedBoxes(t *testing.T) {
	m := editor(t)
	for _, c := range []canvas.Command{
		canvas.BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 2},
		canvas.BoxCmd{X1: 8, Y1: 0, X2: 11, Y2: 2},
		canvas.EdgeCmd{From: "b1", To: "b2", Arrow: canvas.ArrowEnd, Label: "to"},
	} {
		if err := m.d.Apply(c); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}
	m.tool = toolSelect
	m.cursor = [2]int{0, 10}
	m = send(t, m, ctrlA(), ctrlC(), ctrlV())

	if len(m.d.Elements) != 6 {
		t.Fatalf("elements = %d, want 6", len(m.d.Elements))
	}
	edge := m.d.Elements[5]
	if !edge.IsEdge() {
		t.Fatalf("last element = %+v, want an edge", edge)
	}
	if edge.From != "b4" || edge.To != "b5" {
		t.Fatalf("edge links %s->%s, want b4->b5", edge.From, edge.To)
	}
	if edge.Arrow != canvas.ArrowEnd || edge.Label != "to" {
		t.Fatalf("edge = %+v, want arrow end and label to", edge)
	}
}

func TestPasteDropsAnEdgeWithOnlyOneEndpointCopied(t *testing.T) {
	m := editor(t)
	for _, c := range []canvas.Command{
		canvas.BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 2},
		canvas.BoxCmd{X1: 8, Y1: 0, X2: 11, Y2: 2},
		canvas.EdgeCmd{From: "b1", To: "b2"},
	} {
		if err := m.d.Apply(c); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}
	m.tool = toolSelect
	m.selected = map[string]bool{"b1": true, "l3": true}
	m.cursor = [2]int{0, 10}
	m = send(t, m, ctrlC(), ctrlV())

	if len(m.d.Elements) != 4 {
		t.Fatalf("elements = %d, want 4 — one box copied, the edge dropped", len(m.d.Elements))
	}
	if got := m.d.Elements[3]; got.Type != canvas.Box {
		t.Fatalf("pasted = %+v, want a box", got)
	}
}

func TestPasteLandsTheBlockTopLeftAtTheCursorAndKeepsOffsets(t *testing.T) {
	m := editor(t)
	for _, c := range []canvas.Command{
		canvas.BoxCmd{X1: 4, Y1: 3, X2: 6, Y2: 5},
		canvas.TextCmd{X: 9, Y: 7, Text: "ab"},
		canvas.DrawCmd{Cells: []canvas.Cell{{X: 5, Y: 9, Ch: "x"}}},
	} {
		if err := m.d.Apply(c); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}
	m.tool = toolSelect
	m.cursor = [2]int{0, 0}
	m = send(t, m, ctrlA(), ctrlC(), ctrlV())

	box, text, draw := m.d.Elements[3], m.d.Elements[4], m.d.Elements[5]
	if box.X1 != 0 || box.Y1 != 0 || box.X2 != 2 || box.Y2 != 2 {
		t.Errorf("box = (%d,%d)-(%d,%d), want (0,0)-(2,2)", box.X1, box.Y1, box.X2, box.Y2)
	}
	if text.X != 5 || text.Y != 4 {
		t.Errorf("text = (%d,%d), want (5,4)", text.X, text.Y)
	}
	if len(draw.Cells) != 1 || draw.Cells[0].X != 1 || draw.Cells[0].Y != 6 {
		t.Errorf("draw cells = %+v, want one cell at (1,6)", draw.Cells)
	}
}

func TestPasteSelectsTheNewElementsOnly(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	m.tool = toolSelect
	m.selected = map[string]bool{"b1": true}
	m.cursor = [2]int{6, 6}
	m = send(t, m, ctrlC(), ctrlV())

	if !m.selected["b2"] || m.selected["b1"] || len(m.selected) != 1 {
		t.Fatalf("selected = %v, want only b2", m.selected)
	}
}

func TestOneUndoReversesAWholePaste(t *testing.T) {
	m := editor(t)
	for _, c := range []canvas.Command{
		canvas.BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 2},
		canvas.BoxCmd{X1: 8, Y1: 0, X2: 11, Y2: 2},
		canvas.EdgeCmd{From: "b1", To: "b2"},
	} {
		if err := m.d.Apply(c); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}
	m.tool = toolSelect
	m.cursor = [2]int{0, 10}
	m = send(t, m, ctrlA(), ctrlC(), ctrlV())
	if len(m.d.Elements) != 6 {
		t.Fatalf("setup: elements = %d, want 6", len(m.d.Elements))
	}

	m = send(t, m, ctrlZ())

	if len(m.d.Elements) != 3 {
		t.Fatalf("after one undo elements = %d, want 3", len(m.d.Elements))
	}
}

func TestCopyBufferSurvivesADiagramSwitch(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2, Label: "keep"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	m.tool = toolSelect
	m.selected = map[string]bool{"b1": true}
	m = send(t, m, ctrlC())

	other := &canvas.Diagram{Name: "other"}
	if err := m.s.Save(other); err != nil {
		t.Fatalf("Save: %v", err)
	}
	m = send(t, m, key("o"))
	m = send(t, m, key("j"), key("enter"))
	if m.d.Name == "demo" {
		t.Fatalf("setup: still on demo")
	}
	m.cursor = [2]int{1, 1}

	m = send(t, m, ctrlV())

	if len(m.d.Elements) != 1 {
		t.Fatalf("elements on %s = %d, want 1", m.d.Name, len(m.d.Elements))
	}
	if got := m.d.Elements[0]; got.Label != "keep" || got.X1 != 1 || got.Y1 != 1 {
		t.Fatalf("pasted = %+v, want label keep at (1,1)", got)
	}
}

func TestCopyWithNothingSelectedAndPasteWithAnEmptyBufferDoNothing(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	m.tool = toolSelect

	m = send(t, m, ctrlC(), ctrlV())

	if len(m.d.Elements) != 1 {
		t.Fatalf("elements = %d, want 1", len(m.d.Elements))
	}
	if m.status != "" {
		t.Fatalf("status = %q, want empty", m.status)
	}
}

func TestCtrlCWhileTypingDoesNotCopyElements(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	m.selected = map[string]bool{"b1": true}
	m.tool = toolText
	m = send(t, m, leftDown(6, 6), key("h"))
	if !m.typing {
		t.Fatal("setup: expected typing")
	}

	m = send(t, m, ctrlC())

	if len(m.clip.elems) != 0 {
		t.Fatalf("copy buffer = %+v, want empty", m.clip.elems)
	}
	if !m.typing || m.textBuf != "h" {
		t.Fatalf("typing = %v textBuf = %q, want typing with h", m.typing, m.textBuf)
	}
}

func TestHelpDocumentsCopyAndPasteInBothWidths(t *testing.T) {
	for _, width := range []int{40, 80} {
		joined := ""
		for _, line := range helpLines(width) {
			joined += line + "\n"
		}
		for _, want := range []string{"ctrl+c", "ctrl+v"} {
			if indexOf(joined, want) < 0 {
				t.Errorf("help at width %d does not mention %s", width, want)
			}
		}
	}
}
