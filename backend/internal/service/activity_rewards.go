package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Rewards use a fixed UTC+8 business day, independent of server timezone.
var RewardLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type CheckinDay struct {
	Date       string `json:"date"`
	BaseCents  int    `json:"base_cents"`
	BonusCents int    `json:"bonus_cents"`
	Checked    bool   `json:"checked"`
}
type CheckinCalendar struct {
	Eligible   bool         `json:"eligible"`
	Month      string       `json:"month"`
	Today      string       `json:"today"`
	Streak     int          `json:"streak"`
	TotalCents int          `json:"total_cents"`
	Days       []CheckinDay `json:"days"`
}

var ErrCheckinSubscriptionRequired = infraerrors.Forbidden("CHECKIN_SUBSCRIPTION_REQUIRED", "仅当前持有有效订阅的用户可以签到")

type AffiliateRewardCard struct {
	Code         string `json:"code"`
	GroupName    string `json:"group_name"`
	Status       string `json:"status"`
	ValidityDays int    `json:"validity_days"`
}
type ActivityRewardRepository interface {
	CheckinCalendar(context.Context, int64, time.Time, bool) (*CheckinCalendar, error)
	ListRewardCards(context.Context, int64) ([]AffiliateRewardCard, error)
}

// GenerateCheckinAmounts splits exactly $25 into near-average random daily cents.
// The allocation is persisted once per user/month, so retries cannot reroll it.
func GenerateCheckinAmounts(days int) ([]int, error) {
	if days < 28 || days > 31 {
		return nil, fmt.Errorf("invalid calendar month length: %d", days)
	}
	amounts := make([]int, days)
	for i := range amounts {
		amounts[i] = 2500 / days
		if i < 2500%days {
			amounts[i]++
		}
	}
	for i := 0; i < days*4; i++ {
		a, err := rand.Int(rand.Reader, big.NewInt(int64(days)))
		if err != nil {
			return nil, err
		}
		b, err := rand.Int(rand.Reader, big.NewInt(int64(days)))
		if err != nil {
			return nil, err
		}
		if amounts[a.Int64()] > 2500/days-15 && amounts[b.Int64()] < 2500/days+15 {
			amounts[a.Int64()]--
			amounts[b.Int64()]++
		}
	}
	return amounts, nil
}
func (s *AffiliateService) ActivityCalendar(ctx context.Context, userID int64, claim bool) (*CheckinCalendar, error) {
	repo, ok := s.repo.(ActivityRewardRepository)
	if !ok {
		return nil, infraerrors.ServiceUnavailable("REWARDS_UNAVAILABLE", "activity rewards unavailable")
	}
	result, err := repo.CheckinCalendar(ctx, userID, time.Now().In(RewardLocation), claim)
	if err == nil && claim {
		s.invalidateAffiliateCaches(ctx, userID)
	}
	return result, err
}
func (s *AffiliateService) RewardCards(ctx context.Context, userID int64) ([]AffiliateRewardCard, error) {
	repo, ok := s.repo.(ActivityRewardRepository)
	if !ok {
		return []AffiliateRewardCard{}, nil
	}
	return repo.ListRewardCards(ctx, userID)
}
