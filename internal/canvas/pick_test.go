package canvas

import "testing"

func TestElementAt(t *testing.T) {
	d := &Diagram{}
	for _, c := range []Command{
		BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2},
		TextCmd{X: 5, Y: 5, Text: "hi"},
		LineCmd{X1: 10, Y1: 10, X2: 12, Y2: 10},
		DrawCmd{Cells: []Cell{{X: 20, Y: 20, Ch: "#"}}},
	} {
		if err := d.Apply(c); err != nil {
			t.Fatalf("Apply %T: %v", c, err)
		}
	}
	cases := []struct {
		x, y int
		id   string
	}{
		{1, 1, "b1"}, {0, 0, "b1"}, {3, 3, ""},
		{6, 5, "t2"}, {5, 5, "t2"}, {5, 6, ""},
		{11, 10, "l3"}, {12, 10, "l3"}, {11, 11, ""},
		{20, 20, "f4"}, {21, 20, ""},
	}
	for _, c := range cases {
		e := d.ElementAt(c.x, c.y)
		got := ""
		if e != nil {
			got = e.ID
		}
		if got != c.id {
			t.Errorf("ElementAt(%d,%d) = %q, want %q", c.x, c.y, got, c.id)
		}
	}
}

func TestElementsInRect(t *testing.T) {
	d := &Diagram{}
	for _, c := range []Command{
		BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2},
		BoxCmd{X1: 10, Y1: 10, X2: 12, Y2: 12},
		TextCmd{X: 5, Y: 1, Text: "hi"},
	} {
		if err := d.Apply(c); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}
	got := d.ElementsInRect(1, 0, 6, 2)
	if len(got) != 2 {
		t.Fatalf("got %d elements, want 2", len(got))
	}
	ids := map[string]bool{got[0].ID: true, got[1].ID: true}
	if !ids["b1"] || !ids["t3"] {
		t.Errorf("ids = %v, want b1 and t3", ids)
	}
	if n := len(d.ElementsInRect(20, 20, 22, 22)); n != 0 {
		t.Errorf("empty rect hit %d elements", n)
	}
}

func TestElementBounds(t *testing.T) {
	e := Element{Type: Box, X1: 1, Y1: 2, X2: 4, Y2: 5}
	x1, y1, x2, y2, ok := e.Bounds()
	if !ok || x1 != 1 || y1 != 2 || x2 != 4 || y2 != 5 {
		t.Errorf("box bounds = %d,%d,%d,%d ok=%v", x1, y1, x2, y2, ok)
	}
	e = Element{Type: Text, X: 3, Y: 7, Text: "ab"}
	x1, y1, x2, y2, ok = e.Bounds()
	if !ok || x1 != 3 || y1 != 7 || x2 != 4 || y2 != 7 {
		t.Errorf("text bounds = %d,%d,%d,%d ok=%v", x1, y1, x2, y2, ok)
	}
}

func TestMultilineTextHitAndBounds(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(TextCmd{X: 5, Y: 5, Text: "hi\nthere"}); err != nil {
		t.Fatal(err)
	}
	e := d.Elements[0]
	x1, y1, x2, y2, ok := e.Bounds()
	if !ok || x1 != 5 || y1 != 5 || x2 != 9 || y2 != 6 {
		t.Errorf("bounds = %d,%d,%d,%d ok=%v, want 5,5,9,6", x1, y1, x2, y2, ok)
	}
	if got := d.ElementAt(5, 6); got == nil || got.ID != e.ID {
		t.Errorf("second line miss: %v", got)
	}
	if got := d.ElementAt(7, 5); got != nil {
		t.Errorf("gap after hi on first line hit %v", got)
	}
	if n := len(d.ElementsInRect(8, 6, 9, 6)); n != 1 {
		t.Errorf("rect on 're' hit %d, want 1", n)
	}
}
