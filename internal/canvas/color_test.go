package canvas

import (
	"strings"
	"testing"
)

func TestApplyBoxStoresColorAndFill(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 2, Color: "red", Fill: true}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	e := d.Elements[0]
	if e.Color != "red" || !e.Fill {
		t.Errorf("element = %+v, want red fill", e)
	}
}

func TestApplyCreateRejectsBadColor(t *testing.T) {
	d := &Diagram{}
	err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2, Color: "Red"})
	if err == nil {
		t.Fatal("want error for Red, got nil")
	}
	if !strings.Contains(err.Error(), `color "Red"`) {
		t.Errorf("error = %q", err.Error())
	}
	if len(d.Elements) != 0 {
		t.Fatal("diagram mutated on reject")
	}
}

func TestApplyCreateClearsDefaultColor(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(LineCmd{X1: 0, Y1: 0, X2: 1, Y2: 0, Color: "default"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if d.Elements[0].Color != "" {
		t.Errorf("color = %q, want empty", d.Elements[0].Color)
	}
}

func TestApplyColorCmdSetsAndClears(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(ColorCmd{ID: "b1", Color: "blue"}); err != nil {
		t.Fatalf("ColorCmd: %v", err)
	}
	if d.Elements[0].Color != "blue" {
		t.Errorf("color = %q, want blue", d.Elements[0].Color)
	}
	if err := d.Apply(ColorCmd{ID: "b1", Color: "default"}); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if d.Elements[0].Color != "" {
		t.Errorf("color = %q, want empty", d.Elements[0].Color)
	}
}

func TestApplyColorCmdRejectsUnknownID(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(ColorCmd{ID: "b9", Color: "red"}); err == nil {
		t.Fatal("want error for unknown id")
	}
}

func TestApplyFillCmdBoxOnly(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(LineCmd{X1: 0, Y1: 1, X2: 2, Y2: 1}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(FillCmd{ID: "b1", Fill: true}); err != nil {
		t.Fatalf("FillCmd box: %v", err)
	}
	if !d.Elements[0].Fill {
		t.Error("box fill not set")
	}
	err := d.Apply(FillCmd{ID: "l2", Fill: true})
	if err == nil {
		t.Fatal("want error for fill on line")
	}
	if !strings.Contains(err.Error(), "fill l2: not a box") {
		t.Errorf("error = %q", err.Error())
	}
}
