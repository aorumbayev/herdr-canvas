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

func (e *Element) covers(x, y int) bool {
	switch e.Type {
	case Box:
		return x >= e.X1 && x <= e.X2 && y >= e.Y1 && y <= e.Y2
	case Line:
		for _, p := range bresenham(e.X1, e.Y1, e.X2, e.Y2) {
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
