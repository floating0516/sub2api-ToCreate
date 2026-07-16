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

When enabled, every row on the user-facing `API Keys` page shows an `Import to
chat site` action immediately after `Import to CCS`. Its URL is the configured
connect URL with only the non-secret numeric key ID appended:

```text
https://lihe.chat/connect/lihe?api_key_id=123
```

The API key plaintext is never placed in the URL. LibreChat must preserve this
ID through login and include it in the authorization request. The separate
Lihe Chat integration page remains the place to inspect and revoke existing
connections.

## Authorization request

The browser opens:

```text
GET /oauth/authorize?response_type=code
  &api_key_id=123
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
stored and is returned unchanged to the fixed callback URI. `api_key_id` is
mandatory. The server verifies that the selected key belongs to the logged-in
API user, is active, has a supported active provider group, and currently has
an available account. The selected ID is stored with the authorization code
and checked again during the exchange transaction.

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
  "providers": ["openai"],
  "api_key_id": 123,
  "api_key_name": "My OpenAI key",
  "created_at": "2026-07-16T12:00:00Z"
}
```

There is deliberately no `expires_in`: the token remains valid until revoked.
The plaintext access token is returned once and only its SHA-256 hash is
stored. Code consumption, token creation, and the selected API-key binding are
one database transaction. The exchange never creates or copies an API key.

Errors use the OAuth shape and never include a code, verifier, secret, or token:

```json
{
  "error": "invalid_grant",
  "error_description": "authorization code is invalid or expired"
}
```

## Resource requests

LibreChat sends the long-lived token only in an authorization header and uses
the single provider returned by the token exchange:

```http
Authorization: Bearer lihe_...
X-Lihe-Provider: openai
```

LibreChat verifies and loads models with `GET /v1/models`. The provider header
is required so the gateway can verify that it still matches the selected
key's bound group.

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

Token list entries expose only ID, fixed name, scopes, the source API key ID
and display name, provider, creation time, and last-use time. No credential or
credential prefix is returned. Revoking a Lihe token never disables or deletes
the user's source API key. It only disables legacy hidden `lihe-internal-*`
keys created by earlier integration versions.

## Security invariants

- Callback URI and client ID must match the configured values exactly.
- PKCE is mandatory and only `S256` is accepted.
- Client secrets are compared in constant time.
- OAuth responses use `Cache-Control: no-store` and `Pragma: no-cache`.
- Long-lived tokens are never placed in URLs, browser storage, redirects, or
  application logs.
- A token can resolve only to the selected API key owned by the same user
  recorded on both the authorization code and token, preventing cross-user
  binding.
- The selected key must not be deleted, and its current group ID and provider
  must still match the binding captured at issuance. Disabling the key is
  rejected by the normal API-key authorization checks.
- Legacy internal provider keys cannot be authenticated directly and remain
  hidden from ordinary API-key list, detail, search, update, delete, and count
  operations.
