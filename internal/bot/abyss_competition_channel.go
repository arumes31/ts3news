package bot

import (
	"fmt"
	"strings"
	"time"
)

func (b *Bot) abyssCompetitionChannelEmbed(channelID int, at time.Time) string {
	_, start, end := abyssCompetitionWeekAt(at)
	rows, err := b.DB.Query(`WITH best AS (
		SELECT DISTINCT ON (r.client_uid) r.client_uid,r.depth,r.gold_banked
		FROM abyss_runs r JOIN abyss_competition_presence p ON p.client_uid=r.client_uid
		WHERE p.channel_id=$1 AND p.seen_at>NOW()-INTERVAL '30 minutes'
		 AND r.created_at >= $2 AND r.created_at < $3
		ORDER BY r.client_uid,r.depth DESC,r.gold_banked DESC
	)
	SELECT COALESCE(NULLIF(u.nickname,''),'Adventurer'),b.depth FROM best b
	LEFT JOIN users u ON u.client_uid=b.client_uid
	ORDER BY b.depth DESC,b.gold_banked DESC,b.client_uid LIMIT 3`, channelID, start, end)
	if err != nil {
		return ""
	}
	defer func() { _ = rows.Close() }()
	entries := make([]string, 0, 3)
	for rows.Next() {
		var nick string
		var depth int
		if rows.Scan(&nick, &depth) != nil {
			return ""
		}
		entries = append(entries, fmt.Sprintf("#%d %s F%d", len(entries)+1, sanitizeBBCode(nick), depth))
	}
	if len(entries) == 0 {
		return ""
	}
	return "[hr]\n[b]🏆 Channel Abyss · this week[/b]  " + strings.Join(entries, " · ") + "\n"
}
