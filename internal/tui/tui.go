package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	bkey "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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

var (
	keyUndo    = bkey.NewBinding(bkey.WithKeys("ctrl+z"))
	keyRedo    = bkey.NewBinding(bkey.WithKeys("ctrl+shift+z", "ctrl+y"))
	keyZoomIn  = bkey.NewBinding(bkey.WithKeys("+", "="))
	keyZoomOut = bkey.NewBinding(bkey.WithKeys("-"))
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
	SendText(paneID, text string) error
	Focus(paneID string) error
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
	vp            viewport
	hist          history
	ch            chrome
	panning       bool
	panStart      [2]int
	panOrigin     [2]int

	tool          tool
	cursor        [2]int
	mouse         bool
	anchored      bool
	anchor        [2]int
	grabID        string
	pending       []canvas.Cell
	typing        bool
	textPos       [2]int
	textBuf       string
	editID        string
	lastClickCell [2]int
	lastClickAt   time.Time
	status        string

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
		vp:   viewport{zoom: 10},
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
			// Write the new diagram at once. An agent that gets the diagram
			// reads it by name, and a diagram that only exists in memory
			// makes that read fail.
			if serr := s.Save(d); serr != nil {
				return serr
			}
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
		vp: viewport{zoom: 10},
	})
}

func run(m model) error {
	_, err := tea.NewProgram(m).Run()
	return err
}

// pollEvery is how often the editor looks for a change an agent made. A stat
// of one file is cheap, and the person watches the canvas while the agent
// draws, so the picture must not wait for their next edit.
const pollEvery = 400 * time.Millisecond

// pollMsg asks the editor to look at the stored file again.
type pollMsg struct{}

func poll() tea.Cmd {
	return tea.Tick(pollEvery, func(time.Time) tea.Msg { return pollMsg{} })
}

func (m model) Init() tea.Cmd { return poll() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pollMsg:
		// An in-progress drag, keyboard anchor, or text entry holds state that
		// points at the current elements. Reloading under it would move the
		// ground the person is drawing on, so the poll waits.
		if m.phase == phaseEdit && !m.busy() {
			m.reloadIfChanged()
		}
		return m, poll()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ensureVisible()
		return m, nil
	case tea.KeyPressMsg:
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
			if m.typing && bkey.Matches(msg, keyUndo) {
				m.doUndo()
				return m, nil
			}
			if m.typing && bkey.Matches(msg, keyRedo) {
				m.doRedo()
				return m, nil
			}
			if m.typing {
				cmd := m.typeKey(msg)
				return m, cmd
			}
			cmd := m.editKey(msg)
			return m, cmd
		}
	case tea.MouseMsg:
		if m.phase != phaseEdit {
			return m, nil
		}
		cmd := m.mouseRoute(msg)
		return m, cmd
	}
	return m, nil
}

func (m *model) pickKey(msg tea.KeyPressMsg) tea.Cmd {
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
	case "esc":
		// Only a diagram that exists can be gone back to. The picker is the
		// first screen when the canvas opens outside a git repository.
		if m.d != nil && m.d.Name != "" {
			m.phase = phaseEdit
			m.status = ""
		}
	case "q", "ctrl+c":
		return tea.Quit
	}
	return nil
}

// toPicker leaves the editor for the diagram list. It reads the store again,
// because another diagram can appear or go while the editor is open. It puts
// the cursor on the diagram the editor was showing.
func (m *model) toPicker() {
	names, err := m.s.List()
	if err != nil {
		m.status = err.Error()
		return
	}
	m.names = names
	m.sel = 0
	for i, n := range names {
		if n == m.d.Name {
			m.sel = i
			break
		}
	}
	m.status = ""
	m.phase = phasePick
}

func (m *model) nameKey(msg tea.KeyPressMsg) tea.Cmd {
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
		if msg.Text != "" {
			m.nameInput += msg.Text
		}
	}
	return nil
}

func (m *model) typeKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		m.commitTyping()
	case "esc":
		m.typing = false
		m.textBuf = ""
		m.editID = ""
	case "backspace":
		if len(m.textBuf) > 0 {
			m.textBuf = m.textBuf[:len(m.textBuf)-1]
		}
	default:
		if msg.Text != "" {
			m.textBuf += msg.Text
		}
	}
	return nil
}

func (m *model) editKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case bkey.Matches(msg, keyUndo):
		m.doUndo()
		return nil
	case bkey.Matches(msg, keyRedo):
		m.doRedo()
		return nil
	case bkey.Matches(msg, keyZoomIn):
		m.vp.zoomIn()
		return nil
	case bkey.Matches(msg, keyZoomOut):
		m.vp.zoomOut()
		return nil
	}
	switch msg.String() {
	case "b", "l", "a", "t", "d", "m", "x":
		m.switchTool(toolFor(msg.String()))
	case "up":
		m.moveCursor(0, -1)
	case "down":
		m.moveCursor(0, 1)
	case "left":
		m.moveCursor(-1, 0)
	case "right":
		m.moveCursor(1, 0)
	case "space", "enter":
		m.keyCommit(msg.String() == "enter")
	case "s":
		m.startSend()
	case "esc":
		m.toPicker()
	case "q", "ctrl+c":
		m.save()
		return tea.Quit
	}
	return nil
}

func (m *model) onCanvas(x, y int) bool {
	_, ok := m.vp.canvasPoint(x, y, m.width, m.height)
	return ok
}

func (m *model) startPan(x, y int) {
	m.panning = true
	m.panStart = [2]int{x, y}
	m.panOrigin = m.vp.origin
}

func (m *model) mouseRoute(msg tea.MouseMsg) tea.Cmd {
	ev := msg.Mouse()
	switch msg.(type) {
	case tea.MouseWheelMsg:
		if !m.onCanvas(ev.X, ev.Y) {
			return nil
		}
		if ev.Mod.Contains(tea.ModCtrl) {
			if ev.Button == tea.MouseWheelUp {
				m.vp.zoomIn()
			} else if ev.Button == tea.MouseWheelDown {
				m.vp.zoomOut()
			}
			return nil
		}
		dx, dy := 0, 0
		switch ev.Button {
		case tea.MouseWheelUp:
			dy = -1
		case tea.MouseWheelDown:
			dy = 1
		case tea.MouseWheelLeft:
			dx = -1
		case tea.MouseWheelRight:
			dx = 1
		}
		if ev.Mod.Contains(tea.ModShift) {
			if dx == 0 {
				dx, dy = dy, 0
			}
		}
		m.vp.pan(dx, dy)
		return nil
	case tea.MouseClickMsg:
		if ev.Button == tea.MouseMiddle && m.onCanvas(ev.X, ev.Y) {
			m.startPan(ev.X, ev.Y)
			return nil
		}
		if ev.Button == tea.MouseLeft && (ev.Y == 0 || ev.Y == m.height-1) {
			return m.chromeClick(ev.X, ev.Y)
		}
		if m.typing && ev.Button == tea.MouseLeft && ev.Y == m.height-1 {
			return m.chromeClick(ev.X, ev.Y)
		}
		p, on := m.canvasPoint(ev.X, ev.Y)
		if m.typing && ev.Button == tea.MouseLeft {
			if on && m.inTextBuffer(p) {
				return nil
			}
			m.commitTyping()
			return nil
		}
		if ev.Button == tea.MouseLeft && on && m.tool == toolText {
			if e := m.d.ElementAt(p[0], p[1]); e != nil && e.Type == canvas.Text {
				dbl := m.isDoubleClick(p)
				if dbl || (m.typing && m.textBuf == "" && m.textPos == p && m.editID == "") {
					m.startTextEdit(e)
					return nil
				}
				if m.typing && !m.inTextBuffer(p) {
					m.commitTyping()
				}
				return nil
			}
		}
		m.status = ""
		return m.mouseMsg(msg)
	case tea.MouseMotionMsg:
		if !m.panning && ev.Button == tea.MouseMiddle && m.onCanvas(ev.X, ev.Y) {
			m.startPan(ev.X, ev.Y)
		}
		if m.panning {
			t := m.vp.tenths()
			dx := (ev.X - m.panStart[0]) * 10 / t
			dy := (ev.Y - m.panStart[1]) * 10 / t
			m.vp.origin[0] = max(0, m.panOrigin[0]-dx)
			m.vp.origin[1] = max(0, m.panOrigin[1]-dy)
			return nil
		}
		if m.typing {
			return nil
		}
		return m.mouseMsg(msg)
	case tea.MouseReleaseMsg:
		if ev.Button == tea.MouseMiddle {
			m.panning = false
			return nil
		}
		if m.typing {
			return nil
		}
		return m.mouseMsg(msg)
	}
	return nil
}

func (m *model) chromeClick(x, y int) tea.Cmd {
	ch := layoutChrome(m.width, m.d.Name, m.vp.zoom, m.cursor, m.tool, m.hist.canUndo(), m.hist.canRedo(), m.badge())
	hit, ok := ch.hit(x, y, m.width, m.height)
	if !ok || !hit.enabled {
		return nil
	}
	switch hit.kind {
	case chipTool:
		m.switchTool(hit.tool)
	case chipUndo:
		m.doUndo()
	case chipRedo:
		m.doRedo()
	case chipSend:
		m.startSend()
	case chipZoom:
		m.vp.setZoom(zoom1)
	case chipRecenter:
		m.vp.recenter(m.d.Elements, m.width, m.canvasHeight())
	}
	return nil
}

func (m *model) switchTool(t tool) {
	if m.typing {
		m.commitTyping()
	}
	m.tool = t
	m.grabID = ""
	m.mouse = false
	m.anchored = false
	m.anchor = [2]int{}
	m.pending = nil
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
	ev := msg.Mouse()
	p, ok := m.canvasPoint(ev.X, ev.Y)
	if !ok {
		return nil
	}
	m.cursor = p
	if ev.Button != tea.MouseLeft {
		return nil
	}
	switch msg.(type) {
	case tea.MouseClickMsg:
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
	case tea.MouseMotionMsg:
		if m.tool == toolDraw && m.mouse {
			m.addDrawCell(m.cursor)
		}
	case tea.MouseReleaseMsg:
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
// whose top-left cell is the viewport origin. canvasPoint returns false for a
// position outside the canvas.
func (m model) canvasPoint(x, y int) ([2]int, bool) {
	return m.vp.canvasPoint(x, y, m.width, m.height)
}

// ensureVisible moves the viewport to keep the cursor inside the viewport.
func (m *model) ensureVisible() {
	m.vp.ensureVisible(m.cursor, m.width, m.height)
}

func (m *model) apply(cmd canvas.Command) {
	m.reloadIfChanged()
	before := cloneDiagram(m.d)
	if err := m.d.Apply(cmd); err != nil {
		m.status = err.Error()
		return
	}
	m.hist.push(before)
	m.save()
}

func (m *model) restore(d *canvas.Diagram) {
	m.d = d
	m.save()
}

func (m *model) doUndo() {
	if m.discardInFlight() {
		return
	}
	got, ok := m.hist.undo(m.d)
	if !ok {
		return
	}
	m.restore(got)
}

func (m *model) doRedo() {
	if m.discardInFlight() {
		return
	}
	got, ok := m.hist.redo(m.d)
	if !ok {
		return
	}
	m.restore(got)
}

func (m *model) discardInFlight() bool {
	if m.typing {
		m.commitTyping()
		return true
	}
	if m.mouse || m.anchored || len(m.pending) > 0 || m.grabID != "" {
		m.mouse = false
		m.anchored = false
		m.anchor = [2]int{}
		m.pending = nil
		m.grabID = ""
		return true
	}
	return false
}

// reloadIfChanged replaces the in-memory diagram when the stored file changed
// since this session read it, so a CLI edit is not overwritten.
// busy reports whether the person is part-way through an interaction.
func (m model) busy() bool {
	return m.mouse || m.anchored || m.typing || len(m.pending) > 0 || m.grabID != ""
}

func (m model) inTextBuffer(p [2]int) bool {
	if !m.typing {
		return false
	}
	n := len([]rune(m.textBuf))
	return p[1] == m.textPos[1] && p[0] >= m.textPos[0] && p[0] <= m.textPos[0]+n
}

func (m *model) isDoubleClick(p [2]int) bool {
	ok := m.lastClickCell == p && time.Since(m.lastClickAt) < 400*time.Millisecond
	m.lastClickCell = p
	m.lastClickAt = time.Now()
	return ok
}

func (m *model) startTextEdit(e *canvas.Element) {
	if m.typing {
		if m.editID != e.ID || m.textBuf != e.Text {
			m.commitTyping()
		}
	}
	m.tool = toolText
	m.editID = e.ID
	m.textPos = [2]int{e.X, e.Y}
	m.textBuf = e.Text
	m.typing = true
}

func (m *model) commitTyping() {
	if !m.typing {
		return
	}
	if m.textBuf == "" {
		if m.editID != "" {
			m.apply(canvas.DeleteCmd{ID: m.editID})
		}
	} else if m.editID != "" {
		m.apply(canvas.TextSetCmd{ID: m.editID, Text: m.textBuf})
	} else {
		m.apply(canvas.TextCmd{X: m.textPos[0], Y: m.textPos[1], Text: m.textBuf})
	}
	m.typing = false
	m.textBuf = ""
	m.editID = ""
}

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
	m.hist.push(m.d)
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

func (m model) View() tea.View {
	var body string
	switch m.phase {
	case phasePick:
		body = m.pickView()
	case phaseName:
		body = fmt.Sprintf("name: %s\n\nenter create · esc back\n%s\n", m.nameInput, m.status)
	case phaseAgent:
		body = m.agentView()
	default:
		body = m.editView()
	}
	v := tea.NewView(body)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m model) pickView() string {
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
	b.WriteString("\n↑/↓ choose · enter open · n new · esc back · q quit\n")
	if m.status != "" {
		b.WriteString(m.status)
		b.WriteString("\n")
	}
	return b.String()
}

func (m *model) badge() string {
	if m.typing {
		return ""
	}
	if m.grabID != "" {
		return fmt.Sprintf("moving %s %+d%+d", m.grabID, m.cursor[0]-m.anchor[0], m.cursor[1]-m.anchor[1])
	}
	if m.mouse || m.anchored {
		return fmt.Sprintf("%dx%d", abs(m.cursor[0]-m.anchor[0])+1, abs(m.cursor[1]-m.anchor[1])+1)
	}
	return ""
}

func (m model) editView() string {
	g := m.d.Render()
	if m.mouse || m.anchored || m.typing || len(m.pending) > 0 {
		g = m.overlayPreview(g)
	}
	ch := layoutChrome(m.width, m.d.Name, m.vp.zoom, m.cursor, m.tool, m.hist.canUndo(), m.hist.canRedo(), m.badge())
	var b strings.Builder
	b.WriteString(ch.header)
	b.WriteString("\n")
	b.WriteString(m.vp.paint(g, m.width, m.canvasHeight()))
	b.WriteString("\n")
	b.WriteString(styleChromeFooter(ch, m.tool, m.hist.canUndo(), m.hist.canRedo()))
	return b.String()
}

func styleChromeFooter(ch chrome, active tool, canUndo, canRedo bool) string {
	runes := []rune(ch.footer)
	var out strings.Builder
	pos := 0
	for _, c := range ch.chips {
		if c.row != 1 {
			continue
		}
		if c.x0 > pos {
			out.WriteString(string(runes[pos:c.x0]))
		}
		seg := string(runes[c.x0:c.x1])
		style := lipgloss.NewStyle()
		switch {
		case c.kind == chipTool && c.tool == active:
			style = style.Reverse(true)
		case (c.kind == chipUndo && !canUndo) || (c.kind == chipRedo && !canRedo):
			style = style.Faint(true)
		}
		out.WriteString(style.Render(seg))
		pos = c.x1
	}
	if pos < len(runes) {
		out.WriteString(string(runes[pos:]))
	}
	return out.String()
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
		if m.editID != "" {
			filtered := elems[:0]
			for _, e := range elems {
				if e.ID != m.editID {
					filtered = append(filtered, e)
				}
			}
			elems = filtered
		}
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
	return fmt.Sprintf("[%s] @(%d,%d)%s   b/l/a/t/d/m/x · drag · s send · esc picker · q quit",
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

// sendTo writes the diagram into an agent's input and focuses that agent. It
// does not submit the message. The person adds their own words and presses
// enter.
func (m *model) sendTo(a herdr.Agent) {
	if err := m.send.SendText(a.PaneID, canvas.Prompt(m.d)); err != nil {
		m.status = err.Error()
		return
	}
	// A failed focus does not undo a message that already landed.
	_ = m.send.Focus(a.PaneID)
	m.status = fmt.Sprintf("added %s to %s — add your words and press enter", m.d.Name, agentTab(a))
}

func (m *model) agentKey(msg tea.KeyPressMsg) tea.Cmd {
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
	b.WriteString("send " + m.d.Name + " to which agent?")
	if ws := m.agentWorkspace(); ws != "" {
		b.WriteString("   workspace: " + ws)
	}
	b.WriteString("\n\n")

	tabW := 3
	for _, a := range m.agents {
		if n := len([]rune(agentTab(a))); n > tabW {
			tabW = n
		}
	}
	for i, a := range m.agents {
		if i == m.agentSel {
			b.WriteString("> ")
		} else {
			b.WriteString("  ")
		}
		fmt.Fprintf(&b, "%-*s  %-8s  %-7s  %s\n",
			tabW, agentTab(a), a.Agent, a.Status, clip(a.Title, m.width-tabW-24))
	}
	b.WriteString("\n↑/↓ choose · enter send · esc cancel\n")
	return b.String()
}

// agentTab names the tab of an agent the way the herdr tab bar names it. A
// pane id is the fallback, because a tab label can be absent.
func agentTab(a herdr.Agent) string {
	if a.TabLabel != "" {
		return a.TabLabel
	}
	return a.PaneID
}

func (m model) agentWorkspace() string {
	for _, a := range m.agents {
		if a.WorkspaceLabel != "" {
			return a.WorkspaceLabel
		}
	}
	return ""
}

func clip(s string, w int) string {
	if w < 4 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}
