package cli

import (
	"io"
	"os"
	"testing"
)

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
	want := "+--+\n|hi|\n+--+\n"
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
