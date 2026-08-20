package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"herdr-canvas/internal/canvas"
	"herdr-canvas/internal/herdr"
	"herdr-canvas/internal/name"
	"herdr-canvas/internal/store"
)

type phase int

const (
	phasePick phase = iota
	phaseName
	phaseEdit
	phaseAgent
)

type tool int

const (
	toolBox tool = iota
	toolLine
	toolArrow
	toolText
	toolDraw
	toolMove
	toolDelete
)

var toolNames = map[tool]string{
	toolBox:    "box",
	toolLine:   "line",
	toolArrow:  "arrow",
	toolText:   "text",
	toolDraw:   "draw",
	toolMove:   "move",
	toolDelete: "delete",
}

// The editor fills the whole terminal. The editor shows one header row, then
// the canvas, then one status row.
const (
	headerRows = 1
	statusRows = 1
	defaultW   = 80
	defaultH   = 24
)

// cursorGlyph marks the insertion point of the text tool. ghostGlyph marks the
// place a dragged element came from.
const (
	cursorGlyph = "█"
	ghostGlyph  = "·"
)

// sender delivers a diagram to an agent pane. The editor holds the interface,
// not the herdr client, so a test can drive the send without a herdr server.
type sender interface {
	Agents(workspace string) ([]herdr.Agent, error)
	Prompt(paneID, text string) error
}

type model struct {
	s     *store.Store
	d     *canvas.Diagram
	phase phase
	mtime time.Time

	names []string
	sel   int

	nameInput string

	width, height int
	origin        [2]int

	tool     tool
	cursor   [2]int
	mouse    bool
	anchored bool
	anchor   [2]int
	grabID   string
	pending  []canvas.Cell
	typing   bool
	textPos  [2]int
	textBuf  string
	status   string

	send      sender
	workspace string
	agents    []herdr.Agent
	agentSel  int
}

// Run starts the TUI. Run opens the editor for the composite diagram in cwd.
// If cwd is not a git repository, Run opens the picker.
func Run(cwd string) error {
	s := store.New()
	m := model{s: s, d: &canvas.Diagram{}, tool: toolBox, width: defaultW, height: defaultH,
		send: herdr.New(), workspace: herdr.Workspace()}

	nm, err := name.Composite(cwd)
	if err != nil {
		m.phase = phasePick
		names, lerr := s.List()
		if lerr != nil {
			m.status = lerr.Error()
		}
		m.names = names
	} else {
		n := nm.String()
		d, lerr := s.Load(n)
		if lerr != nil {
			if !os.IsNotExist(lerr) {
				return lerr
			}
			d = &canvas.Diagram{Name: n}
		}
		m.d = d
		m.mtime, _ = s.ModTime(n)
		m.phase = phaseEdit
	}

	return run(m)
}

// RunNamed opens an existing named diagram in the editor.
func RunNamed(n string) error {
	s := store.New()
	d, err := s.Load(n)
	if err != nil {
		return err
	}
	mt, _ := s.ModTime(n)
	return run(model{
		s: s, d: d, mtime: mt, phase: phaseEdit, tool: toolBox,
		width: defaultW, height: defaultH,
	})
}

func run(m model) error {
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ensureVisible()
		return m, nil
	case tea.KeyMsg:
		m.status = ""
		switch m.phase {
		case phasePick:
			cmd := m.pickKey(msg)
			return m, cmd
		case phaseName:
			cmd := m.nameKey(msg)
			return m, cmd
		case phaseAgent:
			cmd := m.agentKey(msg)
			return m, cmd
		case phaseEdit:
			if m.typing {
				cmd := m.typeKey(msg)
				return m, cmd
			}
			cmd := m.editKey(msg)
			return m, cmd
		}
	case tea.MouseMsg:
		if m.phase == phaseEdit && !m.typing {
			m.status = ""
			cmd := m.mouseMsg(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *model) pickKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.sel > 0 {
			m.sel--
		}
	case "down", "j":
		if m.sel < len(m.names)-1 {
			m.sel++
		}
	case "enter":
		if m.sel < len(m.names) {
			if d, err := m.s.Load(m.names[m.sel]); err == nil {
				m.d = d
				m.mtime, _ = m.s.ModTime(d.Name)
				m.phase = phaseEdit
			} else {
				m.status = err.Error()
			}
		}
	case "n":
		m.phase = phaseName
		m.nameInput = ""
	case "q", "ctrl+c":
		return tea.Quit
	}
	return nil
}

func (m *model) nameKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		if m.nameInput == "" {
			return nil
		}
		if _, err := m.s.Load(m.nameInput); err == nil {
			m.status = fmt.Sprintf("diagram %q already exists", m.nameInput)
			return nil
		} else if !os.IsNotExist(err) {
			m.status = err.Error()
			return nil
		}
		d := &canvas.Diagram{Name: m.nameInput}
		if err := m.s.Save(d); err != nil {
			m.status = err.Error()
			return nil
		}
		m.d = d
		m.mtime, _ = m.s.ModTime(d.Name)
		m.phase = phaseEdit
	case "esc":
		m.phase = phasePick
	case "backspace":
		if len(m.nameInput) > 0 {
			m.nameInput = m.nameInput[:len(m.nameInput)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.nameInput += msg.String()
		}
	}
	return nil
}

func (m *model) typeKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		m.apply(canvas.TextCmd{X: m.textPos[0], Y: m.textPos[1], Text: m.textBuf})
		m.typing = false
		m.textBuf = ""
	case "esc":
		m.typing = false
		m.textBuf = ""
	case "backspace":
		if len(m.textBuf) > 0 {
			m.textBuf = m.textBuf[:len(m.textBuf)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.textBuf += msg.String()
		}
	}
	return nil
}

func (m *model) editKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "b", "l", "a", "t", "d", "m", "x":
		m.tool = toolFor(msg.String())
		m.grabID = ""
		m.mouse = false
		m.anchored = false
		m.anchor = [2]int{}
		m.pending = nil
	case "up":
		m.moveCursor(0, -1)
	case "down":
		m.moveCursor(0, 1)
	case "left":
		m.moveCursor(-1, 0)
	case "right":
		m.moveCursor(1, 0)
	case " ", "enter":
		m.keyCommit(msg.String() == "enter")
	case "s":
		m.startSend()
	case "q", "ctrl+c":
		m.save()
		return tea.Quit
	}
	return nil
}

func (m *model) moveCursor(dx, dy int) {
	m.cursor[0] = max(0, m.cursor[0]+dx)
	m.cursor[1] = max(0, m.cursor[1]+dy)
	m.ensureVisible()
}

// keyCommit operates the tools from the keyboard. The first space or enter
// places the anchor. The second space or enter commits the element between the
// anchor and the cursor. The draw tool collects one cell for each space and
// commits the cells on enter.
func (m *model) keyCommit(enter bool) {
	switch m.tool {
	case toolText:
		m.textPos = m.cursor
		m.textBuf = ""
		m.typing = true
	case toolDelete:
		m.deleteAt(m.cursor)
	case toolDraw:
		if enter {
			if len(m.pending) > 0 {
				m.apply(canvas.DrawCmd{Cells: m.drawCells()})
			}
			return
		}
		m.addDrawCell(m.cursor)
	default:
		if !m.anchored {
			m.anchor = m.cursor
			m.anchored = true
			if m.tool == toolMove {
				if e := m.d.ElementAt(m.cursor[0], m.cursor[1]); e != nil {
					m.grabID = e.ID
				} else {
					m.anchored = false
					m.status = "nothing to move here"
				}
			}
			return
		}
		m.anchored = false
		m.commit()
	}
}

func (m *model) mouseMsg(msg tea.MouseMsg) tea.Cmd {
	p, ok := m.canvasPoint(msg.X, msg.Y)
	if !ok {
		return nil
	}
	m.cursor = p
	if msg.Button != tea.MouseButtonLeft {
		return nil
	}
	switch msg.Action {
	case tea.MouseActionPress:
		m.mouse = true
		m.anchor = m.cursor
		switch m.tool {
		case toolDraw:
			m.addDrawCell(m.cursor)
		case toolMove:
			if e := m.d.ElementAt(m.cursor[0], m.cursor[1]); e != nil {
				m.grabID = e.ID
			}
		case toolDelete:
			m.deleteAt(m.cursor)
		case toolText:
			m.textPos = m.cursor
			m.textBuf = ""
			m.typing = true
			m.mouse = false // the release arrives while typing and is dropped
		}
	case tea.MouseActionMotion:
		if m.tool == toolDraw && m.mouse {
			m.addDrawCell(m.cursor)
		}
	case tea.MouseActionRelease:
		if !m.mouse {
			return nil
		}
		m.mouse = false
		m.commit()
	}
	return nil
}

// commit makes an element from the anchor and the cursor. The current tool
// gives the type of the element.
func (m *model) commit() {
	switch m.tool {
	case toolBox:
		x1, y1 := min(m.anchor[0], m.cursor[0]), min(m.anchor[1], m.cursor[1])
		x2, y2 := max(m.anchor[0], m.cursor[0]), max(m.anchor[1], m.cursor[1])
		m.apply(canvas.BoxCmd{X1: x1, Y1: y1, X2: x2, Y2: y2})
	case toolLine, toolArrow:
		start, end := m.snap(m.anchor), m.snap(m.cursor)
		m.apply(canvas.LineCmd{
			X1: start[0], Y1: start[1], X2: end[0], Y2: end[1],
			Arrow: m.lineArrow(),
		})
	case toolDraw:
		if len(m.pending) > 0 {
			m.apply(canvas.DrawCmd{Cells: m.drawCells()})
		}
	case toolMove:
		if m.grabID != "" {
			dx, dy := m.cursor[0]-m.anchor[0], m.cursor[1]-m.anchor[1]
			m.apply(canvas.MoveCmd{ID: m.grabID, DX: dx, DY: dy})
			m.grabID = ""
		}
	}
}

func (m *model) deleteAt(p [2]int) {
	if e := m.d.ElementAt(p[0], p[1]); e != nil {
		m.apply(canvas.DeleteCmd{ID: e.ID})
	}
}

func (m model) canvasHeight() int {
	return max(1, m.height-headerRows-statusRows)
}

// canvasPoint translates a terminal mouse position into a grid coordinate.
// The canvas starts below the header row. The canvas shows the block of cells
// whose top-left cell is m.origin. canvasPoint returns false for a position
// outside the canvas.
func (m model) canvasPoint(x, y int) ([2]int, bool) {
	row := y - headerRows
	if row < 0 || row >= m.canvasHeight() || x < 0 || x >= m.width {
		return [2]int{}, false
	}
	return [2]int{m.origin[0] + x, m.origin[1] + row}, true
}

// ensureVisible moves the viewport to keep the cursor inside the viewport.
func (m *model) ensureVisible() {
	h := m.canvasHeight()
	if m.cursor[0] < m.origin[0] {
		m.origin[0] = m.cursor[0]
	}
	if m.cursor[0] >= m.origin[0]+m.width {
		m.origin[0] = m.cursor[0] - m.width + 1
	}
	if m.cursor[1] < m.origin[1] {
		m.origin[1] = m.cursor[1]
	}
	if m.cursor[1] >= m.origin[1]+h {
		m.origin[1] = m.cursor[1] - h + 1
	}
	m.origin[0] = max(0, m.origin[0])
	m.origin[1] = max(0, m.origin[1])
}

// apply reloads the diagram if another process changed the file. apply then
// sends the command to the gate and saves the diagram.
func (m *model) apply(cmd canvas.Command) {
	m.reloadIfChanged()
	if err := m.d.Apply(cmd); err != nil {
		m.status = err.Error()
		return
	}
	m.save()
}

// reloadIfChanged replaces the in-memory diagram when the stored file changed
// since this session read it, so a CLI edit is not overwritten.
func (m *model) reloadIfChanged() {
	if m.d.Name == "" {
		return
	}
	mt, err := m.s.ModTime(m.d.Name)
	if err != nil {
		m.status = err.Error()
		return
	}
	if mt.Equal(m.mtime) {
		return
	}
	d, err := m.s.Load(m.d.Name)
	if err != nil {
		if !os.IsNotExist(err) {
			m.status = err.Error()
		}
		return
	}
	m.d = d
	m.mtime = mt
	m.status = fmt.Sprintf("%s changed on disk; reloaded", d.Name)
}

func (m *model) save() {
	if err := m.s.Save(m.d); err != nil {
		m.status = err.Error()
		return
	}
	if mt, err := m.s.ModTime(m.d.Name); err == nil {
		m.mtime = mt
	}
}

func (m *model) addDrawCell(p [2]int) {
	m.pending = append(m.pending, canvas.Cell{X: p[0], Y: p[1], Ch: "#"})
}

func (m *model) drawCells() []canvas.Cell {
	cells := m.pending
	m.pending = nil
	return cells
}

// snap returns an occupied cell within a Chebyshev radius of 2 cells of p. If
// no cell in that area is occupied, snap returns p. snap searches one ring at a
// time and returns the first occupied cell in the ring.
func (m model) snap(p [2]int) [2]int {
	g := m.d.Render()
	for r := 1; r <= 2; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if abs(dx) != r && abs(dy) != r {
					continue
				}
				if _, ok := g[[2]int{p[0] + dx, p[1] + dy}]; ok {
					return [2]int{p[0] + dx, p[1] + dy}
				}
			}
		}
	}
	return p
}

func (m model) View() string {
	switch m.phase {
	case phasePick:
		var b strings.Builder
		b.WriteString("herdr-canvas — pick a diagram\n\n")
		for i, n := range m.names {
			if i == m.sel {
				b.WriteString("> ")
			} else {
				b.WriteString("  ")
			}
			b.WriteString(n)
			b.WriteString("\n")
		}
		b.WriteString("\n↑/↓ choose · enter open · n new · q quit\n")
		if m.status != "" {
			b.WriteString(m.status)
			b.WriteString("\n")
		}
		return b.String()
	case phaseName:
		return fmt.Sprintf("name: %s\n\nenter create · esc back\n%s\n", m.nameInput, m.status)
	case phaseAgent:
		return m.agentView()
	default:
		return m.editView()
	}
}

func (m model) editView() string {
	g := m.d.Render()
	if m.mouse || m.anchored || m.typing || len(m.pending) > 0 {
		g = m.overlayPreview(g)
	}
	var b strings.Builder
	b.WriteString(m.d.Name)
	b.WriteString("\n")
	b.WriteString(g.Window(m.origin[0], m.origin[1], m.width, m.canvasHeight()))
	b.WriteString("\n")
	b.WriteString(m.statusLine())
	return b.String()
}

// movePreview replaces the grabbed element with a copy at the dragged offset,
// and leaves a ghost outline where the element came from.
func (m model) movePreview(elems []canvas.Element) []canvas.Element {
	dx, dy := m.cursor[0]-m.anchor[0], m.cursor[1]-m.anchor[1]
	out := make([]canvas.Element, 0, len(elems)+1)
	var grabbed canvas.Element
	found := false
	for _, e := range elems {
		if e.ID == m.grabID {
			grabbed, found = e, true
			continue
		}
		out = append(out, e)
	}
	if !found {
		return elems
	}
	moved, err := canvas.Translate(grabbed, dx, dy)
	if err != nil {
		return elems
	}
	if dx != 0 || dy != 0 {
		out = append(out, canvas.Element{Type: canvas.Freeform, Cells: ghostCells(grabbed)})
	}
	return append(out, moved)
}

// ghostCells renders one element on its own and returns its cells as the ghost
// glyph, so the preview can show where a dragged element came from.
func ghostCells(e canvas.Element) []canvas.Cell {
	d := canvas.Diagram{Elements: []canvas.Element{e}}
	g := d.Render()
	cells := make([]canvas.Cell, 0, len(g))
	for p := range g {
		cells = append(cells, canvas.Cell{X: p[0], Y: p[1], Ch: ghostGlyph})
	}
	return cells
}

func (m model) lineArrow() canvas.Arrow {
	if m.tool == toolArrow {
		return canvas.ArrowEnd
	}
	return canvas.ArrowNone
}

func (m model) overlayPreview(g canvas.Grid) canvas.Grid {
	elems := make([]canvas.Element, len(m.d.Elements), len(m.d.Elements)+1)
	copy(elems, m.d.Elements)
	if m.tool == toolDraw && len(m.pending) > 0 {
		elems = append(elems, canvas.Element{Type: canvas.Freeform, Cells: m.pending})
	}
	if m.tool == toolBox {
		x1, y1 := min(m.anchor[0], m.cursor[0]), min(m.anchor[1], m.cursor[1])
		x2, y2 := max(m.anchor[0], m.cursor[0]), max(m.anchor[1], m.cursor[1])
		elems = append(elems, canvas.Element{Type: canvas.Box, X1: x1, Y1: y1, X2: x2, Y2: y2})
	}
	if m.tool == toolLine || m.tool == toolArrow {
		start, end := m.snap(m.anchor), m.snap(m.cursor)
		elems = append(elems, canvas.Element{
			Type: canvas.Line, X1: start[0], Y1: start[1], X2: end[0], Y2: end[1],
			Arrow: m.lineArrow(),
		})
	}
	if m.tool == toolMove && m.grabID != "" {
		elems = m.movePreview(elems)
	}
	if m.typing {
		elems = append(elems, canvas.Element{Type: canvas.Text, X: m.textPos[0], Y: m.textPos[1], Text: m.textBuf})
		elems = append(elems, canvas.Element{Type: canvas.Freeform, Cells: []canvas.Cell{{
			X: m.textPos[0] + len([]rune(m.textBuf)), Y: m.textPos[1], Ch: cursorGlyph,
		}}})
	}
	tmp := canvas.Diagram{Elements: elems}
	return tmp.Render()
}

func (m model) statusLine() string {
	if m.typing {
		return fmt.Sprintf("[text] @(%d,%d) · enter commit · esc cancel", m.textPos[0], m.textPos[1])
	}
	if m.status != "" {
		return m.status
	}
	extra := ""
	if m.grabID != "" {
		extra = fmt.Sprintf(" · moving %s %+d%+d", m.grabID, m.cursor[0]-m.anchor[0], m.cursor[1]-m.anchor[1])
	} else if m.mouse || m.anchored {
		extra = fmt.Sprintf(" · %dx%d", abs(m.cursor[0]-m.anchor[0])+1, abs(m.cursor[1]-m.anchor[1])+1)
	}
	if m.anchored {
		extra += fmt.Sprintf(" · anchor (%d,%d)", m.anchor[0], m.anchor[1])
	}
	return fmt.Sprintf("[%s] @(%d,%d)%s   b/l/a/t/d/m/x tool · drag mouse · arrows+space · s send · q quit",
		toolNames[m.tool], m.cursor[0], m.cursor[1], extra)
}

func toolFor(k string) tool {
	switch k {
	case "b":
		return toolBox
	case "l":
		return toolLine
	case "a":
		return toolArrow
	case "t":
		return toolText
	case "d":
		return toolDraw
	case "m":
		return toolMove
	case "x":
		return toolDelete
	}
	return toolBox
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// startSend picks the agent that receives the diagram. One agent in the
// workspace receives it at once. More than one opens the picker.
func (m *model) startSend() {
	if m.send == nil {
		m.status = "no herdr client; send needs a herdr pane"
		return
	}
	agents, err := m.send.Agents(m.workspace)
	if err != nil {
		m.status = err.Error()
		return
	}
	if len(agents) == 0 {
		m.status = "no agent in this workspace"
		return
	}
	if len(agents) == 1 {
		m.sendTo(agents[0])
		return
	}
	m.agents = agents
	m.agentSel = 0
	m.phase = phaseAgent
}

func (m *model) sendTo(a herdr.Agent) {
	if err := m.send.Prompt(a.PaneID, canvas.Prompt(m.d)); err != nil {
		m.status = err.Error()
		return
	}
	m.status = fmt.Sprintf("sent %s to %s (%s)", m.d.Name, a.Agent, a.PaneID)
}

func (m *model) agentKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.agentSel > 0 {
			m.agentSel--
		}
	case "down", "j":
		if m.agentSel < len(m.agents)-1 {
			m.agentSel++
		}
	case "enter":
		a := m.agents[m.agentSel]
		m.phase = phaseEdit
		m.sendTo(a)
	case "esc", "q":
		m.phase = phaseEdit
		m.status = "send cancelled"
	}
	return nil
}

func (m model) agentView() string {
	var b strings.Builder
	b.WriteString("send " + m.d.Name + " to which agent?\n\n")
	for i, a := range m.agents {
		if i == m.agentSel {
			b.WriteString("> ")
		} else {
			b.WriteString("  ")
		}
		fmt.Fprintf(&b, "%-9s %-8s %s\n", a.Agent, a.PaneID, a.Status)
	}
	b.WriteString("\n↑/↓ choose · enter send · esc cancel\n")
	return b.String()
}
