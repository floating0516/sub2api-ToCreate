-- Benefit-grant audience previews query active non-admin users by a local-day
-- range on last_active_at. Keep that query index-backed without indexing
-- disabled, deleted, or administrative accounts.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_today_active_audience
    ON users (last_active_at, id)
    WHERE deleted_at IS NULL
      AND status = 'active'
      AND role = 'user'
      AND last_active_at IS NOT NULL;
