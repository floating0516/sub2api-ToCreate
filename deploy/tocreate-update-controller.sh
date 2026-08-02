#!/usr/bin/env bash
set -uo pipefail

# Host-side controller for the ToCreate custom update UI. The application can
# only enqueue fixed staging and promotion actions through a shared
# directory; all Git, GitHub Actions, Docker, validation, and rollback work
# remains outside the application container.

SRC_DIR="${SRC_DIR:-/home/ubuntu/sub2api-src}"
DEPLOY_DIR="${DEPLOY_DIR:-/home/ubuntu/sub2api-deploy}"
BRANCH="${BRANCH:-custom/subscription-quota-window}"
UPSTREAM_REMOTE="${UPSTREAM_REMOTE:-upstream}"
UPSTREAM_REF="${UPSTREAM_REF:-main}"
CUSTOM_REPO="${CUSTOM_REPO:-floating0516/sub2api-ToCreate}"
WORKFLOW="${WORKFLOW:-Custom Docker Image}"
IMAGE_REPO="${IMAGE_REPO:-ghcr.io/floating0516/sub2api-tocreate}"
CONTROL_DIR="${CONTROL_DIR:-$DEPLOY_DIR/data/custom-update}"
STATE_DIR="${STATE_DIR:-$DEPLOY_DIR/state/custom-update}"
POLL_INTERVAL_SECONDS="${POLL_INTERVAL_SECONDS:-2}"
RUNTIME_SYNC_INTERVAL_SECONDS="${RUNTIME_SYNC_INTERVAL_SECONDS:-30}"
LOCK_FILE="${LOCK_FILE:-/tmp/sub2api-custom-update-controller.lock}"
UPDATE_SCRIPT="${UPDATE_SCRIPT:-$DEPLOY_DIR/update-custom-sub2api.sh}"
STAGING_SCRIPT="${STAGING_SCRIPT:-$DEPLOY_DIR/deploy-custom-sub2api-staging.sh}"
PROMOTE_SCRIPT="${PROMOTE_SCRIPT:-$DEPLOY_DIR/promote-custom-sub2api.sh}"
RELEASE_SCRIPT="${RELEASE_SCRIPT:-$SRC_DIR/deploy/publish-tocreate-release.sh}"
STAGING_CONTAINER_NAME="${STAGING_CONTAINER_NAME:-sub2api-test}"
PROD_CONTAINER_NAME="${PROD_CONTAINER_NAME:-sub2api}"
STAGING_BASE_URL="${STAGING_BASE_URL:-http://127.0.0.1:18080}"
PROD_BASE_URL="${PROD_BASE_URL:-http://127.0.0.1:8080}"
BLUEGREEN_STATE_FILE="${BLUEGREEN_STATE_FILE:-$DEPLOY_DIR/state/blue-green/active.env}"
MERGE_WORKTREE_ROOT="${MERGE_WORKTREE_ROOT:-$STATE_DIR/merge-worktrees}"

REQUEST_FILE="$CONTROL_DIR/request.json"
PROCESSING_FILE="$CONTROL_DIR/processing.json"
STATUS_FILE="$CONTROL_DIR/status.json"
HEARTBEAT_FILE="$CONTROL_DIR/heartbeat"
LOG_DIR="$STATE_DIR/logs"
REQUEST_ARCHIVE_DIR="$STATE_DIR/requests"

current_action=""
current_request_id=""
current_message=""
current_image=""
current_image_digest=""
current_app_version=""
current_upstream_commit=""
current_source_commit=""
current_started_at=""
current_log_file=""
chosen_suffix=""
previous_state=""
previous_image=""
previous_source_commit=""
previous_steps="[]"
current_steps="[]"
active_stage_step=""
heartbeat_pid=""
current_conflict_files="[]"
current_release_status=""
current_release_tag=""
current_release_url=""
current_release_published_at=""
current_release_error=""
previous_release_status=""
previous_release_tag=""
previous_release_url=""
previous_release_published_at=""
previous_release_error=""

log() {
  printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    log "Missing required command: $1"
    exit 1
  }
}

utc_now() {
  date -u '+%Y-%m-%dT%H:%M:%SZ'
}

write_status() {
  local state="$1"
  local message="$2"
  local error_message="${3:-}"
  local completed_at="${4:-}"
  local temporary=""

  temporary="$(mktemp "$CONTROL_DIR/.status.XXXXXX")" || return 1
  if ! jq -n \
    --arg state "$state" \
    --arg action "$current_action" \
    --arg request_id "$current_request_id" \
    --arg message "$message" \
    --arg image "$current_image" \
    --arg image_digest "$current_image_digest" \
    --arg app_version "$current_app_version" \
    --arg upstream_commit "$current_upstream_commit" \
    --arg source_commit "$current_source_commit" \
    --arg started_at "$current_started_at" \
    --arg updated_at "$(utc_now)" \
    --arg completed_at "$completed_at" \
    --arg error "$error_message" \
    --arg log_file "$current_log_file" \
    --arg staging_url "$STAGING_BASE_URL" \
    --arg production_url "$PROD_BASE_URL" \
    --arg release_status "$current_release_status" \
    --arg release_tag "$current_release_tag" \
    --arg release_url "$current_release_url" \
    --arg release_published_at "$current_release_published_at" \
    --arg release_error "$current_release_error" \
    --argjson steps "$current_steps" \
    --argjson conflict_files "$current_conflict_files" \
    '{
      state: $state,
      action: $action,
      request_id: $request_id,
      message: $message,
      image: $image,
      image_digest: $image_digest,
      app_version: $app_version,
      upstream_commit: $upstream_commit,
      source_commit: $source_commit,
      started_at: $started_at,
      updated_at: $updated_at,
      completed_at: $completed_at,
      error: $error,
      log_file: $log_file,
      staging_url: $staging_url,
      production_url: $production_url,
      steps: $steps,
      conflict_files: $conflict_files,
      release_status: $release_status,
      release_tag: $release_tag,
      release_url: $release_url,
      release_published_at: $release_published_at,
      release_error: $release_error
    } | with_entries(select(.value != "" and .value != []))' > "$temporary"; then
    rm -f -- "$temporary"
    return 1
  fi
  chmod 0644 "$temporary"
  mv -f -- "$temporary" "$STATUS_FILE"
}

status_value() {
  local expression="$1"
  if [ ! -f "$STATUS_FILE" ]; then
    return 0
  fi
  jq -r "$expression // empty" "$STATUS_FILE" 2>/dev/null
}

initialize_stage_steps() {
  current_steps='[
    {"id":"source_check","status":"pending"},
    {"id":"upstream_fetch","status":"pending"},
    {"id":"upstream_merge","status":"pending"},
    {"id":"conflict_resolution","status":"pending"},
    {"id":"source_push","status":"pending"},
    {"id":"image_build","status":"pending"},
    {"id":"staging_deploy","status":"pending"},
    {"id":"staging_validate","status":"pending"},
    {"id":"production_approval","status":"pending"}
  ]'
  active_stage_step=""
}

set_step_status() {
  local step_id="$1"
  local step_status="$2"
  local updated_steps=""

  updated_steps="$(
    jq -c \
      --arg step_id "$step_id" \
      --arg step_status "$step_status" \
      'map(if .id == $step_id then .status = $step_status else . end)' \
      <<<"$current_steps"
  )" || return 1
  current_steps="$updated_steps"
}

begin_stage_step() {
  active_stage_step="$1"
  set_step_status "$active_stage_step" "running"
}

complete_stage_step() {
  local step_id="$1"
  set_step_status "$step_id" "completed" || return 1
  if [ "$active_stage_step" = "$step_id" ]; then
    active_stage_step=""
  fi
}

skip_stage_step() {
  set_step_status "$1" "skipped"
}

restore_or_initialize_stage_steps() {
  current_steps="$previous_steps"
  if ! jq -e 'type == "array" and length > 0' <<<"$current_steps" >/dev/null 2>&1; then
    initialize_stage_steps
    set_step_status "source_check" "completed"
    set_step_status "upstream_fetch" "completed"
    set_step_status "upstream_merge" "completed"
    set_step_status "conflict_resolution" "skipped"
    set_step_status "source_push" "completed"
    set_step_status "image_build" "completed"
    set_step_status "staging_deploy" "completed"
    set_step_status "staging_validate" "completed"
    set_step_status "production_approval" "action_required"
  fi
  active_stage_step=""
}

write_failure() {
  local message="$1"
  if [ -n "$active_stage_step" ]; then
    set_step_status "$active_stage_step" "failed" || true
    active_stage_step=""
  fi
  current_message="Custom update failed"
  write_status "failed" "$current_message" "$message" "$(utc_now)"
  log "$message"
  return 1
}

reset_conflict_metadata() {
  current_conflict_files="[]"
}

reset_release_metadata() {
  current_release_status=""
  current_release_tag=""
  current_release_url=""
  current_release_published_at=""
  current_release_error=""
}

restore_release_metadata() {
  current_release_status="$previous_release_status"
  current_release_tag="$previous_release_tag"
  current_release_url="$previous_release_url"
  current_release_published_at="$previous_release_published_at"
  current_release_error="$previous_release_error"
}

run_logged() {
  "$@" 2>&1 | tee -a "$LOG_DIR/$current_log_file"
  return "${PIPESTATUS[0]}"
}

heartbeat_loop() {
  while true; do
    touch "$HEARTBEAT_FILE"
    sleep 3
  done
}

cleanup() {
  if [ -n "$heartbeat_pid" ]; then
    kill "$heartbeat_pid" >/dev/null 2>&1 || true
    wait "$heartbeat_pid" >/dev/null 2>&1 || true
  fi
}

shutdown() {
  exit 0
}

source_is_clean() {
  [ -z "$(git -C "$SRC_DIR" status --porcelain)" ]
}

cleanup_merge_worktree() {
  local worktree="$1"
  local branch="$2"
  local cleanup_failed=0

  if [ -e "$worktree" ]; then
    if ! git -C "$SRC_DIR" worktree remove --force "$worktree" >/dev/null 2>&1; then
      cleanup_failed=1
    fi
  else
    git -C "$SRC_DIR" worktree prune >/dev/null 2>&1 || true
  fi
  if git -C "$SRC_DIR" show-ref --verify --quiet "refs/heads/$branch"; then
    if ! git -C "$SRC_DIR" branch -D "$branch" >/dev/null 2>&1; then
      cleanup_failed=1
    fi
  fi
  [ "$cleanup_failed" -eq 0 ]
}

create_merge_worktree() {
  local base_commit="$1"
  local worktree="$2"
  local branch="$3"

  mkdir -p "$MERGE_WORKTREE_ROOT" || return 1
  if ! run_logged git -C "$SRC_DIR" worktree add -b "$branch" "$worktree" "$base_commit"; then
    cleanup_merge_worktree "$worktree" "$branch" || true
    return 1
  fi
}

record_merge_conflicts() {
  local worktree="$1"
  local branch="$2"
  local conflict_json=""
  shift 2

  conflict_json="$(printf '%s\0' "$@" | jq -Rs 'split("\u0000") | map(select(length > 0))')" || return 1
  current_conflict_files="$conflict_json"
  set_step_status "conflict_resolution" "action_required" || return 1
  active_stage_step=""
  if ! cleanup_merge_worktree "$worktree" "$branch"; then
    write_failure "Could not clean up the isolated merge after conflicts were detected"
    return 1
  fi

  current_message="Merge conflicts require manual resolution; source and production were not changed"
  write_status "conflict_detected" "$current_message" "" "$(utc_now)" || return 1
  log "Paused the official update for $# conflicted file(s); no model was called"
}

production_image() {
  sudo docker inspect "$PROD_CONTAINER_NAME" --format '{{.Config.Image}}' 2>/dev/null
}

increment_custom_suffix() {
  local suffix="$1"
  if [[ "$suffix" =~ ^tc([0-9]+)\.([0-9]+)$ ]]; then
    printf 'tc%s.%s.1\n' "${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}"
    return
  fi
  if [[ "$suffix" =~ ^tc([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    printf 'tc%s.%s.%s\n' \
      "${BASH_REMATCH[1]}" \
      "${BASH_REMATCH[2]}" \
      "$((BASH_REMATCH[3] + 1))"
    return
  fi
  return 1
}

initial_custom_suffix() {
  local app_version="$1"
  local prod_image="$2"
  local prod_tag="${prod_image##*:}"
  local prod_version=""
  local custom_major=""
  local custom_minor=""
  local custom_patch=""

  if [[ "$prod_tag" =~ ^([0-9]+\.[0-9]+\.[0-9]+)-tc([0-9]+)\.([0-9]+)(\.([0-9]+))?$ ]]; then
    prod_version="${BASH_REMATCH[1]}"
    custom_major="${BASH_REMATCH[2]}"
    custom_minor="${BASH_REMATCH[3]}"
    custom_patch="${BASH_REMATCH[5]:-0}"
    if [ "$app_version" != "$prod_version" ]; then
      printf 'tc%s.%s\n' "$custom_major" "$((custom_minor + 1))"
      return
    fi
    printf 'tc%s.%s.%s\n' "$custom_major" "$custom_minor" "$((custom_patch + 1))"
    return
  fi

  printf '%s\n' "${CUSTOM_VERSION_SEED:-tc1.1}"
}

choose_image() {
  local prod_image="$1"
  local suffix=""
  local candidate=""
  local attempts=0

  suffix="$(initial_custom_suffix "$current_app_version" "$prod_image")" || return 1
  while [ "$attempts" -lt 50 ]; do
    candidate="${IMAGE_REPO}:${current_app_version}-${suffix}"
    if ! sudo docker manifest inspect "$candidate" >/dev/null 2>&1; then
      current_image="$candidate"
      chosen_suffix="$suffix"
      return 0
    fi
    suffix="$(increment_custom_suffix "$suffix")" || return 1
    attempts=$((attempts + 1))
  done
  return 1
}

image_digest() {
  local image="$1"
  sudo docker image inspect "$image" --format '{{json .RepoDigests}}' 2>/dev/null \
    | jq -r --arg repo "$IMAGE_REPO" \
      '[.[] | select(startswith($repo + "@"))][0] // empty' 2>/dev/null
}

production_container_health() {
  sudo docker inspect "$PROD_CONTAINER_NAME" \
    --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' \
    2>/dev/null
}

production_endpoints_are_ready() {
  curl -fsS "$PROD_BASE_URL/health" >/dev/null \
    && curl -fsS "$PROD_BASE_URL/ready" >/dev/null
}

bluegreen_state_value() {
  local key="$1"
  [ -f "$BLUEGREEN_STATE_FILE" ] || return 0
  awk -F= -v key="$key" '$1 == key {sub(/^[^=]*=/, ""); print; exit}' \
    "$BLUEGREEN_STATE_FILE"
}

bluegreen_state_matches_production() {
  local image="$1"
  local phase=""
  local active_container=""
  local active_image=""

  [ -f "$BLUEGREEN_STATE_FILE" ] || return 0
  phase="$(bluegreen_state_value PHASE)"
  active_container="$(bluegreen_state_value ACTIVE_CONTAINER)"
  active_image="$(bluegreen_state_value ACTIVE_IMAGE)"
  [ "$phase" = "active" ] \
    && [ "$active_container" = "$PROD_CONTAINER_NAME" ] \
    && [ "$active_image" = "$image" ]
}

app_version_from_image() {
  local image="$1"
  local tag=""

  case "$image" in
    "$IMAGE_REPO":*) tag="${image#"$IMAGE_REPO:"}" ;;
    *) return 1 ;;
  esac
  if [[ "$tag" =~ ^([0-9]+\.[0-9]+\.[0-9]+)-tc[0-9]+\.[0-9]+(\.[0-9]+)?$ ]]; then
    printf '%s\n' "${BASH_REMATCH[1]}"
    return 0
  fi
  return 1
}

source_commit_from_release_tag() {
  local image="$1"
  local image_digest_value="$2"
  local image_tag="${image#"$IMAGE_REPO:"}"
  local release_tag="tocreate-v$image_tag"
  local recorded_digest=""
  local source_commit=""

  recorded_digest="$(
    git -C "$SRC_DIR" for-each-ref \
      --format='%(contents)' "refs/tags/$release_tag" 2>/dev/null \
      | sed -n 's/^OCI digest: //p' \
      | head -n 1
  )"
  [ -n "$recorded_digest" ] || return 0
  [ "$recorded_digest" = "${image_digest_value#*@}" ] || return 0

  source_commit="$(
    git -C "$SRC_DIR" rev-parse --verify "$release_tag^{commit}" 2>/dev/null
  )"
  [[ "$source_commit" =~ ^[0-9a-f]{40}$ ]] || return 0
  if ! git -C "$SRC_DIR" merge-base --is-ancestor "$source_commit" "$BRANCH" 2>/dev/null; then
    return 0
  fi
  printf '%s\n' "$source_commit"
}

upstream_commit_for_release() {
  local app_version="$1"
  local source_commit="$2"
  local upstream_commit=""

  [ -n "$source_commit" ] || return 0
  upstream_commit="$(
    git -C "$SRC_DIR" rev-parse --verify "v$app_version^{commit}" 2>/dev/null
  )"
  [[ "$upstream_commit" =~ ^[0-9a-f]{40}$ ]] || return 0
  if ! git -C "$SRC_DIR" merge-base --is-ancestor \
    "$upstream_commit" "$source_commit" 2>/dev/null; then
    return 0
  fi
  printf '%s\n' "$upstream_commit"
}

publish_current_github_release() {
  local deployed_at="$1"
  local image_tag="${current_image#"$IMAGE_REPO:"}"
  local expected_tag="tocreate-v$image_tag"
  local expected_url="https://github.com/$CUSTOM_REPO/releases/tag/$expected_tag"
  local release_json=""
  local release_log="$LOG_DIR/${current_log_file:-release-publication.log}"

  reset_release_metadata
  if [ ! -x "$RELEASE_SCRIPT" ]; then
    current_release_status="failed"
    current_release_error="GitHub Release publisher is unavailable"
    log "$current_release_error"
    return 1
  fi

  if ! release_json="$(
    "$RELEASE_SCRIPT" \
      "$current_image" \
      "$current_image_digest" \
      "$current_source_commit" \
      "$current_upstream_commit" \
      "$deployed_at" 2>>"$release_log"
  )"; then
    current_release_status="failed"
    current_release_error="GitHub Release publication failed; review the promotion log"
    log "$current_release_error"
    return 1
  fi
  if ! jq -e \
    --arg tag "$expected_tag" \
    --arg url "$expected_url" \
    '.status == "published" and .tag == $tag and .url == $url and
     (.published_at | type == "string" and length > 0)' \
    <<<"$release_json" >/dev/null 2>&1; then
    current_release_status="failed"
    current_release_error="GitHub Release publisher returned invalid metadata"
    log "$current_release_error"
    return 1
  fi

  current_release_status="published"
  current_release_tag="$(jq -r '.tag' <<<"$release_json")"
  current_release_url="$(jq -r '.url' <<<"$release_json")"
  current_release_published_at="$(jq -r '.published_at' <<<"$release_json")"
  current_release_error=""
  log "Published GitHub Release $current_release_tag"
}

reconcile_completed_status_with_production() {
  local verify_digest="${1:-0}"
  local status_state=""
  local status_image=""
  local status_image_digest=""
  local status_source_commit=""
  local status_upstream_commit=""
  local status_matches_production=0
  local active_image=""
  local prod_image=""
  local prod_image_digest=""
  local prod_health=""
  local synced_at=""

  [ ! -f "$REQUEST_FILE" ] || return 0
  [ ! -f "$PROCESSING_FILE" ] || return 0
  status_state="$(status_value '.state')"
  [ "$status_state" = "completed" ] || return 0

  status_image="$(status_value '.image')"
  status_image_digest="$(status_value '.image_digest')"
  if [ -f "$BLUEGREEN_STATE_FILE" ]; then
    [ "$(bluegreen_state_value PHASE)" = "active" ] || return 1
    [ "$(bluegreen_state_value ACTIVE_CONTAINER)" = "$PROD_CONTAINER_NAME" ] || return 1
    active_image="$(bluegreen_state_value ACTIVE_IMAGE)"
    [ -n "$active_image" ] || return 1
    if [ "$verify_digest" != "1" ] && [ "$active_image" = "$status_image" ]; then
      return 0
    fi
  fi
  prod_image="$(production_image)"
  [ -n "$prod_image" ] || return 1
  case "$prod_image" in
    "$IMAGE_REPO":*) ;;
    *) return 1 ;;
  esac
  prod_image_digest="$(image_digest "$prod_image")"
  [ -n "$prod_image_digest" ] || return 1
  if [ "$prod_image" = "$status_image" ] \
    && [ "$prod_image_digest" = "$status_image_digest" ]; then
    status_matches_production=1
    if [ "$verify_digest" != "1" ]; then
      return 0
    fi
    if [ "$(status_value '.release_status')" = "published" ] \
      && [ "$(status_value '.release_tag')" = "tocreate-v${prod_image#"$IMAGE_REPO:"}" ] \
      && [ "$(status_value '.release_url')" = \
        "https://github.com/$CUSTOM_REPO/releases/tag/tocreate-v${prod_image#"$IMAGE_REPO:"}" ] \
      && [ -n "$(status_value '.release_published_at')" ]; then
      return 0
    fi
  fi

  prod_health="$(production_container_health)"
  [ "$prod_health" = "healthy" ] || return 1
  bluegreen_state_matches_production "$prod_image" || return 1
  production_endpoints_are_ready || return 1

  current_action="promote"
  current_request_id=""
  current_message="Production status synchronized from the active runtime"
  current_image="$prod_image"
  current_image_digest="$prod_image_digest"
  current_app_version="$(app_version_from_image "$prod_image")"
  [ -n "$current_app_version" ] || return 1
  current_source_commit="$(
    source_commit_from_release_tag "$prod_image" "$current_image_digest"
  )"
  if [ -z "$current_source_commit" ] && [ "$status_matches_production" = "1" ]; then
    status_source_commit="$(status_value '.source_commit')"
    if [[ "$status_source_commit" =~ ^[0-9a-f]{40}$ ]] \
      && git -C "$SRC_DIR" cat-file -e "$status_source_commit^{commit}" 2>/dev/null \
      && git -C "$SRC_DIR" merge-base --is-ancestor \
        "$status_source_commit" "$BRANCH" 2>/dev/null; then
      current_source_commit="$status_source_commit"
    fi
  fi
  current_upstream_commit="$(
    upstream_commit_for_release "$current_app_version" "$current_source_commit"
  )"
  if [ -z "$current_upstream_commit" ] && [ "$status_matches_production" = "1" ]; then
    status_upstream_commit="$(status_value '.upstream_commit')"
    if [[ "$status_upstream_commit" =~ ^[0-9a-f]{40}$ ]] \
      && [ -n "$current_source_commit" ] \
      && git -C "$SRC_DIR" merge-base --is-ancestor \
        "$status_upstream_commit" "$current_source_commit" 2>/dev/null; then
      current_upstream_commit="$status_upstream_commit"
    fi
  fi
  current_steps="$(jq -c '.steps // []' "$STATUS_FILE" 2>/dev/null || printf '[]')"
  if ! jq -e 'type == "array" and length > 0' <<<"$current_steps" >/dev/null 2>&1; then
    initialize_stage_steps
    current_steps="$(
      jq -c 'map(.status = if .id == "conflict_resolution" then "skipped" else "completed" end)' \
        <<<"$current_steps"
    )" || return 1
  else
    current_steps="$(
      jq -c 'map(if .id == "production_approval" then .status = "completed" else . end)' \
        <<<"$current_steps"
    )" || return 1
  fi
  active_stage_step=""
  reset_conflict_metadata
  reset_release_metadata
  current_log_file=""
  synced_at="$(bluegreen_state_value UPDATED_AT)"
  if [[ ! "$synced_at" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$ ]]; then
    synced_at="$(utc_now)"
  fi
  current_started_at="$synced_at"

  if publish_current_github_release "$synced_at"; then
    current_message="Production status and GitHub Release synchronized from the active runtime"
  else
    current_message="Production status synchronized; GitHub Release publication needs attention"
  fi
  write_status "completed" "$current_message" "" "$synced_at" || return 1
  log "Synchronized completed update status to $current_image"
}

continue_stage_after_merge() {
  local prod_image=""
  local suffix=""
  local staged_image=""
  local staged_health=""

  current_source_commit="$(git -C "$SRC_DIR" rev-parse HEAD 2>/dev/null)"
  current_app_version="$(tr -d '\r\n' < "$SRC_DIR/backend/cmd/server/VERSION")"
  current_app_version="${current_app_version#v}"
  prod_image="$(production_image)"
  if [ -z "$current_source_commit" ] || [ -z "$current_app_version" ] || [ -z "$prod_image" ]; then
    write_failure "Could not resolve source, version, or production image metadata"
    return 1
  fi
  if [ "$previous_state" = "completed" ] \
    && [ "$previous_source_commit" = "$current_source_commit" ] \
    && [ "$previous_image" = "$prod_image" ]; then
    restore_release_metadata
    current_image="$prod_image"
    current_image_digest="$(image_digest "$prod_image")"
    skip_stage_step "source_push"
    skip_stage_step "image_build"
    skip_stage_step "staging_deploy"
    skip_stage_step "staging_validate"
    skip_stage_step "production_approval"
    current_message="Production already uses the latest merged ToCreate source"
    write_status "completed" "$current_message" "" "$(utc_now)"
    return 0
  fi

  begin_stage_step "source_push"
  if ! choose_image "$prod_image"; then
    write_failure "Could not allocate the next ToCreate image version"
    return 1
  fi
  suffix="$chosen_suffix"

  current_message="Pushing the ToCreate branch"
  write_status "pushing" "$current_message" || return 1
  if ! run_logged git -C "$SRC_DIR" push origin "$BRANCH"; then
    write_failure "Could not push $BRANCH to the custom repository"
    return 1
  fi
  complete_stage_step "source_push" || return 1

  begin_stage_step "image_build"
  current_message="Building $current_image with GitHub Actions"
  write_status "building" "$current_message" || return 1
  if ! run_logged env \
    DEPLOY=0 \
    SRC_DIR="$SRC_DIR" \
    DEPLOY_DIR="$DEPLOY_DIR" \
    BRANCH="$BRANCH" \
    REPO="$CUSTOM_REPO" \
    WORKFLOW="$WORKFLOW" \
    IMAGE_REPO="$IMAGE_REPO" \
    CUSTOM_SUFFIX="$suffix" \
    "$UPDATE_SCRIPT" "$current_app_version" "${current_image##*:}"; then
    write_failure "GitHub Actions custom image build failed"
    return 1
  fi
  complete_stage_step "image_build" || return 1

  begin_stage_step "staging_deploy"
  current_message="Deploying the exact image to port 18080"
  write_status "staging" "$current_message" || return 1
  if ! run_logged "$STAGING_SCRIPT" "$current_image"; then
    write_failure "Staging deployment or validation failed; production was not changed"
    return 1
  fi
  complete_stage_step "staging_deploy" || return 1

  begin_stage_step "staging_validate"
  current_message="Validating the staged image"
  write_status "validating" "$current_message" || return 1
  staged_image="$(sudo docker inspect "$STAGING_CONTAINER_NAME" --format '{{.Config.Image}}' 2>/dev/null)"
  staged_health="$(sudo docker inspect "$STAGING_CONTAINER_NAME" --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' 2>/dev/null)"
  if [ "$staged_image" != "$current_image" ]; then
    write_failure "Staging image does not match the built image"
    return 1
  fi
  if [ "$staged_health" != "healthy" ] && [ "$staged_health" != "running" ]; then
    write_failure "Staging container is not healthy"
    return 1
  fi
  if ! curl -fsS "$STAGING_BASE_URL/health" >/dev/null; then
    write_failure "Staging health endpoint failed"
    return 1
  fi

  current_image_digest="$(image_digest "$current_image")"
  if [ -z "$current_image_digest" ]; then
    write_failure "Could not resolve the staged image digest"
    return 1
  fi
  complete_stage_step "staging_validate" || return 1

  set_step_status "production_approval" "action_required" || return 1
  current_message="Staging is healthy; explicit production approval is required"
  write_status "awaiting_approval" "$current_message" || return 1
  log "Staged $current_image ($current_image_digest) on port 18080"
}

stage_update() {
  local base_commit=""
  local merge_worktree=""
  local merge_branch=""
  local proposal_commit=""
  local -a conflict_paths=()

  current_action="stage"
  current_started_at="$(utc_now)"
  current_log_file="stage-${current_request_id}-$(date '+%Y%m%d%H%M%S').log"
  current_image=""
  current_image_digest=""
  current_app_version=""
  current_upstream_commit=""
  current_source_commit=""
  reset_conflict_metadata
  reset_release_metadata
  initialize_stage_steps
  begin_stage_step "source_check"
  current_message="Checking source and official upstream"
  write_status "checking" "$current_message" || return 1

  if ! source_is_clean; then
    git -C "$SRC_DIR" status --short | tee -a "$LOG_DIR/$current_log_file"
    write_failure "Source worktree is dirty; commit or resolve it before retrying"
    return 1
  fi
  if ! run_logged git -C "$SRC_DIR" checkout "$BRANCH"; then
    write_failure "Could not check out $BRANCH"
    return 1
  fi
  complete_stage_step "source_check" || return 1

  begin_stage_step "upstream_fetch"
  current_message="Fetching official upstream source"
  write_status "checking" "$current_message" || return 1
  if ! run_logged git -C "$SRC_DIR" fetch "$UPSTREAM_REMOTE" "$UPSTREAM_REF" --tags; then
    write_failure "Could not fetch $UPSTREAM_REMOTE/$UPSTREAM_REF"
    return 1
  fi
  current_upstream_commit="$(git -C "$SRC_DIR" rev-parse "$UPSTREAM_REMOTE/$UPSTREAM_REF" 2>/dev/null)"
  base_commit="$(git -C "$SRC_DIR" rev-parse HEAD 2>/dev/null)"
  current_source_commit="$base_commit"
  if [[ ! "$current_upstream_commit" =~ ^[0-9a-f]{40}$ ]] \
    || [[ ! "$base_commit" =~ ^[0-9a-f]{40}$ ]]; then
    write_failure "Could not resolve source or official upstream commits"
    return 1
  fi
  complete_stage_step "upstream_fetch" || return 1

  merge_worktree="$MERGE_WORKTREE_ROOT/$current_request_id"
  merge_branch="tocreate/official-merge-$current_request_id"
  if ! create_merge_worktree "$base_commit" "$merge_worktree" "$merge_branch"; then
    write_failure "Could not create the isolated official merge worktree"
    return 1
  fi

  begin_stage_step "upstream_merge"
  current_message="Merging official upstream in an isolated worktree"
  write_status "merging" "$current_message" || return 1
  if run_logged git -C "$merge_worktree" merge --no-edit "$current_upstream_commit"; then
    proposal_commit="$(git -C "$merge_worktree" rev-parse HEAD 2>/dev/null)"
    if [[ ! "$proposal_commit" =~ ^[0-9a-f]{40}$ ]]; then
      cleanup_merge_worktree "$merge_worktree" "$merge_branch" || true
      write_failure "Could not resolve the isolated merge commit"
      return 1
    fi
    complete_stage_step "upstream_merge" || return 1
    skip_stage_step "conflict_resolution" || return 1
    if [ "$(git -C "$SRC_DIR" rev-parse HEAD 2>/dev/null)" != "$base_commit" ] \
      || ! source_is_clean; then
      cleanup_merge_worktree "$merge_worktree" "$merge_branch" || true
      write_failure "Source branch changed while the isolated merge was running"
      return 1
    fi
    if ! run_logged git -C "$SRC_DIR" merge --ff-only "$proposal_commit"; then
      cleanup_merge_worktree "$merge_worktree" "$merge_branch" || true
      write_failure "Could not fast-forward the source branch to the isolated merge"
      return 1
    fi
    if ! cleanup_merge_worktree "$merge_worktree" "$merge_branch"; then
      write_failure "Could not clean up the isolated merge worktree"
      return 1
    fi
    continue_stage_after_merge
    return $?
  fi

  mapfile -d '' conflict_paths < <(
    git -C "$merge_worktree" diff --name-only --diff-filter=U -z
  )
  if [ "${#conflict_paths[@]}" -eq 0 ]; then
    cleanup_merge_worktree "$merge_worktree" "$merge_branch" || true
    write_failure "Official merge failed without resolvable file conflicts"
    return 1
  fi
  complete_stage_step "upstream_merge" || return 1
  record_merge_conflicts "$merge_worktree" "$merge_branch" "${conflict_paths[@]}"
}

promote_update() {
  local requested_image="$1"
  local staged_image=""
  local staged_health=""
  local prod_image=""
  local prod_image_digest=""
  local completed_at=""

  current_action="promote"
  current_started_at="$(utc_now)"
  current_log_file="promote-${current_request_id}-$(date '+%Y%m%d%H%M%S').log"
  restore_or_initialize_stage_steps
  reset_conflict_metadata
  restore_release_metadata
  set_step_status "production_approval" "completed" || return 1
  current_image="$previous_image"
  current_image_digest="$(status_value '.image_digest')"
  current_app_version="$(status_value '.app_version')"
  current_upstream_commit="$(status_value '.upstream_commit')"
  current_source_commit="$(status_value '.source_commit')"

  if [ "$previous_state" != "awaiting_approval" ] \
    || [ -z "$current_image" ] \
    || [ "$requested_image" != "$current_image" ]; then
    write_failure "Promotion request does not match an image awaiting approval"
    return 1
  fi

  staged_image="$(sudo docker inspect "$STAGING_CONTAINER_NAME" --format '{{.Config.Image}}' 2>/dev/null)"
  staged_health="$(sudo docker inspect "$STAGING_CONTAINER_NAME" --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' 2>/dev/null)"
  if [ "$staged_image" != "$current_image" ]; then
    write_failure "The approved image is no longer the image running on staging"
    return 1
  fi
  if [ "$staged_health" != "healthy" ] && [ "$staged_health" != "running" ]; then
    write_failure "The approved staging container is not healthy"
    return 1
  fi

  current_message="Promoting the approved image to production"
  write_status "promoting" "$current_message" || return 1
  if ! run_logged env APPROVED=1 "$PROMOTE_SCRIPT" "$current_image"; then
    write_failure "Atomic production promotion failed; review the promotion log"
    return 1
  fi

  prod_image="$(production_image)"
  if [ "$prod_image" != "$current_image" ]; then
    write_failure "Production image does not match the approved image after promotion"
    return 1
  fi
  if ! curl -fsS "$PROD_BASE_URL/health" >/dev/null; then
    write_failure "Production health endpoint failed after promotion"
    return 1
  fi
  if ! curl -fsS "$PROD_BASE_URL/ready" >/dev/null; then
    write_failure "Production readiness endpoint failed after promotion"
    return 1
  fi
  prod_image_digest="$(image_digest "$prod_image")"
  if [ -z "$prod_image_digest" ] || [ "$prod_image_digest" != "$current_image_digest" ]; then
    write_failure "Production image digest does not match the approved staged image"
    return 1
  fi

  completed_at="$(utc_now)"
  if publish_current_github_release "$completed_at"; then
    current_message="Production is running the approved image; GitHub Release is published"
  else
    current_message="Production is healthy; GitHub Release publication needs attention"
  fi
  write_status "completed" "$current_message" "" "$completed_at"
  log "Promoted $current_image to production"
}

archive_processing_request() {
  local suffix="$1"
  local destination="$REQUEST_ARCHIVE_DIR/${current_request_id:-unknown}-${suffix}-$(date '+%Y%m%d%H%M%S').json"
  mv -f -- "$PROCESSING_FILE" "$destination" 2>/dev/null || rm -f -- "$PROCESSING_FILE"
}

process_request() {
  local action=""
  local requested_image=""
  local result=0

  previous_state="$(status_value '.state')"
  previous_image="$(status_value '.image')"
  previous_source_commit="$(status_value '.source_commit')"
  previous_steps="$(jq -c '.steps // []' "$STATUS_FILE" 2>/dev/null || printf '[]')"
  previous_release_status="$(status_value '.release_status')"
  previous_release_tag="$(status_value '.release_tag')"
  previous_release_url="$(status_value '.release_url')"
  previous_release_published_at="$(status_value '.release_published_at')"
  previous_release_error="$(status_value '.release_error')"

  if ! jq -e '
    (.id | type == "string" and test("^[0-9a-f]{32}$")) and
    (.action == "stage" or .action == "promote") and
    ((.image // "") | type == "string") and
    (if .action == "promote" then
       ((.image // "") | length > 0)
     else
       ((.image // "") == "")
     end)
  ' "$PROCESSING_FILE" >/dev/null 2>&1; then
    current_steps="[]"
    active_stage_step=""
    reset_conflict_metadata
    reset_release_metadata
    current_action="invalid"
    current_request_id="invalid"
    current_started_at="$(utc_now)"
    write_failure "Rejected an invalid custom update request"
    archive_processing_request "rejected"
    return
  fi

  current_request_id="$(jq -r '.id' "$PROCESSING_FILE")"
  action="$(jq -r '.action' "$PROCESSING_FILE")"
  requested_image="$(jq -r '.image // ""' "$PROCESSING_FILE")"

  case "$action" in
    stage)
      stage_update
      result=$?
      ;;
    promote)
      case "$requested_image" in
        "$IMAGE_REPO":*) ;;
        *)
          current_action="promote"
          current_started_at="$(utc_now)"
          write_failure "Rejected an unexpected promotion image"
          result=1
          ;;
      esac
      if [ "$result" -eq 0 ]; then
        promote_update "$requested_image"
        result=$?
      fi
      ;;
  esac

  if [ "$result" -eq 0 ]; then
    archive_processing_request "completed"
  else
    archive_processing_request "failed"
  fi
}

main() {
  local recovered_request_id=""
  local next_runtime_sync=0

  need_cmd git
  need_cmd gh
  need_cmd jq
  need_cmd curl
  need_cmd sudo
  need_cmd docker
  need_cmd flock
  need_cmd tee
  need_cmd stat
  need_cmd realpath
  need_cmd awk
  need_cmd grep
  need_cmd sort
  need_cmd sed
  need_cmd head
  need_cmd tr
  need_cmd wc
  need_cmd find

  case "$RUNTIME_SYNC_INTERVAL_SECONDS" in
    ''|*[!0-9]*)
      log "RUNTIME_SYNC_INTERVAL_SECONDS must be a positive integer"
      exit 1
      ;;
  esac
  if [ "$RUNTIME_SYNC_INTERVAL_SECONDS" -lt 1 ]; then
    log "RUNTIME_SYNC_INTERVAL_SECONDS must be a positive integer"
    exit 1
  fi

  [ -d "$SRC_DIR/.git" ] || {
    log "Source repository not found: $SRC_DIR"
    exit 1
  }
  [ -x "$UPDATE_SCRIPT" ] || {
    log "Update script is not executable: $UPDATE_SCRIPT"
    exit 1
  }
  [ -x "$STAGING_SCRIPT" ] || {
    log "Staging script is not executable: $STAGING_SCRIPT"
    exit 1
  }
  [ -x "$PROMOTE_SCRIPT" ] || {
    log "Promotion script is not executable: $PROMOTE_SCRIPT"
    exit 1
  }
  [ -x "$RELEASE_SCRIPT" ] || {
    log "Release publisher is not executable: $RELEASE_SCRIPT"
    exit 1
  }

  mkdir -p \
    "$CONTROL_DIR" \
    "$STATE_DIR" \
    "$LOG_DIR" \
    "$REQUEST_ARCHIVE_DIR" \
    "$MERGE_WORKTREE_ROOT"
  chmod 0750 \
    "$CONTROL_DIR" \
    "$STATE_DIR" \
    "$LOG_DIR" \
    "$REQUEST_ARCHIVE_DIR" \
    "$MERGE_WORKTREE_ROOT"

  exec 9>"$LOCK_FILE"
  flock -n 9 || {
    log "Another custom update controller is already running"
    exit 1
  }

  if [ -f "$PROCESSING_FILE" ]; then
    recovered_request_id="$(jq -r '.id // ""' "$PROCESSING_FILE" 2>/dev/null)"
    if [[ "$recovered_request_id" =~ ^[0-9a-f]{32}$ ]]; then
      cleanup_merge_worktree \
        "$MERGE_WORKTREE_ROOT/$recovered_request_id" \
        "tocreate/official-merge-$recovered_request_id" \
        || log "Could not fully clean the interrupted isolated merge"
    fi
    current_action="recovery"
    current_request_id="${recovered_request_id:-recovered}"
    current_started_at="$(utc_now)"
    current_steps="$(jq -c '.steps // []' "$STATUS_FILE" 2>/dev/null || printf '[]')"
    active_stage_step=""
    reset_conflict_metadata
    reset_release_metadata
    write_failure "The controller restarted during an update; retry after checking staging and production"
    archive_processing_request "interrupted"
  fi
  if [ ! -f "$STATUS_FILE" ]; then
    current_message="Ready for a custom update request"
    write_status "idle" "$current_message"
  fi
  if ! reconcile_completed_status_with_production 1; then
    log "Could not reconcile the completed update status with production"
  fi

  trap shutdown INT TERM
  trap cleanup EXIT
  heartbeat_loop &
  heartbeat_pid=$!

  log "ToCreate custom update controller is ready"
  while true; do
    if [ -f "$REQUEST_FILE" ] && mv -- "$REQUEST_FILE" "$PROCESSING_FILE" 2>/dev/null; then
      process_request
    fi
    if [ "$SECONDS" -ge "$next_runtime_sync" ]; then
      if ! reconcile_completed_status_with_production; then
        log "Could not reconcile the completed update status with production"
      fi
      next_runtime_sync=$((SECONDS + RUNTIME_SYNC_INTERVAL_SECONDS))
    fi
    sleep "$POLL_INTERVAL_SECONDS"
  done
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
