CREATE TABLE IF NOT EXISTS user_activity_rewards (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    kind VARCHAR(32) NOT NULL,
    source_key VARCHAR(100) NOT NULL,
    amount_cents INTEGER NOT NULL DEFAULT 0 CHECK (amount_cents >= 0),
    reward_date DATE,
    streak INTEGER NOT NULL DEFAULT 0,
    redeem_code_id BIGINT REFERENCES redeem_codes(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (kind, source_key)
);
CREATE INDEX IF NOT EXISTS idx_activity_rewards_user_date ON user_activity_rewards(user_id, reward_date);
CREATE UNIQUE INDEX IF NOT EXISTS idx_activity_rewards_card ON user_activity_rewards(redeem_code_id) WHERE redeem_code_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS user_checkin_months (
    user_id BIGINT NOT NULL REFERENCES users(id),
    month VARCHAR(7) NOT NULL,
    daily_cents JSONB NOT NULL,
    PRIMARY KEY (user_id, month)
);
