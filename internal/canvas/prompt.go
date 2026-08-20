package canvas

import "fmt"

// Prompt is the text send writes into an agent input. It names the open
// diagram and two commands: export to see the picture, skill to see how to
// draw. It does not paste the picture itself.
func Prompt(d *Diagram) string {
	n := d.Name
	return fmt.Sprintf(
		`The canvas beside you is %q. Run this to see what is on it:

herdr-canvas --name %q export

Then, unless you already ran it in this session, run this to see how to draw on it:

herdr-canvas skill

In one or two sentences, tell me what you see. Ask whether I want to draw or change anything, unless I already said what to do.`,
		n, n,
	)
}
