package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runBatch feeds script to `batch` on stdin and returns stdout and the error.
func runBatch(t *testing.T, script string, args ...string) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	root := newRootCmd()
	root.SetIn(strings.NewReader(script))
	root.SetArgs(append(args, "batch"))
	cmdErr := root.Execute()
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out), cmdErr
}

func diagramJSON(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "herdr-canvas", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBatchMultiElement(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	run(t, "new", "demo")
	out, err := runBatch(t, `
# two boxes and the line between them
box 0 0 10 4 "web server" as web
box 0 8 10 12 "db" as db

line 5 4 5 8 --arrow end --color green
label web "web tier"
move db 2 0
`, "--name", "demo")
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	head, rest, _ := strings.Cut(out, "\n\n")
	if head != "b1 web\nb2 db\nl3" {
		t.Errorf("ids = %q, want %q", head, "b1 web\nb2 db\nl3")
	}
	for _, want := range []string{
		`b1 box 0,0-10,4 "web tier"`,
		`b2 box 2,8-12,12 "db"`,
		"l3 line 5,4-5,8 green",
	} {
		if !strings.Contains(rest, want) {
			t.Errorf("export = %q, want to contain %q", rest, want)
		}
	}
	if got := run(t, "--name", "demo", "export"); got != rest {
		t.Errorf("saved diagram = %q, want %q", got, rest)
	}
}

func TestBatchCreateFlag(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if _, err := runBatch(t, "box 0 0 3 2 hi\n", "--name", "fresh", "--create"); err != nil {
		t.Fatalf("batch --create: %v", err)
	}
	if got := run(t, "--name", "fresh", "export"); !strings.Contains(got, `b1 box 0,0-3,2 "hi"`) {
		t.Errorf("export = %q", got)
	}
}

func TestBatchMissingDiagramFails(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if _, err := runBatch(t, "box 0 0 3 2\n", "--name", "nope"); err == nil {
		t.Fatal("want missing-diagram error, got nil")
	}
}

func TestBatchRejected(t *testing.T) {
	cases := []struct {
		name   string
		script string
		want   string
	}{
		{"unknown verb", "box 0 0 1 1\nboxx 1 2\n", `line 2: unknown verb "boxx"`},
		{"rejected verb open", "box 0 0 1 1\nopen demo\n", `line 2: verb "open" is not allowed`},
		{"rejected verb export", "export\n", `line 1: verb "export" is not allowed`},
		{"rejected verb batch", "batch\n", `line 1: verb "batch" is not allowed`},
		{"bad coordinate", "box 0 0 1 x\n", `line 1: bad coordinate "x"`},
		{"too few args", "box 0 0 1\n", "line 1:"},
		{"bad flag", "box 0 0 1 1 --nope\n", "line 1: unknown flag: --nope"},
		{"unterminated quote", `text 0 0 "hi` + "\n", "line 1: unterminated quote"},
		{"apply error", "box 0 0 3 2\nmove b1 -5 0\n", "line 2:"},
		{"missing element", "box 0 0 3 2\ndelete b9\n", `line 2: unknown element id "b9"`},
		{"duplicate alias", "box 0 0 1 1 as a\nbox 2 2 3 3 as a\n", `line 2: alias "a" is already used`},
		{"id-shaped alias", "box 0 0 1 1 as b1\n", `line 1: alias "b1" looks like an element id`},
		{"bad alias syntax", "box 0 0 1 1 as 9lives\n", `line 1: alias "9lives" must match`},
		{"undefined alias", "label web hello\n", `line 1: no element or alias "web"`},
		{"alias on non-creating verb", "box 0 0 1 1 as a\nmove a 1 1 as b\n", "line 2: only a verb that makes a new element"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("XDG_DATA_HOME", dir)
			run(t, "new", "demo")
			before := diagramJSON(t, dir, "demo")
			_, err := runBatch(t, tc.script, "--name", "demo")
			if err == nil {
				t.Fatalf("want error containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want to contain %q", err, tc.want)
			}
			if got := diagramJSON(t, dir, "demo"); got != before {
				t.Errorf("diagram changed to %q, want untouched %q", got, before)
			}
		})
	}
}

func TestBatchAccepted(t *testing.T) {
	cases := []struct {
		name   string
		script string
		wantID string
		want   []string
	}{
		{
			name:   "comments and blank lines skipped",
			script: "\n# a comment\n   # indented comment\n\nbox 0 0 3 2\n",
			wantID: "b1",
			want:   []string{"b1 box 0,0-3,2"},
		},
		{
			name:   "edge joins two boxes the same batch creates",
			script: "box 0 0 9 2 web as web\nbox 0 8 9 10 db as db\nedge web db writes\n",
			wantID: "b1 web\nb2 db\nl3",
			want:   []string{`l3 edge b1->b2 arrow end "writes"`},
		},
		{
			name:   "unedge takes an alias",
			script: "box 0 0 9 2 a as a\nbox 0 8 9 10 b as b\nedge a b as wire\nunedge wire\n",
			wantID: "b1 a\nb2 b\nl3 wire",
			want:   []string{"b1 box 0,0-9,2", "b2 box 0,8-9,10"},
		},
		{
			name:   "quoted label keeps spaces",
			script: `box 0 0 9 2 "web server tier"` + "\n",
			wantID: "b1",
			want:   []string{`b1 box 0,0-9,2 "web server tier"`},
		},
		{
			name:   "single-quoted text keeps spaces",
			script: "text 0 0 'hello there'\n",
			wantID: "t1",
			want:   []string{`t1 text 0,0 "hello there"`},
		},
		{
			name:   "flags parse",
			script: "box 0 0 3 2 hi --color red --fill\nline 0 4 6 4 --arrow both --color blue\n",
			wantID: "b1\nl2",
			want:   []string{`b1 box 0,0-3,2 red fill "hi"`, "l2 line 0,4-6,4 blue"},
		},
		{
			name:   "alias resolves for every reference verb",
			script: "box 0 0 3 2 as a\nlabel a \"a label\"\ncolor a green\nfill a on\nmove a 1 0\n",
			wantID: "b1 a",
			want:   []string{`b1 box 1,0-4,2 green fill "a label"`},
		},
		{
			name:   "alias delete",
			script: "box 0 0 3 2 as gone\nbox 5 0 8 2\ndelete gone\n",
			wantID: "b1 gone\nb2",
			want:   []string{"b2 box 5,0-8,2"},
		},
		{
			name:   "negative move deltas",
			script: "box 4 4 6 6\nmove b1 -2 -1\n",
			wantID: "b1",
			want:   []string{"b1 box 2,3-4,5"},
		},
		{
			name:   "draw triples",
			script: "draw 0 0 # 1 1 @ --color cyan\n",
			wantID: "f1",
			want:   []string{"f1 draw 2"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			run(t, "new", "demo")
			out, err := runBatch(t, tc.script, "--name", "demo")
			if err != nil {
				t.Fatalf("batch: %v", err)
			}
			head, _, _ := strings.Cut(out, "\n\n")
			if head != tc.wantID {
				t.Errorf("ids = %q, want %q", head, tc.wantID)
			}
			saved := run(t, "--name", "demo", "export")
			for _, w := range tc.want {
				if !strings.Contains(saved, w) {
					t.Errorf("export = %q, want to contain %q", saved, w)
				}
			}
		})
	}
}
