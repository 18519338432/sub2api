package repository

import (
	"context"
	"encoding/json"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCheckinClaimSeventhDayAndReplay(t *testing.T) {
	for _, replay := range []bool{false, true} {
		t.Run(map[bool]string{false: "seventh_day", true: "replay"}[replay], func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			repo := &affiliateRepository{client: client}
			now := time.Date(2026, 10, 7, 12, 0, 0, 0, service.RewardLocation)
			amounts := make([]int, 31)
			for i := range amounts {
				amounts[i] = 80
			}
			amounts[0] = 100
			raw, _ := json.Marshal(amounts)
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT id FROM users").WithArgs(int64(42)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))
			mock.ExpectQuery("SELECT us.id FROM user_subscriptions").WithArgs(int64(42), now).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
			mock.ExpectQuery("SELECT daily_cents").WithArgs(int64(42), "2026-10").WillReturnRows(sqlmock.NewRows([]string{"daily_cents"}).AddRow(raw))
			rows := sqlmock.NewRows([]string{"date", "amount", "streak"})
			for i := 1; i <= 6; i++ {
				rows.AddRow(time.Date(2026, 10, i, 0, 0, 0, 0, time.UTC).Format("2006-01-02"), amounts[i-1], i)
			}
			if replay {
				rows.AddRow("2026-10-07", 280, 7)
			}
			mock.ExpectQuery("SELECT reward_date::text").WillReturnRows(rows)
			if !replay {
				mock.ExpectExec("INSERT INTO user_activity_rewards").WithArgs(int64(42), "42:2026-10-07", 280, "2026-10-07", 7).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("UPDATE users SET balance").WithArgs(280, int64(42)).WillReturnResult(sqlmock.NewResult(0, 1))
			}
			mock.ExpectCommit()
			result, err := repo.CheckinCalendar(context.Background(), 42, now, true)
			require.NoError(t, err)
			require.Equal(t, 7, result.Streak)
			require.Equal(t, 200, result.Days[6].BonusCents)
			require.Equal(t, 780, result.TotalCents)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCheckinBalanceFailureRollsBackReward(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := &affiliateRepository{client: dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))}
	amounts := make([]int, 30)
	for i := range amounts {
		amounts[i] = 83
	}
	amounts[0] = 93
	raw, _ := json.Marshal(amounts)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM users").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))
	mock.ExpectQuery("SELECT us.id FROM user_subscriptions").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
	mock.ExpectQuery("SELECT daily_cents").WillReturnRows(sqlmock.NewRows([]string{"daily_cents"}).AddRow(raw))
	mock.ExpectQuery("SELECT reward_date::text").WillReturnRows(sqlmock.NewRows([]string{"date", "amount", "streak"}))
	mock.ExpectExec("INSERT INTO user_activity_rewards").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE users SET balance").WillReturnError(errors.New("balance unavailable"))
	mock.ExpectRollback()
	_, err = repo.CheckinCalendar(context.Background(), 42, time.Date(2026, 11, 1, 0, 0, 0, 0, service.RewardLocation), true)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckinRequiresActiveSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := &affiliateRepository{client: dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM users").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))
	mock.ExpectQuery("SELECT us.id FROM user_subscriptions").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectRollback()
	_, err = repo.CheckinCalendar(context.Background(), 42, time.Now(), true)
	require.ErrorIs(t, err, service.ErrCheckinSubscriptionRequired)
	require.NoError(t, mock.ExpectationsWereMet())
}
