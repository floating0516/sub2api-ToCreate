-- Lihe Chat confidential OAuth client and provider bindings.
-- Authorization codes and access tokens are stored only as SHA-256 hashes.

CREATE TABLE IF NOT EXISTS lihe_oauth_authorization_codes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash CHAR(64) NOT NULL UNIQUE,
    client_id VARCHAR(100) NOT NULL,
    redirect_uri TEXT NOT NULL,
    scopes TEXT[] NOT NULL,
    code_challenge VARCHAR(128) NOT NULL,
    code_challenge_method VARCHAR(10) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lihe_oauth_code_hash_format CHECK (code_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT lihe_oauth_code_method_s256 CHECK (code_challenge_method = 'S256')
);

CREATE INDEX IF NOT EXISTS idx_lihe_oauth_codes_expires_at
    ON lihe_oauth_authorization_codes (expires_at);

CREATE INDEX IF NOT EXISTS idx_lihe_oauth_codes_user_id
    ON lihe_oauth_authorization_codes (user_id);

CREATE TABLE IF NOT EXISTS lihe_oauth_access_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash CHAR(64) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL DEFAULT 'lihe.chat',
    client_id VARCHAR(100) NOT NULL,
    scopes TEXT[] NOT NULL,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lihe_oauth_token_hash_format CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT lihe_oauth_token_fixed_name CHECK (name = 'lihe.chat')
);

CREATE INDEX IF NOT EXISTS idx_lihe_oauth_tokens_user_active
    ON lihe_oauth_access_tokens (user_id, created_at DESC)
    WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS lihe_oauth_token_bindings (
    id BIGSERIAL PRIMARY KEY,
    token_id BIGINT NOT NULL REFERENCES lihe_oauth_access_tokens(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lihe_oauth_binding_provider CHECK (
        provider IN ('openai', 'anthropic', 'gemini', 'antigravity', 'grok')
    ),
    CONSTRAINT lihe_oauth_binding_token_provider_unique UNIQUE (token_id, provider),
    CONSTRAINT lihe_oauth_binding_api_key_unique UNIQUE (api_key_id)
);

CREATE INDEX IF NOT EXISTS idx_lihe_oauth_bindings_token_id
    ON lihe_oauth_token_bindings (token_id);
