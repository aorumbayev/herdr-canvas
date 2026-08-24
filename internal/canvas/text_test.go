package canvas

import "testing"

func TestPlaceTextCRLFIsOneBreak(t *testing.T) {
	cells, endX, endY := PlaceText(2, 3, "a\r\nb")
	if len(cells) != 2 || cells[0] != (Cell{X: 2, Y: 3, Ch: "a"}) || cells[1] != (Cell{X: 2, Y: 4, Ch: "b"}) {
		t.Fatalf("cells = %+v", cells)
	}
	if endX != 3 || endY != 4 {
		t.Errorf("end = %d,%d, want 3,4", endX, endY)
	}
}
