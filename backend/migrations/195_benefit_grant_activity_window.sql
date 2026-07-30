ALTER TABLE benefit_grant_campaigns
    ADD COLUMN IF NOT EXISTS delivery_mode VARCHAR(32) NOT NULL DEFAULT 'snapshot';

ALTER TABLE benefit_grant_campaigns
    DROP CONSTRAINT IF EXISTS benefit_grant_campaigns_delivery_mode_check;

ALTER TABLE benefit_grant_campaigns
    ADD CONSTRAINT benefit_grant_campaigns_delivery_mode_check
        CHECK (delivery_mode IN ('snapshot', 'activity_window'));

ALTER TABLE benefit_grant_campaigns
    DROP CONSTRAINT IF EXISTS benefit_grant_campaigns_status_check;

ALTER TABLE benefit_grant_campaigns
    ADD CONSTRAINT benefit_grant_campaigns_status_check
        CHECK (status IN ('scheduled', 'running', 'completed', 'partial', 'failed'));

CREATE INDEX IF NOT EXISTS idx_benefit_grant_campaigns_activity_window
    ON benefit_grant_campaigns (window_start, window_end, id)
    WHERE delivery_mode = 'activity_window';

CREATE TABLE IF NOT EXISTS benefit_grant_campaign_announcements (
    campaign_id BIGINT PRIMARY KEY
        REFERENCES benefit_grant_campaigns(id) ON DELETE CASCADE,
    announcement_id BIGINT NOT NULL UNIQUE
        REFERENCES announcements(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_benefit_grant_campaign_announcements_announcement
    ON benefit_grant_campaign_announcements (announcement_id);

COMMENT ON COLUMN benefit_grant_campaigns.delivery_mode IS
    'Grant delivery mode: snapshot for immediate batches, activity_window for authenticated visits.';
COMMENT ON TABLE benefit_grant_campaign_announcements IS
    'Announcements visible only to users successfully granted by the linked campaign.';
