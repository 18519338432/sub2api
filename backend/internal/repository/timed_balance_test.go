package repository

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestTimedSpendableBalanceReadsLiveGrants(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	repo := &userRepository{client: client}
	mock.ExpectQuery("SELECT balance").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(12.5))
	balance, err := repo.GetSpendableBalance(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, 12.5, balance)
	mock.ExpectQuery("SELECT balance").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(2.5))
	balance, err = repo.GetSpendableBalance(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, 2.5, balance)
	mock.ExpectQuery("SELECT balance").WithArgs(int64(99)).WillReturnRows(sqlmock.NewRows([]string{"balance"}))
	_, err = repo.GetSpendableBalance(context.Background(), 99)
	require.ErrorIs(t, err, service.ErrUserNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTimedBalanceDeductsOnlyUncoveredAmountFromOrdinaryBalance(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	mock.ExpectQuery("SELECT deduct_timed_balance").WithArgs(int64(7), 12.5).
		WillReturnRows(sqlmock.NewRows([]string{"ordinary_cost"}).AddRow(2.5))
	mock.ExpectQuery("UPDATE users").WithArgs(2.5, int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(7.5))
	balance, sufficient, err := deductUsageBillingBalance(context.Background(), tx, 7, 12.5)
	require.NoError(t, err)
	require.True(t, sufficient)
	require.Equal(t, 7.5, balance)
	mock.ExpectCommit()
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}
