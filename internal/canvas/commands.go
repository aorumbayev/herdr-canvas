package canvas

// Command is a mutation that the gate can validate and commit. An unexported
// method seals the interface, so Apply must handle every implementation.
type Command interface {
	command()
}

// BoxCmd adds a Box (two corners) with an optional label.
type BoxCmd struct {
	X1, Y1, X2, Y2 int
	Label          string
	Color          string
	Fill           bool
}

// LineCmd adds a Line (two endpoints) with an optional arrow.
type LineCmd struct {
	X1, Y1, X2, Y2 int
	Arrow          Arrow
	Color          string
}

// EdgeCmd adds a Line held by reference to two boxes. The endpoints come from
// the boxes, so the line follows them.
type EdgeCmd struct {
	From, To string
	Label    string
	Arrow    Arrow
	Color    string
}

// UnedgeCmd removes an edge. The two boxes stay.
type UnedgeCmd struct {
	ID string
}

// TextCmd adds a Text string at a coordinate.
type TextCmd struct {
	X, Y  int
	Text  string
	Color string
}

// DrawCmd adds a Freeform element.
type DrawCmd struct {
	Cells []Cell
	Color string
}

// MoveCmd translates an existing element by (DX, DY).
type MoveCmd struct {
	ID     string
	DX, DY int
}

// DeleteCmd removes an existing element.
type DeleteCmd struct {
	ID string
}

// LabelCmd sets the label on an existing element.
type LabelCmd struct {
	ID    string
	Label string
}

// TextSetCmd replaces the string on an existing Text element. It does not
// move the element or change its id.
type TextSetCmd struct {
	ID   string
	Text string
}

// ColorCmd sets an element's foreground color. "default" clears it.
type ColorCmd struct {
	ID    string
	Color string
}

// FillCmd sets whether a box paints its interior.
type FillCmd struct {
	ID   string
	Fill bool
}

func (BoxCmd) command()     {}
func (LineCmd) command()    {}
func (EdgeCmd) command()    {}
func (UnedgeCmd) command()  {}
func (TextCmd) command()    {}
func (DrawCmd) command()    {}
func (MoveCmd) command()    {}
func (DeleteCmd) command()  {}
func (LabelCmd) command()   {}
func (TextSetCmd) command() {}
func (ColorCmd) command()   {}
func (FillCmd) command()    {}
