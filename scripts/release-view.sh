#!/bin/sh
# Fetch one GitHub Release as {isDraft, tagName, assets:[{name}]}.
# Usage: release-view.sh TAG
# A missing tag is always a nonzero exit with HTTP 404 on stderr, even when
# `gh api` writes a 404 JSON body and exits 0 (observed on GitHub Actions).
set -eu

if [ "$#" -ne 1 ]; then
	echo "release-view: usage: release-view.sh TAG" >&2
	exit 1
fi

tag=$1
repo=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}

python3 - "$tag" "$repo" <<'PY'
import json, subprocess, sys

tag, repo = sys.argv[1], sys.argv[2]
proc = subprocess.run(
    ["gh", "api", "repos/%s/releases/tags/%s" % (repo, tag)],
    capture_output=True,
    text=True,
)
raw = (proc.stdout or "").strip()
err = (proc.stderr or "").strip()
if err:
    sys.stderr.write(err + "\n")

data = None
if raw:
    try:
        data = json.loads(raw)
    except json.JSONDecodeError:
        sys.stderr.write(raw + "\n")
        sys.exit(proc.returncode or 1)

if isinstance(data, dict) and (
    str(data.get("status")) == "404" or data.get("message") == "Not Found"
):
    sys.stderr.write("HTTP 404: Not Found (repos/%s/releases/tags/%s)\n" % (repo, tag))
    sys.exit(1)

if proc.returncode != 0:
    sys.exit(proc.returncode)

if not isinstance(data, dict) or "draft" not in data or "tag_name" not in data:
    sys.stderr.write("release-view: unexpected GitHub API JSON\n")
    sys.exit(1)

mapped = {
    "isDraft": bool(data.get("draft")),
    "tagName": data["tag_name"],
    "assets": [{"name": a.get("name", "")} for a in (data.get("assets") or [])],
}
sys.stdout.write(json.dumps(mapped) + "\n")
PY
