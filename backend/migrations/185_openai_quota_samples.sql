CREATE TABLE IF NOT EXISTS openai_quota_samples (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    cycle_id BIGINT NOT NULL REFERENCES openai_quota_cycles(id) ON DELETE CASCADE,
    bucket_started_at TIMESTAMPTZ NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    used_percent DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT openai_quota_samples_used_percent_check
        CHECK (used_percent >= 0 AND used_percent <= 100)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_openai_quota_samples_cycle_bucket
    ON openai_quota_samples (cycle_id, bucket_started_at);

CREATE INDEX IF NOT EXISTS idx_openai_quota_samples_account_observed
    ON openai_quota_samples (account_id, observed_at DESC);
