package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"herdr-canvas/internal/canvas"
)

func keyDelete() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyDelete}
}

func leftDownShift(x, y int) tea.MouseClickMsg {
	return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft, Mod: tea.ModShift}
}

func TestSelectToolKeyIsOne(t *testing.T) {
	m := editor(t)
	m = send(t, m, key("1"))
	if m.tool != toolSelect {
		t.Fatalf("tool = %v, want select", m.tool)
	}
}

func TestLetterToolKeysDoNotSwitch(t *testing.T) {
	m := editor(t)
	m.tool = toolSelect
	for _, k := range []string{"b", "l", "a", "t", "d", "m", "v", "x"} {
		m = send(t, m, key(k))
		if m.tool != toolSelect {
			t.Fatalf("key %q switched tool to %v", k, m.tool)
		}
	}
}

func TestSelectClickReplacesSelection(t *testing.T) {
	m := editor(t)
	for _, c := range []canvas.Command{
		canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2},
		canvas.BoxCmd{X1: 5, Y1: 0, X2: 7, Y2: 2},
	} {
		if err := m.d.Apply(c); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}
	m.tool = toolSelect
	m = send(t, m, leftDown(1, 1), leftUp(1, 1))
	if !m.selected["b1"] || m.selected["b2"] {
		t.Fatalf("first click selected %v, want only b1", m.selected)
	}
	m = send(t, m, leftDown(6, 1), leftUp(6, 1))
	if m.selected["b1"] || !m.selected["b2"] {
		t.Fatalf("click should replace, got %v", m.selected)
	}
}

func TestSelectClickWithoutDragCollapsesExistingMultiSelection(t *testing.T) {
	m := editor(t)
	for _, c := range []canvas.Command{
		canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2},
		canvas.BoxCmd{X1: 5, Y1: 0, X2: 7, Y2: 2},
	} {
		if err := m.d.Apply(c); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}
	m.tool = toolSelect
	m.selected = map[string]bool{"b1": true, "b2": true}
	m = send(t, m, leftDown(1, 1), leftUp(1, 1))
	if !m.selected["b1"] || m.selected["b2"] {
		t.Fatalf("plain click should collapse to b1, got %v", m.selected)
	}
}

func TestSelectShiftClickTogglesMembership(t *testing.T) {
	m := editor(t)
	for _, c := range []canvas.Command{
		canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2},
		canvas.BoxCmd{X1: 5, Y1: 0, X2: 7, Y2: 2},
	} {
		if err := m.d.Apply(c); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}
	m.tool = toolSelect
	m = send(t, m, leftDown(1, 1), leftUp(1, 1))
	m = send(t, m, leftDownShift(6, 1), leftUp(6, 1))
	if !m.selected["b1"] || !m.selected["b2"] {
		t.Fatalf("shift-click should add, got %v", m.selected)
	}
	m = send(t, m, leftDownShift(1, 1), leftUp(1, 1))
	if m.selected["b1"] || !m.selected["b2"] {
		t.Fatalf("shift-click should toggle off b1, got %v", m.selected)
	}
}

func TestSelectMarqueeReplaces(t *testing.T) {
	m := editor(t)
	for _, c := range []canvas.Command{
		canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2},
		canvas.BoxCmd{X1: 5, Y1: 0, X2: 7, Y2: 2},
		canvas.BoxCmd{X1: 20, Y1: 20, X2: 22, Y2: 22},
	} {
		if err := m.d.Apply(c); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}
	m.tool = toolSelect
	m.selected = map[string]bool{"b3": true}
	m = send(t, m, leftDown(8, 1), leftMove(1, 1), leftUp(1, 1))
	if !m.selected["b1"] || !m.selected["b2"] {
		t.Fatalf("marquee selected %v, want b1 and b2", m.selected)
	}
	if m.selected["b3"] {
		t.Fatalf("marquee should replace, not keep b3: %v", m.selected)
	}
}

func TestSelectShiftMarqueeAdds(t *testing.T) {
	m := editor(t)
	for _, c := range []canvas.Command{
		canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2},
		canvas.BoxCmd{X1: 5, Y1: 0, X2: 7, Y2: 2},
	} {
		if err := m.d.Apply(c); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}
	m.tool = toolSelect
	m.selected = map[string]bool{"b1": true}
	m = send(t, m, tea.MouseClickMsg{X: 8, Y: 1, Button: tea.MouseLeft, Mod: tea.ModShift},
		leftMove(5, 1), leftUp(5, 1))
	if !m.selected["b1"] || !m.selected["b2"] {
		t.Fatalf("shift-marquee should add b2, got %v", m.selected)
	}
}

func TestSelectMoveKeepsSelection(t *testing.T) {
	m := editor(t)
	for _, c := range []canvas.Command{
		canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2},
		canvas.BoxCmd{X1: 5, Y1: 0, X2: 7, Y2: 2},
	} {
		if err := m.d.Apply(c); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}
	m.tool = toolSelect
	m.selected = map[string]bool{"b1": true, "b2": true}
	m = send(t, m, leftDown(1, 1), leftMove(1, 3), leftUp(1, 3))
	if !m.selected["b1"] || !m.selected["b2"] {
		t.Fatalf("selection should stay after move, got %v", m.selected)
	}
	byID := map[string]canvas.Element{}
	for _, e := range m.d.Elements {
		byID[e.ID] = e
	}
	if byID["b1"].Y1 != 2 || byID["b2"].Y1 != 2 {
		t.Fatalf("both boxes should move by dy=2: b1=%+v b2=%+v", byID["b1"], byID["b2"])
	}
}

func TestSelectDeleteWithBothDeleteKeys(t *testing.T) {
	for _, del := range []tea.Msg{key("backspace"), keyDelete()} {
		m := editor(t)
		for _, c := range []canvas.Command{
			canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2},
			canvas.BoxCmd{X1: 5, Y1: 0, X2: 7, Y2: 2},
			canvas.TextCmd{X: 10, Y: 0, Text: "keep"},
		} {
			if err := m.d.Apply(c); err != nil {
				t.Fatalf("Apply: %v", err)
			}
		}
		m.tool = toolSelect
		m.selected = map[string]bool{"b1": true, "b2": true}
		m = send(t, m, del)
		if len(m.d.Elements) != 1 || m.d.Elements[0].ID != "t3" {
			t.Fatalf("after %T delete: %+v", del, m.d.Elements)
		}
		if len(m.selected) != 0 {
			t.Fatalf("selection should clear after delete")
		}
	}
}

func TestSelectXIsNeverDestructive(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	m.tool = toolSelect
	m.selected = map[string]bool{"b1": true}
	m = send(t, m, key("x"))
	if m.tool != toolSelect {
		t.Fatalf("tool = %v, want select", m.tool)
	}
	if len(m.d.Elements) != 1 {
		t.Fatalf("x deleted elements: %+v", m.d.Elements)
	}
	if !m.selected["b1"] {
		t.Fatalf("x cleared selection")
	}
}

func TestSelectUndoDeleteReselects(t *testing.T) {
	m := editor(t)
	m.tool = toolSelect
	m.apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2})
	m.apply(canvas.BoxCmd{X1: 5, Y1: 0, X2: 7, Y2: 2})
	m.selected = map[string]bool{"b1": true, "b2": true}
	m = send(t, m, keyDelete())
	if len(m.d.Elements) != 0 {
		t.Fatalf("want empty, got %+v", m.d.Elements)
	}
	m = send(t, m, ctrlZ())
	if len(m.d.Elements) != 2 {
		t.Fatalf("undo should restore both: %+v", m.d.Elements)
	}
	if !m.selected["b1"] || !m.selected["b2"] {
		t.Fatalf("undo delete should reselect, got %v", m.selected)
	}
}

func TestSelectFailedMoveKeepsSelection(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 5, Y1: 5, X2: 7, Y2: 7}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	m.tool = toolSelect
	m.selected = map[string]bool{"b1": true}
	m.selAct = selectMove
	m.anchor = [2]int{6, 6}
	m.cursor = [2]int{6, 0} // dy=-6 → Y1=-1, rejected
	m.commitSelect()
	if len(m.selected) != 1 || !m.selected["b1"] {
		t.Fatalf("failed move must keep selection, got %v", m.selected)
	}
	if m.d.Elements[0].Y1 != 5 {
		t.Fatalf("box should stay put, got %+v", m.d.Elements[0])
	}
}

func TestSelectMultiDeleteIsOneUndo(t *testing.T) {
	m := editor(t)
	for _, c := range []canvas.Command{
		canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2},
		canvas.BoxCmd{X1: 5, Y1: 0, X2: 7, Y2: 2},
	} {
		if err := m.d.Apply(c); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}
	m.tool = toolSelect
	m.selected = map[string]bool{"b1": true, "b2": true}
	m = send(t, m, keyDelete())
	if len(m.d.Elements) != 0 {
		t.Fatalf("want empty diagram, got %+v", m.d.Elements)
	}
	m = send(t, m, ctrlZ())
	if len(m.d.Elements) != 2 {
		t.Fatalf("one undo should restore both: %+v", m.d.Elements)
	}
}

func TestSelectMultiMoveIsOneUndoAndKeepsSelection(t *testing.T) {
	m := editor(t)
	for _, c := range []canvas.Command{
		canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2},
		canvas.BoxCmd{X1: 5, Y1: 0, X2: 7, Y2: 2},
	} {
		if err := m.d.Apply(c); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}
	m.tool = toolSelect
	m.selected = map[string]bool{"b1": true, "b2": true}
	m = send(t, m, leftDown(1, 1), leftMove(3, 1), leftUp(3, 1))
	if !m.selected["b1"] || !m.selected["b2"] {
		t.Fatalf("move should keep selection, got %v", m.selected)
	}
	m = send(t, m, ctrlZ())
	byID := map[string]canvas.Element{}
	for _, e := range m.d.Elements {
		byID[e.ID] = e
	}
	if byID["b1"].X1 != 0 || byID["b2"].X1 != 5 {
		t.Fatalf("one undo should restore both positions: b1=%+v b2=%+v", byID["b1"], byID["b2"])
	}
	if !m.selected["b1"] || !m.selected["b2"] {
		t.Fatalf("undo move should keep selection, got %v", m.selected)
	}
}

func TestSelectHighlightsCorners(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 1, Y1: 0, X2: 3, Y2: 2}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	m.tool = toolSelect
	m.selected = map[string]bool{"b1": true}
	row := []rune(canvasLine(t, m, 0))
	if len(row) < 4 || row[1] != '*' || row[3] != '*' {
		t.Fatalf("row 0 = %q, want * at selected box corners", string(row))
	}
}

func TestChromeHasSelChip(t *testing.T) {
	ch := layoutChrome(80, "demo", [2]int{0, 0}, "", false, toolSelect, true, true, "")
	if indexOf(ch.footer, "[1 sel]") < 0 {
		t.Fatalf("footer %q missing active sel chip", ch.footer)
	}
	short := layoutChrome(20, "demo", [2]int{0, 0}, "", false, toolSelect, false, false, "")
	if indexOf(short.footer, "[1]") < 0 && indexOf(short.footer, "1") < 0 {
		t.Fatalf("short footer %q missing 1/sel", short.footer)
	}
}

func TestSwitchToolPreservesSelection(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	m.tool = toolSelect
	m.selected = map[string]bool{"b1": true}
	m = send(t, m, key("2"))
	if m.tool != toolBox {
		t.Fatalf("tool = %v, want box", m.tool)
	}
	if !m.selected["b1"] {
		t.Fatalf("switching tools cleared selection")
	}
}

func TestNonSelectActionClearsSelection(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	m.tool = toolBox
	m.selected = map[string]bool{"b1": true}
	m = send(t, m, leftDown(10, 5), leftMove(12, 7), leftUp(12, 7))
	if len(m.selected) != 0 {
		t.Fatalf("starting a box should clear selection, got %v", m.selected)
	}
}

func TestClickEmptyClearsSelection(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	m.tool = toolSelect
	m.selected = map[string]bool{"b1": true}
	m = send(t, m, leftDown(15, 8), leftUp(15, 8))
	if len(m.selected) != 0 {
		t.Fatalf("empty click should clear, got %v", m.selected)
	}
}
