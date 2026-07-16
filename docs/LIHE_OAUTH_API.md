# Lihe Chat OAuth API Contract

This document freezes the API-side contract used by the confidential
`lihe.chat` LibreChat integration. The API platform is the OAuth authorization
server and resource server. LibreChat is the only supported OAuth client.

## Fixed server configuration

The following values are configured on the API server and are never accepted
from an import result:

| Setting | Production value |
| --- | --- |
| `client_id` | `lihe-chat` |
| redirect URI | `https://lihe.chat/api/integrations/lihe/callback` |
| connect URL | `https://lihe.chat/connect/lihe` |
| scopes | `models:read chat:write` |
| PKCE method | `S256` |

The OAuth feature is disabled unless all server-side settings are valid.

## Authorization request

The browser opens:

```text
GET /oauth/authorize?response_type=code
  &client_id=lihe-chat
  &redirect_uri=https%3A%2F%2Flihe.chat%2Fapi%2Fintegrations%2Flihe%2Fcallback
  &scope=models%3Aread%20chat%3Awrite
  &state=...
  &code_challenge=...
  &code_challenge_method=S256
```

`/oauth/authorize` is a protected SPA route. If the API user is not logged in,
the existing login flow preserves the complete URL and resumes it after login.
The page calls the authenticated endpoint below with the same query string:

```text
GET /api/v1/oauth/lihe/authorize
Authorization: Bearer <API-site session JWT>
```

On success it returns the only URL the browser may navigate to:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "redirect_to": "https://lihe.chat/api/integrations/lihe/callback?code=...&state=...",
    "expires_in": 60
  }
}
```

The authorization code contains 32 random bytes, is stored only as a SHA-256
hash, expires after 60 seconds, and can be consumed once. `state` is never
stored and is returned unchanged to the fixed callback URI.

## Token exchange

```http
POST /oauth/token
Authorization: Basic base64(lihe-chat:<client-secret>)
Content-Type: application/x-www-form-urlencoded
Cache-Control: no-store

grant_type=authorization_code&code=...&redirect_uri=...&code_verifier=...
```

Success:

```json
{
  "access_token": "lihe_...",
  "token_type": "Bearer",
  "scope": "models:read chat:write",
  "providers": ["openai", "anthropic"],
  "created_at": "2026-07-16T12:00:00Z"
}
```

There is deliberately no `expires_in`: the token remains valid until revoked.
The plaintext access token is returned once and only its SHA-256 hash is
stored. Code consumption, token creation, and all provider bindings are one
database transaction.

Errors use the OAuth shape and never include a code, verifier, secret, or token:

```json
{
  "error": "invalid_grant",
  "error_description": "authorization code is invalid or expired"
}
```

## Resource requests

LibreChat sends the long-lived token only in an authorization header and
selects one of the `providers` returned by the token exchange:

```http
Authorization: Bearer lihe_...
X-Lihe-Provider: openai
```

For each provider, LibreChat verifies and loads models with `GET /v1/models`.
The provider header is required because one Lihe token can map to several
isolated provider groups.

The token is accepted only on these resource routes:

| Scope | Method and path |
| --- | --- |
| `models:read` | `GET /v1/models` |
| `chat:write` | `POST /v1/chat/completions` |
| `chat:write` | `POST /v1/messages` |
| `chat:write` | `POST /v1/messages/count_tokens` |
| `chat:write` | `POST /v1/responses` and subpaths |
| `chat:write` | `GET /v1/responses` (WebSocket) |
| `chat:write` | `POST /v1/alpha/search` |

All account, balance, payment, withdrawal, key-management, administration,
usage, image, video, and native Gemini routes are denied to Lihe tokens.
Revocation and user status are checked from PostgreSQL before every request;
streaming is unaffected after authorization succeeds.

## Revocation

LibreChat revokes its current token with confidential client authentication:

```http
POST /oauth/revoke
Authorization: Basic base64(lihe-chat:<client-secret>)
Content-Type: application/x-www-form-urlencoded

token=lihe_...&token_type_hint=access_token
```

The endpoint is idempotent and returns HTTP 200 even when the token is already
invalid. The API user can list and revoke their own connections through:

```text
GET    /api/v1/oauth/lihe/tokens
DELETE /api/v1/oauth/lihe/tokens/{id}
```

Administrators can inspect one user's active connections and revoke any
connection through audited admin routes:

```text
GET    /api/v1/admin/oauth/lihe/tokens?user_id={user_id}
DELETE /api/v1/admin/oauth/lihe/tokens/{id}
```

Token list entries expose only ID, fixed name, scopes, providers, creation
time, and last-use time. No credential or credential prefix is returned.

## Security invariants

- Callback URI and client ID must match the configured values exactly.
- PKCE is mandatory and only `S256` is accepted.
- Client secrets are compared in constant time.
- OAuth responses use `Cache-Control: no-store` and `Pragma: no-cache`.
- Long-lived tokens are never placed in URLs, browser storage, redirects, or
  application logs.
- Internal provider keys cannot be authenticated directly and are hidden from
  ordinary API-key list, detail, search, update, delete, and count operations.
- A token can resolve only to API keys owned by the same user recorded on the
  token, preventing cross-user binding.
