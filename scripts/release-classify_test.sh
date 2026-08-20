#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
classify=$repo/scripts/release-classify.sh
decide=$repo/scripts/release-decide.sh
fail=0

ok() {
	echo "ok - $1"
}

not_ok() {
	echo "not ok - $1" >&2
	fail=1
}

expect_class() {
	name=$1
	want=$2
	file=$3
	got=$(/bin/sh "$classify" "$file")
	if [ "$got" = "$want" ]; then
		ok "classify $name -> $want"
	else
		not_ok "classify $name: want $want got $got"
	fi
}

expect_decide() {
	name=$1
	d=$2
	t=$3
	class=$4
	want_action=$5
	want_candidate=$6
	eval "$(/bin/sh "$decide" "$d" "$t" "$class")"
	if [ "$action" = "$want_action" ] && [ "$candidate" = "$want_candidate" ]; then
		ok "decide $name"
	else
		not_ok "decide $name: action=$action candidate=$candidate (want $want_action $want_candidate)"
	fi
}

fixtures=$(mktemp -d)
trap 'rm -rf "$fixtures"' EXIT

cat >"$fixtures/notes-only-v0.1.0.json" <<'EOF'
{"isDraft":false,"tagName":"v0.1.0","assets":[]}
EOF

cat >"$fixtures/draft-empty.json" <<'EOF'
{"isDraft":true,"tagName":"v0.2.0","assets":[]}
EOF

cat >"$fixtures/draft-complete.json" <<'EOF'
{
  "isDraft": true,
  "tagName": "v0.2.0",
  "assets": [
    {"name": "herdr-canvas_Darwin_x86_64.tar.gz"},
    {"name": "herdr-canvas_Darwin_arm64.tar.gz"},
    {"name": "herdr-canvas_Linux_x86_64.tar.gz"},
    {"name": "herdr-canvas_Linux_arm64.tar.gz"},
    {"name": "checksums.txt"}
  ]
}
EOF

cat >"$fixtures/published-complete.json" <<'EOF'
{
  "isDraft": false,
  "tagName": "v0.2.0",
  "assets": [
    {"name": "herdr-canvas_Darwin_x86_64.tar.gz"},
    {"name": "herdr-canvas_Darwin_arm64.tar.gz"},
    {"name": "herdr-canvas_Linux_x86_64.tar.gz"},
    {"name": "herdr-canvas_Linux_arm64.tar.gz"},
    {"name": "checksums.txt"}
  ]
}
EOF

cat >"$fixtures/published-four-archives-no-checksums.json" <<'EOF'
{
  "isDraft": false,
  "tagName": "v0.2.0",
  "assets": [
    {"name": "herdr-canvas_Darwin_x86_64.tar.gz"},
    {"name": "herdr-canvas_Darwin_arm64.tar.gz"},
    {"name": "herdr-canvas_Linux_x86_64.tar.gz"},
    {"name": "herdr-canvas_Linux_arm64.tar.gz"}
  ]
}
EOF

expect_class "notes-only v0.1.0" published_incomplete "$fixtures/notes-only-v0.1.0.json"
expect_class "draft without assets" draft "$fixtures/draft-empty.json"
expect_class "draft with five assets" draft "$fixtures/draft-complete.json"
expect_class "published five assets" published_complete "$fixtures/published-complete.json"
expect_class "published missing checksums" published_incomplete "$fixtures/published-four-archives-no-checksums.json"

got=$(/bin/sh "$classify" --missing)
if [ "$got" = missing ]; then
	ok "classify --missing"
else
	not_ok "classify --missing got $got"
fi

got=$(printf '' | /bin/sh "$classify")
if [ "$got" = missing ]; then
	ok "classify empty stdin is missing"
else
	not_ok "classify empty stdin got $got"
fi

if /bin/sh "$classify" --assert-draft-complete "$fixtures/draft-complete.json"; then
	ok "assert complete draft"
else
	not_ok "assert complete draft should pass"
fi

if /bin/sh "$classify" --assert-draft-complete "$fixtures/draft-empty.json"; then
	not_ok "assert incomplete draft should fail"
else
	ok "assert incomplete draft fails"
fi

if /bin/sh "$classify" --assert-draft-complete "$fixtures/published-complete.json"; then
	not_ok "assert published complete should fail (do not undraft/upload)"
else
	ok "assert published complete fails"
fi

if /bin/sh "$classify" --assert-draft-complete "$fixtures/notes-only-v0.1.0.json"; then
	not_ok "assert notes-only published should fail"
else
	ok "assert notes-only published fails"
fi

expect_decide "empty D + notes-only T" "" "0.1.0" published_incomplete fail_published_incomplete 0.1.0
expect_decide "empty D + published complete" "" "0.1.0" published_complete fail_noop 0.1.0
expect_decide "empty D + draft T" "" "0.2.0" draft resume 0.2.0
expect_decide "empty D + missing T" "" "0.1.0" missing fail_noop 0.1.0
expect_decide "D + published incomplete" "0.2.0" "0.1.0" published_incomplete fail_published_incomplete 0.2.0
expect_decide "D != T published complete" "0.2.0" "0.1.0" published_complete already_published 0.2.0
expect_decide "D != T missing" "0.2.0" "0.1.0" missing fresh_cut 0.2.0
expect_decide "D != T draft" "0.2.0" "0.1.0" draft fresh_cut 0.2.0
expect_decide "D == T published complete" "0.2.0" "0.2.0" published_complete already_published 0.2.0
expect_decide "D == T missing" "0.2.0" "0.2.0" missing resume 0.2.0
expect_decide "D == T draft" "0.2.0" "0.2.0" draft resume 0.2.0

status=$repo/scripts/release-gh-status.sh

expect_status() {
	name=$1
	want=$2
	code=$3
	err=$4
	out=${5:-}
	errf=$fixtures/err
	outf=$fixtures/out
	printf '%s\n' "$err" >"$errf"
	if [ -n "$out" ]; then
		printf '%s\n' "$out" >"$outf"
	else
		: >"$outf"
	fi
	set +e
	got=$(/bin/sh "$status" "$code" "$errf" "$outf" 2>"$fixtures/status.err")
	st=$?
	set -e
	if [ "$want" = fail ]; then
		if [ "$st" -ne 0 ] && [ "$got" != missing ]; then
			ok "gh-status $name fails closed"
		else
			not_ok "gh-status $name: st=$st got='$got' (want fail, not missing)"
		fi
		return
	fi
	if [ "$st" -eq 0 ] && [ "$got" = "$want" ]; then
		ok "gh-status $name -> $want"
	else
		not_ok "gh-status $name: st=$st got='$got' (want $want)"
	fi
}

expect_status "successful view" viewed 0 "" '{"isDraft":false,"tagName":"v0.2.0","assets":[]}'
expect_status "canonical release not found" missing 1 "release not found" ""
expect_status "HTTP 404 tag URL" missing 1 "HTTP 404: Not Found (https://api.github.com/repos/aorumbayev/herdr-canvas/releases/tags/v0.2.0)" ""
expect_status "GraphQL missing release" missing 1 "Could not resolve to a Release with tag name v0.2.0" ""
expect_status "empty stderr is not missing" fail 1 "" ""
expect_status "malformed JSON is not missing" fail 0 "" '{"isDraft":'
expect_status "HTTP 404 stderr even when exit is 0" missing 0 "HTTP 404: Not Found (https://api.github.com/repos/aorumbayev/herdr-canvas/releases/tags/v0.2.0)" ""
expect_status "gh api 404 JSON body is missing" missing 0 "" '{"message":"Not Found","documentation_url":"https://docs.github.com/rest","status":"404"}'
expect_status "gh api 404 with HTTP stderr is missing" missing 1 "gh: Not Found (HTTP 404)" '{"message":"Not Found","status":"404"}'
expect_status "HTTP 401 is not missing" fail 1 "HTTP 401: Bad credentials (https://api.github.com/repos/aorumbayev/herdr-canvas/releases/tags/v0.2.0)" ""
expect_status "HTTP 403 permission is not missing" fail 1 "HTTP 403: Resource not accessible by integration" ""
expect_status "rate limit is not missing" fail 1 "HTTP 403: API rate limit exceeded for user ID 1" ""
expect_status "HTTP 429 is not missing" fail 1 "HTTP 429: Too Many Requests" ""
expect_status "network DNS is not missing" fail 1 "Get \"https://api.github.com/repos/aorumbayev/herdr-canvas/releases/tags/v0.2.0\": dial tcp: lookup api.github.com: no such host" ""
expect_status "connection refused is not missing" fail 1 "dial tcp 127.0.0.1:443: connect: connection refused" ""

if grep -q 'if ! ./scripts/release-view.sh' "$repo/.github/workflows/release.yml"; then
	not_ok "workflow must not use if ! to capture release-view exit"
else
	ok "workflow captures release-view exit without if !"
fi

if grep -q 'release-gh-status.sh' "$repo/.github/workflows/release.yml"; then
	ok "workflow uses release-gh-status.sh"
else
	not_ok "workflow must call release-gh-status.sh"
fi

if grep -q 'release-view.sh' "$repo/.github/workflows/release.yml" \
	&& ! grep -q 'gh release view' "$repo/.github/workflows/release.yml"; then
	ok "workflow fetches releases via release-view.sh (gh api)"
else
	not_ok "workflow must use release-view.sh, not gh release view --json"
fi

if grep -n 'resume v0.1.0' "$repo/.github/workflows/release.yml" >/dev/null 2>&1; then
	not_ok "release.yml must not say resume v0.1.0"
else
	ok "release.yml has no resume v0.1.0 comment"
fi

if grep -E 'gh release delete|git push --delete|git push --force|--force-with-lease' "$repo/.github/workflows/release.yml" >/dev/null; then
	not_ok "release.yml must not delete tags/releases or force-push"
else
	ok "release.yml has no delete/force-push"
fi

if grep -q 'replace_existing_draft' "$repo/.goreleaser.yaml"; then
	not_ok ".goreleaser.yaml must not set replace_existing_draft"
else
	ok "replace_existing_draft unset"
fi

if grep -qi windows "$repo/.goreleaser.yaml"; then
	not_ok ".goreleaser.yaml must not mention windows"
else
	ok "no windows goreleaser target"
fi

if grep -E '^[[:space:]]+hooks:' "$repo/.github/workflows/release.yml"; then
	not_ok "must not enable semantic-release hooks"
else
	ok "no semantic-release hooks key"
fi

wf=$repo/.github/workflows/release.yml
vf=$repo/.github/workflows/verify.yml
if grep -q 'version: "^v2.5.0"' "$wf" && ! grep -q 'version: "~> v2"' "$wf"; then
	ok "goreleaser action version is ^v2.5.0"
else
	not_ok "goreleaser action version must be ^v2.5.0, not ~> v2"
fi

if grep -q 'toJSON(steps.dry.outputs.changelog)' "$wf" && grep -q 'write-release-notes.sh' "$wf"; then
	ok "changelog is handed off via toJSON + write-release-notes.sh"
else
	not_ok "changelog must use toJSON and write-release-notes.sh"
fi

if grep -n 'printf.*CHANGELOG' "$wf" >/dev/null 2>&1 || grep -n '${{ steps.dry.outputs.changelog }}' "$wf" | grep -v toJSON >/dev/null; then
	not_ok "changelog must not be interpolated raw into the shell"
else
	ok "changelog is not raw-interpolated into the shell"
fi

if grep -q 'fetch-release_test.sh' "$vf" && grep -q 'release-classify_test.sh' "$vf"; then
	ok "verify.yml runs both Unix shell helper suites"
else
	not_ok "verify.yml must run fetch-release and release-classify tests"
fi

writer=$repo/scripts/write-release-notes.sh
notes_dir=$(mktemp -d)
payload=$(python3 -c 'import json; print(json.dumps("feat: x\nfix: $(rm -rf /)\necho `id` %s '\''\"injected'\''"))')
CHANGELOG_JSON=$payload /bin/sh "$writer" "$notes_dir/release-notes.md"
if grep -q 'feat: x' "$notes_dir/release-notes.md" \
	&& grep -q '$(rm -rf /)' "$notes_dir/release-notes.md" \
	&& grep -q '%s' "$notes_dir/release-notes.md"; then
	ok "write-release-notes keeps multiline metacharacters"
else
	not_ok "write-release-notes mangled multiline notes"
fi

CHANGELOG_JSON=null /bin/sh "$writer" "$notes_dir/empty.md"
CHANGELOG_JSON='""' /bin/sh "$writer" "$notes_dir/empty.md"
if [ ! -f "$notes_dir/empty.md" ]; then
	ok "empty changelog does not write release-notes.md"
else
	not_ok "empty changelog wrote release-notes.md"
fi
rm -rf "$notes_dir"

if [ "$fail" -ne 0 ]; then
	exit 1
fi
echo "all classify/decide tests passed"
