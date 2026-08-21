#!/usr/bin/env sh
set -eu

pattern='^[[:space:]]*(co-authored-by:|generated with|assisted-by:|created-by:).*(cursor|claude|copilot|codex|agent|ai)'

if grep -qiE "$pattern"; then
	echo "commit message contains an AI attribution trailer" >&2
	exit 1
fi
