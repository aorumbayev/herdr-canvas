#!/usr/bin/env python3
"""Fetch one GitHub Release as {isDraft, tagName, assets:[{name}]}."""

from __future__ import annotations

import json
import os
import subprocess
import sys


def die(msg: str, code: int = 1) -> None:
    sys.stderr.write("release-view: %s\n" % msg)
    raise SystemExit(code)


def is_not_found(data: object) -> bool:
    return isinstance(data, dict) and (
        str(data.get("status")) == "404" or data.get("message") == "Not Found"
    )


def main(argv: list[str]) -> None:
    if len(argv) != 2:
        die("usage: release-view.py TAG")
    tag = argv[1]
    repo = os.environ.get("GITHUB_REPOSITORY", "").strip()
    if not repo:
        die("GITHUB_REPOSITORY is required")

    env = os.environ.copy()
    env["GH_PAGER"] = "cat"
    env["GH_PROMPT_DISABLED"] = "1"
    env["NO_COLOR"] = "1"

    proc = subprocess.run(
        ["gh", "api", "repos/%s/releases/tags/%s" % (repo, tag)],
        capture_output=True,
        text=True,
        env=env,
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
            die("gh api returned non-JSON (exit %s): %s" % (proc.returncode, raw[:200]))

    if is_not_found(data):
        die("HTTP 404: Not Found (repos/%s/releases/tags/%s)" % (repo, tag))

    if proc.returncode != 0:
        raise SystemExit(proc.returncode)

    if not raw:
        die("gh api exited %s with empty stdout" % proc.returncode)

    if not isinstance(data, dict) or "draft" not in data or "tag_name" not in data:
        die("unexpected GitHub API JSON")

    mapped = {
        "isDraft": bool(data.get("draft")),
        "tagName": data["tag_name"],
        "assets": [{"name": a.get("name", "")} for a in (data.get("assets") or [])],
    }
    sys.stdout.write(json.dumps(mapped) + "\n")


if __name__ == "__main__":
    main(sys.argv)
