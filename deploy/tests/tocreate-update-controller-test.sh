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

test_runtime_resolver_config_loading
test_structured_response_validation
test_resolution_status_metadata
test_isolated_merge_resolution_commit
printf 'tocreate update controller tests passed\n'
