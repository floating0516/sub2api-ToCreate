-- Account contribution marketplace foundation.
--
-- This migration is intentionally inert: all feature gates are seeded off and
-- no existing account is linked to a contributor. Later releases can add user
-- submission and settlement workers without changing these financial records.

INSERT INTO settings (key, value, updated_at)
VALUES
    ('account_contribution_enabled', 'false', NOW()),
    ('account_contribution_submission_enabled', 'false', NOW()),
    ('account_contribution_payout_enabled', 'false', NOW())
ON CONFLICT (key) DO NOTHING;

CREATE TABLE IF NOT EXISTS contributor_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    user_email_snapshot VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(24) NOT NULL DEFAULT 'pending',
    agreement_version VARCHAR(64) NOT NULL DEFAULT '',
    agreement_accepted_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT contributor_profiles_status_check
        CHECK (status IN ('pending', 'active', 'suspended', 'rejected', 'closed'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_contributor_profiles_user_id
    ON contributor_profiles (user_id)
    WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_contributor_profiles_status
    ON contributor_profiles (status, created_at DESC);

CREATE TABLE IF NOT EXISTS account_contributions (
    id BIGSERIAL PRIMARY KEY,
    contributor_id BIGINT NOT NULL
        REFERENCES contributor_profiles(id) ON DELETE RESTRICT,
    account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    account_name_snapshot VARCHAR(100) NOT NULL DEFAULT '',
    platform_snapshot VARCHAR(50) NOT NULL DEFAULT '',
    upstream_identity_hash CHAR(64),
    status VARCHAR(24) NOT NULL DEFAULT 'pending_review',
    settlement_mode VARCHAR(24) NOT NULL DEFAULT 'cost_share',
    share_rate_bps INTEGER NOT NULL DEFAULT 0,
    daily_earning_cap_usd DECIMAL(20,8),
    approved_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,
    activated_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT account_contributions_status_check
        CHECK (status IN ('pending_review', 'testing', 'active', 'paused', 'quarantined', 'revoked', 'rejected')),
    CONSTRAINT account_contributions_settlement_mode_check
        CHECK (settlement_mode IN ('cost_share', 'revenue_share', 'fixed_price')),
    CONSTRAINT account_contributions_share_rate_check
        CHECK (share_rate_bps BETWEEN 0 AND 10000),
    CONSTRAINT account_contributions_daily_cap_check
        CHECK (daily_earning_cap_usd IS NULL OR daily_earning_cap_usd >= 0),
    CONSTRAINT account_contributions_identity_hash_check
        CHECK (upstream_identity_hash IS NULL OR upstream_identity_hash ~ '^[0-9a-f]{64}$')
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_account_contributions_account_id
    ON account_contributions (account_id)
    WHERE account_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_account_contributions_upstream_identity
    ON account_contributions (upstream_identity_hash)
    WHERE upstream_identity_hash IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_account_contributions_contributor_status
    ON account_contributions (contributor_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS contributor_earning_ledger (
    id BIGSERIAL PRIMARY KEY,
    contributor_id BIGINT NOT NULL
        REFERENCES contributor_profiles(id) ON DELETE RESTRICT,
    contribution_id BIGINT NOT NULL
        REFERENCES account_contributions(id) ON DELETE RESTRICT,
    source_usage_log_id BIGINT,
    source_request_id VARCHAR(64),
    idempotency_key VARCHAR(160) NOT NULL UNIQUE,
    entry_type VARCHAR(24) NOT NULL,
    basis_cost_usd DECIMAL(20,8) NOT NULL DEFAULT 0,
    share_rate_bps INTEGER NOT NULL,
    fx_rate_cny_per_usd DECIMAL(20,8) NOT NULL,
    amount_cny_fen BIGINT NOT NULL,
    available_at TIMESTAMPTZ NOT NULL,
    reverses_entry_id BIGINT
        REFERENCES contributor_earning_ledger(id) ON DELETE RESTRICT,
    account_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    usage_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    pricing_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT contributor_earning_ledger_entry_type_check
        CHECK (entry_type IN ('accrual', 'reversal', 'adjustment')),
    CONSTRAINT contributor_earning_ledger_basis_cost_check
        CHECK (basis_cost_usd >= 0),
    CONSTRAINT contributor_earning_ledger_share_rate_check
        CHECK (share_rate_bps BETWEEN 0 AND 10000),
    CONSTRAINT contributor_earning_ledger_fx_rate_check
        CHECK (fx_rate_cny_per_usd > 0),
    CONSTRAINT contributor_earning_ledger_amount_check
        CHECK (amount_cny_fen <> 0),
    CONSTRAINT contributor_earning_ledger_reversal_check
        CHECK (entry_type <> 'reversal' OR reverses_entry_id IS NOT NULL)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_contributor_earning_ledger_reversal
    ON contributor_earning_ledger (reverses_entry_id)
    WHERE reverses_entry_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_contributor_earning_ledger_contributor_created
    ON contributor_earning_ledger (contributor_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_contributor_earning_ledger_available
    ON contributor_earning_ledger (contributor_id, available_at, id);

CREATE OR REPLACE FUNCTION prevent_contributor_earning_ledger_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'contributor_earning_ledger is append-only; add a reversal entry instead';
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'trg_contributor_earning_ledger_append_only'
          AND tgrelid = 'contributor_earning_ledger'::regclass
          AND NOT tgisinternal
    ) THEN
        CREATE TRIGGER trg_contributor_earning_ledger_append_only
            BEFORE UPDATE OR DELETE ON contributor_earning_ledger
            FOR EACH ROW
            EXECUTE FUNCTION prevent_contributor_earning_ledger_mutation();
    END IF;
END;
$$;

CREATE TABLE IF NOT EXISTS contributor_payout_methods (
    id BIGSERIAL PRIMARY KEY,
    contributor_id BIGINT NOT NULL
        REFERENCES contributor_profiles(id) ON DELETE RESTRICT,
    method_type VARCHAR(24) NOT NULL,
    masked_destination VARCHAR(160) NOT NULL DEFAULT '',
    destination_fingerprint_hash CHAR(64),
    encrypted_payload TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT contributor_payout_methods_type_check
        CHECK (method_type IN ('alipay', 'wechat', 'bank', 'manual')),
    CONSTRAINT contributor_payout_methods_status_check
        CHECK (status IN ('active', 'disabled')),
    CONSTRAINT contributor_payout_methods_fingerprint_check
        CHECK (destination_fingerprint_hash IS NULL OR destination_fingerprint_hash ~ '^[0-9a-f]{64}$')
);

CREATE INDEX IF NOT EXISTS idx_contributor_payout_methods_owner
    ON contributor_payout_methods (contributor_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS contributor_payout_requests (
    id BIGSERIAL PRIMARY KEY,
    contributor_id BIGINT NOT NULL
        REFERENCES contributor_profiles(id) ON DELETE RESTRICT,
    payout_method_id BIGINT
        REFERENCES contributor_payout_methods(id) ON DELETE SET NULL,
    amount_cny_fen BIGINT NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'requested',
    payout_method_snapshot_encrypted TEXT NOT NULL,
    external_reference VARCHAR(160),
    reviewed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    review_notes TEXT NOT NULL DEFAULT '',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT contributor_payout_requests_amount_check
        CHECK (amount_cny_fen > 0),
    CONSTRAINT contributor_payout_requests_status_check
        CHECK (status IN ('requested', 'reviewing', 'approved', 'processing', 'paid', 'rejected', 'failed', 'cancelled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_contributor_payout_requests_external_reference
    ON contributor_payout_requests (external_reference)
    WHERE external_reference IS NOT NULL AND external_reference <> '';

CREATE INDEX IF NOT EXISTS idx_contributor_payout_requests_owner_status
    ON contributor_payout_requests (contributor_id, status, requested_at DESC);

CREATE TABLE IF NOT EXISTS contributor_payout_items (
    id BIGSERIAL PRIMARY KEY,
    payout_request_id BIGINT NOT NULL
        REFERENCES contributor_payout_requests(id) ON DELETE RESTRICT,
    earning_entry_id BIGINT NOT NULL UNIQUE
        REFERENCES contributor_earning_ledger(id) ON DELETE RESTRICT,
    amount_cny_fen BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT contributor_payout_items_amount_check
        CHECK (amount_cny_fen > 0)
);

CREATE INDEX IF NOT EXISTS idx_contributor_payout_items_request
    ON contributor_payout_items (payout_request_id, id);

CREATE TABLE IF NOT EXISTS contributor_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    contributor_id BIGINT
        REFERENCES contributor_profiles(id) ON DELETE SET NULL,
    actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(64) NOT NULL,
    target_type VARCHAR(32) NOT NULL,
    target_id BIGINT,
    before_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    after_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_contributor_audit_logs_target
    ON contributor_audit_logs (target_type, target_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_contributor_audit_logs_contributor
    ON contributor_audit_logs (contributor_id, created_at DESC);

COMMENT ON TABLE contributor_profiles IS
    'Account capacity contributors. User deletion preserves the financial identity snapshot.';
COMMENT ON TABLE account_contributions IS
    'Reviewed ownership and settlement policy for an account offered to the platform pool.';
COMMENT ON TABLE contributor_earning_ledger IS
    'Append-only contributor earnings facts with immutable pricing and usage snapshots.';
COMMENT ON COLUMN contributor_earning_ledger.source_usage_log_id IS
    'Informational source ID only; intentionally has no FK because usage logs can be deleted.';
COMMENT ON TABLE contributor_payout_requests IS
    'Contributor RMB payout workflow. No automatic payout provider is enabled by this migration.';
