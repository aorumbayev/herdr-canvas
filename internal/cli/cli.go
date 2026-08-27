package cli

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"herdr-canvas/internal/canvas"
	"herdr-canvas/internal/name"
	"herdr-canvas/internal/store"
	"herdr-canvas/internal/tui"
	"herdr-canvas/internal/version"
)

//go:embed SKILL.md
var skillDoc string

// Execute runs the herdr-canvas CLI and returns its exit error.
func Execute() error {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "herdr-canvas:", err)
		return err
	}
	return nil
}

func runTUI(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return tui.Run(cwd)
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "herdr-canvas",
		Short:         "Dead-simple ASCII diagram canvas",
		Version:       version.Version,
		RunE:          runTUI,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.InitDefaultVersionFlag()
	root.PersistentFlags().String("name", "", "diagram name (defaults to composite repo@branch)")
	root.PersistentFlags().Bool("create", false, "create the diagram when it does not exist")
	root.AddCommand(
		newCmd(),
		openCmd(),
		listCmd(),
		exportCmd(),
		skillCmd(),
		batchCmd(),
		launchCmd(),
		setupCmd(),
		updateCmd(),
	)
	for _, v := range elementVerbs {
		root.AddCommand(v.newCmd())
	}
	return root
}

// elementVerb ties a verb's cobra command to the builder that turns its
// arguments into a canvas.Command. The batch parser reuses both, so a verb is
// described once and cannot drift between the two entry points.
type elementVerb struct {
	name    string
	newCmd  func() *cobra.Command
	build   func(*cobra.Command, []string) (canvas.Command, error)
	creates bool // assigns a new element id
	refs    bool // first argument is an element id
}

var elementVerbs = []elementVerb{
	{name: "box", newCmd: boxCmd, build: buildBox, creates: true},
	{name: "line", newCmd: lineCmd, build: buildLine, creates: true},
	{name: "text", newCmd: textCmd, build: buildText, creates: true},
	{name: "draw", newCmd: drawCmd, build: buildDraw, creates: true},
	{name: "move", newCmd: moveCmd, build: buildMove, refs: true},
	{name: "delete", newCmd: deleteCmd, build: buildDelete, refs: true},
	{name: "label", newCmd: labelCmd, build: buildLabel, refs: true},
	{name: "color", newCmd: colorCmd, build: buildColor, refs: true},
	{name: "fill", newCmd: fillCmd, build: buildFill, refs: true},
}

func lookupVerb(name string) (elementVerb, bool) {
	for _, v := range elementVerbs {
		if v.name == name {
			return v, true
		}
	}
	return elementVerb{}, false
}

func openCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <name>",
		Short: "Open an existing diagram in the TUI",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.RunNamed(args[0])
		},
	}
}

func newCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new diagram",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s := store.New()
			if _, err := s.Load(args[0]); err == nil {
				return fmt.Errorf("diagram %q already exists", args[0])
			}
			return s.Save(&canvas.Diagram{Name: args[0]})
		},
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List diagrams in the central store",
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := store.New().List()
			if err != nil {
				return err
			}
			for _, n := range names {
				fmt.Println(n)
			}
			return nil
		},
	}
}

func exportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Render a diagram to compact grid text",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, d, err := loadDiagram(cmd)
			if err != nil {
				return err
			}
			fmt.Println(canvas.Export(d))
			return nil
		},
	}
}

func skillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "skill",
		Short: "Print the agent-facing SKILL.md",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Print(skillDoc)
			return nil
		},
	}
}

func boxCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "box <x1> <y1> <x2> <y2> [label]",
		Short: "Add a box (two corners) with an optional label",
		Args:  cobra.MinimumNArgs(4),
		RunE:  runElement(buildBox),
	}
	c.Flags().String("color", "", "foreground color: red|green|yellow|blue|magenta|cyan|white|black|default")
	c.Flags().Bool("fill", false, "paint the box interior")
	return c
}

func buildBox(cmd *cobra.Command, a []string) (canvas.Command, error) {
	n, err := ints(a[:4])
	if err != nil {
		return nil, err
	}
	fill, _ := cmd.Flags().GetBool("fill")
	return canvas.BoxCmd{
		X1: n[0], Y1: n[1], X2: n[2], Y2: n[3],
		Label: strings.Join(a[4:], " "),
		Color: flagString(cmd, "color"),
		Fill:  fill,
	}, nil
}

func lineCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "line <x1> <y1> <x2> <y2>",
		Short: "Add a line (two endpoints) with an optional arrow",
		Args:  cobra.ExactArgs(4),
		RunE:  runElement(buildLine),
	}
	c.Flags().String("arrow", string(canvas.ArrowNone), "arrow placement: none|start|end|both")
	c.Flags().String("color", "", "foreground color: red|green|yellow|blue|magenta|cyan|white|black|default")
	return c
}

func buildLine(cmd *cobra.Command, a []string) (canvas.Command, error) {
	n, err := ints(a)
	if err != nil {
		return nil, err
	}
	arr, err := arrow(flagString(cmd, "arrow"))
	if err != nil {
		return nil, err
	}
	return canvas.LineCmd{
		X1: n[0], Y1: n[1], X2: n[2], Y2: n[3],
		Arrow: arr,
		Color: flagString(cmd, "color"),
	}, nil
}

func textCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "text <x> <y> <text>",
		Short: "Place a text string at a coordinate",
		Args:  cobra.MinimumNArgs(3),
		RunE:  runElement(buildText),
	}
	c.Flags().String("color", "", "foreground color: red|green|yellow|blue|magenta|cyan|white|black|default")
	return c
}

func buildText(cmd *cobra.Command, a []string) (canvas.Command, error) {
	n, err := ints(a[:2])
	if err != nil {
		return nil, err
	}
	return canvas.TextCmd{
		X: n[0], Y: n[1], Text: strings.Join(a[2:], " "),
		Color: flagString(cmd, "color"),
	}, nil
}

func drawCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "draw <x> <y> <ch> [<x> <y> <ch> ...]",
		Short: "Set freeform cells (x y char triples)",
		Args:  cobra.MinimumNArgs(3),
		RunE:  runElement(buildDraw),
	}
	c.Flags().String("color", "", "foreground color: red|green|yellow|blue|magenta|cyan|white|black|default")
	return c
}

func buildDraw(cmd *cobra.Command, a []string) (canvas.Command, error) {
	if len(a)%3 != 0 {
		return nil, fmt.Errorf("draw requires x y ch triples")
	}
	cells := make([]canvas.Cell, 0, len(a)/3)
	for i := 0; i < len(a); i += 3 {
		x, err := strconv.Atoi(a[i])
		if err != nil {
			return nil, fmt.Errorf("bad x %q: %w", a[i], err)
		}
		y, err := strconv.Atoi(a[i+1])
		if err != nil {
			return nil, fmt.Errorf("bad y %q: %w", a[i+1], err)
		}
		cells = append(cells, canvas.Cell{X: x, Y: y, Ch: a[i+2]})
	}
	return canvas.DrawCmd{Cells: cells, Color: flagString(cmd, "color")}, nil
}

func moveCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "move <id> <dx> <dy>",
		Short: "Translate an element",
		Args:  cobra.ExactArgs(3),
		RunE:  runElement(buildMove),
	}
	// dx/dy may be negative; stop flag parsing at the first positional arg.
	c.Flags().SetInterspersed(false)
	return c
}

func buildMove(cmd *cobra.Command, a []string) (canvas.Command, error) {
	n, err := ints(a[1:])
	if err != nil {
		return nil, err
	}
	return canvas.MoveCmd{ID: a[0], DX: n[0], DY: n[1]}, nil
}

func deleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Remove an element",
		Args:  cobra.ExactArgs(1),
		RunE:  runElement(buildDelete),
	}
}

func buildDelete(cmd *cobra.Command, a []string) (canvas.Command, error) {
	return canvas.DeleteCmd{ID: a[0]}, nil
}

func labelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "label <id> <label>",
		Short: "Set an element's label",
		Args:  cobra.MinimumNArgs(2),
		RunE:  runElement(buildLabel),
	}
}

func buildLabel(cmd *cobra.Command, a []string) (canvas.Command, error) {
	return canvas.LabelCmd{ID: a[0], Label: strings.Join(a[1:], " ")}, nil
}

func colorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "color <id> <name>",
		Short: "Set an element's foreground color",
		Args:  cobra.ExactArgs(2),
		RunE:  runElement(buildColor),
	}
}

func buildColor(cmd *cobra.Command, a []string) (canvas.Command, error) {
	return canvas.ColorCmd{ID: a[0], Color: a[1]}, nil
}

func fillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fill <id> <on|off>",
		Short: "Fill or unfill a box interior",
		Args:  cobra.ExactArgs(2),
		RunE:  runElement(buildFill),
	}
}

func buildFill(cmd *cobra.Command, a []string) (canvas.Command, error) {
	on, err := onOff(a[1])
	if err != nil {
		return nil, err
	}
	return canvas.FillCmd{ID: a[0], Fill: on}, nil
}

func onOff(s string) (bool, error) {
	switch s {
	case "on":
		return true, nil
	case "off":
		return false, nil
	default:
		return false, fmt.Errorf("fill state %q (want on|off)", s)
	}
}

// runElement loads the diagram. runElement applies the command that build
// returns. runElement then saves the diagram.
func runElement(build func(*cobra.Command, []string) (canvas.Command, error)) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		c, err := build(cmd, args)
		if err != nil {
			return err
		}
		s, d, err := loadDiagram(cmd)
		if err != nil {
			return err
		}
		if err := d.Apply(c); err != nil {
			return err
		}
		return s.Save(d)
	}
}

func loadDiagram(cmd *cobra.Command) (*store.Store, *canvas.Diagram, error) {
	n, err := resolveName(cmd)
	if err != nil {
		return nil, nil, err
	}
	s := store.New()
	d, err := s.Load(n)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, nil, err
		}
		if create, _ := cmd.Flags().GetBool("create"); !create {
			return nil, nil, fmt.Errorf("no diagram %q; run `herdr-canvas new %s` or pass --create", n, n)
		}
		d = &canvas.Diagram{Name: n}
	}
	return s, d, nil
}

func resolveName(cmd *cobra.Command) (string, error) {
	if n, _ := cmd.Flags().GetString("name"); n != "" {
		return n, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	nm, err := name.Composite(cwd)
	if err != nil {
		return "", fmt.Errorf("no --name given and not inside a git repository")
	}
	return nm.String(), nil
}

func ints(ss []string) ([]int, error) {
	out := make([]int, len(ss))
	for i, s := range ss {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("bad coordinate %q: %w", s, err)
		}
		out[i] = n
	}
	return out, nil
}

func arrow(s string) (canvas.Arrow, error) {
	a := canvas.Arrow(s)
	switch a {
	case canvas.ArrowNone, canvas.ArrowStart, canvas.ArrowEnd, canvas.ArrowBoth:
		return a, nil
	}
	return "", fmt.Errorf("invalid arrow %q (want none|start|end|both)", s)
}

// flagString returns a string flag's value, or "" when the flag is absent.
func flagString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}
