//go:build unit

package service

import (
	"context"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionRewardCardOwnershipAndDeduplication(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)
	_, err := client.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS user_affiliates (user_id INTEGER PRIMARY KEY, inviter_id INTEGER)")
	require.NoError(t, err)
	inviter, err := client.User.Create().SetEmail("blaire-inviter@example.com").SetPasswordHash("hash").Save(ctx)
	require.NoError(t, err)
	invitee, err := client.User.Create().SetEmail("blaire-invitee@example.com").SetPasswordHash("hash").Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().SetName("blaire-subscription").Save(ctx)
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, "INSERT INTO user_affiliates(user_id,inviter_id) VALUES($1,$2)", invitee.ID, inviter.ID)
	require.NoError(t, err)
	settings := NewSettingService(&paymentFulfillmentSettingRepoStub{values: map[string]string{SettingKeyAffiliateEnabled: "true"}}, nil)
	svc := &RedeemService{affiliateService: NewAffiliateService(nil, settings, nil, nil)}
	source := &RedeemCode{ID: 123, GroupID: &group.ID, ValidityDays: 30}
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer tx.Rollback()
	txCtx := dbent.NewTxContext(ctx, tx)
	require.NoError(t, svc.grantSubscriptionRewardCard(txCtx, tx.Client(), invitee.ID, source))
	require.NoError(t, svc.grantSubscriptionRewardCard(txCtx, tx.Client(), invitee.ID, source))
	cards, err := tx.Client().RedeemCode.Query().All(txCtx)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.Equal(t, 3, cards[0].ValidityDays)
	require.Equal(t, group.ID, *cards[0].GroupID)
	reward, err := validateRewardCardOwner(txCtx, tx.Client(), cards[0].ID, inviter.ID)
	require.NoError(t, err)
	require.True(t, reward)
	reward, err = validateRewardCardOwner(txCtx, tx.Client(), cards[0].ID, invitee.ID)
	require.Error(t, err)
	require.True(t, reward)
	// Three-day reward cards cannot trigger the thirty-day reward rule.
	require.NoError(t, svc.grantSubscriptionRewardCard(txCtx, tx.Client(), inviter.ID, &RedeemCode{ID: cards[0].ID, GroupID: &group.ID, ValidityDays: 3}))
	count, err := tx.Client().RedeemCode.Query().Count(txCtx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.NoError(t, tx.Rollback())
	count, err = client.RedeemCode.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count, "reward card must roll back with the original redemption")
}
