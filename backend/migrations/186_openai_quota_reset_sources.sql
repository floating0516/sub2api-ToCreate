ALTER TABLE openai_quota_cycles
    ADD COLUMN IF NOT EXISTS detected_reset_source VARCHAR(16) NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS reset_source_override VARCHAR(16),
    ADD COLUMN IF NOT EXISTS reset_source_evidence VARCHAR(64),
    ADD COLUMN IF NOT EXISTS reset_credit_snapshot_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reset_credit_expirations JSONB;

UPDATE openai_quota_cycles
SET detected_reset_source = CASE
        WHEN detection_reason = 'manual_reset' THEN 'manual'
        WHEN detection_reason = 'window_elapsed' THEN 'provider'
        ELSE 'unknown'
    END,
    reset_source_evidence = CASE
        WHEN detection_reason = 'manual_reset' THEN 'reset_endpoint'
        WHEN detection_reason = 'window_elapsed' THEN 'window_elapsed'
        ELSE reset_source_evidence
    END
WHERE reset_observed_at IS NOT NULL
  AND detected_reset_source = 'unknown';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'openai_quota_cycles_detected_reset_source_check'
    ) THEN
        ALTER TABLE openai_quota_cycles
            ADD CONSTRAINT openai_quota_cycles_detected_reset_source_check
            CHECK (detected_reset_source IN ('manual', 'provider', 'unknown'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'openai_quota_cycles_reset_source_override_check'
    ) THEN
        ALTER TABLE openai_quota_cycles
            ADD CONSTRAINT openai_quota_cycles_reset_source_override_check
            CHECK (reset_source_override IS NULL OR reset_source_override IN ('manual', 'provider'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'openai_quota_cycles_reset_credit_expirations_check'
    ) THEN
        ALTER TABLE openai_quota_cycles
            ADD CONSTRAINT openai_quota_cycles_reset_credit_expirations_check
            CHECK (
                reset_credit_expirations IS NULL
                OR jsonb_typeof(reset_credit_expirations) = 'array'
            );
    END IF;
END $$;
