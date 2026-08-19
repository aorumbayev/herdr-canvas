package canvas

// junction returns the glyph where an incoming line segment drawn as `inc`
// lands on a cell already holding `existing`. When `cross` is true the line
// continues through the cell; otherwise it terminates there (a tee) and the
// stub points in the direction (sdcol, sdrow) — -1/0/1 in each axis.
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
