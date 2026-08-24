package canvas

// PlaceText maps a string onto canvas cells. Each line starts at (x, y+line).
// Newlines move down and are not drawn. CR is ignored so CRLF is one break.
// endX, endY is the insertion point after the last character.
func PlaceText(x, y int, text string) (cells []Cell, endX, endY int) {
	endX, endY = x, y
	for _, r := range text {
		switch r {
		case '\r':
			continue
		case '\n':
			endY++
			endX = x
		default:
			cells = append(cells, Cell{X: endX, Y: endY, Ch: string(r)})
			endX++
		}
	}
	return cells, endX, endY
}
