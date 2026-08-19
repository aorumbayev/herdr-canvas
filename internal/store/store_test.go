package store

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"herdr-canvas/internal/canvas"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	s := &Store{Base: t.TempDir()}
	d := &canvas.Diagram{Name: "demo"}
	for _, c := range []canvas.Command{
		canvas.BoxCmd{X1: 0, Y1: 0, X2: 2, Y2: 2, Label: "hi"},
		canvas.LineCmd{X1: 0, Y1: 0, X2: 3, Y2: 3, Arrow: canvas.ArrowEnd},
		canvas.TextCmd{X: 5, Y: 5, Text: "héllo"},
		canvas.DrawCmd{Cells: []canvas.Cell{{X: 0, Y: 0, Ch: "#"}, {X: 1, Y: 1, Ch: "┼"}}},
	} {
		if err := d.Apply(c); err != nil {
			t.Fatalf("Apply %T: %v", c, err)
		}
	}
	if err := s.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Load("demo")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Name != d.Name {
		t.Errorf("name = %q, want %q", got.Name, d.Name)
	}
	if len(got.Elements) != len(d.Elements) {
		t.Fatalf("got %d elements, want %d", len(got.Elements), len(d.Elements))
	}
	for i := range d.Elements {
		if !reflect.DeepEqual(got.Elements[i], d.Elements[i]) {
			t.Errorf("element %d = %+v, want %+v", i, got.Elements[i], d.Elements[i])
		}
	}
	if got.Next != d.Next {
		t.Errorf("Next counter = %d, want %d (must survive save/load)", got.Next, d.Next)
	}
}

func TestSaveWritesHumanReadableJSON(t *testing.T) {
	s := &Store{Base: t.TempDir()}
	d := &canvas.Diagram{Name: "demo"}
	if err := d.Apply(canvas.DrawCmd{Cells: []canvas.Cell{{X: 0, Y: 0, Ch: "#"}}}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := s.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}
	b, err := os.ReadFile(s.Path("demo"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	for _, want := range []string{`"id": "f1"`, `"ch": "#"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("saved JSON missing %q; got:\n%s", want, b)
		}
	}
}

func TestDirHonorsXDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg")
	if got := Dir(); got != "/tmp/xdg/herdr-canvas" {
		t.Errorf("Dir() = %q, want /tmp/xdg/herdr-canvas", got)
	}
}

func TestSaveRejectsUnsafeName(t *testing.T) {
	s := &Store{Base: t.TempDir()}
	d := &canvas.Diagram{Name: "../evil"}
	if err := s.Save(d); err == nil {
		t.Fatal("want error for path-traversal name, got nil")
	}
}
