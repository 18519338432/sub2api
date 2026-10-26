package service

import (
	"context"
	"fmt"
	"log"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/user"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

const DailyQuotaResetLimit = 4

var (
	ErrDailyResetTooShort        = infraerrors.BadRequest("DAILY_RESET_TOO_SHORT", "subscription must have at least 3 full days remaining")
	ErrDailyResetLimit           = infraerrors.TooManyRequests("DAILY_RESET_LIMIT", "daily quota can only be reset 4 times per user per day")
	ErrDailyResetUnavailable     = infraerrors.BadRequest("DAILY_RESET_UNAVAILABLE", "only active subscriptions with a daily quota can be reset")
	ErrDailyResetRequestConflict = infraerrors.Conflict("DAILY_RESET_REQUEST_CONFLICT", "reset request belongs to another subscription")
)

type DailyQuotaResetStatus struct {
	Used      int       `json:"used"`
	Remaining int       `json:"remaining"`
	Limit     int       `json:"limit"`
	ResetsAt  time.Time `json:"resets_at"`
}

func (s *SubscriptionService) dailyQuotaResetStatus(ctx context.Context, client *dbent.Client, userID int64, now time.Time) (*DailyQuotaResetStatus, error) {
	day := timezone.StartOfDay(now)
	nextDay := day.AddDate(0, 0, 1)
	rows, err := client.QueryContext(ctx, `SELECT COUNT(*) FROM subscription_daily_quota_resets WHERE user_id = $1 AND reset_at >= $2 AND reset_at < $3`, userID, day, nextDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var used int
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("read daily reset count: %w", err)
		}
		return nil, fmt.Errorf("daily reset count returned no row")
	}
	if err := rows.Scan(&used); err != nil {
		return nil, err
	}
	return &DailyQuotaResetStatus{Used: used, Remaining: max(0, DailyQuotaResetLimit-used), Limit: DailyQuotaResetLimit, ResetsAt: nextDay}, nil
}

func (s *SubscriptionService) GetDailyQuotaResetStatus(ctx context.Context, userID int64) (*DailyQuotaResetStatus, error) {
	return s.dailyQuotaResetStatus(ctx, s.entClient, userID, s.now())
}

func validateDailyQuotaReset(sub *UserSubscription, group *Group, userID int64, used int, now time.Time) error {
	// Return the same error for an unknown ID and another user's subscription.
	if sub == nil || sub.UserID != userID {
		return ErrSubscriptionNotFound
	}
	if sub.Status != SubscriptionStatusActive || sub.StartsAt.After(now) || !sub.ExpiresAt.After(now) || group == nil || group.Status != StatusActive || !group.IsSubscriptionType() || group.DailyLimitUSD == nil || *group.DailyLimitUSD <= 0 {
		return ErrDailyResetUnavailable
	}
	if sub.ExpiresAt.Sub(now) < 3*subscriptionDayDuration {
		return ErrDailyResetTooShort
	}
	if used >= DailyQuotaResetLimit {
		return ErrDailyResetLimit
	}
	return nil
}

// ResetOwnDailyQuota commits the quota change, one-day cost and audit record
// together. Locking the user first enforces the cap across different subscriptions.
func (s *SubscriptionService) ResetOwnDailyQuota(ctx context.Context, userID, subscriptionID int64, requestID string) (*UserSubscription, error) {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	client := tx.Client()
	txCtx := dbent.NewTxContext(ctx, tx)
	if _, err := client.User.Query().Where(user.IDEQ(userID)).ForUpdate().Only(ctx); err != nil {
		return nil, err
	}
	sub, err := s.userSubRepo.GetByIDForUpdate(txCtx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub.UserID != userID {
		return nil, ErrSubscriptionNotFound
	}

	rows, err := client.QueryContext(ctx, `SELECT subscription_id FROM subscription_daily_quota_resets WHERE user_id = $1 AND request_id = $2`, userID, requestID)
	if err != nil {
		return nil, err
	}
	var previousID int64
	replay := rows.Next()
	if replay {
		err = rows.Scan(&previousID)
	} else {
		err = rows.Err()
	}
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	if replay {
		if previousID != subscriptionID {
			return nil, ErrDailyResetRequestConflict
		}
	} else {
		// Capture time after acquiring locks, so a wait across midnight uses the
		// correct day and the three-day check uses the actual remaining term.
		now := s.now()
		status, err := s.dailyQuotaResetStatus(ctx, client, userID, now)
		if err != nil {
			return nil, err
		}
		group, err := s.groupRepo.GetByID(txCtx, sub.GroupID)
		if err != nil {
			return nil, err
		}
		if err := validateDailyQuotaReset(sub, group, userID, status.Used, now); err != nil {
			return nil, err
		}
		expiresAt := sub.ExpiresAt.Add(-subscriptionDayDuration)
		if err := s.userSubRepo.ExtendExpiry(txCtx, sub.ID, expiresAt); err != nil {
			return nil, err
		}
		if err := s.userSubRepo.ResetUsageWindows(txCtx, sub.ID, true, false, false, timezone.StartOfDay(now), now); err != nil {
			return nil, err
		}
		_, err = client.ExecContext(ctx, `INSERT INTO subscription_daily_quota_resets (user_id, subscription_id, request_id, reset_at, previous_expires_at, new_expires_at, previous_daily_usage_usd) VALUES ($1,$2,$3,$4,$5,$6,$7)`, userID, sub.ID, requestID, now, sub.ExpiresAt, expiresAt, sub.DailyUsageUSD)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if err := s.invalidateSubscriptionCaches(sub.UserID, sub.GroupID); err != nil {
		// The operation is already committed. A response error would invite a
		// second purchase; caches also have bounded TTLs.
		log.Printf("Warning: daily quota reset cache invalidation: %v", err)
	}
	return s.userSubRepo.GetByID(ctx, subscriptionID)
}
