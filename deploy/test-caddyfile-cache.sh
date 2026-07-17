#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
caddyfile="$repo_root/deploy/Caddyfile"
active_config=$(sed 's/[[:space:]]*#.*$//' "$caddyfile")

if printf '%s\n' "$active_config" | grep -Eiq 'Cache-Control.*immutable'; then
	echo "Caddyfile must not force immutable caching; the backend owns asset cache policy" >&2
	exit 1
fi

if ! printf '%s\n' "$active_config" | grep -Eq '^[[:space:]]*reverse_proxy[[:space:]]+localhost:8080'; then
	echo "Caddyfile must continue proxying all application routes to localhost:8080" >&2
	exit 1
fi

for sensitive_query_key in code state nonce code_verifier code_challenge id_token access_token client_secret token; do
	if ! printf '%s\n' "$active_config" | grep -Eq "replace[[:space:]]+$sensitive_query_key[[:space:]]+REDACTED"; then
		echo "Caddyfile must redact query key: $sensitive_query_key" >&2
		exit 1
	fi
done

echo "Caddyfile preserves backend cache policy, routing, and OIDC query redaction"
