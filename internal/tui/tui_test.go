package tui

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"herdr-canvas/internal/canvas"
	"herdr-canvas/internal/herdr"
	"herdr-canvas/internal/store"
	"herdr-canvas/internal/update"
	"herdr-canvas/internal/version"
)

func editor(t *testing.T) model {
	t.Helper()
	s := &store.Store{Base: t.TempDir()}
	d := &canvas.Diagram{Name: "demo"}
	if err := s.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}
	mt, err := s.ModTime("demo")
	if err != nil {
		t.Fatalf("ModTime: %v", err)
	}
	return model{s: s, d: d, mtime: mt, phase: phaseEdit, tool: toolBox, width: 40, height: 12,
		vp: viewport{zoom: 10}}
}

func send(t *testing.T, m model, msgs ...tea.Msg) model {
	t.Helper()
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(model)
	}
	return m
}

func screen(m model) string { return m.View().Content }

func key(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEsc}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case " ", "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	default:
		r := []rune(s)
		return tea.KeyPressMsg{Code: r[0], Text: s}
	}
}

func ctrlZ() tea.KeyPressMsg { return tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl} }
func ctrlY() tea.KeyPressMsg { return tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl} }
func ctrlShiftZ() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl | tea.ModShift}
}

func leftDown(x, y int) tea.MouseClickMsg {
	return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft}
}
func leftMove(x, y int) tea.MouseMotionMsg {
	return tea.MouseMotionMsg{X: x, Y: y, Button: tea.MouseLeft}
}
func leftUp(x, y int) tea.MouseReleaseMsg {
	return tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft}
}
func midDown(x, y int) tea.MouseClickMsg {
	return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseMiddle}
}
func midMove(x, y int) tea.MouseMotionMsg {
	return tea.MouseMotionMsg{X: x, Y: y, Button: tea.MouseMiddle}
}
func midUp(x, y int) tea.MouseReleaseMsg {
	return tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseMiddle}
}
func wheelAt(x, y int, btn tea.MouseButton, mod tea.KeyMod) tea.MouseWheelMsg {
	return tea.MouseWheelMsg{X: x, Y: y, Button: btn, Mod: mod}
}

func TestCanvasPointUsesFixedOrigin(t *testing.T) {
	m := editor(t)
	m.vp.origin = [2]int{5, 7}
	cases := []struct {
		x, y   int
		want   [2]int
		wantOK bool
	}{
		{0, 0, [2]int{}, false},    // header row
		{0, 1, [2]int{5, 7}, true}, // first canvas row
		{3, 4, [2]int{8, 10}, true},
		{0, 10, [2]int{5, 16}, true}, // last canvas row
		{0, 11, [2]int{}, false},     // status row
		{40, 1, [2]int{}, false},     // past the right edge
	}
	for _, c := range cases {
		got, ok := m.canvasPoint(c.x, c.y)
		if ok != c.wantOK || (ok && got != c.want) {
			t.Errorf("canvasPoint(%d,%d) = %v,%v, want %v,%v", c.x, c.y, got, ok, c.want, c.wantOK)
		}
	}
}

func TestEnsureVisibleMovesOrigin(t *testing.T) {
	cases := []struct {
		name   string
		zoom   int
		cursor [2]int
		want   [2]int
	}{
		{"1x right and down", 10, [2]int{50, 15}, [2]int{11, 6}},
		{"2x right and down", 20, [2]int{25, 8}, [2]int{6, 4}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := editor(t)
			m.vp = viewport{zoom: c.zoom}
			m.cursor = c.cursor
			m.ensureVisible()
			if m.vp.origin != c.want {
				t.Errorf("origin = %v, want %v", m.vp.origin, c.want)
			}
		})
	}
}

func TestCanvasPointIgnoresRenderedContent(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 20, Y1: 20, X2: 22, Y2: 22}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got, ok := m.canvasPoint(3, 2)
	if !ok || got != [2]int{3, 1} {
		t.Errorf("canvasPoint = %v,%v, want [3 1],true — the origin must not follow the drawing", got, ok)
	}
}

func TestMouseDragDrawsBoxUnderThePointer(t *testing.T) {
	m := editor(t)
	m = send(t, m,
		leftDown(10, 5),
		leftMove(14, 8),
		leftUp(14, 8),
	)
	if len(m.d.Elements) != 1 {
		t.Fatalf("elements = %d, want 1", len(m.d.Elements))
	}
	e := m.d.Elements[0]
	if e.Type != canvas.Box || e.X1 != 10 || e.Y1 != 4 || e.X2 != 14 || e.Y2 != 7 {
		t.Errorf("box = %+v, want (10,4)-(14,7)", e)
	}
	if reloaded, err := m.s.Load("demo"); err != nil {
		t.Fatalf("Load: %v", err)
	} else if len(reloaded.Elements) != 1 {
		t.Errorf("saved elements = %d, want 1", len(reloaded.Elements))
	}
}

func TestMiddleDragPansAndDoesNotDraw(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 4, Y2: 4}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	n := len(m.d.Elements)
	m = send(t, m, midDown(8, 4), midMove(5, 2), midUp(5, 2))
	if len(m.d.Elements) != n {
		t.Fatalf("middle drag mutated the diagram")
	}
	if m.vp.origin == [2]int{0, 0} {
		t.Fatal("origin did not move")
	}
}

func TestMiddleDragPansWhenMotionDropsButton(t *testing.T) {
	m := editor(t)
	n := len(m.d.Elements)
	m = send(t, m, midDown(8, 4), tea.MouseMotionMsg{X: 5, Y: 2}, midUp(5, 2))
	if len(m.d.Elements) != n {
		t.Fatalf("pan mutated the diagram")
	}
	if m.vp.origin == [2]int{0, 0} {
		t.Fatal("origin did not move")
	}
}

func TestMiddleDragPansWhenPressNeverArrives(t *testing.T) {
	m := editor(t)
	n := len(m.d.Elements)
	m = send(t, m, midMove(8, 4), midMove(5, 2), midUp(5, 2))
	if len(m.d.Elements) != n {
		t.Fatalf("pan mutated the diagram")
	}
	if m.vp.origin == [2]int{0, 0} {
		t.Fatal("origin did not move")
	}
}

func TestWheelPansCanvasNotFooter(t *testing.T) {
	m := editor(t)
	m = send(t, m, tea.WindowSizeMsg{Width: 40, Height: 12})
	before := m.vp.origin
	m = send(t, m, wheelAt(2, 11, tea.MouseWheelDown, 0))
	if m.vp.origin != before {
		t.Errorf("footer wheel panned: %v", m.vp.origin)
	}
	m = send(t, m, wheelAt(2, 3, tea.MouseWheelDown, 0))
	if m.vp.origin[1] != before[1]+1 {
		t.Errorf("origin = %v, want y+1", m.vp.origin)
	}
	m = send(t, m, wheelAt(2, 3, tea.MouseWheelDown, tea.ModShift))
	if m.vp.origin[0] != 1 {
		t.Errorf("shift+wheel origin.x = %d, want 1", m.vp.origin[0])
	}
}

func TestCtrlWheelStepsZoom(t *testing.T) {
	m := editor(t)
	m = send(t, m, tea.WindowSizeMsg{Width: 40, Height: 12})
	m = send(t, m, wheelAt(2, 3, tea.MouseWheelUp, tea.ModCtrl))
	if m.vp.zoom != 15 {
		t.Errorf("zoom = %d, want 15", m.vp.zoom)
	}
	if m.vp.origin != [2]int{0, 0} {
		t.Errorf("ctrl+wheel panned: %v", m.vp.origin)
	}
	m = send(t, m, wheelAt(2, 3, tea.MouseWheelUp, tea.ModCtrl))
	if m.vp.zoom != 20 {
		t.Errorf("zoom = %d, want 20", m.vp.zoom)
	}
	m = send(t, m, wheelAt(2, 3, tea.MouseWheelUp, tea.ModCtrl))
	if m.vp.zoom != 20 {
		t.Errorf("zoom-in wrapped: %d", m.vp.zoom)
	}
	m = send(t, m, wheelAt(2, 3, tea.MouseWheelDown, tea.ModCtrl))
	if m.vp.zoom != 15 {
		t.Errorf("zoom-out = %d, want 15", m.vp.zoom)
	}
}

func TestPlusMinusStepsZoom(t *testing.T) {
	m := editor(t)
	m = send(t, m, tea.KeyPressMsg{Code: '+', Text: "+"})
	if m.vp.zoom != 15 {
		t.Errorf("zoom = %d after +", m.vp.zoom)
	}
	m = send(t, m, tea.KeyPressMsg{Code: '-', Text: "-"})
	if m.vp.zoom != 10 {
		t.Errorf("zoom = %d after -", m.vp.zoom)
	}
	m = send(t, m, tea.KeyPressMsg{Code: '-', Text: "-"})
	if m.vp.zoom != 5 {
		t.Errorf("zoom = %d after second -, want 5", m.vp.zoom)
	}
}

func TestWheelDoesNotDeleteOrDraw(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 4, Y2: 4}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	m.tool = toolDelete
	for _, msg := range []tea.Msg{
		wheelAt(2, 3, tea.MouseWheelUp, 0),
		wheelAt(2, 3, tea.MouseWheelDown, 0),
		tea.MouseClickMsg{X: 2, Y: 3, Button: tea.MouseRight},
		midDown(2, 3),
	} {
		m = send(t, m, msg)
		if len(m.d.Elements) != 1 {
			t.Fatalf("msg %T deleted the element", msg)
		}
	}
	m.tool = toolBox
	m = send(t, m,
		wheelAt(2, 3, tea.MouseWheelUp, 0),
		wheelAt(6, 6, tea.MouseWheelUp, 0),
	)
	if len(m.d.Elements) != 1 {
		t.Errorf("wheel committed a box: %d elements", len(m.d.Elements))
	}
}

func TestReleaseWithoutPressCommitsNothing(t *testing.T) {
	m := editor(t)
	m.anchor = [2]int{2, 2}
	m = send(t, m, leftUp(9, 9))
	if len(m.d.Elements) != 0 {
		t.Errorf("elements = %d, want 0", len(m.d.Elements))
	}
}

func TestToolSwitchClearsAnchorAndPending(t *testing.T) {
	m := editor(t)
	m.tool = toolDraw
	m = send(t, m, leftDown(3, 3), leftMove(4, 3))
	if len(m.pending) == 0 {
		t.Fatal("draw collected no cells")
	}
	m = send(t, m, key("b"))
	if m.pending != nil || m.anchored || m.mouse || m.anchor != [2]int{} {
		t.Errorf("tool switch left state: pending=%v anchored=%v mouse=%v anchor=%v",
			m.pending, m.anchored, m.mouse, m.anchor)
	}
	m = send(t, m, leftUp(9, 9))
	if len(m.d.Elements) != 0 {
		t.Errorf("stale anchor committed an element: %+v", m.d.Elements)
	}
}

func TestKeyboardAnchorAndCommitDrawsBox(t *testing.T) {
	m := editor(t)
	m = send(t, m,
		key(" "),
		key("right"), key("right"),
		key("down"),
		key(" "),
	)
	if len(m.d.Elements) != 1 {
		t.Fatalf("elements = %d, want 1", len(m.d.Elements))
	}
	e := m.d.Elements[0]
	if e.X1 != 0 || e.Y1 != 0 || e.X2 != 2 || e.Y2 != 1 {
		t.Errorf("box = (%d,%d)-(%d,%d), want (0,0)-(2,1)", e.X1, e.Y1, e.X2, e.Y2)
	}
}

func TestStatusClearsOnNextInput(t *testing.T) {
	m := editor(t)
	m.status = "old error"
	m = send(t, m, key("l"))
	if m.status != "" {
		t.Errorf("status = %q, want empty", m.status)
	}
}

func TestExternalEditIsReloadedNotOverwritten(t *testing.T) {
	m := editor(t)
	other := &canvas.Diagram{Name: "demo"}
	if err := other.Apply(canvas.TextCmd{X: 1, Y: 1, Text: "cli"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := m.s.Save(other); err != nil {
		t.Fatalf("Save: %v", err)
	}
	m.mtime = m.mtime.Add(-1) // the store rewrote the file behind the editor
	m = send(t, m, leftDown(3, 3), leftUp(6, 6))
	if len(m.d.Elements) != 2 {
		t.Fatalf("elements = %d, want 2 (reloaded text plus the new box)", len(m.d.Elements))
	}
	if m.status == "" {
		t.Error("conflict was not surfaced in the status line")
	}
	saved, err := m.s.Load("demo")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(saved.Elements) != 2 {
		t.Errorf("saved elements = %d, want 2", len(saved.Elements))
	}
}

func TestNamePhaseRefusesExistingName(t *testing.T) {
	m := editor(t)
	m.phase = phaseName
	m.nameInput = "demo"
	m = send(t, m, key("enter"))
	if m.phase != phaseName {
		t.Errorf("phase = %v, want phaseName", m.phase)
	}
	if m.status == "" {
		t.Error("no status for the name collision")
	}
	d, err := m.s.Load("demo")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if d.Name != "demo" {
		t.Errorf("stored diagram = %+v", d)
	}
}

func TestPickerSurfacesListError(t *testing.T) {
	m := editor(t)
	m.phase = phasePick
	m.status = "read store: permission denied"
	if got := screen(m); !strings.Contains(got, "permission denied") {
		t.Errorf("picker view hides the error: %q", got)
	}
}

func TestLineSnapsBothEndpoints(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 4, Y2: 4}); err != nil {
		t.Fatalf("Apply box: %v", err)
	}
	if err := m.d.Apply(canvas.BoxCmd{X1: 10, Y1: 0, X2: 14, Y2: 4}); err != nil {
		t.Fatalf("Apply box: %v", err)
	}
	m.tool = toolLine
	m.anchor = [2]int{5, 2} // one cell right of the first box
	m.cursor = [2]int{9, 2} // one cell left of the second box
	m.mouse = true
	m.commit()
	line := m.d.Elements[len(m.d.Elements)-1]
	if line.Type != canvas.Line {
		t.Fatalf("last element = %v, want line", line.Type)
	}
	if line.X1 != 4 || line.X2 != 10 {
		t.Errorf("line = (%d,%d)-(%d,%d), want both ends snapped to x=4 and x=10",
			line.X1, line.Y1, line.X2, line.Y2)
	}
}

func TestClickArrowChipSelectsArrow(t *testing.T) {
	m := editor(t)
	m = send(t, m, tea.WindowSizeMsg{Width: 80, Height: 12})
	ch := layoutChrome(80, m.d.Name, m.vp.zoom, m.cursor, m.tool, m.hist.canUndo(), m.hist.canRedo(), "")
	idx := indexOf(ch.footer, "arrow")
	m = send(t, m, leftDown(idx, 11), leftUp(idx, 11))
	if m.tool != toolArrow {
		t.Errorf("tool = %v, want arrow", m.tool)
	}
}

func TestClickPaddingDoesNotSwitchTool(t *testing.T) {
	m := editor(t)
	m = send(t, m, tea.WindowSizeMsg{Width: 80, Height: 12})
	m = send(t, m, leftDown(79, 11), leftUp(79, 11))
	if m.tool != toolBox {
		t.Errorf("tool = %v", m.tool)
	}
}

func TestClickZoomChipResetsZoomOnly(t *testing.T) {
	m := editor(t)
	m = send(t, m, tea.WindowSizeMsg{Width: 40, Height: 12})
	if err := m.d.Apply(canvas.BoxCmd{X1: 80, Y1: 40, X2: 90, Y2: 50}); err != nil {
		t.Fatal(err)
	}
	m.vp.zoom = 20
	m.vp.origin = [2]int{3, 3}
	ch := layoutChrome(40, m.d.Name, m.vp.zoom, m.cursor, m.tool, false, false, "")
	idx := indexOf(ch.header, "2x")
	m = send(t, m, leftDown(idx, 0), leftUp(idx, 0))
	if m.vp.zoom != 10 {
		t.Errorf("zoom = %d, want 10", m.vp.zoom)
	}
	if m.vp.origin != [2]int{3, 3} {
		t.Errorf("reset panned: %v", m.vp.origin)
	}
}

func TestClickRecenterCentersSmallBox(t *testing.T) {
	m := editor(t)
	m = send(t, m, tea.WindowSizeMsg{Width: 40, Height: 12})
	if err := m.d.Apply(canvas.BoxCmd{X1: 2, Y1: 2, X2: 4, Y2: 3}); err != nil {
		t.Fatal(err)
	}
	m.vp.origin = [2]int{20, 20}
	ch := layoutChrome(40, m.d.Name, m.vp.zoom, m.cursor, m.tool, false, false, "")
	idx := indexOf(ch.header, "recenter")
	if idx < 0 {
		t.Fatalf("header %q", ch.header)
	}
	m = send(t, m, leftDown(idx, 0), leftUp(idx, 0))
	if m.vp.zoom != 10 {
		t.Errorf("recenter changed zoom: %d", m.vp.zoom)
	}
	if m.vp.origin[0] == 20 && m.vp.origin[1] == 20 {
		t.Fatal("origin did not move")
	}
}

func TestViewFooterHasChips(t *testing.T) {
	m := editor(t)
	m = send(t, m, tea.WindowSizeMsg{Width: 80, Height: 12})
	lines := strings.Split(screen(m), "\n")
	if !strings.Contains(lines[0], "1x") {
		t.Errorf("header = %q", lines[0])
	}
	if !strings.Contains(lines[len(lines)-1], "[box]") {
		t.Errorf("footer = %q", lines[len(lines)-1])
	}
}

func TestViewFillsTheTerminal(t *testing.T) {
	m := editor(t)
	m = send(t, m, tea.WindowSizeMsg{Width: 40, Height: 12})
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 30}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	lines := strings.Split(screen(m), "\n")
	if len(lines) != 12 {
		t.Fatalf("view has %d lines, want 12", len(lines))
	}
	if !strings.Contains(lines[11], "[box]") {
		t.Errorf("last line = %q, want the status line", lines[11])
	}
}

// canvasLine returns one rendered canvas row of the editor view. Row 0 is the
// first row below the header.
func canvasLine(t *testing.T, m model, row int) string {
	t.Helper()
	lines := strings.Split(screen(m), "\n")
	if row+headerRows >= len(lines) {
		t.Fatalf("row %d is outside the view (%d lines)", row, len(lines))
	}
	return lines[row+headerRows]
}

func TestMoveDragPreviewsLiveAndLeavesAGhost(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	m.tool = toolMove
	m = send(t, m, leftDown(1, 1), leftMove(6, 1))

	if m.d.Elements[0].X1 != 0 {
		t.Errorf("the drag mutated the diagram before release: %+v", m.d.Elements[0])
	}
	ghost := strings.Repeat(ghostGlyph, 3)
	if got, want := canvasLine(t, m, 0), ghost+"  ┌─┐"; got != want {
		t.Errorf("row 0 = %q, want %q (ghost at the source, box under the cursor)", got, want)
	}

	m = send(t, m, leftUp(6, 1))
	if got := m.d.Elements[0].X1; got != 5 {
		t.Errorf("after release x1 = %d, want 5", got)
	}
	if got, want := canvasLine(t, m, 0), "     ┌─┐"; got != want {
		t.Errorf("row 0 = %q, want %q — the ghost must not survive the commit", got, want)
	}
}

func TestTextRendersAtTheClickedCellWhileTyping(t *testing.T) {
	m := editor(t)
	m.tool = toolText
	m = send(t, m, leftDown(3, 2), key("h"), key("i"))

	if len(m.d.Elements) != 0 {
		t.Fatalf("typing committed early: %+v", m.d.Elements)
	}
	row := canvasLine(t, m, 1)
	if want := "   hi" + cursorGlyph; row != want {
		t.Errorf("row = %q, want %q", row, want)
	}
	if strings.Contains(m.statusLine(), "hi") {
		t.Errorf("the buffer is still echoed in the status line: %q", m.statusLine())
	}

	m = send(t, m, key("enter"))
	if len(m.d.Elements) != 1 || m.d.Elements[0].Text != "hi" {
		t.Fatalf("enter did not commit the text: %+v", m.d.Elements)
	}
	if got := canvasLine(t, m, 1); got != "   hi" {
		t.Errorf("committed row = %q, want %q", got, "   hi")
	}
}

func TestTextEscapeDiscardsTheInPlacePreview(t *testing.T) {
	m := editor(t)
	m.tool = toolText
	m = send(t, m, leftDown(3, 2), key("h"), key("esc"))
	if len(m.d.Elements) != 0 {
		t.Fatalf("escape committed text: %+v", m.d.Elements)
	}
	if got := canvasLine(t, m, 1); strings.TrimSpace(got) != "" {
		t.Errorf("row = %q, want empty after escape", got)
	}
}

func TestLineAndArrowToolsRouteAsElbows(t *testing.T) {
	cases := []struct {
		name      string
		tool      tool
		to        [2]int
		wantArrow canvas.Arrow
		wantRows  []string
	}{
		{
			name: "line bends and carries no arrow",
			tool: toolLine, to: [2]int{4, 3}, wantArrow: canvas.ArrowNone,
			wantRows: []string{"│", "│", "│", "└────"},
		},
		{
			name: "arrow bends and ends in an arrowhead",
			tool: toolArrow, to: [2]int{4, 3}, wantArrow: canvas.ArrowEnd,
			wantRows: []string{"│", "│", "│", "└───►"},
		},
		{
			name: "a horizontal drag stays one straight run",
			tool: toolArrow, to: [2]int{4, 0}, wantArrow: canvas.ArrowEnd,
			wantRows: []string{"────►"},
		},
		{
			name: "a vertical drag stays one straight run",
			tool: toolLine, to: [2]int{0, 3}, wantArrow: canvas.ArrowNone,
			wantRows: []string{"│", "│", "│", "│"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := editor(t)
			m.tool = c.tool
			m = send(t, m,
				leftDown(0, headerRows),
				leftMove(c.to[0], c.to[1]+headerRows),
			)
			for i, want := range c.wantRows {
				if got := canvasLine(t, m, i); got != want {
					t.Errorf("preview row %d = %q, want %q", i, got, want)
				}
			}
			m = send(t, m, leftUp(c.to[0], c.to[1]+headerRows))
			if len(m.d.Elements) != 1 {
				t.Fatalf("elements = %d, want 1", len(m.d.Elements))
			}
			e := m.d.Elements[0]
			if e.Type != canvas.Line || e.Arrow != c.wantArrow {
				t.Errorf("element = %+v, want a line with arrow %q", e, c.wantArrow)
			}
			for i, want := range c.wantRows {
				if got := canvasLine(t, m, i); got != want {
					t.Errorf("committed row %d = %q, want %q", i, got, want)
				}
			}
		})
	}
}

func TestArrowToolKeyIsSeparateFromLine(t *testing.T) {
	m := editor(t)
	m = send(t, m, key("a"))
	if m.tool != toolArrow {
		t.Fatalf("tool = %v, want the arrow tool", m.tool)
	}
	m = send(t, m, key("l"))
	if m.tool != toolLine {
		t.Errorf("tool = %v, want the line tool", m.tool)
	}
}

type fakeSender struct {
	agents   []herdr.Agent
	listErr  error
	sentTo   string
	sentText string
	focused  string
	sendErr  error
}

func (f *fakeSender) Agents(string) ([]herdr.Agent, error) { return f.agents, f.listErr }

func (f *fakeSender) SendText(paneID, text string) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sentTo, f.sentText = paneID, text
	return nil
}

func (f *fakeSender) Focus(paneID string) error {
	f.focused = paneID
	return nil
}

func editorWithAgents(t *testing.T, agents ...herdr.Agent) (model, *fakeSender) {
	t.Helper()
	m := editor(t)
	f := &fakeSender{agents: agents}
	m.send = f
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 4, Y2: 2, Label: "hi"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return m, f
}

func TestSendGoesStraightToASingleAgent(t *testing.T) {
	m, f := editorWithAgents(t, herdr.Agent{PaneID: "w1:p9", Agent: "claude", Status: "idle"})
	m = send(t, m, key("s"))

	if m.phase != phaseEdit {
		t.Errorf("phase = %v, want the editor — one agent needs no picker", m.phase)
	}
	if f.sentTo != "w1:p9" {
		t.Errorf("sent to %q, want w1:p9", f.sentTo)
	}
	if !strings.Contains(f.sentText, `herdr-canvas --name "demo" export`) {
		t.Errorf("the prompt does not name export:\n%s", f.sentText)
	}
	if !strings.Contains(f.sentText, "herdr-canvas skill") {
		t.Errorf("the prompt does not name skill:\n%s", f.sentText)
	}
	if strings.Contains(f.sentText, "┌───┐") {
		t.Errorf("the prompt pasted the picture:\n%s", f.sentText)
	}
	if !strings.Contains(m.status, "added demo") {
		t.Errorf("status = %q, want an added report", m.status)
	}
}

func TestSendOpensAPickerForSeveralAgents(t *testing.T) {
	m, f := editorWithAgents(t,
		herdr.Agent{PaneID: "w1:p1", Agent: "claude", Status: "idle"},
		herdr.Agent{PaneID: "w1:p2", Agent: "opencode", Status: "working"},
	)
	m = send(t, m, key("s"))
	if m.phase != phaseAgent {
		t.Fatalf("phase = %v, want the agent picker", m.phase)
	}
	if f.sentTo != "" {
		t.Errorf("sent to %q before a choice was made", f.sentTo)
	}
	view := screen(m)
	for _, want := range []string{"w1:p1", "w1:p2", "claude", "opencode"} {
		if !strings.Contains(view, want) {
			t.Errorf("picker does not list %q:\n%s", want, view)
		}
	}

	m = send(t, m, key("j"), key("enter"))
	if f.sentTo != "w1:p2" {
		t.Errorf("sent to %q, want the chosen agent w1:p2", f.sentTo)
	}
	if m.phase != phaseEdit {
		t.Errorf("phase = %v, want the editor after the send", m.phase)
	}
}

func TestSendPickerEscapeSendsNothing(t *testing.T) {
	m, f := editorWithAgents(t,
		herdr.Agent{PaneID: "w1:p1", Agent: "claude"},
		herdr.Agent{PaneID: "w1:p2", Agent: "claude"},
	)
	m = send(t, m, key("s"), key("esc"))
	if f.sentTo != "" {
		t.Errorf("escape still sent to %q", f.sentTo)
	}
	if m.phase != phaseEdit {
		t.Errorf("phase = %v, want the editor", m.phase)
	}
}

func TestSendReportsWhenTheWorkspaceHasNoAgent(t *testing.T) {
	m, f := editorWithAgents(t)
	m = send(t, m, key("s"))
	if f.sentTo != "" {
		t.Errorf("sent to %q with no agent present", f.sentTo)
	}
	if !strings.Contains(m.status, "no agent") {
		t.Errorf("status = %q, want a no-agent report", m.status)
	}
}

func TestAgentPickerNamesTheTabAndWorkspace(t *testing.T) {
	m, _ := editorWithAgents(t,
		herdr.Agent{PaneID: "w4M:p3", Agent: "claude", Status: "idle",
			TabID: "w4M:t2", TabLabel: "control", WorkspaceLabel: "herdr-canvas",
			Title: "Editable grid validation and mouse bugs"},
		herdr.Agent{PaneID: "w4M:pS", Agent: "claude", Status: "idle",
			TabID: "w4M:tD", TabLabel: "review-prose", WorkspaceLabel: "herdr-canvas",
			Title: "Herdr-canvas prose review"},
		herdr.Agent{PaneID: "w4M:pV", Agent: "claude", Status: "working",
			TabID: "w4M:tE", TabLabel: "ship", WorkspaceLabel: "herdr-canvas",
			Title: "herdr canvas pull request ship stage"},
	)
	m.width = 78
	m = send(t, m, key("s"))
	view := screen(m)
	for _, want := range []string{"control", "review-prose", "ship", "workspace: herdr-canvas"} {
		if !strings.Contains(view, want) {
			t.Errorf("picker does not name %q:\n%s", want, view)
		}
	}
	if os.Getenv("SHOW_PICKER") != "" {
		t.Log("\n" + view)
	}
}

func TestSendWritesTheInputWithoutSubmitting(t *testing.T) {
	m, f := editorWithAgents(t, herdr.Agent{PaneID: "w1:p9", Agent: "claude", TabLabel: "ship"})
	m = send(t, m, key("s"))

	// The person adds their own words and presses enter. The canvas must not
	// submit, so the payload carries no trailing newline of its own.
	if strings.HasSuffix(f.sentText, "\n\n") {
		t.Errorf("the payload ends in a blank line, which can submit:\n%q", f.sentText[len(f.sentText)-20:])
	}
	if f.focused != "w1:p9" {
		t.Errorf("focused %q, want the agent pane so the person can type", f.focused)
	}
	if !strings.Contains(m.status, "press enter") {
		t.Errorf("status = %q, want it to say the person submits", m.status)
	}
}

func TestEscapeReturnsToThePickerWithAFreshList(t *testing.T) {
	m := editor(t)
	// A second diagram appears while the editor is open. The old picker list
	// was built before it existed.
	if err := m.s.Save(&canvas.Diagram{Name: "later"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	m.names = []string{"demo"}

	m = send(t, m, key("esc"))
	if m.phase != phasePick {
		t.Fatalf("phase = %v, want the picker", m.phase)
	}
	if strings.Join(m.names, ",") != "demo,later" {
		t.Errorf("names = %v, want the store read again", m.names)
	}
	if m.names[m.sel] != "demo" {
		t.Errorf("cursor on %q, want the diagram the editor was showing", m.names[m.sel])
	}

	// Opening the other diagram from the picker switches the editor to it.
	m = send(t, m, key("j"), key("enter"))
	if m.phase != phaseEdit || m.d.Name != "later" {
		t.Errorf("phase = %v, diagram = %q, want the editor on later", m.phase, m.d.Name)
	}
}

func TestPickerEscapeGoesBackToTheOpenDiagram(t *testing.T) {
	m := editor(t)
	m = send(t, m, key("esc"))
	if m.phase != phasePick {
		t.Fatalf("phase = %v, want the picker", m.phase)
	}
	m = send(t, m, key("esc"))
	if m.phase != phaseEdit {
		t.Errorf("phase = %v, want the editor again", m.phase)
	}
}

func TestEscapeWhileTypingDoesNotLeaveTheEditor(t *testing.T) {
	m := editor(t)
	m.tool = toolText
	m = send(t, m, leftDown(3, 2), key("h"), key("esc"))
	if m.phase != phaseEdit {
		t.Errorf("phase = %v, want the editor — escape cancels the text entry", m.phase)
	}
	if m.typing {
		t.Error("still typing after escape")
	}
}

func TestPollShowsAChangeAnAgentMade(t *testing.T) {
	m := editor(t)
	// The agent adds a box through the CLI while the person only watches.
	d, err := m.s.Load("demo")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 4, Y2: 2}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := m.s.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	next, cmd := m.Update(pollMsg{})
	m = next.(model)
	if cmd == nil {
		t.Error("the poll did not schedule the next look")
	}
	if len(m.d.Elements) != 1 {
		t.Fatalf("elements = %d, want the agent's box without the person drawing", len(m.d.Elements))
	}
	if !strings.Contains(m.status, "reloaded") {
		t.Errorf("status = %q, want the reload reported", m.status)
	}
}

func TestApplyThenUndoRestores(t *testing.T) {
	m := editor(t)
	m = send(t, m, leftDown(2, 2), leftMove(6, 5), leftUp(6, 5))
	if len(m.d.Elements) != 1 {
		t.Fatalf("elements = %d", len(m.d.Elements))
	}
	m = send(t, m, ctrlZ())
	if len(m.d.Elements) != 0 {
		t.Errorf("undo left %d elements", len(m.d.Elements))
	}
	reloaded, err := m.s.Load("demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Elements) != 0 {
		t.Error("undo did not save")
	}
	m = send(t, m, ctrlY())
	if len(m.d.Elements) != 1 || m.d.Elements[0].ID != "b1" {
		t.Errorf("redo = %+v", m.d.Elements)
	}
}

func TestFailedApplyDoesNotPush(t *testing.T) {
	m := editor(t)
	m.tool = toolMove
	m = send(t, m, key(" "))
	if m.hist.canUndo() {
		t.Fatal("empty move pushed history")
	}
}

func TestReloadIsOneUndoStep(t *testing.T) {
	m := editor(t)
	m = send(t, m, leftDown(2, 2), leftUp(4, 4))
	other := &canvas.Diagram{Name: "demo"}
	if err := other.Apply(canvas.TextCmd{X: 1, Y: 1, Text: "agent"}); err != nil {
		t.Fatal(err)
	}
	if err := m.s.Save(other); err != nil {
		t.Fatal(err)
	}
	m = send(t, m, pollMsg{})
	if len(m.d.Elements) != 1 || m.d.Elements[0].Type != canvas.Text {
		t.Fatalf("reload = %+v", m.d.Elements)
	}
	m = send(t, m, ctrlZ())
	if len(m.d.Elements) != 1 || m.d.Elements[0].Type != canvas.Box {
		t.Errorf("undo reload = %+v", m.d.Elements)
	}
}

func TestPollLeavesADragAlone(t *testing.T) {
	m := editor(t)
	m.tool = toolBox
	m = send(t, m, leftDown(2, 2), leftMove(6, 5))

	d, _ := m.s.Load("demo")
	_ = d.Apply(canvas.BoxCmd{X1: 20, Y1: 0, X2: 24, Y2: 2})
	_ = m.s.Save(d)

	next, cmd := m.Update(pollMsg{})
	m = next.(model)
	if cmd == nil {
		t.Error("the poll did not schedule the next look")
	}
	// Reloading here would move the ground under an unfinished drag.
	if len(m.d.Elements) != 0 {
		t.Errorf("elements = %d, want the drag left alone", len(m.d.Elements))
	}
	if !m.mouse {
		t.Error("the drag was cancelled by the poll")
	}
}

func TestUndoDuringBoxDragDiscardsPreview(t *testing.T) {
	m := editor(t)
	m = send(t, m, leftDown(2, 2), leftMove(6, 5))
	if !m.mouse {
		t.Fatal("expected drag")
	}
	m = send(t, m, ctrlZ())
	if m.mouse || m.anchored {
		t.Fatal("drag still live")
	}
	if len(m.d.Elements) != 0 {
		t.Errorf("discard undid committed work: %+v", m.d.Elements)
	}
}

func TestClickOutsideCommitsText(t *testing.T) {
	m := editor(t)
	m.tool = toolText
	m = send(t, m, leftDown(3, 2), key("h"), key("i"), leftDown(10, 5), leftUp(10, 5))
	if m.typing {
		t.Fatal("still typing")
	}
	if len(m.d.Elements) != 1 || m.d.Elements[0].Text != "hi" {
		t.Fatalf("commit = %+v", m.d.Elements)
	}
	if m.d.Elements[0].X != 3 || m.d.Elements[0].Y != 1 {
		t.Errorf("text pos = %d,%d", m.d.Elements[0].X, m.d.Elements[0].Y)
	}
}

func TestClickOutsideEmptyDoesNotWrite(t *testing.T) {
	m := editor(t)
	m.tool = toolText
	m = send(t, m, leftDown(3, 2), leftDown(10, 5), leftUp(10, 5))
	if m.typing {
		t.Fatal("still typing")
	}
	if len(m.d.Elements) != 0 {
		t.Errorf("empty commit wrote %+v", m.d.Elements)
	}
}

func TestClickOutsideDoesNotStartNewText(t *testing.T) {
	m := editor(t)
	m.tool = toolText
	m = send(t, m, leftDown(3, 2), key("a"), leftDown(10, 5), leftUp(10, 5))
	if m.typing {
		t.Fatal("click-outside started a new buffer")
	}
}

func TestDoubleClickTextEntersEditAndKeepsID(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.TextCmd{X: 3, Y: 1, Text: "hi"}); err != nil {
		t.Fatal(err)
	}
	m.tool = toolText
	// canvas cell (3,1) is terminal (3, 1+headerRows) = (3,2)
	m = send(t, m, leftDown(3, 2), leftUp(3, 2), leftDown(3, 2), leftUp(3, 2))
	if !m.typing || m.textBuf != "hi" || m.editID != "t1" {
		t.Fatalf("edit typing=%v buf=%q id=%q", m.typing, m.textBuf, m.editID)
	}
	m = send(t, m, key("!"), key("enter"))
	if len(m.d.Elements) != 1 || m.d.Elements[0].ID != "t1" || m.d.Elements[0].Text != "hi!" {
		t.Errorf("after edit %+v", m.d.Elements)
	}
}

func TestUndoDuringTextCommitsThenSecondUndoRemovesIt(t *testing.T) {
	m := editor(t)
	m.tool = toolText
	m = send(t, m, leftDown(3, 2), key("h"), key("i"), ctrlZ())
	if !m.typing {
		if len(m.d.Elements) != 1 || m.d.Elements[0].Text != "hi" {
			t.Fatalf("first undo did not commit: %+v", m.d.Elements)
		}
	} else {
		t.Fatal("still typing")
	}
	m = send(t, m, ctrlZ())
	if len(m.d.Elements) != 0 {
		t.Errorf("second undo = %+v", m.d.Elements)
	}
}

func setVersion(t *testing.T, v string) {
	t.Helper()
	prev := version.Version
	version.Version = v
	t.Cleanup(func() { version.Version = prev })
}

func stateClient(t *testing.T) *update.Client {
	t.Helper()
	dir := t.TempDir()
	return &update.Client{
		LookupEnv: func(k string) string {
			if k == "XDG_STATE_HOME" {
				return dir
			}
			return ""
		},
	}
}

func newerCheck(latest string) func(context.Context) (update.Result, error) {
	return func(context.Context) (update.Result, error) {
		return update.Result{Current: version.Version, Latest: latest, Newer: true}, nil
	}
}

func applyReleaseInit(t *testing.T, m model) model {
	t.Helper()
	if !version.IsRelease() {
		t.Fatal("applyReleaseInit needs a 0.x.y Version")
	}
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned nil")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Init msg = %T, want BatchMsg", msg)
	}
	var sawCheck bool
	for _, c := range batch {
		inner := c()
		if _, isPoll := inner.(pollMsg); isPoll {
			continue
		}
		sawCheck = true
		next, _ := m.Update(inner)
		m = next.(model)
	}
	if !sawCheck {
		t.Fatal("release Init did not run a check")
	}
	return m
}

func TestDevInitDoesNotCheck(t *testing.T) {
	setVersion(t, "dev")
	var checks int
	m := editor(t)
	m.check = func(context.Context) (update.Result, error) {
		checks++
		return update.Result{}, errors.New("check must not run")
	}
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.BatchMsg); ok {
		t.Fatal("development Init must not batch a check")
	}
	if _, ok := msg.(pollMsg); !ok {
		t.Fatalf("Init msg = %T, want pollMsg", msg)
	}
	if checks != 0 {
		t.Fatalf("check ran %d times", checks)
	}
	if m.notice != "" {
		t.Fatalf("notice = %q", m.notice)
	}
}

func TestNewerNotice(t *testing.T) {
	setVersion(t, "0.1.0")
	c := stateClient(t)
	m := editor(t)
	m.check = newerCheck("0.2.0")
	m.hidden = c.Hidden
	m.dismiss = c.Dismiss
	m = applyReleaseInit(t, m)
	want := "newer 0.2.0 · herdr-canvas update · i dismiss"
	if m.notice != "0.2.0" {
		t.Fatalf("notice = %q", m.notice)
	}
	got := screen(m)
	if !strings.Contains(got, want) {
		t.Fatalf("edit view missing notice: %q", got)
	}
	if !strings.Contains(got, "[box]") {
		t.Fatalf("edit view dropped footer chips: %q", got)
	}
	lines := strings.Split(got, "\n")
	if !strings.Contains(lines[len(lines)-1], want) {
		t.Fatalf("notice must be its own last row: %q", lines[len(lines)-1])
	}
	if !strings.Contains(lines[len(lines)-2], "[box]") {
		t.Fatalf("footer chips must stay above the notice: %q", lines[len(lines)-2])
	}
	m.phase = phasePick
	if got := screen(m); !strings.Contains(got, want) {
		t.Fatalf("pick view missing notice: %q", got)
	}
}

func TestCheckErrorIsSilent(t *testing.T) {
	setVersion(t, "0.1.0")
	m := editor(t)
	m.status = "keep-me"
	m.check = func(context.Context) (update.Result, error) {
		return update.Result{}, errors.New("boom")
	}
	m = applyReleaseInit(t, m)
	if m.notice != "" {
		t.Fatalf("notice = %q", m.notice)
	}
	if m.status != "keep-me" {
		t.Fatalf("status = %q, check errors must not write status", m.status)
	}
	if strings.Contains(screen(m), "newer") {
		t.Fatalf("view leaked an update notice: %q", screen(m))
	}
}

func TestSameTagStaysHidden(t *testing.T) {
	setVersion(t, "0.1.0")
	c := stateClient(t)
	if err := c.Dismiss("0.2.0"); err != nil {
		t.Fatalf("Dismiss: %v", err)
	}
	m := editor(t)
	m.check = newerCheck("0.2.0")
	m.hidden = c.Hidden
	m.dismiss = c.Dismiss
	m = applyReleaseInit(t, m)
	if m.notice != "" {
		t.Fatalf("dismissed tag still shown: %q", m.notice)
	}
	if strings.Contains(screen(m), "newer") {
		t.Fatalf("view shows a dismissed notice: %q", screen(m))
	}
}

func TestDismissIInPickAndEdit(t *testing.T) {
	setVersion(t, "0.1.0")
	c := stateClient(t)
	m := editor(t)
	m.check = newerCheck("0.2.0")
	m.hidden = c.Hidden
	m.dismiss = c.Dismiss
	m = applyReleaseInit(t, m)

	m.phase = phasePick
	m = send(t, m, key("i"))
	if m.notice != "" {
		t.Fatalf("pick i left notice %q", m.notice)
	}
	if m.phase != phasePick {
		t.Fatalf("phase = %v, want pick", m.phase)
	}
	got, err := c.DismissedVersion()
	if err != nil {
		t.Fatalf("DismissedVersion: %v", err)
	}
	if got != "0.2.0" {
		t.Fatalf("persisted %q, want 0.2.0", got)
	}

	c2 := stateClient(t)
	m = editor(t)
	m.check = newerCheck("0.2.0")
	m.hidden = c2.Hidden
	m.dismiss = c2.Dismiss
	m = applyReleaseInit(t, m)
	if m.phase != phaseEdit {
		t.Fatal("want edit")
	}
	m = send(t, m, key("i"))
	if m.notice != "" {
		t.Fatalf("edit i left notice %q", m.notice)
	}
	if m.phase != phaseEdit {
		t.Fatalf("phase = %v, want edit", m.phase)
	}
	got, err = c2.DismissedVersion()
	if err != nil {
		t.Fatalf("DismissedVersion: %v", err)
	}
	if got != "0.2.0" {
		t.Fatalf("persisted %q, want 0.2.0", got)
	}
}

func TestDismissKeepsNoticeWhenPersistFails(t *testing.T) {
	setVersion(t, "0.1.0")
	m := editor(t)
	m.check = newerCheck("0.2.0")
	m.hidden = func(string) (bool, error) { return false, nil }
	m.dismiss = func(string) error { return errors.New("disk full") }
	m = applyReleaseInit(t, m)
	m = send(t, m, key("i"))
	if m.notice != "0.2.0" {
		t.Fatalf("failed persist cleared notice: %q", m.notice)
	}
	if m.phase != phaseEdit {
		t.Fatalf("phase = %v, want edit", m.phase)
	}
	got := screen(m)
	if !strings.Contains(got, "newer 0.2.0 · herdr-canvas update · i dismiss") {
		t.Fatalf("failed persist hid the notice: %q", got)
	}
	if !strings.Contains(got, "[box]") {
		t.Fatalf("failed persist dropped footer chips: %q", got)
	}
}

func TestTypingAndNamingKeepI(t *testing.T) {
	setVersion(t, "0.1.0")
	c := stateClient(t)
	m := editor(t)
	m.check = newerCheck("0.2.0")
	m.hidden = c.Hidden
	m.dismiss = c.Dismiss
	m = applyReleaseInit(t, m)

	m.tool = toolText
	m = send(t, m, leftDown(3, 2), key("i"))
	if !m.typing {
		t.Fatal("want typing")
	}
	if m.textBuf != "i" {
		t.Fatalf("textBuf = %q, want i", m.textBuf)
	}
	if m.notice != "0.2.0" {
		t.Fatalf("typing dismissed the notice: %q", m.notice)
	}

	m.typing = false
	m.textBuf = ""
	m.phase = phaseName
	m.nameInput = ""
	m = send(t, m, key("i"))
	if m.nameInput != "i" {
		t.Fatalf("nameInput = %q, want i", m.nameInput)
	}
	if m.notice != "0.2.0" {
		t.Fatalf("naming dismissed the notice: %q", m.notice)
	}
	if m.phase != phaseName {
		t.Fatalf("phase = %v", m.phase)
	}
}

func TestPickerNStillNewDiagram(t *testing.T) {
	setVersion(t, "0.1.0")
	c := stateClient(t)
	m := editor(t)
	m.check = newerCheck("0.2.0")
	m.hidden = c.Hidden
	m.dismiss = c.Dismiss
	m = applyReleaseInit(t, m)
	m.phase = phasePick
	m.names = []string{"demo"}
	m = send(t, m, key("n"))
	if m.phase != phaseName {
		t.Fatalf("phase = %v, want phaseName", m.phase)
	}
	if m.notice != "0.2.0" {
		t.Fatalf("n dismissed the notice: %q", m.notice)
	}
}
