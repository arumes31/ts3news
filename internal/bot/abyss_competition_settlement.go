package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

func (b *Bot) settleAbyssCompetition(at time.Time) error {
	_, currentWeekStart, _ := abyssCompetitionWeekAt(at)
	weekKey, weekStart, weekEnd := abyssCompetitionWeekAt(currentWeekStart.Add(-time.Second))
	if err := b.settleAbyssCompetitionPeriod("weekly", weekKey, weekStart, weekEnd); err != nil {
		return err
	}
	if err := b.settleAbyssWagerWeek(weekKey, weekStart, weekEnd); err != nil {
		return err
	}
	_, currentSeasonStart, _ := abyssCompetitionSeasonAt(at)
	seasonKey, seasonStart, seasonEnd := abyssCompetitionSeasonAt(currentSeasonStart.Add(-time.Second))
	return b.settleAbyssCompetitionPeriod("season", seasonKey, seasonStart, seasonEnd)
}

func (b *Bot) settleAbyssCompetitionPeriod(kind, key string, start, end time.Time) error {
	tx, err := b.DB.Begin()
	if err != nil {
		return fmt.Errorf("begin abyss competition settlement: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var inserted string
	err = tx.QueryRow(`INSERT INTO abyss_competition_periods (period_key,period_kind,starts_at,ends_at)
		VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING RETURNING period_key`, key, kind, start, end).Scan(&inserted)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("claim abyss competition period: %w", err)
	}
	if _, err := tx.Exec(`WITH best AS (
		SELECT DISTINCT ON (tier,client_uid) tier,client_uid,depth,gold_banked,build_key
		FROM abyss_runs WHERE created_at >= $3 AND created_at < $4
		ORDER BY tier,client_uid,depth DESC,gold_banked DESC,duration_ms ASC,id ASC
	), ranked AS (
		SELECT b.*,RANK() OVER (PARTITION BY tier ORDER BY depth DESC,gold_banked DESC,client_uid) AS rank
		FROM best b
	)
	INSERT INTO abyss_competition_snapshots
		(period_kind,period_key,tier,rank,client_uid,nickname,depth,gold_banked,build_key,reward_kind)
	SELECT $1,$2,r.tier,r.rank,r.client_uid,COALESCE(NULLIF(u.nickname,''),'Adventurer'),
		r.depth,r.gold_banked,r.build_key,
		CASE WHEN $1='season' AND r.rank<=10 THEN 'season_champion'
		     WHEN $1='season' THEN 'season_participant'
		     WHEN r.rank<=10 THEN 'weekly_ranked' ELSE '' END
	FROM ranked r LEFT JOIN users u ON u.client_uid=r.client_uid WHERE r.rank<=100`, kind, key, start, end); err != nil {
		return fmt.Errorf("snapshot abyss competition period: %w", err)
	}
	if _, err := tx.Exec(`WITH best AS (
		SELECT DISTINCT ON (tier,client_uid) tier,client_uid,depth,gold_banked
		FROM abyss_runs WHERE created_at >= $3 AND created_at < $4
		ORDER BY tier,client_uid,depth DESC,gold_banked DESC,duration_ms ASC,id ASC
	), ranked AS (
		SELECT b.*,RANK() OVER (PARTITION BY tier ORDER BY depth DESC,gold_banked DESC,client_uid) AS rank
		FROM best b
	)
	INSERT INTO abyss_competition_rewards
		(period_kind,period_key,tier,client_uid,rank,reward_kind,tokens,cosmetic_key)
	SELECT $1,$2,tier,client_uid,rank,
		CASE WHEN $1='season' AND rank<=10 THEN 'season_champion'
		     WHEN $1='season' AND rank<=100 THEN 'season_ranked'
		     WHEN $1='season' THEN 'season_participant'
		     ELSE 'weekly_ranked' END,
		CASE WHEN $1='season' AND rank<=100 THEN GREATEST(1,101-rank)
		     WHEN $1='weekly' AND rank<=10 THEN 11-rank ELSE 0 END,
		CASE WHEN $1='season' AND rank<=10 THEN 'season_trophy_'||$2||'_'||tier ELSE '' END
	FROM ranked WHERE $1='season' OR rank<=10`, kind, key, start, end); err != nil {
		return fmt.Errorf("record abyss competition rewards: %w", err)
	}
	if _, err := tx.Exec(`UPDATE users u SET abyss_tokens=u.abyss_tokens+r.tokens
		FROM (SELECT client_uid,SUM(tokens)::BIGINT AS tokens FROM abyss_competition_rewards
		      WHERE period_kind=$1 AND period_key=$2 GROUP BY client_uid) r
		WHERE u.client_uid=r.client_uid AND r.tokens>0`, kind, key); err != nil {
		return fmt.Errorf("award abyss competition tokens: %w", err)
	}
	if kind == "season" {
		if _, err := tx.Exec(`INSERT INTO abyss_shop_cosmetics (client_uid,cosmetic_key)
			SELECT client_uid,'season_badge_'||$1 FROM abyss_competition_rewards
			WHERE period_kind='season' AND period_key=$1 ON CONFLICT DO NOTHING`, key); err != nil {
			return fmt.Errorf("award abyss season participant badges: %w", err)
		}
		if _, err := tx.Exec(`INSERT INTO abyss_shop_cosmetics (client_uid,cosmetic_key)
			SELECT client_uid,cosmetic_key FROM abyss_competition_rewards
			WHERE period_kind='season' AND period_key=$1 AND cosmetic_key<>'' ON CONFLICT DO NOTHING`, key); err != nil {
			return fmt.Errorf("award abyss season trophies: %w", err)
		}
	}
	if err := b.recordAbyssRankChanges(tx, kind, key); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit abyss competition settlement: %w", err)
	}
	return nil
}

func (b *Bot) recordAbyssRankChanges(tx *sql.Tx, kind, key string) error {
	if _, err := tx.Exec(`WITH previous_period AS (
		SELECT period_key FROM abyss_competition_periods
		WHERE period_kind=$1 AND period_key<>$2 ORDER BY ends_at DESC LIMIT 1
	), changes AS (
		SELECT now.client_uid,now.tier,now.rank,old.rank AS previous_rank
		FROM abyss_competition_snapshots now
		JOIN previous_period p ON TRUE
		JOIN abyss_competition_snapshots old ON old.period_kind=now.period_kind
		 AND old.period_key=p.period_key AND old.tier=now.tier AND old.client_uid=now.client_uid
		WHERE now.period_kind=$1 AND now.period_key=$2 AND now.rank>old.rank
	)
	INSERT INTO abyss_rank_notifications (client_uid,board_key,period_key,rank,previous_rank)
	SELECT client_uid,$1||'_depth_'||tier,$2,rank,previous_rank FROM changes ON CONFLICT DO NOTHING`, kind, key); err != nil {
		return fmt.Errorf("record abyss rank changes: %w", err)
	}
	_, err := tx.Exec(`INSERT INTO abyss_social_notifications (client_uid,kind,message)
		SELECT n.client_uid,'rank_change','You were passed by '
		 ||COALESCE(NULLIF(u.nickname,''),'another delver')||' on the '||n.board_key||' board: #'
		 ||n.previous_rank||' → #'||n.rank||'.'
		FROM abyss_rank_notifications n
		LEFT JOIN abyss_competition_snapshots s ON s.period_kind=$2 AND s.period_key=$1
		 AND s.tier=split_part(n.board_key,'_',3) AND s.rank=n.previous_rank
		LEFT JOIN users u ON u.client_uid=s.client_uid
		WHERE n.period_key=$1 AND n.board_key LIKE $2||'%'`, key, kind)
	if err != nil {
		return fmt.Errorf("notify abyss rank changes: %w", err)
	}
	return nil
}

func (b *Bot) settleAbyssWagerWeek(key string, start, end time.Time) error {
	tx, err := b.DB.Begin()
	if err != nil {
		return fmt.Errorf("begin abyss wager settlement: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.Query(`WITH scores AS (
		SELECT e.week_key,e.bracket,e.client_uid,
		 COALESCE(MAX(r.depth),0) AS depth,COALESCE(MIN(r.duration_ms) FILTER (WHERE r.depth>=20),9223372036854775807) AS speed
		FROM abyss_wager_entries e LEFT JOIN abyss_runs r ON r.client_uid=e.client_uid
		 AND r.created_at >= $2 AND r.created_at < $3
		WHERE e.week_key=$1 AND e.settled_at IS NULL GROUP BY e.week_key,e.bracket,e.client_uid
	), ranked AS (
		SELECT s.*,RANK() OVER (PARTITION BY bracket ORDER BY depth DESC,speed ASC,client_uid) AS rank,
		 SUM(e.entry_fee) OVER (PARTITION BY e.bracket) AS pool
		FROM scores s JOIN abyss_wager_entries e USING (week_key,bracket,client_uid)
	), paid AS (
		UPDATE abyss_wager_entries e SET settled_at=NOW(),payout=CASE r.rank
		 WHEN 1 THEN r.pool*50/100 WHEN 2 THEN r.pool*30/100 WHEN 3 THEN r.pool*20/100 ELSE 0 END
		FROM ranked r WHERE e.week_key=r.week_key AND e.bracket=r.bracket AND e.client_uid=r.client_uid
		 AND e.settled_at IS NULL
		RETURNING e.client_uid,e.payout
	) SELECT client_uid,payout FROM paid`, key, start, end)
	if err != nil {
		return fmt.Errorf("settle abyss wager standings: %w", err)
	}
	payouts := make(map[string]int64)
	for rows.Next() {
		var uid string
		var payout int64
		if err := rows.Scan(&uid, &payout); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan abyss wager payout: %w", err)
		}
		payouts[uid] += payout
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close abyss wager payouts: %w", err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate abyss wager payouts: %w", err)
	}
	for uid, payout := range payouts {
		if payout <= 0 {
			continue
		}
		if _, err := tx.Exec("UPDATE users SET gold=gold+$1 WHERE client_uid=$2", payout, uid); err != nil {
			return fmt.Errorf("credit abyss wager payout: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit abyss wager settlement: %w", err)
	}
	return nil
}
