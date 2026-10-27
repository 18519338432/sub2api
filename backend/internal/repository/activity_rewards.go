package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *affiliateRepository) CheckinCalendar(ctx context.Context, userID int64, now time.Time, claim bool) (*service.CheckinCalendar, error) {
	now = now.In(service.RewardLocation)
	month, today := now.Format("2006-01"), now.Format("2006-01-02")
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, service.RewardLocation)
	count := first.AddDate(0, 1, -1).Day()
	out := &service.CheckinCalendar{Month: month, Today: today, Days: make([]service.CheckinDay, count)}
	err := r.withTx(ctx, func(txCtx context.Context, client *dbent.Client) error {
		rows, err := client.QueryContext(txCtx, "SELECT id FROM users WHERE id=$1 AND deleted_at IS NULL FOR UPDATE", userID)
		if err != nil {
			return err
		}
		exists := rows.Next()
		if err = rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		if !exists {
			return service.ErrUserNotFound
		}
		// Check entitlement inside the claim transaction and hold a shared lock
		// against cancellation or group deactivation until the credit commits.
		rows, err = client.QueryContext(txCtx, `SELECT us.id FROM user_subscriptions us
			JOIN groups g ON g.id=us.group_id
			WHERE us.user_id=$1 AND us.status='active' AND us.deleted_at IS NULL
			AND us.starts_at <= $2 AND us.expires_at > $2
			AND g.deleted_at IS NULL AND g.status='active' AND g.subscription_type='subscription'
			LIMIT 1 FOR SHARE OF us,g`, userID, now)
		if err != nil {
			return err
		}
		out.Eligible = rows.Next()
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if claim && !out.Eligible {
			return service.ErrCheckinSubscriptionRequired
		}
		rows, err = client.QueryContext(txCtx, "SELECT daily_cents FROM user_checkin_months WHERE user_id=$1 AND month=$2", userID, month)
		if err != nil {
			return err
		}
		var raw []byte
		if rows.Next() {
			err = rows.Scan(&raw)
		}
		if err == nil {
			err = rows.Err()
		}
		rows.Close()
		if err != nil {
			return err
		}
		var amounts []int
		if raw == nil {
			amounts, err = service.GenerateCheckinAmounts(count)
			if err != nil {
				return err
			}
			raw, err = json.Marshal(amounts)
			if err != nil {
				return err
			}
			if _, err = client.ExecContext(txCtx, "INSERT INTO user_checkin_months(user_id,month,daily_cents) VALUES($1,$2,$3::jsonb)", userID, month, string(raw)); err != nil {
				return err
			}
		} else if err = json.Unmarshal(raw, &amounts); err != nil {
			return err
		}
		if len(amounts) != count {
			return fmt.Errorf("invalid checkin month allocation")
		}
		for i := range out.Days {
			out.Days[i] = service.CheckinDay{Date: first.AddDate(0, 0, i).Format("2006-01-02"), BaseCents: amounts[i]}
		}
		rows, err = client.QueryContext(txCtx, "SELECT reward_date::text, amount_cents, streak FROM user_activity_rewards WHERE user_id=$1 AND kind='checkin' AND reward_date >= $2::date AND reward_date < $3::date ORDER BY reward_date", userID, first.Format("2006-01-02"), first.AddDate(0, 1, 0).Format("2006-01-02"))
		if err != nil {
			return err
		}
		streaks := make(map[string]int)
		for rows.Next() {
			var date string
			var amount, streak int
			if err = rows.Scan(&date, &amount, &streak); err != nil {
				rows.Close()
				return err
			}
			d, e := time.Parse("2006-01-02", date)
			if e != nil {
				rows.Close()
				return e
			}
			day := &out.Days[d.Day()-1]
			day.Checked = true
			day.BonusCents = amount - day.BaseCents
			out.TotalCents += amount
			streaks[date] = streak
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		out.Streak = streaks[now.AddDate(0, 0, -1).Format("2006-01-02")]
		day := &out.Days[now.Day()-1]
		if day.Checked {
			out.Streak = streaks[today]
		}
		if claim && !day.Checked {
			out.Streak++
			if out.Streak%7 == 0 {
				day.BonusCents = 200
			}
			amount := day.BaseCents + day.BonusCents
			if _, err = client.ExecContext(txCtx, "INSERT INTO user_activity_rewards(user_id,kind,source_key,amount_cents,reward_date,streak) VALUES($1,'checkin',$2,$3,$4::date,$5)", userID, fmt.Sprintf("%d:%s", userID, today), amount, today, out.Streak); err != nil {
				return err
			}
			if _, err = client.ExecContext(txCtx, "UPDATE users SET balance=balance+$1::numeric/100,updated_at=NOW() WHERE id=$2", amount, userID); err != nil {
				return err
			}
			day.Checked = true
			out.TotalCents += amount
		}
		return nil
	})
	return out, err
}

func (r *affiliateRepository) ListRewardCards(ctx context.Context, userID int64) ([]service.AffiliateRewardCard, error) {
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `SELECT rc.code,COALESCE(g.name,''),rc.status,rc.validity_days FROM user_activity_rewards ar JOIN redeem_codes rc ON rc.id=ar.redeem_code_id LEFT JOIN groups g ON g.id=rc.group_id WHERE ar.user_id=$1 AND ar.kind='subscription_card' ORDER BY ar.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []service.AffiliateRewardCard{}
	for rows.Next() {
		var card service.AffiliateRewardCard
		if err = rows.Scan(&card.Code, &card.GroupName, &card.Status, &card.ValidityDays); err != nil {
			return nil, err
		}
		result = append(result, card)
	}
	return result, rows.Err()
}
