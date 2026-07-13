CREATE TABLE IF NOT EXISTS subscription_addon_packs (
    id BIGSERIAL PRIMARY KEY,
    subscription_id BIGINT NOT NULL REFERENCES user_subscriptions(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    group_id BIGINT NOT NULL REFERENCES groups(id),
    quota_usd NUMERIC(20, 10) NOT NULL,
    used_usd NUMERIC(20, 10) NOT NULL DEFAULT 0,
    starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    assigned_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT NOT NULL DEFAULT '',
    revoked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT subscription_addon_packs_quota_positive CHECK (quota_usd > 0),
    CONSTRAINT subscription_addon_packs_used_nonnegative CHECK (used_usd >= 0),
    CONSTRAINT subscription_addon_packs_window_valid CHECK (expires_at > starts_at),
    CONSTRAINT subscription_addon_packs_status_valid CHECK (status IN ('active', 'exhausted', 'revoked'))
);

CREATE INDEX IF NOT EXISTS subscription_addon_packs_subscription_idx
    ON subscription_addon_packs (subscription_id, created_at DESC);

CREATE INDEX IF NOT EXISTS subscription_addon_packs_active_lookup_idx
    ON subscription_addon_packs (subscription_id, expires_at, id)
    WHERE status = 'active';

CREATE TABLE IF NOT EXISTS subscription_addon_usage (
    id BIGSERIAL PRIMARY KEY,
    addon_pack_id BIGINT NOT NULL REFERENCES subscription_addon_packs(id),
    subscription_id BIGINT NOT NULL REFERENCES user_subscriptions(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id),
    request_id VARCHAR(64) NOT NULL,
    cost_usd NUMERIC(20, 10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT subscription_addon_usage_cost_positive CHECK (cost_usd > 0),
    CONSTRAINT subscription_addon_usage_request_unique UNIQUE (request_id, api_key_id)
);

CREATE INDEX IF NOT EXISTS subscription_addon_usage_pack_created_idx
    ON subscription_addon_usage (addon_pack_id, created_at DESC);

CREATE INDEX IF NOT EXISTS subscription_addon_usage_subscription_created_idx
    ON subscription_addon_usage (subscription_id, created_at DESC);
