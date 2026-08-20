package tui

import (
	"fmt"
	"strings"
)

type chipKind int

const (
	chipNone chipKind = iota
	chipTool
	chipUndo
	chipRedo
	chipSend
	chipZoom
	chipRecenter
)

type chip struct {
	kind    chipKind
	tool    tool
	x0, x1  int
	row     int
	enabled bool
}

type chrome struct {
	header string
	footer string
	chips  []chip
}

type chipSpec struct {
	label   string
	kind    chipKind
	tool    tool
	enabled bool
	group   int
}

func layoutChrome(width int, name string, zoom int, cursor [2]int, active tool, canUndo, canRedo bool, badge string) chrome {
	if width < 1 {
		width = 1
	}
	zoomLabel := formatZoom(zoom)
	right := fmt.Sprintf("%s  recenter  (%d,%d)", zoomLabel, cursor[0], cursor[1])
	left := name
	leftW := len([]rune(left))
	rightW := len([]rune(right))
	gap := width - leftW - rightW
	if gap < 1 {
		gap = 1
		if leftW+1+rightW > width {
			left = truncate(left, max(1, width-rightW-1))
			leftW = len([]rune(left))
		}
	}
	header := left + strings.Repeat(" ", gap) + right
	if hr := []rune(header); len(hr) > width {
		header = string(hr[:width])
	}

	var ch chrome
	ch.header = header
	zx := lastIndexOfRunes(header, zoomLabel)
	if zx >= 0 {
		zw := len([]rune(zoomLabel))
		ch.chips = append(ch.chips, chip{kind: chipZoom, x0: zx, x1: zx + zw, row: 0, enabled: true})
	}
	rx := lastIndexOfRunes(header, "recenter")
	if rx >= 0 {
		rw := len([]rune("recenter"))
		ch.chips = append(ch.chips, chip{kind: chipRecenter, x0: rx, x1: rx + rw, row: 0, enabled: true})
	}

	full := []chipSpec{
		{"box", chipTool, toolBox, true, 0},
		{"line", chipTool, toolLine, true, 0},
		{"arrow", chipTool, toolArrow, true, 0},
		{"text", chipTool, toolText, true, 0},
		{"draw", chipTool, toolDraw, true, 0},
		{"move", chipTool, toolMove, true, 0},
		{"del", chipTool, toolDelete, true, 0},
		{"undo", chipUndo, 0, canUndo, 1},
		{"redo", chipRedo, 0, canRedo, 1},
		{"send", chipSend, 0, true, 2},
	}
	short := []chipSpec{
		{"b", chipTool, toolBox, true, 0},
		{"l", chipTool, toolLine, true, 0},
		{"a", chipTool, toolArrow, true, 0},
		{"t", chipTool, toolText, true, 0},
		{"d", chipTool, toolDraw, true, 0},
		{"m", chipTool, toolMove, true, 0},
		{"x", chipTool, toolDelete, true, 0},
		{"↶", chipUndo, 0, canUndo, 1},
		{"↷", chipRedo, 0, canRedo, 1},
		{"s", chipSend, 0, true, 2},
	}

	specs := full
	if !fits(specs, width, active, badge) {
		specs = dropGroup(specs, 2)
	}
	if !fits(specs, width, active, badge) {
		specs = dropGroup(specs, 1)
	}
	if !fits(specs, width, active, badge) {
		specs = short
		if !fits(specs, width, active, badge) {
			specs = dropGroup(specs, 2)
		}
		if !fits(specs, width, active, badge) {
			specs = dropGroup(specs, 1)
		}
	}

	footer, chips := renderFooter(specs, active, badge, width, true)
	for i := range chips {
		chips[i].row = 1
	}
	ch.footer = footer
	ch.chips = append(ch.chips, chips...)
	return ch
}

func dropGroup(in []chipSpec, group int) []chipSpec {
	out := make([]chipSpec, 0, len(in))
	for _, s := range in {
		if s.group != group {
			out = append(out, s)
		}
	}
	return out
}

func fits(specs []chipSpec, width int, active tool, badge string) bool {
	_, chips := renderFooter(specs, active, "", 1<<30, false)
	groupW := 0
	if len(chips) > 0 {
		groupW = chips[len(chips)-1].x1
	}
	if groupW > width {
		return false
	}
	if badge != "" {
		groupW += 1 + len([]rune(badge))
	}
	return groupW <= width
}

func renderFooter(specs []chipSpec, active tool, badge string, width int, center bool) (string, []chip) {
	var b strings.Builder
	var chips []chip
	x := 0
	lastGroup := 0
	for i, s := range specs {
		if i > 0 && s.group != lastGroup {
			b.WriteString(" │ ")
			x += 3
		} else if i > 0 {
			b.WriteByte(' ')
			x++
		}
		lastGroup = s.group
		label := s.label
		if s.kind == chipTool && s.tool == active {
			label = "[" + s.label + "]"
		}
		start := x
		b.WriteString(label)
		x += len([]rune(label))
		chips = append(chips, chip{kind: s.kind, tool: s.tool, x0: start, x1: x, enabled: s.enabled})
	}
	groupW := x
	leftPad := 0
	if center {
		leftPad = (width - groupW) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for i := range chips {
			chips[i].x0 += leftPad
			chips[i].x1 += leftPad
		}
	}
	out := strings.Repeat(" ", leftPad) + b.String()
	if badge != "" && leftPad+groupW+1+len([]rune(badge)) <= width {
		out += " " + badge
	}
	if w := len([]rune(out)); w > width {
		out = string([]rune(out)[:width])
	}
	clipped := chips[:0]
	for _, c := range chips {
		if c.x0 >= width {
			continue
		}
		if c.x1 > width {
			c.x1 = width
		}
		clipped = append(clipped, c)
	}
	return out, clipped
}

func (c chrome) hit(x, y, termW, termH int) (chip, bool) {
	row := -1
	if y == 0 {
		row = 0
	}
	if y == termH-1 {
		row = 1
	}
	if row < 0 {
		return chip{}, false
	}
	for _, ch := range c.chips {
		if ch.row == row && x >= ch.x0 && x < ch.x1 {
			return ch, true
		}
	}
	return chip{}, false
}

func lastIndexOfRunes(s, sub string) int {
	runes := []rune(s)
	subRunes := []rune(sub)
	if len(subRunes) == 0 {
		return -1
	}
	for i := len(runes) - len(subRunes); i >= 0; i-- {
		match := true
		for j := range subRunes {
			if runes[i+j] != subRunes[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func formatZoom(tenths int) string {
	switch tenths {
	case zoom05:
		return "0.5x"
	case zoom15:
		return "1.5x"
	case zoom2:
		return "2x"
	default:
		return "1x"
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
