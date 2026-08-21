package tui

import (
	"bytes"
	"encoding/json"

	"herdr-canvas/internal/canvas"
)

const historyCap = 50

type snap struct {
	d   *canvas.Diagram
	sel map[string]bool
}

type history struct {
	undoStack []snap
	redoStack []snap
}

func cloneDiagram(d *canvas.Diagram) *canvas.Diagram {
	b, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}
	out := &canvas.Diagram{}
	if err := json.Unmarshal(b, out); err != nil {
		panic(err)
	}
	return out
}

func cloneSel(sel map[string]bool) map[string]bool {
	if len(sel) == 0 {
		return nil
	}
	out := make(map[string]bool, len(sel))
	for id, v := range sel {
		out[id] = v
	}
	return out
}

func sameDiagram(a, b *canvas.Diagram) bool {
	if a == nil || b == nil {
		return a == b
	}
	left, err1 := json.Marshal(a)
	right, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return bytes.Equal(left, right)
}

func sameSel(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for id := range a {
		if !b[id] {
			return false
		}
	}
	return true
}

func (h *history) push(d *canvas.Diagram, sel map[string]bool) {
	c := snap{d: cloneDiagram(d), sel: cloneSel(sel)}
	if n := len(h.undoStack); n > 0 && sameDiagram(h.undoStack[n-1].d, c.d) && sameSel(h.undoStack[n-1].sel, c.sel) {
		return
	}
	h.undoStack = append(h.undoStack, c)
	if len(h.undoStack) > historyCap {
		h.undoStack = h.undoStack[len(h.undoStack)-historyCap:]
	}
	h.redoStack = nil
}

func (h *history) canUndo() bool { return len(h.undoStack) > 0 }
func (h *history) canRedo() bool { return len(h.redoStack) > 0 }

func (h *history) undo(current *canvas.Diagram, sel map[string]bool) (*canvas.Diagram, map[string]bool, bool) {
	if !h.canUndo() {
		return current, sel, false
	}
	prev := h.undoStack[len(h.undoStack)-1]
	h.undoStack = h.undoStack[:len(h.undoStack)-1]
	h.redoStack = append(h.redoStack, snap{d: cloneDiagram(current), sel: cloneSel(sel)})
	return cloneDiagram(prev.d), cloneSel(prev.sel), true
}

func (h *history) redo(current *canvas.Diagram, sel map[string]bool) (*canvas.Diagram, map[string]bool, bool) {
	if !h.canRedo() {
		return current, sel, false
	}
	next := h.redoStack[len(h.redoStack)-1]
	h.redoStack = h.redoStack[:len(h.redoStack)-1]
	h.undoStack = append(h.undoStack, snap{d: cloneDiagram(current), sel: cloneSel(sel)})
	return cloneDiagram(next.d), cloneSel(next.sel), true
}
