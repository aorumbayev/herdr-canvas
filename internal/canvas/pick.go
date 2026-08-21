package canvas

// ElementAt returns a pointer to the topmost element whose geometry covers
// (x, y), or nil.
func (d *Diagram) ElementAt(x, y int) *Element {
	for i := len(d.Elements) - 1; i >= 0; i-- {
		if d.Elements[i].covers(x, y) {
			return &d.Elements[i]
		}
	}
	return nil
}

// ElementsInRect returns every element that intersects the inclusive rectangle
// (x1,y1)-(x2,y2). Corner order does not matter.
func (d *Diagram) ElementsInRect(x1, y1, x2, y2 int) []*Element {
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	var out []*Element
	for i := range d.Elements {
		if d.Elements[i].intersectsRect(x1, y1, x2, y2) {
			out = append(out, &d.Elements[i])
		}
	}
	return out
}

// Bounds returns the inclusive axis-aligned box that covers the element.
func (e *Element) Bounds() (x1, y1, x2, y2 int, ok bool) {
	switch e.Type {
	case Box, Line:
		x1, y1, x2, y2 = e.X1, e.Y1, e.X2, e.Y2
		if x2 < x1 {
			x1, x2 = x2, x1
		}
		if y2 < y1 {
			y1, y2 = y2, y1
		}
		return x1, y1, x2, y2, true
	case Text:
		n := len([]rune(e.Text))
		if n == 0 {
			return 0, 0, 0, 0, false
		}
		return e.X, e.Y, e.X + n - 1, e.Y, true
	case Freeform:
		if len(e.Cells) == 0 {
			return 0, 0, 0, 0, false
		}
		x1, y1, x2, y2 = e.Cells[0].X, e.Cells[0].Y, e.Cells[0].X, e.Cells[0].Y
		for _, c := range e.Cells[1:] {
			if c.X < x1 {
				x1 = c.X
			}
			if c.Y < y1 {
				y1 = c.Y
			}
			if c.X > x2 {
				x2 = c.X
			}
			if c.Y > y2 {
				y2 = c.Y
			}
		}
		return x1, y1, x2, y2, true
	}
	return 0, 0, 0, 0, false
}

func (e *Element) covers(x, y int) bool {
	switch e.Type {
	case Box:
		return x >= e.X1 && x <= e.X2 && y >= e.Y1 && y <= e.Y2
	case Line:
		pts, _ := elbow(e.X1, e.Y1, e.X2, e.Y2)
		for _, p := range pts {
			if p == [2]int{x, y} {
				return true
			}
		}
	case Text:
		return y == e.Y && x >= e.X && x < e.X+len([]rune(e.Text))
	case Freeform:
		for _, c := range e.Cells {
			if c.X == x && c.Y == y {
				return true
			}
		}
	}
	return false
}

func (e *Element) intersectsRect(x1, y1, x2, y2 int) bool {
	switch e.Type {
	case Box:
		return !(e.X2 < x1 || e.X1 > x2 || e.Y2 < y1 || e.Y1 > y2)
	case Line:
		pts, _ := elbow(e.X1, e.Y1, e.X2, e.Y2)
		for _, p := range pts {
			if p[0] >= x1 && p[0] <= x2 && p[1] >= y1 && p[1] <= y2 {
				return true
			}
		}
	case Text:
		for i := range []rune(e.Text) {
			x := e.X + i
			if x >= x1 && x <= x2 && e.Y >= y1 && e.Y <= y2 {
				return true
			}
		}
	case Freeform:
		for _, c := range e.Cells {
			if c.X >= x1 && c.X <= x2 && c.Y >= y1 && c.Y <= y2 {
				return true
			}
		}
	}
	return false
}
