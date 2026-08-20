<h3 align="center">
  herdr-canvas
</h3>

<p align="center">Draw ASCII diagrams in your terminal. Let your agent read and edit them.</p>

<p align="center">
  <a href="#install">Install</a> · <a href="#draw-your-first-diagram">First diagram</a> · <a href="#let-your-agent-edit-the-diagram">Agents</a>
</p>

```
┌───────────────┐         ┌───────────────┐         ┌───────────────┐
│               │         │               │         │               │
│   you draw    │◄───────►│    one JSON   │◄───────►│  agent edits  │
│               │         │               │         │               │
└───────────────┘         └───────────────┘         └───────────────┘
```

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

You need [herdr](https://herdr.dev) 0.8.2 or later and Go 1.26 or later. Linux
and macOS. On Windows, install herdr and this plugin inside WSL2.

```bash
herdr plugin install aorumbayev/herdr-canvas
```

herdr compiles the binary from source on your machine. This project publishes
no binaries. The install runs `go build`, so the Go toolchain must be on your
`PATH`.

> [!IMPORTANT]
> The install writes one hotkey binding into your shared
> `~/.config/herdr/config.toml`. herdr uses `prefix+c` for a new tab, so this
> plugin uses `prefix+d`. The install writes the binding once. The uninstall
> does not remove it.

## Draw your first diagram

Press `prefix+d`. herdr opens the canvas in a split beside the active pane.
Press `prefix+d` again to close it. When you are inside a git repository, the
canvas opens the diagram named for that repository and branch, for example
`herdr-canvas@main`. When you are not, a picker lists your diagrams and offers
to create one.

Pick a tool with one key, then drag with the left mouse button:

| Key | Tool             |
| --- | ---------------- |
| `b` | Box              |
| `l` | Line             |
| `a` | Arrow            |
| `t` | Text             |
| `d` | Draw (freeform)  |
| `m` | Move             |
| `x` | Delete           |
| `s` | Send to agent    |
| `q` | Quit             |

A line and an arrow run on one axis, then on the other axis. The bend gets a
corner glyph. An arrow gets one arrowhead at its end. Boxes and lines use one
Unicode box-drawing set (`┌ ─ ┐ │ └ ┘ ┼ ►`), so an edge and a junction join
without a change of style.

The arrow keys move a cursor, and space or enter anchors and commits, so you
can draw the same shapes without a mouse. Every change saves at once, so the
canvas has no save key.

## Send the diagram to the agent beside you

Press `s`. The canvas sends the agent the diagram as text, the list of its
elements, and the commands that change it. The agent can answer about the
picture and edit it without any more context from you.

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

You keep drawing while the agent works. The canvas reloads the file when the
agent changes it, so you both look at the same diagram.

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

Element commands read the diagram for the current repository. Pass `--name` to
choose another one, and `--create` to create a diagram that does not exist yet.

## Releases

Releases follow [Conventional Commits](https://www.conventionalcommits.org).
A `feat` commit raises the minor version and a `fix` commit raises the patch
version. The major version stays at 0, so a breaking change also raises the
minor version. A maintainer starts a release by hand from the `release`
workflow.

## License

MIT
