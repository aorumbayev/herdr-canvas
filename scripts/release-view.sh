#!/bin/sh
# Fetch one GitHub Release as {isDraft, tagName, assets:[{name}]}.
# Usage: release-view.sh TAG
set -eu
dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
exec python3 "$dir/release-view.py" "$@"
