package tui

import (
	"bytes"
	"encoding/json"

	"herdr-canvas/internal/canvas"
)

const historyCap = 50

type history struct {
	undoStack []*canvas.Diagram
	redoStack []*canvas.Diagram
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

func (h *history) push(d *canvas.Diagram) {
	c := cloneDiagram(d)
	if n := len(h.undoStack); n > 0 && sameDiagram(h.undoStack[n-1], c) {
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

func (h *history) undo(current *canvas.Diagram) (*canvas.Diagram, bool) {
	if !h.canUndo() {
		return current, false
	}
	prev := h.undoStack[len(h.undoStack)-1]
	h.undoStack = h.undoStack[:len(h.undoStack)-1]
	h.redoStack = append(h.redoStack, cloneDiagram(current))
	return cloneDiagram(prev), true
}

func (h *history) redo(current *canvas.Diagram) (*canvas.Diagram, bool) {
	if !h.canRedo() {
		return current, false
	}
	next := h.redoStack[len(h.redoStack)-1]
	h.redoStack = h.redoStack[:len(h.redoStack)-1]
	h.undoStack = append(h.undoStack, cloneDiagram(current))
	return cloneDiagram(next), true
}
