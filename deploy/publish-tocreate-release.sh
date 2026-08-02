#!/usr/bin/env bash
set -Eeuo pipefail

SRC_DIR="${SRC_DIR:-/home/ubuntu/sub2api-src}"
BRANCH="${BRANCH:-custom/subscription-quota-window}"
CUSTOM_REPO="${CUSTOM_REPO:-floating0516/sub2api-ToCreate}"
IMAGE_REPO="${IMAGE_REPO:-ghcr.io/floating0516/sub2api-tocreate}"
RELEASE_COMMAND_TIMEOUT_SECONDS="${RELEASE_COMMAND_TIMEOUT_SECONDS:-120}"
RELEASE_GIT_NAME="${RELEASE_GIT_NAME:-ToCreate Release Bot}"
RELEASE_GIT_EMAIL="${RELEASE_GIT_EMAIL:-tocreate-release@users.noreply.github.com}"

IMAGE="${1:-}"
IMAGE_DIGEST="${2:-}"
SOURCE_COMMIT="${3:-}"
UPSTREAM_COMMIT="${4:-}"
DEPLOYED_AT="${5:-}"

notes_file=""

log() {
  printf '[release] %s\n' "$*" >&2
}

die() {
  log "$*"
  exit 1
}

cleanup() {
  if [ -n "$notes_file" ]; then
    rm -f -- "$notes_file"
  fi
}
trap cleanup EXIT

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

run_network() {
  timeout --foreground "$RELEASE_COMMAND_TIMEOUT_SECONDS" "$@"
}

local_tag_is_valid() {
  local release_tag="$1"
  local source_commit="$2"
  local expected_digest="$3"
  local tag_type=""
  local tag_commit=""
  local recorded_digest=""

  tag_type="$(git -C "$SRC_DIR" cat-file -t "refs/tags/$release_tag" 2>/dev/null)"
  [ "$tag_type" = "tag" ] || return 1
  tag_commit="$(
    git -C "$SRC_DIR" rev-parse --verify "$release_tag^{commit}" 2>/dev/null
  )"
  [ "$tag_commit" = "$source_commit" ] || return 1
  recorded_digest="$(
    git -C "$SRC_DIR" for-each-ref \
      --format='%(contents)' "refs/tags/$release_tag" \
      | sed -n 's/^OCI digest: //p' \
      | head -n 1
  )"
  [ "$recorded_digest" = "$expected_digest" ]
}

release_json_result() {
  local release_tag="$1"
  local release_json="$2"
  local release_url=""
  local published_at=""

  release_url="$(jq -r '.url // empty' <<<"$release_json")"
  published_at="$(jq -r '.publishedAt // empty' <<<"$release_json")"
  [ -n "$release_url" ] || die "GitHub Release URL is missing"
  jq -n \
    --arg status "published" \
    --arg tag "$release_tag" \
    --arg url "$release_url" \
    --arg published_at "$published_at" \
    '{status: $status, tag: $tag, url: $url, published_at: $published_at}'
}

for command in git gh jq timeout awk sed head mktemp; do
  need_cmd "$command"
done

case "$RELEASE_COMMAND_TIMEOUT_SECONDS" in
  ''|*[!0-9]*) die "RELEASE_COMMAND_TIMEOUT_SECONDS must be a positive integer" ;;
esac
[ "$RELEASE_COMMAND_TIMEOUT_SECONDS" -ge 1 ] \
  || die "RELEASE_COMMAND_TIMEOUT_SECONDS must be a positive integer"
[ -d "$SRC_DIR/.git" ] || die "source repository not found: $SRC_DIR"
[[ "$CUSTOM_REPO" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] \
  || die "invalid GitHub repository"

case "$IMAGE" in
  "$IMAGE_REPO":*) image_tag="${IMAGE#"$IMAGE_REPO:"}" ;;
  *) die "unexpected image repository" ;;
esac
if [[ "$image_tag" =~ ^([0-9]+\.[0-9]+\.[0-9]+)-(tc[0-9]+\.[0-9]+(\.[0-9]+)?)$ ]]; then
  app_version="${BASH_REMATCH[1]}"
else
  die "image tag is not a formal ToCreate release"
fi
case "$IMAGE_DIGEST" in
  "$IMAGE_REPO"@sha256:*) digest_value="${IMAGE_DIGEST#"$IMAGE_REPO@"}" ;;
  *) die "invalid image digest repository" ;;
esac
[[ "$digest_value" =~ ^sha256:[0-9a-f]{64}$ ]] || die "invalid image digest"
[[ "$SOURCE_COMMIT" =~ ^[0-9a-f]{40}$ ]] || die "invalid source commit"
[[ "$UPSTREAM_COMMIT" =~ ^[0-9a-f]{40}$ ]] || die "invalid upstream commit"
[[ "$DEPLOYED_AT" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$ ]] \
  || die "invalid deployment timestamp"

git -C "$SRC_DIR" cat-file -e "$SOURCE_COMMIT^{commit}" 2>/dev/null \
  || die "source commit is unavailable locally"
git -C "$SRC_DIR" cat-file -e "$UPSTREAM_COMMIT^{commit}" 2>/dev/null \
  || die "upstream commit is unavailable locally"
git -C "$SRC_DIR" merge-base --is-ancestor "$UPSTREAM_COMMIT" "$SOURCE_COMMIT" \
  || die "upstream commit is not an ancestor of the release source"

remote_branch_commit="$(
  run_network git -C "$SRC_DIR" ls-remote origin "refs/heads/$BRANCH" \
    | awk 'NR == 1 {print $1}'
)"
[[ "$remote_branch_commit" =~ ^[0-9a-f]{40}$ ]] \
  || die "custom branch is unavailable on GitHub"
git -C "$SRC_DIR" cat-file -e "$remote_branch_commit^{commit}" 2>/dev/null \
  || die "remote branch commit is unavailable locally"
git -C "$SRC_DIR" merge-base --is-ancestor "$SOURCE_COMMIT" "$remote_branch_commit" \
  || die "release source is not contained in the pushed custom branch"

release_tag="tocreate-v$image_tag"
expected_digest="$digest_value"
remote_refs="$(
  run_network git -C "$SRC_DIR" ls-remote --tags origin \
    "refs/tags/$release_tag" "refs/tags/$release_tag^{}"
)"
remote_direct="$(
  awk -v ref="refs/tags/$release_tag" '$2 == ref {print $1; exit}' <<<"$remote_refs"
)"
remote_peeled="$(
  awk -v ref="refs/tags/$release_tag^{}" '$2 == ref {print $1; exit}' <<<"$remote_refs"
)"

if [ -n "$remote_direct" ]; then
  remote_tag_commit="${remote_peeled:-$remote_direct}"
  [ "$remote_tag_commit" = "$SOURCE_COMMIT" ] \
    || die "remote release tag points to a different source commit"
  if ! git -C "$SRC_DIR" show-ref --verify --quiet "refs/tags/$release_tag"; then
    run_network git -C "$SRC_DIR" fetch origin \
      "refs/tags/$release_tag:refs/tags/$release_tag" >/dev/null
  fi
  local_tag_is_valid "$release_tag" "$SOURCE_COMMIT" "$expected_digest" \
    || die "existing release tag is not an annotated tag with the expected digest"
else
  if git -C "$SRC_DIR" show-ref --verify --quiet "refs/tags/$release_tag"; then
    local_tag_is_valid "$release_tag" "$SOURCE_COMMIT" "$expected_digest" \
      || die "local release tag conflicts with the requested release"
  else
    GIT_COMMITTER_NAME="$RELEASE_GIT_NAME" \
      GIT_COMMITTER_EMAIL="$RELEASE_GIT_EMAIL" \
      git -C "$SRC_DIR" tag -a "$release_tag" "$SOURCE_COMMIT" \
        -m "ToCreate $image_tag" \
        -m "Official base: v$app_version
Image: $IMAGE
OCI digest: $expected_digest
Source commit: $SOURCE_COMMIT
Upstream commit: $UPSTREAM_COMMIT"
  fi
  log "Pushing immutable tag $release_tag"
  run_network env GIT_TERMINAL_PROMPT=0 \
    git -C "$SRC_DIR" push origin "refs/tags/$release_tag"
  remote_peeled="$(
    run_network git -C "$SRC_DIR" ls-remote --tags origin "refs/tags/$release_tag^{}" \
      | awk 'NR == 1 {print $1}'
  )"
  [ "$remote_peeled" = "$SOURCE_COMMIT" ] \
    || die "pushed release tag could not be verified"
fi

if release_json="$(
  run_network gh release view "$release_tag" --repo "$CUSTOM_REPO" \
    --json tagName,url,isDraft,isPrerelease,publishedAt 2>/dev/null
)"; then
  [ "$(jq -r '.tagName' <<<"$release_json")" = "$release_tag" ] \
    || die "existing GitHub Release tag does not match"
  [ "$(jq -r '.isPrerelease' <<<"$release_json")" = "false" ] \
    || die "existing GitHub Release is unexpectedly marked as a prerelease"
  if [ "$(jq -r '.isDraft' <<<"$release_json")" = "true" ]; then
    log "Publishing existing draft Release $release_tag"
    run_network gh release edit "$release_tag" --repo "$CUSTOM_REPO" \
      --draft=false --prerelease=false --latest >/dev/null
    release_json="$(
      run_network gh release view "$release_tag" --repo "$CUSTOM_REPO" \
        --json tagName,url,isDraft,isPrerelease,publishedAt
    )"
  fi
  release_json_result "$release_tag" "$release_json"
  exit 0
fi

previous_release_tag="$(
  git -C "$SRC_DIR" for-each-ref --merged "$SOURCE_COMMIT" \
    --sort=-taggerdate --format='%(refname:strip=2)' 'refs/tags/tocreate-v*' \
    | awk -v current="$release_tag" '$0 != current {print; exit}'
)"
notes_file="$(mktemp)"
chmod 0600 "$notes_file"
{
  printf '%s\n\n' '## Version'
  printf '%s\n' "- Official base: \`v$app_version\`"
  printf '%s\n' "- Custom release: \`$image_tag\`"
  printf '%s\n' "- Source commit: [\`$SOURCE_COMMIT\`](https://github.com/$CUSTOM_REPO/commit/$SOURCE_COMMIT)"
  printf '%s\n\n' "- Upstream commit: \`$UPSTREAM_COMMIT\`"
  printf '%s\n\n' '## Image'
  printf '%s\n' "- Release image: \`$IMAGE\`"
  printf '%s\n' "- Immutable source image: \`$IMAGE_REPO:$SOURCE_COMMIT\`"
  printf '%s\n' "- OCI Index Digest: \`$expected_digest\`"
  printf '%s\n\n' '- Platform: `linux/amd64`'
  printf '%s\n\n' '## Deployment'
  printf '%s\n' "The exact image was validated on staging and promoted to production at \`$DEPLOYED_AT\`."
  if [ -n "$previous_release_tag" ]; then
    printf '\nCompare: https://github.com/%s/compare/%s...%s\n' \
      "$CUSTOM_REPO" "$previous_release_tag" "$release_tag"
  fi
} > "$notes_file"

log "Creating GitHub Release $release_tag"
run_network gh release create "$release_tag" --repo "$CUSTOM_REPO" \
  --verify-tag --latest --title "ToCreate $image_tag" --notes-file "$notes_file" >/dev/null
release_json="$(
  run_network gh release view "$release_tag" --repo "$CUSTOM_REPO" \
    --json tagName,url,isDraft,isPrerelease,publishedAt
)"
[ "$(jq -r '.tagName' <<<"$release_json")" = "$release_tag" ] \
  || die "created GitHub Release could not be verified"
[ "$(jq -r '.isDraft' <<<"$release_json")" = "false" ] \
  || die "created GitHub Release is still a draft"
release_json_result "$release_tag" "$release_json"
