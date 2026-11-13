package service

import (
	"context"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAdminTimedBalancesPaginationAndUserFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	svc := &SubscriptionService{entClient: dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))}
	mock.ExpectQuery("SELECT COUNT").WithArgs(int64(7), "active").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))
	now := time.Now()
	mock.ExpectQuery("SELECT id, amount").WithArgs(int64(7), "active", 6, 6).WillReturnRows(sqlmock.NewRows([]string{"id", "amount", "remaining", "created_at", "expires_at", "state", "user_id", "email"}).AddRow(3, 10.0, 2.82, now, now.Add(24*time.Hour), "active", 7, "user@example.test"))
	items, total, err := svc.ListAdminTimedBalances(context.Background(), 7, "active", 2, 6)
	require.NoError(t, err)
	require.EqualValues(t, 7, total)
	require.Len(t, items, 1)
	require.EqualValues(t, 7, items[0].UserID)
	require.Equal(t, "user@example.test", items[0].Email)
	require.InDelta(t, 2.82, items[0].Remaining, 0.001)
	require.NoError(t, mock.ExpectationsWereMet())
}
