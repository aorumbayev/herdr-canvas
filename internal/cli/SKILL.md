# herdr-canvas — agent skill

herdr-canvas is a dead-simple ASCII diagram canvas. You read a diagram with
`export` and edit it with `batch`. You never write raw grid characters and
never rewrite the whole JSON file.

**Write one batch, not one command per element.** A batch is a script of
element commands on stdin. herdr-canvas applies the whole script or none of
it, saves once, and prints the new ids and the picture. A single-verb
invocation is the special case for one small tweak.

```sh
herdr-canvas --name demo batch <<'EOF'
box 0 0 20 4 "web server" as web
box 0 10 20 14 "database" as db
line 10 4 10 10 --arrow end
label web "web tier"
EOF
```

```
b1 web
b2 db
l3

┌───────────────────┐
│web tier           │
...
```

## Model

- **Diagram** — one JSON file `<name>.json` in the central store.
- **Element** — a box, line, text, or freeform cell group, each with a
  stable id (`b1`, `l2`, `t3`, `f4`, …). Ids are assigned by the tool,
  monotonically increasing, never reused after deletion.
- **Grid** — the rendered picture. herdr-canvas derives the grid from the
  elements. Later elements in the array cover earlier elements. herdr-canvas
  never stores the grid. The grid is never the source of truth.

## Central store

`$XDG_DATA_HOME/herdr-canvas/` (or `~/.local/share/herdr-canvas/`). One JSON
file per diagram.

## The batch script

`herdr-canvas batch` reads the script from stdin. One command per line. Each
line uses the same words as the single-verb form, without the
`herdr-canvas` prefix.

- A blank line does nothing.
- A line whose first character is `#` is a comment.
- Quote a label or a text string that holds spaces: `box 0 0 8 3 "web server"`.
  Single quotes work too.
- Flags work as they do on the command line: `--color`, `--fill`, `--arrow`.
- One batch edits one diagram. Use `--name` once, on the `batch` command.
  Pass `--create` to make the diagram when it does not exist.
- Legal verbs: `box`, `line`, `edge`, `text`, `draw`, `move`, `delete`,
  `unedge`, `label`, `color`, `fill`. No other verb is legal. `open`, `new`,
  `list`, `export`, `skill`, `setup`, `update` and `batch` are rejected.

### Aliases

A new element gets its id only when the batch runs. To point at an element
that the same batch creates, give it an alias. Write ` as <alias>` at the end
of a line that makes a new element: `box`, `line`, `edge`, `text` or `draw`.
Then use the alias anywhere an id is accepted. `edge` takes an id or an alias
in both of its first two arguments, so one batch can make two boxes and
connect them.

```
box 0 0 20 4 "web" as web
box 0 10 20 14 "db" as db
edge web db "writes"
color web green
```

An alias starts with a letter and holds letters, digits, `_` and `-`. An
alias must not look like an id (`b1`, `l2`, `t3`, `f4`). An alias must be
unique in the batch. An alias must be defined before the line that uses it.
An alias lives only for the run. herdr-canvas never writes an alias to the
JSON file.

### All or nothing

herdr-canvas parses the whole script first. A bad line stops the batch before
anything is applied. herdr-canvas then applies every command to a diagram in
memory. A command that fails stops the batch too. herdr-canvas saves the file
once, at the end, and only when every line succeeded. A failed batch writes
nothing and exits non-zero.

An error names the line number and echoes the line:

```
herdr-canvas: batch: line 4: unknown verb "boxx"
  boxx 1 2 3 4
```

### Output

A batch prints the id of each new element, in the order the batch created
them, with its alias when it has one. Then a blank line. Then the same text
that `export` prints. You do not need a second call to read the result.

```
b1 web
b2 db
l3

<the grid, a blank line, then the legend>
```

## Commands

```
herdr-canvas batch [--name <name>] [--create]   # apply a script from stdin
herdr-canvas export [--name <name>]      # print the grid as text + legend
herdr-canvas new <name>                  # create an empty diagram
herdr-canvas open <name>                 # open an existing diagram in the TUI
herdr-canvas list                        # list diagrams
herdr-canvas box <x1> <y1> <x2> <y2> [label] [--color <name>] [--fill]
herdr-canvas line <x1> <y1> <x2> <y2> [--arrow none|start|end|both] [--color <name>]
herdr-canvas edge <from> <to> [label] [--arrow none|start|end|both] [--color <name>]
herdr-canvas unedge <id>                 # remove an edge, keep its boxes
herdr-canvas text <x> <y> <text> [--color <name>]
herdr-canvas draw <x> <y> <ch> [<x> <y> <ch> ...] [--color <name>]
herdr-canvas move <id> <dx> <dy>                 # dx/dy may be negative
herdr-canvas delete <id>
herdr-canvas label <id> <label>
herdr-canvas color <id> <name>                   # default clears color
herdr-canvas fill <id> on|off                    # box only
herdr-canvas skill                       # print this document
herdr-canvas setup                       # install the herdr hotkey binding
```

These are the batch verbs. Each one also works as a single command, for a
one-element tweak:

```
box <x1> <y1> <x2> <y2> [label] [--color <name>] [--fill]
line <x1> <y1> <x2> <y2> [--arrow none|start|end|both] [--color <name>]
text <x> <y> <text> [--color <name>]
draw <x> <y> <ch> [<x> <y> <ch> ...] [--color <name>]
move <id> <dx> <dy>                      # dx/dy may be negative
delete <id>
label <id> <label>
color <id> <name>                        # default clears color
fill <id> on|off                         # box only
```

```sh
herdr-canvas --name demo color b1 red
```

All commands take a diagram name via `--name <name>`; without it they target
the composite name `repo@branch[@worktree]` (see below), which requires
running inside a git repo.

A batch, an element command or `export` fails when the diagram does not exist.
Create it with `new`, or pass `--create` to the command.

## The gate

Every command is parsed, applied, and validated before commit. Exactly two
rejections:

1. **Referential integrity** — a command names an id that does not exist;
   the error echoes the id.
2. **Well-formed commands** — `x2 ≥ x1` and `y2 ≥ y1` for boxes, non-empty
   text, non-negative coordinates, known color names (`red`, `green`, `yellow`,
   `blue`, `magenta`, `cyan`, `white`, `black`, or `default` to clear),
   `fill` only on boxes, and both ends of an `edge` two different boxes.

Everything spatial is allowed: nesting, overlap, crossing lines, dangling
lines. Cell conflicts resolve by z-order; crossing lines render junction
glyphs (`┼`, `├`, `┤`, `┬`, `┴`).

## Edges

An **edge** is a line held by reference to two boxes. Use `edge b1 b3` instead
of computing endpoints. The endpoints come from the two boxes, so the edge
follows them:

- A move of either box re-derives the endpoints.
- A delete of either box deletes the edge, in the same undo step.
- The endpoints are never authored. A `move` of the edge itself does nothing.

`edge` is rejected when an id does not exist, when an id is not a box, or when
both ids are the same box. The default arrow is `end`, from the first box to
the second. An optional label is painted near the middle of the longest
straight run of the edge, and is skipped when it does not fit there.

## Character set

Boxes and lines use one Unicode box-drawing set: `┌ ─ ┐ │ └ ┘` for a box,
`─` and `│` for a line run, `┼ ├ ┤ ┬ ┴` for a junction, and `► ◄ ▲ ▼` for an
arrowhead. A freeform cell holds any character you give it.

An edge uses the double-line set: `═ ║` for a run and `╔ ╗ ╚ ╝` for a bend, so
a line that sticks to two boxes reads differently from a line that floats. A
junction keeps the weight of each arm: `╬ ╠ ╣ ╦ ╩` where two edges meet, and
`╪ ╫ ╞ ╡ ╤ ╧ ╟ ╢ ╥ ╨` where a plain line meets an edge. Arrowheads do not
change.

## Line routing

A line is orthogonal. The line runs on the y axis from (x1,y1) to y2, then on
the x axis to x2. The bend gets a corner glyph (`└`, `┘`, `┌`, `┐`). A line
with x1 = x2 or y1 = y2 stays one straight run and has no bend. An arrow
places an arrowhead (`►`, `◄`, `▲`, `▼`) on the end that `--arrow` names.

An edge routes differently, because it must arrive at a box. The larger gap
between the boxes decides which pair of borders carries the edge. The
attachment then follows three rules, in order:

1. The two spans across the edge overlap: the edge attaches at the middle of
   that overlap and is one straight run with no bend.
2. The spans do not overlap, and the gap holds a free lane: the edge attaches
   at the middle of each box, runs straight out, crosses in the free lane, and
   turns back to the first axis for the last run. Two bends.
3. Neither holds, so the boxes are diagonally adjacent: the edge attaches at
   the two facing corners and turns through the one free cell between them.

The last run always points into the target box, so an edge never travels along
a border of either box. The `vertical` field records which pair of borders the
edge attaches to.

## JSON schema

```json
{
  "name": "demo",
  "next": 4,
  "elements": [
    { "id": "b1", "type": "box", "x1": 0, "y1": 0, "x2": 3, "y2": 2, "label": "hi", "color": "red", "fill": true },
    { "id": "l2", "type": "line", "x1": 0, "y1": 0, "x2": 5, "y2": 5, "arrow": "end" },
    { "id": "l5", "type": "line", "x1": 2, "y1": 2, "x2": 9, "y2": 8, "arrow": "end", "label": "retry", "from": "b1", "to": "b6", "vertical": true },
    { "id": "t3", "type": "text", "x": 1, "y": 1, "text": "hello" },
    { "id": "f4", "type": "freeform", "cells": [ { "x": 0, "y": 0, "ch": "#" } ] }
  ]
}
```

## Export format

`export` prints the picture, then a blank line, then one legend line per
element. Read the legend for ids, colors, fill, and text. Color names:
`red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, `black`.
Omit `color` when default. `fill` appears only on filled boxes. `arrow`
appears on a line or an edge that carries an arrowhead. An edge prints the two
box ids in place of the coordinates.

```
┌──┐
│hi│
└──┘

b1 box 0,0-3,2 red fill "hi"
l2 line 3,1-10,1 blue arrow end
l5 edge b1->b6 arrow end "retry"
t3 text 1,1 "hello"
f4 draw 12
```

## The canvas can send you the diagram

A person drawing in the canvas presses `s`. herdr writes two commands into
your input: `herdr-canvas --name <name> export` to print what they see, and
`herdr-canvas skill` to print this document. Skip `skill` if you already ran
it in this session. Run `export` every time — the picture may have changed.

After `export`, say in one or two sentences what you see. Ask whether they
want to draw or change anything, unless they already said what to do. They
submit the message. Nothing runs until they do.

Make the whole change in one `batch`. The person sees one update instead of
one per element. Read the diagram again with `export` before the next change,
because the person keeps drawing while you work.

## Installation side effect

`herdr-canvas setup` runs as a `[[build]]` step of the herdr plugin. It appends
a `prefix+d` keybinding to the shared `~/.config/herdr/config.toml`. It writes
the binding once and never corrects it afterwards. Nothing removes the binding
when the plugin is uninstalled — delete the block by hand. A locally linked
plugin (`herdr plugin link`) runs the same build step, so linking this repo also
writes to that shared config.

## Composite name

`repo@branch[@worktree]`:
- repo = origin remote basename, else the cwd basename;
- branch = current branch with `/` slugged to `-`; a short SHA on detached
  HEAD; omitted when the repo has no commits;
- worktree = worktree-root basename, present only in a linked worktree.

Outside a git repo there is no auto-name — use `--name`, or the TUI shows a
picker.
