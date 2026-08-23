package bot

// The Abyss — 300 improvements (docs/ABYSS_IMPROVEMENTS_300.md), groups A
// (AB-1..25, core loop & risk/reward) and B (AB-26..50, floors & events).
// Small run-loop mechanics that hook into the handlers in web_abyss.go /
// web_abyss_econ.go / web_abyss_features.go.
//
// Per-run state lives in a single app_meta JSON map (no schema change) that
// handleAbyssEnter resets on every fresh descent. Persistent per-player tracks
// (killer experience, revive streak, mirror memory, well lifetime, echo seed)
// use their own app_meta keys so they survive the run boundary. Everything
// here runs under the per-uid Abyss lock (lockAbyss).

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ts3news/internal/content"
)

// ---- Per-run flags ----------------------------------------------------------

// abyssRunFlagsKey stores the active run's ad-hoc flags (death wish, anchor
// rune, cold muscles, …) as one JSON map — reset on every fresh descent.
func abyssRunFlagsKey(uid string) string { return "abyss_run_flags_" + uid }

func (b *Bot) loadRunFlags(uid string) map[string]int64 {
	out := map[string]int64{}
	var js string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssRunFlagsKey(uid)).Scan(&js)
	if js != "" {
		_ = json.Unmarshal([]byte(js), &out)
	}
	return out
}

func (b *Bot) saveRunFlags(uid string, m map[string]int64) error {
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal run flags: %w", err)
	}
	_, err = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssRunFlagsKey(uid), string(data))
	if err != nil {
		return fmt.Errorf("save run flags: %w", err)
	}
	return nil
}

// setRunFlag updates a single run flag (read-modify-write under the Abyss lock).
func (b *Bot) setRunFlag(uid, key string, v int64) error {
	m := b.loadRunFlags(uid)
	m[key] = v
	return b.saveRunFlags(uid, m)
}

// ---- Small shared scales ----------------------------------------------------

// abyssGreedyGripStacks (AB-1): one stack per 10 unbanked floors, capped at 10.
// Each stack adds +2% cache interest and −2% DEF.
func abyssGreedyGripStacks(depth int) int {
	s := depth / 10
	if s > 10 {
		s = 10
	}
	return s
}

// abyssHeavyPocketsPct (AB-2): a cache above 100k slows the delver −1% SPD per
// 50k held, capped at −20%.
func abyssHeavyPocketsPct(escrow int64) int {
	if escrow <= 100_000 {
		return 0
	}
	p := int((escrow - 100_000) / 50_000)
	if p > 20 {
		p = 20
	}
	return p
}

// abyssNextTier returns the tier one step up from key (AB-25 hybrid runs).
func abyssNextTier(key string) (abyssTier, bool) {
	for i, k := range abyssTierOrder {
		if k == key && i+1 < len(abyssTierOrder) {
			return abyssTiers[abyssTierOrder[i+1]], true
		}
	}
	return abyssTier{}, false
}

// abyssLootHint (AB-40) labels the expected loot quality of an offered floor,
// shown when the delver owns the Cartographer node.
func abyssLootHint(floorType string, depth int) string {
	if floorType == "combat" && depth%abyssBossEvery == 0 {
		return "Boss hoard — rare or better expected"
	}
	switch {
	case depth >= 40:
		return "Rich veins — rare or better expected"
	case depth >= 25:
		return "Decent finds — uncommon or better expected"
	default:
		return "Mostly common finds expected"
	}
}

// ---- AB-12 experience vs killer ----------------------------------------------

// abyssKillerExpCap is the per-family damage bonus cap in tenths of a percent
// (+5%). Each death to a mob family adds +0.1%.
const abyssKillerExpCap = 50

func abyssKillerExpKey(uid string) string { return "abyss_killer_exp_" + uid }

func (b *Bot) loadKillerExp(uid string) map[string]int {
	out := map[string]int{}
	var js string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssKillerExpKey(uid)).Scan(&js)
	if js != "" {
		_ = json.Unmarshal([]byte(js), &out)
	}
	return out
}

// killerExpBonusTenths returns the strongest grudge (tenths of a percent) the
// delver holds against any of the given mobs' families.
func (b *Bot) killerExpBonusTenths(uid string, mobs []content.Mob) int {
	exp := b.loadKillerExp(uid)
	best := 0
	for _, m := range mobs {
		if v := exp[m.Name]; v > best {
			best = v
		}
	}
	return best
}

// bumpKillerExp records a death: +0.1% permanent damage vs each killer's family.
func (b *Bot) bumpKillerExp(uid string, mobNames []string) {
	if len(mobNames) == 0 {
		return
	}
	exp := b.loadKillerExp(uid)
	changed := false
	for _, n := range mobNames {
		if exp[n] < abyssKillerExpCap {
			exp[n]++
			changed = true
		}
	}
	if !changed {
		return
	}
	data, err := json.Marshal(exp)
	if err != nil {
		return
	}
	_, _ = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssKillerExpKey(uid), string(data))
}

// ---- AB-22 revive pity streak -------------------------------------------------

func abyssReviveStreakKey(uid string) string { return "abyss_revive_streak_" + uid }

// abyssReviveStreak counts consecutive daily deaths without a successful revive
// gamble; each adds +5% to the next double-or-nothing offer (cap +25%).
func (b *Bot) abyssReviveStreak(uid string) int {
	var s string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssReviveStreakKey(uid)).Scan(&s)
	n, _ := strconv.Atoi(s)
	return n
}

func (b *Bot) setAbyssReviveStreak(uid string, n int) {
	_, _ = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssReviveStreakKey(uid), strconv.Itoa(n))
}

// ---- AB-5 echo banking / AB-23 double bank -------------------------------------

func abyssEchoSeedKey(uid string) string { return "abyss_echo_seed_" + uid }

// setAbyssEchoSeed stores the head-start cache the next descent begins with.
func (b *Bot) setAbyssEchoSeed(uid string, amt int64) {
	if amt <= 0 {
		return
	}
	_, _ = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssEchoSeedKey(uid), strconv.FormatInt(amt, 10))
}

// peekAbyssEchoSeed reads the stored head-start without consuming it (the enter
// handler clears it only after the run-insert commits).
func (b *Bot) peekAbyssEchoSeed(uid string) int64 {
	var s string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssEchoSeedKey(uid)).Scan(&s)
	seed, _ := strconv.ParseInt(s, 10, 64)
	return seed
}

func (b *Bot) clearAbyssEchoSeed(uid string) {
	_, _ = b.DB.Exec("DELETE FROM app_meta WHERE key=$1", abyssEchoSeedKey(uid))
}

// ---- AB-27 event memory ---------------------------------------------------------

func abyssEventVisitsKey(uid string) string { return "abyss_run_event_visits_" + uid }

// loadEventVisits returns how often each event type was visited this run; every
// revisit improves the event's offer by 10% (cap +50%).
func (b *Bot) loadEventVisits(uid string) map[string]int {
	out := map[string]int{}
	var js string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssEventVisitsKey(uid)).Scan(&js)
	if js != "" {
		_ = json.Unmarshal([]byte(js), &out)
	}
	return out
}

func (b *Bot) saveEventVisits(uid string, m map[string]int) {
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	_, _ = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssEventVisitsKey(uid), string(data))
}

// ---- AB-48 wishing well lifetime contributions -----------------------------------

func abyssWellLifetimeKey(uid string) string { return "abyss_well_lifetime_" + uid }

func (b *Bot) abyssWellLifetime(uid string) int64 {
	var s string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssWellLifetimeKey(uid)).Scan(&s)
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// abyssAddWellLifetime credits a toss and returns the new lifetime total.
func (b *Bot) abyssAddWellLifetime(uid string, n int64) int64 {
	var s string
	err := b.DB.QueryRow(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = (COALESCE(NULLIF(app_meta.value, '')::bigint, 0) + $3)::text
		RETURNING value`, abyssWellLifetimeKey(uid), strconv.FormatInt(n, 10), n).Scan(&s)
	if err != nil {
		return 0
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// ---- AB-50 hall of mirrors memory --------------------------------------------------

type abyssMirrorMemory struct {
	Pick   string `json:"pick"`
	Streak int    `json:"streak"`
}

func abyssMirrorKey(uid string) string { return "abyss_mirror_memory_" + uid }

func (b *Bot) loadAbyssMirrorMemory(uid string) abyssMirrorMemory {
	var m abyssMirrorMemory
	var js string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssMirrorKey(uid)).Scan(&js)
	if js != "" {
		_ = json.Unmarshal([]byte(js), &m)
	}
	return m
}

func (b *Bot) saveAbyssMirrorMemory(uid string, m abyssMirrorMemory) {
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	_, _ = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssMirrorKey(uid), string(data))
}

// ---- AB-38 map table: event cadence reveal ------------------------------------------

// abyssNextEventIn reports how many floors until the next scheduled event floor
// (0 when unknown). Shown to delvers owning the sanctuary Map Table upgrade.
func (b *Bot) abyssNextEventIn(uid string, depth int) int {
	var s string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", "abyss_next_event_depth_"+uid).Scan(&s)
	next, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	n := next - depth
	if n < 0 {
		n = 0
	}
	return n
}

// ---- AB-27/AB-48 event-state enrichment ----------------------------------------------

// enrichEventState post-processes a freshly rolled event payload before it is
// stored: it counts the visit (AB-27 event memory → mem_mult the action
// handlers apply to their offers) and stamps the wishing well with the delver's
// lifetime contributions (AB-48).
func (b *Bot) enrichEventState(uid, eventState string) string {
	if eventState == "" {
		return eventState
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(eventState), &m); err != nil {
		return eventState
	}
	typ, _ := m["type"].(string)
	if typ == "" {
		return eventState
	}
	visits := b.loadEventVisits(uid)
	visits[typ]++
	b.saveEventVisits(uid, visits)
	if visits[typ] > 1 {
		mult := 1 + 0.10*float64(visits[typ]-1)
		if mult > 1.5 {
			mult = 1.5
		}
		m["mem_mult"] = mult
	}
	if typ == "wishing_well" {
		m["lifetime"] = b.abyssWellLifetime(uid)
	}
	out, err := json.Marshal(m)
	if err != nil {
		return eventState
	}
	return string(out)
}

// ---- AB-17 downed timeout --------------------------------------------------------------

// abyssDownedTimeout is how long a downed delver may weigh revive vs concede
// before the Abyss loses patience and auto-concedes with a 10% pity cache.
const abyssDownedTimeout = 5 * time.Minute

// autoConcedeIfTimedOut enforces AB-17: a downed run left undecided for five
// minutes is forfeited with at least a 10% pity refund. Returns true when it
// wrote a response (the caller must stop).
func (s *WebServer) autoConcedeIfTimedOut(w http.ResponseWriter, uid string, run abyssRun) bool {
	if !run.Active || !run.Downed || run.LastActionAt.IsZero() || time.Since(run.LastActionAt) < abyssDownedTimeout {
		return false
	}
	if run.Insured < 10 {
		run.Insured = 10 // the pity cache
	}
	payout, err := s.bot.forfeitAbyss(uid, run)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	writeJSON(w, map[string]any{
		"ok": true, "auto_conceded": true, "insured_refund": payout,
		"gold": gold, "tokens": s.bot.abyssTokens(uid),
		"msg": "⏳ Five minutes in the dirt — the Abyss loses patience and drags you out. A 10% pity cache is paid.",
	})
	return true
}

// ---- AB-16 bankers' raffle ---------------------------------------------------------------

func abyssRaffleDay(t time.Time) string { return t.UTC().Format("2006-01-02") }

// abyssRaffleEnter records today's bank: 1% of the payout feeds the daily pot
// and the delver joins today's draw.
func (b *Bot) abyssRaffleEnter(uid string, fee int64) {
	day := abyssRaffleDay(time.Now())
	if fee > 0 {
		_, _ = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
			ON CONFLICT (key) DO UPDATE SET value = (COALESCE(NULLIF(app_meta.value, '')::bigint, 0) + $3)::text`,
			"abyss_raffle_pot_"+day, strconv.FormatInt(fee, 10), fee)
	}
	_, _ = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, '1')
		ON CONFLICT (key) DO NOTHING`, "abyss_raffle_entry_"+day+"_"+uid)
}

// abyssRafflePot returns today's accumulated pot (for display).
func (b *Bot) abyssRafflePot() int64 {
	var s string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", "abyss_raffle_pot_"+abyssRaffleDay(time.Now())).Scan(&s)
	pot, _ := strconv.ParseInt(s, 10, 64)
	return pot
}

// abyssRaffleSettle lazily draws yesterday's raffle on the first bank of a new
// day: a random entrant takes the pot. Only yesterday is settled — a day nobody
// banks on never disturbs older pots. Returns the winnings when the caller won.
func (b *Bot) abyssRaffleSettle(uid string) int64 {
	yesterday := abyssRaffleDay(time.Now().Add(-24 * time.Hour))
	res, err := b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, '1')
		ON CONFLICT (key) DO NOTHING`, "abyss_raffle_settled_"+yesterday)
	if err != nil {
		return 0
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0 // already drawn
	}
	var s string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", "abyss_raffle_pot_"+yesterday).Scan(&s)
	pot, _ := strconv.ParseInt(s, 10, 64)
	if pot <= 0 {
		return 0
	}
	prefix := "abyss_raffle_entry_" + yesterday + "_"
	rows, err := b.DB.Query("SELECT key FROM app_meta WHERE key LIKE $1", prefix+"%")
	if err != nil {
		return 0
	}
	var entrants []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err == nil {
			entrants = append(entrants, strings.TrimPrefix(k, prefix))
		}
	}
	_ = rows.Close()
	if len(entrants) == 0 {
		return 0
	}
	winner := entrants[rand.IntN(len(entrants))] // #nosec G404 -- non-cryptographic raffle draw
	_, _ = b.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", pot, winner)
	if winner == uid {
		return pot
	}
	return 0
}

// ---- AB-4 safe-word bank confirm ---------------------------------------------------------

func abyssBankConfirmKey(uid string) string { return "abyss_bank_confirm_" + uid }

// abyssBankConfirmDisabled reports whether the delver opted out of typing BANK
// for banks above 1M.
func (b *Bot) abyssBankConfirmDisabled(uid string) bool {
	var s string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssBankConfirmKey(uid)).Scan(&s)
	return s == "off"
}

// handleAbyssBankConfirmToggle (AB-4) switches the safe-word confirm for
// banks above 1M on or off (default on).
func (s *WebServer) handleAbyssBankConfirmToggle(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		On bool `json:"on"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	v := "on"
	if !req.On {
		v = "off"
	}
	if _, err := s.bot.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssBankConfirmKey(uid), v); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	msg := "Safe-word confirm enabled — banks above 1M will ask you to type BANK."
	if !req.On {
		msg = "Safe-word confirm disabled — big banks go through without asking."
	}
	writeJSON(w, map[string]any{"ok": true, "on": req.On, "msg": msg})
}

// ---- AB-9 death wish -----------------------------------------------------------------------

// handleAbyssDeathWish toggles the per-floor death wish: the next fought floor
// is 3× as deadly and pays double, with a skull chip in the response.
func (s *WebServer) handleAbyssDeathWish(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		On bool `json:"on"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "no live run"})
		return
	}
	v := int64(0)
	if req.On {
		v = 1
	}
	if err := s.bot.setRunFlag(uid, "death_wish", v); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	msg := "Death wish lifted. The dark pretends not to be disappointed."
	if req.On {
		msg = "💀 Death wish accepted — the next floor is 3× as deadly and pays double."
	}
	writeJSON(w, map[string]any{"ok": true, "death_wish": req.On, "msg": msg})
}

// ---- AB-6 anchor rune ------------------------------------------------------------------------

// abyssAnchorRuneCost is the token price of the one-charge anchor rune.
const abyssAnchorRuneCost = 20

// handleAbyssAnchorRune sets a one-charge anchor rune on the active run: the
// next death forfeits only half the cache (see forfeitAbyss).
func (s *WebServer) handleAbyssAnchorRune(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "no live run"})
		return
	}
	if s.bot.loadRunFlags(uid)["anchor_rune"] == 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "the anchor rune is already set"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if !deductTokens(w, tx, uid, abyssAnchorRuneCost) {
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := s.bot.setRunFlag(uid, "anchor_rune", 1); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "tokens": s.bot.abyssTokens(uid),
		"msg": fmt.Sprintf("⚓ Anchor rune set (−🜲%d) — if you fall this run, half your cache is hauled up with you. One charge.", abyssAnchorRuneCost),
	})
}
