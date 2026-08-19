#!/bin/sh
# Write a release version into the version field of herdr-plugin.toml.
# herdr reads that field from the checked-out source, so the field must hold
# the released version. Usage: scripts/stamp-version.sh 0.2.0 [path]
set -eu

version=$1
target=${2:-herdr-plugin.toml}

case "$version" in
	[0-9]*.[0-9]*.[0-9]*) ;;
	*)
		echo "stamp-version: expected x.y.z, got '$version'" >&2
		exit 1
		;;
esac

if ! grep -q '^version = "' "$target"; then
	echo "stamp-version: $target has no version field" >&2
	exit 1
fi

tmp=$(mktemp)
sed "s|^version = \".*\"|version = \"$version\"|" "$target" >"$tmp"
mv "$tmp" "$target"
echo "stamp-version: $target -> $version"
