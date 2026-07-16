-- Bind each Lihe authorization and access token to one user-selected API key.
-- Existing authorization codes are ephemeral and cannot be upgraded safely,
-- so discard them before activating selected-key semantics.

ALTER TABLE lihe_oauth_authorization_codes
    ADD COLUMN IF NOT EXISTS api_key_id BIGINT;

DELETE FROM lihe_oauth_authorization_codes
WHERE api_key_id IS NULL;

-- Keep the column nullable at the schema level so the previous application
-- image can still be used for rollback. The new application requires a
-- positive api_key_id and rejects rollback-era codes that contain NULL.

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'lihe_oauth_authorization_codes_api_key_id_fkey'
          AND conrelid = 'lihe_oauth_authorization_codes'::regclass
    ) THEN
        ALTER TABLE lihe_oauth_authorization_codes
            ADD CONSTRAINT lihe_oauth_authorization_codes_api_key_id_fkey
            FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE;
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_lihe_oauth_codes_api_key_id
    ON lihe_oauth_authorization_codes (api_key_id);

-- Keep the legacy api_key_id nullable and store new direct bindings separately.
-- If the application is rolled back, the old resolver cannot use a new token
-- and its old revoke query cannot disable the user's source API key.
ALTER TABLE lihe_oauth_token_bindings
    ALTER COLUMN api_key_id DROP NOT NULL;

ALTER TABLE lihe_oauth_token_bindings
    ADD COLUMN IF NOT EXISTS source_api_key_id BIGINT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'lihe_oauth_token_bindings_source_api_key_id_fkey'
          AND conrelid = 'lihe_oauth_token_bindings'::regclass
    ) THEN
        ALTER TABLE lihe_oauth_token_bindings
            ADD CONSTRAINT lihe_oauth_token_bindings_source_api_key_id_fkey
            FOREIGN KEY (source_api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE;
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_lihe_oauth_bindings_source_api_key_id
    ON lihe_oauth_token_bindings (source_api_key_id);

-- Reconnect can briefly create a replacement token for the same source key.
ALTER TABLE lihe_oauth_token_bindings
    DROP CONSTRAINT IF EXISTS lihe_oauth_binding_api_key_unique;

-- A hard-deleted source key invalidates the connection instead of blocking the
-- key deletion. Normal user deletion remains a soft delete.
ALTER TABLE lihe_oauth_token_bindings
    DROP CONSTRAINT IF EXISTS lihe_oauth_token_bindings_api_key_id_fkey;

ALTER TABLE lihe_oauth_token_bindings
    ADD CONSTRAINT lihe_oauth_token_bindings_api_key_id_fkey
    FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE;
