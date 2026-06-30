package bot

import (
	"net/http"
	"time"
)

// Daily Bounties
// -----------------------------------------------------------------------------
// Each calendar day the Abyss offers one server-chosen objective (the same for
// every player, like the daily challenge affix). Progress is derived on demand
// from the rows the run loop already writes — abyss_runs and abyss_boss_kills —
// so no per-action tracking is needed; the bounty is simply a query over "today".
// A one-row-per-day claim guard (abyss_bounty_claims) makes the reward one-shot.

// abyssBountyKind identifies how a bounty's progress is measured.
type abyssBountyKind string

const (
	bountyDepth  abyssBountyKind = "depth"  // reach this depth in a single run today
	bountyBosses abyssBountyKind = "bosses" // defeat this many bosses today
	bountyBank   abyssBountyKind = "bank"   // bank this much gold today
)

// abyssBounty is a fully-resolved daily objective with its reward.
type abyssBounty struct {
	Kind     abyssBountyKind
	Target   int64
	Desc     string
	RewardTk int   // Abyss tokens granted on completion
	RewardGd int64 // gold granted on completion
}

// abyssBountyTable is the rotation of bounty templates. The active one is chosen
// deterministically from the calendar day, so every player shares the day's bounty.
var abyssBountyTable = []abyssBounty{
	{Kind: bountyDepth, Target: 15, Desc: "Reach depth 15 in a single descent today", RewardTk: 25, RewardGd: 5_000},
	{Kind: bountyBosses, Target: 3, Desc: "Defeat 3 Abyss bosses today", RewardTk: 30, RewardGd: 6_000},
	{Kind: bountyBank, Target: 50_000, Desc: "Bank 50,000 gold from the Abyss today", RewardTk: 20, RewardGd: 4_000},
	{Kind: bountyDepth, Target: 25, Desc: "Reach depth 25 in a single descent today", RewardTk: 40, RewardGd: 10_000},
	{Kind: bountyBosses, Target: 5, Desc: "Defeat 5 Abyss bosses today", RewardTk: 45, RewardGd: 9_000},
}

// abyssDailyBounty returns the bounty for the given UTC day, chosen deterministically
// so it matches the daily-challenge cadence and is stable across a calendar day.
func abyssDailyBounty(now time.Time) abyssBounty {
	now = now.UTC()
	seed := now.Year()*1000 + now.YearDay()
	return abyssBountyTable[seed%len(abyssBountyTable)]
}

// abyssBountyProgress computes the player's progress toward today's bounty from the
// run-history tables, scoped to the current UTC day.
func (b *Bot) abyssBountyProgress(uid string, bounty abyssBounty) int64 {
	start := time.Now().UTC().Truncate(24 * time.Hour)
	var p int64
	switch bounty.Kind {
	case bountyDepth:
		_ = b.DB.QueryRow(
			"SELECT COALESCE(MAX(depth), 0) FROM abyss_runs WHERE client_uid=$1 AND created_at >= $2",
			uid, start).Scan(&p)
	case bountyBosses:
		_ = b.DB.QueryRow(
			"SELECT COUNT(*) FROM abyss_boss_kills WHERE client_uid=$1 AND killed_at >= $2",
			uid, start).Scan(&p)
	case bountyBank:
		_ = b.DB.QueryRow(
			"SELECT COALESCE(SUM(gold_banked), 0) FROM abyss_runs WHERE client_uid=$1 AND victory = TRUE AND created_at >= $2",
			uid, start).Scan(&p)
	}
	return p
}

// abyssBountyClaimed reports whether the player has already claimed today's bounty.
func (b *Bot) abyssBountyClaimed(uid string) bool {
	var exists bool
	_ = b.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM abyss_bounty_claims WHERE client_uid=$1 AND bounty_day = (NOW() AT TIME ZONE 'UTC')::date)",
		uid).Scan(&exists)
	return exists
}

// abyssBountyView is the template-facing snapshot of the player's daily bounty.
type abyssBountyView struct {
	Desc     string
	Progress int64
	Target   int64
	RewardTk int
	RewardGd int64
	Met      bool
	Claimed  bool
}

func (b *Bot) abyssBountyStatus(uid string) abyssBountyView {
	bounty := abyssDailyBounty(time.Now())
	prog := b.abyssBountyProgress(uid, bounty)
	return abyssBountyView{
		Desc:     bounty.Desc,
		Progress: prog,
		Target:   bounty.Target,
		RewardTk: bounty.RewardTk,
		RewardGd: bounty.RewardGd,
		Met:      prog >= bounty.Target,
		Claimed:  b.abyssBountyClaimed(uid),
	}
}

// handleAbyssBountyClaim grants the daily bounty reward once, if the objective is
// met. The claim guard (insert into abyss_bounty_claims) and the reward credit run
// in one transaction so a failure can't leave the day marked claimed without paying
// out, nor pay out twice.
func (s *WebServer) handleAbyssBountyClaim(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	bounty := abyssDailyBounty(time.Now())
	if s.bot.abyssBountyProgress(uid, bounty) < bounty.Target {
		writeJSON(w, map[string]any{"ok": false, "error": "bounty not complete yet"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	// The PK on (client_uid, bounty_day) makes this the one-shot guard: a second
	// claim for the same day affects zero rows and is rejected.
	res, err := tx.Exec(
		"INSERT INTO abyss_bounty_claims (client_uid, bounty_day) VALUES ($1, (NOW() AT TIME ZONE 'UTC')::date) ON CONFLICT DO NOTHING",
		uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "already claimed today"})
		return
	}
	if bounty.RewardGd > 0 {
		if _, err := tx.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", bounty.RewardGd, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if bounty.RewardTk > 0 {
		if _, err := tx.Exec("UPDATE users SET abyss_tokens = abyss_tokens + $1 WHERE client_uid=$2", bounty.RewardTk, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	writeJSON(w, map[string]any{
		"ok": true, "reward_tokens": bounty.RewardTk, "reward_gold": bounty.RewardGd,
		"gold": gold, "tokens": s.bot.abyssTokens(uid),
	})
}
