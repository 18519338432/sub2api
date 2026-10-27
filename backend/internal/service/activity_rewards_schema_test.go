package service

import (
	"context"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func ensureActivityRewardTestSchema(t *testing.T, client *dbent.Client) {
	t.Helper()
	_, err := client.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS user_activity_rewards (
		id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, kind TEXT NOT NULL,
		source_key TEXT NOT NULL, amount_cents INTEGER NOT NULL DEFAULT 0,
		reward_date TEXT, streak INTEGER NOT NULL DEFAULT 0,
		redeem_code_id INTEGER, created_at TEXT DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(kind, source_key))`)
	require.NoError(t, err)
}
