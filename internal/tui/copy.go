package tui

import "herdr-canvas/internal/canvas"

// copyBuffer holds the elements taken by ctrl+c and the top-left corner of the
// block they came from, so a paste can rebase the block onto the cursor. The
// buffer lives for the session and is shared by every diagram.
type copyBuffer struct {
	elems []canvas.Element
	x, y  int
}

func (m *model) copySelection() {
	m.reloadIfChanged()
	m.pruneSelection()
	if len(m.selected) == 0 {
		return
	}
	var buf copyBuffer
	first := true
	for _, e := range m.d.Elements {
		if !m.selected[e.ID] {
			continue
		}
		buf.elems = append(buf.elems, cloneElement(e))
		if e.IsEdge() {
			// An edge takes its endpoints from its boxes, so it cannot set
			// the origin of the block.
			continue
		}
		x1, y1, _, _, ok := e.Bounds()
		if !ok {
			continue
		}
		if first || x1 < buf.x {
			buf.x = x1
		}
		if first || y1 < buf.y {
			buf.y = y1
		}
		first = false
	}
	if len(buf.elems) == 0 {
		return
	}
	m.clip = buf
}

func (m *model) pasteBuffer() {
	if len(m.clip.elems) == 0 {
		return
	}
	m.reloadIfChanged()
	dx, dy := m.cursor[0]-m.clip.x, m.cursor[1]-m.clip.y

	var (
		cmds   []canvas.Command
		source []string
	)
	for _, e := range m.clip.elems {
		if e.IsEdge() {
			continue
		}
		cmds = append(cmds, shiftCmd(e, dx, dy))
		source = append(source, e.ID)
	}
	if len(cmds) == 0 {
		return
	}
	base := len(m.d.Elements)

	edges := func() []canvas.Command {
		fresh := make(map[string]string, len(source))
		for i, id := range source {
			fresh[id] = m.d.Elements[base+i].ID
		}
		var out []canvas.Command
		for _, e := range m.clip.elems {
			if !e.IsEdge() {
				continue
			}
			from, okFrom := fresh[e.From]
			to, okTo := fresh[e.To]
			if !okFrom || !okTo {
				continue
			}
			out = append(out, canvas.EdgeCmd{From: from, To: to, Label: e.Label, Arrow: e.Arrow, Color: e.Color})
		}
		return out
	}

	if !m.commitStaged(cmds, edges) {
		return
	}
	m.selected = make(map[string]bool, len(m.d.Elements)-base)
	for _, e := range m.d.Elements[base:] {
		m.selected[e.ID] = true
	}
}

func shiftCmd(e canvas.Element, dx, dy int) canvas.Command {
	switch e.Type {
	case canvas.Box:
		return canvas.BoxCmd{
			X1: e.X1 + dx, Y1: e.Y1 + dy, X2: e.X2 + dx, Y2: e.Y2 + dy,
			Label: e.Label, Color: e.Color, Fill: e.Fill,
		}
	case canvas.Line:
		return canvas.LineCmd{
			X1: e.X1 + dx, Y1: e.Y1 + dy, X2: e.X2 + dx, Y2: e.Y2 + dy,
			Arrow: e.Arrow, Color: e.Color,
		}
	case canvas.Text:
		return canvas.TextCmd{X: e.X + dx, Y: e.Y + dy, Text: e.Text, Color: e.Color}
	case canvas.Freeform:
		cells := make([]canvas.Cell, len(e.Cells))
		for i, c := range e.Cells {
			cells[i] = canvas.Cell{X: c.X + dx, Y: c.Y + dy, Ch: c.Ch}
		}
		return canvas.DrawCmd{Cells: cells, Color: e.Color}
	}
	return nil
}

func cloneElement(e canvas.Element) canvas.Element {
	if e.Cells != nil {
		e.Cells = append([]canvas.Cell(nil), e.Cells...)
	}
	return e
}
