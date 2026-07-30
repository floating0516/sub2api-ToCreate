CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_recent_registered_audience
    ON users (created_at, id)
    WHERE deleted_at IS NULL
      AND status = 'active'
      AND role = 'user';
