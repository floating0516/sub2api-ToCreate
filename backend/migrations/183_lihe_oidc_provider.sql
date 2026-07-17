-- Lihe unified-account OIDC provider storage.
-- Protocol credentials are persisted only as SHA-256 hashes. These tables are
-- intentionally separate from the existing long-lived Lihe API Key OAuth.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS oidc_subject UUID DEFAULT gen_random_uuid();

UPDATE users
SET oidc_subject = gen_random_uuid()
WHERE oidc_subject IS NULL;

ALTER TABLE users
    ALTER COLUMN oidc_subject SET DEFAULT gen_random_uuid(),
    ALTER COLUMN oidc_subject SET NOT NULL;

-- This is deliberately not a partial index: soft-deleted users keep ownership
-- of their OIDC subject and can only be restored as the same account.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oidc_subject_unique
    ON users (oidc_subject);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS email_verification_source VARCHAR(64);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'users_email_verification_evidence_consistent'
          AND conrelid = 'users'::regclass
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT users_email_verification_evidence_consistent CHECK (
                (email_verified_at IS NULL AND email_verification_source IS NULL)
                OR
                (email_verified_at IS NOT NULL AND NULLIF(BTRIM(email_verification_source), '') IS NOT NULL)
            );
    END IF;
END;
$$;

CREATE OR REPLACE FUNCTION clear_user_email_verification_on_change()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF LOWER(BTRIM(NEW.email)) IS DISTINCT FROM LOWER(BTRIM(OLD.email)) THEN
        NEW.email_verified_at := NULL;
        NEW.email_verification_source := NULL;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_users_email_verification_reset ON users;
CREATE TRIGGER trg_users_email_verification_reset
BEFORE UPDATE OF email ON users
FOR EACH ROW
EXECUTE FUNCTION clear_user_email_verification_on_change();

CREATE OR REPLACE FUNCTION prevent_user_oidc_subject_change()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.oidc_subject IS DISTINCT FROM OLD.oidc_subject THEN
        RAISE EXCEPTION 'users.oidc_subject is immutable'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_users_oidc_subject_immutable ON users;
CREATE TRIGGER trg_users_oidc_subject_immutable
BEFORE UPDATE OF oidc_subject ON users
FOR EACH ROW
EXECUTE FUNCTION prevent_user_oidc_subject_change();

-- A short-lived, opaque handle replaces the full OIDC query while the API SPA
-- sends an unauthenticated user through the existing login page.
CREATE TABLE IF NOT EXISTS lihe_oidc_pending_requests (
    id BIGSERIAL PRIMARY KEY,
    request_hash CHAR(64) NOT NULL UNIQUE,
    browser_binding_hash CHAR(64) NOT NULL,
    request_params JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lihe_oidc_pending_request_hash_format
        CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT lihe_oidc_pending_browser_hash_format
        CHECK (browser_binding_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT lihe_oidc_pending_params_object
        CHECK (
            jsonb_typeof(request_params) = 'object'
            AND request_params->>'v' = '1'
            AND NULLIF(request_params->>'n', '') IS NOT NULL
            AND NULLIF(request_params->>'c', '') IS NOT NULL
        )
);

CREATE INDEX IF NOT EXISTS idx_lihe_oidc_pending_expires_at
    ON lihe_oidc_pending_requests (expires_at);

-- Fosite supplies both an opaque code and an independently derived signature.
-- Hash both forms so neither a usable code nor its lookup signature is stored.
CREATE TABLE IF NOT EXISTS lihe_oidc_authorization_codes (
    id BIGSERIAL PRIMARY KEY,
    request_id UUID NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    signature_hash CHAR(64) NOT NULL UNIQUE,
    code_hash CHAR(64) UNIQUE,
    client_id VARCHAR(100) NOT NULL,
    redirect_uri TEXT NOT NULL,
    scopes TEXT[] NOT NULL,
    code_challenge_hash CHAR(64) NOT NULL,
    code_challenge_method VARCHAR(10) NOT NULL,
    nonce_hash CHAR(64) NOT NULL,
    request_data JSONB NOT NULL,
    oidc_request_data JSONB,
    pkce_request_data JSONB,
    expires_at TIMESTAMPTZ NOT NULL,
    invalidated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lihe_oidc_code_signature_hash_format
        CHECK (signature_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT lihe_oidc_code_hash_format
        CHECK (code_hash IS NULL OR code_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT lihe_oidc_code_nonce_hash_format
        CHECK (nonce_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT lihe_oidc_code_challenge_hash_format
        CHECK (code_challenge_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT lihe_oidc_code_method_s256
        CHECK (code_challenge_method = 'S256'),
    CONSTRAINT lihe_oidc_code_request_object
        CHECK (jsonb_typeof(request_data) = 'object' AND request_data->>'v' = '1'),
    CONSTRAINT lihe_oidc_code_oidc_request_object
        CHECK (oidc_request_data IS NULL OR (jsonb_typeof(oidc_request_data) = 'object' AND oidc_request_data->>'v' = '1')),
    CONSTRAINT lihe_oidc_code_pkce_request_object
        CHECK (pkce_request_data IS NULL OR (jsonb_typeof(pkce_request_data) = 'object' AND pkce_request_data->>'v' = '1'))
);

CREATE INDEX IF NOT EXISTS idx_lihe_oidc_codes_expires_at
    ON lihe_oidc_authorization_codes (expires_at);

CREATE INDEX IF NOT EXISTS idx_lihe_oidc_codes_user_id
    ON lihe_oidc_authorization_codes (user_id);

CREATE TABLE IF NOT EXISTS lihe_oidc_access_tokens (
    id BIGSERIAL PRIMARY KEY,
    request_id UUID NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    oidc_subject UUID NOT NULL,
    signature_hash CHAR(64) NOT NULL UNIQUE,
    client_id VARCHAR(100) NOT NULL,
    scopes TEXT[] NOT NULL,
    request_data JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lihe_oidc_access_signature_hash_format
        CHECK (signature_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT lihe_oidc_access_request_object
        CHECK (jsonb_typeof(request_data) = 'object' AND request_data->>'v' = '1')
);

CREATE INDEX IF NOT EXISTS idx_lihe_oidc_access_request_id
    ON lihe_oidc_access_tokens (request_id);

CREATE INDEX IF NOT EXISTS idx_lihe_oidc_access_user_active
    ON lihe_oidc_access_tokens (user_id, expires_at)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_lihe_oidc_access_expires_at
    ON lihe_oidc_access_tokens (expires_at);
