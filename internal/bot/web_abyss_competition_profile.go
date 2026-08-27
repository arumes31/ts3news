package bot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

func (b *Bot) abyssCompetitionArchives() []abyssCompetitionArchive {
	rows, err := b.DB.Query(`SELECT period_key,tier,rank,nickname,depth,reward_kind
		FROM abyss_competition_snapshots WHERE period_kind='season' AND rank<=10
		ORDER BY created_at DESC,tier,rank LIMIT 120`)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	archives := make([]abyssCompetitionArchive, 0, 120)
	for rows.Next() {
		var item abyssCompetitionArchive
		if rows.Scan(&item.Season, &item.Tier, &item.Rank, &item.Nick, &item.Depth, &item.Reward) != nil {
			return nil
		}
		archives = append(archives, item)
	}
	return archives
}

func (b *Bot) abyssCompetitionPersonalDelta(uid string, currentDepth int, at time.Time) (int, int) {
	_, currentStart, _ := abyssCompetitionWeekAt(at)
	previousStart := currentStart.AddDate(0, 0, -7)
	var depth int
	_ = b.DB.QueryRow(`SELECT COALESCE(MAX(depth),0) FROM abyss_runs
		WHERE client_uid=$1 AND created_at >= $2 AND created_at < $3`, uid, previousStart, currentStart).Scan(&depth)
	return depth, currentDepth - depth
}

func (b *Bot) abyssCompetitionDepthHistory(uid string) []abyssCompetitionDepthPoint {
	rows, err := b.DB.Query(`SELECT created_at::date,MAX(depth) FROM abyss_runs
		WHERE client_uid=$1 AND created_at>=CURRENT_DATE-INTERVAL '29 days'
		GROUP BY created_at::date ORDER BY created_at::date`, uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	points := make([]abyssCompetitionDepthPoint, 0, 30)
	maxDepth := 1
	for rows.Next() {
		var at time.Time
		var point abyssCompetitionDepthPoint
		if rows.Scan(&at, &point.Depth) != nil {
			return nil
		}
		point.Date = at.Format("Jan 2")
		maxDepth = max(maxDepth, point.Depth)
		points = append(points, point)
	}
	for i := range points {
		points[i].Pct = max(4, points[i].Depth*100/maxDepth)
	}
	return points
}

func (b *Bot) abyssCompetitionAudits(uid string) []abyssCompetitionAuditView {
	rows, err := b.DB.Query(`SELECT depth,created_at,audit_hash,audit_data
		FROM abyss_runs WHERE client_uid=$1 AND audit_hash<>'' ORDER BY id DESC LIMIT 10`, uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	items := make([]abyssCompetitionAuditView, 0, 10)
	for rows.Next() {
		var item abyssCompetitionAuditView
		var at time.Time
		var hash, raw string
		if rows.Scan(&item.Depth, &at, &hash, &raw) != nil {
			return nil
		}
		var audit abyssCompetitionAudit
		if json.Unmarshal([]byte(raw), &audit) == nil {
			if canonical, err := json.Marshal(audit); err == nil {
				digest := sha256.Sum256(canonical)
				item.Valid = hex.EncodeToString(digest[:]) == hash
			}
		}
		item.When = at.Format("Jan 2 15:04")
		item.Hash = hash
		if len(item.Hash) > 12 {
			item.Hash = item.Hash[:12]
		}
		items = append(items, item)
	}
	return items
}

func (b *Bot) abyssCompetitionTrophies(uid string) []string {
	rows, err := b.DB.Query(`SELECT period_key,tier,rank FROM abyss_competition_rewards
		WHERE client_uid=$1 AND period_kind='season' AND cosmetic_key<>''
		ORDER BY awarded_at DESC LIMIT 20`, uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var trophies []string
	for rows.Next() {
		var key, tier string
		var rank int
		if rows.Scan(&key, &tier, &rank) != nil {
			return nil
		}
		trophies = append(trophies, fmt.Sprintf("%s · %s #%d", key, tier, rank))
	}
	return trophies
}

func (b *Bot) abyssCompetitionWagers(uid string, at time.Time) []abyssCompetitionWagerView {
	key, _, _ := abyssCompetitionWeekAt(at)
	views := make([]abyssCompetitionWagerView, 0, 3)
	for _, bracket := range []int{100, 1000, 10000} {
		view := abyssCompetitionWagerView{Bracket: bracket, Fee: int64(bracket)}
		_ = b.DB.QueryRow(`SELECT COUNT(*),COALESCE(SUM(entry_fee),0),
			COALESCE(BOOL_OR(client_uid=$3),FALSE) FROM abyss_wager_entries
			WHERE week_key=$1 AND bracket=$2`, key, bracket, uid).Scan(&view.Entrants, &view.Pool, &view.Joined)
		views = append(views, view)
	}
	return views
}
