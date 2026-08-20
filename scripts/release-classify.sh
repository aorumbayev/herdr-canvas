#!/bin/sh
# Classify a GitHub Release from `gh release view --json isDraft,tagName,assets`.
# Usage:
#   release-classify.sh --missing
#   release-classify.sh [file.json]
#   release-classify.sh --assert-draft-complete [file.json]
# Empty stdin without a file is missing.
set -eu

mode=classify
if [ "${1:-}" = "--missing" ]; then
	echo missing
	exit 0
fi
if [ "${1:-}" = "--assert-draft-complete" ]; then
	mode=assert
	shift
fi

json_file=""
if [ "${1:-}" != "" ]; then
	json_file=$1
fi

python3 - "$mode" "$json_file" <<'PY'
import json, sys

mode = sys.argv[1]
path = sys.argv[2]
required = (
    "herdr-canvas_Darwin_x86_64.tar.gz",
    "herdr-canvas_Darwin_arm64.tar.gz",
    "herdr-canvas_Linux_x86_64.tar.gz",
    "herdr-canvas_Linux_arm64.tar.gz",
    "checksums.txt",
)

if path:
    raw = open(path, encoding="utf-8").read().strip()
else:
    raw = sys.stdin.read().strip()

if not raw:
    if mode == "assert":
        sys.stderr.write("release-classify: missing release; not a complete draft\n")
        sys.exit(1)
    print("missing")
    sys.exit(0)

data = json.loads(raw)
names = {asset.get("name", "") for asset in data.get("assets") or []}
complete = all(name in names for name in required)
is_draft = bool(data.get("isDraft"))

if mode == "assert":
    if not is_draft:
        sys.stderr.write("release-classify: release is not a draft; refusing to treat it as upload target\n")
        sys.exit(1)
    missing = [name for name in required if name not in names]
    if missing:
        sys.stderr.write("release-classify: draft missing assets: %s\n" % ", ".join(missing))
        sys.exit(1)
    sys.exit(0)

if is_draft:
    print("draft")
elif complete:
    print("published_complete")
else:
    print("published_incomplete")
PY
