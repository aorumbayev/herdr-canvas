package canvas

import (
	"fmt"
	"strconv"
)

// Apply parses, applies, and validates a command against the diagram,
// committing it or rejecting it with an actionable error.
func (d *Diagram) Apply(cmd any) error {
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
		e.X1, e.Y1, e.X2, e.Y2 = e.X1+c.DX, e.Y1+c.DY, e.X2+c.DX, e.Y2+c.DY
		e.X, e.Y = e.X+c.DX, e.Y+c.DY
		for i := range e.Cells {
			e.Cells[i].X += c.DX
			e.Cells[i].Y += c.DY
		}
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
	}
	return nil
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

// nextID returns the next stable id for the given type letter, continuing a
// single monotonic counter shared across all types; never reused after
// deletion.
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
