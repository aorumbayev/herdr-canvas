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
	h.push(before)
	d := sample(t, "a")
	got, ok := h.undo(d)
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
	h.push(empty)
	d := sample(t, "a")
	id := d.Elements[0].ID
	undone, _ := h.undo(d)
	got, ok := h.redo(undone)
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
	h.push(d)
	h.push(d)
	_, ok := h.undo(d)
	if !ok {
		t.Fatal("first undo")
	}
	if _, ok := h.undo(d); ok {
		t.Fatal("duplicate push created a second undo")
	}
}

func TestHistoryCapsAt50(t *testing.T) {
	h := history{}
	for i := 0; i < 51; i++ {
		d := &canvas.Diagram{Name: "demo", Next: i}
		h.push(d)
	}
	if len(h.undoStack) != 50 {
		t.Errorf("len = %d, want 50", len(h.undoStack))
	}
	got, _ := h.undo(&canvas.Diagram{Name: "demo"})
	if got.Next != 50 {
		t.Errorf("oldest kept Next = %d, want 50 (dropped 0)", got.Next)
	}
}

func TestHistoryCloneIsDeep(t *testing.T) {
	h := history{}
	d := sample(t, "a")
	h.push(d)
	d.Elements[0].Label = "mutated"
	got, _ := h.undo(d)
	if got.Elements[0].Label == "mutated" {
		t.Fatal("undo saw the later mutation")
	}
}

func TestHistoryPushClearsRedo(t *testing.T) {
	h := history{}
	h.push(&canvas.Diagram{Name: "demo"})
	d := sample(t, "a")
	if _, ok := h.undo(d); !ok {
		t.Fatal("undo")
	}
	if !h.canRedo() {
		t.Fatal("expected redo after undo")
	}
	h.push(sample(t, "b"))
	if h.canRedo() {
		t.Fatal("push after undo kept redo")
	}
}

func TestHistoryEmptyUndoRedoAreNoops(t *testing.T) {
	h := history{}
	d := sample(t, "a")
	if _, ok := h.undo(d); ok {
		t.Fatal("undo on empty")
	}
	if _, ok := h.redo(d); ok {
		t.Fatal("redo on empty")
	}
}

func TestHistoryDropsTheOldest(t *testing.T) {
	h := history{}
	for i := 0; i < 51; i++ {
		h.push(&canvas.Diagram{Name: "demo", Next: i})
	}
	var n int
	cur := &canvas.Diagram{Name: "demo"}
	for {
		got, ok := h.undo(cur)
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
