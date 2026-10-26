-- User-initiated resets are immutable records. The user row serializes resets
-- across all subscriptions; request_id makes retries safe after a lost response.
CREATE TABLE IF NOT EXISTS subscription_daily_quota_resets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    subscription_id BIGINT NOT NULL REFERENCES user_subscriptions(id),
    request_id UUID NOT NULL,
    reset_at TIMESTAMPTZ NOT NULL,
    previous_expires_at TIMESTAMPTZ NOT NULL,
    new_expires_at TIMESTAMPTZ NOT NULL,
    previous_daily_usage_usd DECIMAL(20,10) NOT NULL,
    UNIQUE (user_id, request_id)
);

CREATE INDEX IF NOT EXISTS idx_subscription_daily_quota_resets_user_time
    ON subscription_daily_quota_resets (user_id, reset_at);
