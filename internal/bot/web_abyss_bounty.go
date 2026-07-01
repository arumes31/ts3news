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

// abyssBountyDay returns the UTC bounty day (midnight-aligned) for now. Capturing
// it once and threading it through progress/claim checks keeps validation and the
// claim write on the same calendar day even if the flow straddles midnight.
func abyssBountyDay(now time.Time) time.Time {
	return now.UTC().Truncate(24 * time.Hour)
}

// abyssBountyProgress computes the player's progress toward the day's bounty from the
// run-history tables, scoped to the supplied UTC bounty day.
func (b *Bot) abyssBountyProgress(uid string, bounty abyssBounty, start time.Time) int64 {
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

// abyssBountyClaimedDay reports whether the player has already claimed the bounty
// for the given UTC day.
func (b *Bot) abyssBountyClaimedDay(uid string, day time.Time) bool {
	var exists bool
	_ = b.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM abyss_bounty_claims WHERE client_uid=$1 AND bounty_day = $2::date)",
		uid, day).Scan(&exists)
	return exists
}

// abyssStreakBonusTokens is the extra tokens granted for a claim at the given
// streak length: +5 per consecutive day beyond the first, capped at +30 (day 7+).
func abyssStreakBonusTokens(streak int) int {
	n := streak - 1
	if n < 0 {
		n = 0
	}
	if n > 6 {
		n = 6
	}
	return n * 5
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
	Streak   int // live streak length (0 if broken)
}

func (b *Bot) abyssBountyStatus(uid string) abyssBountyView {
	now := time.Now()
	day := abyssBountyDay(now)
	bounty := abyssDailyBounty(now)
	prog := b.abyssBountyProgress(uid, bounty, day)
	claimedToday := b.abyssBountyClaimedDay(uid, day)

	// The stored streak is "live" only if today or yesterday was claimed; otherwise a
	// day was missed and the streak is effectively broken until the next claim.
	live := 0
	if claimedToday || b.abyssBountyClaimedDay(uid, day.AddDate(0, 0, -1)) {
		_ = b.DB.QueryRow("SELECT abyss_bounty_streak FROM users WHERE client_uid=$1", uid).Scan(&live)
	}

	return abyssBountyView{
		Desc:     bounty.Desc,
		Progress: prog,
		Target:   bounty.Target,
		RewardTk: bounty.RewardTk,
		RewardGd: bounty.RewardGd,
		Met:      prog >= bounty.Target,
		Claimed:  claimedToday,
		Streak:   live,
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

	// Capture the UTC bounty day once so the progress check, streak check and the
	// claim insert all reference the same calendar day even across a midnight tick.
	now := time.Now()
	day := abyssBountyDay(now)
	bounty := abyssDailyBounty(now)
	if s.bot.abyssBountyProgress(uid, bounty, day) < bounty.Target {
		writeJSON(w, map[string]any{"ok": false, "error": "bounty not complete yet"})
		return
	}

	// Streak continues only if yesterday was also claimed; otherwise it resets to 1.
	newStreak := 1
	if s.bot.abyssBountyClaimedDay(uid, day.AddDate(0, 0, -1)) {
		var prev int
		_ = s.bot.DB.QueryRow("SELECT abyss_bounty_streak FROM users WHERE client_uid=$1", uid).Scan(&prev)
		newStreak = prev + 1
	}
	streakBonus := abyssStreakBonusTokens(newStreak)
	totalTokens := bounty.RewardTk + streakBonus

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	// The PK on (client_uid, bounty_day) makes this the one-shot guard: a second
	// claim for the same day affects zero rows and is rejected.
	res, err := tx.Exec(
		"INSERT INTO abyss_bounty_claims (client_uid, bounty_day) VALUES ($1, $2::date) ON CONFLICT DO NOTHING",
		uid, day)
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
	if totalTokens > 0 {
		if _, err := tx.Exec("UPDATE users SET abyss_tokens = abyss_tokens + $1 WHERE client_uid=$2", totalTokens, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if _, err := tx.Exec("UPDATE users SET abyss_bounty_streak = $1 WHERE client_uid=$2", newStreak, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	writeJSON(w, map[string]any{
		"ok": true, "reward_tokens": totalTokens, "reward_gold": bounty.RewardGd,
		"streak": newStreak, "streak_bonus": streakBonus,
		"gold": gold, "tokens": s.bot.abyssTokens(uid),
	})
}
