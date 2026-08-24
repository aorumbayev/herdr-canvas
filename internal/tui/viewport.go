package tui

import "herdr-canvas/internal/canvas"

type viewport struct {
	origin [2]int
}

func (v *viewport) pan(dx, dy int) {
	v.origin[0] = max(0, v.origin[0]+dx)
	v.origin[1] = max(0, v.origin[1]+dy)
}

func (v viewport) canvasHeight(termH int) int {
	return max(1, termH-headerRows-statusRows)
}

func (v viewport) canvasPoint(x, y, termW, termH int) ([2]int, bool) {
	row := y - headerRows
	h := v.canvasHeight(termH)
	if row < 0 || row >= h || x < 0 || x >= termW {
		return [2]int{}, false
	}
	return [2]int{v.origin[0] + x, v.origin[1] + row}, true
}

func (v viewport) cells(termW, canvasH int) (int, int) {
	cw, ch := termW, canvasH
	if cw < 1 {
		cw = 1
	}
	if ch < 1 {
		ch = 1
	}
	return cw, ch
}

func elementBBox(elems []canvas.Element) (minX, minY, maxX, maxY int, ok bool) {
	if len(elems) == 0 {
		return 0, 0, 0, 0, false
	}
	first := true
	touch := func(x, y int) {
		if first {
			minX, minY, maxX, maxY = x, y, x, y
			first = false
			return
		}
		minX, minY = min(minX, x), min(minY, y)
		maxX, maxY = max(maxX, x), max(maxY, y)
	}
	for _, e := range elems {
		switch e.Type {
		case canvas.Box, canvas.Line:
			touch(e.X1, e.Y1)
			touch(e.X2, e.Y2)
		case canvas.Text:
			if x1, y1, x2, y2, ok := e.Bounds(); ok {
				touch(x1, y1)
				touch(x2, y2)
			}
		case canvas.Freeform:
			for _, c := range e.Cells {
				touch(c.X, c.Y)
			}
		}
	}
	return minX, minY, maxX, maxY, !first
}

func (v *viewport) recenter(elems []canvas.Element, termW, canvasH int) {
	minX, minY, maxX, maxY, ok := elementBBox(elems)
	if !ok {
		v.origin = [2]int{0, 0}
		return
	}
	cw, ch := v.cells(termW, canvasH)
	bw, bh := maxX-minX+1, maxY-minY+1
	if bw > cw || bh > ch {
		v.origin = [2]int{max(0, minX), max(0, minY)}
		return
	}
	ox, oy := minX-(cw-bw)/2, minY-(ch-bh)/2
	v.origin = [2]int{max(0, ox), max(0, oy)}
}

func (v viewport) paint(g canvas.Grid, w, h int) string {
	return g.Window(v.origin[0], v.origin[1], w, h)
}

func (v *viewport) ensureVisible(cursor [2]int, termW, termH int) {
	h := v.canvasHeight(termH)
	cellsW, cellsH := v.cells(termW, h)
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
