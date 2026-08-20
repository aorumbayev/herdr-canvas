package tui

import (
	"strings"

	"herdr-canvas/internal/canvas"
)

type viewport struct {
	origin [2]int
	zoom   int
}

func (v *viewport) setZoom(n int) {
	if n != 1 && n != 2 {
		return
	}
	v.zoom = n
}

func (v *viewport) pan(dx, dy int) {
	v.origin[0] = max(0, v.origin[0]+dx)
	v.origin[1] = max(0, v.origin[1]+dy)
}

func (v viewport) canvasHeight(termH int) int {
	return max(1, termH-headerRows-statusRows)
}

func (v viewport) canvasPoint(x, y, termW, termH int) ([2]int, bool) {
	z := v.zoom
	if z < 1 {
		z = 1
	}
	row := y - headerRows
	h := v.canvasHeight(termH)
	if row < 0 || row >= h || x < 0 || x >= termW {
		return [2]int{}, false
	}
	return [2]int{v.origin[0] + x/z, v.origin[1] + row/z}, true
}

func (v *viewport) fit(elems []canvas.Element, termW, canvasH int) {
	v.zoom = 1
	if len(elems) == 0 {
		v.origin = [2]int{0, 0}
		return
	}
	minX, minY := elems[0].X1, elems[0].Y1
	if elems[0].Type == canvas.Text {
		minX, minY = elems[0].X, elems[0].Y
	}
	if elems[0].Type == canvas.Freeform && len(elems[0].Cells) > 0 {
		minX, minY = elems[0].Cells[0].X, elems[0].Cells[0].Y
	}
	for _, e := range elems {
		switch e.Type {
		case canvas.Box, canvas.Line:
			minX = min(minX, min(e.X1, e.X2))
			minY = min(minY, min(e.Y1, e.Y2))
		case canvas.Text:
			minX = min(minX, e.X)
			minY = min(minY, e.Y)
		case canvas.Freeform:
			for _, c := range e.Cells {
				minX = min(minX, c.X)
				minY = min(minY, c.Y)
			}
		}
	}
	v.origin = [2]int{max(0, minX), max(0, minY)}
}

func (v viewport) paint(g canvas.Grid, w, h int) string {
	z := v.zoom
	if z < 1 {
		z = 1
	}
	if z == 1 {
		return g.Window(v.origin[0], v.origin[1], w, h)
	}
	srcW := (w + z - 1) / z
	srcH := (h + z - 1) / z
	src := strings.Split(g.Window(v.origin[0], v.origin[1], srcW, srcH), "\n")
	var lines []string
	for _, row := range src {
		runes := []rune(row)
		var doubled []rune
		for _, r := range runes {
			if r == '─' {
				doubled = append(doubled, '─', '─')
			} else {
				doubled = append(doubled, r, ' ')
			}
		}
		s := string(doubled)
		if len([]rune(s)) > w {
			s = string([]rune(s)[:w])
		}
		lines = append(lines, s)
		if len(lines) >= h {
			break
		}
		lines = append(lines, s)
		if len(lines) >= h {
			break
		}
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines[:h], "\n")
}

func (v *viewport) ensureVisible(cursor [2]int, termW, termH int) {
	z := v.zoom
	if z < 1 {
		z = 1
	}
	h := v.canvasHeight(termH)
	cellsW := termW / z
	cellsH := h / z
	if cellsW < 1 {
		cellsW = 1
	}
	if cellsH < 1 {
		cellsH = 1
	}
	if cursor[0] < v.origin[0] {
		v.origin[0] = cursor[0]
	}
	if cursor[0] >= v.origin[0]+cellsW {
		v.origin[0] = cursor[0] - cellsW + 1
	}
	if cursor[1] < v.origin[1] {
		v.origin[1] = cursor[1]
	}
	if cursor[1] >= v.origin[1]+cellsH {
		v.origin[1] = cursor[1] - cellsH + 1
	}
	v.origin[0] = max(0, v.origin[0])
	v.origin[1] = max(0, v.origin[1])
}
