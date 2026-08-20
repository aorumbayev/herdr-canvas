package tui

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"herdr-canvas/internal/canvas"
	"herdr-canvas/internal/herdr"
	"herdr-canvas/internal/store"
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
	return model{s: s, d: d, mtime: mt, phase: phaseEdit, tool: toolBox, width: 40, height: 12}
}

func send(t *testing.T, m model, msgs ...tea.Msg) model {
	t.Helper()
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(model)
	}
	return m
}

func mouse(x, y int, action tea.MouseAction, button tea.MouseButton) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Action: action, Button: button}
}

func left(x, y int, action tea.MouseAction) tea.MouseMsg {
	return mouse(x, y, action, tea.MouseButtonLeft)
}

func key(s string) tea.KeyMsg {
	if s == " " {
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestCanvasPointUsesFixedOrigin(t *testing.T) {
	m := editor(t)
	m.origin = [2]int{5, 7}
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
		left(10, 5, tea.MouseActionPress),
		left(14, 8, tea.MouseActionMotion),
		left(14, 8, tea.MouseActionRelease),
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

func TestWheelDoesNotDeleteOrDraw(t *testing.T) {
	m := editor(t)
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 4, Y2: 4}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	m.tool = toolDelete
	for _, b := range []tea.MouseButton{tea.MouseButtonWheelUp, tea.MouseButtonWheelDown, tea.MouseButtonRight, tea.MouseButtonMiddle} {
		m = send(t, m, mouse(2, 3, tea.MouseActionPress, b))
		if len(m.d.Elements) != 1 {
			t.Fatalf("button %v deleted the element", b)
		}
	}
	m.tool = toolBox
	m = send(t, m, mouse(2, 3, tea.MouseActionPress, tea.MouseButtonWheelUp), mouse(6, 6, tea.MouseActionRelease, tea.MouseButtonWheelUp))
	if len(m.d.Elements) != 1 {
		t.Errorf("wheel committed a box: %d elements", len(m.d.Elements))
	}
}

func TestReleaseWithoutPressCommitsNothing(t *testing.T) {
	m := editor(t)
	m.anchor = [2]int{2, 2}
	m = send(t, m, left(9, 9, tea.MouseActionRelease))
	if len(m.d.Elements) != 0 {
		t.Errorf("elements = %d, want 0", len(m.d.Elements))
	}
}

func TestToolSwitchClearsAnchorAndPending(t *testing.T) {
	m := editor(t)
	m.tool = toolDraw
	m = send(t, m, left(3, 3, tea.MouseActionPress), left(4, 3, tea.MouseActionMotion))
	if len(m.pending) == 0 {
		t.Fatal("draw collected no cells")
	}
	m = send(t, m, key("b"))
	if m.pending != nil || m.anchored || m.mouse || m.anchor != [2]int{} {
		t.Errorf("tool switch left state: pending=%v anchored=%v mouse=%v anchor=%v",
			m.pending, m.anchored, m.mouse, m.anchor)
	}
	m = send(t, m, left(9, 9, tea.MouseActionRelease))
	if len(m.d.Elements) != 0 {
		t.Errorf("stale anchor committed an element: %+v", m.d.Elements)
	}
}

func TestKeyboardAnchorAndCommitDrawsBox(t *testing.T) {
	m := editor(t)
	m = send(t, m,
		key(" "),
		tea.KeyMsg{Type: tea.KeyRight}, tea.KeyMsg{Type: tea.KeyRight},
		tea.KeyMsg{Type: tea.KeyDown},
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
	m = send(t, m, left(3, 3, tea.MouseActionPress), left(6, 6, tea.MouseActionRelease))
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
	m = send(t, m, tea.KeyMsg{Type: tea.KeyEnter})
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
	if got := m.View(); !strings.Contains(got, "permission denied") {
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

func TestViewFillsTheTerminal(t *testing.T) {
	m := editor(t)
	m = send(t, m, tea.WindowSizeMsg{Width: 40, Height: 12})
	if err := m.d.Apply(canvas.BoxCmd{X1: 0, Y1: 0, X2: 3, Y2: 30}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	lines := strings.Split(m.View(), "\n")
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
	lines := strings.Split(m.View(), "\n")
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
	m = send(t, m, left(1, 1, tea.MouseActionPress), left(6, 1, tea.MouseActionMotion))

	if m.d.Elements[0].X1 != 0 {
		t.Errorf("the drag mutated the diagram before release: %+v", m.d.Elements[0])
	}
	ghost := strings.Repeat(ghostGlyph, 3)
	if got, want := canvasLine(t, m, 0), ghost+"  ┌─┐"; got != want {
		t.Errorf("row 0 = %q, want %q (ghost at the source, box under the cursor)", got, want)
	}

	m = send(t, m, left(6, 1, tea.MouseActionRelease))
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
	m = send(t, m, left(3, 2, tea.MouseActionPress), key("h"), key("i"))

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

	m = send(t, m, tea.KeyMsg{Type: tea.KeyEnter})
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
	m = send(t, m, left(3, 2, tea.MouseActionPress), key("h"), tea.KeyMsg{Type: tea.KeyEsc})
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
				left(0, headerRows, tea.MouseActionPress),
				left(c.to[0], c.to[1]+headerRows, tea.MouseActionMotion),
			)
			for i, want := range c.wantRows {
				if got := canvasLine(t, m, i); got != want {
					t.Errorf("preview row %d = %q, want %q", i, got, want)
				}
			}
			m = send(t, m, left(c.to[0], c.to[1]+headerRows, tea.MouseActionRelease))
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
	sendErr  error
}

func (f *fakeSender) Agents(string) ([]herdr.Agent, error) { return f.agents, f.listErr }

func (f *fakeSender) Prompt(paneID, text string) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sentTo, f.sentText = paneID, text
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
	if !strings.Contains(f.sentText, "┌───┐") {
		t.Errorf("the prompt carries no diagram:\n%s", f.sentText)
	}
	if !strings.Contains(f.sentText, "herdr-canvas --name \"demo\" export") {
		t.Errorf("the prompt does not say how to read the diagram back:\n%s", f.sentText)
	}
	if !strings.Contains(m.status, "sent demo") {
		t.Errorf("status = %q, want a sent report", m.status)
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
	view := m.View()
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
	m = send(t, m, key("s"), tea.KeyMsg{Type: tea.KeyEsc})
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
	view := m.View()
	for _, want := range []string{"control", "review-prose", "ship", "workspace: herdr-canvas"} {
		if !strings.Contains(view, want) {
			t.Errorf("picker does not name %q:\n%s", want, view)
		}
	}
	if os.Getenv("SHOW_PICKER") != "" {
		t.Log("\n" + view)
	}
}
