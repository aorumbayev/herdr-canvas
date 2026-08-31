# AGENTS.md

herdr-canvas: an ASCII diagram tool. One Go module, one binary (`cmd/herdr-canvas`), doubling as a herdr plugin (`herdr-plugin.toml`).

## Commands

```sh
make setup            # installs git hooks (.githooks) — do this once
go build -o bin/herdr-canvas ./cmd/herdr-canvas
go test ./...
go test ./internal/tui -run TestName   # single test
```

Verify before pushing — CI (`.github/workflows/verify.yml`) runs on linux and macos, in this order:
`gofmt -l .` (must be empty) → `go vet ./...` → `go test ./...` → shell helper tests `sh scripts/fetch-release_test.sh` and `sh scripts/release-classify_test.sh` → `go build ./cmd/herdr-canvas`. On linux only, CI also checks every commit message and cross-compiles the four release targets (darwin/linux × amd64/arm64).

## Commits

- Conventional Commits; `feat` bumps minor, `fix` bumps patch, major stays 0. Releases run via `workflow_dispatch` on the `release` workflow.
- The commit-msg hook and CI **reject AI attribution trailers** (`Co-authored-by: Claude`, `Generated with ...`, etc. — see `scripts/check-commit-message.sh`). Never add them.

## Architecture

- Elements (box/line/text/draw) in a JSON file per diagram are the only source of truth; the grid is always re-rendered, never stored. Element ids (`b1`, `l2`, `t3`, `f4`) only increase and are never reused after deletion — preserve this invariant.
- `internal/canvas` — model, apply, render. `internal/store` — JSON files under `$XDG_DATA_HOME/herdr-canvas/`. `internal/tui` — the interactive canvas. `internal/cli` — cobra commands. `internal/herdr` — herdr plugin integration. `internal/update` — self-update from GitHub releases. `internal/name` — default diagram name from the git repository. `internal/version` — the version stamped into release builds. `internal/welcome` — first-run walkthrough state.
- TUI uses Bubble Tea v2 with **`charm.land/...` import paths** (`charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`), not `github.com/charmbracelet/...`.
- `internal/cli/SKILL.md` is embedded into the binary (`go:embed`) and printed by `herdr-canvas skill`; it is the agent-facing contract for the CLI. Update it whenever CLI commands, flags, or behavior change.
- Plugin quirk: the herdr pane runs with a foreign cwd, so the binary must be addressed via `$HERDR_PLUGIN_ROOT` (see `herdr-plugin.toml`); the plugin build fetches a prebuilt release via `scripts/fetch-release.sh` rather than compiling.

## Style

- README.md and SKILL.md are written in Simplified Technical English: short declarative sentences, plain words, one instruction per sentence. Match that register when editing docs.
- Release builds are `CGO_ENABLED=0` for darwin/linux × amd64/arm64 — keep the code pure Go, no cgo.
