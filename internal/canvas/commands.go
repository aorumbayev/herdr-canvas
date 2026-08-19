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
}

// LineCmd adds a Line (two endpoints) with an optional arrow.
type LineCmd struct {
	X1, Y1, X2, Y2 int
	Arrow          Arrow
}

// TextCmd adds a Text string at a coordinate.
type TextCmd struct {
	X, Y int
	Text string
}

// DrawCmd adds a Freeform element.
type DrawCmd struct {
	Cells []Cell
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

func (BoxCmd) command()    {}
func (LineCmd) command()   {}
func (TextCmd) command()   {}
func (DrawCmd) command()   {}
func (MoveCmd) command()   {}
func (DeleteCmd) command() {}
func (LabelCmd) command()  {}
