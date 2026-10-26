package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

type dailyResetTestRepo struct {
	transactionalBulkSubscriptionRepo
}

func (r *dailyResetTestRepo) GetByID(ctx context.Context, id int64) (*UserSubscription, error) {
	if dbent.TxFromContext(ctx) == nil {
		copy := r.committed
		return &copy, nil
	}
	return r.transactionalBulkSubscriptionRepo.GetByID(ctx, id)
}

type dailyResetTestGroupRepo struct {
	GroupRepository
	group *Group
}

func (r *dailyResetTestGroupRepo) GetByID(context.Context, int64) (*Group, error) {
	return r.group, nil
}

func TestDailyQuotaResetTransaction(t *testing.T) {
	for _, scenario := range []string{"success", "replay", "conflict", "audit failure", "daily limit"} {
		t.Run(scenario, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			t.Cleanup(func() { _ = client.Close() })
			now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
			repo := &dailyResetTestRepo{transactionalBulkSubscriptionRepo{
				committed: UserSubscription{ID: 1, UserID: 10, GroupID: 20, StartsAt: now.Add(-time.Hour), ExpiresAt: now.Add(10 * 24 * time.Hour), Status: SubscriptionStatusActive, DailyUsageUSD: 8, WeeklyUsageUSD: 17, MonthlyUsageUSD: 30},
				pending:   make(map[*dbent.Tx]*UserSubscription), reads: make(map[*dbent.Tx]int),
			}}
			limit := 10.0
			groups := &dailyResetTestGroupRepo{group: &Group{Status: StatusActive, SubscriptionType: "subscription", DailyLimitUSD: &limit}}
			svc := NewSubscriptionService(groups, repo, nil, client, nil)
			svc.now = func() time.Time { return now }
			t.Cleanup(svc.Stop)
			before := repo.committed
			requestID := "7f35ec2b-dd08-4448-8a86-906f33da542e"
			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT .* FROM "users" .* FOR UPDATE`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
			replayRows := sqlmock.NewRows([]string{"subscription_id"})
			if scenario == "replay" {
				replayRows.AddRow(1)
			}
			if scenario == "conflict" {
				replayRows.AddRow(2)
			}
			mock.ExpectQuery(`SELECT subscription_id FROM subscription_daily_quota_resets`).WithArgs(int64(10), requestID).WillReturnRows(replayRows)
			if scenario != "replay" && scenario != "conflict" {
				used := 0
				if scenario == "daily limit" {
					used = 4
				}
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM subscription_daily_quota_resets`).WithArgs(int64(10), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(used))
				if scenario != "daily limit" {
					exec := mock.ExpectExec(`INSERT INTO subscription_daily_quota_resets`).WithArgs(int64(10), int64(1), requestID, now, before.ExpiresAt, before.ExpiresAt.Add(-24*time.Hour), float64(8))
					if scenario == "audit failure" {
						exec.WillReturnError(errors.New("audit unavailable"))
					} else {
						exec.WillReturnResult(sqlmock.NewResult(1, 1))
					}
				}
			}
			if scenario == "success" || scenario == "replay" {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			result, err := svc.ResetOwnDailyQuota(context.Background(), 10, 1, requestID)
			if scenario == "success" {
				require.NoError(t, err)
				require.Equal(t, before.ExpiresAt.Add(-24*time.Hour), result.ExpiresAt)
				require.Zero(t, result.DailyUsageUSD)
				require.Equal(t, before.WeeklyUsageUSD, result.WeeklyUsageUSD)
				require.Equal(t, before.MonthlyUsageUSD, result.MonthlyUsageUSD)
			} else if scenario == "replay" {
				require.NoError(t, err)
				require.Equal(t, before, *result)
			} else {
				require.Error(t, err)
				require.Equal(t, before, repo.committed)
				if scenario == "conflict" {
					require.ErrorIs(t, err, ErrDailyResetRequestConflict)
				}
				if scenario == "daily limit" {
					require.ErrorIs(t, err, ErrDailyResetLimit)
				}
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestValidateDailyQuotaReset(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	limit := 10.0
	for _, tc := range []struct {
		name   string
		change func(*UserSubscription, *Group)
		used   int
		want   error
	}{
		{name: "exactly 72 hours allowed"},
		{name: "less than 72 hours", change: func(s *UserSubscription, _ *Group) { s.ExpiresAt = s.ExpiresAt.Add(-time.Nanosecond) }, want: ErrDailyResetTooShort},
		{name: "another user", change: func(s *UserSubscription, _ *Group) { s.UserID = 2 }, want: ErrSubscriptionNotFound},
		{name: "fifth reset blocked", used: 4, want: ErrDailyResetLimit},
		{name: "fourth reset allowed", used: 3},
		{name: "expired", change: func(s *UserSubscription, _ *Group) { s.ExpiresAt = now }, want: ErrDailyResetUnavailable},
		{name: "suspended", change: func(s *UserSubscription, _ *Group) { s.Status = "suspended" }, want: ErrDailyResetUnavailable},
		{name: "not started", change: func(s *UserSubscription, _ *Group) { s.StartsAt = now.Add(time.Second) }, want: ErrDailyResetUnavailable},
		{name: "no daily quota", change: func(_ *UserSubscription, g *Group) { g.DailyLimitUSD = nil }, want: ErrDailyResetUnavailable},
		{name: "disabled group", change: func(_ *UserSubscription, g *Group) { g.Status = "disabled" }, want: ErrDailyResetUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sub := &UserSubscription{UserID: 1, StartsAt: now.Add(-time.Hour), ExpiresAt: now.Add(72 * time.Hour), Status: SubscriptionStatusActive}
			group := &Group{SubscriptionType: "subscription", Status: StatusActive, DailyLimitUSD: &limit}
			if tc.change != nil {
				tc.change(sub, group)
			}
			require.ErrorIs(t, validateDailyQuotaReset(sub, group, 1, tc.used, now), tc.want)
		})
	}
	require.ErrorIs(t, validateDailyQuotaReset(nil, nil, 1, 0, now), ErrSubscriptionNotFound)
}
