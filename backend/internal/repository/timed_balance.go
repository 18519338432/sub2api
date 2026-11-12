package repository

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// Read live grants instead of caching a total that could survive an expiry.
func (r *userRepository) GetSpendableBalance(ctx context.Context, userID int64) (float64, error) {
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `SELECT balance + COALESCE((
 SELECT SUM(remaining) FROM user_timed_balances WHERE user_id = users.id
 AND expires_at > clock_timestamp() AND remaining > 0), 0)
 FROM users WHERE id = $1 AND deleted_at IS NULL`, userID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, service.ErrUserNotFound
	}
	var balance float64
	err = rows.Scan(&balance)
	return balance, err
}
