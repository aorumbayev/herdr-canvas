#!/bin/sh
# Fetch one GitHub Release as {isDraft, tagName, assets:[{name}]}.
# Usage: release-view.sh TAG
set -eu
export GH_PAGER=cat
export GH_PROMPT_DISABLED=1
export NO_COLOR=1
dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
exec python3 "$dir/release-view.py" "$@"
