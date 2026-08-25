package bot

import (
	"fmt"
	"time"
)

type abyssBestKillView struct {
	Available  bool
	Boss       string
	Depth      int
	KillTimeMS int64
	KillTime   string
	Tier       string
	TierName   string
	KilledAt   string
}

func (b *Bot) abyssBestKill(uid string) abyssBestKillView {
	var view abyssBestKillView
	var killedAt time.Time
	err := b.DB.QueryRow(`SELECT boss_name,depth,kill_time_ms,tier,killed_at
		FROM abyss_boss_kills WHERE client_uid=$1
		ORDER BY depth DESC,kill_time_ms ASC,killed_at DESC LIMIT 1`, uid).
		Scan(&view.Boss, &view.Depth, &view.KillTimeMS, &view.Tier, &killedAt)
	if err != nil {
		return abyssBestKillView{}
	}
	view.Available = true
	view.KillTime = fmt.Sprintf("%.1fs", float64(view.KillTimeMS)/1000)
	view.KilledAt = killedAt.UTC().Format("2006-01-02")
	view.TierName = view.Tier
	if tier, ok := abyssTierByKey(view.Tier); ok {
		view.TierName = tier.Name
	}
	return view
}
