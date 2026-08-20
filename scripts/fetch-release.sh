#!/bin/sh
# Download the herdr-plugin.toml version's GitHub Release archive for this
# machine, verify SHA-256, and install bin/herdr-canvas. POSIX sh; no go.
set -eu

toml=${HERDR_CANVAS_PLUGIN_TOML:-herdr-plugin.toml}
base=${HERDR_CANVAS_RELEASE_BASE:-https://github.com/aorumbayev/herdr-canvas/releases/download}

die() {
	echo "fetch-release: $*" >&2
	exit 1
}

if [ ! -f "$toml" ]; then
	die "missing $toml"
fi

version=$(sed -n 's/^version = "\([^"]*\)"/\1/p' "$toml" | head -n 1)
if [ -z "$version" ]; then
	die "no version field in $toml"
fi

os=$(uname -s)
arch=$(uname -m)

case "$os" in
MINGW* | MSYS* | CYGWIN*)
	die "native Windows is unsupported; use WSL2 (Linux artifacts)"
	;;
esac

case "$os/$arch" in
Linux/x86_64) archive=herdr-canvas_Linux_x86_64.tar.gz ;;
Linux/aarch64 | Linux/arm64) archive=herdr-canvas_Linux_arm64.tar.gz ;;
Darwin/x86_64) archive=herdr-canvas_Darwin_x86_64.tar.gz ;;
Darwin/arm64) archive=herdr-canvas_Darwin_arm64.tar.gz ;;
*)
	die "unsupported platform $os/$arch; native Windows is unsupported; use WSL2 (Linux artifacts)"
	;;
esac

download() {
	url=$1
	dest=$2
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$url" -o "$dest"
	elif command -v wget >/dev/null 2>&1; then
		wget -q -O "$dest" "$url"
	else
		die "need curl or wget; neither is on PATH"
	fi
}

sha256_of() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{ print $1 }'
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | awk '{ print $1 }'
	else
		die "need sha256sum or shasum"
	fi
}

root=$PWD
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

download "$base/v$version/$archive" "$tmp/$archive"
download "$base/v$version/checksums.txt" "$tmp/checksums.txt"

expected=$(awk -v f="$archive" '$NF == f { print $1; exit }' "$tmp/checksums.txt")
if [ -z "$expected" ]; then
	rm -f "$root/bin/herdr-canvas"
	die "no checksum line for $archive"
fi

actual=$(sha256_of "$tmp/$archive")
if [ "$actual" != "$expected" ]; then
	rm -f "$root/bin/herdr-canvas"
	die "SHA-256 mismatch for $archive"
fi

tar -xzf "$tmp/$archive" -C "$tmp" herdr-canvas
mkdir -p "$root/bin"
mv "$tmp/herdr-canvas" "$root/bin/herdr-canvas"
chmod 755 "$root/bin/herdr-canvas"
