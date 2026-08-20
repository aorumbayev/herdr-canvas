package canvas

import (
	"fmt"
	"strings"
)

// Prompt returns the message the canvas sends to an agent. The message holds
// the diagram as text, and the commands that edit it. An agent that reads the
// message can answer about the diagram and change it without more context.
func Prompt(d *Diagram) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Here is the herdr-canvas diagram %q, beside this pane.\n\n", d.Name)
	sb.WriteString("```\n")
	body := Export(d)
	if body == "" {
		body = "(the diagram is empty)"
	}
	sb.WriteString(body)
	sb.WriteString("\n```\n\n")

	sb.WriteString("Elements, in the order they render. A later element covers an earlier one:\n\n")
	if len(d.Elements) == 0 {
		sb.WriteString("- none\n")
	}
	for _, e := range d.Elements {
		sb.WriteString("- " + describe(e) + "\n")
	}

	fmt.Fprintf(&sb, `
To change the diagram, run these commands. Each one takes --name %q:

    herdr-canvas --name %q export
    herdr-canvas --name %q box <x1> <y1> <x2> <y2> [label]
    herdr-canvas --name %q line <x1> <y1> <x2> <y2> [--arrow none|start|end|both]
    herdr-canvas --name %q text <x> <y> <text>
    herdr-canvas --name %q draw <x> <y> <ch> [<x> <y> <ch> ...]
    herdr-canvas --name %q move <id> <dx> <dy>
    herdr-canvas --name %q label <id> <label>
    herdr-canvas --name %q delete <id>

Run `+"`herdr-canvas skill`"+` for the full rules. Read the diagram again with
`+"`export`"+` after you change it. I am looking at the same diagram while you
work, so tell me what you changed.
`, d.Name, d.Name, d.Name, d.Name, d.Name, d.Name, d.Name, d.Name, d.Name)
	return sb.String()
}

func describe(e Element) string {
	switch e.Type {
	case Box:
		s := fmt.Sprintf("%s box (%d,%d) to (%d,%d)", e.ID, e.X1, e.Y1, e.X2, e.Y2)
		if e.Label != "" {
			s += fmt.Sprintf(" labelled %q", e.Label)
		}
		return s
	case Line:
		s := fmt.Sprintf("%s line (%d,%d) to (%d,%d)", e.ID, e.X1, e.Y1, e.X2, e.Y2)
		if e.Arrow != "" && e.Arrow != ArrowNone {
			s += fmt.Sprintf(", arrow %s", e.Arrow)
		}
		return s
	case Text:
		return fmt.Sprintf("%s text at (%d,%d): %q", e.ID, e.X, e.Y, e.Text)
	case Freeform:
		return fmt.Sprintf("%s freeform, %d cells", e.ID, len(e.Cells))
	}
	return e.ID
}
