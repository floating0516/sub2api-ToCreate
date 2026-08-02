#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTROLLER="$SCRIPT_DIR/../tocreate-update-controller.sh"

# shellcheck source=../tocreate-update-controller.sh
source "$CONTROLLER"

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

assert_status_has_no_resolver_metadata() {
  if jq -e '
    has("resolution_id") or
    has("resolution_summary") or
    has("resolution_risk_level") or
    has("resolution_warnings") or
    has("resolution_diff_stat") or
    has("resolver_model")
  ' "$STATUS_FILE" >/dev/null; then
    fail "status retained retired conflict resolver metadata"
  fi
}

test_merge_conflict_pauses_without_source_changes() (
  local test_dir="$TEST_ROOT/manual-conflict"
  local repo="$test_dir/repo"
  local upstream_repo="$test_dir/upstream.git"
  local base_commit=""
  local source_commit=""
  local request_id="0123456789abcdef0123456789abcdef"
  local merge_branch="tocreate/official-merge-$request_id"

  mkdir -p "$repo"
  git -C "$repo" init -q -b main
  git -C "$repo" config user.name "Controller Test"
  git -C "$repo" config user.email "controller-test@example.invalid"
  printf 'mode=base\n' > "$repo/config.txt"
  git -C "$repo" add config.txt
  git -C "$repo" commit -qm base
  base_commit="$(git -C "$repo" rev-parse HEAD)"

  git -C "$repo" checkout -qb official
  printf 'mode=official\n' > "$repo/config.txt"
  git -C "$repo" commit -qam official
  git init --bare -q "$upstream_repo"
  git -C "$repo" push -q "$upstream_repo" official:main

  git -C "$repo" checkout -qb custom "$base_commit"
  printf 'mode=custom\n' > "$repo/config.txt"
  git -C "$repo" commit -qam custom
  source_commit="$(git -C "$repo" rev-parse HEAD)"
  git -C "$repo" remote add upstream "$upstream_repo"

  SRC_DIR="$repo"
  BRANCH="custom"
  UPSTREAM_REMOTE="upstream"
  UPSTREAM_REF="main"
  CONTROL_DIR="$test_dir/control"
  STATE_DIR="$test_dir/state"
  STATUS_FILE="$CONTROL_DIR/status.json"
  LOG_DIR="$STATE_DIR/logs"
  MERGE_WORKTREE_ROOT="$STATE_DIR/merge-worktrees"
  current_request_id="$request_id"
  mkdir -p "$CONTROL_DIR" "$LOG_DIR" "$MERGE_WORKTREE_ROOT"

  stage_update

  assert_eq "conflict_detected" "$(jq -r '.state' "$STATUS_FILE")" \
    "merge conflicts did not pause the update"
  assert_eq "config.txt" "$(jq -r '.conflict_files[0]' "$STATUS_FILE")" \
    "status omitted the conflicted path"
  assert_eq "action_required" \
    "$(jq -r '.steps[] | select(.id == "conflict_resolution") | .status' "$STATUS_FILE")" \
    "conflict check was not marked for manual action"
  assert_eq "$source_commit" "$(git -C "$repo" rev-parse HEAD)" \
    "source branch changed after a conflicted merge"
  [ -z "$(git -C "$repo" status --porcelain)" ] || \
    fail "source worktree became dirty after a conflicted merge"
  [ ! -e "$MERGE_WORKTREE_ROOT/$request_id" ] || \
    fail "isolated conflict worktree was not removed"
  if git -C "$repo" show-ref --verify --quiet "refs/heads/$merge_branch"; then
    fail "isolated conflict branch was not removed"
  fi
  assert_status_has_no_resolver_metadata
)

test_retired_resolution_action_is_rejected() (
  local test_dir="$TEST_ROOT/retired-action"

  CONTROL_DIR="$test_dir/control"
  STATE_DIR="$test_dir/state"
  STATUS_FILE="$CONTROL_DIR/status.json"
  PROCESSING_FILE="$CONTROL_DIR/processing.json"
  REQUEST_ARCHIVE_DIR="$STATE_DIR/requests"
  mkdir -p "$CONTROL_DIR" "$REQUEST_ARCHIVE_DIR"
  jq -n '{state: "completed"}' > "$STATUS_FILE"
  jq -n '{
    id: "11111111111111111111111111111111",
    action: "accept_resolution",
    resolution_id: "22222222222222222222222222222222"
  }' > "$PROCESSING_FILE"

  if ! process_request; then
    :
  fi

  assert_eq "failed" "$(jq -r '.state' "$STATUS_FILE")" \
    "retired resolver action was not rejected"
  assert_eq "invalid" "$(jq -r '.action' "$STATUS_FILE")" \
    "retired resolver action was not classified as invalid"
  [ ! -e "$PROCESSING_FILE" ] || fail "rejected request was not archived"
  assert_eq "1" "$(find "$REQUEST_ARCHIVE_DIR" -maxdepth 1 -type f | wc -l | tr -d ' ')" \
    "rejected request archive was not created"
  assert_status_has_no_resolver_metadata
)

test_completed_status_runtime_reconciliation() (
  local status_dir="$TEST_ROOT/runtime-status"
  local bluegreen_dir="$TEST_ROOT/blue-green"
  local old_image="ghcr.io/floating0516/sub2api-tocreate:0.1.168-tc1.23.1"
  local expected_prod_image="ghcr.io/floating0516/sub2api-tocreate:0.1.169-tc1.24"
  local prod_digest="ghcr.io/floating0516/sub2api-tocreate@sha256:1234"
  local source_commit="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  local upstream_commit="bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

  mkdir -p "$status_dir" "$bluegreen_dir"
  CONTROL_DIR="$status_dir"
  STATUS_FILE="$CONTROL_DIR/status.json"
  REQUEST_FILE="$CONTROL_DIR/request.json"
  PROCESSING_FILE="$CONTROL_DIR/processing.json"
  BLUEGREEN_STATE_FILE="$bluegreen_dir/active.env"
  PROD_CONTAINER_NAME="sub2api"

  jq -n --arg image "$old_image" '{
    state: "completed",
    image: $image,
    steps: [
      {id: "source_check", status: "completed"},
      {id: "production_approval", status: "completed"}
    ]
  }' > "$STATUS_FILE"
  printf '%s\n' \
    'ACTIVE_CONTAINER=sub2api' \
    "ACTIVE_IMAGE=$expected_prod_image" \
    'PHASE=active' \
    'UPDATED_AT=2026-08-01T16:04:17Z' > "$BLUEGREEN_STATE_FILE"

  production_image() { printf '%s\n' "$expected_prod_image"; }
  production_container_health() { printf 'healthy\n'; }
  production_endpoints_are_ready() { return 0; }
  image_digest() { printf '%s\n' "$prod_digest"; }
  source_commit_from_release_tag() { printf '%s\n' "$source_commit"; }
  upstream_commit_for_release() { printf '%s\n' "$upstream_commit"; }
  publish_current_github_release() {
    reset_release_metadata
    current_release_status="published"
    current_release_tag="tocreate-v0.1.169-tc1.24"
    current_release_url="https://github.com/floating0516/sub2api-ToCreate/releases/tag/$current_release_tag"
    current_release_published_at="2026-08-02T06:11:04Z"
  }

  reconcile_completed_status_with_production
  assert_eq "$expected_prod_image" "$(jq -r '.image' "$STATUS_FILE")" \
    "runtime image was not synchronized"
  assert_eq "$prod_digest" "$(jq -r '.image_digest' "$STATUS_FILE")" \
    "runtime digest was not synchronized"
  assert_eq "0.1.169" "$(jq -r '.app_version' "$STATUS_FILE")" \
    "application version was not derived from the image"
  assert_eq "$source_commit" "$(jq -r '.source_commit' "$STATUS_FILE")" \
    "release source commit was not synchronized"
  assert_eq "$upstream_commit" "$(jq -r '.upstream_commit' "$STATUS_FILE")" \
    "upstream commit was not synchronized"
  assert_eq "2026-08-01T16:04:17Z" "$(jq -r '.completed_at' "$STATUS_FILE")" \
    "blue-green activation time was not preserved"

  jq --arg digest "${prod_digest}stale" '.image_digest = $digest' \
    "$STATUS_FILE" > "$status_dir/stale-digest.json"
  mv "$status_dir/stale-digest.json" "$STATUS_FILE"
  reconcile_completed_status_with_production 1
  assert_eq "$prod_digest" "$(jq -r '.image_digest' "$STATUS_FILE")" \
    "forced startup reconciliation did not repair a stale digest"

  jq -n --arg image "$old_image" '{
    state: "awaiting_approval",
    image: $image,
    steps: [{id: "production_approval", status: "action_required"}]
  }' > "$STATUS_FILE"
  reconcile_completed_status_with_production
  assert_eq "$old_image" "$(jq -r '.image' "$STATUS_FILE")" \
    "active update status was overwritten"
  assert_eq "awaiting_approval" "$(jq -r '.state' "$STATUS_FILE")" \
    "active update state was overwritten"
)

test_release_publication_metadata() (
  local release_dir="$TEST_ROOT/release-metadata"
  local fake_publisher="$release_dir/publisher.sh"
  local expected_tag="tocreate-v0.1.200-tc2.1"
  local expected_url="https://github.com/floating0516/sub2api-ToCreate/releases/tag/$expected_tag"

  mkdir -p "$release_dir/logs"
  LOG_DIR="$release_dir/logs"
  current_log_file="release-test.log"
  RELEASE_SCRIPT="$fake_publisher"
  current_image="ghcr.io/floating0516/sub2api-tocreate:0.1.200-tc2.1"
  current_image_digest="ghcr.io/floating0516/sub2api-tocreate@sha256:$(printf 'a%.0s' {1..64})"
  current_source_commit="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  current_upstream_commit="bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

  printf '%s\n' \
    '#!/usr/bin/env bash' \
    "printf '%s\\n' '{\"status\":\"published\",\"tag\":\"$expected_tag\",\"url\":\"$expected_url\",\"published_at\":\"2026-08-02T07:00:00Z\"}'" \
    > "$fake_publisher"
  chmod 0755 "$fake_publisher"

  publish_current_github_release '2026-08-02T07:00:00Z'
  assert_eq "published" "$current_release_status" \
    "controller did not retain the published release status"
  assert_eq "$expected_tag" "$current_release_tag" \
    "controller did not retain the release tag"
  assert_eq "$expected_url" "$current_release_url" \
    "controller did not retain the release URL"

  printf '%s\n' '#!/usr/bin/env bash' 'exit 1' > "$fake_publisher"
  chmod 0755 "$fake_publisher"
  if publish_current_github_release '2026-08-02T07:00:00Z'; then
    fail "controller accepted a failed GitHub Release publication"
  fi
  assert_eq "failed" "$current_release_status" \
    "controller did not retain the failed release status"
  [ -n "$current_release_error" ] || fail "controller omitted the release publication error"
)

test_merge_conflict_pauses_without_source_changes
test_retired_resolution_action_is_rejected
test_completed_status_runtime_reconciliation
test_release_publication_metadata
printf 'tocreate update controller tests passed\n'
