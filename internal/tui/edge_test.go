package tui

import (
	"strings"
	"testing"

	"herdr-canvas/internal/canvas"
)

// twoBoxes puts b1 at rows 0-2 and b2 at rows 6-8, both five cells wide.
func twoBoxes(t *testing.T) model {
	t.Helper()
	m := editor(t)
	for _, c := range []canvas.BoxCmd{
		{X1: 0, Y1: 0, X2: 4, Y2: 2},
		{X1: 0, Y1: 6, X2: 4, Y2: 8},
	} {
		if err := m.d.Apply(c); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}
	m.tool = toolArrow
	return m
}

func TestArrowDragBetweenBoxesMakesAnEdge(t *testing.T) {
	cases := []struct {
		name           string
		fromX, fromY   int
		toX, toY       int
		wantEdge       bool
		wantFrom, want string
	}{
		{name: "inside to inside", fromX: 2, fromY: 1, toX: 2, toY: 7, wantEdge: true, wantFrom: "b1", want: "b2"},
		{name: "border counts as inside", fromX: 0, fromY: 0, toX: 4, toY: 8, wantEdge: true, wantFrom: "b1", want: "b2"},
		{name: "into empty space stays a line", fromX: 2, fromY: 1, toX: 20, toY: 4},
		{name: "from empty space stays a line", fromX: 20, fromY: 4, toX: 2, toY: 7},
		{name: "one box to itself stays a line", fromX: 1, fromY: 1, toX: 3, toY: 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := twoBoxes(t)
			m = send(t, m,
				leftDown(c.fromX, c.fromY+headerRows),
				leftMove(c.toX, c.toY+headerRows),
				leftUp(c.toX, c.toY+headerRows))
			if len(m.d.Elements) != 3 {
				t.Fatalf("elements = %d, want 3: %+v", len(m.d.Elements), m.d.Elements)
			}
			got := m.d.Elements[2]
			if got.IsEdge() != c.wantEdge {
				t.Fatalf("IsEdge = %v, want %v (%+v)", got.IsEdge(), c.wantEdge, got)
			}
			if c.wantEdge && (got.From != c.wantFrom || got.To != c.want) {
				t.Errorf("edge = %s->%s, want %s->%s", got.From, got.To, c.wantFrom, c.want)
			}
			if got.Arrow != canvas.ArrowEnd {
				t.Errorf("arrow = %q, want end", got.Arrow)
			}
		})
	}
}

func TestArrowDragHighlightsTheBoxUnderTheCursor(t *testing.T) {
	m := twoBoxes(t)
	m = send(t, m, leftDown(2, 1+headerRows), leftMove(2, 7+headerRows))
	painted := (&canvas.Diagram{Elements: m.overlayElements()}).Paint()
	if cp := painted[[2]int{0, 6}]; cp.Reverse {
		t.Fatal("paint marks the box before the highlight runs")
	}
	highlightBox(painted, m.d.BoxAt(m.cursor[0], m.cursor[1]))
	for _, p := range [][2]int{{0, 6}, {4, 8}, {2, 6}} {
		if !painted[p].Reverse {
			t.Errorf("cell %v is not highlighted", p)
		}
	}
	if painted[[2]int{0, 0}].Reverse {
		t.Error("the box the drag came from is highlighted")
	}
	if !strings.Contains(m.View().Content, "║") {
		t.Error("the drag preview does not show a doubled line")
	}
}

func TestDeletingABoxAndUndoingRestoresItsEdges(t *testing.T) {
	m := twoBoxes(t)
	if !m.commitCmds([]canvas.Command{canvas.EdgeCmd{From: "b1", To: "b2", Arrow: canvas.ArrowEnd}}) {
		t.Fatalf("EdgeCmd: %s", m.status)
	}
	m.tool = toolSelect
	m.selectOnly("b1")
	m.deleteSelected()
	if len(m.d.Elements) != 1 {
		t.Fatalf("after delete elements = %+v, want only b2", m.d.Elements)
	}
	m.doUndo()
	if len(m.d.Elements) != 3 {
		t.Fatalf("after undo elements = %+v, want the box and its edge back", m.d.Elements)
	}
	if !m.d.Elements[2].IsEdge() {
		t.Errorf("restored element = %+v, want an edge", m.d.Elements[2])
	}
}

func TestDeletingABoxAndItsEdgeTogetherDeletesBoth(t *testing.T) {
	m := twoBoxes(t)
	if !m.commitCmds([]canvas.Command{canvas.EdgeCmd{From: "b1", To: "b2"}}) {
		t.Fatalf("EdgeCmd: %s", m.status)
	}
	edge := m.d.Elements[2].ID
	m.tool = toolSelect
	m.selected = map[string]bool{"b1": true, edge: true}
	m.deleteSelected()
	if m.status != "" {
		t.Fatalf("status = %q, want no error", m.status)
	}
	if len(m.d.Elements) != 1 || m.d.Elements[0].ID != "b2" {
		t.Fatalf("after delete elements = %+v, want only b2", m.d.Elements)
	}
	if len(m.selected) != 0 {
		t.Errorf("selection = %v, want empty", m.selected)
	}
}

func TestMovingBothBoxesInOneCommitRederivesOnce(t *testing.T) {
	m := twoBoxes(t)
	if !m.commitCmds([]canvas.Command{canvas.EdgeCmd{From: "b1", To: "b2"}}) {
		t.Fatalf("EdgeCmd: %s", m.status)
	}
	if !m.commitCmds([]canvas.Command{
		canvas.MoveCmd{ID: "b1", DX: 6, DY: 2},
		canvas.MoveCmd{ID: "b2", DX: 6, DY: 2},
	}) {
		t.Fatalf("MoveCmd: %s", m.status)
	}
	e := m.d.Elements[2]
	if e.X1 != 8 || e.Y1 != 4 || e.X2 != 8 || e.Y2 != 8 {
		t.Errorf("endpoints = %d,%d-%d,%d, want 8,4-8,8", e.X1, e.Y1, e.X2, e.Y2)
	}
}
