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
	X  int    `json:"x"`
	Y  int    `json:"y"`
	Ch string `json:"ch"`
}

// Element is a single named entity in a Diagram. Each Type uses a different
// set of fields.
//
//	Box uses X1, Y1, X2, Y2 and Label.
//	Line uses X1, Y1, X2, Y2, Arrow and Label.
//	Text uses X, Y and Text.
//	Freeform uses Cells.
//
// A Line that names two boxes in From and To is an edge. The endpoints of an
// edge are derived from the two boxes and are never authored.
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
	Color string `json:"color,omitempty"`
	Fill  bool   `json:"fill,omitempty"`
	From  string `json:"from,omitempty"`
	To    string `json:"to,omitempty"`
	// Vertical says an edge leaves and enters its boxes through their top and
	// bottom borders. It is derived with the endpoints and never authored.
	Vertical bool `json:"vertical,omitempty"`
}

// IsEdge reports whether the element is a Line held by reference to two boxes.
func (e Element) IsEdge() bool {
	return e.Type == Line && e.From != "" && e.To != ""
}

// Diagram is the aggregate root. A Diagram is a named set of Elements. Later
// elements in the slice cover earlier elements.
type Diagram struct {
	Name     string    `json:"name"`
	Elements []Element `json:"elements"`
	Next     int       `json:"next,omitempty"`
}
