#!/usr/bin/env bash
set -uo pipefail

# Host-side controller for the ToCreate custom update UI. The application can
# only enqueue the fixed "stage" and "promote" actions through a shared
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
LOCK_FILE="${LOCK_FILE:-/tmp/sub2api-custom-update-controller.lock}"
UPDATE_SCRIPT="${UPDATE_SCRIPT:-$DEPLOY_DIR/update-custom-sub2api.sh}"
STAGING_SCRIPT="${STAGING_SCRIPT:-$DEPLOY_DIR/deploy-custom-sub2api-staging.sh}"
PROMOTE_SCRIPT="${PROMOTE_SCRIPT:-$DEPLOY_DIR/promote-custom-sub2api.sh}"
STAGING_CONTAINER_NAME="${STAGING_CONTAINER_NAME:-sub2api-test}"
PROD_CONTAINER_NAME="${PROD_CONTAINER_NAME:-sub2api}"
STAGING_BASE_URL="${STAGING_BASE_URL:-http://127.0.0.1:18080}"
PROD_BASE_URL="${PROD_BASE_URL:-http://127.0.0.1:8080}"

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
heartbeat_pid=""

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
      production_url: $production_url
    } | with_entries(select(.value != ""))' > "$temporary"; then
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

write_failure() {
  local message="$1"
  current_message="Custom update failed"
  write_status "failed" "$current_message" "$message" "$(utc_now)"
  log "$message"
  return 1
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

stage_update() {
  local prod_image=""
  local suffix=""
  local staged_image=""
  local staged_health=""

  current_action="stage"
  current_started_at="$(utc_now)"
  current_log_file="stage-${current_request_id}-$(date '+%Y%m%d%H%M%S').log"
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
  if ! run_logged git -C "$SRC_DIR" fetch "$UPSTREAM_REMOTE" "$UPSTREAM_REF" --tags; then
    write_failure "Could not fetch $UPSTREAM_REMOTE/$UPSTREAM_REF"
    return 1
  fi

  current_upstream_commit="$(git -C "$SRC_DIR" rev-parse "$UPSTREAM_REMOTE/$UPSTREAM_REF" 2>/dev/null)"
  if [ -z "$current_upstream_commit" ]; then
    write_failure "Could not resolve the official upstream commit"
    return 1
  fi

  current_message="Merging official upstream into the ToCreate branch"
  write_status "merging" "$current_message" || return 1
  if ! run_logged git -C "$SRC_DIR" merge --no-edit "$UPSTREAM_REMOTE/$UPSTREAM_REF"; then
    write_failure "Official update has merge conflicts; production was not changed"
    return 1
  fi

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
    current_image="$prod_image"
    current_image_digest="$(image_digest "$prod_image")"
    current_message="Production already uses the latest merged ToCreate source"
    write_status "completed" "$current_message" "" "$(utc_now)"
    return 0
  fi

  if ! choose_image "$prod_image"; then
    write_failure "Could not allocate the next ToCreate image version"
    return 1
  fi
  suffix="$chosen_suffix"

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

  current_message="Deploying the exact image to port 18080"
  write_status "staging" "$current_message" || return 1
  if ! run_logged "$STAGING_SCRIPT" "$current_image"; then
    write_failure "Staging deployment or validation failed; production was not changed"
    return 1
  fi

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

  current_message="Staging is healthy; explicit production approval is required"
  write_status "awaiting_approval" "$current_message" || return 1
  log "Staged $current_image ($current_image_digest) on port 18080"
}

promote_update() {
  local requested_image="$1"
  local staged_image=""
  local staged_health=""
  local prod_image=""

  current_action="promote"
  current_started_at="$(utc_now)"
  current_log_file="promote-${current_request_id}-$(date '+%Y%m%d%H%M%S').log"
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

  current_message="Production is running the approved ToCreate image"
  write_status "completed" "$current_message" "" "$(utc_now)"
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

  if ! jq -e '
    (.id | type == "string" and test("^[0-9a-f]{32}$")) and
    (.action == "stage" or .action == "promote") and
    ((.image // "") | type == "string")
  ' "$PROCESSING_FILE" >/dev/null 2>&1; then
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

need_cmd git
need_cmd gh
need_cmd jq
need_cmd curl
need_cmd sudo
need_cmd docker
need_cmd flock
need_cmd tee

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

mkdir -p "$CONTROL_DIR" "$LOG_DIR" "$REQUEST_ARCHIVE_DIR"
chmod 0750 "$CONTROL_DIR" "$STATE_DIR" "$LOG_DIR" "$REQUEST_ARCHIVE_DIR"

exec 9>"$LOCK_FILE"
flock -n 9 || {
  log "Another custom update controller is already running"
  exit 1
}

if [ -f "$PROCESSING_FILE" ]; then
  current_action="recovery"
  current_request_id="recovered"
  current_started_at="$(utc_now)"
  write_failure "The controller restarted during an update; retry after checking staging and production"
  archive_processing_request "interrupted"
fi
if [ ! -f "$STATUS_FILE" ]; then
  current_message="Ready for a custom update request"
  write_status "idle" "$current_message"
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
  sleep "$POLL_INTERVAL_SECONDS"
done
