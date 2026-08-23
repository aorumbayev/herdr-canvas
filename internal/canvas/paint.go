package canvas

// CellPaint is one rendered cell with optional foreground and background colors.
type CellPaint struct {
	Ch rune
	FG string
	BG string
}

// Paint produces a colored render of the diagram. Later elements cover earlier
// ones completely (character and color).
func (d *Diagram) Paint() map[[2]int]CellPaint {
	out := map[[2]int]CellPaint{}
	for _, e := range d.Elements {
		switch e.Type {
		case Box:
			paintBox(out, e)
		case Line:
			paintLine(out, e)
		case Text:
			for i, r := range []rune(e.Text) {
				setPaint(out, [2]int{e.X + i, e.Y}, CellPaint{Ch: r, FG: e.Color})
			}
		case Freeform:
			for _, c := range e.Cells {
				if r := []rune(c.Ch); len(r) > 0 {
					setPaint(out, [2]int{c.X, c.Y}, CellPaint{Ch: r[0], FG: e.Color})
				}
			}
		}
	}
	return out
}

func setPaint(m map[[2]int]CellPaint, p [2]int, cp CellPaint) {
	m[p] = cp
}

func paintBox(m map[[2]int]CellPaint, e Element) {
	for x := e.X1; x <= e.X2; x++ {
		setPaint(m, [2]int{x, e.Y1}, CellPaint{Ch: glyphHoriz, FG: e.Color})
		setPaint(m, [2]int{x, e.Y2}, CellPaint{Ch: glyphHoriz, FG: e.Color})
	}
	for y := e.Y1; y <= e.Y2; y++ {
		setPaint(m, [2]int{e.X1, y}, CellPaint{Ch: glyphVert, FG: e.Color})
		setPaint(m, [2]int{e.X2, y}, CellPaint{Ch: glyphVert, FG: e.Color})
	}
	setPaint(m, [2]int{e.X1, e.Y1}, CellPaint{Ch: glyphTopLeft, FG: e.Color})
	setPaint(m, [2]int{e.X2, e.Y1}, CellPaint{Ch: glyphTopRight, FG: e.Color})
	setPaint(m, [2]int{e.X1, e.Y2}, CellPaint{Ch: glyphLowLeft, FG: e.Color})
	setPaint(m, [2]int{e.X2, e.Y2}, CellPaint{Ch: glyphLowRight, FG: e.Color})
	if e.Fill && e.X2-e.X1 >= 2 && e.Y2-e.Y1 >= 2 {
		bg := e.Color
		if bg == "" {
			bg = "8"
		}
		for y := e.Y1 + 1; y <= e.Y2-1; y++ {
			for x := e.X1 + 1; x <= e.X2-1; x++ {
				setPaint(m, [2]int{x, y}, CellPaint{Ch: ' ', FG: "", BG: bg})
			}
		}
	}
	paintBoxLabel(m, e)
}

func paintBoxLabel(m map[[2]int]CellPaint, e Element) {
	if e.Label == "" {
		return
	}
	width := e.X2 - e.X1 - 1
	if width <= 0 || e.Y2-e.Y1 < 2 {
		return
	}
	fg := labelContrast(e)
	for i, r := range []rune(e.Label) {
		if i >= width {
			break
		}
		p := [2]int{e.X1 + 1 + i, e.Y1 + 1}
		cp := m[p]
		cp.Ch = r
		cp.FG = fg
		setPaint(m, p, cp)
	}
}

func labelContrast(e Element) string {
	if e.Fill {
		bg := e.Color
		if bg == "" {
			return "white"
		}
		switch bg {
		case "yellow", "white", "cyan", "green":
			return "black"
		default:
			return "white"
		}
	}
	return e.Color
}

func paintLine(m map[[2]int]CellPaint, e Element) {
	g := Grid{}
	for p, cp := range m {
		g[p] = cp.Ch
	}
	drawLine(g, e)
	pts, _ := elbow(e.X1, e.Y1, e.X2, e.Y2)
	for _, p := range pts {
		setPaint(m, p, CellPaint{Ch: g[p], FG: e.Color})
	}
}
