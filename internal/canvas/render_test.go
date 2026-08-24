package canvas

import (
	"strings"
	"testing"
)

func TestRenderBox(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	g := d.Render()
	want := map[[2]int]rune{
		{0, 0}: '┌', {1, 0}: '─', {2, 0}: '┐',
		{0, 1}: '│', {2, 1}: '│',
		{0, 2}: '└', {1, 2}: '─', {2, 2}: '┘',
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
	wantPic := "┌──┐\n│hi│\n└──┘"
	if got := d.Render().String(); got != wantPic {
		t.Errorf("Render = %q, want %q", got, wantPic)
	}
	want := wantPic + "\n" +
		"b1 box 0,0-3,2\n" +
		"t2 text 1,1 \"hi\""
	if got := Export(d); got != want {
		t.Errorf("Export = %q, want %q", got, want)
	}
}

func TestExportLegendWithColorFill(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 2, Label: "hi", Color: "red", Fill: true}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(LineCmd{X1: 3, Y1: 1, X2: 10, Y2: 1, Color: "blue"}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(TextCmd{X: 1, Y: 1, Text: "hello", Color: "green"}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(DrawCmd{Cells: []Cell{{X: 0, Y: 0, Ch: "#"}}, Color: "cyan"}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(Export(d), "\n")
	if len(lines) < 5 {
		t.Fatalf("export lines = %d", len(lines))
	}
	want := []string{
		"b1 box 0,0-3,2 red fill \"hi\"",
		"l2 line 3,1-10,1 blue",
		"t3 text 1,1 green \"hello\"",
		"f4 draw 1 cyan",
	}
	for i, w := range want {
		if lines[len(lines)-len(want)+i] != w {
			t.Errorf("legend[%d] = %q, want %q", i, lines[len(lines)-len(want)+i], w)
		}
	}
}

func TestExportEmptyDiagram(t *testing.T) {
	if got := Export(&Diagram{}); got != "" {
		t.Errorf("Export = %q, want empty", got)
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
		{0, 0}: '─', {1, 0}: '─', {2, 0}: '─', {3, 0}: '►',
		{0, 1}: '▲', {0, 2}: '│', {0, 3}: '│',
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

func TestRenderMultilineText(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(TextCmd{X: 1, Y: 1, Text: "hi\nthere"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	g := d.Render()
	want := map[[2]int]rune{
		{1, 1}: 'h', {2, 1}: 'i',
		{1, 2}: 't', {2, 2}: 'h', {3, 2}: 'e', {4, 2}: 'r', {5, 2}: 'e',
	}
	if len(g) != len(want) {
		t.Fatalf("got %d cells, want %d: %v", len(g), len(want), g)
	}
	for k, v := range want {
		if got := g[k]; got != v {
			t.Errorf("cell %v = %q, want %q", k, got, v)
		}
	}
	if _, ok := g[[2]int{3, 1}]; ok {
		t.Errorf("newline occupied a cell: %v", g)
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
		{1, 1}: '─', {2, 1}: '┼', {3, 1}: '─', {4, 1}: '─',
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
	if got, want := d.Render().String(), "┌──┐\n│hi│\n└──┘"; got != want {
		t.Errorf("Render = %q, want %q", got, want)
	}
}

func TestRenderBoxLabelClippedToInnerWidth(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 2, Label: "hello"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got, want := d.Render().String(), "┌──┐\n│he│\n└──┘"; got != want {
		t.Errorf("Render = %q, want %q", got, want)
	}
}

func TestRenderBoxLabelSkippedWithoutInnerRow(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 1, Label: "hi"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got, want := d.Render().String(), "┌──┐\n└──┘"; got != want {
		t.Errorf("Render = %q, want %q", got, want)
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
	if got := d.Render()[[2]int{0, 0}]; got != '┌' {
		t.Errorf("corner = %q, want '┌'", got)
	}
}

func TestGridWindowHasFixedHeightAndOrigin(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 4, Y1: 2, X2: 6, Y2: 4}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got := d.Render().Window(3, 1, 5, 5)
	want := "\n ┌─┐\n │ │\n └─┘\n"
	if got != want {
		t.Errorf("Window = %q, want %q", got, want)
	}
	if lines := strings.Count(got, "\n") + 1; lines != 5 {
		t.Errorf("lines = %d, want 5", lines)
	}
}

func TestRenderLineIsAnOrthogonalElbow(t *testing.T) {
	cases := []struct {
		name           string
		x1, y1, x2, y2 int
		arrow          Arrow
		want           string
	}{
		{
			name: "down then right",
			x2:   4, y2: 3,
			want: "│\n│\n│\n└────",
		},
		{
			name: "down then right with an end arrow",
			x2:   4, y2: 3, arrow: ArrowEnd,
			want: "│\n│\n│\n└───►",
		},
		{
			name: "up then left",
			x1:   4, y1: 3, x2: 0, y2: 0,
			want: "────┐\n    │\n    │\n    │",
		},
		{
			name: "up then right",
			y1:   3, x2: 3,
			want: "┌───\n│\n│\n│",
		},
		{
			name: "down then left",
			x1:   3, x2: 0, y2: 3,
			want: "   │\n   │\n   │\n───┘",
		},
		{
			name: "straight horizontal keeps one run",
			x2:   3,
			want: "────",
		},
		{
			name: "straight vertical keeps one run",
			y2:   2,
			want: "│\n│\n│",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := &Diagram{}
			if err := d.Apply(LineCmd{X1: c.x1, Y1: c.y1, X2: c.x2, Y2: c.y2, Arrow: c.arrow}); err != nil {
				t.Fatalf("Apply: %v", err)
			}
			if got := d.Render().String(); got != c.want {
				t.Errorf("Render =\n%s\nwant\n%s", got, c.want)
			}
		})
	}
}

func TestRenderElbowTeesIntoAnExistingLine(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(LineCmd{X1: 4, Y1: 0, X2: 4, Y2: 4}); err != nil {
		t.Fatalf("Apply vline: %v", err)
	}
	// The elbow runs down from (0,0) and ends on the vertical line at (4,2).
	if err := d.Apply(LineCmd{X1: 0, Y1: 0, X2: 4, Y2: 2}); err != nil {
		t.Fatalf("Apply elbow: %v", err)
	}
	g := d.Render()
	if got := g[[2]int{0, 1}]; got != '│' {
		t.Errorf("first leg cell = %q, want a vertical run", got)
	}
	if got := g[[2]int{0, 2}]; got != '└' {
		t.Errorf("corner = %q, want up-right corner", got)
	}
	if got := g[[2]int{4, 2}]; got != '┤' {
		t.Errorf("tee = %q, want a left tee", got)
	}
}
