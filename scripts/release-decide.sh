#!/bin/sh
# Map dry-run D, toml T, and classify(v$CANDIDATE) to a workflow action.
# Usage: release-decide.sh D T CLASS
# Prints candidate=, class=, action= for eval.
set -eu

D=${1:-}
T=${2:-}
CLASS=${3:-}

if [ -z "$T" ] || [ -z "$CLASS" ]; then
	echo "release-decide: usage: release-decide.sh D T CLASS" >&2
	exit 1
fi

case "$CLASS" in
missing | draft | published_complete | published_incomplete) ;;
*)
	echo "release-decide: unknown class '$CLASS'" >&2
	exit 1
	;;
esac

if [ -n "$D" ]; then
	CANDIDATE=$D
else
	CANDIDATE=$T
fi

if [ -z "$D" ]; then
	case "$CLASS" in
	published_incomplete) ACTION=fail_published_incomplete ;;
	published_complete) ACTION=fail_noop ;;
	draft) ACTION=resume ;;
	missing) ACTION=fail_noop ;;
	esac
else
	case "$CLASS" in
	published_incomplete) ACTION=fail_published_incomplete ;;
	published_complete) ACTION=already_published ;;
	missing | draft)
		if [ "$T" = "$D" ]; then
			ACTION=resume
		else
			ACTION=fresh_cut
		fi
		;;
	esac
fi

printf 'candidate=%s\n' "$CANDIDATE"
printf 'class=%s\n' "$CLASS"
printf 'action=%s\n' "$ACTION"
