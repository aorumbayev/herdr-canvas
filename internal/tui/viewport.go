package tui

import (
	"strings"

	"herdr-canvas/internal/canvas"
)

const (
	zoom05 = 5
	zoom1  = 10
	zoom15 = 15
	zoom2  = 20
)

type viewport struct {
	origin [2]int
	zoom   int
}

func (v viewport) tenths() int {
	switch v.zoom {
	case zoom05, zoom1, zoom15, zoom2:
		return v.zoom
	default:
		return zoom1
	}
}

func (v *viewport) setZoom(n int) {
	switch n {
	case zoom05, zoom1, zoom15, zoom2:
		v.zoom = n
	}
}

func (v *viewport) zoomIn() {
	t := v.tenths()
	steps := []int{zoom05, zoom1, zoom15, zoom2}
	for i, s := range steps {
		if s == t && i+1 < len(steps) {
			v.zoom = steps[i+1]
			return
		}
	}
}

func (v *viewport) zoomOut() {
	t := v.tenths()
	steps := []int{zoom05, zoom1, zoom15, zoom2}
	for i, s := range steps {
		if s == t && i > 0 {
			v.zoom = steps[i-1]
			return
		}
	}
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
	dx, dy := v.termToDiag(x, row)
	return [2]int{v.origin[0] + dx, v.origin[1] + dy}, true
}

func (v viewport) termToDiag(x, row int) (int, int) {
	switch v.tenths() {
	case zoom05:
		return x * 2, row * 2
	case zoom15:
		return x * 2 / 3, row * 2 / 3
	case zoom2:
		return x / 2, row / 2
	default:
		return x, row
	}
}

func (v viewport) cells(termW, canvasH int) (int, int) {
	t := v.tenths()
	cw := termW * 10 / t
	ch := canvasH * 10 / t
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
			touch(e.X, e.Y)
			if n := len([]rune(e.Text)); n > 0 {
				touch(e.X+n-1, e.Y)
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
	switch v.tenths() {
	case zoom05:
		return v.paint05(g, w, h)
	case zoom15:
		return v.paint15(g, w, h)
	case zoom2:
		return v.paint2(g, w, h)
	default:
		return g.Window(v.origin[0], v.origin[1], w, h)
	}
}

func glyphAt(g canvas.Grid, x, y int) rune {
	ch, ok := g[[2]int{x, y}]
	if !ok {
		return ' '
	}
	return ch
}

func (v viewport) paint05(g canvas.Grid, w, h int) string {
	var lines []string
	for ty := 0; ty < h; ty++ {
		var b []rune
		for tx := 0; tx < w; tx++ {
			sx, sy := v.origin[0]+tx*2, v.origin[1]+ty*2
			ch := ' '
			for dy := 0; dy < 2 && ch == ' '; dy++ {
				for dx := 0; dx < 2 && ch == ' '; dx++ {
					if r := glyphAt(g, sx+dx, sy+dy); r != ' ' {
						ch = r
					}
				}
			}
			b = append(b, ch)
		}
		lines = append(lines, string(b))
	}
	return strings.Join(lines, "\n")
}

func scale15Row(src []rune) []rune {
	var out []rune
	for i := 0; i < len(src); i += 2 {
		a := src[i]
		b := rune(' ')
		if i+1 < len(src) {
			b = src[i+1]
		}
		extra := rune(' ')
		if a == '─' && b == '─' {
			extra = '─'
		}
		out = append(out, a, extra, b)
	}
	return out
}

func (v viewport) paint15(g canvas.Grid, w, h int) string {
	srcW := (w*2 + 2) / 3
	if srcW < 1 {
		srcW = 1
	}
	srcH := (h*2 + 2) / 3
	if srcH < 1 {
		srcH = 1
	}
	src := strings.Split(g.Window(v.origin[0], v.origin[1], srcW, srcH), "\n")
	var lines []string
	for sy := 0; sy < len(src); sy += 2 {
		row := []rune(src[sy])
		line := string(scale15Row(row))
		if len([]rune(line)) > w {
			line = string([]rune(line)[:w])
		}
		lines = append(lines, line)
		if len(lines) >= h {
			break
		}
		lines = append(lines, line)
		if len(lines) >= h {
			break
		}
		if sy+1 < len(src) {
			lineB := string(scale15Row([]rune(src[sy+1])))
			if len([]rune(lineB)) > w {
				lineB = string([]rune(lineB)[:w])
			}
			lines = append(lines, lineB)
			if len(lines) >= h {
				break
			}
		}
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines[:h], "\n")
}

func (v viewport) paint2(g canvas.Grid, w, h int) string {
	srcW := (w + 1) / 2
	srcH := (h + 1) / 2
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
