package tui

import "testing"

func TestChromeHitArrowSetsTool(t *testing.T) {
	ch := layoutChrome(80, "demo", 1, [2]int{4, 2}, toolBox, true, true, "")
	idx := indexOf(ch.footer, "arrow")
	if idx < 0 {
		t.Fatalf("footer %q has no arrow", ch.footer)
	}
	hit, ok := ch.hit(idx, 11, 80, 12)
	if !ok || hit.kind != chipTool || hit.tool != toolArrow {
		t.Errorf("hit = %+v ok=%v, want arrow tool", hit, ok)
	}
}

func TestChromePaddingIsAMiss(t *testing.T) {
	ch := layoutChrome(80, "demo", 1, [2]int{0, 0}, toolBox, true, true, "")
	if _, ok := ch.hit(0, 5, 80, 12); ok {
		t.Fatal("canvas row must miss chrome")
	}
}

func TestChromeDimUndoDoesNotHitUndo(t *testing.T) {
	ch := layoutChrome(80, "demo", 1, [2]int{0, 0}, toolBox, false, false, "")
	idx := indexOf(ch.footer, "undo")
	if idx < 0 {
		t.Fatalf("footer %q", ch.footer)
	}
	hit, ok := ch.hit(idx, 11, 80, 12)
	if !ok || hit.kind != chipUndo || hit.enabled {
		t.Fatalf("hit = %+v ok=%v, want disabled undo chip", hit, ok)
	}
}

func TestChromeNarrowDropsSendThenUndo(t *testing.T) {
	wide := layoutChrome(80, "demo", 1, [2]int{0, 0}, toolBox, true, true, "")
	if indexOf(wide.footer, "send") < 0 {
		t.Fatalf("wide footer %q", wide.footer)
	}
	mid := layoutChrome(36, "demo", 1, [2]int{0, 0}, toolBox, true, true, "")
	if indexOf(mid.footer, "send") >= 0 {
		t.Errorf("mid footer still has send: %q", mid.footer)
	}
	if indexOf(mid.footer, "undo") >= 0 {
		t.Errorf("mid footer still has undo: %q", mid.footer)
	}
	tiny := layoutChrome(20, "demo", 1, [2]int{0, 0}, toolBox, true, true, "")
	if indexOf(tiny.footer, "box") >= 0 && indexOf(tiny.footer, "b ") < 0 && indexOf(tiny.footer, "b l") < 0 {
		t.Errorf("tiny footer was not shortened: %q", tiny.footer)
	}
}

func TestChromeZoomHitUsesRuneColumns(t *testing.T) {
	ch := layoutChrome(30, "日本語", 1, [2]int{4, 2}, toolBox, true, true, "")
	zx := lastIndexOfRunes(ch.header, "1x")
	if zx < 0 {
		t.Fatalf("header %q has no 1x", ch.header)
	}
	var zoom chip
	for _, c := range ch.chips {
		if c.kind == chipZoom {
			zoom = c
			break
		}
	}
	if zoom.kind != chipZoom {
		t.Fatal("no zoom chip")
	}
	if zoom.x0 != zx {
		t.Errorf("zoom chip x0=%d, want rune index %d in %q", zoom.x0, zx, ch.header)
	}
	hit, ok := ch.hit(zx, 0, 30, 12)
	if !ok || hit.kind != chipZoom {
		t.Errorf("hit at rune col %d = %+v ok=%v, want chipZoom", zx, hit, ok)
	}
}

func TestChromeBadgeDoesNotDropChipsAtWidth80(t *testing.T) {
	ch := layoutChrome(80, "demo", 1, [2]int{0, 0}, toolBox, true, true, "13x5")
	if indexOf(ch.footer, "send") < 0 {
		t.Errorf("footer with badge dropped send: %q", ch.footer)
	}
	if indexOf(ch.footer, "arrow") < 0 {
		t.Errorf("footer with badge dropped arrow: %q", ch.footer)
	}
	if indexOf(ch.footer, "[box]") < 0 || indexOf(ch.footer, "line") < 0 {
		t.Errorf("footer with badge used short labels: %q", ch.footer)
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
