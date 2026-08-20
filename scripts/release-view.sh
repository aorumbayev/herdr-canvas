#!/bin/sh
# Fetch one GitHub Release as {isDraft, tagName, assets:[{name}]}.
# Usage: release-view.sh TAG
# Missing releases fail with HTTP 404 on stderr (do not use `gh release view --json`;
# on Actions that path can exit 0 with empty stdout for a missing tag).
set -eu

if [ "$#" -ne 1 ]; then
	echo "release-view: usage: release-view.sh TAG" >&2
	exit 1
fi

tag=$1
repo=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}

gh api "repos/${repo}/releases/tags/${tag}" \
	--jq '{isDraft: .draft, tagName: .tag_name, assets: [(.assets // [])[] | {name}]}'
