package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"herdr-canvas/internal/canvas"
)

var paletteColors = []struct {
	digit string
	name  string
}{
	{"1", "red"},
	{"2", "green"},
	{"3", "yellow"},
	{"4", "blue"},
	{"5", "magenta"},
	{"6", "cyan"},
	{"7", "white"},
	{"8", "black"},
}

func colorDigit(d string) string {
	for _, p := range paletteColors {
		if p.digit == d {
			return p.name
		}
	}
	return ""
}

func layoutPaletteLine(width int) string {
	parts := []string{"0 default"}
	for _, p := range paletteColors {
		parts = append(parts, p.digit+" "+p.name)
	}
	line := strings.Join(parts, "  ")
	if w := len([]rune(line)); w > width {
		return string([]rune(line)[:width])
	}
	return line
}

func paletteColorAt(line string, x int) (string, bool) {
	x0 := 0
	parts := []string{"0 default"}
	for _, p := range paletteColors {
		parts = append(parts, p.digit+" "+p.name)
	}
	for i, part := range parts {
		if i > 0 {
			x0 += 2
		}
		w := len([]rune(part))
		if x >= x0 && x < x0+w {
			if i == 0 {
				return "", true
			}
			return paletteColors[i-1].name, true
		}
		x0 += w
	}
	return "", false
}

func (m *model) paletteClick(x int) {
	line := layoutPaletteLine(m.width)
	if color, ok := paletteColorAt(line, x); ok {
		if color == "" {
			m.pickBrushColor("")
		} else {
			m.pickBrushColor(color)
		}
	}
}

func brushHeaderSuffix(color string, fill bool) string {
	if color == "" && !fill {
		return ""
	}
	var parts []string
	if color != "" {
		parts = append(parts, color)
	}
	if fill {
		parts = append(parts, "fill")
	}
	return strings.Join(parts, " ")
}

func styleCell(cp canvas.CellPaint) string {
	if cp.FG == "" && cp.BG == "" && !cp.Reverse {
		return string(cp.Ch)
	}
	style := lipgloss.NewStyle()
	if cp.Reverse {
		style = style.Reverse(true)
	}
	if cp.FG != "" {
		style = style.Foreground(lipgloss.Color(ansiIndex(cp.FG)))
	}
	if cp.BG != "" {
		style = style.Background(lipgloss.Color(ansiIndex(cp.BG)))
	}
	return style.Render(string(cp.Ch))
}

func ansiIndex(name string) string {
	switch name {
	case "black":
		return "0"
	case "red":
		return "1"
	case "green":
		return "2"
	case "yellow":
		return "3"
	case "blue":
		return "4"
	case "magenta":
		return "5"
	case "cyan":
		return "6"
	case "white":
		return "7"
	case "8":
		return "8"
	default:
		return name
	}
}

func (v viewport) paintColored(cells map[[2]int]canvas.CellPaint, w, h int) string {
	var sb strings.Builder
	for row := 0; row < h; row++ {
		if row > 0 {
			sb.WriteByte('\n')
		}
		line := make([]rune, 0, w)
		styled := make([]string, 0, w)
		for col := 0; col < w; col++ {
			pt := [2]int{v.origin[0] + col, v.origin[1] + row}
			if cp, ok := cells[pt]; ok {
				seg := styleCell(cp)
				styled = append(styled, seg)
				line = append(line, cp.Ch)
			} else {
				styled = append(styled, " ")
				line = append(line, ' ')
			}
		}
		trim := len(line)
		for trim > 0 && line[trim-1] == ' ' {
			trim--
		}
		if trim == 0 {
			continue
		}
		sb.WriteString(strings.Join(styled[:trim], ""))
	}
	return sb.String()
}
