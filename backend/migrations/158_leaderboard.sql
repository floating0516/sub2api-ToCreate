-- Token leaderboard privacy preferences and reward generation records.

CREATE TABLE IF NOT EXISTS leaderboard_preferences (
    user_id     BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    anonymous   BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS leaderboard_rewards (
    id              BIGSERIAL PRIMARY KEY,
    period          VARCHAR(16) NOT NULL,
    period_start    TIMESTAMPTZ NOT NULL,
    period_end      TIMESTAMPTZ NOT NULL,
    rank            INT NOT NULL,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_count     BIGINT NOT NULL DEFAULT 0,
    reward_type     VARCHAR(32) NOT NULL,
    redeem_code_id  BIGINT NOT NULL REFERENCES redeem_codes(id) ON DELETE CASCADE,
    created_by      BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT leaderboard_rewards_period_check CHECK (period IN ('week', 'month')),
    CONSTRAINT leaderboard_rewards_rank_check CHECK (rank > 0),
    CONSTRAINT leaderboard_rewards_token_count_check CHECK (token_count >= 0),
    CONSTRAINT leaderboard_rewards_reward_type_check CHECK (reward_type IN ('daily_card', 'weekly_card'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_leaderboard_rewards_unique
    ON leaderboard_rewards (period, period_start, rank, reward_type);

CREATE INDEX IF NOT EXISTS idx_leaderboard_rewards_user_created
    ON leaderboard_rewards (user_id, created_at DESC);

