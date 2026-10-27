package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Reward ownership lives in the immutable activity ledger, not editable code notes.
func validateRewardCardOwner(ctx context.Context, client *dbent.Client, codeID, userID int64) (bool, error) {
	rows, err := client.QueryContext(ctx, "SELECT user_id FROM user_activity_rewards WHERE redeem_code_id=$1 AND kind='subscription_card'", codeID)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return false, rows.Err()
	}
	var owner int64
	if err = rows.Scan(&owner); err != nil {
		return false, err
	}
	if owner != userID {
		return true, infraerrors.BadRequest("REWARD_CARD_OWNER_REQUIRED", "此奖励卡仅限邀请者本人兑换")
	}
	return true, nil
}

func (s *RedeemService) grantSubscriptionRewardCard(ctx context.Context, client *dbent.Client, userID int64, source *RedeemCode) error {
	if s.affiliateService == nil || !s.affiliateService.IsEnabled(ctx) {
		return nil
	}
	days := source.ValidityDays
	if days == 0 {
		days = 30
	}
	if days != 30 || source.GroupID == nil {
		return nil
	}
	rows, err := client.QueryContext(ctx, "SELECT inviter_id FROM user_affiliates WHERE user_id=$1 AND inviter_id IS NOT NULL", userID)
	if err != nil {
		return err
	}
	var inviterID int64
	if !rows.Next() {
		err = rows.Err()
		rows.Close()
		return err
	}
	err = rows.Scan(&inviterID)
	rows.Close()
	if err != nil {
		return err
	}
	if inviterID == userID {
		return nil
	}
	key := fmt.Sprintf("redeem:%d", source.ID)
	result, err := client.ExecContext(ctx, "INSERT INTO user_activity_rewards(user_id,kind,source_key) VALUES($1,'subscription_card',$2) ON CONFLICT(kind,source_key) DO NOTHING", inviterID, key)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	var random [16]byte
	if _, err = rand.Read(random[:]); err != nil {
		return err
	}
	card, err := client.RedeemCode.Create().SetCode(hex.EncodeToString(random[:])).SetType(RedeemTypeSubscription).SetStatus("unused").SetValue(0).SetGroupID(*source.GroupID).SetValidityDays(3).SetNotes("邀请月订阅奖励，仅限获奖者本人兑换").Save(ctx)
	if err != nil {
		return err
	}
	_, err = client.ExecContext(ctx, "UPDATE user_activity_rewards SET redeem_code_id=$1 WHERE kind='subscription_card' AND source_key=$2", card.ID, key)
	return err
}
