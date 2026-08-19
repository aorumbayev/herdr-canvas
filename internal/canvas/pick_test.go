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
