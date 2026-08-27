package cli

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"herdr-canvas/internal/canvas"
	"herdr-canvas/internal/store"
)

var (
	aliasPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
	idPattern    = regexp.MustCompile(`^[blft][0-9]+$`)
)

// batchLine is one parsed script line, ready to apply.
type batchLine struct {
	num   int
	text  string
	verb  elementVerb
	cmd   canvas.Command
	alias string   // name this line's new element takes, if any
	refs  []string // alias per leading id argument, empty where the arg was a real id
}

func batchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "batch",
		Short: "Apply a script of element commands from stdin, all or nothing",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			script, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return err
			}
			lines, err := parseBatch(string(script))
			if err != nil {
				return err
			}
			s, d, err := loadDiagram(cmd)
			if err != nil {
				return err
			}
			return applyBatch(s, d, lines)
		},
	}
}

// applyBatch applies every line to the in-memory diagram and saves once. A
// failure returns before the save, so the file on disk is untouched.
func applyBatch(s *store.Store, d *canvas.Diagram, lines []batchLine) error {
	ids := map[string]string{}
	var created []string
	for _, l := range lines {
		c := l.cmd
		if len(l.refs) > 0 {
			c = withRefs(c, l.refs, ids)
		}
		if err := d.Apply(c); err != nil {
			return lineErr(l, err)
		}
		if l.verb.creates {
			id := d.Elements[len(d.Elements)-1].ID
			if l.alias != "" {
				ids[l.alias] = id
				id += " " + l.alias
			}
			created = append(created, id)
		}
	}
	if err := s.Save(d); err != nil {
		return err
	}
	for _, c := range created {
		fmt.Println(c)
	}
	fmt.Println()
	fmt.Println(canvas.Export(d))
	return nil
}

// parseBatch turns the whole script into commands before anything is applied.
func parseBatch(script string) ([]batchLine, error) {
	var out []batchLine
	aliases := map[string]bool{}
	for i, text := range strings.Split(script, "\n") {
		l := batchLine{num: i + 1, text: strings.TrimRight(text, "\r")}
		trimmed := strings.TrimSpace(l.text)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if err := parseLine(&l, aliases); err != nil {
			return nil, lineErr(l, err)
		}
		out = append(out, l)
	}
	return out, nil
}

func parseLine(l *batchLine, aliases map[string]bool) error {
	toks, err := tokenize(l.text)
	if err != nil {
		return err
	}
	v, ok := lookupVerb(toks[0].text)
	if !ok {
		if rootCommandNames()[toks[0].text] {
			return fmt.Errorf("verb %q is not allowed in a batch", toks[0].text)
		}
		return fmt.Errorf("unknown verb %q", toks[0].text)
	}
	l.verb = v

	rest := toks[1:]
	if alias, n := aliasSuffix(rest); alias != "" {
		if !v.creates {
			return fmt.Errorf("only a verb that makes a new element can name an alias, not %s", v.name)
		}
		if err := checkAlias(alias, aliases); err != nil {
			return err
		}
		aliases[alias] = true
		l.alias = alias
		rest = rest[:n]
	}

	cmd := v.newCmd()
	words := make([]string, len(rest))
	for i, t := range rest {
		words[i] = t.text
	}
	if err := cmd.ParseFlags(words); err != nil {
		return err
	}
	args := cmd.Flags().Args()
	if err := cmd.ValidateArgs(args); err != nil {
		return err
	}
	if v.refs > 0 {
		l.refs = make([]string, v.refs)
		for i := 0; i < v.refs && i < len(args); i++ {
			if idPattern.MatchString(args[i]) {
				continue
			}
			if !aliases[args[i]] {
				return fmt.Errorf("no element or alias %q", args[i])
			}
			l.refs[i] = args[i]
		}
	}
	l.cmd, err = v.build(cmd, args)
	return err
}

func checkAlias(alias string, taken map[string]bool) error {
	switch {
	case !aliasPattern.MatchString(alias):
		return fmt.Errorf("alias %q must match [a-zA-Z][a-zA-Z0-9_-]*", alias)
	case idPattern.MatchString(alias):
		return fmt.Errorf("alias %q looks like an element id", alias)
	case taken[alias]:
		return fmt.Errorf("alias %q is already used in this batch", alias)
	}
	return nil
}

// aliasSuffix reports the alias in a trailing unquoted `as <name>`, and the
// token count that precedes it.
func aliasSuffix(toks []token) (string, int) {
	n := len(toks)
	if n < 2 || toks[n-2].quoted || toks[n-2].text != "as" {
		return "", n
	}
	return toks[n-1].text, n - 2
}

// withRefs swaps each aliased argument for the id the alias resolved to. An
// empty entry means that argument was already a real id.
func withRefs(c canvas.Command, refs []string, ids map[string]string) canvas.Command {
	id := func(i int) (string, bool) {
		if i >= len(refs) || refs[i] == "" {
			return "", false
		}
		return ids[refs[i]], true
	}
	switch t := c.(type) {
	case canvas.MoveCmd:
		if v, ok := id(0); ok {
			t.ID = v
		}
		return t
	case canvas.DeleteCmd:
		if v, ok := id(0); ok {
			t.ID = v
		}
		return t
	case canvas.UnedgeCmd:
		if v, ok := id(0); ok {
			t.ID = v
		}
		return t
	case canvas.LabelCmd:
		if v, ok := id(0); ok {
			t.ID = v
		}
		return t
	case canvas.ColorCmd:
		if v, ok := id(0); ok {
			t.ID = v
		}
		return t
	case canvas.FillCmd:
		if v, ok := id(0); ok {
			t.ID = v
		}
		return t
	case canvas.EdgeCmd:
		if v, ok := id(0); ok {
			t.From = v
		}
		if v, ok := id(1); ok {
			t.To = v
		}
		return t
	}
	return c
}

func rootCommandNames() map[string]bool {
	out := map[string]bool{}
	for _, c := range newRootCmd().Commands() {
		out[c.Name()] = true
	}
	return out
}

func lineErr(l batchLine, err error) error {
	return fmt.Errorf("batch: line %d: %v\n  %s", l.num, err, strings.TrimSpace(l.text))
}

type token struct {
	text   string
	quoted bool
}

// tokenize splits a script line on whitespace, honouring single and double
// quotes so a label may contain spaces.
func tokenize(line string) ([]token, error) {
	var (
		out   []token
		cur   strings.Builder
		open  bool
		quote rune
		have  bool
	)
	flush := func() {
		if have {
			out = append(out, token{text: cur.String(), quoted: quote != 0})
			cur.Reset()
			quote = 0
			have = false
		}
	}
	for _, r := range line {
		switch {
		case open && r == quote:
			open = false
		case open:
			cur.WriteRune(r)
		case r == '"' || r == '\'':
			open, quote, have = true, r, true
		case r == ' ' || r == '\t':
			flush()
		default:
			cur.WriteRune(r)
			have = true
		}
	}
	if open {
		return nil, fmt.Errorf("unterminated quote")
	}
	flush()
	if len(out) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	return out, nil
}
