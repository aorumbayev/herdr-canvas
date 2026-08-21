package tui

import (
	"testing"

	"herdr-canvas/internal/canvas"
)

func sample(t *testing.T, label string) *canvas.Diagram {
	t.Helper()
	d := &canvas.Diagram{Name: "demo"}
	if err := d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2, Label: label}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return d
}

func TestHistoryUndoRestoresElementsAndNext(t *testing.T) {
	h := history{}
	before := &canvas.Diagram{Name: "demo"}
	h.push(before, nil)
	d := sample(t, "a")
	got, _, ok := h.undo(d, nil)
	if !ok {
		t.Fatal("undo = false")
	}
	if len(got.Elements) != 0 {
		t.Errorf("elements = %d, want 0", len(got.Elements))
	}
	if got.Next != 0 {
		t.Errorf("next = %d, want 0", got.Next)
	}
}

func TestHistoryRedoRestoresTheSameID(t *testing.T) {
	h := history{}
	empty := &canvas.Diagram{Name: "demo"}
	h.push(empty, nil)
	d := sample(t, "a")
	id := d.Elements[0].ID
	undone, _, _ := h.undo(d, nil)
	got, _, ok := h.redo(undone, nil)
	if !ok {
		t.Fatal("redo = false")
	}
	if got.Elements[0].ID != id {
		t.Errorf("id = %s, want %s", got.Elements[0].ID, id)
	}
}

func TestHistorySkipsEqualClone(t *testing.T) {
	h := history{}
	d := sample(t, "a")
	h.push(d, nil)
	h.push(d, nil)
	_, _, ok := h.undo(d, nil)
	if !ok {
		t.Fatal("first undo")
	}
	if _, _, ok := h.undo(d, nil); ok {
		t.Fatal("duplicate push created a second undo")
	}
}

func TestHistoryCapsAt50(t *testing.T) {
	h := history{}
	for i := 0; i < 51; i++ {
		d := &canvas.Diagram{Name: "demo", Next: i}
		h.push(d, nil)
	}
	if len(h.undoStack) != 50 {
		t.Errorf("len = %d, want 50", len(h.undoStack))
	}
	got, _, _ := h.undo(&canvas.Diagram{Name: "demo"}, nil)
	if got.Next != 50 {
		t.Errorf("oldest kept Next = %d, want 50 (dropped 0)", got.Next)
	}
}

func TestHistoryCloneIsDeep(t *testing.T) {
	h := history{}
	d := sample(t, "a")
	h.push(d, nil)
	d.Elements[0].Label = "mutated"
	got, _, _ := h.undo(d, nil)
	if got.Elements[0].Label == "mutated" {
		t.Fatal("undo saw the later mutation")
	}
}

func TestHistoryPushClearsRedo(t *testing.T) {
	h := history{}
	h.push(&canvas.Diagram{Name: "demo"}, nil)
	d := sample(t, "a")
	if _, _, ok := h.undo(d, nil); !ok {
		t.Fatal("undo")
	}
	if !h.canRedo() {
		t.Fatal("expected redo after undo")
	}
	h.push(sample(t, "b"), nil)
	if h.canRedo() {
		t.Fatal("push after undo kept redo")
	}
}

func TestHistoryEmptyUndoRedoAreNoops(t *testing.T) {
	h := history{}
	d := sample(t, "a")
	if _, _, ok := h.undo(d, nil); ok {
		t.Fatal("undo on empty")
	}
	if _, _, ok := h.redo(d, nil); ok {
		t.Fatal("redo on empty")
	}
}

func TestHistoryRestoresSelection(t *testing.T) {
	h := history{}
	before := &canvas.Diagram{Name: "demo"}
	h.push(before, map[string]bool{"b1": true})
	d := sample(t, "a")
	got, sel, ok := h.undo(d, nil)
	if !ok {
		t.Fatal("undo")
	}
	if len(got.Elements) != 0 {
		t.Fatalf("elements = %d", len(got.Elements))
	}
	if !sel["b1"] {
		t.Fatalf("sel = %v, want b1 from snapshot", sel)
	}
	_, sel, ok = h.redo(got, sel)
	if !ok {
		t.Fatal("redo")
	}
	if len(sel) != 0 {
		t.Fatalf("redo sel = %v, want empty current-at-undo", sel)
	}
}

func TestHistoryDropsTheOldest(t *testing.T) {
	h := history{}
	for i := 0; i < 51; i++ {
		h.push(&canvas.Diagram{Name: "demo", Next: i}, nil)
	}
	var n int
	cur := &canvas.Diagram{Name: "demo"}
	for {
		got, _, ok := h.undo(cur, nil)
		if !ok {
			break
		}
		n++
		cur = got
	}
	if n != 50 {
		t.Errorf("undos = %d, want 50", n)
	}
	if cur.Next != 1 {
		t.Errorf("oldest Next = %d, want 1", cur.Next)
	}
}
