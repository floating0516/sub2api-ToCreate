CREATE TABLE IF NOT EXISTS subscription_term_snapshots (
    id BIGSERIAL PRIMARY KEY,
    subscription_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    settled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    settlement_reason VARCHAR(32) NOT NULL,
    total_quota_usd NUMERIC(20, 8) NOT NULL DEFAULT 0,
    used_quota_usd NUMERIC(20, 8) NOT NULL DEFAULT 0,
    unused_quota_usd NUMERIC(20, 8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT subscription_term_snapshots_term_unique
        UNIQUE (subscription_id, starts_at, expires_at)
);

CREATE INDEX IF NOT EXISTS subscription_term_snapshots_user_id_idx
    ON subscription_term_snapshots (user_id);

CREATE INDEX IF NOT EXISTS subscription_term_snapshots_settled_at_idx
    ON subscription_term_snapshots (settled_at DESC);

