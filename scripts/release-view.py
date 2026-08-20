#!/usr/bin/env python3
"""Fetch one GitHub Release as {isDraft, tagName, assets:[{name}]}."""

from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request


def die(msg: str, code: int = 1) -> None:
    sys.stderr.write("release-view: %s\n" % msg)
    raise SystemExit(code)


def main(argv: list[str]) -> None:
    if len(argv) != 2:
        die("usage: release-view.py TAG")
    tag = argv[1]
    repo = os.environ.get("GITHUB_REPOSITORY", "").strip()
    if not repo:
        die("GITHUB_REPOSITORY is required")
    token = (os.environ.get("GITHUB_TOKEN") or os.environ.get("GH_TOKEN") or "").strip()

    url = "https://api.github.com/repos/%s/releases/tags/%s" % (repo, tag)
    headers = {
        "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
        "User-Agent": "herdr-canvas-release",
    }
    if token:
        headers["Authorization"] = "Bearer %s" % token
    req = urllib.request.Request(url, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            data = json.load(resp)
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", "replace")
        if exc.code == 404:
            die("HTTP 404: Not Found (%s)" % url)
        die("HTTP %s fetching %s\n%s" % (exc.code, url, body[:500]))
    except urllib.error.URLError as exc:
        die("network error fetching %s: %s" % (url, exc.reason))

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
