package service

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGenerateCheckinAmountsMonthlyBudget(t *testing.T) {
	for _, days := range []int{28, 29, 30, 31} {
		for sample := 0; sample < 100; sample++ {
			amounts, err := GenerateCheckinAmounts(days)
			require.NoError(t, err)
			require.Len(t, amounts, days)
			sum := 0
			for _, amount := range amounts {
				sum += amount
				require.GreaterOrEqual(t, amount, 2500/days-15)
				require.LessOrEqual(t, amount, 2500/days+15)
			}
			require.Equal(t, 2500, sum)
		}
	}
	for _, days := range []int{-1, 0, 27, 32} {
		_, err := GenerateCheckinAmounts(days)
		require.Error(t, err)
	}
}
func TestRewardBusinessDateCrossesUTCMonth(t *testing.T) {
	instant := time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC)
	require.Equal(t, "2026-11-01", instant.In(RewardLocation).Format("2006-01-02"))
}
