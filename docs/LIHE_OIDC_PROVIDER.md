# Lihe Unified Account OIDC Provider

This document freezes the API-side login protocol. It is separate from the
long-lived Lihe API Key OAuth contract in `LIHE_OAUTH_API.md`; the two flows do
not share clients, secrets, database tables, authorization codes, or tokens.

## Frozen production contract

| Setting | Value |
| --- | --- |
| Issuer | `https://api.lihe.chat` |
| Discovery | `https://api.lihe.chat/.well-known/openid-configuration` |
| Client ID | `lihe-chat-login` |
| Redirect URI | `https://lihe.chat/oauth/openid/callback` |
| Scope | `openid profile email` |
| Flow | Authorization Code with PKCE S256 |
| Client authentication | `client_secret_basic` |
| ID Token | RS256 with `kid` |
| Access Token lifetime | 300 seconds |
| Authorization code lifetime | 60 seconds |
| Refresh Token | Not issued |

The OIDC Access Token is independent from the long-lived `lihe_` token. It can
only be used as a Bearer token on `GET /oidc/userinfo`.

## Endpoints

| Method | Endpoint | Purpose |
| --- | --- | --- |
| `GET` | `/.well-known/openid-configuration` | Discovery |
| `GET` | `/oidc/authorize` | API site's embedded authorization SPA |
| `POST` | `/api/v1/oidc/prepare` | Validate and replace the full request with an opaque handle |
| `POST` | `/api/v1/oidc/authorize` | Issue a code using the current API login session |
| `POST` | `/oidc/token` | Confidential code exchange |
| `GET` | `/oidc/userinfo` | Claims for the short-lived Access Token |
| `GET` | `/oidc/jwks` | Current and previous public signing keys |

The authorization SPA removes the original query from browser history before
its first asynchronous request. The backend validates the fixed client,
redirect URI, exact scopes, nonce, state, and PKCE challenge, encrypts the
short-lived request, and returns a random opaque handle. If the API user is not
logged in, only that handle is preserved through the normal API login flow.

The authenticated authorization endpoint returns a fixed-host callback URL.
It never accepts a callback supplied by the SPA after preparation.
`prompt=none` returns `login_required` to that callback when no API session is
available, and `prompt=login` forces API reauthentication. `max_age` is rejected
in this phase because API refresh sessions do not yet retain a separate,
original authentication timestamp.

## Token response

```json
{
  "access_token": "<independent short-lived OIDC token>",
  "token_type": "Bearer",
  "expires_in": 300,
  "id_token": "<RS256 JWT with kid>",
  "scope": "openid profile email"
}
```

No Refresh Token is issued. The ID Token and UserInfo response contain:

```json
{
  "sub": "11111111-1111-4111-8111-111111111111",
  "email": "user@example.com",
  "email_verified": false,
  "preferred_username": "user",
  "name": "User"
}
```

The ID Token additionally contains `iss`, `aud`, `exp`, `iat`, and the exact
authorization-request `nonce`. `sub` is a generated UUID that is immutable and
unique across active and soft-deleted API users. An email is verified only
when `email_verified_at` and its reliable source were recorded by the API
site. Historical accounts without reliable evidence return `false`.

## Persistence and key rotation

- Pending handles, browser bindings, authorization codes, code signatures,
  Access Tokens, nonce bindings, and PKCE challenge bindings are stored only as
  SHA-256 hashes.
- Fosite request/session payloads needed for code exchange are encrypted with
  AES-256-GCM using a domain-separated key derived from
  `LIHE_OIDC_HMAC_SECRET`. This includes transient state, nonce, code, and PKCE
  verifier values.
- Authorization code consumption and Access Token creation use PostgreSQL
  transactions. A code is single-use; replay also revokes Access Tokens tied
  to the same Fosite request ID.
- The active RSA private key and immediately previous private key are stored
  under `LIHE_OIDC_KEY_DIRECTORY` with mode `0600`. Only public keys are
  returned by JWKS. The persistent `/app/data` volume must be mounted.
- API user deletion is soft deletion. The ordinary unique index on
  `users.oidc_subject` covers deleted rows, so a subject cannot be reassigned.

## Configuration

Generate three unrelated secrets for API JWT sessions, the login client, and
OIDC HMAC/encrypted storage:

```text
JWT_SECRET=<random value 1>
LIHE_OAUTH_CLIENT_SECRET=<random value 2 for API Key OAuth>
LIHE_OIDC_CLIENT_SECRET=<random value 3 for login OIDC>
LIHE_OIDC_HMAC_SECRET=<random value 4>
```

The server rejects OIDC startup configuration when these values are reused.
Keep `LIHE_OIDC_ENABLED=false` until HTTPS staging passes the complete code,
PKCE, ID Token, UserInfo, and API Key import flow. The browser-binding cookie
is always `Secure`, so a plain HTTP port is not a valid end-to-end OIDC test
origin.

## Mock and staging fixtures

The non-secret fixtures under `backend/internal/oidcprovider/testdata` contain:

- the frozen Discovery document;
- a mock-only public JWKS;
- an ID Token Claim example;
- the fixed client metadata.

The mock JWKS is not the API server's runtime key. Configure the fixed client
secret through the staging secret manager and share it out of band; never
commit it. Runtime JWKS is generated from the persistent API signing key.

## Logging checklist

The application access logger records URL paths, not raw query strings. Audit
body and query redaction covers `code`, `state`, `nonce`, `code_verifier`,
`code_challenge`, `id_token`, `access_token`, `client_secret`, and generic
token/secret fields. The example Caddy access log applies the same query
redaction.

Before enabling production, verify the active reverse proxy and Cloudflare
Logpush/HTTP request logs do not persist raw OIDC query strings. Disable query
string fields for the authorization route or apply equivalent named-parameter
redaction. Authorization parameters may appear transiently in the protocol
request, but must not survive in the final browser URL, Referer, browser
storage, application logs, proxy logs, or Cloudflare logs.
