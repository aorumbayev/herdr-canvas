package canvas

import (
	"fmt"
	"strings"
)

// Prompt returns the message the canvas writes into an agent input. The
// message must be enough on its own. An agent that stops to find the binary or
// to read SKILL.md spends minutes before it draws, so the message states the
// few rules that decide whether a first edit is correct and says no more.
func Prompt(d *Diagram) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Diagram %q, open beside this pane. `herdr-canvas` is on your PATH.\n\n", d.Name)

	sb.WriteString("```\n")
	body := Export(d)
	if body == "" {
		body = "(empty)"
	}
	sb.WriteString(body)
	sb.WriteString("\n```\n\n")

	if len(d.Elements) > 0 {
		for _, e := range d.Elements {
			sb.WriteString(describe(e) + "\n")
		}
		sb.WriteString("\n")
	}

	fmt.Fprintf(&sb, "```sh\nN=%q\n", d.Name)
	sb.WriteString(`herdr-canvas --name "$N" box <x1> <y1> <x2> <y2> [label]
herdr-canvas --name "$N" line <x1> <y1> <x2> <y2> [--arrow end]
herdr-canvas --name "$N" text <x> <y> <text>
herdr-canvas --name "$N" move <id> <dx> <dy>
herdr-canvas --name "$N" label <id> <label>
herdr-canvas --name "$N" delete <id>
herdr-canvas --name "$N" export
` + "```\n\n")

	sb.WriteString(`x grows right, y grows down, both start at 0 and never go negative.
A line runs on y first, then on x, so it bends once; give it a corner cell,
not a diagonal. Box borders and junction glyphs are automatic. Each command
prints the new element id. Run export to see your work.
`)
	return sb.String()
}

func describe(e Element) string {
	switch e.Type {
	case Box:
		s := fmt.Sprintf("%s box (%d,%d)-(%d,%d)", e.ID, e.X1, e.Y1, e.X2, e.Y2)
		if e.Label != "" {
			s += fmt.Sprintf(" %q", e.Label)
		}
		return s
	case Line:
		s := fmt.Sprintf("%s line (%d,%d)-(%d,%d)", e.ID, e.X1, e.Y1, e.X2, e.Y2)
		if e.Arrow != "" && e.Arrow != ArrowNone {
			s += " arrow " + string(e.Arrow)
		}
		return s
	case Text:
		return fmt.Sprintf("%s text (%d,%d) %q", e.ID, e.X, e.Y, e.Text)
	case Freeform:
		return fmt.Sprintf("%s freeform %d cells", e.ID, len(e.Cells))
	}
	return e.ID
}
