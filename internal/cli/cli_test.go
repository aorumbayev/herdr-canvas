package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestCLIVersionFlags(t *testing.T) {
	var long bytes.Buffer
	root := newRootCmd()
	root.SetOut(&long)
	root.SetArgs([]string{"--version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("--version: %v", err)
	}

	var short bytes.Buffer
	root = newRootCmd()
	root.SetOut(&short)
	root.SetArgs([]string{"-v"})
	if err := root.Execute(); err != nil {
		t.Fatalf("-v: %v", err)
	}

	if long.String() != short.String() {
		t.Fatalf("--version = %q, -v = %q", long.String(), short.String())
	}
	if !strings.Contains(long.String(), "dev") {
		t.Fatalf("version output = %q, want to contain dev", long.String())
	}
	if f := newRootCmd().Flags().ShorthandLookup("v"); f == nil || f.Name != "version" {
		t.Fatal("-v must be the version flag")
	}
}

func run(t *testing.T, args ...string) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	root := newRootCmd()
	root.SetArgs(args)
	cmdErr := root.Execute()
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	if cmdErr != nil {
		t.Logf("execute %v: %v", args, cmdErr)
	}
	return string(out)
}

func TestCLIBoxTextExport(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	run(t, "new", "demo")
	run(t, "--name", "demo", "box", "0", "0", "3", "2")
	run(t, "--name", "demo", "text", "1", "1", "hi")
	got := run(t, "--name", "demo", "export")
	want := "┌──┐\n│hi│\n└──┘\n" +
		"b1 box 0,0-3,2\n" +
		"t2 text 1,1 \"hi\"\n"
	if got != want {
		t.Errorf("export = %q, want %q", got, want)
	}
}

func TestCLIReferentialReject(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	run(t, "new", "demo")
	root := newRootCmd()
	root.SetArgs([]string{"--name", "demo", "delete", "nope"})
	if err := root.Execute(); err == nil {
		t.Fatal("want referential error, got nil")
	}
}

func TestCLIMissingDiagramFails(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	root := newRootCmd()
	root.SetArgs([]string{"--name", "nope", "box", "0", "0", "1", "1"})
	if err := root.Execute(); err == nil {
		t.Fatal("want missing-diagram error, got nil")
	}
	root = newRootCmd()
	root.SetArgs([]string{"--name", "nope", "export"})
	if err := root.Execute(); err == nil {
		t.Fatal("want missing-diagram error from export, got nil")
	}
}

func TestCLICreateFlagCreatesDiagram(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	root := newRootCmd()
	root.SetArgs([]string{"--name", "demo", "--create", "box", "0", "0", "3", "2", "hi"})
	if err := root.Execute(); err != nil {
		t.Fatalf("box --create: %v", err)
	}
	got := run(t, "--name", "demo", "export")
	want := "┌──┐\n│hi│\n└──┘\n" +
		"b1 box 0,0-3,2 \"hi\"\n"
	if got != want {
		t.Errorf("export = %q, want %q", got, want)
	}
}

func TestCLIColorAndFill(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	run(t, "new", "demo")
	run(t, "--name", "demo", "box", "0", "0", "3", "2", "--color", "red", "--fill")
	run(t, "--name", "demo", "color", "b1", "blue")
	run(t, "--name", "demo", "fill", "b1", "off")
	got := run(t, "--name", "demo", "export")
	if !strings.Contains(got, "b1 box 0,0-3,2 blue") {
		t.Errorf("export = %q, want blue box without fill", got)
	}
	root := newRootCmd()
	root.SetArgs([]string{"--name", "demo", "fill", "b1", "maybe"})
	if err := root.Execute(); err == nil {
		t.Fatal("want error for fill maybe")
	}
}

func TestCLIEdgeFollowsItsBoxes(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	run(t, "new", "demo")
	run(t, "--name", "demo", "box", "0", "0", "4", "2")
	run(t, "--name", "demo", "box", "0", "8", "4", "10")
	run(t, "--name", "demo", "edge", "b1", "b2", "retry")
	if got := run(t, "--name", "demo", "export"); !strings.Contains(got, `l3 edge b1->b2 arrow end "retry"`) {
		t.Errorf("export = %q, want an edge legend line", got)
	}
	run(t, "--name", "demo", "move", "b2", "20", "0")
	got := run(t, "--name", "demo", "export")
	if !strings.Contains(got, "═") {
		t.Errorf("export = %q, want a doubled run after the move", got)
	}
	run(t, "--name", "demo", "unedge", "l3")
	if got := run(t, "--name", "demo", "export"); strings.Contains(got, "edge") {
		t.Errorf("export = %q, want no edge after unedge", got)
	}
}

func TestCLIEdgeRejectsBadEnds(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	run(t, "new", "demo")
	run(t, "--name", "demo", "box", "0", "0", "4", "2")
	run(t, "--name", "demo", "text", "0", "8", "hi")
	cases := [][]string{
		{"--name", "demo", "edge", "b1", "b9"},
		{"--name", "demo", "edge", "b1", "t2"},
	}
	for _, args := range cases {
		root := newRootCmd()
		root.SetArgs(args)
		if err := root.Execute(); err == nil {
			t.Errorf("%v = nil, want an error", args)
		}
	}
}

func TestCLIDeleteBoxRemovesItsEdges(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	run(t, "new", "demo")
	run(t, "--name", "demo", "box", "0", "0", "4", "2")
	run(t, "--name", "demo", "box", "0", "8", "4", "10")
	run(t, "--name", "demo", "edge", "b1", "b2")
	run(t, "--name", "demo", "delete", "b1")
	if got := run(t, "--name", "demo", "export"); strings.Contains(got, "l3") {
		t.Errorf("export = %q, want the edge gone with its box", got)
	}
}
