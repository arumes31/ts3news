package bot

import (
	"log"

	"ts3news/internal/content"
)

func (b *Bot) abyssEscrowHighWater(uid string) int64 {
	var id int64
	_ = b.DB.QueryRow("SELECT COALESCE(MAX(id),0) FROM abyss_escrow_loot WHERE client_uid=$1", uid).Scan(&id)
	return id
}

func (b *Bot) routeAbyssPartyLoot(ownerUID string, partyUIDs []string, starts map[string]int64, rule string, bossFloor bool) {
	if len(partyUIDs) == 0 {
		return
	}
	recipient := ownerUID
	if rule == "need_before_greed" && bossFloor {
		lowestCR := b.abyssPlayerCR(ownerUID)
		for _, uid := range partyUIDs {
			if cr := b.abyssPlayerCR(uid); cr < lowestCR {
				lowestCR = cr
				recipient = uid
			}
		}
		_, _ = b.DB.Exec(`UPDATE abyss_escrow_loot SET party_recipient_uid=$1
			WHERE client_uid=$2 AND id>$3`, recipient, ownerUID, starts[ownerUID])
	}
	for _, uid := range partyUIDs {
		recipient = ownerUID
		if rule == "round_robin" {
			recipient = uid
		} else if rule == "need_before_greed" && bossFloor {
			lowestCR := b.abyssPlayerCR(ownerUID)
			for _, candidate := range partyUIDs {
				if cr := b.abyssPlayerCR(candidate); cr < lowestCR {
					lowestCR = cr
					recipient = candidate
				}
			}
		}
		_, _ = b.DB.Exec(`UPDATE abyss_escrow_loot SET client_uid=$1,party_recipient_uid=$2
			WHERE client_uid=$3 AND id>$4`, ownerUID, recipient, uid, starts[uid])
	}
}

func (b *Bot) abyssPartyMembers(ownerUID string) []string {
	rows, err := b.DB.Query(`SELECT member_uid FROM abyss_party_members
		WHERE owner_uid=$1 ORDER BY joined_at LIMIT $2`, ownerUID, abyssPartyMaxMembers-1)
	if err == nil {
		defer func() { _ = rows.Close() }()
		members := make([]string, 0, abyssPartyMaxMembers-1)
		for rows.Next() {
			var uid string
			if rows.Scan(&uid) == nil && uid != "" && uid != ownerUID {
				members = append(members, uid)
			}
		}
		if len(members) > 0 {
			return members
		}
	}
	var legacy string
	_ = b.DB.QueryRow("SELECT COALESCE(coop_uid,'') FROM abyss_active WHERE client_uid=$1", ownerUID).Scan(&legacy)
	if legacy != "" && legacy != ownerUID {
		return []string{legacy}
	}
	return nil
}

func scaleAbyssCheerStats(stats content.Stats) content.Stats {
	stats.HP += stats.HP * abyssFriendCheerBonusPct / 100
	stats.STR += stats.STR * abyssFriendCheerBonusPct / 100
	stats.DEF += stats.DEF * abyssFriendCheerBonusPct / 100
	stats.SPD += stats.SPD * abyssFriendCheerBonusPct / 100
	stats.INT += stats.INT * abyssFriendCheerBonusPct / 100
	stats.MNA += stats.MNA * abyssFriendCheerBonusPct / 100
	return stats
}

func (b *Bot) applyAbyssCheer(user *UserInCombat) bool {
	var active bool
	_ = b.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM abyss_friend_cheers
		WHERE recipient_uid=$1 AND cheer_date=CURRENT_DATE AND remaining_fights>0)`, user.UID).Scan(&active)
	if !active {
		return false
	}
	oldHP := max(1, user.Stats.HP)
	user.Stats = scaleAbyssCheerStats(user.Stats)
	user.CurrentHP = min(user.Stats.HP, max(1, user.CurrentHP*user.Stats.HP/oldHP))
	return true
}

func (b *Bot) consumeAbyssCheer(uid string) {
	_, _ = b.DB.Exec(`UPDATE abyss_friend_cheers SET remaining_fights=remaining_fights-1
		WHERE recipient_uid=$1 AND cheer_date=CURRENT_DATE AND remaining_fights>0`, uid)
}

func (b *Bot) settleAbyssSocialFloor(uid string, depth int) {
	week := abyssCurrentWeek(timeNowUTC())
	_, _ = b.DB.Exec(`INSERT INTO abyss_guild_weekly_progress (guild_id,week_key,floors,goal)
		SELECT guild_id,$2,1,$3 FROM abyss_guild_members WHERE client_uid=$1
		ON CONFLICT (guild_id,week_key) DO UPDATE SET floors=abyss_guild_weekly_progress.floors+1`,
		uid, week, abyssGuildWeeklyGoal)
	referralTx, referralErr := b.DB.Begin()
	if referralErr == nil {
		var referrer string
		if referralTx.QueryRow(`UPDATE abyss_referrals r SET rewarded_at=NOW() FROM users u
			WHERE r.referred_uid=$1 AND r.referred_uid=u.client_uid AND r.rewarded_at IS NULL
			AND GREATEST(u.abyss_best_depth,$2)>=5 RETURNING r.referrer_uid`, uid, depth).Scan(&referrer) == nil {
			if _, err := referralTx.Exec(`UPDATE users SET abyss_tokens=abyss_tokens+$1
				WHERE client_uid IN ($2,$3)`, abyssReferralRewardTokens, referrer, uid); err == nil {
				if _, err = referralTx.Exec(`INSERT INTO abyss_social_notifications (client_uid,kind,message)
					VALUES ($1,'referral_reward','Referral milestone reached: both delvers received 20 Abyss Tokens.')`, referrer); err == nil {
					_ = referralTx.Commit()
				}
			}
		}
		_ = referralTx.Rollback()
	}
	tx, err := b.DB.Begin()
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback() }()
	var owner string
	var lostCache int64
	var deathID int64
	if tx.QueryRow(`UPDATE abyss_rescue_missions SET completed_at=NOW()
		WHERE death_id=(SELECT death_id FROM abyss_rescue_missions WHERE rescuer_uid=$1
		AND completed_at IS NULL AND depth<=$2 ORDER BY accepted_at LIMIT 1 FOR UPDATE)
		RETURNING death_id,owner_uid,lost_cache`, uid, depth).Scan(&deathID, &owner, &lostCache) != nil {
		return
	}
	reward := max(int64(1), lostCache/10)
	if _, err := tx.Exec(`UPDATE users SET gold=gold+$1 WHERE client_uid IN ($2,$3)`, reward, uid, owner); err != nil {
		return
	}
	if _, err := tx.Exec("UPDATE abyss_deaths SET rescued_at=NOW() WHERE id=$1", deathID); err != nil {
		return
	}
	if _, err := tx.Exec(`INSERT INTO abyss_social_notifications (client_uid,kind,message)
		VALUES ($1,'rescue_complete',$2)`, owner, "A friend completed your rescue mission; both delvers recovered 10% of the lost cache."); err != nil {
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("abyss rescue settlement commit failed for %s: %v", uid, err)
	}
}

func (b *Bot) abyssSocialHappyHour() bool {
	var floors int64
	_ = b.DB.QueryRow(`SELECT COALESCE(SUM(floors_cleared),0) FROM abyss_runs
		WHERE created_at>=CURRENT_DATE AND created_at<CURRENT_DATE+INTERVAL '1 day'`).Scan(&floors)
	return floors >= abyssDailyServerGoal
}
