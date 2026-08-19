package canvas

// Type is the concrete kind of an Element.
type Type string

const (
	Box      Type = "box"
	Line     Type = "line"
	Text     Type = "text"
	Freeform Type = "freeform"
)

// Arrow is the arrow placement on a Line.
type Arrow string

const (
	ArrowNone  Arrow = "none"
	ArrowStart Arrow = "start"
	ArrowEnd   Arrow = "end"
	ArrowBoth  Arrow = "both"
)

// Cell is one character at one (x, y) position, used by Freeform.
type Cell struct {
	X  int  `json:"x"`
	Y  int  `json:"y"`
	Ch byte `json:"ch"`
}

// Element is a single named entity in a Diagram. Fields are used per Type:
// Box uses X1,Y1,X2,Y2 + Label; Line uses X1,Y1,X2,Y2 + Arrow; Text uses
// X,Y + Text; Freeform uses Cells.
type Element struct {
	ID    string `json:"id"`
	Type  Type   `json:"type"`
	X1    int    `json:"x1,omitempty"`
	Y1    int    `json:"y1,omitempty"`
	X2    int    `json:"x2,omitempty"`
	Y2    int    `json:"y2,omitempty"`
	X     int    `json:"x,omitempty"`
	Y     int    `json:"y,omitempty"`
	Label string `json:"label,omitempty"`
	Text  string `json:"text,omitempty"`
	Arrow Arrow  `json:"arrow,omitempty"`
	Cells []Cell `json:"cells,omitempty"`
}

// Diagram is the aggregate root: a named set of Elements in z-order
// (slice order, later wins).
type Diagram struct {
	Name     string    `json:"name"`
	Elements []Element `json:"elements"`
	Next     int       `json:"next,omitempty"`
}
