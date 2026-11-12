package service

import (
	"context"
	"math"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const RedeemTypeTimedBalance = "timed_balance"

func (s *SubscriptionService) SpendableBalance(ctx context.Context, userID int64, fallback float64) (float64, error) {
	if s.entClient == nil {
		return fallback, nil
	}
	rows, err := s.entClient.QueryContext(ctx, `SELECT balance + COALESCE((SELECT SUM(remaining) FROM user_timed_balances WHERE user_id = users.id AND remaining > 0 AND expires_at > clock_timestamp()), 0) FROM users WHERE id = $1 AND deleted_at IS NULL`, userID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, ErrUserNotFound
	}
	var balance float64
	err = rows.Scan(&balance)
	return balance, err
}

type TimedBalance struct {
	ID        int64     `json:"id"`
	Amount    float64   `json:"amount"`
	Remaining float64   `json:"remaining"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Status    string    `json:"status"`
}

func validateTimedBalance(amount float64, days int) error {
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 || days < 1 || days > 3650 {
		return infraerrors.BadRequest("INVALID_TIMED_BALANCE", "timed balance requires a positive amount and validity_days between 1 and 3650")
	}
	return nil
}

func (s *SubscriptionService) ListTimedBalances(ctx context.Context, userID int64) ([]TimedBalance, error) {
	rows, err := s.entClient.QueryContext(ctx, `SELECT id, amount, remaining, created_at, expires_at,
 CASE WHEN expires_at <= clock_timestamp() THEN 'expired' WHEN remaining = 0 THEN 'exhausted' ELSE 'active' END
 FROM user_timed_balances WHERE user_id = $1 ORDER BY expires_at, id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]TimedBalance, 0)
	for rows.Next() {
		var item TimedBalance
		if err := rows.Scan(&item.ID, &item.Amount, &item.Remaining, &item.CreatedAt, &item.ExpiresAt, &item.Status); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
