package canvas

// junction returns the glyph for a cell where two line segments meet. The
// parameter existing is the glyph already in the cell. The parameter inc is
// the glyph of the incoming segment. If cross is true, the incoming segment
// continues through the cell. If cross is false, the incoming segment stops in
// the cell and makes a tee. The parameters sdcol and sdrow give the direction
// of the tee stub. Each of these two parameters is -1, 0 or 1.
func junction(existing, inc rune, cross bool, sdcol, sdrow int) rune {
	if existing == glyphVert && isHorizIncoming(inc) {
		if cross {
			return glyphCross
		}
		switch {
		case sdcol > 0:
			return glyphTeeRight
		case sdcol < 0:
			return glyphTeeLeft
		default:
			return glyphCross
		}
	}
	if existing == glyphHoriz && isVertIncoming(inc) {
		if cross {
			return glyphCross
		}
		switch {
		case sdrow > 0:
			return glyphTeeDown
		case sdrow < 0:
			return glyphTeeUp
		default:
			return glyphCross
		}
	}
	// A box corner and a junction glyph already carry the shape of the cell.
	// A second segment does not improve either of them.
	if isCorner(existing) || isJunction(existing) {
		return existing
	}
	return inc
}

func isHorizIncoming(r rune) bool {
	return r == glyphHoriz || r == '\\' || r == '/'
}

func isVertIncoming(r rune) bool {
	return r == glyphVert || r == '\\' || r == '/'
}

// isCorner reports whether r is a box-drawing corner glyph.
func isCorner(r rune) bool {
	switch r {
	case glyphTopLeft, glyphTopRight, glyphLowLeft, glyphLowRight:
		return true
	}
	return false
}

// isJunction reports whether r is a box-drawing junction glyph.
func isJunction(r rune) bool {
	switch r {
	case glyphCross, glyphTeeRight, glyphTeeLeft, glyphTeeDown, glyphTeeUp:
		return true
	}
	return false
}
