package service

import (
	"math"
	"testing"
)

func TestValidateTimedBalance(t *testing.T) {
	for _, tc := range []struct {
		amount float64
		days   int
		valid  bool
	}{
		{10, 30, true}, {0.01, 1, true}, {10, 3650, true},
		{0, 30, false}, {-1, 30, false}, {1, 0, false}, {1, 3651, false},
		{math.NaN(), 1, false}, {math.Inf(1), 1, false},
	} {
		if got := validateTimedBalance(tc.amount, tc.days); (got == nil) != tc.valid {
			t.Errorf("validateTimedBalance(%v, %d) = %v", tc.amount, tc.days, got)
		}
	}
}
