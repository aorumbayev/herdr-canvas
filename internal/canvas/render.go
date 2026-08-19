package canvas

import (
	"math"
	"strings"
)

// Grid is the sparse (x, y) -> char render of a Diagram.
type Grid map[[2]int]rune

// Render produces the grid as a pure function of the elements in z-order
// (slice order, later wins).
func (d *Diagram) Render() Grid {
	g := Grid{}
	for _, e := range d.Elements {
		switch e.Type {
		case Box:
			drawBox(g, e)
		case Line:
			drawLine(g, e)
		case Text:
			for i, r := range []rune(e.Text) {
				g[[2]int{e.X + i, e.Y}] = r
			}
		case Freeform:
			for _, c := range e.Cells {
				if r := []rune(c.Ch); len(r) > 0 {
					g[[2]int{c.X, c.Y}] = r[0]
				}
			}
		}
	}
	return g
}

func drawBox(g Grid, e Element) {
	for x := e.X1; x <= e.X2; x++ {
		g[[2]int{x, e.Y1}] = '-'
		g[[2]int{x, e.Y2}] = '-'
	}
	for y := e.Y1; y <= e.Y2; y++ {
		g[[2]int{e.X1, y}] = '|'
		g[[2]int{e.X2, y}] = '|'
	}
	g[[2]int{e.X1, e.Y1}] = '+'
	g[[2]int{e.X2, e.Y1}] = '+'
	g[[2]int{e.X1, e.Y2}] = '+'
	g[[2]int{e.X2, e.Y2}] = '+'
	drawLabel(g, e)
}

// drawLabel writes the box label on the first inner row, clipped to the inner
// width. A box with no inner cell carries no label.
func drawLabel(g Grid, e Element) {
	if e.Label == "" {
		return
	}
	width := e.X2 - e.X1 - 1
	if width <= 0 || e.Y2-e.Y1 < 2 {
		return
	}
	for i, r := range []rune(e.Label) {
		if i >= width {
			break
		}
		g[[2]int{e.X1 + 1 + i, e.Y1 + 1}] = r
	}
}

func drawLine(g Grid, e Element) {
	pts := bresenham(e.X1, e.Y1, e.X2, e.Y2)
	body := lineBody(e.X1, e.Y1, e.X2, e.Y2)
	dx, dy := sign(e.X2-e.X1), sign(e.Y2-e.Y1)
	for i, p := range pts {
		ch := body
		switch {
		case (e.Arrow == ArrowStart || e.Arrow == ArrowBoth) && i == 0:
			ch = arrowAt(e.X1, e.Y1, e.X2, e.Y2, true)
		case (e.Arrow == ArrowEnd || e.Arrow == ArrowBoth) && i == len(pts)-1:
			ch = arrowAt(e.X1, e.Y1, e.X2, e.Y2, false)
		default:
			if existing, ok := g[p]; ok && isLineChar(existing) {
				switch {
				case i == 0:
					ch = junction(existing, body, false, dx, dy)
				case i == len(pts)-1:
					ch = junction(existing, body, false, -dx, -dy)
				default:
					ch = junction(existing, body, true, 0, 0)
				}
			}
		}
		g[p] = ch
	}
}

func isLineChar(r rune) bool {
	switch r {
	case '-', '|', '+', '\\', '/', '┼', '├', '┤', '┬', '┴':
		return true
	}
	return false
}

func sign(n int) int {
	if n > 0 {
		return 1
	}
	if n < 0 {
		return -1
	}
	return 0
}

func lineBody(x1, y1, x2, y2 int) rune {
	switch {
	case y1 == y2:
		return '-'
	case x1 == x2:
		return '|'
	case (x2-x1)*(y2-y1) > 0:
		return '\\'
	default:
		return '/'
	}
}

func arrowAt(x1, y1, x2, y2 int, start bool) rune {
	dx, dy := x2-x1, y2-y1
	if start {
		dx, dy = -dx, -dy
	}
	if abs(dx) >= abs(dy) {
		if dx > 0 {
			return '>'
		}
		return '<'
	}
	if dy > 0 {
		return 'v'
	}
	return '^'
}

func bresenham(x1, y1, x2, y2 int) [][2]int {
	dx, dy := abs(x2-x1), -abs(y2-y1)
	sx, sy := 1, 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx + dy
	var pts [][2]int
	x, y := x1, y1
	for {
		pts = append(pts, [2]int{x, y})
		if x == x2 && y == y2 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x += sx
		}
		if e2 <= dx {
			err += dx
			y += sy
		}
	}
	return pts
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Export renders a diagram to compact grid text an LLM (or human) can read:
// the bounding rectangle of the grid with trailing whitespace trimmed.
func Export(d *Diagram) string {
	return d.Render().String()
}

// String renders the grid as a compact, space-filled rectangle.
func (g Grid) String() string {
	if len(g) == 0 {
		return ""
	}
	minX, minY := math.MaxInt, math.MaxInt
	maxX, maxY := math.MinInt, math.MinInt
	for p := range g {
		if p[0] < minX {
			minX = p[0]
		}
		if p[0] > maxX {
			maxX = p[0]
		}
		if p[1] < minY {
			minY = p[1]
		}
		if p[1] > maxY {
			maxY = p[1]
		}
	}
	var sb strings.Builder
	for y := minY; y <= maxY; y++ {
		line := make([]rune, 0, maxX-minX+1)
		for x := minX; x <= maxX; x++ {
			if r, ok := g[[2]int{x, y}]; ok {
				line = append(line, r)
			} else {
				line = append(line, ' ')
			}
		}
		sb.WriteString(strings.TrimRight(string(line), " "))
		sb.WriteByte('\n')
	}
	return strings.TrimRight(sb.String(), "\n")
}
