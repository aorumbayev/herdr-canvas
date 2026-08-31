package tui

import (
	"context"
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
	"herdr-canvas/internal/update"
	"herdr-canvas/internal/version"
	"herdr-canvas/internal/welcome"
)

type phase int

const (
	phasePick phase = iota
	phaseName
	phaseEdit
	phaseAgent
	phaseHelp
	phasePalette
	phaseWelcome
)

type tool int

const (
	toolSelect tool = iota
	toolBox
	toolLine
	toolArrow
	toolText
	toolDraw
)

var (
	keyUndo      = bkey.NewBinding(bkey.WithKeys("ctrl+z"))
	keyRedo      = bkey.NewBinding(bkey.WithKeys("ctrl+shift+z", "ctrl+y"))
	keySelectAll = bkey.NewBinding(bkey.WithKeys("ctrl+a"))
)

var toolNames = map[tool]string{
	toolSelect: "select",
	toolBox:    "box",
	toolLine:   "line",
	toolArrow:  "arrow",
	toolText:   "text",
	toolDraw:   "draw",
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
	histories     map[string]history
	selections    map[string]map[string]bool
	ch            chrome
	panning       bool
	panStart      [2]int
	panOrigin     [2]int

	tool          tool
	cursor        [2]int
	mouse         bool
	anchored      bool
	anchor        [2]int
	selected      map[string]bool
	selAct        selectAct
	selAdd        bool
	pickConfirm   string
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

	check   func(context.Context) (update.Result, error)
	dismiss func(string) error
	hidden  func(string) (bool, error)
	notice  string

	brushColor string
	brushFill  bool

	welcomeTab     int
	welcomeDismiss bool
	welcomeChecked bool
	welcomeSeen    func() (bool, error)
	welcomeMark    func() error
}

// Run starts the TUI. Run opens the editor for the composite diagram in cwd.
// If cwd is not a git repository, Run opens the picker.
func Run(cwd string) error {
	s := store.New()
	m := model{s: s, d: &canvas.Diagram{}, tool: toolSelect, width: defaultW, height: defaultH,
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
		m.maybeWelcome()
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
	m := model{
		s: s, d: d, mtime: mt, phase: phaseEdit, tool: toolSelect,
		width: defaultW, height: defaultH,
	}
	m.maybeWelcome()
	return run(m)
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

type checkMsg struct {
	res update.Result
	err error
}

func (m model) Init() tea.Cmd {
	if !version.IsRelease() {
		return poll()
	}
	return tea.Batch(poll(), m.checkCmd())
}

func (m model) checkCmd() tea.Cmd {
	check := m.check
	if check == nil {
		check = update.Check
	}
	return func() tea.Msg {
		res, err := check(context.Background())
		return checkMsg{res: res, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case checkMsg:
		if msg.err != nil || !msg.res.Newer || msg.res.Latest == "" {
			return m, nil
		}
		hidden := m.hidden
		if hidden == nil {
			hidden = update.Hidden
		}
		hide, err := hidden(msg.res.Latest)
		if err != nil || hide {
			return m, nil
		}
		m.notice = msg.res.Latest
		return m, nil
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
	case tea.PasteMsg:
		m.acceptPaste(msg.Content)
		return m, nil
	case tea.ClipboardMsg:
		m.acceptPaste(msg.Content)
		return m, nil
	case tea.KeyPressMsg:
		if m.dismissNotice(msg) {
			return m, nil
		}
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
		case phaseHelp:
			cmd := m.helpKey(msg)
			return m, cmd
		case phasePalette:
			cmd := m.paletteKey(msg)
			return m, cmd
		case phaseWelcome:
			cmd := m.welcomeKey(msg)
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
		if m.phase == phasePalette {
			if click, ok := msg.(tea.MouseClickMsg); ok && click.Y == 1 {
				m.paletteClick(click.X)
			}
			return m, nil
		}
		if m.phase != phaseEdit {
			return m, nil
		}
		cmd := m.mouseRoute(msg)
		return m, cmd
	}
	return m, nil
}

func (m *model) dismissNotice(msg tea.KeyPressMsg) bool {
	if m.notice == "" || m.phase == phaseName || m.typing {
		return false
	}
	if msg.String() != "i" {
		return false
	}
	fn := m.dismiss
	if fn == nil {
		fn = update.Dismiss
	}
	if err := fn(m.notice); err != nil {
		return true
	}
	m.notice = ""
	return true
}

func (m model) noticeLine() string {
	if m.notice == "" {
		return ""
	}
	return "newer " + m.notice + " · herdr-canvas update · i dismiss"
}

func (m *model) pickKey(msg tea.KeyPressMsg) tea.Cmd {
	if m.pickConfirm != "" {
		return m.pickConfirmKey(msg)
	}
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
				m.openCanvas(d)
			} else {
				m.status = err.Error()
			}
		}
	case "n":
		m.phase = phaseName
		m.nameInput = ""
	case "delete", "backspace":
		m.beginPickDelete()
	case "esc":
		// Only a diagram that exists can be gone back to. The picker is the
		// first screen when the canvas opens outside a git repository.
		if m.d != nil && m.d.Name != "" {
			if _, err := m.s.Load(m.d.Name); err != nil {
				m.d = &canvas.Diagram{}
				m.mtime = time.Time{}
				return nil
			}
			m.phase = phaseEdit
			m.status = ""
		}
	case "q", "ctrl+c":
		return tea.Quit
	}
	return nil
}

func (m *model) pickConfirmKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y", "enter":
		m.pickDelete(m.pickConfirm)
		m.pickConfirm = ""
	case "n", "N", "esc":
		m.pickConfirm = ""
		m.status = ""
	}
	return nil
}

func (m *model) beginPickDelete() {
	if m.sel < 0 || m.sel >= len(m.names) {
		return
	}
	m.pickConfirm = m.names[m.sel]
	m.status = fmt.Sprintf("delete %q? y/N", m.pickConfirm)
}

func (m *model) pickDelete(name string) {
	if err := m.s.Delete(name); err != nil {
		m.status = err.Error()
		return
	}
	if m.histories != nil {
		delete(m.histories, name)
	}
	if m.selections != nil {
		delete(m.selections, name)
	}
	if m.d != nil && m.d.Name == name {
		m.d = &canvas.Diagram{}
		m.mtime = time.Time{}
		m.hist = history{}
		m.clearSelection()
	}
	names, err := m.s.List()
	if err != nil {
		m.status = err.Error()
		return
	}
	m.names = names
	if m.sel >= len(m.names) && m.sel > 0 {
		m.sel--
	}
	if len(m.names) == 0 {
		m.sel = 0
	}
	m.status = fmt.Sprintf("deleted %s", name)
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
	m.pickConfirm = ""
	m.phase = phasePick
}

func (m *model) stashCanvasState() {
	if m.d == nil || m.d.Name == "" {
		return
	}
	if m.histories == nil {
		m.histories = map[string]history{}
	}
	if m.selections == nil {
		m.selections = map[string]map[string]bool{}
	}
	m.histories[m.d.Name] = cloneHistory(m.hist)
	m.selections[m.d.Name] = cloneSel(m.selected)
}

func cloneHistory(h history) history {
	cloneStack := func(in []snap) []snap {
		out := make([]snap, len(in))
		for i, s := range in {
			out[i] = snap{d: cloneDiagram(s.d), sel: cloneSel(s.sel)}
		}
		return out
	}
	return history{undoStack: cloneStack(h.undoStack), redoStack: cloneStack(h.redoStack)}
}

func (m *model) openCanvas(d *canvas.Diagram) {
	m.stashCanvasState()
	m.d = d
	m.mtime, _ = m.s.ModTime(d.Name)
	if m.histories != nil {
		m.hist = cloneHistory(m.histories[d.Name])
	} else {
		m.hist = history{}
	}
	m.tool = toolSelect
	m.selected = cloneSel(m.selections[d.Name])
	m.pruneSelection()
	m.mouse = false
	m.anchored = false
	m.anchor = [2]int{}
	m.pending = nil
	m.selAct = selectNone
	m.selAdd = false
	m.panning = false
	m.typing = false
	m.textBuf = ""
	m.editID = ""
	m.phase = phaseEdit
	m.status = ""
	m.maybeWelcome()
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
		m.openCanvas(d)
	case "esc":
		m.phase = phasePick
	case "backspace":
		if len(m.nameInput) > 0 {
			m.nameInput = m.nameInput[:len(m.nameInput)-1]
		}
	case "ctrl+v":
		return readClipboardCmd()
	default:
		if msg.Text != "" {
			m.nameInput += msg.Text
		}
	}
	return nil
}

func (m *model) typeKey(msg tea.KeyPressMsg) tea.Cmd {
	if textNewlineKey(msg) {
		m.textBuf += "\n"
		m.ensureTextCaretVisible()
		return nil
	}
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
	case "ctrl+v":
		return readClipboardCmd()
	default:
		if msg.Text != "" {
			m.textBuf += msg.Text
		}
	}
	if m.typing {
		m.ensureTextCaretVisible()
	}
	return nil
}

func textNewlineKey(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "shift+enter", "ctrl+enter", "alt+enter", "ctrl+j":
		return true
	}
	if !msg.Mod.Contains(tea.ModShift) {
		return false
	}
	return msg.Code == tea.KeyEnter || msg.Code == tea.KeyKpEnter
}

// acceptPaste inserts clipboard/bracketed-paste text into the active text
// field. Newlines become spaces so a paste does not jump the caret. Paste is
// ignored when not naming or typing.
func (m *model) acceptPaste(content string) {
	text := flattenPaste(content)
	if text == "" {
		return
	}
	switch {
	case m.phase == phaseName:
		m.nameInput += text
	case m.phase == phaseEdit && m.typing:
		m.textBuf += text
		m.ensureTextCaretVisible()
	}
}

// flattenPaste turns pasted text into one line for the text tool / name field.
// Newlines and tabs become spaces, control characters are dropped, and runs of
// spaces collapse so a multi-line paste does not leave gaps.
func flattenPaste(s string) string {
	var b strings.Builder
	space := false
	for _, r := range s {
		switch {
		case r == '\n' || r == '\r' || r == '\t' || r == ' ':
			space = true
		case r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f):
			continue
		default:
			if space {
				b.WriteByte(' ')
				space = false
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (m *model) editKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case bkey.Matches(msg, keyUndo):
		m.doUndo()
		return nil
	case bkey.Matches(msg, keyRedo):
		m.doRedo()
		return nil
	case bkey.Matches(msg, keySelectAll):
		m.selectAll()
		return nil
	}
	switch msg.String() {
	case "1", "2", "3", "4", "5", "6":
		m.switchTool(toolFor(msg.String()))
	case "c":
		m.openPalette()
	case "f":
		m.toggleFill()
	case "delete", "backspace":
		if m.tool == toolSelect {
			m.deleteSelected()
		}
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
	case "o":
		m.openPicker()
	case "?", "h":
		m.openHelp()
	case "t":
		m.openWelcome()
	case "esc":
		m.editEsc()
	case "q", "ctrl+c":
		m.save()
		return tea.Quit
	}
	return nil
}

func (m *model) editEsc() {
	if m.cancelInFlight() {
		return
	}
	if m.tool != toolSelect {
		m.tool = toolSelect
		return
	}
	m.clearSelection()
}

func (m *model) openPalette() {
	m.mouse = false
	m.anchored = false
	m.anchor = [2]int{}
	m.pending = nil
	m.selAct = selectNone
	m.panning = false
	m.phase = phasePalette
}

func (m *model) paletteKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "c":
		m.phase = phaseEdit
	case "0":
		m.pickBrushColor("")
	case "1", "2", "3", "4", "5", "6", "7", "8":
		m.pickBrushColor(colorDigit(msg.String()))
	case "q", "ctrl+c":
		m.save()
		return tea.Quit
	}
	return nil
}

func (m *model) pickBrushColor(name string) {
	m.brushColor = name
	m.phase = phaseEdit
	if len(m.selected) == 0 {
		return
	}
	cmds := make([]canvas.Command, 0, len(m.selected))
	for id := range m.selected {
		cmds = append(cmds, canvas.ColorCmd{ID: id, Color: name})
	}
	m.applyMany(cmds)
}

func (m *model) toggleFill() {
	var boxIDs []string
	for id := range m.selected {
		for i := range m.d.Elements {
			e := &m.d.Elements[i]
			if e.ID == id && e.Type == canvas.Box {
				boxIDs = append(boxIDs, id)
				break
			}
		}
	}
	if len(boxIDs) == 0 {
		m.brushFill = !m.brushFill
		return
	}
	fillAll := false
	for _, id := range boxIDs {
		for i := range m.d.Elements {
			e := &m.d.Elements[i]
			if e.ID == id && !e.Fill {
				fillAll = true
				break
			}
		}
		if fillAll {
			break
		}
	}
	m.brushFill = fillAll
	cmds := make([]canvas.Command, len(boxIDs))
	for i, id := range boxIDs {
		cmds[i] = canvas.FillCmd{ID: id, Fill: fillAll}
	}
	m.applyMany(cmds)
}

func (m *model) openHelp() {
	m.mouse = false
	m.anchored = false
	m.anchor = [2]int{}
	m.pending = nil
	m.selAct = selectNone
	m.panning = false
	m.phase = phaseHelp
}

func (m *model) helpKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "?", "h":
		m.phase = phaseEdit
	case "q", "ctrl+c":
		m.save()
		return tea.Quit
	}
	return nil
}

func (m *model) openWelcome() {
	m.mouse = false
	m.anchored = false
	m.anchor = [2]int{}
	m.pending = nil
	m.selAct = selectNone
	m.panning = false
	m.welcomeTab = 0
	m.welcomeDismiss = true
	m.phase = phaseWelcome
}

func (m *model) welcomeKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "tab", "right", "l":
		m.welcomeTab = (m.welcomeTab + 1) % len(wtabs)
	case "shift+tab", "left", "h":
		m.welcomeTab = (m.welcomeTab - 1 + len(wtabs)) % len(wtabs)
	case "d":
		m.welcomeDismiss = !m.welcomeDismiss
	case "esc", "q":
		m.closeWelcome()
	}
	return nil
}

func (m *model) closeWelcome() {
	if m.welcomeDismiss {
		mark := m.welcomeMark
		if mark == nil {
			mark = welcome.Mark
		}
		if err := mark(); err != nil {
			m.status = err.Error()
		}
	}
	m.phase = phaseEdit
}

func (m model) welcomeView() string {
	rows := welcomeLines(wtabs, m.welcomeTab, max(1, m.width), max(1, m.height), m.welcomeDismiss)
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.styled
	}
	return strings.Join(out, "\n")
}

// maybeWelcome opens the tour once per session on a true first run, from every
// entry path (git startup, the picker's first open, and RunNamed). The
// once-guard keeps it off later in-session canvas switches. A read error
// (missing or corrupt flag) counts as unseen, so the next close writes a good file.
func (m *model) maybeWelcome() {
	if m.welcomeChecked {
		return
	}
	m.welcomeChecked = true
	seen := m.welcomeSeen
	if seen == nil {
		seen = welcome.Seen
	}
	ok, _ := seen()
	if shouldWelcome(ok, m.d) {
		m.openWelcome()
	}
}

// shouldWelcome opens the tour only on a true first run: never seen, and the
// diagram is still empty, so it never interrupts existing work.
func shouldWelcome(seen bool, d *canvas.Diagram) bool {
	return !seen && d != nil && len(d.Elements) == 0
}

func (m model) noticeRows() int {
	if m.notice == "" {
		return 0
	}
	return 1
}

func (m model) layoutHeight() int {
	return max(1, m.height-m.noticeRows())
}

func (m *model) onCanvas(x, y int) bool {
	_, ok := m.vp.canvasPoint(x, y, m.width, m.layoutHeight())
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
		if ev.Button == tea.MouseLeft && (ev.Y == 0 || ev.Y == m.layoutHeight()-1) {
			return m.chromeClick(ev.X, ev.Y)
		}
		if m.typing && ev.Button == tea.MouseLeft && ev.Y == m.layoutHeight()-1 {
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
			dx := ev.X - m.panStart[0]
			dy := ev.Y - m.panStart[1]
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
	ch := layoutChrome(m.width, m.d.Name, m.cursor, m.brushColor, m.brushFill, m.tool, m.hist.canUndo(), m.hist.canRedo(), m.badge())
	hit, ok := ch.hit(x, y, m.width, m.layoutHeight())
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
	case chipHelp:
		m.openHelp()
	case chipName, chipCanvases:
		m.openPicker()
	case chipRecenter:
		m.vp.recenter(m.d.Elements, m.width, m.canvasHeight())
	}
	return nil
}

func (m *model) openPicker() {
	if m.typing {
		m.commitTyping()
	}
	m.cancelInFlight()
	m.toPicker()
}

func (m *model) switchTool(t tool) {
	if m.typing {
		m.commitTyping()
	}
	m.tool = t
	m.mouse = false
	m.anchored = false
	m.anchor = [2]int{}
	m.pending = nil
	m.selAct = selectNone
	m.selAdd = false
}

func (m *model) beginNonSelectAction() {
	if m.tool != toolSelect {
		m.clearSelection()
	}
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
		m.beginNonSelectAction()
		m.textPos = m.cursor
		m.textBuf = ""
		m.typing = true
	case toolSelect:
		e := m.d.ElementAt(m.cursor[0], m.cursor[1])
		if e == nil {
			m.clearSelection()
			return
		}
		m.selectOnly(e.ID)
	case toolDraw:
		m.beginNonSelectAction()
		if enter {
			if len(m.pending) > 0 {
				m.apply(canvas.DrawCmd{Cells: m.drawCells(), Color: m.brushColor})
			}
			return
		}
		m.addDrawCell(m.cursor)
	default:
		m.beginNonSelectAction()
		if !m.anchored {
			m.anchor = m.cursor
			m.anchored = true
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
		shift := ev.Mod.Contains(tea.ModShift)
		switch m.tool {
		case toolDraw:
			m.beginNonSelectAction()
			m.addDrawCell(m.cursor)
		case toolSelect:
			m.selectClick(shift)
		case toolText:
			m.beginNonSelectAction()
			m.textPos = m.cursor
			m.textBuf = ""
			m.typing = true
			m.mouse = false // the release arrives while typing and is dropped
		default:
			m.beginNonSelectAction()
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
		m.apply(canvas.BoxCmd{
			X1: x1, Y1: y1, X2: x2, Y2: y2,
			Color: m.brushColor, Fill: m.brushFill,
		})
	case toolLine, toolArrow:
		if from, to, ok := m.edgeEnds(); ok {
			m.apply(canvas.EdgeCmd{
				From: from.ID, To: to.ID,
				Arrow: m.lineArrow(),
				Color: m.brushColor,
			})
			return
		}
		start, end := m.snap(m.anchor), m.snap(m.cursor)
		m.apply(canvas.LineCmd{
			X1: start[0], Y1: start[1], X2: end[0], Y2: end[1],
			Arrow: m.lineArrow(),
			Color: m.brushColor,
		})
	case toolDraw:
		if len(m.pending) > 0 {
			m.apply(canvas.DrawCmd{Cells: m.drawCells(), Color: m.brushColor})
		}
	case toolSelect:
		m.commitSelect()
	}
}

func (m model) canvasHeight() int {
	extra := 0
	if m.phase == phasePalette {
		extra = 1
	}
	return max(1, m.height-headerRows-statusRows-m.noticeRows()-extra)
}

// canvasPoint translates a terminal mouse position into a grid coordinate.
// The canvas starts below the header row. The canvas shows the block of cells
// whose top-left cell is the viewport origin. canvasPoint returns false for a
// position outside the canvas.
func (m model) canvasPoint(x, y int) ([2]int, bool) {
	return m.vp.canvasPoint(x, y, m.width, m.layoutHeight())
}

// ensureVisible moves the viewport to keep the cursor inside the viewport.
func (m *model) ensureVisible() {
	m.vp.ensureVisible(m.cursor, m.width, m.layoutHeight())
}

func (m *model) ensureTextCaretVisible() {
	if !m.typing {
		return
	}
	_, cx, cy := canvas.PlaceText(m.textPos[0], m.textPos[1], m.textBuf)
	m.vp.ensureVisible([2]int{cx, cy}, m.width, m.layoutHeight())
}

func (m *model) apply(cmd canvas.Command) {
	m.applyMany([]canvas.Command{cmd})
}

func (m *model) applyMany(cmds []canvas.Command) bool {
	if len(cmds) == 0 {
		return false
	}
	m.reloadIfChanged()
	return m.commitCmds(cmds)
}

func (m *model) commitCmds(cmds []canvas.Command) bool {
	if len(cmds) == 0 {
		return false
	}
	before := cloneDiagram(m.d)
	beforeSel := cloneSel(m.selected)
	for _, cmd := range cmds {
		if err := m.d.Apply(cmd); err != nil {
			m.d = before
			m.status = err.Error()
			return false
		}
	}
	m.hist.push(before, beforeSel)
	m.save()
	return true
}

func (m *model) restore(d *canvas.Diagram) {
	m.d = d
	m.save()
}

func (m *model) doUndo() {
	if m.discardInFlight() {
		return
	}
	got, sel, ok := m.hist.undo(m.d, m.selected)
	if !ok {
		return
	}
	m.restore(got)
	m.selected = sel
	m.pruneSelection()
}

func (m *model) doRedo() {
	if m.discardInFlight() {
		return
	}
	got, sel, ok := m.hist.redo(m.d, m.selected)
	if !ok {
		return
	}
	m.restore(got)
	m.selected = sel
	m.pruneSelection()
}

func (m *model) discardInFlight() bool {
	if m.typing {
		m.commitTyping()
		return true
	}
	if m.cancelInFlight() {
		return true
	}
	return false
}

func (m *model) cancelInFlight() bool {
	if m.mouse || m.anchored || len(m.pending) > 0 || m.selAct != selectNone {
		m.mouse = false
		m.anchored = false
		m.anchor = [2]int{}
		m.pending = nil
		m.selAct = selectNone
		m.selAdd = false
		return true
	}
	return false
}

// reloadIfChanged replaces the in-memory diagram when the stored file changed
// since this session read it, so a CLI edit is not overwritten.
// busy reports whether the person is part-way through an interaction.
func (m model) busy() bool {
	return m.mouse || m.anchored || m.typing || len(m.pending) > 0 || m.selAct != selectNone
}

func (m model) inTextBuffer(p [2]int) bool {
	if !m.typing {
		return false
	}
	cells, endX, endY := canvas.PlaceText(m.textPos[0], m.textPos[1], m.textBuf)
	if p[0] == endX && p[1] == endY {
		return true
	}
	for _, c := range cells {
		if c.X == p[0] && c.Y == p[1] {
			return true
		}
	}
	return false
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
	m.clearSelection()
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
		m.apply(canvas.TextCmd{
			X: m.textPos[0], Y: m.textPos[1], Text: m.textBuf,
			Color: m.brushColor,
		})
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
	m.hist.push(m.d, m.selected)
	m.d = d
	m.mtime = mt
	m.pruneSelection()
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
		body = fmt.Sprintf("name: %s\n\nenter create · esc back\n%s\n%s\n", m.nameInput, m.status, m.noticeLine())
	case phaseAgent:
		body = m.agentView()
	case phaseHelp:
		body = m.helpView()
	case phaseWelcome:
		body = m.welcomeView()
	default:
		body = m.editView()
	}
	v := tea.NewView(body)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m model) helpView() string {
	w := max(1, m.width)
	h := max(1, m.height)
	raw := helpLines(w)
	var lines []string
	for _, line := range raw {
		lines = append(lines, wrapHelpLine(line, w)...)
	}
	close := "esc / ? / h close"
	if len(lines) > h {
		keep := max(1, h-1)
		lines = append(lines[:keep], clipRunes(close, w))
	}
	for i, line := range lines {
		lines[i] = clipRunes(line, w)
	}
	return strings.Join(lines, "\n")
}

func helpLines(width int) []string {
	if width < 48 {
		return []string{
			"herdr-canvas — help",
			"Tools  1 sel 2 box 3 line 4 arr 5 txt 6 draw",
			"Mouse  drag draw  click select/text",
			"  double-click edit text  mid-drag pan",
			"  wheel pan  shift=side  ctrl=same pan",
			"Keys  arrows space/enter c color f fill",
			"  del/back ctrl+a all ctrl+z/y s send o picker t tour esc",
			"Send/Picker  ↑↓/jk enter esc  n=new",
			"esc / ? / h close",
		}
	}
	return []string{
		"herdr-canvas — help",
		"",
		"Tools  1 select  2 box  3 line  4 arrow  5 text  6 draw  ? help",
		"",
		"Mouse",
		"  left-drag      box, line, arrow, draw · box→box arrow sticks",
		"  left-click     select one · shift-click toggle · text to type",
		"  double-click   edit existing text",
		"  middle-drag    pan",
		"  wheel          pan (shift: sideways; ctrl: same pan)",
		"",
		"Keyboard",
		"  arrows         move cursor",
		"  space / enter  anchor, then commit",
		"  draw           space adds a cell; enter commits",
		"  text           type · shift+enter newline · enter commit · esc cancel",
		"  delete         selected elements · both delete keys",
		"  c              color palette · f fill brush / selected boxes",
		"",
		"View   click canvases (o) to open picker · recenter fits · t tour",
		"Edit   ctrl+a all · ctrl+z/y undo/redo · s send · o picker · esc select · q quit",
		"Send   ↑↓/jk choose · enter send · esc cancel",
		"Picker ↑↓/jk · enter open · n new · delete confirm · esc back",
		"",
		"esc / ? / h close",
	}
}

func wrapHelpLine(s string, width int) []string {
	if width < 1 {
		width = 1
	}
	r := []rune(s)
	if len(r) <= width {
		return []string{s}
	}
	var out []string
	for len(r) > width {
		out = append(out, string(r[:width]))
		r = r[width:]
	}
	if len(r) > 0 {
		out = append(out, string(r))
	}
	return out
}

func clipRunes(s string, width int) string {
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	return string(r[:width])
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
	b.WriteString("\n↑/↓ choose · enter open · n new · del delete · esc back · q quit\n")
	if line := m.noticeLine(); line != "" {
		b.WriteString(line)
		b.WriteString("\n")
	}
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
	if m.tool == toolSelect && m.selAct == selectMove {
		return fmt.Sprintf("moving %d %+d%+d", len(m.selected), m.cursor[0]-m.anchor[0], m.cursor[1]-m.anchor[1])
	}
	if m.tool == toolSelect && len(m.selected) > 0 && m.selAct != selectMarquee {
		return fmt.Sprintf("%d sel", len(m.selected))
	}
	if m.mouse || m.anchored {
		return fmt.Sprintf("%dx%d", abs(m.cursor[0]-m.anchor[0])+1, abs(m.cursor[1]-m.anchor[1])+1)
	}
	return ""
}

func (m model) editView() string {
	elems := m.d.Elements
	if m.mouse || m.anchored || m.typing || len(m.pending) > 0 || len(m.selected) > 0 {
		elems = m.overlayElements()
	}
	painted := (&canvas.Diagram{Elements: elems}).Paint()
	if m.tool == toolArrow && (m.mouse || m.anchored) {
		if e := m.d.BoxAt(m.cursor[0], m.cursor[1]); e != nil {
			highlightBox(painted, e)
		}
	}
	ch := layoutChrome(m.width, m.d.Name, m.cursor, m.brushColor, m.brushFill, m.tool, m.hist.canUndo(), m.hist.canRedo(), m.badge())
	var b strings.Builder
	b.WriteString(ch.header)
	b.WriteString("\n")
	if m.phase == phasePalette {
		b.WriteString(layoutPaletteLine(m.width))
		b.WriteString("\n")
	}
	b.WriteString(m.vp.paintColored(painted, m.width, m.canvasHeight()))
	b.WriteString("\n")
	b.WriteString(styleChromeFooter(ch, m.tool, m.hist.canUndo(), m.hist.canRedo()))
	if line := m.noticeLine(); line != "" {
		b.WriteString("\n")
		b.WriteString(line)
	}
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

// movePreview replaces grabbed elements with copies at the dragged offset,
// and leaves a ghost outline where each element came from.
func (m model) movePreview(elems []canvas.Element, ids map[string]bool) []canvas.Element {
	dx, dy := m.cursor[0]-m.anchor[0], m.cursor[1]-m.anchor[1]
	out := make([]canvas.Element, 0, len(elems)+len(ids))
	var grabbed []canvas.Element
	for _, e := range elems {
		if ids[e.ID] {
			grabbed = append(grabbed, e)
			continue
		}
		out = append(out, e)
	}
	if len(grabbed) == 0 {
		return elems
	}
	if dx != 0 || dy != 0 {
		for _, e := range grabbed {
			out = append(out, canvas.Element{Type: canvas.Freeform, Cells: ghostCells(e), Color: m.brushColor})
		}
	}
	for _, e := range grabbed {
		moved, err := canvas.Translate(e, dx, dy)
		if err != nil {
			return elems
		}
		out = append(out, moved)
	}
	preview := canvas.Diagram{Elements: out}
	preview.RederiveEdges()
	return preview.Elements
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

// edgeEnds returns the two boxes that a tool-4 drag connects. The test uses
// the drag points as the person left them, not the snapped points, so an arrow
// sticks only when the drag really began and ended in a box.
func (m model) edgeEnds() (from, to *canvas.Element, ok bool) {
	if m.tool != toolArrow {
		return nil, nil, false
	}
	from = m.d.BoxAt(m.anchor[0], m.anchor[1])
	to = m.d.BoxAt(m.cursor[0], m.cursor[1])
	if from == nil || to == nil || from.ID == to.ID {
		return nil, nil, false
	}
	return from, to, true
}

// highlightBox marks the cells of one box, so a person dragging an arrow sees
// the box the arrow is about to stick to.
func highlightBox(painted map[[2]int]canvas.CellPaint, e *canvas.Element) {
	for y := e.Y1; y <= e.Y2; y++ {
		for x := e.X1; x <= e.X2; x++ {
			p := [2]int{x, y}
			if cp, ok := painted[p]; ok {
				cp.Reverse = true
				painted[p] = cp
			}
		}
	}
}

func (m model) lineArrow() canvas.Arrow {
	if m.tool == toolArrow {
		return canvas.ArrowEnd
	}
	return canvas.ArrowNone
}

func (m model) overlayElements() []canvas.Element {
	elems := make([]canvas.Element, len(m.d.Elements), len(m.d.Elements)+1)
	copy(elems, m.d.Elements)
	brush := func(e *canvas.Element) {
		e.Color = m.brushColor
		if e.Type == canvas.Box {
			e.Fill = m.brushFill
		}
	}
	if m.tool == toolDraw && len(m.pending) > 0 {
		e := canvas.Element{Type: canvas.Freeform, Cells: m.pending}
		brush(&e)
		elems = append(elems, e)
	}
	if m.tool == toolBox {
		x1, y1 := min(m.anchor[0], m.cursor[0]), min(m.anchor[1], m.cursor[1])
		x2, y2 := max(m.anchor[0], m.cursor[0]), max(m.anchor[1], m.cursor[1])
		e := canvas.Element{Type: canvas.Box, X1: x1, Y1: y1, X2: x2, Y2: y2}
		brush(&e)
		elems = append(elems, e)
	}
	if m.tool == toolLine || m.tool == toolArrow {
		e := canvas.Element{Type: canvas.Line, Arrow: m.lineArrow()}
		if from, to, ok := m.edgeEnds(); ok {
			e.From, e.To = from.ID, to.ID
			e.X1, e.Y1, e.X2, e.Y2, e.Vertical = canvas.EdgeEndpoints(*from, *to)
		} else {
			start, end := m.snap(m.anchor), m.snap(m.cursor)
			e.X1, e.Y1, e.X2, e.Y2 = start[0], start[1], end[0], end[1]
		}
		brush(&e)
		elems = append(elems, e)
	}
	if m.tool == toolSelect && m.selAct == selectMove && len(m.selected) > 0 {
		elems = m.movePreview(elems, m.selected)
	}
	if m.tool == toolSelect && m.selAct == selectMarquee {
		x1, y1 := min(m.anchor[0], m.cursor[0]), min(m.anchor[1], m.cursor[1])
		x2, y2 := max(m.anchor[0], m.cursor[0]), max(m.anchor[1], m.cursor[1])
		e := canvas.Element{Type: canvas.Box, X1: x1, Y1: y1, X2: x2, Y2: y2}
		brush(&e)
		elems = append(elems, e)
	}
	if len(m.selected) > 0 && m.selAct != selectMove {
		var marks []canvas.Cell
		for _, e := range elems {
			if m.selected[e.ID] {
				marks = append(marks, selectionMarkers(e)...)
			}
		}
		if len(marks) > 0 {
			elems = append(elems, canvas.Element{Type: canvas.Freeform, Cells: marks})
		}
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
		e := canvas.Element{Type: canvas.Text, X: m.textPos[0], Y: m.textPos[1], Text: m.textBuf}
		brush(&e)
		elems = append(elems, e)
		_, cx, cy := canvas.PlaceText(m.textPos[0], m.textPos[1], m.textBuf)
		elems = append(elems, canvas.Element{Type: canvas.Freeform, Cells: []canvas.Cell{{
			X: cx, Y: cy, Ch: cursorGlyph,
		}}})
	}
	return elems
}

func (m model) statusLine() string {
	if m.typing {
		return fmt.Sprintf("[text] @(%d,%d) · shift+enter or ctrl+j newline · enter commit · esc cancel", m.textPos[0], m.textPos[1])
	}
	if m.status != "" {
		return m.status
	}
	extra := ""
	if m.tool == toolSelect && m.selAct == selectMove {
		extra = fmt.Sprintf(" · moving %d %+d%+d", len(m.selected), m.cursor[0]-m.anchor[0], m.cursor[1]-m.anchor[1])
	} else if m.tool == toolSelect && len(m.selected) > 0 {
		extra = fmt.Sprintf(" · %d selected", len(m.selected))
	} else if m.mouse || m.anchored {
		extra = fmt.Sprintf(" · %dx%d", abs(m.cursor[0]-m.anchor[0])+1, abs(m.cursor[1]-m.anchor[1])+1)
	}
	if m.anchored {
		extra += fmt.Sprintf(" · anchor (%d,%d)", m.anchor[0], m.anchor[1])
	}
	return fmt.Sprintf("[%s] @(%d,%d)%s   1-6 tools · drag · s send · o picker · esc select · q quit",
		toolNames[m.tool], m.cursor[0], m.cursor[1], extra)
}

func toolFor(k string) tool {
	switch k {
	case "1":
		return toolSelect
	case "2":
		return toolBox
	case "3":
		return toolLine
	case "4":
		return toolArrow
	case "5":
		return toolText
	case "6":
		return toolDraw
	}
	return toolSelect
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
	if line := m.noticeLine(); line != "" {
		b.WriteString(line)
		b.WriteString("\n")
	}
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
