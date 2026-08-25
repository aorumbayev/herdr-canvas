package tui

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"herdr-canvas/internal/canvas"
)

// 8-color-safe styles, so they hold under NO_COLOR and low-color terminals.
var (
	styBanner = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	styFaint  = lipgloss.NewStyle().Faint(true)
	styActive = lipgloss.NewStyle().Reverse(true)
	styOn     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

// wline is one row in both plain form (for width math and tests) and styled
// form (for display). Styling wraps the whole clipped row, so it never changes
// the row's terminal width.
type wline struct{ plain, styled string }

type wtab struct {
	title string
	body  []string
	ex    *canvas.Diagram // live example, nil when the tab is prose only
}

func boxEl(x1, y1, x2, y2 int, label string) canvas.Element {
	return canvas.Element{Type: canvas.Box, X1: x1, Y1: y1, X2: x2, Y2: y2, Label: label}
}

func lineEl(x1, y1, x2, y2 int) canvas.Element {
	return canvas.Element{Type: canvas.Line, X1: x1, Y1: y1, X2: x2, Y2: y2, Arrow: canvas.ArrowEnd}
}

// wtabs teach the flow: draw a sketch, send it to an agent, the agent draws
// back, you loop, and here is what the loop is good for.
var wtabs = []wtab{
	{"Draw", []string{
		"Draw on a free canvas: boxes, lines, arrows, text, freeform.",
		"Tools: 1 select 2 box 3 line 4 arrow 5 text 6 draw.",
		"Drag to place each shape. Each shape has a stable id.",
	}, &canvas.Diagram{Elements: []canvas.Element{
		boxEl(0, 0, 8, 3, "api"),
		lineEl(9, 1, 13, 1),
		boxEl(14, 0, 22, 3, "db"),
	}}},

	{"Send", []string{
		"Press s to send the full canvas to an agent.",
		"Select any agent in any herdr harness.",
		"herdr puts two commands into that agent's input:",
		"  export - the picture and a legend of each element",
		"  skill  - how the agent draws",
		"You see them first. Nothing operates until you send.",
	}, nil},

	{"Agent", []string{
		"The agent reads your canvas and draws on it.",
		"It can add boxes, arrows, and notes, then apply the result.",
		"You see the agent changes on your canvas immediately.",
	}, &canvas.Diagram{Elements: []canvas.Element{
		boxEl(0, 0, 7, 3, "web"),
		lineEl(8, 1, 12, 1),
		boxEl(13, 0, 21, 3, "api"),
		lineEl(22, 1, 26, 1),
		boxEl(27, 0, 34, 3, "db"),
		lineEl(35, 1, 39, 1),
		boxEl(40, 0, 49, 3, "cache"),
	}}},

	{"Loop", []string{
		"Now it is a two-way loop:",
		"1. You draw or change some shapes.",
		"2. You send the canvas to an agent.",
		"3. The agent expands it and applies the result.",
		"4. You see the change, then send again.",
		"Continue back and forth until the diagram is correct.",
	}, nil},

	{"Uses", []string{
		"Use it to:",
		"- See how a complex process or data flow works.",
		"- Show an unfamiliar codebase as a diagram.",
		"- Tell an agent detailed requirements with a picture.",
		"- Make a rough drawing into a clear diagram together.",
	}, nil},
}

func tabCell(i, active int) string {
	if i == active {
		return "[" + wtabs[i].title + "]"
	}
	return " " + wtabs[i].title + " "
}

func tabStrip(active int) string {
	var b strings.Builder
	for i := range wtabs {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(tabCell(i, active))
	}
	return b.String()
}

// tabStripStyled reverses the active cell; other cells are faint.
func tabStripStyled(active int) string {
	var b strings.Builder
	for i := range wtabs {
		if i > 0 {
			b.WriteString("  ")
		}
		if i == active {
			b.WriteString(styActive.Render(tabCell(i, active)))
		} else {
			b.WriteString(styFaint.Render(tabCell(i, active)))
		}
	}
	return b.String()
}

func checkbox(dismiss bool) string {
	mark := " "
	if dismiss {
		mark = "x"
	}
	return "[" + mark + "] Do not show at next start   (d to toggle)"
}

// welcomeLines lays out one tab: banner, tab strip, live example, prose, and
// the remembered-dismiss checkbox. Each row carries a plain and a styled form.
func welcomeLines(tabs []wtab, active, w, h int, dismiss bool) []wline {
	if len(tabs) == 0 {
		return nil
	}
	if active < 0 || active >= len(tabs) {
		active = 0
	}
	t := tabs[active]
	counter := "(" + strconv.Itoa(active+1) + " of " + strconv.Itoa(len(tabs)) + ")"

	plain := func(s string) wline { c := clipRunes(s, w); return wline{c, c} }
	styled := func(s string, st lipgloss.Style) wline { c := clipRunes(s, w); return wline{c, st.Render(c)} }

	tab := plain(tabStrip(active))
	if tab.plain == tabStrip(active) { // unclipped: safe to style per cell
		tab.styled = tabStripStyled(active)
	}

	lines := []wline{
		styled(banner(counter, w), styBanner),
		styled(rule(w), styFaint),
		tab,
		styled(rule(w), styFaint),
		plain(""),
	}
	if t.ex != nil && len(t.body) > 0 {
		lines = append(lines, plain(t.body[0]), plain(""))
		for _, l := range strings.Split(t.ex.Render().String(), "\n") {
			lines = append(lines, plain("  "+l))
		}
		lines = append(lines, plain(""))
		for _, l := range t.body[1:] {
			lines = append(lines, plain(l))
		}
	} else {
		for _, l := range t.body {
			lines = append(lines, plain(l))
		}
	}
	cb := plain(checkbox(dismiss))
	if dismiss {
		cb = styled(checkbox(dismiss), styOn)
	}
	lines = append(lines, plain(""), cb, styled(rule(w), styFaint),
		styled("tab switch   d hide   q close", styFaint))

	if h > 0 && len(lines) > h {
		lines = append(lines[:max(1, h-1)], lines[len(lines)-1])
	}
	return lines
}

// welcomeBody is the plain text of a tab, for width math and tests.
func welcomeBody(tabs []wtab, active, w, h int, dismiss bool) string {
	rows := welcomeLines(tabs, active, w, h, dismiss)
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.plain
	}
	return strings.Join(out, "\n")
}

func banner(counter string, w int) string {
	left := "Welcome to herdr-canvas - a quick tour"
	gap := w - len([]rune(left)) - len([]rune(counter))
	if gap < 2 {
		return clipRunes(left+"  "+counter, w)
	}
	return left + strings.Repeat(" ", gap) + counter
}

func rule(w int) string {
	if w < 1 {
		w = 1
	}
	return strings.Repeat("─", w)
}
