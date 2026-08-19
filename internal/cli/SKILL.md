# herdr-canvas — agent skill

herdr-canvas is a dead-simple ASCII diagram canvas. You read a diagram with
`export` and edit it with element commands. You never write raw grid
characters and never rewrite the whole JSON file.

## Model

- **Diagram** — one JSON file `<name>.json` in the central store.
- **Element** — a box, line, text, or freeform cell group, each with a
  stable id (`b1`, `l2`, `t3`, `f4`, …). Ids are assigned by the tool,
  monotonically increasing, never reused after deletion.
- **Grid** — the rendered picture. It is derived from the elements (z-order:
  array order, later wins), never stored, never the source of truth.

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
```

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
+--+
|hi|
+--+
```

## Composite name

`repo@branch[@worktree]`:
- repo = origin remote basename, else the cwd basename;
- branch = current branch with `/` slugged to `-`; a short SHA on detached
  HEAD; omitted when the repo has no commits;
- worktree = worktree-root basename, present only in a linked worktree.

Outside a git repo there is no auto-name — use `--name`, or the TUI shows a
picker.
