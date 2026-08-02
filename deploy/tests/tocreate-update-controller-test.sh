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

test_runtime_resolver_config_loading() {
  local original_runtime_config_file="$CONFLICT_RESOLVER_RUNTIME_CONFIG_FILE"
  local original_runtime_api_key_file="$CONFLICT_RESOLVER_RUNTIME_API_KEY_FILE"
  local original_base_url="$CONFLICT_RESOLVER_BASE_URL"
  local original_internal_base_url="$CONFLICT_RESOLVER_INTERNAL_BASE_URL"
  local original_internal_match_base_url="$CONFLICT_RESOLVER_INTERNAL_MATCH_BASE_URL"
  local original_model="$CONFLICT_RESOLVER_MODEL"
  local original_reasoning_effort="$CONFLICT_RESOLVER_REASONING_EFFORT"
  local original_api_key_file="$CONFLICT_RESOLVER_API_KEY_FILE"
  local config_dir="$TEST_ROOT/runtime-config"

  mkdir -p "$config_dir"
  CONFLICT_RESOLVER_RUNTIME_CONFIG_FILE="$config_dir/resolver-config.json"
  CONFLICT_RESOLVER_RUNTIME_API_KEY_FILE="$config_dir/resolver-api-key"
  jq -n '{
    base_url: "https://gateway.example.com/v1",
    model: "gpt-5.6-terra",
    reasoning_effort: "max"
  }' > "$CONFLICT_RESOLVER_RUNTIME_CONFIG_FILE"
  printf 'test-api-key\n' > "$CONFLICT_RESOLVER_RUNTIME_API_KEY_FILE"
  chmod 0600 "$CONFLICT_RESOLVER_RUNTIME_API_KEY_FILE"

  load_conflict_resolver_config
  CONFLICT_RESOLVER_INTERNAL_BASE_URL="http://127.0.0.1:8080"
  CONFLICT_RESOLVER_INTERNAL_MATCH_BASE_URL="https://gateway.example.com/v1"
  validate_resolver_config
  assert_eq \
    "https://gateway.example.com/v1" \
    "$CONFLICT_RESOLVER_BASE_URL" \
    "runtime resolver base URL was not loaded"
  assert_eq \
    "gpt-5.6-terra" \
    "$CONFLICT_RESOLVER_MODEL" \
    "runtime resolver model was not loaded"
  assert_eq \
    "$CONFLICT_RESOLVER_RUNTIME_API_KEY_FILE" \
    "$CONFLICT_RESOLVER_API_KEY_FILE" \
    "runtime resolver API key file was not selected"
  assert_eq \
    "http://127.0.0.1:8080" \
    "$(conflict_resolver_request_base_url)" \
    "internal resolver transport was not selected"

  CONFLICT_RESOLVER_INTERNAL_MATCH_BASE_URL="https://another.example.com"
  assert_eq \
    "https://gateway.example.com/v1" \
    "$(conflict_resolver_request_base_url)" \
    "internal resolver transport ignored its configured match URL"
  CONFLICT_RESOLVER_INTERNAL_MATCH_BASE_URL="https://gateway.example.com/v1"

  CONFLICT_RESOLVER_INTERNAL_BASE_URL="http://127.0.0.1:8080@external.example.com"
  if validate_resolver_config; then
    fail "resolver accepted a non-loopback internal transport URL"
  fi
  CONFLICT_RESOLVER_INTERNAL_BASE_URL="http://127.0.0.1:8080"

  jq '.base_url = "http://insecure.example.com"' \
    "$CONFLICT_RESOLVER_RUNTIME_CONFIG_FILE" > "$config_dir/invalid.json"
  mv "$config_dir/invalid.json" "$CONFLICT_RESOLVER_RUNTIME_CONFIG_FILE"
  load_conflict_resolver_config
  if validate_resolver_config; then
    fail "resolver accepted an insecure runtime base URL"
  fi

  CONFLICT_RESOLVER_RUNTIME_CONFIG_FILE="$original_runtime_config_file"
  CONFLICT_RESOLVER_RUNTIME_API_KEY_FILE="$original_runtime_api_key_file"
  CONFLICT_RESOLVER_BASE_URL="$original_base_url"
  CONFLICT_RESOLVER_INTERNAL_BASE_URL="$original_internal_base_url"
  CONFLICT_RESOLVER_INTERNAL_MATCH_BASE_URL="$original_internal_match_base_url"
  CONFLICT_RESOLVER_MODEL="$original_model"
  CONFLICT_RESOLVER_REASONING_EFFORT="$original_reasoning_effort"
  CONFLICT_RESOLVER_API_KEY_FILE="$original_api_key_file"
  resolution_error=""
}

test_structured_response_validation() {
  local response_file="$TEST_ROOT/response.json"
  local resolution_file="$TEST_ROOT/resolution.json"
  local resolution_text=""

  resolution_text="$(jq -cn '{
    summary: "Preserved the custom validation and incorporated the official field.",
    risk_level: "medium",
    warnings: [],
    files: [{
      path: "frontend/src/example.ts",
      action: "write",
      content: "export const value = 1\n",
      rationale: "Combined both sides"
    }]
  }')"
  jq -n --arg text "$resolution_text" '{
    status: "completed",
    output: [
      {type: "reasoning", summary: []},
      {type: "message", content: [{type: "output_text", text: $text}]}
    ]
  }' > "$response_file"

  extract_conflict_resolution "$response_file" "$resolution_file"
  current_resolution_risk_level="low"
  current_resolution_warnings='[]'
  validate_conflict_resolution "$resolution_file" "frontend/src/example.ts"
  assert_eq "medium" "$current_resolution_risk_level" "model risk should raise local risk"

  if validate_conflict_resolution "$resolution_file" "frontend/src/other.ts"; then
    fail "validator accepted a changed path set"
  fi
  if conflict_path_is_safe "pnpm-lock.yaml"; then
    fail "lockfiles must not be model-edited"
  fi
  if conflict_path_is_safe "deploy/.env.production"; then
    fail "environment files must not be sent to the resolver"
  fi

  jq '.files[0].content = "invalid\u0000content"' "$resolution_file" > "$response_file"
  if validate_conflict_resolution "$response_file" "frontend/src/example.ts"; then
    fail "validator accepted NUL content"
  fi
}

test_resolution_status_metadata() {
  local status_dir="$TEST_ROOT/status"

  mkdir -p "$status_dir"
  CONTROL_DIR="$status_dir"
  STATUS_FILE="$CONTROL_DIR/status.json"
  current_action="stage"
  current_request_id="0123456789abcdef0123456789abcdef"
  current_started_at="2026-07-31T00:00:00Z"
  current_resolution_id="$current_request_id"
  current_conflict_files='["frontend/src/example.ts"]'
  current_resolution_summary="Combined the custom and official behavior."
  current_resolution_risk_level="medium"
  current_resolution_warnings='["Review the API contract."]'
  current_resolution_diff_stat="1 file changed, 2 insertions(+), 1 deletion(-)"
  current_resolver_model="gpt-5.6-luna"
  initialize_stage_steps
  set_step_status "conflict_resolution" "action_required"

  write_status "resolution_ready" "Review the proposed resolution"
  assert_eq \
    "$current_resolution_id" \
    "$(jq -r '.resolution_id' "$STATUS_FILE")" \
    "status omitted the resolution ID"
  assert_eq \
    "frontend/src/example.ts" \
    "$(jq -r '.conflict_files[0]' "$STATUS_FILE")" \
    "status omitted conflict files"
  assert_eq \
    "gpt-5.6-luna" \
    "$(jq -r '.resolver_model' "$STATUS_FILE")" \
    "status omitted the resolver model"
}

test_isolated_merge_resolution_commit() {
  local request_id="0123456789abcdef0123456789abcdef"
  local repo="$TEST_ROOT/state/worktrees/$request_id"
  local proposal_branch="tocreate/official-merge-$request_id"
  local base_commit=""
  local upstream_commit=""
  local resolution_file="$TEST_ROOT/git-resolution.json"
  local request_dir="$TEST_ROOT/request"
  local request_file="$request_dir/request.json"

  mkdir -p "$repo" "$TEST_ROOT/state/logs" "$TEST_ROOT/state/worktrees"
  git -C "$repo" init -q
  git -C "$repo" config user.name "Controller Test"
  git -C "$repo" config user.email "controller-test@example.invalid"
  printf 'mode=base\n' > "$repo/config.txt"
  git -C "$repo" add config.txt
  git -C "$repo" commit -qm base
  git -C "$repo" branch official

  git -C "$repo" checkout -qb "$proposal_branch"
  printf 'mode=custom\n' > "$repo/config.txt"
  git -C "$repo" commit -qam custom
  base_commit="$(git -C "$repo" rev-parse HEAD)"

  git -C "$repo" checkout -q official
  printf 'mode=official\n' > "$repo/config.txt"
  git -C "$repo" commit -qam official
  upstream_commit="$(git -C "$repo" rev-parse HEAD)"
  git -C "$repo" checkout -q "$proposal_branch"
  if git -C "$repo" merge --no-edit "$upstream_commit" >/dev/null 2>&1; then
    fail "test repository did not create a merge conflict"
  fi

  jq -n '{
    summary: "Kept the custom mode and documented the official option.",
    risk_level: "low",
    warnings: [],
    files: [{
      path: "config.txt",
      action: "write",
      content: "mode=custom\nofficial_mode=available\n",
      rationale: "Preserve the deployment default while retaining the official capability"
    }]
  }' > "$resolution_file"

  SRC_DIR="$repo"
  STATE_DIR="$TEST_ROOT/state"
  LOG_DIR="$STATE_DIR/logs"
  MERGE_WORKTREE_ROOT="$STATE_DIR/worktrees"
  RESOLUTION_CONTEXT_FILE="$STATE_DIR/resolution-context.json"
  current_log_file="controller-test.log"
  current_resolution_id="$request_id"
  current_upstream_commit="$upstream_commit"
  current_resolution_risk_level="low"
  current_resolution_warnings='[]'

  mkdir -p "$request_dir"
  write_resolution_context \
    "$request_id" "$repo" "$proposal_branch" "$base_commit" "$upstream_commit"
  build_conflict_resolver_request "$repo" "$request_file" "$request_dir" config.txt
  assert_eq "gpt-5.6-luna" "$(jq -r '.model' "$request_file")" "resolver model is incorrect"
  assert_eq "max" "$(jq -r '.reasoning.effort' "$request_file")" "reasoning effort is incorrect"
  assert_eq \
    "json_schema" \
    "$(jq -r '.text.format.type' "$request_file")" \
    "structured output format is missing"

  validate_conflict_resolution "$resolution_file" config.txt
  apply_conflict_resolution \
    "$repo" "$base_commit" "$proposal_branch" "$resolution_file" config.txt

  assert_eq "mode=custom" "$(sed -n '1p' "$repo/config.txt")" "custom behavior was not preserved"
  assert_eq "official_mode=available" "$(sed -n '2p' "$repo/config.txt")" "official behavior was not incorporated"
  [ -z "$(git -C "$repo" status --porcelain)" ] || fail "proposal worktree is dirty"
  resolution_context_is_valid || fail "proposal context is invalid"
  assert_eq \
    "$(git -C "$repo" rev-parse HEAD)" \
    "$(jq -r '.proposal_commit' "$RESOLUTION_CONTEXT_FILE")" \
    "proposal commit was not persisted"
}

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
  RESOLUTION_CONTEXT_FILE="$TEST_ROOT/runtime-resolution-context.json"
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

test_runtime_resolver_config_loading
test_structured_response_validation
test_resolution_status_metadata
test_isolated_merge_resolution_commit
test_completed_status_runtime_reconciliation
test_release_publication_metadata
printf 'tocreate update controller tests passed\n'
