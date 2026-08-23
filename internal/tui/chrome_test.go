package tui

import "testing"

func TestChromeHitArrowSetsTool(t *testing.T) {
	ch := layoutChrome(80, "demo", [2]int{4, 2}, "", false, toolBox, true, true, "")
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
	ch := layoutChrome(80, "demo", [2]int{0, 0}, "", false, toolBox, true, true, "")
	if _, ok := ch.hit(0, 5, 80, 12); ok {
		t.Fatal("canvas row must miss chrome")
	}
}

func TestChromeDimUndoDoesNotHitUndo(t *testing.T) {
	ch := layoutChrome(80, "demo", [2]int{0, 0}, "", false, toolBox, false, false, "")
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
	wide := layoutChrome(80, "demo", [2]int{0, 0}, "", false, toolBox, true, true, "")
	if indexOf(wide.footer, "send") < 0 || indexOf(wide.footer, "help") < 0 {
		t.Fatalf("wide footer %q", wide.footer)
	}
	if indexOf(wide.footer, "1 sel") < 0 {
		t.Fatalf("wide footer missing select mapping: %q", wide.footer)
	}
	mid := layoutChrome(36, "demo", [2]int{0, 0}, "", false, toolBox, true, true, "")
	if indexOf(mid.footer, "send") >= 0 {
		t.Errorf("mid footer still has send: %q", mid.footer)
	}
	if indexOf(mid.footer, "help") >= 0 {
		t.Errorf("mid footer still has help: %q", mid.footer)
	}
	tiny := layoutChrome(20, "demo", [2]int{0, 0}, "", false, toolBox, false, false, "")
	if indexOf(tiny.footer, "2 box") >= 0 && indexOf(tiny.footer, "2") < 0 {
		t.Errorf("tiny footer was not shortened: %q", tiny.footer)
	}
}

func TestChromeHelpChipHits(t *testing.T) {
	ch := layoutChrome(80, "demo", [2]int{0, 0}, "", false, toolBox, true, true, "")
	idx := lastIndexOfRunes(ch.footer, "help")
	if idx < 0 {
		t.Fatalf("footer %q", ch.footer)
	}
	hit, ok := ch.hit(idx, 11, 80, 12)
	if !ok || hit.kind != chipHelp {
		t.Errorf("hit = %+v ok=%v, want chipHelp", hit, ok)
	}
}

func TestChromeNameHitUsesRuneColumns(t *testing.T) {
	ch := layoutChrome(30, "日本語", [2]int{4, 2}, "", false, toolBox, true, true, "")
	var name chip
	for _, c := range ch.chips {
		if c.kind == chipName {
			name = c
			break
		}
	}
	if name.kind != chipName {
		t.Fatal("no name chip")
	}
	if name.x0 != 0 {
		t.Errorf("name chip x0=%d, want 0 in %q", name.x0, ch.header)
	}
	hit, ok := ch.hit(0, 0, 30, 12)
	if !ok || hit.kind != chipName {
		t.Errorf("hit at 0 = %+v ok=%v, want chipName", hit, ok)
	}
}

func TestChromeHeaderHasRecenterNotZoom(t *testing.T) {
	ch := layoutChrome(80, "demo", [2]int{4, 2}, "", false, toolBox, true, true, "")
	if indexOf(ch.header, "1x") >= 0 {
		t.Fatalf("header still has zoom: %q", ch.header)
	}
	if indexOf(ch.header, "recenter") < 0 {
		t.Fatalf("header %q", ch.header)
	}
	idx := indexOf(ch.header, "recenter")
	hit, ok := ch.hit(idx, 0, 80, 12)
	if !ok || hit.kind != chipRecenter {
		t.Errorf("hit = %+v ok=%v, want chipRecenter", hit, ok)
	}
	idx = indexOf(ch.header, "canvases")
	if idx < 0 {
		t.Fatalf("header has no canvases control: %q", ch.header)
	}
	hit, ok = ch.hit(idx, 0, 80, 12)
	if !ok || hit.kind != chipCanvases {
		t.Errorf("hit = %+v ok=%v, want chipCanvases", hit, ok)
	}
}

func TestChromeFooterGroupIsCentered(t *testing.T) {
	ch := layoutChrome(80, "demo", [2]int{0, 0}, "", false, toolBox, true, true, "")
	idx := indexOf(ch.footer, "[2 box]")
	if idx < 1 {
		t.Fatalf("footer still left-aligned: %q", ch.footer)
	}
	hit, ok := ch.hit(idx, 11, 80, 12)
	if !ok || hit.tool != toolBox {
		t.Errorf("centered [2 box] miss: %+v ok=%v", hit, ok)
	}
}

func TestChromeSelectIsFirst(t *testing.T) {
	ch := layoutChrome(80, "demo", [2]int{0, 0}, "", false, toolSelect, true, true, "")
	sel := indexOf(ch.footer, "[1 sel]")
	box := indexOf(ch.footer, "2 box")
	if sel < 0 || box < 0 || sel > box {
		t.Fatalf("select should come first: %q", ch.footer)
	}
}

func TestChromeBadgeDoesNotDropChipsAtWidth80(t *testing.T) {
	ch := layoutChrome(80, "demo", [2]int{0, 0}, "", false, toolBox, true, true, "13x5")
	if indexOf(ch.footer, "send") < 0 {
		t.Errorf("footer with badge dropped send: %q", ch.footer)
	}
	if indexOf(ch.footer, "arrow") < 0 {
		t.Errorf("footer with badge dropped arrow: %q", ch.footer)
	}
	if indexOf(ch.footer, "[2 box]") < 0 || indexOf(ch.footer, "3 line") < 0 {
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
