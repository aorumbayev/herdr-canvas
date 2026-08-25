package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"herdr-canvas/internal/canvas"
	"herdr-canvas/internal/store"
)

var errStub = errors.New("stub")

func keyTab(shift bool) tea.KeyPressMsg {
	m := tea.KeyMod(0)
	if shift {
		m = tea.ModShift
	}
	return tea.KeyPressMsg{Code: tea.KeyTab, Mod: m}
}

func TestWelcomeBodyBannerAndCounter(t *testing.T) {
	out := welcomeBody(wtabs, 0, 66, 40, true)
	if !strings.Contains(out, "Welcome to herdr-canvas") {
		t.Error("banner missing")
	}
	if !strings.Contains(out, "(1 of 5)") {
		t.Error("counter missing")
	}
}

func TestWelcomeBodyMarksActiveTab(t *testing.T) {
	out := welcomeBody(wtabs, 1, 66, 40, true)
	if !strings.Contains(out, "[Send]") {
		t.Error("active tab not bracketed")
	}
	if strings.Contains(out, "[Draw]") {
		t.Error("inactive tab must not be bracketed")
	}
}

func TestWelcomeBodyCheckboxReflectsDismiss(t *testing.T) {
	if !strings.Contains(welcomeBody(wtabs, 0, 66, 40, true), "[x] Do not show") {
		t.Error("want checked box when dismiss")
	}
	if !strings.Contains(welcomeBody(wtabs, 0, 66, 40, false), "[ ] Do not show") {
		t.Error("want empty box when not dismiss")
	}
}

func TestWelcomeBodyShowsRealExample(t *testing.T) {
	out := welcomeBody(wtabs, 0, 66, 40, true)
	if !strings.Contains(out, "api") {
		t.Error("Draw tab must render the real example diagram")
	}
	if !strings.Contains(out, "┌") {
		t.Error("example should be a real box render")
	}
}

func TestWelcomeBodyClipsToWidth(t *testing.T) {
	w := 30
	for _, line := range strings.Split(welcomeBody(wtabs, 4, w, 40, false), "\n") {
		if n := len([]rune(line)); n > w {
			t.Fatalf("line exceeds width %d: %q (%d)", w, line, n)
		}
	}
}

func welcomeModel() model {
	return model{phase: phaseWelcome, welcomeTab: 0, welcomeDismiss: true, width: 66, height: 40}
}

func TestWelcomeTabCyclesForwardAndWraps(t *testing.T) {
	m := welcomeModel()
	m.welcomeKey(keyTab(false))
	if m.welcomeTab != 1 {
		t.Fatalf("tab: want 1, got %d", m.welcomeTab)
	}
	m.welcomeTab = len(wtabs) - 1
	m.welcomeKey(keyTab(false))
	if m.welcomeTab != 0 {
		t.Fatalf("wrap: want 0, got %d", m.welcomeTab)
	}
}

func TestWelcomeTabBackwardWraps(t *testing.T) {
	m := welcomeModel()
	m.welcomeKey(keyTab(true))
	if m.welcomeTab != len(wtabs)-1 {
		t.Fatalf("shift+tab wrap: want %d, got %d", len(wtabs)-1, m.welcomeTab)
	}
}

func TestWelcomeArrowsSwitchTabs(t *testing.T) {
	m := welcomeModel()
	m.welcomeKey(key("right"))
	m.welcomeKey(key("right"))
	m.welcomeKey(key("left"))
	if m.welcomeTab != 1 {
		t.Fatalf("arrows: want 1, got %d", m.welcomeTab)
	}
}

func TestWelcomeDToggles(t *testing.T) {
	m := welcomeModel()
	m.welcomeKey(key("d"))
	if m.welcomeDismiss {
		t.Fatal("d should toggle dismiss off")
	}
	m.welcomeKey(key("d"))
	if !m.welcomeDismiss {
		t.Fatal("d should toggle dismiss back on")
	}
}

func TestWelcomeCloseMarksWhenDismiss(t *testing.T) {
	m := welcomeModel()
	marked := false
	m.welcomeMark = func() error { marked = true; return nil }
	if cmd := m.welcomeKey(key("q")); cmd != nil {
		t.Fatal("close must not quit the app")
	}
	if m.phase != phaseEdit {
		t.Fatalf("want phaseEdit, got %v", m.phase)
	}
	if !marked {
		t.Fatal("dismiss set: close should mark seen")
	}
}

func TestWelcomeCloseSkipsMarkWhenNotDismiss(t *testing.T) {
	m := welcomeModel()
	m.welcomeDismiss = false
	marked := false
	m.welcomeMark = func() error { marked = true; return nil }
	m.welcomeKey(key("esc"))
	if marked {
		t.Fatal("no dismiss: close must not mark seen")
	}
}

func TestOpenWelcomeResetsState(t *testing.T) {
	m := model{phase: phaseEdit, welcomeTab: 3, welcomeDismiss: false, anchored: true}
	m.openWelcome()
	if m.phase != phaseWelcome || m.welcomeTab != 0 || !m.welcomeDismiss || m.anchored {
		t.Fatalf("openWelcome did not reset: %+v", m)
	}
}

func TestWelcomeMarkErrorSurfaces(t *testing.T) {
	m := welcomeModel()
	m.welcomeMark = func() error { return errStub }
	m.welcomeKey(key("q"))
	if m.status == "" {
		t.Fatal("a Mark error should surface to status")
	}
}

func TestMaybeWelcomeOpensWhenUnseenAndEmpty(t *testing.T) {
	m := model{phase: phaseEdit, d: &canvas.Diagram{}, welcomeSeen: func() (bool, error) { return false, nil }}
	m.maybeWelcome()
	if m.phase != phaseWelcome {
		t.Fatalf("want phaseWelcome, got %v", m.phase)
	}
}

func TestMaybeWelcomeSkipsWhenSeen(t *testing.T) {
	m := model{phase: phaseEdit, d: &canvas.Diagram{}, welcomeSeen: func() (bool, error) { return true, nil }}
	m.maybeWelcome()
	if m.phase != phaseEdit {
		t.Fatalf("seen: want phaseEdit, got %v", m.phase)
	}
}

func TestMaybeWelcomeSkipsWhenNonEmpty(t *testing.T) {
	m := model{phase: phaseEdit, d: &canvas.Diagram{Elements: []canvas.Element{boxEl(0, 0, 2, 2, "")}},
		welcomeSeen: func() (bool, error) { return false, nil }}
	m.maybeWelcome()
	if m.phase != phaseEdit {
		t.Fatalf("non-empty: want phaseEdit, got %v", m.phase)
	}
}

func TestPickerFirstOpenShowsWelcomeOnce(t *testing.T) {
	s := &store.Store{Base: t.TempDir()}
	a, b := &canvas.Diagram{Name: "a"}, &canvas.Diagram{Name: "b"}
	for _, d := range []*canvas.Diagram{a, b} {
		if err := s.Save(d); err != nil {
			t.Fatal(err)
		}
	}
	m := model{s: s, d: &canvas.Diagram{}, phase: phasePick, width: 40, height: 12,
		welcomeSeen: func() (bool, error) { return false, nil }}

	m.openCanvas(a)
	if m.phase != phaseWelcome {
		t.Fatalf("first picker open: want welcome, got %v", m.phase)
	}
	m.phase = phaseEdit // user closes the tour
	m.openCanvas(b)
	if m.phase == phaseWelcome {
		t.Fatal("tour reopened on a later in-session switch")
	}
}

func TestShouldWelcome(t *testing.T) {
	empty := &canvas.Diagram{}
	full := &canvas.Diagram{Elements: []canvas.Element{boxEl(0, 0, 2, 2, "")}}
	cases := []struct {
		seen bool
		d    *canvas.Diagram
		want bool
	}{
		{false, empty, true},
		{true, empty, false},
		{false, full, false},
	}
	for _, c := range cases {
		if got := shouldWelcome(c.seen, c.d); got != c.want {
			t.Errorf("shouldWelcome(%v, %d elems) = %v, want %v", c.seen, len(c.d.Elements), got, c.want)
		}
	}
}
