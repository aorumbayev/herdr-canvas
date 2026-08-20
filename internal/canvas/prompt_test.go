package canvas

import (
	"strings"
	"testing"
)

func TestPromptPointsAtExportAndSkill(t *testing.T) {
	d := &Diagram{Name: "herdr-canvas@main"}
	if err := d.Apply(BoxCmd{X1: 0, Y1: 0, X2: 4, Y2: 2, Label: "hi"}); err != nil {
		t.Fatal(err)
	}
	p := Prompt(d)
	if !strings.Contains(p, `herdr-canvas --name "herdr-canvas@main" export`) {
		t.Errorf("missing export:\n%s", p)
	}
	if !strings.Contains(p, "herdr-canvas skill") {
		t.Errorf("missing skill:\n%s", p)
	}
	if !strings.Contains(p, "unless you already ran it in this session") {
		t.Errorf("skill must be skippable:\n%s", p)
	}
	if strings.Contains(p, "┌") {
		t.Errorf("prompt pasted the picture:\n%s", p)
	}
	if strings.Contains(p, "box <x1>") {
		t.Errorf("prompt pasted the command table:\n%s", p)
	}
	if strings.HasSuffix(p, "\n\n") {
		t.Errorf("trailing blank line can submit: %q", p[len(p)-10:])
	}
}
