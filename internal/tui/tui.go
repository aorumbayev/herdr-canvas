package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"herdr-canvas/internal/canvas"
	"herdr-canvas/internal/name"
	"herdr-canvas/internal/store"
)

type phase int

const (
	phasePick phase = iota
	phaseName
	phaseEdit
)

type tool int

const (
	toolBox tool = iota
	toolLine
	toolText
	toolDraw
	toolMove
	toolDelete
)

var toolNames = map[tool]string{
	toolBox:    "box",
	toolLine:   "line",
	toolText:   "text",
	toolDraw:   "draw",
	toolMove:   "move",
	toolDelete: "delete",
}

type model struct {
	s     *store.Store
	d     *canvas.Diagram
	phase phase

	names []string
	sel   int

	nameInput string

	tool    tool
	cursor  [2]int
	mouse   bool
	anchor  [2]int
	grabID  string
	pending []canvas.Cell
	typing  bool
	textPos [2]int
	textBuf string
	status  string
}

// Run launches the TUI: the editor for the composite diagram in cwd, or the
// picker when cwd is not a git repository.
func Run(cwd string) error {
	s := store.New()
	m := model{s: s, d: &canvas.Diagram{}, tool: toolBox}

	nm, err := name.Composite(cwd)
	if err != nil {
		m.phase = phasePick
		m.names, _ = s.List()
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
	return run(model{s: s, d: d, phase: phaseEdit, tool: toolBox})
}

func run(m model) error {
	p := tea.NewProgram(m, tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.phase {
		case phasePick:
			cmd := m.pickKey(msg)
			return m, cmd
		case phaseName:
			cmd := m.nameKey(msg)
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
		m.d = &canvas.Diagram{Name: m.nameInput}
		if err := m.s.Save(m.d); err != nil {
			m.status = err.Error()
			return nil
		}
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
		if err := m.d.Apply(canvas.TextCmd{X: m.textPos[0], Y: m.textPos[1], Text: m.textBuf}); err != nil {
			m.status = err.Error()
		} else {
			m.save()
		}
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
	case "b", "l", "t", "d", "m", "x":
		m.tool = toolFor(msg.String())
		m.grabID = ""
		m.mouse = false
	case "up":
		m.cursor[1]--
	case "down":
		m.cursor[1]++
	case "left":
		m.cursor[0]--
	case "right":
		m.cursor[0]++
	case "s":
		m.save()
	case "q", "ctrl+c":
		m.save()
		return tea.Quit
	}
	return nil
}

func (m *model) mouseMsg(msg tea.MouseMsg) tea.Cmd {
	m.cursor = m.canvasPoint(msg.X, msg.Y)
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
			if e := m.d.ElementAt(m.cursor[0], m.cursor[1]); e != nil {
				if err := m.d.Apply(canvas.DeleteCmd{ID: e.ID}); err != nil {
					m.status = err.Error()
				} else {
					m.save()
				}
			}
		case toolText:
			m.textPos = m.cursor
			m.textBuf = ""
			m.typing = true
		}
	case tea.MouseActionMotion:
		if m.tool == toolDraw && m.mouse {
			m.addDrawCell(m.cursor)
		}
	case tea.MouseActionRelease:
		m.mouse = false
		switch m.tool {
		case toolBox:
			x1, y1 := min(m.anchor[0], m.cursor[0]), min(m.anchor[1], m.cursor[1])
			x2, y2 := max(m.anchor[0], m.cursor[0]), max(m.anchor[1], m.cursor[1])
			if err := m.d.Apply(canvas.BoxCmd{X1: x1, Y1: y1, X2: x2, Y2: y2}); err != nil {
				m.status = err.Error()
			} else {
				m.save()
			}
		case toolLine:
			end := m.snap(m.cursor)
			if err := m.d.Apply(canvas.LineCmd{X1: m.anchor[0], Y1: m.anchor[1], X2: end[0], Y2: end[1]}); err != nil {
				m.status = err.Error()
			} else {
				m.save()
			}
		case toolDraw:
			if err := m.d.Apply(canvas.DrawCmd{Cells: m.drawCells()}); err != nil {
				m.status = err.Error()
			} else {
				m.save()
			}
		case toolMove:
			if m.grabID != "" {
				dx, dy := m.cursor[0]-m.anchor[0], m.cursor[1]-m.anchor[1]
				if err := m.d.Apply(canvas.MoveCmd{ID: m.grabID, DX: dx, DY: dy}); err != nil {
					m.status = err.Error()
				} else {
					m.save()
				}
				m.grabID = ""
			}
		}
	}
	return nil
}

// canvasPoint translates terminal mouse coordinates into the grid coordinate
// system. The title consumes the first terminal row and Grid.String starts at
// the grid's minimum occupied coordinate.
func (m model) canvasPoint(x, y int) [2]int {
	minX, minY := 0, 0
	first := true
	for p := range m.d.Render() {
		if first || p[0] < minX {
			minX = p[0]
		}
		if first || p[1] < minY {
			minY = p[1]
		}
		first = false
	}
	return [2]int{minX + x, minY + y - 1}
}

func (m *model) save() {
	if err := m.s.Save(m.d); err != nil {
		m.status = err.Error()
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

// snap returns the nearest occupied cell within radius 2, else p.
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
		return b.String()
	case phaseName:
		return fmt.Sprintf("name: %s\n\nenter create · esc back\n", m.nameInput)
	default:
		return m.editView()
	}
}

func (m model) editView() string {
	g := m.d.Render()
	if m.mouse {
		// overlay the in-progress box/line/draw preview
		g = m.overlayPreview(g)
	}
	var b strings.Builder
	b.WriteString(m.d.Name)
	b.WriteString("\n")
	b.WriteString(g.String())
	b.WriteString("\n\n")
	b.WriteString(m.statusLine())
	return b.String()
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
	if m.tool == toolLine {
		end := m.snap(m.cursor)
		elems = append(elems, canvas.Element{Type: canvas.Line, X1: m.anchor[0], Y1: m.anchor[1], X2: end[0], Y2: end[1]})
	}
	tmp := canvas.Diagram{Elements: elems}
	return tmp.Render()
}

func (m model) statusLine() string {
	if m.typing {
		return fmt.Sprintf("[text] %s_  enter commit · esc cancel", m.textBuf)
	}
	if m.status != "" {
		return m.status
	}
	extra := ""
	if m.grabID != "" {
		extra = " · moving " + m.grabID
	}
	return fmt.Sprintf("[%s] @(%d,%d)%s   b/l/t/d/m/x tool · drag mouse · s save · q quit",
		toolNames[m.tool], m.cursor[0], m.cursor[1], extra)
}

func toolFor(k string) tool {
	switch k {
	case "b":
		return toolBox
	case "l":
		return toolLine
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
