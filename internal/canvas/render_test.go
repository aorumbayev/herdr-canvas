package canvas

import "testing"

func TestRenderBox(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	g := d.Render()
	want := map[[2]int]rune{
		{0, 0}: '+', {1, 0}: '-', {2, 0}: '+',
		{0, 1}: '|', {2, 1}: '|',
		{0, 2}: '+', {1, 2}: '-', {2, 2}: '+',
	}
	if len(g) != len(want) {
		t.Fatalf("got %d cells, want %d: %v", len(g), len(want), g)
	}
	for k, v := range want {
		if got := g[k]; got != v {
			t.Errorf("cell %v = %q, want %q", k, got, v)
		}
	}
}

func TestExport(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 2}); err != nil {
		t.Fatalf("Apply box: %v", err)
	}
	if err := d.Apply(TextCmd{X: 1, Y: 1, Text: "hi"}); err != nil {
		t.Fatalf("Apply text: %v", err)
	}
	want := "+--+\n|hi|\n+--+"
	if got := Export(d); got != want {
		t.Errorf("Export = %q, want %q", got, want)
	}
}

func TestRenderLineWithArrows(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(LineCmd{X1: 0, Y1: 0, X2: 3, Y2: 0, Arrow: ArrowEnd}); err != nil {
		t.Fatalf("Apply hline: %v", err)
	}
	if err := d.Apply(LineCmd{X1: 0, Y1: 1, X2: 0, Y2: 3, Arrow: ArrowStart}); err != nil {
		t.Fatalf("Apply vline: %v", err)
	}
	g := d.Render()
	want := map[[2]int]rune{
		{0, 0}: '-', {1, 0}: '-', {2, 0}: '-', {3, 0}: '>',
		{0, 1}: '^', {0, 2}: '|', {0, 3}: '|',
	}
	for k, v := range want {
		if got := g[k]; got != v {
			t.Errorf("cell %v = %q, want %q", k, got, v)
		}
	}
	if len(g) != len(want) {
		t.Fatalf("got %d cells, want %d: %v", len(g), len(want), g)
	}
}

func TestRenderTextFreeformAndZOrder(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply box: %v", err)
	}
	if err := d.Apply(TextCmd{X: 1, Y: 1, Text: "A"}); err != nil {
		t.Fatalf("Apply text: %v", err)
	}
	if err := d.Apply(DrawCmd{Cells: []Cell{{X: 2, Y: 1, Ch: "#"}}}); err != nil {
		t.Fatalf("Apply draw: %v", err)
	}
	g := d.Render()
	if got := g[[2]int{1, 1}]; got != 'A' {
		t.Errorf("text cell = %q, want 'A'", got)
	}
	if got := g[[2]int{2, 1}]; got != '#' {
		t.Errorf("freeform over box edge = %q, want '#' (later wins)", got)
	}
}

func TestRenderLineCrossesBoxEdge(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply box: %v", err)
	}
	if err := d.Apply(LineCmd{X1: 1, Y1: 1, X2: 4, Y2: 1}); err != nil {
		t.Fatalf("Apply line: %v", err)
	}
	g := d.Render()
	want := map[[2]int]rune{
		{1, 1}: '-', {2, 1}: '┼', {3, 1}: '-', {4, 1}: '-',
	}
	for k, v := range want {
		if got := g[k]; got != v {
			t.Errorf("cell %v = %q, want %q", k, got, v)
		}
	}
}

func TestRenderBoxLabel(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 2, Label: "hi"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got, want := Export(d), "+--+\n|hi|\n+--+"; got != want {
		t.Errorf("Export = %q, want %q", got, want)
	}
}

func TestRenderBoxLabelClippedToInnerWidth(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 2, Label: "hello"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got, want := Export(d), "+--+\n|he|\n+--+"; got != want {
		t.Errorf("Export = %q, want %q", got, want)
	}
}

func TestRenderBoxLabelSkippedWithoutInnerRow(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 1, Label: "hi"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got, want := Export(d), "+--+\n+--+"; got != want {
		t.Errorf("Export = %q, want %q", got, want)
	}
}

func TestRenderLineKeepsBoxCorner(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply box: %v", err)
	}
	if err := d.Apply(LineCmd{X1: 0, Y1: 0, X2: 4, Y2: 0}); err != nil {
		t.Fatalf("Apply line: %v", err)
	}
	if got := d.Render()[[2]int{0, 0}]; got != '+' {
		t.Errorf("corner = %q, want '+'", got)
	}
}
