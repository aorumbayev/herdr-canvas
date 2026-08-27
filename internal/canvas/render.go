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

// The double-line set. An edge draws with these glyphs, so a reader can tell a
// line that sticks to two boxes from a line that floats. Every glyph is one
// terminal cell wide, like the single set.
const (
	dblHoriz    = '═'
	dblVert     = '║'
	dblTopLeft  = '╔'
	dblTopRight = '╗'
	dblLowLeft  = '╚'
	dblLowRight = '╝'
)

// lineSet is the run and bend glyphs of one line weight.
type lineSet struct {
	horiz, vert                          rune
	topLeft, topRight, lowLeft, lowRight rune
}

var (
	singleLine = lineSet{glyphHoriz, glyphVert, glyphTopLeft, glyphTopRight, glyphLowLeft, glyphLowRight}
	doubleLine = lineSet{dblHoriz, dblVert, dblTopLeft, dblTopRight, dblLowLeft, dblLowRight}
)

func lineSetOf(e Element) lineSet {
	if e.IsEdge() {
		return doubleLine
	}
	return singleLine
}

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
	pts := linePoints(e)
	glyphs := routeGlyphs(pts, lineSetOf(e))
	sdx, sdy, edx, edy := pathDirs(pts)
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
	drawLineLabel(g, e)
}

// drawLineLabel writes the label of a line near the middle of its longest
// straight run. A label that does not fit is skipped; the legend still carries
// it.
func drawLineLabel(g Grid, e Element) {
	for _, c := range lineLabelCells(e) {
		if r := []rune(c.Ch); len(r) > 0 {
			g[[2]int{c.X, c.Y}] = r[0]
		}
	}
}

// lineLabelCells returns the cells of a line label. A horizontal run carries
// the label inside the run, keeping one glyph at each end of the run. A
// vertical run carries the label beside the line, one column to the right of
// the middle cell.
func lineLabelCells(e Element) []Cell {
	label := []rune(e.Label)
	if e.Type != Line || len(label) == 0 {
		return nil
	}
	run, vertical := longestRun(linePoints(e))
	if len(run) < 3 {
		return nil
	}
	mid := run[len(run)/2]
	if vertical {
		return labelRow(label, mid[0]+1, mid[1])
	}
	if len(label) > len(run)-2 {
		return nil
	}
	return labelRow(label, mid[0]-len(label)/2, mid[1])
}

func labelRow(label []rune, x, y int) []Cell {
	if x < 0 {
		return nil
	}
	cells := make([]Cell, 0, len(label))
	for i, r := range label {
		cells = append(cells, Cell{X: x + i, Y: y, Ch: string(r)})
	}
	return cells
}

// longestRun returns the longest straight leg of a path, and whether that leg
// is vertical. A tie goes to the horizontal leg, because a label reads along
// it.
func longestRun(pts [][2]int) ([][2]int, bool) {
	if len(pts) < 2 {
		return pts, false
	}
	var best [][2]int
	bestVert := false
	start := 0
	for i := 1; i <= len(pts); i++ {
		end := i == len(pts)
		vertical := pts[start][0] == pts[start+1][0]
		if !end && sameAxis(pts[start], pts[i], vertical) {
			continue
		}
		run := pts[start:i]
		if len(run) > len(best) || (len(run) == len(best) && !vertical) {
			best, bestVert = run, vertical
		}
		start = i - 1
	}
	return best, bestVert
}

// sameAxis reports whether p stays on the run that started at from.
func sameAxis(from, p [2]int, vertical bool) bool {
	if vertical {
		return p[0] == from[0]
	}
	return p[1] == from[1]
}

// elbowPoints returns the cells of an orthogonal path from (x1,y1) to (x2,y2).
// The path runs on the y axis first, then on the x axis.
func elbowPoints(x1, y1, x2, y2 int) [][2]int {
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
	return pts
}

// linePoints returns the cells a line occupies. An edge takes the three
// segment route; every other line keeps the documented elbow.
func linePoints(e Element) [][2]int {
	if e.IsEdge() {
		return edgePoints(e.X1, e.Y1, e.X2, e.Y2, e.Vertical)
	}
	return elbowPoints(e.X1, e.Y1, e.X2, e.Y2)
}

// edgePoints returns the cells of an edge route. The route leaves the source
// box straight out of the border it starts on, crosses the gap between the two
// boxes, and arrives the same way, so its last run always points into the
// target box and never travels along a border of either box. A route with
// nothing to cross collapses to one straight run.
func edgePoints(x1, y1, x2, y2 int, vertical bool) [][2]int {
	var pts [][2]int
	if vertical {
		mid := between(y1, y2)
		pts = appendRun(pts, x1, y1, x1, mid)
		pts = appendRun(pts, x1, mid, x2, mid)
		return appendRun(pts, x2, mid, x2, y2)
	}
	mid := between(x1, x2)
	pts = appendRun(pts, x1, y1, mid, y1)
	pts = appendRun(pts, mid, y1, mid, y2)
	return appendRun(pts, mid, y2, x2, y2)
}

// between returns a coordinate strictly between a and b. Adjacent cells have
// no such coordinate; the route then turns in the cell next to a, which is the
// diagonally adjacent case EdgeEndpoints attaches at the corners for.
func between(a, b int) int {
	if abs(b-a) < 2 {
		return a
	}
	return (a + b) / 2
}

// appendRun adds the cells from (x1,y1) to (x2,y2) along one axis, without
// repeating the cell the previous run ended on.
func appendRun(pts [][2]int, x1, y1, x2, y2 int) [][2]int {
	sx, sy := sign(x2-x1), sign(y2-y1)
	x, y := x1, y1
	for {
		p := [2]int{x, y}
		if len(pts) == 0 || pts[len(pts)-1] != p {
			pts = append(pts, p)
		}
		if x == x2 && y == y2 {
			return pts
		}
		x, y = x+sx, y+sy
	}
}

// routeGlyphs gives each cell of a path its glyph in the given weight. A cell
// with one vertical and one horizontal neighbour is a bend.
func routeGlyphs(pts [][2]int, s lineSet) []rune {
	glyphs := make([]rune, len(pts))
	for i, p := range pts {
		var up, down, left, right bool
		mark := func(q [2]int) {
			switch {
			case q[1] < p[1]:
				up = true
			case q[1] > p[1]:
				down = true
			case q[0] < p[0]:
				left = true
			case q[0] > p[0]:
				right = true
			}
		}
		if i > 0 {
			mark(pts[i-1])
		}
		if i < len(pts)-1 {
			mark(pts[i+1])
		}
		switch {
		case (up || down) && (left || right):
			glyphs[i] = bend(up, right, s)
		case up || down:
			glyphs[i] = s.vert
		default:
			glyphs[i] = s.horiz
		}
	}
	return glyphs
}

// bend returns the corner glyph of a cell whose two arms point up or down, and
// left or right.
func bend(up, right bool, s lineSet) rune {
	if up {
		if right {
			return s.lowLeft
		}
		return s.lowRight
	}
	if right {
		return s.topLeft
	}
	return s.topRight
}

// pathDirs gives the direction a path leaves its first cell and the direction
// it takes into its last cell.
func pathDirs(pts [][2]int) (sdx, sdy, edx, edy int) {
	if len(pts) < 2 {
		return 0, 0, 0, 0
	}
	last := len(pts) - 1
	return sign(pts[1][0] - pts[0][0]), sign(pts[1][1] - pts[0][1]),
		sign(pts[last][0] - pts[last-1][0]), sign(pts[last][1] - pts[last-1][1])
}

func isLineChar(r rune) bool {
	if _, ok := vertWeight(r); ok {
		return true
	}
	if _, ok := horizWeight(r); ok {
		return true
	}
	return isCorner(r) || isJunction(r)
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
	case Line:
		if e.Arrow != "" && e.Arrow != ArrowNone {
			fmt.Fprintf(&b, " arrow %s", e.Arrow)
		}
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
		if e.IsEdge() {
			return "edge"
		}
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
		if e.IsEdge() {
			return e.From + "->" + e.To
		}
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
