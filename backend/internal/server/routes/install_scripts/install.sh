#!/usr/bin/env bash

set -u

BASE_URL="${TOCREATE_BASE_URL:-__TOCREATE_BASE_URL__}"
TOKEN=""
TMP_DIR=""
OS=""
ARCH=""
CLIENT=""
CLIENT_LABEL=""
CLI_COMMAND=""
CLI_PACKAGE=""
MIN_NODE_VERSION=22
CC_SWITCH_READY=0
CC_SWITCH_BIN=""

if [ -t 1 ]; then
  COLOR_GREEN='\033[0;32m'
  COLOR_BLUE='\033[0;36m'
  COLOR_YELLOW='\033[0;33m'
  COLOR_RED='\033[0;31m'
  COLOR_BOLD='\033[1m'
  COLOR_RESET='\033[0m'
else
  COLOR_GREEN=''
  COLOR_BLUE=''
  COLOR_YELLOW=''
  COLOR_RED=''
  COLOR_BOLD=''
  COLOR_RESET=''
fi

cleanup() {
  if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
  fi
}

trap cleanup EXIT INT TERM

print_banner() {
  printf '%b' "${COLOR_BOLD}${COLOR_BLUE}"
  cat <<'BANNER'
 ______   ______   ______     ______     ______     ______   ______   ______
/\__  _\ /\  __ \ /\  ___\   /\  == \   /\  ___\   /\  __ \ /\__  _\ /\  ___\
\/_/\ \/ \ \ \/\ \\ \ \____  \ \  __<   \ \  __\   \ \  __ \\/_/\ \/ \ \  __\
   \ \_\  \ \_____\\ \_____\  \ \_\ \_\  \ \_____\  \ \_\ \_\  \ \_\  \ \_____\
    \/_/   \/_____/ \/_____/   \/_/ /_/   \/_____/   \/_/\/_/   \/_/   \/_____/
BANNER
  printf '%b\n' "${COLOR_RESET}"
  printf '%b%s%b\n\n' "${COLOR_BOLD}${COLOR_GREEN}" '                               ToCreate Quick Start' "${COLOR_RESET}"
  printf '%s\n\n' 'Install one CLI, CC Switch, and your ToCreate provider.'
}

section() {
  printf '\n%b%s%b\n' "${COLOR_BOLD}${COLOR_BLUE}" "$1" "$COLOR_RESET"
}

progress() {
  printf '%b›%b %s\n' "$COLOR_BLUE" "$COLOR_RESET" "$1"
}

success() {
  printf '%b✓%b %s\n' "$COLOR_GREEN" "$COLOR_RESET" "$1"
}

warning() {
  printf '%b!%b %s\n' "$COLOR_YELLOW" "$COLOR_RESET" "$1"
}

fail() {
  printf '%b✗%b %s\n' "$COLOR_RED" "$COLOR_RESET" "$1" >&2
}

die() {
  fail "$1"
  exit 1
}

as_root() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
  elif command -v sudo >/dev/null 2>&1; then
    sudo "$@"
  else
    return 1
  fi
}

parse_args() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --token)
        [ "$#" -ge 2 ] || die 'Missing value for --token.'
        TOKEN="$2"
        shift 2
        ;;
      --base-url)
        [ "$#" -ge 2 ] || die 'Missing value for --base-url.'
        BASE_URL="${2%/}"
        shift 2
        ;;
      -h|--help)
        printf '%s\n' 'Usage: install.sh --token <INSTALL_TOKEN> [--base-url <URL>]'
        exit 0
        ;;
      *)
        die "Unknown argument: $1"
        ;;
    esac
  done
}

validate_token_shape() {
  case "$TOKEN" in
    tcinst_*) ;;
    *) die 'The install token is invalid. Return to Quick Start and refresh the command.' ;;
  esac

  token_body="${TOKEN#tcinst_}"
  [ "${#token_body}" -ge 32 ] && [ "${#TOKEN}" -le 160 ] ||
    die 'The install token is invalid. Return to Quick Start and refresh the command.'
  case "$token_body" in
    *[!A-Za-z0-9_-]*)
      die 'The install token is invalid. Return to Quick Start and refresh the command.'
      ;;
  esac
}

detect_platform() {
  case "$(uname -s)" in
    Darwin) OS='darwin' ;;
    Linux) OS='linux' ;;
    *) die 'This installer currently supports macOS and Linux.' ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64) ARCH='amd64' ;;
    arm64|aarch64) ARCH='arm64' ;;
    *) die "Unsupported CPU architecture: $(uname -m)" ;;
  esac
}

node_major_version() {
  node -p 'Number(process.versions.node.split(".")[0])' 2>/dev/null || printf '0'
}

install_node_linux() {
  if command -v apt-get >/dev/null 2>&1; then
    as_root apt-get update >/dev/null 2>&1 &&
      as_root apt-get install -y nodejs npm >/dev/null 2>&1
  elif command -v dnf >/dev/null 2>&1; then
    as_root dnf install -y nodejs npm >/dev/null 2>&1
  elif command -v yum >/dev/null 2>&1; then
    as_root yum install -y nodejs npm >/dev/null 2>&1
  elif command -v pacman >/dev/null 2>&1; then
    as_root pacman -Sy --noconfirm nodejs npm >/dev/null 2>&1
  else
    return 1
  fi
}

ensure_node() {
  if command -v node >/dev/null 2>&1 && command -v npm >/dev/null 2>&1; then
    if [ "$(node_major_version)" -ge "$MIN_NODE_VERSION" ]; then
      success "Node.js $(node --version) and npm are available"
      return
    fi
    warning "Node.js $(node --version) is too old; version ${MIN_NODE_VERSION} or newer is required"
  else
    progress 'Node.js or npm was not found'
  fi

  if [ "$OS" = 'darwin' ] && command -v brew >/dev/null 2>&1; then
    progress 'Installing Node.js with Homebrew'
    brew install node >/dev/null 2>&1 || die "Homebrew could not install Node.js. Install Node.js ${MIN_NODE_VERSION}+ from https://nodejs.org/ and run this command again."
  elif [ "$OS" = 'linux' ]; then
    progress 'Installing Node.js with the system package manager'
    install_node_linux || die "Node.js could not be installed automatically. Install Node.js ${MIN_NODE_VERSION}+ from https://nodejs.org/ and run this command again."
  else
    die "Node.js could not be installed automatically. Install Node.js ${MIN_NODE_VERSION}+ from https://nodejs.org/ and run this command again."
  fi

  command -v node >/dev/null 2>&1 || die 'Node.js installation finished but node is not on PATH.'
  command -v npm >/dev/null 2>&1 || die 'Node.js installation finished but npm is not on PATH.'
  [ "$(node_major_version)" -ge "$MIN_NODE_VERSION" ] ||
    die "The installed Node.js version is older than ${MIN_NODE_VERSION}. Update it from https://nodejs.org/."
  success "Node.js $(node --version) and npm are ready"
}

json_get() {
  node - "$1" "$2" <<'NODE'
const fs = require('fs')
const [file, path] = process.argv.slice(2)
const value = path.split('.').reduce((current, key) => current == null ? undefined : current[key], JSON.parse(fs.readFileSync(file, 'utf8')))
if (value !== undefined && value !== null) process.stdout.write(String(value))
NODE
}

request_json() {
  node -e 'process.stdout.write(JSON.stringify({token: process.argv[1], os: process.argv[2], arch: process.argv[3]}))' "$TOKEN" "$OS" "$ARCH"
}

preflight_request_json() {
  printf '{"token":"%s","os":"%s","arch":"%s"}' "$TOKEN" "$OS" "$ARCH"
}

post_api() {
  endpoint="$1"
  payload="$2"
  output="$3"
  curl -sS \
    -o "$output" \
    -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    --data "$payload" \
    "${BASE_URL%/}${endpoint}"
}

explain_api_error() {
  file="$1"
  status="$2"
  reason="$(json_get "$file" reason 2>/dev/null || true)"
  message="$(json_get "$file" message 2>/dev/null || true)"
  case "$reason" in
    token_expired) fail 'The install token expired. Return to Quick Start and choose "Refresh command".' ;;
    token_used) fail 'This install token was already used. Return to Quick Start and refresh the command.' ;;
    token_revoked) fail 'This install token was revoked. Return to Quick Start and refresh the command.' ;;
    key_disabled) fail 'The selected API key was deleted, disabled, expired, or exhausted.' ;;
    no_credit) fail 'This account needs balance or an active subscription before installation.' ;;
    client_mismatch) fail 'The selected API key group no longer matches this CLI.' ;;
    install_token_rate_limited) fail 'Too many token requests were made. Wait a minute and try again.' ;;
    *) fail "ToCreate returned HTTP ${status}: ${message:-unknown error}" ;;
  esac
}

explain_preflight_api_error() {
  file="$1"
  status="$2"
  body="$(tr -d '\r\n' <"$file" 2>/dev/null || true)"
  case "$body" in
    *'"reason":"token_expired"'*) fail 'The install token expired. Return to Quick Start and choose "Refresh command".' ;;
    *'"reason":"token_used"'*) fail 'This install token was already used. Return to Quick Start and refresh the command.' ;;
    *'"reason":"token_revoked"'*) fail 'This install token was revoked. Return to Quick Start and refresh the command.' ;;
    *'"reason":"key_disabled"'*) fail 'The selected API key was deleted, disabled, expired, or exhausted.' ;;
    *'"reason":"no_credit"'*) fail 'This account needs balance or an active subscription before installation.' ;;
    *'"reason":"client_mismatch"'*) fail 'The selected API key group no longer matches this CLI.' ;;
    *'"reason":"install_token_rate_limited"'*) fail 'Too many token requests were made. Wait a minute and try again.' ;;
    *'"reason":"token_invalid"'*|*'"reason":"token_not_found"'*)
      fail 'The install token is invalid. Return to Quick Start and refresh the command.'
      ;;
    *) fail "ToCreate returned HTTP ${status} while validating the install token." ;;
  esac
}

fetch_install_metadata() {
  metadata_file="$TMP_DIR/peek.json"
  status="$(post_api '/api/v1/install-token/peek' "$(preflight_request_json)" "$metadata_file")" ||
    die 'Could not reach ToCreate. Check the network connection and run the command again.'
  case "$status" in
    2??) ;;
    *)
      explain_preflight_api_error "$metadata_file" "$status"
      exit 1
      ;;
  esac
  success 'Install token is valid'
}

load_install_metadata() {
  metadata_file="$TMP_DIR/peek.json"
  CLIENT="$(json_get "$metadata_file" data.client)"
  case "$CLIENT" in
    claude-code)
      CLIENT_LABEL='Claude Code'
      CLI_COMMAND='claude'
      CLI_PACKAGE='@anthropic-ai/claude-code@latest'
      ;;
    codex)
      CLIENT_LABEL='Codex'
      CLI_COMMAND='codex'
      CLI_PACKAGE='@openai/codex@latest'
      ;;
    gemini-cli)
      CLIENT_LABEL='Gemini CLI'
      CLI_COMMAND='gemini'
      CLI_PACKAGE='@google/gemini-cli@latest'
      ;;
    *)
      die 'The install token returned an unsupported client.'
      ;;
  esac
  success "Selected client: ${CLIENT_LABEL}"
}

install_cli() {
  if command -v "$CLI_COMMAND" >/dev/null 2>&1; then
    success "${CLIENT_LABEL} is already installed"
    return
  fi

  progress "Installing ${CLIENT_LABEL}"
  log_file="$TMP_DIR/npm-install.log"
  if ! npm install -g "$CLI_PACKAGE" >"$log_file" 2>&1; then
    if command -v sudo >/dev/null 2>&1 && [ "$(id -u)" -ne 0 ]; then
      progress 'Retrying the global npm install with administrator privileges'
      sudo npm install -g "$CLI_PACKAGE" >"$log_file" 2>&1 || {
        fail "${CLIENT_LABEL} installation failed."
        tail -n 12 "$log_file" >&2
        printf '%s\n' 'Check npm network access and global install permissions, then run the command again.' >&2
        exit 1
      }
    else
      fail "${CLIENT_LABEL} installation failed."
      tail -n 12 "$log_file" >&2
      printf '%s\n' 'Check npm network access and global install permissions, then run the command again.' >&2
      exit 1
    fi
  fi

  npm_prefix="$(npm prefix -g 2>/dev/null || true)"
  if [ -n "$npm_prefix" ]; then
    PATH="$npm_prefix/bin:$PATH"
    export PATH
  fi
  hash -r 2>/dev/null || true
  command -v "$CLI_COMMAND" >/dev/null 2>&1 ||
    die "${CLIENT_LABEL} was installed, but ${CLI_COMMAND} is not on PATH. Open a new terminal and run the command again."
  success "${CLIENT_LABEL} installed"
}

detect_cc_switch_macos() {
  if open -Ra 'CC Switch' >/dev/null 2>&1; then
    CC_SWITCH_READY=1
    return
  fi
  for app in '/Applications/CC Switch.app' "$HOME/Applications/CC Switch.app"; do
    if [ -d "$app" ]; then
      CC_SWITCH_READY=1
      return
    fi
  done
}

install_cc_switch_macos() {
  progress 'Downloading CC Switch for macOS'
  dmg="$TMP_DIR/cc-switch.dmg"
  mount_dir="$TMP_DIR/cc-switch-mount"
  mkdir -p "$mount_dir"
  curl -fL "${BASE_URL%/}/download/cc-switch/macos" -o "$dmg" >/dev/null 2>&1 || return 1
  hdiutil attach "$dmg" -nobrowse -readonly -mountpoint "$mount_dir" >/dev/null 2>&1 || return 1
  app_path="$(find "$mount_dir" -type d -name '*.app' -prune -print | head -n 1)"
  if [ -z "$app_path" ]; then
    hdiutil detach "$mount_dir" >/dev/null 2>&1 || true
    return 1
  fi
  mkdir -p "$HOME/Applications"
  rm -rf "$HOME/Applications/CC Switch.app"
  cp -R "$app_path" "$HOME/Applications/CC Switch.app" >/dev/null 2>&1
  copy_status=$?
  hdiutil detach "$mount_dir" >/dev/null 2>&1 || true
  [ "$copy_status" -eq 0 ] || return 1
  CC_SWITCH_READY=1
  return 0
}

detect_cc_switch_linux() {
  for candidate in \
    "$(command -v cc-switch 2>/dev/null || true)" \
    "$HOME/.local/bin/cc-switch.AppImage" \
    "$HOME/Applications/CC-Switch.AppImage"; do
    if [ -n "$candidate" ] && [ -x "$candidate" ]; then
      CC_SWITCH_BIN="$candidate"
      CC_SWITCH_READY=1
      return
    fi
  done
}

install_cc_switch_linux() {
  case "$ARCH" in
    amd64) platform='linux-x86_64' ;;
    arm64) platform='linux-arm64' ;;
    *) return 1 ;;
  esac
  progress 'Downloading CC Switch AppImage'
  mkdir -p "$HOME/.local/bin"
  target="$HOME/.local/bin/cc-switch.AppImage"
  curl -fL "${BASE_URL%/}/download/cc-switch/${platform}" -o "$target" >/dev/null 2>&1 || return 1
  chmod +x "$target" || return 1
  CC_SWITCH_BIN="$target"
  CC_SWITCH_READY=1
}

ensure_cc_switch() {
  if [ "$OS" = 'darwin' ]; then
    detect_cc_switch_macos
    if [ "$CC_SWITCH_READY" -eq 0 ]; then
      install_cc_switch_macos || true
    fi
  else
    detect_cc_switch_linux
    if [ "$CC_SWITCH_READY" -eq 0 ]; then
      install_cc_switch_linux || true
    fi
  fi

  if [ "$CC_SWITCH_READY" -eq 1 ]; then
    success 'CC Switch is available'
  else
    warning 'CC Switch could not be installed automatically; browser confirmation will be used'
  fi
}

open_deeplink() {
  deeplink="$1"
  if [ "$OS" = 'darwin' ]; then
    open "$deeplink" >/dev/null 2>&1
    return $?
  fi
  if command -v xdg-open >/dev/null 2>&1 && xdg-open "$deeplink" >/dev/null 2>&1; then
    return 0
  fi
  if [ -n "$CC_SWITCH_BIN" ] && [ -x "$CC_SWITCH_BIN" ]; then
    "$CC_SWITCH_BIN" "$deeplink" >/dev/null 2>&1 &
    app_pid=$!
    sleep 2
    if kill -0 "$app_pid" >/dev/null 2>&1; then
      return 0
    fi
  fi
  return 1
}

open_url() {
  target="$1"
  if [ "$OS" = 'darwin' ]; then
    open "$target" >/dev/null 2>&1
  elif command -v xdg-open >/dev/null 2>&1; then
    xdg-open "$target" >/dev/null 2>&1
  else
    return 1
  fi
}

redeem_and_import() {
  response_file="$TMP_DIR/redeem.json"
  status="$(post_api '/api/v1/install-token/redeem' "$(request_json)" "$response_file")" ||
    die 'Could not redeem the install token. Check the network connection and run the command again.'
  case "$status" in
    2??) ;;
    *)
      explain_api_error "$response_file" "$status"
      exit 1
      ;;
  esac

  deeplink="$(json_get "$response_file" data.deeplink)"
  confirm_url="$(json_get "$response_file" data.confirm_url)"
  [ -n "$deeplink" ] || die 'ToCreate returned an incomplete CC Switch import payload.'
  [ -n "$confirm_url" ] || die 'ToCreate returned an incomplete fallback URL.'

  if [ "$CC_SWITCH_READY" -eq 1 ] && open_deeplink "$deeplink"; then
    success 'CC Switch import opened'
    printf '\n%b✨ Installation complete%b\n' "${COLOR_BOLD}${COLOR_GREEN}" "$COLOR_RESET"
    print_next_steps
    return
  fi

  warning 'Automatic CC Switch import needs browser confirmation'
  if open_url "$confirm_url"; then
    success 'Opened the one-click import confirmation page'
  else
    printf '%s\n' 'Open this URL in a browser to finish the import:'
    printf '%s\n' "$confirm_url"
  fi
  printf '\n%bInstaller finished - confirmation required%b\n' "${COLOR_BOLD}${COLOR_YELLOW}" "$COLOR_RESET"
  print_next_steps
}

print_next_steps() {
  printf '\n%bNext steps%b\n' "$COLOR_BOLD" "$COLOR_RESET"
  printf '%s\n' "1. Finish and enable the ${CLIENT_LABEL} provider in CC Switch."
  printf '%s\n' '2. Close any existing CLI session and start a new one:'
  printf '   %b%s%b\n' "$COLOR_BLUE" "$CLI_COMMAND" "$COLOR_RESET"
  printf '%s\n' '3. Send a short test request, for example: "Reply with ToCreate connected."'
  printf '%s\n' "Docs: ${BASE_URL%/}/custom/codex-claude-import"
}

main() {
  parse_args "$@"
  [ -n "$TOKEN" ] || die 'Missing --token. Copy a fresh command from ToCreate Quick Start.'
  BASE_URL="${BASE_URL%/}"
  print_banner

  section '1. Preflight'
  detect_platform
  success "Detected ${OS}/${ARCH}"
  success 'Install token is present'

  validate_token_shape
  TMP_DIR="$(mktemp -d)" || die 'Could not create a temporary installer directory.'
  progress 'Validating install token'
  fetch_install_metadata
  ensure_node
  load_install_metadata

  section '2. Install tools'
  install_cli
  ensure_cc_switch

  section '3. Import config'
  progress 'Redeeming the one-time install token'
  redeem_and_import
}

main "$@"
