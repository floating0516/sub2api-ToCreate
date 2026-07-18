CREATE TABLE IF NOT EXISTS openai_quota_cycles (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    window_type VARCHAR(16) NOT NULL DEFAULT '7d',
    cycle_started_at TIMESTAMPTZ NOT NULL,
    last_observed_at TIMESTAMPTZ NOT NULL,
    last_used_percent DOUBLE PRECISION NOT NULL,
    peak_used_percent DOUBLE PRECISION NOT NULL,
    provider_reset_at TIMESTAMPTZ,
    reset_observed_at TIMESTAMPTZ,
    reset_to_percent DOUBLE PRECISION,
    detection_reason VARCHAR(32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT openai_quota_cycles_window_type_check
        CHECK (window_type IN ('7d')),
    CONSTRAINT openai_quota_cycles_last_used_percent_check
        CHECK (last_used_percent >= 0 AND last_used_percent <= 100),
    CONSTRAINT openai_quota_cycles_peak_used_percent_check
        CHECK (peak_used_percent >= 0 AND peak_used_percent <= 100),
    CONSTRAINT openai_quota_cycles_reset_to_percent_check
        CHECK (reset_to_percent IS NULL OR (reset_to_percent >= 0 AND reset_to_percent <= 100))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_openai_quota_cycles_active
    ON openai_quota_cycles (account_id, window_type)
    WHERE reset_observed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_openai_quota_cycles_history
    ON openai_quota_cycles (account_id, window_type, reset_observed_at DESC)
    WHERE reset_observed_at IS NOT NULL;
