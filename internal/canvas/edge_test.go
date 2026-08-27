package canvas

import (
	"encoding/json"
	"strings"
	"testing"
)

func box(x1, y1, x2, y2 int) Element {
	return Element{Type: Box, X1: x1, Y1: y1, X2: x2, Y2: y2}
}

func TestEdgeEndpoints(t *testing.T) {
	cases := []struct {
		name           string
		a, b           Element
		x1, y1, x2, y2 int
		vertical       bool
	}{
		{
			name: "below attaches bottom to top",
			a:    box(0, 4, 4, 6), b: box(0, 10, 4, 12),
			x1: 2, y1: 6, x2: 2, y2: 10, vertical: true,
		},
		{
			name: "above attaches top to bottom",
			a:    box(0, 10, 4, 12), b: box(0, 4, 4, 6),
			x1: 2, y1: 10, x2: 2, y2: 6, vertical: true,
		},
		{
			name: "right attaches right to left",
			a:    box(0, 0, 4, 2), b: box(10, 0, 14, 2),
			x1: 4, y1: 1, x2: 10, y2: 1,
		},
		{
			name: "left attaches left to right",
			a:    box(10, 0, 14, 2), b: box(0, 0, 4, 2),
			x1: 10, y1: 1, x2: 4, y2: 1,
		},
		{
			name: "wider apart sideways than down stays sideways",
			a:    box(0, 0, 4, 2), b: box(20, 6, 24, 8),
			x1: 4, y1: 1, x2: 20, y2: 7,
		},
		{
			name: "overlapping boxes fall back to sideways",
			a:    box(0, 0, 4, 4), b: box(2, 2, 6, 6),
			x1: 4, y1: 3, x2: 2, y2: 3,
		},
		{
			name: "boxes in the same place still give one answer",
			a:    box(0, 0, 4, 2), b: box(0, 0, 4, 2),
			x1: 4, y1: 1, x2: 0, y2: 1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			x1, y1, x2, y2, vertical := EdgeEndpoints(c.a, c.b)
			if x1 != c.x1 || y1 != c.y1 || x2 != c.x2 || y2 != c.y2 {
				t.Errorf("EdgeEndpoints = %d,%d-%d,%d, want %d,%d-%d,%d",
					x1, y1, x2, y2, c.x1, c.y1, c.x2, c.y2)
			}
			if vertical != c.vertical {
				t.Errorf("vertical = %v, want %v", vertical, c.vertical)
			}
			again1, again2, again3, again4, againV := EdgeEndpoints(c.a, c.b)
			if again1 != x1 || again2 != y1 || again3 != x2 || again4 != y2 || againV != vertical {
				t.Errorf("EdgeEndpoints is not deterministic")
			}
		})
	}
}

func TestEdgeCreatesLineWithDerivedEndpoints(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 4, Y2: 2}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 8, X2: 4, Y2: 10}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(EdgeCmd{From: "b1", To: "b2", Label: "retry", Arrow: ArrowEnd}); err != nil {
		t.Fatalf("EdgeCmd: %v", err)
	}
	e := d.Elements[2]
	if !e.IsEdge() {
		t.Fatalf("element %+v is not an edge", e)
	}
	if e.ID != "l3" || e.Type != Line {
		t.Errorf("edge = %s %s, want l3 line", e.ID, e.Type)
	}
	if e.X1 != 2 || e.Y1 != 2 || e.X2 != 2 || e.Y2 != 8 {
		t.Errorf("endpoints = %d,%d-%d,%d, want 2,2-2,8", e.X1, e.Y1, e.X2, e.Y2)
	}
	if e.Label != "retry" || e.Arrow != ArrowEnd {
		t.Errorf("label/arrow = %q/%q", e.Label, e.Arrow)
	}
}

func TestEdgeRejects(t *testing.T) {
	cases := []struct {
		name string
		cmd  EdgeCmd
		want string
	}{
		{"unknown from", EdgeCmd{From: "b9", To: "b2"}, `unknown element id "b9"`},
		{"unknown to", EdgeCmd{From: "b1", To: "b9"}, `unknown element id "b9"`},
		{"end is not a box", EdgeCmd{From: "b1", To: "t3"}, "edge t3: not a box"},
		{"both ends are one box", EdgeCmd{From: "b1", To: "b1"}, "edge b1: an edge needs two different boxes"},
		{"unknown color", EdgeCmd{From: "b1", To: "b2", Color: "puce"}, `edge: color "puce"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := &Diagram{}
			if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 4, Y2: 2}); err != nil {
				t.Fatal(err)
			}
			if err := d.Apply(BoxCmd{X1: 0, Y1: 8, X2: 4, Y2: 10}); err != nil {
				t.Fatal(err)
			}
			if err := d.Apply(TextCmd{X: 0, Y: 20, Text: "hi"}); err != nil {
				t.Fatal(err)
			}
			err := d.Apply(c.cmd)
			if err == nil {
				t.Fatalf("Apply = nil, want error")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error = %q, want it to contain %q", err, c.want)
			}
			if len(d.Elements) != 3 {
				t.Errorf("rejected command changed the diagram: %d elements", len(d.Elements))
			}
		})
	}
}

func TestUnedgeRemovesOnlyTheEdge(t *testing.T) {
	d := edgeDiagram(t)
	if err := d.Apply(UnedgeCmd{ID: "l3"}); err != nil {
		t.Fatalf("UnedgeCmd: %v", err)
	}
	if len(d.Elements) != 2 {
		t.Fatalf("elements = %d, want 2", len(d.Elements))
	}
	if err := d.Apply(UnedgeCmd{ID: "b1"}); err == nil {
		t.Error("unedge on a box = nil, want error")
	}
}

func TestMoveRederivesEdgeEndpoints(t *testing.T) {
	d := edgeDiagram(t)
	if err := d.Apply(MoveCmd{ID: "b2", DX: 10, DY: 0}); err != nil {
		t.Fatalf("MoveCmd: %v", err)
	}
	e := d.Elements[2]
	wantX1, wantY1, wantX2, wantY2, _ := EdgeEndpoints(d.Elements[0], d.Elements[1])
	if e.X1 != wantX1 || e.Y1 != wantY1 || e.X2 != wantX2 || e.Y2 != wantY2 {
		t.Errorf("endpoints = %d,%d-%d,%d, want %d,%d-%d,%d",
			e.X1, e.Y1, e.X2, e.Y2, wantX1, wantY1, wantX2, wantY2)
	}
}

func TestMoveBothBoxesDerivesFromFinalPositions(t *testing.T) {
	d := edgeDiagram(t)
	// The editor commits a multi-box move as one MoveCmd per box.
	for _, id := range []string{"b1", "b2"} {
		if err := d.Apply(MoveCmd{ID: id, DX: 3, DY: 5}); err != nil {
			t.Fatalf("MoveCmd %s: %v", id, err)
		}
	}
	e := d.Elements[2]
	if e.X1 != 5 || e.Y1 != 7 || e.X2 != 5 || e.Y2 != 13 {
		t.Errorf("endpoints = %d,%d-%d,%d, want 5,7-5,13", e.X1, e.Y1, e.X2, e.Y2)
	}
}

func TestMoveAnEdgeAloneKeepsItAttached(t *testing.T) {
	d := edgeDiagram(t)
	before := d.Elements[2]
	if err := d.Apply(MoveCmd{ID: "l3", DX: 4, DY: 4}); err != nil {
		t.Fatalf("MoveCmd: %v", err)
	}
	if got := d.Elements[2]; got.X1 != before.X1 || got.Y1 != before.Y1 || got.X2 != before.X2 || got.Y2 != before.Y2 {
		t.Errorf("endpoints moved to %d,%d-%d,%d, want %d,%d-%d,%d",
			got.X1, got.Y1, got.X2, got.Y2, before.X1, before.Y1, before.X2, before.Y2)
	}
}

func TestDeleteBoxCascadesToItsEdges(t *testing.T) {
	d := edgeDiagram(t)
	if err := d.Apply(DeleteCmd{ID: "b1"}); err != nil {
		t.Fatalf("DeleteCmd: %v", err)
	}
	if len(d.Elements) != 1 || d.Elements[0].ID != "b2" {
		t.Fatalf("elements = %+v, want only b2", d.Elements)
	}
}

func TestDeleteLineLeavesOtherElements(t *testing.T) {
	d := edgeDiagram(t)
	if err := d.Apply(DeleteCmd{ID: "l3"}); err != nil {
		t.Fatalf("DeleteCmd: %v", err)
	}
	if len(d.Elements) != 2 {
		t.Fatalf("elements = %d, want 2", len(d.Elements))
	}
}

func TestRenderEdgeUsesDoubleGlyphs(t *testing.T) {
	d := edgeDiagram(t)
	g := d.Render()
	want := map[[2]int]rune{
		{2, 3}: '║', {2, 4}: '║', {2, 2}: '╥', {2, 8}: '▼',
	}
	for p, r := range want {
		if got := g[p]; got != r {
			t.Errorf("cell %v = %q, want %q", p, got, r)
		}
	}
}

func TestEdgeKeepsBothBoxesWholeAndPointsIntoTheTarget(t *testing.T) {
	cases := []struct {
		name      string
		from, to  Element
		arrowhead rune
		straight  bool
	}{
		{name: "target below and left", from: box(14, 0, 25, 2), to: box(0, 8, 11, 10), arrowhead: '▼'},
		{name: "target below and right", from: box(0, 0, 11, 2), to: box(16, 8, 27, 10), arrowhead: '▼'},
		{name: "target above and left", from: box(14, 8, 25, 10), to: box(0, 0, 11, 2), arrowhead: '▲'},
		{name: "target above and right", from: box(0, 8, 11, 10), to: box(16, 0, 27, 2), arrowhead: '▲'},
		{name: "target right and below", from: box(0, 0, 11, 2), to: box(20, 6, 31, 8), arrowhead: '►'},
		{name: "target left and above", from: box(20, 6, 31, 8), to: box(0, 0, 11, 2), arrowhead: '◄'},
		{
			name: "touching rows with overlapping columns",
			from: box(0, 0, 10, 2), to: box(6, 3, 16, 5),
			arrowhead: '▼', straight: true,
		},
		{
			name: "touching columns with overlapping rows",
			from: box(0, 0, 2, 10), to: box(3, 6, 5, 16),
			arrowhead: '►', straight: true,
		},
		{
			name: "one free row with overlapping columns",
			from: box(0, 0, 10, 2), to: box(6, 4, 16, 6),
			arrowhead: '▼', straight: true,
		},
		{
			name: "far apart with overlapping columns",
			from: box(0, 0, 10, 2), to: box(6, 12, 16, 14),
			arrowhead: '▼', straight: true,
		},
		{
			name: "far apart with overlapping rows",
			from: box(0, 0, 10, 8), to: box(30, 4, 40, 14),
			arrowhead: '►', straight: true,
		},
		{
			name: "diagonally adjacent, nowhere to run",
			from: box(0, 0, 10, 2), to: box(11, 3, 20, 5),
			arrowhead: '▼',
		},
		{
			name: "diagonally adjacent the other way",
			from: box(11, 3, 20, 5), to: box(0, 0, 10, 2),
			arrowhead: '▲',
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := &Diagram{}
			for _, b := range []Element{c.from, c.to} {
				if err := d.Apply(BoxCmd{X1: b.X1, Y1: b.Y1, X2: b.X2, Y2: b.Y2}); err != nil {
					t.Fatal(err)
				}
			}
			bare := d.Render()
			if err := d.Apply(EdgeCmd{From: "b1", To: "b2", Arrow: ArrowEnd}); err != nil {
				t.Fatal(err)
			}
			e := d.Elements[2]
			g := d.Render()
			ends := map[[2]int]bool{{e.X1, e.Y1}: true, {e.X2, e.Y2}: true}
			for _, b := range []Element{d.Elements[0], d.Elements[1]} {
				for _, p := range borderCells(b) {
					if ends[p] {
						continue
					}
					if g[p] != bare[p] {
						t.Errorf("border cell %v = %q, want %q: the edge runs over the box",
							p, g[p], bare[p])
					}
				}
			}
			if got := g[[2]int{e.X2, e.Y2}]; got != c.arrowhead {
				t.Errorf("arrowhead = %q, want %q", got, c.arrowhead)
			}
			pts := linePoints(e)
			if got := lastStep(pts); got != stepInto(d.Elements[1], [2]int{e.X2, e.Y2}) {
				t.Errorf("the last run arrives from %v, want it to point into the box", got)
			}
			if runs := straightRuns(pts); (runs == 1) != c.straight {
				t.Errorf("route has %d straight runs, want straight = %v", runs, c.straight)
			}
			assertChain(t, pts, [2]int{e.X1, e.Y1}, [2]int{e.X2, e.Y2})
		})
	}
}

func borderCells(b Element) [][2]int {
	var out [][2]int
	for x := b.X1; x <= b.X2; x++ {
		out = append(out, [2]int{x, b.Y1}, [2]int{x, b.Y2})
	}
	for y := b.Y1; y <= b.Y2; y++ {
		out = append(out, [2]int{b.X1, y}, [2]int{b.X2, y})
	}
	return out
}

// assertChain checks that the route is one unbroken walk from the first
// attachment cell to the second, with no cell off that walk and none repeated.
func assertChain(t *testing.T, pts [][2]int, start, end [2]int) {
	t.Helper()
	if len(pts) == 0 || pts[0] != start || pts[len(pts)-1] != end {
		t.Fatalf("route runs %v..%v, want %v..%v", pts[0], pts[len(pts)-1], start, end)
	}
	seen := map[[2]int]bool{pts[0]: true}
	for i := 1; i < len(pts); i++ {
		step := abs(pts[i][0]-pts[i-1][0]) + abs(pts[i][1]-pts[i-1][1])
		if step != 1 {
			t.Errorf("cell %v does not follow %v", pts[i], pts[i-1])
		}
		if seen[pts[i]] {
			t.Errorf("cell %v appears twice", pts[i])
		}
		seen[pts[i]] = true
	}
}

// straightRuns counts the straight legs of a route.
func straightRuns(pts [][2]int) int {
	if len(pts) < 3 {
		return 1
	}
	runs := 1
	for i := 2; i < len(pts); i++ {
		turnX := pts[i][0] != pts[i-1][0] && pts[i-1][0] != pts[i-2][0]
		turnY := pts[i][1] != pts[i-1][1] && pts[i-1][1] != pts[i-2][1]
		if !turnX && !turnY {
			runs++
		}
	}
	return runs
}

func lastStep(pts [][2]int) [2]int {
	last := len(pts) - 1
	return [2]int{sign(pts[last][0] - pts[last-1][0]), sign(pts[last][1] - pts[last-1][1])}
}

// stepInto is the direction that crosses the border cell p into the box.
func stepInto(b Element, p [2]int) [2]int {
	switch {
	case p[1] == b.Y1:
		return [2]int{0, 1}
	case p[1] == b.Y2:
		return [2]int{0, -1}
	case p[0] == b.X1:
		return [2]int{1, 0}
	default:
		return [2]int{-1, 0}
	}
}

func TestRenderSingleLineCrossingAnEdge(t *testing.T) {
	d := edgeDiagram(t)
	if err := d.Apply(LineCmd{X1: 0, Y1: 5, X2: 6, Y2: 5}); err != nil {
		t.Fatalf("LineCmd: %v", err)
	}
	if got := d.Render()[[2]int{2, 5}]; got != '╫' {
		t.Errorf("crossing cell = %q, want '╫'", got)
	}
}

func TestRenderSingleLineTeesIntoAnEdge(t *testing.T) {
	d := edgeDiagram(t)
	if err := d.Apply(LineCmd{X1: 8, Y1: 5, X2: 2, Y2: 5}); err != nil {
		t.Fatalf("LineCmd: %v", err)
	}
	// The stub points back the way the line came, so the tee opens right.
	if got := d.Render()[[2]int{2, 5}]; got != '╟' {
		t.Errorf("tee cell = %q, want '╟'", got)
	}
}

func TestRenderEdgeCrossingAnEdge(t *testing.T) {
	d := edgeDiagram(t)
	if err := d.Apply(BoxCmd{X1: 8, Y1: 4, X2: 12, Y2: 6}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 4, X2: 1, Y2: 6}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(EdgeCmd{From: "b5", To: "b4"}); err != nil {
		t.Fatalf("EdgeCmd: %v", err)
	}
	if got := d.Render()[[2]int{2, 5}]; got != '╬' {
		t.Errorf("crossing cell = %q, want '╬'", got)
	}
}

func TestEdgeLabelPaintsBesideAVerticalRun(t *testing.T) {
	d := edgeDiagram(t)
	if err := d.Apply(LabelCmd{ID: "l3", Label: "ok"}); err != nil {
		t.Fatal(err)
	}
	g := d.Render()
	if g[[2]int{3, 5}] != 'o' || g[[2]int{4, 5}] != 'k' {
		t.Errorf("label cells = %q%q, want \"ok\"", g[[2]int{3, 5}], g[[2]int{4, 5}])
	}
}

func TestEdgeLabelInsideAHorizontalRun(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 4, Y2: 2}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(BoxCmd{X1: 20, Y1: 0, X2: 24, Y2: 2}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(EdgeCmd{From: "b1", To: "b2", Label: "retry"}); err != nil {
		t.Fatal(err)
	}
	row := ""
	g := d.Render()
	for x := 4; x <= 20; x++ {
		row += string(g[[2]int{x, 1}])
	}
	if !strings.Contains(row, "retry") {
		t.Errorf("run = %q, want it to contain retry", row)
	}
}

func TestEdgeLabelSkippedWhenItDoesNotFit(t *testing.T) {
	d := &Diagram{}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 4, Y2: 2}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(BoxCmd{X1: 8, Y1: 0, X2: 12, Y2: 2}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(EdgeCmd{From: "b1", To: "b2", Label: "a very long label"}); err != nil {
		t.Fatal(err)
	}
	g := d.Render()
	for x := 5; x <= 7; x++ {
		if got := g[[2]int{x, 1}]; got != dblHoriz {
			t.Errorf("cell %d,1 = %q, want an unbroken run", x, got)
		}
	}
}

func TestLegendNamesEdgesAndArrows(t *testing.T) {
	d := edgeDiagram(t)
	if err := d.Apply(LabelCmd{ID: "l3", Label: "retry"}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(ColorCmd{ID: "l3", Color: "red"}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(LineCmd{X1: 0, Y1: 20, X2: 4, Y2: 20, Arrow: ArrowBoth}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(LineCmd{X1: 0, Y1: 22, X2: 4, Y2: 22}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(Export(d), "\n")
	want := []string{
		`l3 edge b1->b2 red arrow end "retry"`,
		"l4 line 0,20-4,20 arrow both",
		"l5 line 0,22-4,22",
	}
	for i, w := range want {
		got := lines[len(lines)-len(want)+i]
		if got != w {
			t.Errorf("legend[%d] = %q, want %q", i, got, w)
		}
	}
}

func TestOldDiagramJSONLoadsAndRendersUnchanged(t *testing.T) {
	const old = `{"name":"demo","elements":[` +
		`{"id":"b1","type":"box","x2":3,"y2":2,"label":"hi"},` +
		`{"id":"l2","type":"line","y1":4,"x2":6,"y2":4,"arrow":"end"}],"next":2}`
	var d Diagram
	if err := json.Unmarshal([]byte(old), &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	want := "┌──┐\n│hi│\n└──┘\n\n──────►"
	if got := d.Render().String(); got != want {
		t.Errorf("Render = %q, want %q", got, want)
	}
	out, err := json.Marshal(&d)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(out) != old {
		t.Errorf("round trip = %s, want %s", out, old)
	}
}

// edgeDiagram is two boxes, one above the other, joined by an edge l3.
func edgeDiagram(t *testing.T) *Diagram {
	t.Helper()
	d := &Diagram{}
	for _, c := range []BoxCmd{
		{X1: 0, Y1: 0, X2: 4, Y2: 2},
		{X1: 0, Y1: 8, X2: 4, Y2: 10},
	} {
		if err := d.Apply(c); err != nil {
			t.Fatalf("BoxCmd: %v", err)
		}
	}
	if err := d.Apply(EdgeCmd{From: "b1", To: "b2", Arrow: ArrowEnd}); err != nil {
		t.Fatalf("EdgeCmd: %v", err)
	}
	return d
}
