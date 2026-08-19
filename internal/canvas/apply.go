package canvas

import (
	"fmt"
	"strconv"
)

// Apply validates a command against the diagram. Apply then commits the
// command, or returns an error that names the id and the rule.
func (d *Diagram) Apply(cmd Command) error {
	switch c := cmd.(type) {
	case BoxCmd:
		if err := validateCorners(c.X1, c.Y1, c.X2, c.Y2); err != nil {
			return fmt.Errorf("box: %w", err)
		}
		d.Elements = append(d.Elements, Element{
			ID:    d.nextID("b"),
			Type:  Box,
			X1:    c.X1,
			Y1:    c.Y1,
			X2:    c.X2,
			Y2:    c.Y2,
			Label: c.Label,
		})
	case LineCmd:
		if err := validateEndpoints(c.X1, c.Y1, c.X2, c.Y2); err != nil {
			return fmt.Errorf("line: %w", err)
		}
		d.Elements = append(d.Elements, Element{
			ID:    d.nextID("l"),
			Type:  Line,
			X1:    c.X1,
			Y1:    c.Y1,
			X2:    c.X2,
			Y2:    c.Y2,
			Arrow: c.Arrow,
		})
	case TextCmd:
		if c.X < 0 || c.Y < 0 {
			return fmt.Errorf("text: coordinates must be non-negative, got (%d,%d)", c.X, c.Y)
		}
		if c.Text == "" {
			return fmt.Errorf("text: text must be non-empty")
		}
		d.Elements = append(d.Elements, Element{
			ID:   d.nextID("t"),
			Type: Text,
			X:    c.X,
			Y:    c.Y,
			Text: c.Text,
		})
	case DrawCmd:
		if len(c.Cells) == 0 {
			return fmt.Errorf("draw: cell list must be non-empty")
		}
		for _, cell := range c.Cells {
			if cell.X < 0 || cell.Y < 0 {
				return fmt.Errorf("draw: cell coordinates must be non-negative, got (%d,%d)", cell.X, cell.Y)
			}
		}
		d.Elements = append(d.Elements, Element{
			ID:    d.nextID("f"),
			Type:  Freeform,
			Cells: c.Cells,
		})
	case MoveCmd:
		e, err := d.find(c.ID)
		if err != nil {
			return err
		}
		moved, err := translate(*e, c.DX, c.DY)
		if err != nil {
			return fmt.Errorf("move %s: %w", c.ID, err)
		}
		*e = moved
	case DeleteCmd:
		if _, err := d.find(c.ID); err != nil {
			return err
		}
		for i := range d.Elements {
			if d.Elements[i].ID == c.ID {
				d.Elements = append(d.Elements[:i], d.Elements[i+1:]...)
				break
			}
		}
	case LabelCmd:
		e, err := d.find(c.ID)
		if err != nil {
			return err
		}
		e.Label = c.Label
	default:
		return fmt.Errorf("unsupported command %T", cmd)
	}
	return nil
}

// translate moves the element e by (dx, dy). translate changes only the
// fields that belong to the type of the element. translate returns an error if
// the new geometry is not well-formed.
func translate(e Element, dx, dy int) (Element, error) {
	switch e.Type {
	case Box:
		e.X1, e.Y1, e.X2, e.Y2 = e.X1+dx, e.Y1+dy, e.X2+dx, e.Y2+dy
		if err := validateCorners(e.X1, e.Y1, e.X2, e.Y2); err != nil {
			return e, err
		}
	case Line:
		e.X1, e.Y1, e.X2, e.Y2 = e.X1+dx, e.Y1+dy, e.X2+dx, e.Y2+dy
		if err := validateEndpoints(e.X1, e.Y1, e.X2, e.Y2); err != nil {
			return e, err
		}
	case Text:
		e.X, e.Y = e.X+dx, e.Y+dy
		if e.X < 0 || e.Y < 0 {
			return e, fmt.Errorf("coordinates must be non-negative, got (%d,%d)", e.X, e.Y)
		}
	case Freeform:
		cells := make([]Cell, len(e.Cells))
		for i, c := range e.Cells {
			c.X, c.Y = c.X+dx, c.Y+dy
			if c.X < 0 || c.Y < 0 {
				return e, fmt.Errorf("cell coordinates must be non-negative, got (%d,%d)", c.X, c.Y)
			}
			cells[i] = c
		}
		e.Cells = cells
	default:
		return e, fmt.Errorf("unknown element type %q", e.Type)
	}
	return e, nil
}

// find returns a pointer to the element with the given id, or a
// referential-integrity error naming it.
func (d *Diagram) find(id string) (*Element, error) {
	for i := range d.Elements {
		if d.Elements[i].ID == id {
			return &d.Elements[i], nil
		}
	}
	return nil, fmt.Errorf("unknown element id %q", id)
}

// validateCorners enforces non-negative coordinates and x2>=x1, y2>=y1.
func validateCorners(x1, y1, x2, y2 int) error {
	if x1 < 0 || y1 < 0 || x2 < 0 || y2 < 0 {
		return fmt.Errorf("coordinates must be non-negative, got (%d,%d)-(%d,%d)", x1, y1, x2, y2)
	}
	if x2 < x1 || y2 < y1 {
		return fmt.Errorf("corner (%d,%d) precedes (%d,%d): x2>=x1 and y2>=y1 required", x2, y2, x1, y1)
	}
	return nil
}

// validateEndpoints enforces non-negative coordinates; lines may point in
// any direction.
func validateEndpoints(x1, y1, x2, y2 int) error {
	if x1 < 0 || y1 < 0 || x2 < 0 || y2 < 0 {
		return fmt.Errorf("coordinates must be non-negative, got (%d,%d)-(%d,%d)", x1, y1, x2, y2)
	}
	return nil
}

// nextID returns the next stable id for the given type letter. All types
// share one counter, and the counter only increases. The tool never reuses an
// id after a delete.
func (d *Diagram) nextID(letter string) string {
	n := d.Next
	for _, e := range d.Elements {
		if m := idNum(e.ID); m > n {
			n = m
		}
	}
	n++
	d.Next = n
	return fmt.Sprintf("%s%d", letter, n)
}

// idNum extracts the numeric suffix of a stable id like "b12".
func idNum(id string) int {
	i := 0
	for i < len(id) && (id[i] < '0' || id[i] > '9') {
		i++
	}
	n, err := strconv.Atoi(id[i:])
	if err != nil {
		return 0
	}
	return n
}
