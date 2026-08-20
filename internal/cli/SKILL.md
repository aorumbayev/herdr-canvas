# herdr-canvas — agent skill

herdr-canvas is a dead-simple ASCII diagram canvas. You read a diagram with
`export` and edit it with element commands. You never write raw grid
characters and never rewrite the whole JSON file.

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

## Commands

All element commands take a diagram name via `--name <name>`; without it they
target the composite name `repo@branch[@worktree]` (see below), which requires
running inside a git repo.

```
herdr-canvas new <name>                  # create an empty diagram
herdr-canvas open <name>                 # open an existing diagram in the TUI
herdr-canvas list                        # list diagrams
herdr-canvas export [--name <name>]      # print the grid as text
herdr-canvas box <x1> <y1> <x2> <y2> [label]
herdr-canvas line <x1> <y1> <x2> <y2> [--arrow none|start|end|both]
herdr-canvas text <x> <y> <text>
herdr-canvas draw <x> <y> <ch> [<x> <y> <ch> ...]   # freeform cells
herdr-canvas move <id> <dx> <dy>                 # dx/dy may be negative
herdr-canvas delete <id>
herdr-canvas label <id> <label>
herdr-canvas skill                       # print this document
herdr-canvas setup                       # install the herdr hotkey binding
```

An element command or `export` fails when the diagram does not exist. Create it
with `new`, or pass `--create` to the command.

## The gate

Every command is parsed, applied, and validated before commit. Exactly two
rejections:

1. **Referential integrity** — a command names an id that does not exist;
   the error echoes the id.
2. **Well-formed commands** — `x2 ≥ x1` and `y2 ≥ y1` for boxes, non-empty
   text, non-negative coordinates.

Everything spatial is allowed: nesting, overlap, crossing lines, dangling
lines. Cell conflicts resolve by z-order; crossing lines render junction
glyphs (`┼`, `├`, `┤`, `┬`, `┴`).

## Character set

Boxes and lines use one Unicode box-drawing set: `┌ ─ ┐ │ └ ┘` for a box,
`─` and `│` for a line run, `┼ ├ ┤ ┬ ┴` for a junction, and `► ◄ ▲ ▼` for an
arrowhead. A freeform cell holds any character you give it.

## Line routing

A line is orthogonal. The line runs on the y axis from (x1,y1) to y2, then on
the x axis to x2. The bend gets a corner glyph (`└`, `┘`, `┌`, `┐`). A line
with x1 = x2 or y1 = y2 stays one straight run and has no bend. An arrow
places an arrowhead (`►`, `◄`, `▲`, `▼`) on the end that `--arrow` names.

## JSON schema

```json
{
  "name": "demo",
  "next": 4,
  "elements": [
    { "id": "b1", "type": "box", "x1": 0, "y1": 0, "x2": 3, "y2": 2, "label": "hi" },
    { "id": "l2", "type": "line", "x1": 0, "y1": 0, "x2": 5, "y2": 5, "arrow": "end" },
    { "id": "t3", "type": "text", "x": 1, "y": 1, "text": "hello" },
    { "id": "f4", "type": "freeform", "cells": [ { "x": 0, "y": 0, "ch": "#" } ] }
  ]
}
```

## Export format

`export` renders the bounding rectangle of the grid as space-filled text with
trailing whitespace trimmed, e.g.:

```
┌──┐
│hi│
└──┘
```

## The canvas can send you the diagram

A person drawing in the canvas presses `s`. herdr writes into your input the
diagram as text, the list of its elements, and this command set. The person
then adds their own request and submits it, so the diagram arrives as context
for what they ask next. Treat it as the state of the diagram at the moment they
pressed `s`. Read the diagram again with `export` after every change you make,
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
