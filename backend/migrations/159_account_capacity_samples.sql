CREATE TABLE IF NOT EXISTS account_capacity_samples (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    sampled_at TIMESTAMPTZ NOT NULL,
    current_concurrency BIGINT NOT NULL DEFAULT 0,
    max_concurrency BIGINT NOT NULL DEFAULT 0,
    waiting_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_capacity_samples_account_sampled_at
    ON account_capacity_samples (account_id, sampled_at DESC);

CREATE INDEX IF NOT EXISTS idx_account_capacity_samples_sampled_at
    ON account_capacity_samples (sampled_at DESC);
