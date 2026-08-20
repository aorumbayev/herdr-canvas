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

    repo_url = "https://api.github.com/repos/%s" % repo
    tag_url = "%s/releases/tags/%s" % (repo_url, tag)
    list_url = "%s/releases?per_page=30" % repo_url
    headers = {
        "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
        "User-Agent": "herdr-canvas-release",
    }
    if token:
        headers["Authorization"] = "Bearer %s" % token

    def get(url: str):
        req = urllib.request.Request(url, headers=headers)
        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                return json.load(resp)
        except urllib.error.HTTPError as exc:
            body = exc.read().decode("utf-8", "replace")
            if exc.code == 404:
                return None
            die("HTTP %s fetching %s\n%s" % (exc.code, url, body[:500]))
        except urllib.error.URLError as exc:
            die("network error fetching %s: %s" % (url, exc.reason))

    data = get(tag_url)
    if data is None:
        releases = get(list_url) or []
        if not isinstance(releases, list):
            die("unexpected GitHub releases list JSON")
        data = next((item for item in releases if item.get("tag_name") == tag), None)
    if data is None:
        die("HTTP 404: Not Found (%s)" % tag_url)

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
