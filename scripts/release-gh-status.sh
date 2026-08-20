#!/bin/sh
# Interpret `gh release view` success vs failure.
# Usage: release-gh-status.sh EXITCODE STDERR_FILE [STDOUT_FILE]
# Prints viewed | missing on stdout.
# Exit 0 only for a successful view or an authoritative not-found.
# Auth, permission, rate-limit, network, empty, and malformed results exit 1.
set -eu

if [ "$#" -lt 2 ]; then
	echo "release-gh-status: usage: release-gh-status.sh EXITCODE STDERR_FILE [STDOUT_FILE]" >&2
	exit 1
fi

exitcode=$1
err_file=$2
out_file=${3:-}

python3 - "$exitcode" "$err_file" "$out_file" <<'PY'
import json, sys

exitcode = int(sys.argv[1])
err_file = sys.argv[2]
out_file = sys.argv[3]

def die(msg: str) -> None:
    sys.stderr.write("release-gh-status: %s\n" % msg)
    sys.exit(1)

def load_stdout():
    if not out_file:
        return None
    try:
        raw = open(out_file, encoding="utf-8").read().strip()
    except OSError as exc:
        die("cannot read stdout file: %s" % exc)
    if not raw:
        return None
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return None

def is_not_found_payload(data) -> bool:
    return isinstance(data, dict) and (
        str(data.get("status")) == "404" or data.get("message") == "Not Found"
    )

try:
    err = open(err_file, encoding="utf-8", errors="replace").read()
except OSError as exc:
    die("cannot read stderr file: %s" % exc)

err_l = err.lower()
if any(
    needle in err_l
    for needle in (
        "release not found",
        "http 404",
        "could not resolve to a release",
        "could not find release",
    )
):
    print("missing")
    sys.exit(0)

stdout_data = load_stdout()
if is_not_found_payload(stdout_data):
    print("missing")
    sys.exit(0)

if exitcode == 0:
    if stdout_data is None:
        die("gh release view exited 0 with empty JSON")
    if not isinstance(stdout_data, dict) or "isDraft" not in stdout_data:
        die("gh release view JSON is missing isDraft")
    print("viewed")
    sys.exit(0)

err_l = err.lower()
if not err.strip():
    die("gh release view exited %s with empty stderr; not treating as missing" % exitcode)

blocked = (
    "http 401",
    "http 403",
    "http 429",
    "http 500",
    "http 502",
    "http 503",
    "http 504",
    "bad credentials",
    "requires authentication",
    "api rate limit",
    "rate limit exceeded",
    "resource not accessible",
    "must specify gh_token",
    "gh_token environment",
    "dial tcp",
    "connection refused",
    "connection reset",
    "network is unreachable",
    "no such host",
    "temporary failure in name resolution",
    "i/o timeout",
    "tls handshake",
    "certificate",
    "eof",
    "command not found",
)

for needle in blocked:
    if needle in err_l:
        die("gh release view failed (%s); not treating as missing:\n%s" % (exitcode, err.rstrip()))

not_found = (
    "release not found",
    "http 404",
    "could not resolve to a release",
    "could not find release",
)
if any(needle in err_l for needle in not_found):
    print("missing")
    sys.exit(0)

die("gh release view failed (%s); not treating as missing:\n%s" % (exitcode, err.rstrip()))
PY
