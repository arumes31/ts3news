package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"

	"ts3news/internal/content"
)

// readJSON decodes a JSON request body, tolerating an empty body.
func readJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return nil
	}
	return json.NewDecoder(r.Body).Decode(v)
}

// ---- Tiers ---------------------------------------------------------------

// abyssTier is a difficulty mode: harder tiers multiply both danger and reward,
// gate behind a best-depth requirement, and cost gold to enter. [16][54]
type abyssTier struct {
	Key        string
	Name       string
	DiffMult   float64
	RewardMult float64
	EntryGold  int64
	MinBest    int
}

var abyssTiers = map[string]abyssTier{
	"normal":    {"normal", "Normal", 1.0, 1.0, 0, 0},
	"nightmare": {"nightmare", "Nightmare", 1.6, 2.0, 500, 15},
	"hell":      {"hell", "Hell", 2.5, 4.0, 5000, 30},
}

var abyssTierOrder = []string{"normal", "nightmare", "hell"}

func abyssTierByKey(k string) (abyssTier, bool) {
	t, ok := abyssTiers[k]
	return t, ok
}

// abyssTierView is the template-facing tier with its unlock state.
type abyssTierView struct {
	abyssTier
	Unlocked bool
}

func abyssTierList(bestDepth int) []abyssTierView {
	out := make([]abyssTierView, 0, len(abyssTierOrder))
	for _, k := range abyssTierOrder {
		t := abyssTiers[k]
		out = append(out, abyssTierView{abyssTier: t, Unlocked: bestDepth >= t.MinBest})
	}
	return out
}

// ---- Player stats / meta -------------------------------------------------

// abyssStats is the player's persistent Abyss profile (best depth, tokens,
// Deep-Delver upgrade levels, lifetime tallies, streak).
type abyssStats struct {
	BestDepth      int
	Tokens         int64
	LifetimeFloors int64
	LifetimeBanked int64
	Deaths         int
	Streak         int
	UpVigor        int
	UpGreed        int
	UpFortune      int
	UpWard         int
	UpInterest     int
	UpTribute      int
	UpInsight      int
	UpSwiftness     int
	UpScavenger     int
	UpMercy         int
	UpCartographer  int
	UpQuartermaster int
	AbyssPrestige  int
}

func (b *Bot) loadAbyssStats(uid string) abyssStats {
	var st abyssStats
	_ = b.DB.QueryRow(
		`SELECT abyss_best_depth, abyss_tokens, abyss_lifetime_floors, abyss_lifetime_banked,
		        abyss_deaths, abyss_bank_streak, abyss_up_vigor, abyss_up_greed, abyss_up_fortune, abyss_up_ward,
		        abyss_up_interest, abyss_up_tribute, abyss_up_insight,
		        abyss_up_swiftness, abyss_up_scavenger, abyss_up_mercy, abyss_up_cartographer, abyss_up_quartermaster,
		        abyss_prestige
		   FROM users WHERE client_uid=$1`, uid,
	).Scan(&st.BestDepth, &st.Tokens, &st.LifetimeFloors, &st.LifetimeBanked,
		&st.Deaths, &st.Streak, &st.UpVigor, &st.UpGreed, &st.UpFortune, &st.UpWard,
		&st.UpInterest, &st.UpTribute, &st.UpInsight,
		&st.UpSwiftness, &st.UpScavenger, &st.UpMercy, &st.UpCartographer, &st.UpQuartermaster,
		&st.AbyssPrestige)
	return st
}

func (b *Bot) abyssTokens(uid string) int64 {
	var t int64
	_ = b.DB.QueryRow("SELECT abyss_tokens FROM users WHERE client_uid=$1", uid).Scan(&t)
	return t
}

func (b *Bot) grantAbyssTokens(uid string, n int) {
	if n <= 0 {
		return
	}
	_, _ = b.DB.Exec("UPDATE users SET abyss_tokens = abyss_tokens + $1 WHERE client_uid=$2", n, uid)
}

// abyssDailyFirstDescent atomically claims the once-per-day first-descent flag,
// returning true only for the first descend of the calendar day. [11]
func (b *Bot) abyssDailyFirstDescent(uid string) bool {
	res, err := b.DB.Exec(
		`UPDATE users SET abyss_last_descent = CURRENT_DATE
		  WHERE client_uid=$1 AND (abyss_last_descent IS NULL OR abyss_last_descent < CURRENT_DATE)`, uid)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

// abyssBankMultiplier rewards banking deep and on a long banked-run streak. [2][12]
func (b *Bot) abyssBankMultiplier(depth, streak int) float64 {
	d := depth
	if d > 100 {
		d = 100
	}
	s := streak
	if s > 25 {
		s = 25
	}
	return 1.0 + float64(d)*0.01 + float64(s)*0.02
}

// dbExecQuerier is satisfied by both *sql.DB and *sql.Tx, letting a helper run
// either standalone or inside an existing transaction.
type dbExecQuerier interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

// capAbyssDayGold clamps payout so a player's Abyss-banked gold can't exceed the
// daily cap, protecting the shared economy. It runs through the supplied querier
// so the cap consumption can participate in the caller's payout transaction and
// roll back together with it. [59]
func (b *Bot) capAbyssDayGold(q dbExecQuerier, uid string, payout int64) int64 {
	if payout <= 0 {
		return 0
	}
	_, _ = q.Exec(
		`UPDATE users SET abyss_day = CURRENT_DATE, abyss_day_gold = 0
		  WHERE client_uid=$1 AND (abyss_day IS NULL OR abyss_day < CURRENT_DATE)`, uid)
	var dayGold int64
	_ = q.QueryRow("SELECT abyss_day_gold FROM users WHERE client_uid=$1", uid).Scan(&dayGold)
	remaining := int64(abyssDayGoldCap) - dayGold
	if remaining <= 0 {
		return 0
	}
	if payout > remaining {
		payout = remaining
	}
	_, _ = q.Exec("UPDATE users SET abyss_day_gold = abyss_day_gold + $1 WHERE client_uid=$2", payout, uid)
	return payout
}

// forfeitAbyss ends a downed run atomically: pays insurance back to gold, feeds
// the rest of the cache to the shared deep-cache jackpot, records the death and
// resets the streak. Returns the insured refund and an error if the transaction
// could not be committed (so callers don't report a successful concede/revive on
// a refund that never landed). [1][62]
func (b *Bot) forfeitAbyss(uid string, run abyssRun) (refund int64, jackpot int64, err error) {
	if run.Insured > 0 {
		refund = run.Escrow * int64(run.Insured) / 100
	}
	remainder := run.Escrow - refund

	tx, err := b.DB.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()

	if refund > 0 {
		if _, err := tx.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", refund, uid); err != nil {
			return 0, 0, err
		}
	}
	// Feed the rest of the cache to the shared deep-cache jackpot inside the same
	// transaction, so it only persists if the refund + run-delete also commit.
	if remainder > 0 {
		inc := int64(float64(remainder) * jackpotRate)
		if inc < 1 {
			inc = 1
		}
		if _, err := tx.Exec("UPDATE arcade_jackpots SET amount = amount + $1, updated_at = NOW() WHERE game_key='abyss'", inc); err != nil {
			return 0, 0, err
		}
	}
	if run.Depth > 0 {
		if _, err := tx.Exec(
			"INSERT INTO abyss_runs (client_uid, depth, gold_banked, victory, tier) VALUES ($1,$2,$3,FALSE,$4)",
			uid, run.Depth, refund, run.Tier); err != nil {
			return 0, 0, err
		}
		if _, err := tx.Exec(
			`UPDATE users SET abyss_best_depth = GREATEST(abyss_best_depth, $1),
			        abyss_deaths = abyss_deaths + 1, abyss_bank_streak = 0 WHERE client_uid=$2`,
			run.Depth, uid); err != nil {
			return 0, 0, err
		}
		// Daily death counter feeds the comeback buff (#24): 3 deaths in one day
		// grant +10% stats on the next run.
		if _, err := tx.Exec(
			`UPDATE users SET abyss_deaths_today = CASE WHEN abyss_deaths_date = CURRENT_DATE THEN abyss_deaths_today + 1 ELSE 1 END,
			        abyss_deaths_date = CURRENT_DATE WHERE client_uid=$1`, uid); err != nil {
			return 0, 0, err
		}
	}
	// End of run: clear the per-run win streak so its combat buff can't leak into
	// regular cycle combat (which reads abyss_win_streak too).
	if _, err := tx.Exec("UPDATE users SET abyss_win_streak = 0 WHERE client_uid=$1", uid); err != nil {
		return 0, 0, err
	}
	if _, err := tx.Exec("DELETE FROM abyss_active WHERE client_uid=$1", uid); err != nil {
		return 0, 0, err
	}
	// Death forfeits the locked loot cache along with the gold.
	if _, err := tx.Exec("DELETE FROM abyss_escrow_loot WHERE client_uid=$1", uid); err != nil {
		return 0, 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	if run.Depth > 0 {
		b.grantAbyssTokens(uid, run.Depth/10) // small consolation
		b.recordGameResult(uid, "abyss", false, refund)
	}
	return refund, 0, nil
}

// awardAbyssBonusGear grants a guaranteed gear reward on a deep bank, with a
// rarity floor that rises with depth (re-rolling until the floor is met). [55][57]
func (b *Bot) awardAbyssBonusGear(uid string, depth int) string {
	floor := content.RarityCommon
	switch {
	case depth >= 50:
		floor = content.RarityEpic
	case depth >= 25:
		floor = content.RarityRare
	case depth >= 10:
		floor = content.RarityUncommon
	}
	// Draw from the Abyss-exclusive pool so deep banks can actually return ABYSS_
	// gear; the retry/fallback keeps the rarity floor met.
	g := content.RandomAbyssGearDrop()
	for i := 0; i < 20 && g.Rarity < floor; i++ {
		g = content.RandomAbyssGearDrop()
	}
	// Fallback: pick directly from the eligible pool so the floor is always met.
	if g.Rarity < floor {
		if candidates := content.GearByMinRarity(floor); len(candidates) > 0 {
			// #nosec G404 -- non-cryptographic loot pick
			g = candidates[rand.IntN(len(candidates))]
		}
	}
	res := b.awardGearDrop(uid, g)
	return res.Prefix + res.ItemName
}

// tryAbyssJackpot gives a deep bank a small chance at the shared deep-cache pot. [62]
func (b *Bot) tryAbyssJackpot(uid string, depth int) int64 {
	if depth < abyssJackpotDepth {
		return 0
	}
	// #nosec G404 -- non-cryptographic reward roll
	if rand.IntN(100) < 5 {
		return b.claimJackpot(uid, "abyss")
	}
	return 0
}

// abyssWinStreakMaxStacks caps how many consecutive-floor-win stacks count
// toward the streak combat buff.
const abyssWinStreakMaxStacks = 10

// abyssStreakBuff returns the small stacking combat buff granted for
// consecutive Abyss floor wins within a single run (resets on Downed or on
// resting). Capped well below a full Abyss gear-set bonus.
func abyssStreakBuff(streak int) content.Stats {
	if streak > abyssWinStreakMaxStacks {
		streak = abyssWinStreakMaxStacks
	}
	if streak <= 0 {
		return content.Stats{}
	}
	return content.Stats{HP: 10 * streak, STR: 5 * streak, DEF: 5 * streak, SPD: 2 * streak, CRT: 1 * streak}
}

// ---- Achievements --------------------------------------------------------

var abyssDepthAchievements = map[int]string{
	10:  "depth_10",
	25:  "depth_25",
	50:  "depth_50",
	100: "depth_100",
}

// abyssAchievementNames maps an achievement code to its player-facing name.
var abyssAchievementNames = map[string]string{
	"depth_10":    "Threshold Breaker (Depth 10)",
	"depth_25":    "Deep Diver (Depth 25)",
	"depth_50":    "Abyssal Veteran (Depth 50)",
	"depth_100":   "Voidwalker (Depth 100)",
	"boss_1":      "Giant Slayer (First Boss)",
	"boss_25":     "Boss Hunter (25 Bosses)",
	"boss_100":    "Worldbreaker (100 Bosses)",
	"bank_1m":     "Treasurer (1M Banked)",
	"bank_10m":    "Tycoon (10M Banked)",
	"bestiary_25": "Naturalist (25 Species)",
	"bestiary_50": "Zoologist (50 Species)",
	"prestige_1":  "Reborn (First Abyss Prestige)",
}

// achTier is a count threshold that, once reached, awards an achievement code.
type achTier struct {
	N    int64
	Code string
}

// Cumulative milestone ladders for the count-based achievement categories. Tiers
// are listed ascending; checkThresholdAchievements awards every newly-crossed tier.
var (
	abyssBossTiers     = []achTier{{1, "boss_1"}, {25, "boss_25"}, {100, "boss_100"}}
	abyssBankTiers     = []achTier{{1_000_000, "bank_1m"}, {10_000_000, "bank_10m"}}
	abyssBestiaryTiers = []achTier{{25, "bestiary_25"}, {50, "bestiary_50"}}
)

func abyssAchievementName(code string) string {
	if n, ok := abyssAchievementNames[code]; ok {
		return n
	}
	return code
}

// awardAchievement records an achievement once, returning true if newly earned.
func (b *Bot) awardAchievement(uid, code string) bool {
	res, err := b.DB.Exec(
		"INSERT INTO abyss_achievements (client_uid, code) VALUES ($1,$2) ON CONFLICT DO NOTHING", uid, code)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

// checkDepthAchievements awards a milestone achievement, returning its localized
// name if it was newly earned. [47]
func (b *Bot) checkDepthAchievements(uid string, depth int) string {
	if code, ok := abyssDepthAchievements[depth]; ok && b.awardAchievement(uid, code) {
		return abyssAchievementName(code)
	}
	return ""
}

// checkThresholdAchievements awards every milestone tier the player has newly
// crossed for a count-based category, returning the highest newly-earned name (or
// "" if none). awardAchievement is idempotent, so already-earned tiers are skipped.
func (b *Bot) checkThresholdAchievements(uid string, have int64, tiers []achTier) string {
	newest := ""
	for _, t := range tiers {
		if have >= t.N && b.awardAchievement(uid, t.Code) {
			newest = abyssAchievementName(t.Code)
		}
	}
	return newest
}

// checkBossKillAchievements awards boss-kill milestones from the player's total
// recorded Abyss boss kills.
func (b *Bot) checkBossKillAchievements(uid string) string {
	var n int64
	_ = b.DB.QueryRow("SELECT COUNT(*) FROM abyss_boss_kills WHERE client_uid=$1", uid).Scan(&n)
	return b.checkThresholdAchievements(uid, n, abyssBossTiers)
}

// checkBestiaryAchievements awards milestones for the number of distinct monster
// species the player has vanquished.
func (b *Bot) checkBestiaryAchievements(uid string) string {
	var n int64
	_ = b.DB.QueryRow("SELECT COUNT(*) FROM abyss_bestiary WHERE client_uid=$1", uid).Scan(&n)
	return b.checkThresholdAchievements(uid, n, abyssBestiaryTiers)
}

// checkBankAchievements awards lifetime-banked-gold milestones.
func (b *Bot) checkBankAchievements(uid string, lifetimeBanked int64) string {
	return b.checkThresholdAchievements(uid, lifetimeBanked, abyssBankTiers)
}

func (b *Bot) abyssAchievements(uid string) []string {
	rows, err := b.DB.Query("SELECT code FROM abyss_achievements WHERE client_uid=$1 ORDER BY earned_at", uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			continue
		}
		out = append(out, abyssAchievementName(code))
	}
	return out
}

// abyssAchievementCodes returns the raw earned achievement codes (as opposed to
// abyssAchievements' display names), used to drive the badge picker.
func (b *Bot) abyssAchievementCodes(uid string) []string {
	rows, err := b.DB.Query("SELECT code FROM abyss_achievements WHERE client_uid=$1 ORDER BY earned_at", uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			continue
		}
		out = append(out, code)
	}
	return out
}

// handleAbyssSetBadge lets a player display one earned achievement as a
// persistent cosmetic badge next to their name. An empty code clears it.
func (s *WebServer) handleAbyssSetBadge(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	_ = readJSON(r, &req)

	if req.Code != "" {
		var has bool
		if err := s.bot.DB.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM abyss_achievements WHERE client_uid=$1 AND code=$2)", uid, req.Code,
		).Scan(&has); err != nil || !has {
			writeJSON(w, map[string]any{"ok": false, "error": "achievement not earned"})
			return
		}
	}
	if _, err := s.bot.DB.Exec("UPDATE users SET abyss_active_badge=$1 WHERE client_uid=$2", req.Code, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "badge": req.Code, "badge_name": abyssAchievementName(req.Code)})
}

// ---- Run history ---------------------------------------------------------

type abyssHistoryRow struct {
	Depth   int
	Gold    int64
	Victory bool
	When    string
}

func (b *Bot) abyssHistory(uid string, limit int) []abyssHistoryRow {
	rows, err := b.DB.Query(
		"SELECT depth, gold_banked, victory, created_at FROM abyss_runs WHERE client_uid=$1 ORDER BY id DESC LIMIT $2",
		uid, limit)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []abyssHistoryRow
	for rows.Next() {
		var h abyssHistoryRow
		var when time.Time
		if err := rows.Scan(&h.Depth, &h.Gold, &h.Victory, &when); err != nil {
			continue
		}
		h.When = when.Format("Jan 2 15:04")
		out = append(out, h)
	}
	return out
}

// ---- Leaderboards (per tier) ---------------------------------------------

type abyssRow struct {
	Rank     int
	Nickname string
	Depth    int
	Gold     int64
}

type bossKillRow struct {
	Rank       int
	Nickname   string
	BossName   string
	Depth      int
	KillTimeMs int64
	KilledAt   string
}

type abyssBoards struct {
	Tier      string
	Day       []abyssRow
	Season    []abyssRow
	AllTime   []abyssRow
	BossKills []bossKillRow
}

func (b *Bot) topDescents(tier string, since time.Time, limit int) []abyssRow {
	rows, err := b.DB.Query(
		`SELECT COALESCE(NULLIF(u.nickname, ''), 'Adventurer') AS nick,
		        MAX(a.depth) AS depth, COALESCE(SUM(a.gold_banked), 0) AS gold
		   FROM abyss_runs a
		   LEFT JOIN users u ON u.client_uid = a.client_uid
		  WHERE a.tier = $1 AND a.created_at >= $2
		  GROUP BY a.client_uid, u.nickname
		  ORDER BY depth DESC, gold DESC
		  LIMIT $3`, tier, since, limit)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []abyssRow
	rank := 1
	for rows.Next() {
		var r abyssRow
		if err := rows.Scan(&r.Nickname, &r.Depth, &r.Gold); err != nil {
			continue
		}
		r.Rank = rank
		rank++
		out = append(out, r)
	}
	return out
}

func (b *Bot) topBossKills(limit int, tier string) []bossKillRow {
	var rows *sql.Rows
	var err error
	if tier != "" {
		rows, err = b.DB.Query(
			`SELECT COALESCE(NULLIF(u.nickname, ''), 'Adventurer') AS nick,
			        k.boss_name, k.depth, k.kill_time_ms, k.killed_at
			   FROM abyss_boss_kills k
			   LEFT JOIN users u ON u.client_uid = k.client_uid
			   WHERE k.tier = $2
			  ORDER BY k.depth DESC, k.kill_time_ms ASC
			  LIMIT $1`, limit, tier)
	} else {
		rows, err = b.DB.Query(
			`SELECT COALESCE(NULLIF(u.nickname, ''), 'Adventurer') AS nick,
			        k.boss_name, k.depth, k.kill_time_ms, k.killed_at
			   FROM abyss_boss_kills k
			   LEFT JOIN users u ON u.client_uid = k.client_uid
			  ORDER BY k.depth DESC, k.kill_time_ms ASC
			  LIMIT $1`, limit)
	}
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []bossKillRow
	rank := 1
	for rows.Next() {
		var r bossKillRow
		var t time.Time
		if err := rows.Scan(&r.Nickname, &r.BossName, &r.Depth, &r.KillTimeMs, &t); err == nil {
			r.Rank = rank
			r.KilledAt = t.Format("2006-01-02 15:04")
			out = append(out, r)
			rank++
		}
	}
	return out
}

func (b *Bot) abyssLeaderboards(tier string) abyssBoards {
	const top = 10
	now := time.Now()
	return abyssBoards{
		Tier:      tier,
		Day:       b.topDescents(tier, now.AddDate(0, 0, -1), top),
		Season:    b.topDescents(tier, abyssSeasonStart(), top),
		AllTime:   b.topDescents(tier, time.Unix(0, 0), top),
		BossKills: b.topBossKills(top, tier),
	}
}

// ---- Mob escalation & zone flavour ---------------------------------------

// escalateMobs deepens the threat with depth by layering mob effects and, on
// world-boss floors, promoting and empowering the lead enemy. [15][64]
func escalateMobs(mobs []content.Mob, depth int, worldBoss bool) {
	for i := range mobs {
		// #nosec G404 -- cosmetic/balance roll, not security-sensitive
		if depth >= 8 && rand.Float64() < 0.15+float64(depth)*0.005 {
			mobs[i].Effects = append(mobs[i].Effects, content.EffectEnraged)
		}
		// #nosec G404
		if depth >= 12 && rand.Float64() < 0.10+float64(depth)*0.004 {
			mobs[i].Effects = append(mobs[i].Effects, content.EffectArmored)
		}
	}
	if worldBoss && len(mobs) > 0 {
		m := &mobs[0]
		m.Type = content.MobLegendary
		m.Stats.HP = m.Stats.HP * 2
		m.MaxHP = m.Stats.HP
		m.CurrentHP = m.MaxHP
		m.Stats.STR = int(float64(m.Stats.STR) * 1.5)
		m.Effects = append(m.Effects, content.EffectRegen)
		m.RewardXP = m.RewardXP * 2
	}
}

var abyssZonesShallow = []string{
	"The Cracked Threshold", "Gloomwell Steps", "The Weeping Stair",
	"Ashen Antechamber", "The First Dark", "Mournhollow",
}
var abyssZonesMid = []string{
	"The Sunless Vault", "Marrowdeep", "The Choking Galleries",
	"Veins of the World", "The Drowned Catacomb", "Emberfall Reach",
}
var abyssZonesDeep = []string{
	"The Throat of the World", "The Nadir", "Where Light Forgets",
	"The Maw Eternal", "Abyssal Heart", "The Last Descent",
}

// abyssZoneName picks a depth-appropriate Abyss zone name. [33]
func abyssZoneName(depth int) string {
	var pool []string
	switch {
	case depth < 10:
		pool = abyssZonesShallow
	case depth < 30:
		pool = abyssZonesMid
	default:
		pool = abyssZonesDeep
	}
	// #nosec G404 -- cosmetic name selection
	return pool[rand.IntN(len(pool))]
}

// ---- Salvage & upgrades --------------------------------------------------

// handleAbyssSalvage vendors all common/uncommon junk from the inventory for
// immediate (non-escrow) gold — a merchant at the Abyss threshold. [60]
func (s *WebServer) handleAbyssSalvage(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	rows, err := s.bot.DB.Query("SELECT id, gear_id FROM user_inventory WHERE client_uid=$1", uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	type junk struct {
		id    int64
		value int64
		mat   string
		matN  int
	}
	var toSell []junk
	for rows.Next() {
		var id int64
		var gid string
		if err := rows.Scan(&id, &gid); err != nil {
			continue
		}
		g, ok := content.GetGearByID(gid)
		if !ok || g.Rarity > content.RarityUncommon {
			continue
		}
		v := gearPrice(g) / 2
		if v < 1 {
			v = 1
		}
		mat, matN := materialYieldForRarity(g.Rarity)
		toSell = append(toSell, junk{id, v, mat, matN})
	}
	_ = rows.Close()

	if len(toSell) == 0 {
		var gold int64
		_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
		writeJSON(w, map[string]any{"ok": true, "sold": 0, "value": 0, "gold": gold})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var total int64
	var count int
	matGained := map[string]int{}
	for _, j := range toSell {
		res, err := tx.Exec("DELETE FROM user_inventory WHERE id=$1 AND client_uid=$2", j.id, uid)
		if err != nil {
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			total += j.value
			count++
			matGained[j.mat] += j.matN
		}
	}
	var gold int64
	if total > 0 {
		if err := tx.QueryRow("UPDATE users SET gold = gold + $1 WHERE client_uid=$2 RETURNING gold", total, uid).Scan(&gold); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	} else {
		_ = tx.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	// Crafting materials (#101), boosted by the Scavenger node (#155).
	scav := s.bot.loadAbyssStats(uid).UpScavenger
	for mat, n := range matGained {
		s.bot.grantMaterial(uid, mat, scavengerYield(n, scav))
	}
	writeJSON(w, map[string]any{"ok": true, "sold": count, "value": total, "gold": gold,
		"materials": s.bot.loadMaterials(uid)})
}

// abyssDismantleTokens is the Abyss-token yield for dismantling a spare gear piece
// of the given rarity. Below Rare yields nothing (use Salvage for those).
func abyssDismantleTokens(rarity content.Rarity) int64 {
	switch {
	case rarity >= content.RarityMythic:
		return 10
	case rarity >= content.RarityLegendary:
		return 6
	case rarity >= content.RarityEpic:
		return 3
	case rarity >= content.RarityRare:
		return 1
	}
	return 0
}

// handleAbyssDismantle breaks down all Rare-or-better spares sitting in the backpack
// (user_inventory — never equipped gear) into Abyss Tokens, giving the token economy
// a faucet to match the Token Shop sink. Common/uncommon junk still goes to Salvage.
func (s *WebServer) handleAbyssDismantle(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	// Batch filters + preview (#110): limit which rarities break and dry-run the
	// yield before committing.
	var req struct {
		Preview   bool `json:"preview"`
		MaxRarity int  `json:"max_rarity"` // 0 = no cap; e.g. 4 = keep Legendary+ safe
	}
	_ = readJSON(r, &req)

	rows, err := s.bot.DB.Query("SELECT id, gear_id, item_data FROM user_inventory WHERE client_uid=$1", uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	type spare struct {
		id     int64
		tokens int64
		mat    string
		matN   int
	}
	var toBreak []spare
	for rows.Next() {
		var id int64
		var gid string
		var itemData sql.NullString
		if err := rows.Scan(&id, &gid, &itemData); err != nil {
			continue
		}
		// Reconstruct the item from its persisted data so upgraded/generated gear is
		// valued at its actual rarity, not the static catalog entry.
		g, ok := s.bot.makeGear(gid, itemData)
		if !ok {
			continue
		}
		if req.MaxRarity > 0 && int(g.Rarity) > req.MaxRarity {
			continue
		}
		if tk := abyssDismantleTokens(g.Rarity); tk > 0 {
			mat, matN := materialYieldForRarity(g.Rarity)
			toBreak = append(toBreak, spare{id, tk, mat, matN})
		}
	}
	_ = rows.Close()

	if req.Preview {
		var tk int64
		mats := map[string]int{}
		for _, sp := range toBreak {
			tk += sp.tokens
			mats[sp.mat] += sp.matN
		}
		writeJSON(w, map[string]any{"ok": true, "preview": true, "count": len(toBreak), "tokens_gained": tk, "materials_gained": mats})
		return
	}

	if len(toBreak) == 0 {
		writeJSON(w, map[string]any{"ok": true, "dismantled": 0, "tokens_gained": 0, "tokens": s.bot.abyssTokens(uid)})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var total int64
	var count int
	matGained := map[string]int{}
	for _, sp := range toBreak {
		res, err := tx.Exec("DELETE FROM user_inventory WHERE id=$1 AND client_uid=$2", sp.id, uid)
		if err != nil {
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			total += sp.tokens
			count++
			matGained[sp.mat] += sp.matN
		}
	}
	if total > 0 {
		if _, err := tx.Exec("UPDATE users SET abyss_tokens = abyss_tokens + $1 WHERE client_uid=$2", total, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	// Crafting materials (#101), boosted by the Scavenger node (#155).
	scav := s.bot.loadAbyssStats(uid).UpScavenger
	for mat, n := range matGained {
		s.bot.grantMaterial(uid, mat, scavengerYield(n, scav))
	}
	writeJSON(w, map[string]any{"ok": true, "dismantled": count, "tokens_gained": total, "tokens": s.bot.abyssTokens(uid),
		"materials": s.bot.loadMaterials(uid)})
}

// handleAbyssInsure buys death-insurance on the active run: pay gold now to
// protect a % of the escrow if you die. The Ward upgrade discounts the premium. [1]
func (s *WebServer) handleAbyssInsure(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Pct int `json:"pct"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if req.Pct != 25 && req.Pct != 50 && req.Pct != 75 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid amount"})
		return
	}

	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "no live run"})
		return
	}
	if run.Insured >= req.Pct {
		writeJSON(w, map[string]any{"ok": false, "error": "already insured"})
		return
	}

	st := s.bot.loadAbyssStats(uid)
	cost := abyssInsuranceCost(run.Escrow, req.Pct, st.UpWard)
	if cost < 1 {
		cost = 1
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid=$2 AND gold >= $1", cost, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
		return
	}
	if _, err := tx.Exec("UPDATE abyss_active SET insured=$1 WHERE client_uid=$2", req.Pct, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	writeJSON(w, map[string]any{"ok": true, "insured": req.Pct, "cost": cost, "gold": gold})
}

// abyssInsuranceCost is the premium to protect pct% of an escrow, discounted by
// the Ward upgrade level.
func abyssInsuranceCost(escrow int64, pct, ward int) int64 {
	rate := 0.5 - float64(ward)*0.05
	if rate < 0.25 {
		rate = 0.25
	}
	return int64(float64(escrow) * float64(pct) / 100.0 * rate)
}

// abyssUpgradeCols maps a Deep-Delver node to its column; the whitelist prevents
// any SQL-identifier injection from the request.
var abyssUpgradeCols = map[string]string{
	"vigor":    "abyss_up_vigor",
	"greed":    "abyss_up_greed",
	"fortune":  "abyss_up_fortune",
	"ward":     "abyss_up_ward",
	"interest": "abyss_up_interest",
	"tribute":  "abyss_up_tribute",
	"insight":  "abyss_up_insight",
	"swiftness":     "abyss_up_swiftness",
	"scavenger":     "abyss_up_scavenger",
	"mercy":         "abyss_up_mercy",
	"cartographer":  "abyss_up_cartographer",
	"quartermaster": "abyss_up_quartermaster",
}

// abyssUpgradeMinDepth gates the expansion nodes behind depth records (#160):
// they unlock by descending, not just by hoarding tokens.
var abyssUpgradeMinDepth = map[string]int{
	"swiftness":     10,
	"scavenger":     15,
	"mercy":         20,
	"cartographer":  25,
	"quartermaster": 30,
}

const abyssUpgradeMaxLevel = 5

// handleAbyssUpgrade spends tokens on a permanent Deep-Delver upgrade. [44][45]
func (s *WebServer) handleAbyssUpgrade(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Node string `json:"node"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	col, ok := abyssUpgradeCols[req.Node]
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown upgrade"})
		return
	}
	if minDepth := abyssUpgradeMinDepth[req.Node]; minDepth > 0 {
		if s.bot.loadAbyssStats(uid).BestDepth < minDepth {
			writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("locked — reach depth %d first", minDepth)})
			return
		}
	}

	var level int
	var tokens int64
	if err := s.bot.DB.QueryRow("SELECT "+col+", abyss_tokens FROM users WHERE client_uid=$1", uid).Scan(&level, &tokens); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if level >= abyssUpgradeMaxLevel {
		writeJSON(w, map[string]any{"ok": false, "error": "maxed"})
		return
	}
	cost := int64(level+1) * 10
	if tokens < cost {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough tokens"})
		return
	}
	// Enforce the spend and level cap in one guarded statement (col is whitelisted
	// via abyssUpgradeCols) so the token debit and increment can't overspend or
	// exceed the max even if the pre-check raced.
	res, err := s.bot.DB.Exec(
		"UPDATE users SET abyss_tokens = abyss_tokens - $1, "+col+" = "+col+" + 1 "+
			"WHERE client_uid=$2 AND abyss_tokens >= $1 AND "+col+" < $3", cost, uid, abyssUpgradeMaxLevel)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough tokens"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "node": req.Node, "level": level + 1, "tokens": tokens - cost})
}

func (b *Bot) loadUnlockedLore(uid string) []int {
	rows, err := b.DB.Query("SELECT lore_id FROM abyss_lore_unlocked WHERE client_uid = $1", uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []int
	for rows.Next() {
		var lid int
		if err := rows.Scan(&lid); err == nil {
			out = append(out, lid)
		}
	}
	return out
}

type abyssBestiaryRow struct {
	MobName string
	Kills   int
}

func (b *Bot) loadAbyssBestiary(uid string) []abyssBestiaryRow {
	rows, err := b.DB.Query("SELECT mob_name, kills FROM abyss_bestiary WHERE client_uid = $1 ORDER BY kills DESC", uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []abyssBestiaryRow
	for rows.Next() {
		var r abyssBestiaryRow
		if err := rows.Scan(&r.MobName, &r.Kills); err == nil {
			out = append(out, r)
		}
	}
	return out
}

func (b *Bot) recordAbyssKills(uid string, mobNames []string) {
	for _, name := range mobNames {
		_, _ = b.DB.Exec(
			`INSERT INTO abyss_bestiary (client_uid, mob_name, kills, first_kill_at)
			 VALUES ($1, $2, 1, NOW())
			 ON CONFLICT (client_uid, mob_name)
			 DO UPDATE SET kills = abyss_bestiary.kills + 1`,
			uid, name,
		)
	}
}

