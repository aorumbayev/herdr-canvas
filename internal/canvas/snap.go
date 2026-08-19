package canvas

// junction returns the glyph for a cell where two line segments meet. The
// parameter existing is the glyph already in the cell. The parameter inc is
// the glyph of the incoming segment. If cross is true, the incoming segment
// continues through the cell. If cross is false, the incoming segment stops in
// the cell and makes a tee. The parameters sdcol and sdrow give the direction
// of the tee stub. Each of these two parameters is -1, 0 or 1.
func junction(existing, inc rune, cross bool, sdcol, sdrow int) rune {
	if existing == '|' && (inc == '-' || inc == '\\' || inc == '/') {
		if cross {
			return '┼'
		}
		switch {
		case sdcol > 0:
			return '├'
		case sdcol < 0:
			return '┤'
		default:
			return '┼'
		}
	}
	if existing == '-' && (inc == '|' || inc == '\\' || inc == '/') {
		if cross {
			return '┼'
		}
		switch {
		case sdrow > 0:
			return '┬'
		case sdrow < 0:
			return '┴'
		default:
			return '┼'
		}
	}
	if existing == '+' || isJunction(existing) {
		return existing
	}
	return inc
}

// isJunction reports whether r is a box-drawing junction glyph.
func isJunction(r rune) bool {
	switch r {
	case '┼', '├', '┤', '┬', '┴':
		return true
	}
	return false
}
