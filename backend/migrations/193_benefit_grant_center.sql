CREATE TABLE IF NOT EXISTS benefit_grant_campaigns (
    id BIGSERIAL PRIMARY KEY,
    operation_key VARCHAR(128) NOT NULL,
    request_hash CHAR(64) NOT NULL,
    audience_type VARCHAR(32) NOT NULL,
    audience_date DATE NOT NULL,
    audience_days INTEGER NOT NULL DEFAULT 1,
    timezone VARCHAR(64) NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    benefit_type VARCHAR(32) NOT NULL,
    conflict_policy VARCHAR(32) NOT NULL DEFAULT 'none',
    group_id BIGINT,
    group_name_snapshot VARCHAR(100) NOT NULL DEFAULT '',
    validity_days INTEGER,
    balance_amount DECIMAL(20, 8),
    notes TEXT NOT NULL DEFAULT '',
    marker VARCHAR(192) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'running',
    matched_count INTEGER NOT NULL DEFAULT 0,
    eligible_count INTEGER NOT NULL DEFAULT 0,
    already_granted_count INTEGER NOT NULL DEFAULT 0,
    conflict_count INTEGER NOT NULL DEFAULT 0,
    granted_count INTEGER NOT NULL DEFAULT 0,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    created_count INTEGER NOT NULL DEFAULT 0,
    renewed_count INTEGER NOT NULL DEFAULT 0,
    extended_count INTEGER NOT NULL DEFAULT 0,
    balance_granted_count INTEGER NOT NULL DEFAULT 0,
    created_by BIGINT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT benefit_grant_campaigns_operation_key_uq UNIQUE (created_by, operation_key),
    CONSTRAINT benefit_grant_campaigns_request_hash_check
        CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT benefit_grant_campaigns_audience_days_check
        CHECK (audience_days BETWEEN 1 AND 365),
    CONSTRAINT benefit_grant_campaigns_window_check
        CHECK (window_end > window_start),
    CONSTRAINT benefit_grant_campaigns_status_check
        CHECK (status IN ('running', 'completed', 'partial', 'failed')),
    CONSTRAINT benefit_grant_campaigns_counts_check
        CHECK (
            matched_count >= 0
            AND eligible_count >= 0
            AND already_granted_count >= 0
            AND conflict_count >= 0
            AND granted_count >= 0
            AND skipped_count >= 0
            AND failed_count >= 0
            AND created_count >= 0
            AND renewed_count >= 0
            AND extended_count >= 0
            AND balance_granted_count >= 0
        )
);

CREATE INDEX IF NOT EXISTS idx_benefit_grant_campaigns_created_at
    ON benefit_grant_campaigns (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_benefit_grant_campaigns_status
    ON benefit_grant_campaigns (status, created_at DESC);

CREATE TABLE IF NOT EXISTS benefit_grant_recipients (
    id BIGSERIAL PRIMARY KEY,
    campaign_id BIGINT NOT NULL
        REFERENCES benefit_grant_campaigns(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    email_snapshot VARCHAR(255) NOT NULL DEFAULT '',
    username_snapshot VARCHAR(100) NOT NULL DEFAULT '',
    eligibility VARCHAR(24) NOT NULL,
    planned_action VARCHAR(32) NOT NULL,
    status VARCHAR(20) NOT NULL,
    result_type VARCHAR(32),
    subscription_id BIGINT,
    balance_before DECIMAL(20, 8),
    balance_after DECIMAL(20, 8),
    error TEXT,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT benefit_grant_recipients_campaign_user_uq UNIQUE (campaign_id, user_id),
    CONSTRAINT benefit_grant_recipients_eligibility_check
        CHECK (eligibility IN ('eligible', 'already_granted', 'conflict')),
    CONSTRAINT benefit_grant_recipients_status_check
        CHECK (status IN ('pending', 'processing', 'granted', 'skipped', 'failed')),
    CONSTRAINT benefit_grant_recipients_attempt_count_check
        CHECK (attempt_count >= 0)
);

CREATE INDEX IF NOT EXISTS idx_benefit_grant_recipients_campaign_status
    ON benefit_grant_recipients (campaign_id, status, id);

CREATE INDEX IF NOT EXISTS idx_benefit_grant_recipients_user
    ON benefit_grant_recipients (user_id, created_at DESC);

COMMENT ON TABLE benefit_grant_campaigns IS
    'Admin benefit grant batches with immutable audience and benefit parameters.';
COMMENT ON TABLE benefit_grant_recipients IS
    'Per-user audience snapshot and execution result for a benefit grant batch.';
