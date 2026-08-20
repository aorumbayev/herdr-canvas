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
	want := "┌──┐\n│hi│\n└──┘\n"
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
	want := "┌──┐\n│hi│\n└──┘\n"
	if got != want {
		t.Errorf("export = %q, want %q", got, want)
	}
}
