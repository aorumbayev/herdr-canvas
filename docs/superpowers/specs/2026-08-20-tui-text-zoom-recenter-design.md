# TUI text commit, zoom steps, and recenter

Date: 2026-08-20

Extends `docs/superpowers/specs/2026-08-20-tui-chrome-pan-undo-design.md`. Where this file and that file disagree, this file wins.

Reference: [lewish/asciiflow](https://github.com/lewish/asciiflow) `main` (`client/controller.ts`, `client/store/canvas.ts`, `client/toolbar.tsx`, `client/draw/text.ts`).

## Goal

Make pan, text, zoom, and chrome match the live pass against asciiflow, within what a terminal can draw.

## Non-goals

- Continuous zoom (no 1.1× per wheel tick)
- Zoom below 0.5× or above 2×
- Space+drag or alt+left pan
- A second “view” row
- Change the store, CLI flags, or the agent prompt
- Persist undo

## Header

Left: diagram name (truncate if needed).

Right: zoom chip, then a **recenter** chip, then cursor `(x,y)`.

```
 herdr-canvas@cbdd616          1x  recenter  (48,12)
```

| Chip | Click |
| --- | --- |
| Zoom (`0.5x` / `1x` / `1.5x` / `2x`) | Set zoom to 1×. Do not pan. (asciiflow `reset`) |
| Recenter | Keep zoom. Pan so the bounding box of the elements sits in the **middle** of the pane. |

Recenter math: origin = bbox min − (pane cells − bbox size) / 2, then clamp origin to ≥ 0. If the bbox is larger than the pane, origin is the bbox minimum. Empty diagram: origin `(0,0)`, zoom unchanged.

## Footer

Same chips as today: tools, undo/redo, send. Live size badge stays on the right.

The **tool/undo/send group is centered on the row**, not left-aligned. Left pad is `(width - groupWidth) / 2`. Chip `x0`/`x1` include that pad so hit-test matches the mouse cell. The live size badge sits to the right of the group when there is at least one space of gap; if it would overlap the group, drop the badge.

Narrow pane: same drop order as the parent spec (send, then undo/redo, then short labels). Center whatever remains.

## Zoom

View-only. JSON coordinates do not change. `canvas.Window` and export stay 1×.

Allowed values: **0.5, 1, 1.5, 2**. Any other value is ignored. Store zoom as tenths (`5`, `10`, `15`, `20`) so the code does not use floats for the step list.

| Zoom | Terminal cells per diagram cell | Paint |
| --- | --- | --- |
| 0.5× | 1 terminal cell covers 2×2 diagram cells | One glyph per 2×2 block. If any cell is non-space, use the first non-space in row-major order. Else space. |
| 1× | 1:1 | `Grid.Window` as today |
| 1.5× | 3 terminal cells cover 2 diagram cells (both axes) | For each pair of source cells, emit three columns: glyph, extra column, glyph. Extra column is `─` when both neighbours are `─`, else space. Emit three copies of that pattern for every two source rows (row, extra row, row). Extra row copies the first of the pair. |
| 2× | 2:1 | As today: `─` → `──`; other glyphs glyph+space; each source row twice |

`canvasPoint` inverts that mapping (header and footer still excluded). A point that lands on an extra 1.5× gap column or row maps to the diagram cell of the preceding source cell.

Ctrl+wheel and `+`/`=` / `-` **step** the list: zoom-in 0.5→1→1.5→2 and stop; zoom-out 2→1.5→1→0.5 and stop. Do not wrap. Do not toggle per tick.

Plain wheel still pans. Shift+wheel pans on x. Header/footer wheel and middle-drag do nothing.

## Pan

Middle-press on the canvas starts pan. While pan is live, **any motion** pans, even if the event no longer names the middle button. Release of the middle button ends pan.

Origin stays ≥ 0. Pan does not `Apply` and does not push history.

## Text

Click-outside and double-click extend the parent spec. Enter, esc, tool chip, and in-flight undo stay as in that spec.

**Click outside while typing.** Left-click on a canvas cell that is **not** covered by the in-progress buffer (origin + length of the current string, current row only) commits like Enter. Empty buffer: leave typing, write nothing. That click does not start a new string. A later click on empty canvas starts a new string (today’s text-tool click).

**Double-click existing text.** Text tool. Two left clicks on the same canvas cell within 400 ms, or a mouse event with click count 2 if the terminal sends it. The cell must sit on a `Text` element.

1. Commit in-progress typing first (same as click-outside).
2. Load that element’s string into the buffer at its origin. Keep the same id. Hide the committed element in the overlay while typing so the buffer is what the person sees.
3. Enter or click-outside writes the new string onto **that id**. Empty commit deletes the element.

New canvas command, no new JSON fields:

```go
type TextSetCmd struct {
    ID   string
    Text string
}
```

`Apply` finds the id, requires `Type == Text`, requires non-empty `Text`, writes `Text` (and does not move `X`,`Y`). Empty rewrite is `DeleteCmd`, not `TextSetCmd`.

Do not add a CLI flag for this command in this change. The TUI calls `Apply`.

Double-click on the in-progress buffer does not commit and re-open. Stay typing.

## Files

- `internal/tui/viewport.go` — zoom tenths, `canvasPoint` and paint for 0.5 / 1.5, `recenter` (center bbox; replace today’s fit-to-min as the recenter action)
- `internal/tui/chrome.go` — zoom + recenter header chips; footer group centered
- `internal/tui/tui.go` — pan motion without middle flag; click-outside commit; double-click edit; zoom step; chromeClick split reset vs recenter
- `internal/canvas` — `TextSetCmd` in commands + `Apply` + tests
- Matching `*_test.go`

Store, CLI, and the agent prompt stay put.

## Tests

Keep existing tests. Update ones that assume zoom is only 1 or 2, that the zoom chip recenters, or that the footer starts at x=0.

Add:

- Middle-drag pans when motion events have button none / left
- Click-outside while typing commits; empty click-outside does not write; that click does not start a new string
- Double-click a text element loads it; commit keeps the same id
- Zoom steps 0.5 → 1 → 1.5 → 2 and back; `canvasPoint` at each step
- Recenter: small box in the middle of the pane; box larger than the pane uses bbox min
- Footer: chip x-range includes center pad; click on the visible label still hits
- `TextSetCmd` rewrites text; rejects empty text and non-text ids

After unit tests pass: one herdr split pass — middle-drag, click-outside text, double-click edit, zoom steps, recenter, centered footer.
