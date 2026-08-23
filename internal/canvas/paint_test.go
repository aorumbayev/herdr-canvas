package canvas

import "testing"

func TestPaintBoxFillAndContrast(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 2, Label: "hi", Color: "red", Fill: true}); err != nil {
		t.Fatal(err)
	}
	p := d.Paint()
	if got := p[[2]int{0, 0}]; got.FG != "red" || got.BG != "" {
		t.Errorf("corner = %+v, want red fg no bg", got)
	}
	if got := p[[2]int{1, 1}]; got.BG != "red" || got.FG != "white" {
		t.Errorf("label cell = %+v, want white on red", got)
	}
	if got := p[[2]int{2, 1}]; got.BG != "red" || got.FG != "white" {
		t.Errorf("label cell = %+v, want white on red", got)
	}
}

func TestPaintDefaultFillUsesGray(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 2, Fill: true}); err != nil {
		t.Fatal(err)
	}
	if got := d.Paint()[[2]int{1, 1}]; got.BG != "8" || got.FG != "" {
		t.Errorf("interior = %+v, want empty fg on gray", got)
	}
}

func TestPaintLinePunchesFill(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 4, Y2: 2, Fill: true, Color: "blue"}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(LineCmd{X1: 1, Y1: 1, X2: 3, Y2: 1, Color: "red"}); err != nil {
		t.Fatal(err)
	}
	p := d.Paint()
	if got := p[[2]int{2, 1}]; got.FG != "red" || got.BG != "" {
		t.Errorf("line cell = %+v, want red fg empty bg", got)
	}
}

func TestPaintZOrderLaterWins(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(TextCmd{X: 1, Y: 1, Text: "A", Color: "red"}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(TextCmd{X: 1, Y: 1, Text: "B", Color: "blue"}); err != nil {
		t.Fatal(err)
	}
	if got := d.Paint()[[2]int{1, 1}]; got.Ch != 'B' || got.FG != "blue" {
		t.Errorf("cell = %+v, want B blue", got)
	}
}

func TestPaintLineGlyphsMatchRenderJunctions(t *testing.T) {
	crossing := &Diagram{}
	if err := crossing.Apply(LineCmd{X1: 2, Y1: 0, X2: 2, Y2: 4}); err != nil {
		t.Fatal(err)
	}
	if err := crossing.Apply(LineCmd{X1: 0, Y1: 2, X2: 4, Y2: 2}); err != nil {
		t.Fatal(err)
	}

	tee := &Diagram{}
	if err := tee.Apply(LineCmd{X1: 4, Y1: 0, X2: 4, Y2: 4}); err != nil {
		t.Fatal(err)
	}
	if err := tee.Apply(LineCmd{X1: 0, Y1: 0, X2: 4, Y2: 2}); err != nil {
		t.Fatal(err)
	}

	boxEdge := &Diagram{}
	if err := boxEdge.Apply(BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatal(err)
	}
	if err := boxEdge.Apply(LineCmd{X1: 1, Y1: 1, X2: 4, Y2: 1}); err != nil {
		t.Fatal(err)
	}

	for _, d := range []*Diagram{crossing, tee, boxEdge} {
		g := d.Render()
		p := d.Paint()
		for pt, ch := range g {
			if got := p[pt].Ch; got != ch {
				t.Errorf("cell %v paint %q, render %q", pt, got, ch)
			}
		}
	}
}

func TestPaintLightFillLabelIsBlack(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 2, Label: "x", Color: "yellow", Fill: true}); err != nil {
		t.Fatal(err)
	}
	if got := d.Paint()[[2]int{1, 1}]; got.FG != "black" {
		t.Errorf("label fg = %q, want black", got.FG)
	}
}
