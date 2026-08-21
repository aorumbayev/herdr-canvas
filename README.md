<h3 align="center">
  herdr-canvas
</h3>

<p align="center">Draw ASCII diagrams in your terminal. Let your agent read and edit them.</p>

<p align="center">
  <img src="docs/assets/canvas.svg" alt="herdr split: draw on the canvas, send to the agent, the agent adds a box" width="1180" />
</p>

<p align="center">
  <a href="#install">Install</a> · <a href="#draw-your-first-diagram">First diagram</a> · <a href="#let-your-agent-edit-the-diagram">Agents</a>
</p>

---

You sketch a box-and-arrow picture to explain a system. You draw it on a
whiteboard, or in a web app, and then it is not in your repository and your
agent cannot read it. herdr-canvas is [asciiflow](https://asciiflow.com) as a
terminal program: you draw with the mouse, the diagram is one JSON file, and an
agent reads and edits the same file through a small command set.

It is a plugin for [herdr](https://herdr.dev), and it also runs as a standalone
binary.

A diagram holds four kinds of element:

| Element    | What it is                                            |
| ---------- | ----------------------------------------------------- |
| `box`      | A rectangle with an optional label                    |
| `line`     | An orthogonal connector with an optional arrow at either end |
| `text`     | A string at one coordinate                            |
| `draw`     | Freeform cells — the pencil                           |

Every element has a stable id (`b1`, `l2`, `t3`, `f4`). The tool never reuses
an id. The picture you see is a pure function of the elements, so the JSON file
is the only source of truth.

## Install

Linux and macOS. Native Windows is not supported: install herdr and this
plugin inside **WSL2**, and use the matching Linux archive.

[GitHub Releases](https://github.com/aorumbayev/herdr-canvas/releases) publish
four archives plus `checksums.txt`:

| Machine | Archive |
| --- | --- |
| macOS amd64 | `herdr-canvas_Darwin_x86_64.tar.gz` |
| macOS arm64 | `herdr-canvas_Darwin_arm64.tar.gz` |
| Linux amd64 | `herdr-canvas_Linux_x86_64.tar.gz` |
| Linux arm64 | `herdr-canvas_Linux_arm64.tar.gz` |

You do not need a Go toolchain to install or update.

### Herdr plugin

You need [herdr](https://herdr.dev) 0.8.2 or later, git, and either curl or
wget. You do not need Go.

```bash
herdr plugin install aorumbayev/herdr-canvas
```

herdr clones the plugin, downloads the Release archive for the version in
`herdr-plugin.toml` (not `/releases/latest`), checks the checksum, and runs
`setup`. `herdr-canvas --version` prints that version as `0.x.y`.

`v0.1.0` is notes only and has no archives, so a default-ref install stays
broken until the next tagged release is published.

### Standalone archive

Download the archive for your OS and CPU from
[GitHub Releases](https://github.com/aorumbayev/herdr-canvas/releases). Unpack
`herdr-canvas` onto your `PATH`. After the download you do not need git, curl,
or wget. On WSL2, use a `Linux_*` archive.

> [!IMPORTANT]
> The install writes one hotkey binding into your shared
> `~/.config/herdr/config.toml`. herdr uses `prefix+c` for a new tab, so this
> plugin uses `prefix+d`. The install writes the binding once. The uninstall
> does not remove it.

### Updates

`herdr-canvas update` installs a newer published (non-draft) tag. A
herdr-managed install goes through `herdr plugin install`. A standalone binary
replaces itself from Release assets.

A development build (`herdr-canvas --version` prints `dev`, or anything other
than `0.x.y`) refuses update, does not contact GitHub, and does not show a TUI
notice. Install from a GitHub Release or `herdr plugin install` instead.

On a release binary, the canvas can show
`newer 0.2.0 · herdr-canvas update · i dismiss`. Press `i` to hide that tag
until a newer one exists. `i` does not apply the update. While you name a
diagram or type on the canvas, `i` still inserts.

## Draw your first diagram

Press `prefix+d`. herdr opens the canvas in a split beside the active pane.
Press `prefix+d` again to close it. When you are inside a git repository, the
canvas opens the diagram named for that repository and branch, for example
`herdr-canvas@main`. When you are not, a picker lists your diagrams and offers
to create one.

Pick a tool with a number key (or the footer chips), then drag with the left
mouse button. Select is the default when a canvas opens.

| Key | Tool             |
| --- | ---------------- |
| `1` | Select           |
| `2` | Box              |
| `3` | Line             |
| `4` | Arrow            |
| `5` | Text             |
| `6` | Draw (freeform)  |
| `s` | Send to agent    |
| `o` / `canvases` header control | Diagram picker |
| `?` / `h` / help chip | Help (controls) |
| `esc` | Cancel, then Select, then clear selection |
| `q` | Quit             |

The canvas draws at 1×. The wheel pans (shift pans sideways; ctrl pans the
same way). Middle-drag pans. Recenter fits the drawing in the view.

In Select, click replaces the selection, shift-click toggles membership, and
a drag on empty space marquees. Shift-marquee adds. Drag a selected element
to move the whole selection. Delete and Backspace remove the selection as
one undo step. `x` never deletes and never changes tools.

Press `o` to leave the drawing for the list of your diagrams, and `esc` to
go back to it. The list reads the store each time, so a diagram your agent
made while you drew is in it. In the picker, Delete or Backspace asks
`delete "name"? y/N` before removing a diagram. Canvas deletion is not an
undo step.

A line and an arrow run on one axis, then on the other axis. The bend gets a
corner glyph. An arrow gets one arrowhead at its end. Boxes and lines use one
Unicode box-drawing set (`┌ ─ ┐ │ └ ┘ ┼ ►`), so an edge and a junction join
without a change of style.

The arrow keys move a cursor, and space or enter anchors and commits, so you
can draw the same shapes without a mouse. Every change saves at once, so the
canvas has no save key.

## Send the diagram to the agent beside you

Press `s`. The canvas writes two commands into the agent's input: `export` of
the open diagram, and `skill` (skip `skill` if the agent already ran it in
this session). It does not paste the picture and it does not press enter.
herdr focuses the agent pane, you add what you want done, and you submit it.
Nothing reaches the agent until you do.

One agent in the workspace receives the diagram at once. When the workspace
holds more than one agent, the canvas lists them and you choose:

```
send herdr-canvas@main to which agent?   workspace: herdr-canvas

> control       claude    idle     Editable grid validation and mouse bugs
  review-prose  claude    idle     Herdr-canvas prose review
  ship          claude    working  herdr canvas pull request ship stage

↑/↓ choose · enter send · esc cancel
```

Each row names the tab the way your tab bar names it, then the agent, its
state, and its terminal title.

You keep drawing while the agent works. The canvas looks at the file about
twice a second and reloads it when the agent changes it, so the picture updates
under your eyes. It waits while you hold a drag, type text, or hold a keyboard
anchor, so a reload never moves the ground you draw on.

## Let your agent edit the diagram

Diagrams live in `~/.local/share/herdr-canvas/`, outside your repository. The
agent reads a diagram as text and changes it one element at a time:

```bash
herdr-canvas list
herdr-canvas --name mydiagram export
herdr-canvas --name mydiagram box 2 1 20 6 "web server"
herdr-canvas --name mydiagram line 20 3 34 3 --arrow end
herdr-canvas --name mydiagram delete b1
```

Every command goes through one validation gate, on the terminal and in the
command line alike. The gate rejects a command that names an element that does
not exist, and a command with impossible geometry. It reports the id and the
rule that failed. It allows everything else: nested boxes, crossing lines, and
lines that connect to nothing.

The tool ships its own agent instructions:

```bash
herdr-canvas skill
```

Or paste this to your agent:

```
Read the herdr-canvas instructions with `herdr-canvas skill` and follow them.
Then run `herdr-canvas list`, open the diagram for this repository with
`export`, and describe what it shows.
```

## Commands

| Command                          | What it is for                                  |
| -------------------------------- | ----------------------------------------------- |
| `herdr-canvas`                    | Open the canvas                                 |
| `herdr-canvas new <name>`         | Create a diagram                                |
| `herdr-canvas open <name>`        | Open a diagram by name                          |
| `herdr-canvas list`               | List every diagram in the store                 |
| `herdr-canvas export`             | Print the diagram as text                       |
| `herdr-canvas box\|line\|text\|draw` | Add an element                               |
| `herdr-canvas move\|delete\|label`  | Change an element                            |
| `herdr-canvas skill`              | Print the agent instructions                    |
| `herdr-canvas setup`              | Install the herdr hotkey                        |
| `herdr-canvas update`             | Install a newer published release               |
| `herdr-canvas --version`          | Print the binary version (`0.x.y` or `dev`)     |

Element commands read the diagram for the current repository. Pass `--name` to
choose another one, and `--create` to create a diagram that does not exist yet.

## Releases

Releases follow [Conventional Commits](https://www.conventionalcommits.org).
A `feat` commit raises the minor version and a `fix` commit raises the patch
version. The major version stays at 0, so a breaking change also raises the
minor version.

A maintainer starts a release by hand from the `release` workflow
(`workflow_dispatch`). `v0.1.0` stays notes-only; binaries begin at the next
tagged version. A failed run may resume only an in-flight **draft** of that
new version. A published release is never edited.

## License

MIT
