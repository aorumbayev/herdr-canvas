package canvas

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Grid is the sparse (x, y) -> char render of a Diagram.
type Grid map[[2]int]rune

// The one character set for boxes and lines. asciiflow calls this set
// "extended". Every glyph here comes from the Unicode box-drawing block, so a
// box edge and a line junction join without a change of style.
const (
	glyphHoriz    = '─'
	glyphVert     = '│'
	glyphTopLeft  = '┌'
	glyphTopRight = '┐'
	glyphLowLeft  = '└'
	glyphLowRight = '┘'
	glyphCross    = '┼'
	glyphTeeRight = '├'
	glyphTeeLeft  = '┤'
	glyphTeeDown  = '┬'
	glyphTeeUp    = '┴'
)

// Render produces the grid. Render is a pure function of the elements. Later
// elements in the slice cover earlier elements.
func (d *Diagram) Render() Grid {
	g := Grid{}
	for _, e := range d.Elements {
		switch e.Type {
		case Box:
			drawBox(g, e)
		case Line:
			drawLine(g, e)
		case Text:
			cells, _, _ := PlaceText(e.X, e.Y, e.Text)
			for _, c := range cells {
				if r := []rune(c.Ch); len(r) > 0 {
					g[[2]int{c.X, c.Y}] = r[0]
				}
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
		g[[2]int{x, e.Y1}] = glyphHoriz
		g[[2]int{x, e.Y2}] = glyphHoriz
	}
	for y := e.Y1; y <= e.Y2; y++ {
		g[[2]int{e.X1, y}] = glyphVert
		g[[2]int{e.X2, y}] = glyphVert
	}
	g[[2]int{e.X1, e.Y1}] = glyphTopLeft
	g[[2]int{e.X2, e.Y1}] = glyphTopRight
	g[[2]int{e.X1, e.Y2}] = glyphLowLeft
	g[[2]int{e.X2, e.Y2}] = glyphLowRight
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
	pts, glyphs := elbow(e.X1, e.Y1, e.X2, e.Y2)
	sdx, sdy := startDir(e.X1, e.Y1, e.X2, e.Y2)
	edx, edy := endDir(e.X1, e.Y1, e.X2, e.Y2)
	last := len(pts) - 1
	for i, p := range pts {
		ch := glyphs[i]
		switch {
		case (e.Arrow == ArrowStart || e.Arrow == ArrowBoth) && i == 0:
			ch = arrowGlyph(-sdx, -sdy)
		case (e.Arrow == ArrowEnd || e.Arrow == ArrowBoth) && i == last:
			ch = arrowGlyph(edx, edy)
		default:
			if existing, ok := g[p]; ok && isLineChar(existing) {
				switch {
				case i == 0 && last > 0:
					ch = junction(existing, ch, false, sdx, sdy)
				case i == last && last > 0:
					ch = junction(existing, ch, false, -edx, -edy)
				default:
					ch = junction(existing, ch, true, 0, 0)
				}
			}
		}
		g[p] = ch
	}
}

// elbow returns the cells of an orthogonal path from (x1,y1) to (x2,y2), with
// the glyph of each cell. The path runs on the y axis first, then on the x
// axis. A path that turns has one corner cell. A straight path has none.
func elbow(x1, y1, x2, y2 int) ([][2]int, []rune) {
	sx, sy := sign(x2-x1), sign(y2-y1)
	var pts [][2]int
	for y := y1; ; y += sy {
		pts = append(pts, [2]int{x1, y})
		if sy == 0 || y == y2 {
			break
		}
	}
	for x := x1 + sx; sx != 0; x += sx {
		pts = append(pts, [2]int{x, y2})
		if x == x2 {
			break
		}
	}
	glyphs := make([]rune, len(pts))
	for i, p := range pts {
		vert := (i > 0 && pts[i-1][0] == p[0]) || (i < len(pts)-1 && pts[i+1][0] == p[0])
		horiz := (i > 0 && pts[i-1][1] == p[1]) || (i < len(pts)-1 && pts[i+1][1] == p[1])
		switch {
		case vert && horiz:
			glyphs[i] = corner(sx, sy)
		case vert:
			glyphs[i] = glyphVert
		default:
			glyphs[i] = glyphHoriz
		}
	}
	return pts, glyphs
}

// corner returns the bend glyph of an elbow that arrives on the y axis in the
// direction sy and leaves on the x axis in the direction sx.
func corner(sx, sy int) rune {
	if sy > 0 {
		if sx > 0 {
			return glyphLowLeft
		}
		return glyphLowRight
	}
	if sx > 0 {
		return glyphTopLeft
	}
	return glyphTopRight
}

// startDir gives the direction the path takes out of its first cell.
func startDir(x1, y1, x2, y2 int) (int, int) {
	if y1 != y2 {
		return 0, sign(y2 - y1)
	}
	return sign(x2 - x1), 0
}

// endDir gives the direction the path takes into its last cell.
func endDir(x1, y1, x2, y2 int) (int, int) {
	if x1 != x2 {
		return sign(x2 - x1), 0
	}
	return 0, sign(y2 - y1)
}

func isLineChar(r rune) bool {
	switch r {
	case glyphHoriz, glyphVert, glyphCross, glyphTeeRight, glyphTeeLeft, glyphTeeDown, glyphTeeUp:
		return true
	case glyphTopLeft, glyphTopRight, glyphLowLeft, glyphLowRight:
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

// arrowGlyph returns the arrowhead that points in the direction (dx, dy).
func arrowGlyph(dx, dy int) rune {
	if abs(dx) >= abs(dy) {
		if dx > 0 {
			return '►'
		}
		return '◄'
	}
	if dy > 0 {
		return '▼'
	}
	return '▲'
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Export renders a diagram to compact grid text plus a legend. The picture is
// character-only; color and fill appear only in the legend lines.
func Export(d *Diagram) string {
	pic := d.Render().String()
	if len(d.Elements) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(pic)
	sb.WriteByte('\n')
	for i, e := range d.Elements {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(legendLine(e))
	}
	return sb.String()
}

func legendLine(e Element) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s %s", e.ID, legendKind(e), legendGeom(e))
	if e.Color != "" {
		b.WriteByte(' ')
		b.WriteString(e.Color)
	}
	if e.Fill {
		b.WriteString(" fill")
	}
	switch e.Type {
	case Box:
		if e.Label != "" {
			fmt.Fprintf(&b, " %q", e.Label)
		}
	case Text:
		fmt.Fprintf(&b, " %q", e.Text)
	}
	return b.String()
}

func legendKind(e Element) string {
	switch e.Type {
	case Box:
		return "box"
	case Line:
		return "line"
	case Text:
		return "text"
	case Freeform:
		return "draw"
	default:
		return string(e.Type)
	}
}

func legendGeom(e Element) string {
	switch e.Type {
	case Box, Line:
		return fmt.Sprintf("%d,%d-%d,%d", e.X1, e.Y1, e.X2, e.Y2)
	case Text:
		return fmt.Sprintf("%d,%d", e.X, e.Y)
	case Freeform:
		return strconv.Itoa(len(e.Cells))
	default:
		return ""
	}
}

// Window renders the w by h block of cells whose top-left cell is (x0, y0).
// Window always returns h lines, so a caller can keep a fixed viewport.
func (g Grid) Window(x0, y0, w, h int) string {
	var sb strings.Builder
	for row := 0; row < h; row++ {
		if row > 0 {
			sb.WriteByte('\n')
		}
		line := make([]rune, 0, w)
		for col := 0; col < w; col++ {
			if r, ok := g[[2]int{x0 + col, y0 + row}]; ok {
				line = append(line, r)
			} else {
				line = append(line, ' ')
			}
		}
		sb.WriteString(strings.TrimRight(string(line), " "))
	}
	return sb.String()
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
