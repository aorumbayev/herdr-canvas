#!/bin/sh
# Local HTTP fixture tests for scripts/fetch-release.sh. No GitHub.
set -eu

repo=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
script=$repo/scripts/fetch-release.sh
fail=0

assert() {
	msg=$1
	shift
	if "$@"; then
		echo "ok - $msg"
	else
		echo "not ok - $msg" >&2
		fail=1
	fi
}

assert_false() {
	msg=$1
	shift
	if "$@"; then
		echo "not ok - $msg" >&2
		fail=1
	else
		echo "ok - $msg"
	fi
}

make_archive() {
	dir=$1
	name=$2
	payload=$3
	(
		cd "$dir"
		printf '%s\n' "$payload" >herdr-canvas
		tar -czf "$name" herdr-canvas
		rm -f herdr-canvas
		if command -v sha256sum >/dev/null 2>&1; then
			sum=$(sha256sum "$name" | awk '{ print $1 }')
		else
			sum=$(shasum -a 256 "$name" | awk '{ print $1 }')
		fi
		printf '%s  %s\n' "$sum" "$name"
	)
}

write_toml() {
	printf 'version = "%s"\n' "$1" >herdr-plugin.toml
}

start_httpd() {
	root=$1
	portfile=$2
	python3 - "$root" "$portfile" <<'PY' &
import http.server, os, socketserver, sys, time
root, portfile = sys.argv[1], sys.argv[2]
os.chdir(root)

class Reuse(socketserver.TCPServer):
    allow_reuse_address = True

httpd = Reuse(("127.0.0.1", 0), http.server.SimpleHTTPRequestHandler)
with open(portfile, "w", encoding="utf-8") as f:
    f.write(str(httpd.server_address[1]))
httpd.serve_forever()
PY
	httpd_pid=$!
	i=0
	while [ ! -s "$portfile" ]; do
		i=$((i + 1))
		if [ "$i" -gt 50 ]; then
			echo "httpd did not start" >&2
			exit 1
		fi
		sleep 0.05
	done
}

stop_httpd() {
	if [ -n "${httpd_pid:-}" ]; then
		kill "$httpd_pid" 2>/dev/null || true
		wait "$httpd_pid" 2>/dev/null || true
	fi
}

mode_of() {
	python3 -c "import os,stat; print(oct(os.stat('$1').st_mode & 0o777))"
}

run_with_uname() {
	s=$1
	m=$2
	shift 2
	bindir=$PWD/uname-bin
	mkdir -p "$bindir"
	cat >"$bindir/uname" <<EOF
#!/bin/sh
case "\$1" in
-s) printf '%s\\n' "$s" ;;
-m) printf '%s\\n' "$m" ;;
*) printf '%s\\n' "$s" ;;
esac
EOF
	chmod 755 "$bindir/uname"
	PATH="$bindir:$PATH" "$@"
}

hide_curl_keep() {
	# PATH sandbox: every /bin and /usr/bin tool except curl (and optionally wget).
	sandbox=$1
	keep_wget=$2
	mkdir -p "$sandbox"
	for d in /bin /usr/bin /usr/local/bin; do
		[ -d "$d" ] || continue
		for f in "$d"/*; do
			[ -e "$f" ] || continue
			b=$(basename "$f")
			case "$b" in
			curl) continue ;;
			wget)
				if [ "$keep_wget" = no ]; then
					continue
				fi
				;;
			esac
			[ -e "$sandbox/$b" ] || ln -sf "$f" "$sandbox/$b"
		done
	done
}

workdir=$(mktemp -d)
trap 'stop_httpd; rm -rf "$workdir"' EXIT
cd "$workdir"

web=$workdir/www
mkdir -p "$web/v0.9.9"
checksum_line=$(make_archive "$web/v0.9.9" herdr-canvas_Darwin_arm64.tar.gz payload-arm64)
printf '%s\n' "$checksum_line" >"$web/v0.9.9/checksums.txt"
make_archive "$web/v0.9.9" herdr-canvas_Darwin_x86_64.tar.gz payload-amd64 >/dev/null
make_archive "$web/v0.9.9" herdr-canvas_Linux_x86_64.tar.gz payload-linux-amd64 >/dev/null
make_archive "$web/v0.9.9" herdr-canvas_Linux_arm64.tar.gz payload-linux-arm64 >/dev/null
{
	make_archive "$web/v0.9.9" herdr-canvas_Darwin_arm64.tar.gz payload-arm64
	make_archive "$web/v0.9.9" herdr-canvas_Darwin_x86_64.tar.gz payload-amd64
	make_archive "$web/v0.9.9" herdr-canvas_Linux_x86_64.tar.gz payload-linux-amd64
	make_archive "$web/v0.9.9" herdr-canvas_Linux_arm64.tar.gz payload-linux-arm64
} >"$web/v0.9.9/checksums.txt"

mkdir -p "$web/v0.9.8"
cp "$web/v0.9.9/herdr-canvas_Darwin_arm64.tar.gz" "$web/v0.9.8/"
printf '%s  %s\n' "0000000000000000000000000000000000000000000000000000000000000000" \
	"herdr-canvas_Darwin_arm64.tar.gz" >"$web/v0.9.8/checksums.txt"

portfile=$workdir/port
start_httpd "$web" "$portfile"
port=$(cat "$portfile")
base=http://127.0.0.1:$port
export HERDR_CANVAS_RELEASE_BASE=$base

# Match on Darwin/arm64
mkdir case-match
cd case-match
write_toml 0.9.9
run_with_uname Darwin arm64 /bin/sh "$script"
assert "match creates bin/herdr-canvas" test -f bin/herdr-canvas
assert "match is executable" test -x bin/herdr-canvas
got=$(mode_of bin/herdr-canvas)
case "$got" in
*755 | *775 | *555) echo "ok - mode $got" ;;
*)
	echo "not ok - mode $got" >&2
	fail=1
	;;
esac
assert "match payload" grep -q payload-arm64 bin/herdr-canvas
cd "$workdir"

# Mismatch leaves no binary
mkdir case-mismatch
cd case-mismatch
write_toml 0.9.8
if run_with_uname Darwin arm64 /bin/sh "$script"; then
	echo "not ok - mismatch should exit 1" >&2
	fail=1
else
	echo "ok - mismatch exits 1"
fi
assert_false "mismatch leaves no binary" test -e bin/herdr-canvas
cd "$workdir"

# Four archive names
for spec in "Darwin/x86_64:herdr-canvas_Darwin_x86_64.tar.gz:payload-amd64" \
	"Darwin/arm64:herdr-canvas_Darwin_arm64.tar.gz:payload-arm64" \
	"Linux/x86_64:herdr-canvas_Linux_x86_64.tar.gz:payload-linux-amd64" \
	"Linux/aarch64:herdr-canvas_Linux_arm64.tar.gz:payload-linux-arm64"; do
	osarch=${spec%%:*}
	rest=${spec#*:}
	want=${rest%%:*}
	payload=${rest#*:}
	os=${osarch%/*}
	arch=${osarch#*/}
	d=$workdir/map-$os-$arch
	mkdir "$d"
	cd "$d"
	write_toml 0.9.9
	run_with_uname "$os" "$arch" /bin/sh "$script"
	assert "map $os/$arch -> $want" grep -q "$payload" bin/herdr-canvas
	cd "$workdir"
done

# Linux arm64 alias
mkdir case-linux-arm64
cd case-linux-arm64
write_toml 0.9.9
run_with_uname Linux arm64 /bin/sh "$script"
assert "Linux/arm64 alias" grep -q payload-linux-arm64 bin/herdr-canvas
cd "$workdir"

# Native Windows
mkdir case-win
cd case-win
write_toml 0.9.9
if run_with_uname MINGW64_NT-10.0 x86_64 /bin/sh "$script"; then
	echo "not ok - windows should fail" >&2
	fail=1
else
	echo "ok - MINGW exits 1"
fi
assert_false "windows leaves no binary" test -e bin/herdr-canvas
cd "$workdir"

# Neither curl nor wget
mkdir case-neither
cd case-neither
write_toml 0.9.9
hide_curl_keep "$PWD/limited" no
set +e
out=$(
	PATH="$PWD/limited" run_with_uname Darwin arm64 /bin/sh "$script" 2>&1
)
st=$?
set -e
assert "neither curl nor wget exits 1" test "$st" -eq 1
printf '%s\n' "$out" >"$workdir/neither.err"
assert "error names curl" grep -q curl "$workdir/neither.err"
assert "error names wget" grep -q wget "$workdir/neither.err"
cd "$workdir"

# wget when curl is absent
mkdir case-wget
cd case-wget
write_toml 0.9.9
hide_curl_keep "$PWD/limited" no
cat >"$PWD/limited/wget" <<'EOF'
#!/bin/sh
set -eu
out=""
url=""
while [ $# -gt 0 ]; do
	case "$1" in
	-q)
		shift
		;;
	-O)
		out=$2
		shift 2
		;;
	*)
		url=$1
		shift
		;;
	esac
done
python3 -c 'import sys, urllib.request; urllib.request.urlretrieve(sys.argv[1], sys.argv[2])' "$url" "$out"
EOF
chmod 755 "$PWD/limited/wget"
PATH="$PWD/limited" run_with_uname Darwin arm64 /bin/sh "$script"
assert "wget fallback installs binary" test -f bin/herdr-canvas
cd "$workdir"

assert_false "no go command in fetch-release.sh" grep -E '(^|[[:space:]])go[[:space:]]' "$script"

assert "downloads v\$version assets" grep -q 'v$version/' "$script"
if grep -q releases/latest "$script"; then
	echo "not ok - must not use /releases/latest" >&2
	fail=1
else
	echo "ok - no /releases/latest"
fi

if [ "$fail" -ne 0 ]; then
	exit 1
fi
echo "all fetch-release tests passed"
