package canvas

import (
	"strings"
	"testing"
)

func TestApplyBoxCommitsWithFreshID(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2, Label: "hi"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := d.Apply(BoxCmd{X1: 5, Y1: 5, X2: 7, Y2: 7}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(d.Elements) != 2 {
		t.Fatalf("got %d elements, want 2", len(d.Elements))
	}
	first := d.Elements[0]
	if first.ID != "b1" {
		t.Errorf("first ID = %q, want b1", first.ID)
	}
	if first.Type != Box || first.X1 != 0 || first.Y1 != 0 || first.X2 != 2 || first.Y2 != 2 || first.Label != "hi" {
		t.Errorf("first element = %+v, want box 0,0,2,2 label hi", first)
	}
	if d.Elements[1].ID != "b2" {
		t.Errorf("second ID = %q, want b2", d.Elements[1].ID)
	}
}

func TestApplyBoxRejectsInvertedCorner(t *testing.T) {
	d := &Diagram{}
	err := d.Apply(BoxCmd{X1: 2, Y1: 2, X2: 1, Y2: 3})
	if err == nil {
		t.Fatal("want error for x2<x1, got nil")
	}
	if len(d.Elements) != 0 {
		t.Fatalf("diagram mutated on reject: %+v", d.Elements)
	}
}

func TestApplyLineCommits(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(LineCmd{X1: 3, Y1: 3, X2: 1, Y2: 5, Arrow: ArrowEnd}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(d.Elements) != 1 {
		t.Fatalf("got %d elements, want 1", len(d.Elements))
	}
	e := d.Elements[0]
	if e.ID != "l1" || e.Type != Line || e.X1 != 3 || e.Y1 != 3 || e.X2 != 1 || e.Y2 != 5 || e.Arrow != ArrowEnd {
		t.Errorf("element = %+v, want line l1 3,3->1,5 arrow end", e)
	}
}

func TestApplyTextCommitsAndRejectsEmpty(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(TextCmd{X: 1, Y: 1, Text: "hello"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := d.Apply(TextCmd{X: 2, Y: 2, Text: ""}); err == nil {
		t.Fatal("want error for empty text, got nil")
	}
	if len(d.Elements) != 1 {
		t.Fatalf("got %d elements, want 1", len(d.Elements))
	}
	e := d.Elements[0]
	if e.ID != "t1" || e.Type != Text || e.X != 1 || e.Y != 1 || e.Text != "hello" {
		t.Errorf("element = %+v, want text t1 at 1,1 'hello'", e)
	}
}

func TestApplyDrawCommitsFreeform(t *testing.T) {
	d := &Diagram{}
	cells := []Cell{{X: 0, Y: 0, Ch: "#"}, {X: 1, Y: 0, Ch: "#"}}
	if err := d.Apply(DrawCmd{Cells: cells}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(d.Elements) != 1 {
		t.Fatalf("got %d elements, want 1", len(d.Elements))
	}
	e := d.Elements[0]
	if e.ID != "f1" || e.Type != Freeform || len(e.Cells) != 2 {
		t.Errorf("element = %+v, want freeform f1 with 2 cells", e)
	}
}

func TestApplyMoveTranslatesAndRejectsUnknownID(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply box: %v", err)
	}
	if err := d.Apply(MoveCmd{ID: "b1", DX: 3, DY: 4}); err != nil {
		t.Fatalf("Apply move: %v", err)
	}
	got := d.Elements[0]
	if got.X1 != 3 || got.Y1 != 4 || got.X2 != 5 || got.Y2 != 6 {
		t.Errorf("after move: %+v, want box at 3,4-5,6", got)
	}
	err := d.Apply(MoveCmd{ID: "nope", DX: 1, DY: 1})
	if err == nil {
		t.Fatal("want error for unknown id, got nil")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("error should echo the id, got %q", err.Error())
	}
}

func TestApplyDeleteRemovesAndNeverReusesID(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply box1: %v", err)
	}
	if err := d.Apply(DeleteCmd{ID: "b1"}); err != nil {
		t.Fatalf("Apply delete: %v", err)
	}
	if len(d.Elements) != 0 {
		t.Fatalf("want 0 elements after delete, got %d", len(d.Elements))
	}
	if err := d.Apply(DeleteCmd{ID: "b1"}); err == nil {
		t.Fatal("want error deleting unknown id, got nil")
	}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply box2: %v", err)
	}
	if d.Elements[0].ID != "b2" {
		t.Errorf("id after delete+recreate = %q, want b2 (never reused)", d.Elements[0].ID)
	}
}

func TestApplyLabelSetsAndRejectsUnknownID(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply box: %v", err)
	}
	if err := d.Apply(LabelCmd{ID: "b1", Label: "name"}); err != nil {
		t.Fatalf("Apply label: %v", err)
	}
	if d.Elements[0].Label != "name" {
		t.Errorf("label = %q, want name", d.Elements[0].Label)
	}
	if err := d.Apply(LabelCmd{ID: "missing", Label: "x"}); err == nil {
		t.Fatal("want error for unknown id, got nil")
	}
}

func TestApplyRejectsEmptyDraw(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(DrawCmd{}); err == nil {
		t.Fatal("want error for empty cell list, got nil")
	}
	if len(d.Elements) != 0 {
		t.Errorf("elements = %d, want 0", len(d.Elements))
	}
}

func TestApplyRejectsMoveOffCanvas(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply box: %v", err)
	}
	if err := d.Apply(TextCmd{X: 1, Y: 1, Text: "hi"}); err != nil {
		t.Fatalf("Apply text: %v", err)
	}
	if err := d.Apply(DrawCmd{Cells: []Cell{{X: 0, Y: 0, Ch: "#"}}}); err != nil {
		t.Fatalf("Apply draw: %v", err)
	}
	for _, id := range []string{"b1", "t2", "f3"} {
		if err := d.Apply(MoveCmd{ID: id, DX: -100, DY: -100}); err == nil {
			t.Errorf("move %s off canvas: want error, got nil", id)
		}
	}
	if got := d.Elements[0]; got.X1 != 0 || got.Y1 != 0 || got.X2 != 2 || got.Y2 != 2 {
		t.Errorf("box changed after rejected move: %+v", got)
	}
	if got := d.Elements[1]; got.X != 1 || got.Y != 1 {
		t.Errorf("text changed after rejected move: %+v", got)
	}
	if got := d.Elements[2].Cells[0]; got.X != 0 || got.Y != 0 {
		t.Errorf("freeform changed after rejected move: %+v", got)
	}
}

func TestApplyMoveTouchesOnlyTypeFields(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 1, Y1: 1, X2: 3, Y2: 3}); err != nil {
		t.Fatalf("Apply box: %v", err)
	}
	if err := d.Apply(MoveCmd{ID: "b1", DX: 2, DY: 1}); err != nil {
		t.Fatalf("Apply move: %v", err)
	}
	e := d.Elements[0]
	if e.X1 != 3 || e.Y1 != 2 || e.X2 != 5 || e.Y2 != 4 {
		t.Errorf("corners = (%d,%d)-(%d,%d), want (3,2)-(5,4)", e.X1, e.Y1, e.X2, e.Y2)
	}
	if e.X != 0 || e.Y != 0 {
		t.Errorf("box got x/y stamped: (%d,%d)", e.X, e.Y)
	}
}
