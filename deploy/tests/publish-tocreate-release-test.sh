#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUBLISHER="$SCRIPT_DIR/../publish-tocreate-release.sh"
TEST_ROOT="$(mktemp -d)"
trap 'rm -rf -- "$TEST_ROOT"' EXIT

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

assert_eq() {
  local expected="$1"
  local actual="$2"
  local message="$3"
  [ "$expected" = "$actual" ] || fail "$message (expected=$expected actual=$actual)"
}

repo="$TEST_ROOT/repo"
remote="$TEST_ROOT/remote.git"
fake_bin="$TEST_ROOT/bin"
fake_release="$TEST_ROOT/release-created"
fake_log="$TEST_ROOT/gh.log"
branch="custom/test"
custom_repo="example/sub2api"
image_repo="ghcr.io/example/sub2api"
image="$image_repo:0.1.200-tc2.1"
digest="$image_repo@sha256:$(printf 'a%.0s' {1..64})"

mkdir -p "$repo" "$fake_bin"
git init -q --bare "$remote"
git -C "$repo" init -q
git -C "$repo" config user.name "Release Test"
git -C "$repo" config user.email "release-test@example.invalid"
git -C "$repo" checkout -qb "$branch"
printf 'official\n' > "$repo/source.txt"
git -C "$repo" add source.txt
git -C "$repo" commit -qm official
upstream_commit="$(git -C "$repo" rev-parse HEAD)"
printf 'custom\n' >> "$repo/source.txt"
git -C "$repo" commit -qam custom
source_commit="$(git -C "$repo" rev-parse HEAD)"
git -C "$repo" remote add origin "$remote"
git -C "$repo" push -q -u origin "$branch"

printf '%s\n' \
  '#!/usr/bin/env bash' \
  'set -Eeuo pipefail' \
  'printf "%s\n" "$*" >> "$FAKE_GH_LOG"' \
  'case "${1:-}:${2:-}" in' \
  '  release:view)' \
  '    [ -f "$FAKE_GH_RELEASE" ] || exit 1' \
  '    tag="${3:-}"' \
  '    printf "{\"tagName\":\"%s\",\"url\":\"https://github.com/example/sub2api/releases/tag/%s\",\"isDraft\":false,\"isPrerelease\":false,\"publishedAt\":\"2026-08-02T07:00:00Z\"}\n" "$tag" "$tag"' \
  '    ;;' \
  '  release:create)' \
  '    touch "$FAKE_GH_RELEASE"' \
  '    ;;' \
  '  release:edit)' \
  '    touch "$FAKE_GH_RELEASE"' \
  '    ;;' \
  '  *) exit 1 ;;' \
  'esac' > "$fake_bin/gh"
chmod 0755 "$fake_bin/gh"

run_publisher() {
  PATH="$fake_bin:$PATH" \
  FAKE_GH_LOG="$fake_log" \
  FAKE_GH_RELEASE="$fake_release" \
  SRC_DIR="$repo" \
  BRANCH="$branch" \
  CUSTOM_REPO="$custom_repo" \
  IMAGE_REPO="$image_repo" \
  RELEASE_COMMAND_TIMEOUT_SECONDS=10 \
    "$PUBLISHER" \
      "$image" "$digest" "$source_commit" "$upstream_commit" \
      '2026-08-02T07:00:00Z'
}

result="$(run_publisher)"
release_tag="tocreate-v0.1.200-tc2.1"
assert_eq "published" "$(jq -r '.status' <<<"$result")" \
  "publisher did not return a published status"
assert_eq "$release_tag" "$(jq -r '.tag' <<<"$result")" \
  "publisher returned the wrong release tag"
assert_eq "$source_commit" "$(
  git -C "$repo" ls-remote --tags origin "refs/tags/$release_tag^{}" | awk '{print $1}'
)" "remote tag does not point to the source commit"
assert_eq "tag" "$(git -C "$repo" cat-file -t "refs/tags/$release_tag")" \
  "release tag is not annotated"
grep -q "OCI digest: ${digest#*@}" < <(
  git -C "$repo" for-each-ref --format='%(contents)' "refs/tags/$release_tag"
) || fail "release tag omitted the image digest"
grep -q '^release create ' "$fake_log" || fail "GitHub Release was not created"

: > "$fake_log"
result="$(run_publisher)"
assert_eq "published" "$(jq -r '.status' <<<"$result")" \
  "idempotent publisher run did not recognize the existing release"
if grep -q '^release create ' "$fake_log"; then
  fail "idempotent publisher run tried to recreate the release"
fi

conflict_tag="tocreate-v0.1.200-tc2.2"
conflict_digest="$image_repo@sha256:$(printf 'b%.0s' {1..64})"
GIT_COMMITTER_NAME="Release Test" GIT_COMMITTER_EMAIL="release-test@example.invalid" \
  git -C "$repo" tag -a "$conflict_tag" "$upstream_commit" \
    -m "ToCreate 0.1.200-tc2.2" \
    -m "OCI digest: ${conflict_digest#*@}"
git -C "$repo" push -q origin "refs/tags/$conflict_tag"
if PATH="$fake_bin:$PATH" \
  FAKE_GH_LOG="$fake_log" \
  FAKE_GH_RELEASE="$fake_release" \
  SRC_DIR="$repo" \
  BRANCH="$branch" \
  CUSTOM_REPO="$custom_repo" \
  IMAGE_REPO="$image_repo" \
  RELEASE_COMMAND_TIMEOUT_SECONDS=10 \
    "$PUBLISHER" \
      "$image_repo:0.1.200-tc2.2" "$conflict_digest" \
      "$source_commit" "$upstream_commit" '2026-08-02T07:00:00Z' \
      >/dev/null 2>&1; then
  fail "publisher accepted an existing tag that points to a different commit"
fi

printf 'tocreate release publisher tests passed\n'
