#!/bin/sh
# Decode GitHub Actions toJSON(changelog) from CHANGELOG_JSON and write DEST.
# Empty, missing, or JSON null/"" leaves DEST unwritten so goreleaser keeps notes.
set -eu

if [ "${1:-}" = "" ]; then
	echo "write-release-notes: dest path required" >&2
	exit 1
fi
dest=$1

python3 -c '
import json, os, pathlib, sys

raw = os.environ.get("CHANGELOG_JSON", "")
if not raw.strip():
    raise SystemExit(0)
notes = json.loads(raw)
if not notes:
    raise SystemExit(0)
text = notes if isinstance(notes, str) else json.dumps(notes)
if not text.endswith("\n"):
    text += "\n"
pathlib.Path(sys.argv[1]).write_text(text, encoding="utf-8")
' "$dest"
