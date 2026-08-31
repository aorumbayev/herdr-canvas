package tui

import "herdr-canvas/internal/canvas"

type selectAct int

const (
	selectNone selectAct = iota
	selectMarquee
	selectMove
)

const selectGlyph = "*"

func (m *model) clearSelection() {
	m.selected = nil
}

func (m *model) ensureSelected() {
	if m.selected == nil {
		m.selected = make(map[string]bool)
	}
}

// pruneSelection drops ids that are no longer on the diagram (a reload from
// disk). Undo/redo restore selection from history; prune is a safety net.
func (m *model) pruneSelection() {
	if len(m.selected) == 0 {
		return
	}
	live := make(map[string]bool, len(m.selected))
	for _, e := range m.d.Elements {
		if m.selected[e.ID] {
			live[e.ID] = true
		}
	}
	if len(live) == 0 {
		m.selected = nil
		return
	}
	m.selected = live
}

func (m *model) isSelected(id string) bool {
	return m.selected[id]
}

func (m *model) toggleAt(p [2]int) {
	e := m.d.ElementAt(p[0], p[1])
	if e == nil {
		return
	}
	m.ensureSelected()
	if m.selected[e.ID] {
		delete(m.selected, e.ID)
	} else {
		m.selected[e.ID] = true
	}
}

func (m *model) selectOnly(id string) {
	m.selected = map[string]bool{id: true}
}

func (m *model) selectClick(shift bool) {
	e := m.d.ElementAt(m.cursor[0], m.cursor[1])
	m.selAdd = shift
	if e == nil {
		if !shift {
			m.clearSelection()
		}
		m.selAct = selectMarquee
		return
	}
	if shift {
		m.toggleAt(m.cursor)
		m.selAct = selectNone
		m.mouse = false
		return
	}
	if m.isSelected(e.ID) {
		m.selAct = selectMove
		return
	}
	m.selectOnly(e.ID)
	m.selAct = selectNone
	m.mouse = false
}

func (m *model) commitSelect() {
	switch m.selAct {
	case selectMarquee:
		if !m.selAdd {
			m.clearSelection()
		}
		m.ensureSelected()
		for _, e := range m.d.ElementsInRect(m.anchor[0], m.anchor[1], m.cursor[0], m.cursor[1]) {
			m.selected[e.ID] = true
		}
		if len(m.selected) == 0 {
			m.selected = nil
		}
	case selectMove:
		dx, dy := m.cursor[0]-m.anchor[0], m.cursor[1]-m.anchor[1]
		if dx == 0 && dy == 0 {
			if e := m.d.ElementAt(m.anchor[0], m.anchor[1]); e != nil {
				m.selectOnly(e.ID)
			}
			break
		}
		m.reloadIfChanged()
		m.pruneSelection()
		cmds := make([]canvas.Command, 0, len(m.selected))
		for id := range m.selected {
			cmds = append(cmds, canvas.MoveCmd{ID: id, DX: dx, DY: dy})
		}
		m.commitCmds(cmds)
	}
	m.selAct = selectNone
	m.selAdd = false
}

func (m *model) deleteSelected() {
	m.reloadIfChanged()
	m.pruneSelection()
	if len(m.selected) == 0 {
		return
	}
	cmds := make([]canvas.Command, 0, len(m.selected))
	for _, e := range m.d.Elements {
		if !m.selected[e.ID] {
			continue
		}
		// Deleting a box already drops its edges, so asking for the edge
		// again would fail on an id that is gone.
		if e.IsEdge() && (m.selected[e.From] || m.selected[e.To]) {
			continue
		}
		cmds = append(cmds, canvas.DeleteCmd{ID: e.ID})
	}
	if m.commitCmds(cmds) {
		m.clearSelection()
	}
	m.selAct = selectNone
}

func selectionMarkers(e canvas.Element) []canvas.Cell {
	x1, y1, x2, y2, ok := e.Bounds()
	if !ok {
		return nil
	}
	return []canvas.Cell{
		{X: x1, Y: y1, Ch: selectGlyph},
		{X: x2, Y: y1, Ch: selectGlyph},
		{X: x1, Y: y2, Ch: selectGlyph},
		{X: x2, Y: y2, Ch: selectGlyph},
	}
}

// selectAll selects every element of the current diagram. A drawing tool
// clears the selection as soon as it acts, so switch to select first.
func (m *model) selectAll() {
	if m.tool != toolSelect {
		m.switchTool(toolSelect)
	}
	if len(m.d.Elements) == 0 {
		return
	}
	m.selected = make(map[string]bool, len(m.d.Elements))
	for _, e := range m.d.Elements {
		m.selected[e.ID] = true
	}
}
