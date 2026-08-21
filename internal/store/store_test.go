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

func TestSaveLeavesNoTempFileAndReplacesAtomically(t *testing.T) {
	base := t.TempDir()
	s := &Store{Base: base}
	if err := s.Save(&canvas.Diagram{Name: "demo"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Save(&canvas.Diagram{Name: "demo", Next: 7}); err != nil {
		t.Fatalf("Save again: %v", err)
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "demo.json" {
		t.Errorf("store holds %v, want only demo.json", entries)
	}
	d, err := s.Load("demo")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if d.Next != 7 {
		t.Errorf("Next = %d, want 7", d.Next)
	}
	info, err := os.Stat(s.Path("demo"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("mode = %v, want -rw-r--r--", info.Mode().Perm())
	}
}

func TestModTimeMissingDiagramIsZero(t *testing.T) {
	s := &Store{Base: t.TempDir()}
	got, err := s.ModTime("demo")
	if err != nil {
		t.Fatalf("ModTime: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("ModTime = %v, want zero", got)
	}
	if err := s.Save(&canvas.Diagram{Name: "demo"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err = s.ModTime("demo")
	if err != nil {
		t.Fatalf("ModTime: %v", err)
	}
	if got.IsZero() {
		t.Error("ModTime = zero after Save")
	}
}

func TestDeleteRemovesDiagram(t *testing.T) {
	s := &Store{Base: t.TempDir()}
	if err := s.Save(&canvas.Diagram{Name: "demo"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Delete("demo"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Load("demo"); !os.IsNotExist(err) {
		t.Fatalf("Load after Delete: %v, want not exist", err)
	}
	if err := s.Delete("demo"); err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
	if err := s.Delete("../evil"); err == nil {
		t.Fatal("want error for unsafe name")
	}
}
