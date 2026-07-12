ALTER TABLE subscription_term_snapshots
    ADD COLUMN IF NOT EXISTS overage_usd NUMERIC(20, 8) NOT NULL DEFAULT 0;

