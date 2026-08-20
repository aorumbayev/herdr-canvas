# TUI chrome, pan, zoom, and undo

Date: 2026-08-20

Reference: [lewish/asciiflow](https://github.com/lewish/asciiflow) `main` (`client/controller.ts`, `client/store/canvas.ts`, `client/toolbar.tsx`, `client/draw/text.ts`). Behaviour below is from that source, not from memory.

## Goal

Give the editor a mouse-first chrome like asciiflow, an unbounded view over the same grid, and undo/redo. Hotkeys stay. The diagram file and `canvas.Apply` do not change.

## Non-goals

- ASCII fallback for terminals that cannot draw box-drawing characters
- A `select` tool (move and delete stay separate)
- Persist undo in the JSON file
- Continuous zoom below one character per cell
- Space+drag pan (removed in current asciiflow; space still places or commits)
- Reconcile `setup` keybindings, clean stale store names, or change the agent prompt

## Layout (option A)

One header row. Canvas. One footer row of chips.

```
 herdr-canvas@cbdd616                    1x  (48,12)
┌──────────────────────────────────────────────────┐
│ … diagram …                                      │
└──────────────────────────────────────────────────┘
 [box] line arrow text draw move del │ [undo] [redo] │ send
```

The active tool is the inverted chip. During a box, line, or arrow drag the same footer row appends the live size badge on the right (`13x5`), as today.

## Viewport

The grid is already unbounded. `origin` chooses which cells the pane shows. Pan writes `origin`. `origin` stays ≥ 0. The person may pan into empty space to the right and down and draw there.

| Input | Result |
| --- | --- |
| Middle-press and drag on the canvas | Pan |
| Wheel on the canvas | Pan one cell per tick |
| Shift+wheel on the canvas | Pan on x |
| Ctrl+wheel on the canvas | Zoom in sets 2×; zoom out sets 1×. Do not pan. Do not toggle per tick (trackpad pinch repeats). |
| `+` / `-` | 2× / 1× |

Middle-drag and wheel on the header or footer do nothing. Space with no mouse still places or commits. Space does not pan.

Zoom is view-only. JSON coordinates do not change. 1× is one terminal cell per diagram cell. 2× walks the same grid and emits two columns and two rows per cell: `─` becomes `──`; other glyphs are glyph+space; each logical row is emitted twice. Zoom is only 1 or 2; any other value is ignored.

Left-click the header zoom chip: set 1× and pan so the bounding box of the elements sits in the pane. If the box is larger than the pane, origin is the box minimum. An empty diagram leaves origin `(0,0)` and zoom 1×. This combines asciiflow's `reset` and `recenter` into one control. Recenter uses the bounding box, not the cell centroid, because the herdr split is small.

`canvas.Window` stays 1× for tests and export. 2× paint lives in the TUI viewport.

`canvasPoint` maps a terminal cell to a diagram cell. It accounts for the header row, the footer row, `origin`, and zoom. A left-drag that starts on a canvas row still hits the canvas.

## Chrome

Each chip has a label, an x-range on its row, and an action: tool, undo, redo, send, or zoom.

A left-press on a chip runs that action. A left-press on padding is a miss. A left-press on a dim undo/redo chip is a no-op. Chrome hit-test is left button only.

`bubbles/key` owns `ctrl+z`, `ctrl+shift+z`, `ctrl+y`, `+`, `-`. Tool keys `b l a t d m x` stay. `s` and the send chip both open the agent picker. `esc` and `q` stay. While typing, those tool keys still insert text; only enter, esc, and a tool chip leave typing.

If the pane is too narrow for the full footer, chips drop from the right: send first, then undo/redo, tools last. When even the tools do not fit, labels shorten to `b l a t d m x · ↶ ↷ · s`.

## History

A ring of at most 50 deep copies of the diagram, in memory only.

`push` clones the diagram, appends to undo, and clears redo. Skip `push` when the clone equals the current diagram.

`undo` / `redo` restore a copy and save. They do not go through `Apply`, so ids and `next` come back exactly. Empty stack: no write, no status line. Dim the matching chip when that side is empty.

Call `push` only after `Apply` succeeds, and once before a reload from disk. Pan, zoom, and chrome misses do not push and do not save.

A save failure after undo/redo keeps the restored diagram in memory and sets the status to the I/O error. The next poll does not treat that as an agent edit unless mtime moved.

A corrupt file on poll keeps the current diagram, sets the status to the load error, and does not push.

## Gestures (asciiflow)

**In-flight undo does not block.** If a drag, keyboard anchor, or draw buffer is open, `ctrl+z` / the undo chip discards that preview, does not call `Apply`, and does not undo the last committed command (asciiflow #332). Text is the exception: the first undo commits the in-progress text through `Apply` (their `cleanup()` → `commitScratch()`), which pushes history; a second undo restores the pre-text snapshot.

**Tool chip during text commits**, not cancels. That matches `setToolMode` → `currentTool.cleanup()`. A tool chip during a box/line/draw preview discards the preview and switches tool.

Mouse route in the editor: left-press on the header or footer is chrome; middle-drag or wheel on the canvas is pan/zoom; otherwise the current tool runs.

The 400 ms poll still skips while busy. When the file changed, the editor pushes the in-memory diagram, replaces it with the file, and does not save. Undo then brings back the pre-reload diagram and writes it.

If the terminal has no mouse, chips and pan do nothing. Keys still work.

## Packages

Charm moves to:

- `charm.land/bubbletea/v2 v2.0.9`
- `charm.land/lipgloss/v2 v2.0.6`
- `charm.land/bubbles/v2 v2.1.1` (`help` is unused; `key` only)

Cobra stays at `github.com/spf13/cobra v1.10.2`.

There is no Charm button widget. Chips are Lip Gloss boxes plus hit-test on the mouse cell.

## Files

- `internal/tui/viewport.go` — origin, zoom, `canvasPoint`, pan, fit, 2× paint
- `internal/tui/chrome.go` — header, footer, chip bounds, hit-test
- `internal/tui/history.go` — clone, push, undo, redo, cap 50
- `internal/tui/tui.go` — Bubble Tea model; routes keys and mouse: chrome hit, then pan, then the current tool

`canvas.Apply`, the store, the CLI, and the agent prompt stay put. No new JSON fields.

## Tests

Keep the existing TUI tests. Mouse-left still draws. Wheel, right, and middle still must not draw a box. After this change, middle-press plus motion must move `origin` and leave the diagram unchanged.

Add table-driven tests for:

- Viewport: `canvasPoint` at 1× and 2×; pan clamps `origin` to ≥ 0; fit on empty; fit on a box off-screen
- Chrome: click `[arrow]` sets the arrow tool; padding is a miss; dim undo does not write; narrow width drops send, then undo/redo, then shortens labels
- History: `Apply` box then undo restores elements and `next`; redo restores the same id; failed `Apply` does not push; reload from disk is one undo step; in-flight text then undo commits, second undo removes that text
- Wheel: plain wheel on the canvas pans; shift+wheel pans on x; ctrl+wheel zoom-in sets 2× and zoom-out sets 1×; wheel on the footer does not pan
- Charm v2: the program still starts with mouse cell motion; `View()` returns a `tea.View`. Existing tests that read the view as a string use that view's content.

Do not add tests in canvas, store, CLI, or the agent prompt for this change.

After unit tests pass, one pass in a herdr split: middle-drag, wheel pan, chip click, undo/redo, send.
