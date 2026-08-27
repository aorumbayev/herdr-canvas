package canvas

// EdgeEndpoints returns the attachment points of an edge that runs from box a
// to box b, and whether the edge attaches through the horizontal borders of
// the boxes. EdgeEndpoints is pure: the same two boxes always give the same
// points.
//
// The larger of the two gaps decides which pair of borders carries the edge.
// The attachment then follows three rules, in order:
//
//  1. The spans across the edge overlap: attach at the middle of the overlap,
//     which makes one straight run.
//  2. The spans do not overlap, and the gap holds a free lane: attach at the
//     middle of each box and cross in that lane.
//  3. Neither holds. The boxes are diagonally adjacent, so the only free cell
//     is where the free row and the free column cross. Attach at the two
//     facing corners and turn through it.
func EdgeEndpoints(a, b Element) (x1, y1, x2, y2 int, vertical bool) {
	rowGap := gap(a.Y1, a.Y2, b.Y1, b.Y2)
	colGap := gap(a.X1, a.X2, b.X1, b.X2)
	ay, by := facingRows(a, b)
	ax, bx := facingCols(a, b)
	if rowGap > colGap {
		if colGap == 0 {
			x := centre(max(a.X1, b.X1), min(a.X2, b.X2))
			return x, ay, x, by, true
		}
		return centre(a.X1, a.X2), ay, centre(b.X1, b.X2), by, true
	}
	if rowGap == 0 {
		y := centre(max(a.Y1, b.Y1), min(a.Y2, b.Y2))
		return ax, y, bx, y, false
	}
	if colGap >= 2 {
		return ax, centre(a.Y1, a.Y2), bx, centre(b.Y1, b.Y2), false
	}
	return ax, ay, bx, by, true
}

// facingRows returns the border row of each box that faces the other box.
func facingRows(a, b Element) (ay, by int) {
	if before(centre(a.Y1, a.Y2), centre(b.Y1, b.Y2), a.Y1, b.Y1) {
		return a.Y2, b.Y1
	}
	return a.Y1, b.Y2
}

// facingCols returns the border column of each box that faces the other box.
func facingCols(a, b Element) (ax, bx int) {
	if before(centre(a.X1, a.X2), centre(b.X1, b.X2), a.X1, b.X1) {
		return a.X2, b.X1
	}
	return a.X1, b.X2
}

// gap returns the free distance between two spans on one axis. Overlapping
// spans have a gap of zero.
func gap(a1, a2, b1, b2 int) int {
	if b1 > a2 {
		return b1 - a2
	}
	if a1 > b2 {
		return a1 - b2
	}
	return 0
}

func centre(lo, hi int) int { return (lo + hi) / 2 }

// before decides which of two boxes comes first on an axis. It compares
// centres, then near edges, so two boxes in the same place still give one
// stable answer.
func before(ca, cb, ea, eb int) bool {
	if ca != cb {
		return ca < cb
	}
	return ea <= eb
}

// RederiveEdges recomputes the endpoints of every edge in the diagram. An edge
// whose ends are missing, or no longer boxes, keeps the endpoints it has.
// Apply calls it after every command; a caller that builds elements by hand,
// such as a drag preview, calls it itself.
func (d *Diagram) RederiveEdges() {
	boxes := map[string]Element{}
	for _, e := range d.Elements {
		if e.Type == Box {
			boxes[e.ID] = e
		}
	}
	for i := range d.Elements {
		e := &d.Elements[i]
		if !e.IsEdge() {
			continue
		}
		from, ok := boxes[e.From]
		if !ok {
			continue
		}
		to, ok := boxes[e.To]
		if !ok {
			continue
		}
		e.X1, e.Y1, e.X2, e.Y2, e.Vertical = EdgeEndpoints(from, to)
	}
}

// dropEdgesTo removes every edge that references the given id.
func (d *Diagram) dropEdgesTo(id string) {
	kept := d.Elements[:0]
	for _, e := range d.Elements {
		if e.IsEdge() && (e.From == id || e.To == id) {
			continue
		}
		kept = append(kept, e)
	}
	d.Elements = kept
}
