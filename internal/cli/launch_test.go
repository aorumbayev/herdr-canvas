package cli

import (
	"errors"
	"strings"
	"testing"
)

type fakeHost struct {
	panes      []string
	canvas     map[string]bool
	readErr    error
	closed     []string
	opened     []string
	focusedCwd string
}

func (f *fakeHost) PaneIDs(string) ([]string, error) { return f.panes, nil }

func (f *fakeHost) RunsProgram(paneID, prog string) (bool, error) {
	if f.readErr != nil {
		return false, f.readErr
	}
	if prog != binaryName {
		return false, nil
	}
	return f.canvas[paneID], nil
}

func (f *fakeHost) ClosePane(paneID string) error {
	f.closed = append(f.closed, paneID)
	return nil
}

func (f *fakeHost) OpenSplit(cwd string) error {
	f.opened = append(f.opened, cwd)
	return nil
}

func (f *fakeHost) FocusedCwd(string) (string, error) {
	if f.focusedCwd == "" {
		return "", errors.New("no focused pane")
	}
	return f.focusedCwd, nil
}

func TestToggleOpensWhenNoCanvasPaneIsOpen(t *testing.T) {
	f := &fakeHost{panes: []string{"w1:p1", "w1:p2"}, canvas: map[string]bool{},
		focusedCwd: "/repos/thing"}
	if err := toggle(f, "w1"); err != nil {
		t.Fatalf("toggle: %v", err)
	}
	// The split must open on the repository the person is looking at. The
	// working directory here is the plugin root, a detached checkout of this
	// repository, which would name the diagram after the plugin version.
	if len(f.opened) != 1 || f.opened[0] != "/repos/thing" {
		t.Errorf("opened = %v, want one split on the focused pane directory", f.opened)
	}
	if len(f.closed) != 0 {
		t.Errorf("closed = %v, want none", f.closed)
	}
}

func TestToggleClosesEveryCanvasPane(t *testing.T) {
	f := &fakeHost{
		panes:  []string{"w1:p1", "w1:p2", "w1:p3"},
		canvas: map[string]bool{"w1:p2": true, "w1:p3": true},
	}
	if err := toggle(f, "w1"); err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if got := strings.Join(f.closed, ","); got != "w1:p2,w1:p3" {
		t.Errorf("closed = %q, want both canvas panes", got)
	}
	if len(f.opened) != 0 {
		t.Errorf("opened = %v, want none — a second canvas must not stack", f.opened)
	}
}

func TestToggleRefusesWhenAPaneReadFails(t *testing.T) {
	f := &fakeHost{
		panes:   []string{"w1:p1"},
		canvas:  map[string]bool{},
		readErr: errors.New("socket closed"),
	}
	// A failed read must not read as "no canvas pane"; that would stack a
	// second canvas on top of the first one.
	if err := toggle(f, "w1"); err == nil {
		t.Fatal("toggle returned no error after a failed pane read")
	}
	if len(f.opened) != 0 {
		t.Errorf("opened = %v, want none", f.opened)
	}
}

func TestToggleNeedsAWorkspace(t *testing.T) {
	f := &fakeHost{}
	if err := toggle(f, ""); err == nil {
		t.Fatal("toggle returned no error outside herdr")
	}
}
