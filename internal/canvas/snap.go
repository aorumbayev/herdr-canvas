package canvas

// junction returns the glyph for a cell where two line segments meet. The
// parameter existing is the glyph already in the cell. The parameter inc is
// the glyph of the incoming segment. If cross is true, the incoming segment
// continues through the cell. If cross is false, the incoming segment stops in
// the cell and makes a tee. The parameters sdcol and sdrow give the direction
// of the tee stub. Each of these two parameters is -1, 0 or 1.
//
// Each arm of the junction keeps its own weight, so a single line that crosses
// an edge reads as a single line crossing a doubled one.
func junction(existing, inc rune, cross bool, sdcol, sdrow int) rune {
	if v, ok := vertWeight(existing); ok {
		if h, ok := horizIncoming(inc); ok {
			return meetOnVert(v, h, cross, sdcol)
		}
	}
	if h, ok := horizWeight(existing); ok {
		if v, ok := vertIncoming(inc); ok {
			return meetOnHoriz(h, v, cross, sdrow)
		}
	}
	// A box corner and a junction glyph already carry the shape of the cell.
	// A second segment does not improve either of them.
	if isCorner(existing) || isJunction(existing) {
		return existing
	}
	return inc
}

// meetOnVert returns the glyph for a horizontal segment arriving on a vertical
// run. The parameters vd and hd say whether each arm is doubled.
func meetOnVert(vd, hd, cross bool, sdcol int) rune {
	if cross || sdcol == 0 {
		return crossGlyph(vd, hd)
	}
	if sdcol > 0 {
		switch {
		case vd && hd:
			return '╠'
		case vd:
			return '╟'
		case hd:
			return '╞'
		}
		return glyphTeeRight
	}
	switch {
	case vd && hd:
		return '╣'
	case vd:
		return '╢'
	case hd:
		return '╡'
	}
	return glyphTeeLeft
}

// meetOnHoriz returns the glyph for a vertical segment arriving on a
// horizontal run.
func meetOnHoriz(hd, vd, cross bool, sdrow int) rune {
	if cross || sdrow == 0 {
		return crossGlyph(vd, hd)
	}
	if sdrow > 0 {
		switch {
		case vd && hd:
			return '╦'
		case vd:
			return '╥'
		case hd:
			return '╤'
		}
		return glyphTeeDown
	}
	switch {
	case vd && hd:
		return '╩'
	case vd:
		return '╨'
	case hd:
		return '╧'
	}
	return glyphTeeUp
}

func crossGlyph(vd, hd bool) rune {
	switch {
	case vd && hd:
		return '╬'
	case vd:
		return '╫'
	case hd:
		return '╪'
	}
	return glyphCross
}

// vertWeight reports whether r is a vertical run, and whether it is doubled.
func vertWeight(r rune) (double, ok bool) {
	switch r {
	case glyphVert:
		return false, true
	case dblVert:
		return true, true
	}
	return false, false
}

// horizWeight reports whether r is a horizontal run, and whether it is doubled.
func horizWeight(r rune) (double, ok bool) {
	switch r {
	case glyphHoriz:
		return false, true
	case dblHoriz:
		return true, true
	}
	return false, false
}

// vertIncoming and horizIncoming are the run tests for the arriving segment. A
// freeform slash arrives as a single run in either direction.
func vertIncoming(r rune) (double, ok bool) {
	if r == '\\' || r == '/' {
		return false, true
	}
	return vertWeight(r)
}

func horizIncoming(r rune) (double, ok bool) {
	if r == '\\' || r == '/' {
		return false, true
	}
	return horizWeight(r)
}

// isCorner reports whether r is a box-drawing corner glyph.
func isCorner(r rune) bool {
	switch r {
	case glyphTopLeft, glyphTopRight, glyphLowLeft, glyphLowRight:
		return true
	case dblTopLeft, dblTopRight, dblLowLeft, dblLowRight:
		return true
	}
	return false
}

// isJunction reports whether r is a box-drawing junction glyph.
func isJunction(r rune) bool {
	switch r {
	case glyphCross, glyphTeeRight, glyphTeeLeft, glyphTeeDown, glyphTeeUp:
		return true
	case '╬', '╠', '╣', '╦', '╩':
		return true
	case '╪', '╫', '╞', '╡', '╤', '╧', '╟', '╢', '╥', '╨':
		return true
	}
	return false
}
