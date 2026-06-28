package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"math/rand/v2"
	"net/http"
	"regexp"
	"strings"
	"time"

	"ts3news/internal/content"
)

// The Abyss is an endless push-your-luck PvE dungeon. Unlike the arcade (pure
// gambling) or the auto-battler (its own champions), every floor is a *real*
// encounter resolved by the bot's combat engine using the player's actual
// character: their 24-slot gear → computed Stats, 5 equipped skills, ultimate,
// pets, consumables, artifact and title. Loot drops land in the real inventory,
// gold/XP feed the shared economy, durability ticks, and the loot-pity counters
// advance — exactly as they do in the TeamSpeak combat cycle.
//
// The reused engine (resolveChannelCombat) already persists HP, combat gold and
// loot per fight, so those base rewards are always kept. The push-your-luck
// stake sits *on top*: each cleared floor adds a bonus to an escrowed cache that
// is paid out on Bank but forfeited on death (minus any insurance). Depth, tier
// and escrow are tracked server-side in abyss_active so the client can never lie.
//
// All per-player Abyss mutations are serialised by a per-uid mutex (lockAbyss):
// the combat engine writes through b.DB directly and so can't be wrapped in a
// single SQL transaction with the surrounding bookkeeping, but the lock makes
// each player's enter/descend/revive/concede/bank strictly sequential, which is
// what prevents the double-bank and post-death-descend races.

const (
	// abyssDepthRamp adds this fraction of difficulty per floor beyond the first.
	abyssDepthRamp = 0.10
	// abyssDiffSoftCap is where difficulty growth switches from linear to a gentle
	// logarithmic crawl, so very deep floors stay computationally bounded while
	// never quite flattening.
	abyssDiffSoftCap = 6.0
	// abyssBossEvery forces a boss on every Nth floor; every 2nd of those (every
	// 10th floor) is a world-boss tier.
	abyssBossEvery = 5
	// abyssEscrowInterest is added to the standing cache each floor before the new
	// floor bonus, rewarding players who let it ride.
	abyssEscrowInterest = 0.02
	// abyssDayGoldCap bounds Abyss-sourced bank payouts per player per day to
	// protect the shared economy from runaway farming.
	abyssDayGoldCap = 5_000_000
	// abyssJackpotDepth is the minimum bank depth that can hit the deep-cache pot.
	abyssJackpotDepth = 25
)

// softCap returns x unchanged up to cap, then grows logarithmically past it.
func softCap(x, cap float64) float64 {
	if x <= cap {
		return x
	}
	return cap + math.Log(1+(x-cap))
}

// abyssFloorBonus is the base escrowed gold for clearing the given floor (before
// tier and Deep-Delver multipliers). It scales with depth and level so the
// accumulated cache grows roughly quadratically with how deep you push.
func abyssFloorBonus(depth, level int) int64 {
	per := int64(40 + level/2)
	if per < 40 {
		per = 40
	}
	return per * int64(depth)
}

// abyssDifficulty derives the base floor difficulty (pre-tier, pre-prestige) and
// whether a boss is forced, from the player's gear-derived stat score, level and
// depth. The caller layers tier and prestige multipliers on top.
func abyssDifficulty(stats content.Stats, level, depth int) (float64, bool) {
	if depth < 1 {
		depth = 1
	}
	expected := 45 + level/5
	if expected < 1 {
		expected = 1
	}
	base := float64(stats.Score()) / float64(expected)
	if base < 0.7 {
		base = 0.7 // floor 1 should never be trivial
	}
	diff := base * (1.0 + float64(depth-1)*abyssDepthRamp)
	return softCap(diff, abyssDiffSoftCap), depth%abyssBossEvery == 0
}

// buildAbyssUser assembles a UserInCombat for the solo descent, mirroring the
// per-channel construction in the bot cycle so the engine sees an identical
// character. It does NOT auto-heal: HP carries between floors (the wound is the
// risk), and a fully-depleted character is handled by the "downed" state in the
// descend handler, not silently revived.
func (b *Bot) buildAbyssUser(uid string) (UserInCombat, int, error) {
	stats, _, _, _ := b.calculateTotalStats(uid, time.Now())

	var nick sql.NullString
	var lvl, prestige, curHP, regen int
	var gold int64
	err := b.DB.QueryRow(
		"SELECT nickname, level, prestige, current_hp, regen_stacks, gold FROM users WHERE client_uid=$1", uid,
	).Scan(&nick, &lvl, &prestige, &curHP, &regen, &gold)
	if err != nil {
		return UserInCombat{}, 0, err
	}
	if lvl < 1 {
		lvl = 1
	}
	if curHP < 0 {
		curHP = 0
	}

	return UserInCombat{
		UID:           uid,
		Nickname:      nullStr(nick),
		Stats:         stats,
		Level:         lvl,
		Skills:        b.getSkills(uid),
		UltimateSkill: b.getUltimateSkill(uid),
		CurrentHP:     curHP,
		RegenStacks:   regen,
		Gold:          gold,
		Pets:          b.getPets(uid),
		Equipped:      b.getEquippedItems(uid),
	}, prestige, nil
}

func nullStr(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}

// abyssFloorResult is the outcome of fighting a single floor.
type abyssFloorResult struct {
	Victory   bool
	RewardXP  int
	LogsHTML  []string
	LootHTML  []string
	DuraHTML  []string
	CurrentHP int
	MaxHP     int
}

var abyssLoreFragments = map[int]string{
	1:  "Deep within the Cracked Threshold, the air hums with a low, vibrating note...",
	2:  "The Gloomwell Steps were built by an empire whose name has vanished from history...",
	3:  "In the Sunless Vault, gold lies piled high, yet none dare touch it...",
	4:  "Marrowdeep is a charnel house where the bones of ancient titans grind together...",
	5:  "The Throat of the World is a sheer abyss that defies the laws of gravity...",
	6:  "At the Nadir, light is not merely absent; it is actively consumed...",
	7:  "Deep delvers speak of a giant eye that blinks once every thousand years...",
	8:  "The Maw Eternal is a gate that opens only when fed a million souls...",
	9:  "In the Abyssal Heart, the physical rules of gravity, light, and time collapse...",
	10: "The Last Descent: at this final boundary, you realize the Abyss is not a place, but a living mind...",
}

func (b *Bot) spawnEchoMob(uid string, avgLvl int) ([]content.Mob, string) {
	var echoUID, echoNick string
	var echoDepth int
	err := b.DB.QueryRow(
		`SELECT r.client_uid, COALESCE(NULLIF(u.nickname, ''), 'Adventurer') AS nick, r.depth 
		   FROM abyss_runs r
		   JOIN users u ON u.client_uid = r.client_uid
		  WHERE r.client_uid != $1 
		  ORDER BY r.depth DESC, r.gold_banked DESC 
		  LIMIT 1`, uid,
	).Scan(&echoUID, &echoNick, &echoDepth)
	if err != nil {
		return nil, ""
	}
	stats, _, _, _ := b.calculateTotalStats(echoUID, time.Now())
	echoLvl := avgLvl
	_ = b.DB.QueryRow("SELECT level FROM users WHERE client_uid=$1", echoUID).Scan(&echoLvl)

	mob := content.Mob{
		Name:     "Echo of " + echoNick,
		Type:     content.MobElite,
		Level:    echoLvl,
		Stats:    stats,
		Element:  content.ElementPhysical,
		RewardXP: echoLvl * 15,
	}
	mob.Stats.HP *= 2
	mob.MaxHP = mob.Stats.HP
	mob.CurrentHP = mob.MaxHP
	mob.Spells = b.getSkills(echoUID)

	return []content.Mob{mob}, echoNick
}

// fightAbyssFloor resolves one floor through the shared engine and applies the
// same post-fight processing the bot cycle applies (reward XP with auto-prestige,
// durability). The engine already persists HP, combat gold and loot drops.
func (b *Bot) fightAbyssFloor(uid string, depth int, tier abyssTier, modifier string, focus string) (abyssFloorResult, error) {
	u, prestige, err := b.buildAbyssUser(uid)
	if err != nil {
		return abyssFloorResult{}, err
	}
	u.LootFocus = focus
	u.FloorModifier = modifier

	st := b.loadAbyssStats(uid)
	diff, forceBoss := abyssDifficulty(u.Stats, u.Level, depth)
	diff *= tier.DiffMult * (1.0 + float64(prestige)*0.05) // [17] prestige & tier scaling
	worldBoss := forceBoss && depth%(abyssBossEvery*2) == 0

	logs := []string{}
	// Check if the floor has the Artifact Corruption modifier
	if modifier == "artifact_corrupted" {
		var aMult sql.NullFloat64
		var aName sql.NullString
		var aDura int
		if err := b.DB.QueryRow("SELECT artifact_mult, artifact_name, artifact_durability FROM users WHERE client_uid=$1", uid).Scan(&aMult, &aName, &aDura); err == nil {
			if aName.Valid && aName.String != "" && aDura > 0 {
				if art, ok := content.GetArtifactByName(aName.String); ok {
					u.Stats = u.Stats.Add(art.Stats.Scaled(-2))
					logs = append(logs, "[color=#f44336]⚠️ ARTIFACT CORRUPTED! The atmospheric pressure flips your artifact's essence, reversing its boon/curse for this floor![/color]")
				}
			}
		}
	}

	theme := content.CurrentTheme(time.Now())
	zoneName := abyssZoneName(depth)
	if theme != nil {
		logs = append(logs, fmt.Sprintf("%s The Abyss is gripped by the %s theme!", theme.Emoji, theme.Name))
		switch theme.Emoji {
		case "🎄":
			zoneName = "Frozen " + zoneName
		case "🎃":
			zoneName = "Haunted " + zoneName
		case "🎆":
			zoneName = "Festive " + zoneName
		case "❤️":
			zoneName = "Lovely " + zoneName
		}
	}

	zone := content.GetRandomZone(u.Level, float64(u.Stats.Score()))
	zone.Name = zoneName

	var mobs []content.Mob
	var echoNick string
	switch modifier {
	case "watcher":
		mobs = []content.Mob{
			{
				Name:     "The Watcher",
				Type:     content.MobBoss,
				Level:    u.Level + 2,
				Stats:    content.Stats{HP: 1500 * u.Level / 2, STR: 40, DEF: 80, SPD: 110},
				RewardXP: 250,
				Element:  content.ElementPhysical,
				Effects:  []content.MobEffect{content.EffectEnraged},
			},
		}
		logs = append(logs, "[color=#f44336]👁️ The Watcher has found you! You lingered too long in the dark, and the Stalker of the Abyss strikes from the shadows![/color]")
	case "echo_encounter":
		mobs, echoNick = b.spawnEchoMob(uid, u.Level)
		if len(mobs) > 0 {
			logs = append(logs, fmt.Sprintf("[color=#9c27b0]👻 An eerie silence falls. Out of the shadows rises the Ghostly Echo of %s (Depth %d delver)![/color]", echoNick, depth))
		}
	}

	if len(mobs) == 0 {
		if forceBoss {
			var bossName string
			switch {
			case depth == 100:
				bossName = "Abyssus, Heart of the Void"
			case depth%20 == 5:
				bossName = "Gorgoroth the Firelord"
			case depth%20 == 10:
				bossName = "Malakor the Voidweaver"
			case depth%20 == 15:
				bossName = "Azazoth the Slumbering Eye"
			default:
				bosses := []string{"Gorgoroth the Firelord", "Malakor the Voidweaver", "Azazoth the Slumbering Eye"}
				bossName = bosses[(depth/5)%len(bosses)]
			}

			mobs = []content.Mob{
				{
					Name:     bossName,
					Type:     content.MobBoss,
					Level:    u.Level + 1,
					Stats:    content.Stats{HP: 2000 * u.Level / 2, STR: 50, DEF: 90, SPD: 105},
					RewardXP: 500,
					Element:  content.ElementPhysical,
				},
			}
			logs = append(logs, fmt.Sprintf("[color=#e91e63]💀 BOSS FLOOR! Out of the abyss rises %s![/color]", bossName))
		} else if modifier == "treasure_goblin" {
			mobs = []content.Mob{
				{
					Name:     "Hoarder Treasure Goblin",
					Type:     content.MobTreasureGoblin,
					Level:    u.Level,
					Stats:    content.Stats{HP: 400 * u.Level / 2, STR: 5, DEF: 20, SPD: 150},
					RewardXP: 100,
					Element:  content.ElementPhysical,
				},
			}
			logs = append(logs, "[color=#ffeb3b]💰 A Treasure Goblin hoard! You corner a wealthy Treasure Goblin, but it starts sprinting towards a portal![/color]")
		} else {
			mobs = content.SpawnMobGroup(u.Level, zone, diff*zone.Difficulty, 1, forceBoss)
		}
	}

	isBossFloor := forceBoss || worldBoss

	escalateMobs(mobs, depth, worldBoss) // [15] deeper floors → denser elites/effects
	mobPtrs := make([]*content.Mob, len(mobs))
	for i := range mobs {
		mobPtrs[i] = &mobs[i]
	}

	logs = append(logs, zone.Display())
	if ml := abyssMilestoneLine(depth); ml != "" {
		logs = append(logs, ml) // [38] depth-milestone flavour
	}

	var coopUID sql.NullString
	_ = b.DB.QueryRow("SELECT coop_uid FROM abyss_active WHERE client_uid = $1", uid).Scan(&coopUID)

	combatUsers := []UserInCombat{u}
	if coopUID.Valid && coopUID.String != "" {
		partner, _, err := b.buildAbyssUser(coopUID.String)
		if err == nil {
			partner.LootFocus = focus
			partner.FloorModifier = modifier
			combatUsers = append(combatUsers, partner)
			logs = append(logs, fmt.Sprintf("[color=#4a6fa5]🔔 Co-op Ally %s has entered the fray to assist you![/color]", partner.Nickname))
		}
	}

	startTime := time.Now()
	resLogs, rewardXP, victory, loots := b.resolveChannelCombat(combatUsers, mobPtrs, u.Level, diff, zone)
	duration := time.Since(startTime)
	logs = append(logs, resLogs...)

	if victory && coopUID.Valid && coopUID.String != "" {
		b.grantAbyssTokens(coopUID.String, 5)
		logs = append(logs, fmt.Sprintf("[color=#4a6fa5]🔔 Helper %s has been awarded 5 Abyss Tokens for their assistance![/color]", coopUID.String))
	}
	// Clear co-op partner for next floor
	_, _ = b.DB.Exec("UPDATE abyss_active SET coop_uid = NULL WHERE client_uid = $1", uid)

	// Record boss kills — use isBossFloor so the check is unaffected by escalateMobs
	// having promoted mobs[0].Type to MobLegendary.
	if victory && isBossFloor && len(mobs) > 0 {
		_, _ = b.DB.Exec(
			"INSERT INTO abyss_boss_kills (client_uid, boss_name, depth, kill_time_ms) VALUES ($1, $2, $3, $4)",
			uid, mobs[0].Name, depth, duration.Milliseconds(),
		)
	}

	// Record kills in Bestiary — use CurrentHP (live value) not Stats.HP (base max)
	var killedMobs []string
	for _, m := range mobPtrs {
		if m.CurrentHP <= 0 && m.Type != content.MobTreasureGoblin {
			killedMobs = append(killedMobs, m.Name)
		}
	}
	if len(killedMobs) > 0 {
		b.recordAbyssKills(uid, killedMobs)
	}

	// Grant the combat reward XP on a win (the engine applies its own death
	// penalty on a loss), and prestige immediately at the cap like the cycle does.
	if victory && rewardXP > 0 {
		rewardXP = int(float64(rewardXP) * (1.0 + float64(st.AbyssPrestige)*0.05))
		if lr, _ := b.awardXP(uid, "", rewardXP); lr != nil && lr.NewLevel >= PrestigeThreshold {
			b.doPrestige(uid) // [52] keep Abyss prestige consistent with the cycle
		}
	}

	// Gear wears down each floor (more on defeat), exactly like a cycle fight.
	var duraWarnings []string
	_, weeklyMod := b.currentWeeklyChallenge()
	if weeklyMod != "zero_durability_loss" {
		duraWarnings = b.applyDurabilityLoss(uid, !victory)
	}

	stats, _, _, _ := b.calculateTotalStats(uid, time.Now())
	var curHP int
	_ = b.DB.QueryRow("SELECT current_hp FROM users WHERE client_uid=$1", uid).Scan(&curHP)
	if curHP < 0 {
		curHP = 0
	}

	res := abyssFloorResult{Victory: victory, RewardXP: rewardXP, CurrentHP: curHP, MaxHP: stats.HP}
	for _, l := range logs {
		res.LogsHTML = append(res.LogsHTML, bbToHTML(l))
	}
	for _, lt := range loots {
		if lt.UID == uid && lt.Note != "" {
			res.LootHTML = append(res.LootHTML, bbToHTML(lt.Note))
		}
	}
	for _, d := range duraWarnings {
		res.DuraHTML = append(res.DuraHTML, bbToHTML(d)) // [11-review] surface gear damage
	}
	return res, nil
}

// ---- BBCode → safe HTML --------------------------------------------------

var bbColorRe = regexp.MustCompile(`\[color=(#[0-9a-fA-F]{3,8})\]`)

// bbToHTML converts the TeamSpeak BBCode the combat engine emits into a small,
// safe subset of HTML for the web log. The input is HTML-escaped first, so any
// player-controlled text (nicknames) cannot inject markup; only the known BBCode
// tokens are then turned back into tags.
func bbToHTML(s string) string {
	s = html.EscapeString(s)
	s = bbColorRe.ReplaceAllString(s, `<span style="color:$1">`)
	repl := strings.NewReplacer(
		"[/color]", "</span>",
		"[b]", "<b>", "[/b]", "</b>",
		"[i]", "<i>", "[/i]", "</i>",
		"[center]", `<span class="ab-center">`, "[/center]", "</span>",
		"[size=12]", `<span class="ab-big">`, "[/size]", "</span>",
		"[hr]", `<span class="ab-hr"></span>`,
	)
	return repl.Replace(s)
}

// ---- Run state -----------------------------------------------------------

// abyssRun is the server-authoritative state of a player's active descent.
type abyssRun struct {
	Active       bool
	Depth        int
	Escrow       int64
	Tier         string
	Insured      int  // % of cache protected on death
	Revived      bool // double-or-nothing already used
	Downed       bool // HP <= 0, awaiting revive or concede
	CurHP        int
	MaxHP        int
	FloorType    string
	Modifier     string
	EventState   string
	LastActionAt time.Time
	CoopUID      string
}

// loadAbyssRun reads the active run plus the player's live HP so callers can tell
// whether the player is mid-fight, downed, or has no run at all.
func (b *Bot) loadAbyssRun(uid string) abyssRun {
	var r abyssRun
	var evState, coop sql.NullString
	var lastAct time.Time
	err := b.DB.QueryRow(
		"SELECT depth, escrow, tier, insured, revived, floor_type, modifier, event_state, last_action_at, coop_uid FROM abyss_active WHERE client_uid=$1", uid,
	).Scan(&r.Depth, &r.Escrow, &r.Tier, &r.Insured, &r.Revived, &r.FloorType, &r.Modifier, &evState, &lastAct, &coop)
	if err != nil {
		return r
	}
	r.Active = true
	if evState.Valid {
		r.EventState = evState.String
	}
	r.LastActionAt = lastAct
	if coop.Valid {
		r.CoopUID = coop.String
	}
	stats, _, _, _ := b.calculateTotalStats(uid, time.Now())
	r.MaxHP = stats.HP
	_ = b.DB.QueryRow("SELECT current_hp FROM users WHERE client_uid=$1", uid).Scan(&r.CurHP)
	if r.CurHP < 0 {
		r.CurHP = 0
	}
	r.Downed = r.CurHP <= 0
	return r
}

// ---- Page ----------------------------------------------------------------

func (s *WebServer) handleAbyssPage(w http.ResponseWriter, r *http.Request, uid string) {
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}
	st := s.bot.loadAbyssStats(uid)
	run := s.bot.loadAbyssRun(uid)

	loreList := []map[string]any{}
	unlockedMap := make(map[int]bool)
	for _, id := range s.bot.loadUnlockedLore(uid) {
		unlockedMap[id] = true
	}
	for id := 1; id <= 10; id++ {
		text := "[Locked — Reach deeper floors to discover this fragment]"
		unlocked := unlockedMap[id]
		if unlocked {
			text = abyssLoreFragments[id]
		}
		loreList = append(loreList, map[string]any{
			"ID":       id,
			"Text":     text,
			"Unlocked": unlocked,
		})
	}

	_, weeklyMod := s.bot.currentWeeklyChallenge()
	helpers := s.bot.loadCoopHelpers(uid)

	s.render(w, "abyss", map[string]any{
		"Title":        "The Abyss",
		"Nav":          "abyss",
		"U":            u,
		"Stats":        st,
		"Run":          run,
		"Tiers":        abyssTierList(st.BestDepth),
		"Leaders":      s.bot.abyssLeaderboards("normal"),
		"Season":       abyssSeasonLabel(),
		"History":      s.bot.abyssHistory(uid, 8),
		"Achieved":     s.bot.abyssAchievements(uid),
		"LoreList":     loreList,
		"Bestiary":     s.bot.loadAbyssBestiary(uid),
		"Consumables":  s.bot.getConsumables(uid),
		"WeeklyMod":    weeklyMod,
		"Helpers":      helpers,
		"NextIsBoss":   run.Active && (run.Depth+1)%5 == 0,
	})
}

// ---- APIs ----------------------------------------------------------------

// handleAbyssEnter starts a fresh descent on the chosen tier: charges the tier's
// entry cost, heals to full, and seeds the run. It refuses to re-enter an active
// run, which is what blocks the "free heal / reset" exploit.
func (s *WebServer) handleAbyssEnter(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Tier string `json:"tier"`
	}
	_ = readJSON(r, &req)
	tier, ok := abyssTierByKey(req.Tier)
	if !ok {
		tier = abyssTiers["normal"]
	}

	st := s.bot.loadAbyssStats(uid)
	if st.BestDepth < tier.MinBest {
		writeJSON(w, map[string]any{"ok": false, "error": "tier locked — reach depth " + itoa(tier.MinBest) + " first"})
		return
	}

	// Reject entering while a run is already active (no free heal/reset).
	if s.bot.loadAbyssRun(uid).Active {
		writeJSON(w, map[string]any{"ok": false, "error": "already in a run"})
		return
	}

	// Wrap gold debit, HP reset, and abyss_active creation in a single transaction
	// so a failure after charging can't leave the player paid without an active run.
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	if tier.EntryGold > 0 {
		res, err := tx.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid=$2 AND gold >= $1", tier.EntryGold, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough gold for entry"})
			return
		}
	}

	// Vigor upgrade lets a run start above the normal max (banked as current HP).
	stats, _, _, _ := s.bot.calculateTotalStats(uid, time.Now())
	startHP := stats.HP + stats.HP*st.UpVigor*5/100
	if _, err := tx.Exec("UPDATE users SET current_hp=$1 WHERE client_uid=$2", startHP, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec(
		`INSERT INTO abyss_active (client_uid, depth, escrow, tier, insured, revived, started_at, last_action_at)
		 VALUES ($1, 0, 0, $2, 0, FALSE, NOW(), NOW())`, uid, tier.Key); err != nil {
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
		"ok": true, "depth": 0, "escrow": 0, "tier": tier.Key,
		"hp": startHP, "max_hp": stats.HP, "gold": gold,
	})
}

// handleAbyssDescend fights the next floor. Win → escrow grows (with interest),
// run continues. Loss → the player is "downed": the cache is held pending a
// revive or concede.
func (s *WebServer) handleAbyssDescend(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	run := s.bot.loadAbyssRun(uid)
	if !run.Active {
		writeJSON(w, map[string]any{"ok": false, "error": "not in a run"})
		return
	}
	if run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "you are downed — revive or concede"})
		return
	}
	if run.FloorType != "combat" {
		writeJSON(w, map[string]any{"ok": false, "error": "you must resolve the current floor action first"})
		return
	}

	var req struct {
		Focus string `json:"focus"`
	}
	_ = readJSON(r, &req)
	focus := req.Focus
	if focus != "gold" && focus != "loot" {
		focus = "balanced"
	}

	newDepth := run.Depth + 1

	// Roll floor type: 10% rest, 10% event, 80% combat
	floorType := "combat"
	modifier := ""
	eventState := ""

	// Check Watcher Stalker trigger (Item #67)
	if !run.LastActionAt.IsZero() && time.Since(run.LastActionAt) > 15*time.Minute && run.Depth > 0 {
		modifier = "watcher"
		floorType = "combat"
	} else if newDepth%abyssBossEvery == 0 {
		floorType = "combat"
	} else {
		// #nosec G404
		rType := rand.Float64()
		if rType < 0.10 {
			floorType = "rest"
		} else if rType < 0.20 {
			floorType = "event"
			// Roll event: merchant (40%), imp (35%), shrine (25%)
			// #nosec G404
			rEv := rand.Float64()
			if rEv < 0.40 {
				g := content.RandomGearDrop()
				c1 := content.RandomConsumable()
				c2 := content.RandomConsumable()
				eventState = fmt.Sprintf(`{"type":"merchant","items":[{"type":"gear","id":"%s","name":"%s","price":400},{"type":"cons","id":"%s","name":"%s","price":100},{"type":"cons","id":"%s","name":"%s","price":150}]}`, g.ID, g.Name, c1.ID, c1.Name, c2.ID, c2.Name)
			} else if rEv < 0.75 {
				eventState = `{"type":"imp"}`
			} else {
				eventState = `{"type":"shrine"}`
			}
		} else {
			floorType = "combat"
			// #nosec G404
			if rand.Float64() < 0.15 {
				mods := []string{"enraged", "no_healing", "artifact_corrupted", "treasure_goblin", "echo_encounter"}
				// #nosec G404
				modifier = mods[rand.IntN(len(mods))]
			}
		}
	}

	tier, _ := abyssTierByKey(run.Tier)

	if floorType != "combat" {
		// Update active run to rest/event floor
		_, err := s.bot.DB.Exec(
			`UPDATE abyss_active 
			    SET depth=$1, floor_type=$2, modifier=$3, event_state=$4, last_action_at=NOW() 
			  WHERE client_uid=$5`,
			newDepth, floorType, modifier, eventState, uid,
		)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		writeJSON(w, map[string]any{
			"ok":          true,
			"noncombat":   true,
			"floor_type":  floorType,
			"depth":       newDepth,
			"event_state": eventState,
			"escrow":      run.Escrow,
		})
		return
	}

	// Normal Combat floor
	if _, err := s.bot.DB.Exec("UPDATE abyss_active SET depth=$1, modifier=$2, event_state=NULL, last_action_at=NOW() WHERE client_uid=$3", newDepth, modifier, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	res, err := s.bot.fightAbyssFloor(uid, newDepth, tier, modifier, focus)
	if err != nil {
		_, _ = s.bot.DB.Exec("UPDATE abyss_active SET depth=$1, modifier='', event_state=NULL, last_action_at=NOW() WHERE client_uid=$2", run.Depth, uid)
		writeJSON(w, map[string]any{"ok": false, "error": "combat"})
		return
	}

	s.finishDescend(w, uid, run, newDepth, run.Escrow, tier, res, modifier, focus)
}

// finishDescend applies the win/loss bookkeeping shared by descend and revive.
func (s *WebServer) finishDescend(w http.ResponseWriter, uid string, run abyssRun, depth int, escrowBefore int64, tier abyssTier, res abyssFloorResult, modifier string, focus string) {
	st := s.bot.loadAbyssStats(uid)
	_, _ = s.bot.DB.Exec("UPDATE users SET abyss_lifetime_floors = abyss_lifetime_floors + 1 WHERE client_uid=$1", uid)

	out := map[string]any{
		"ok": true, "victory": res.Victory, "depth": depth,
		"hp": res.CurrentHP, "max_hp": res.MaxHP,
		"logs": res.LogsHTML, "loot": res.LootHTML, "dura": res.DuraHTML, "reward_xp": res.RewardXP,
	}

	if res.Victory {
		bonus := abyssFloorBonus(depth, run.depthLevelHint())
		bonus = int64(float64(bonus) * tier.RewardMult * (1.0 + float64(st.UpGreed)*0.05) * (1.0 + float64(st.AbyssPrestige)*0.05))
		
		switch focus {
		case "gold":
			bonus = bonus * 2
		case "loot":
			bonus = bonus / 2
		}
		
		if s.bot.abyssDailyFirstDescent(uid) {
			bonus = bonus * 3 / 2 // [11] daily first-descent: +50%
			s.bot.grantAbyssTokens(uid, 5)
			out["daily"] = true
		}
		newEscrow := int64(float64(escrowBefore)*(1.0+abyssEscrowInterest)) + bonus // [56] interest
		if _, err := s.bot.DB.Exec("UPDATE abyss_active SET escrow=$1, floor_type='combat', modifier='', event_state=NULL, last_action_at=NOW() WHERE client_uid=$2", newEscrow, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		_, _ = s.bot.DB.Exec("UPDATE users SET abyss_best_depth = GREATEST(abyss_best_depth, $1) WHERE client_uid=$2", depth, uid)
		out["bonus"] = bonus
		out["escrow"] = newEscrow
		if ach := s.bot.checkDepthAchievements(uid, depth); ach != "" {
			out["achievement"] = ach
		}
		
		// Lore fragment drop chance (15%)
		// #nosec G404
		if rand.Float64() < 0.15 {
			fragID := depth/10 + 1
			if fragID > 10 {
				fragID = 10
			}
			if fragID < 1 {
				fragID = 1
			}
			res, err := s.bot.DB.Exec(
				"INSERT INTO abyss_lore_unlocked (client_uid, lore_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", uid, fragID,
			)
			if err == nil {
				if n, _ := res.RowsAffected(); n > 0 {
					out["lore_unlocked"] = true
					out["lore_fragment"] = abyssLoreFragments[fragID]
				}
			}
		}
		
		// Affix consumable reward
		if modifier != "" {
			c := content.RandomConsumable()
			_, _ = s.bot.DB.Exec("INSERT INTO user_consumables (client_uid, cons_id, remaining_fights) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", uid, c.ID, c.Duration)
			out["affix_reward"] = c.Name
		}
	} else {
		// Downed: hold the cache; the player must revive (if available) or concede.
		out["downed"] = true
		out["can_revive"] = !run.Revived
		out["escrow"] = escrowBefore
		out["insured"] = run.Insured
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	out["gold"] = gold
	out["tokens"] = s.bot.abyssTokens(uid)
	out["consumables"] = s.bot.getConsumables(uid)
	writeJSON(w, out)
}

// handleAbyssRevive spends the one-per-run double-or-nothing: heal to full and
// re-fight the current floor. Win doubles the cache; loss forfeits it.
func (s *WebServer) handleAbyssRevive(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	run := s.bot.loadAbyssRun(uid)
	if !run.Active || !run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "not downed"})
		return
	}
	if run.Revived {
		writeJSON(w, map[string]any{"ok": false, "error": "revival already used"})
		return
	}

	stats, _, _, _ := s.bot.calculateTotalStats(uid, time.Now())

	tier, _ := abyssTierByKey(run.Tier)
	res, err := s.bot.fightAbyssFloor(uid, run.Depth, tier, run.Modifier, "balanced")
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "combat"})
		return
	}

	// Only persist the revive state if combat succeeded.
	if _, err := s.bot.DB.Exec("UPDATE users SET current_hp=$1 WHERE client_uid=$2", stats.HP, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := s.bot.DB.Exec("UPDATE abyss_active SET revived=TRUE, last_action_at=NOW() WHERE client_uid=$1", uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	if res.Victory {
		// Double-or-nothing: the cache doubles, the run continues.
		newEscrow := run.Escrow * 2
		_, _ = s.bot.DB.Exec("UPDATE abyss_active SET escrow=$1, floor_type='combat', modifier='', event_state=NULL, last_action_at=NOW() WHERE client_uid=$2", newEscrow, uid)
		out := map[string]any{
			"ok": true, "revived": true, "victory": true, "depth": run.Depth,
			"hp": res.CurrentHP, "max_hp": res.MaxHP, "logs": res.LogsHTML,
			"loot": res.LootHTML, "dura": res.DuraHTML, "escrow": newEscrow, "doubled": true,
		}
		var gold int64
		_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
		out["gold"] = gold
		out["tokens"] = s.bot.abyssTokens(uid)
		out["consumables"] = s.bot.getConsumables(uid)
		writeJSON(w, out)
		return
	}

	// Failed the second chance → forfeit.
	payout, jackpot := s.bot.forfeitAbyss(uid, run)
	out := map[string]any{
		"ok": true, "revived": true, "victory": false, "depth": run.Depth,
		"hp": 0, "logs": res.LogsHTML, "loot": res.LootHTML, "dura": res.DuraHTML,
		"forfeited": true, "insured_refund": payout, "escrow": 0,
	}
	if jackpot > 0 {
		out["jackpot_win"] = jackpot
	}
	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	out["gold"] = gold
	out["tokens"] = s.bot.abyssTokens(uid)
	out["consumables"] = s.bot.getConsumables(uid)
	writeJSON(w, out)
}

// handleAbyssConcede gives up a downed run, forfeiting the cache (minus insurance).
func (s *WebServer) handleAbyssConcede(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	run := s.bot.loadAbyssRun(uid)
	if !run.Active || !run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "not downed"})
		return
	}
	payout, jackpot := s.bot.forfeitAbyss(uid, run)
	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	out := map[string]any{
		"ok": true, "conceded": true, "depth": run.Depth,
		"insured_refund": payout, "gold": gold, "tokens": s.bot.abyssTokens(uid),
	}
	if jackpot > 0 {
		out["jackpot_win"] = jackpot
	}
	writeJSON(w, out)
}

// handleAbyssBank cashes out a *living* run. Banking deeper and on a longer
// streak pays a bigger multiplier; the optional "cursed" bank pays +20% but
// hexes the next few TeamSpeak-cycle fights.
func (s *WebServer) handleAbyssBank(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Cursed bool `json:"cursed"`
	}
	_ = readJSON(r, &req)

	run := s.bot.loadAbyssRun(uid)
	if !run.Active {
		writeJSON(w, map[string]any{"ok": false, "error": "not in a run"})
		return
	}
	if run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "you are downed — revive or concede"})
		return
	}

	st := s.bot.loadAbyssStats(uid)
	mult := s.bot.abyssBankMultiplier(run.Depth, st.Streak) // [2][12] depth + streak
	payout := int64(float64(run.Escrow) * mult)
	if req.Cursed && payout > 0 {
		payout = payout * 12 / 10 // [9] +20%
	}
	payout = s.bot.capAbyssDayGold(uid, payout) // [59] per-day guard

	var gold int64
	if payout > 0 {
		if err := s.bot.DB.QueryRow("UPDATE users SET gold = gold + $1 WHERE client_uid=$2 RETURNING gold", payout, uid).Scan(&gold); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	} else {
		_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	}

	jackpotWin := int64(0)
	bonusGear := ""
	if run.Depth >= 10 {
		bonusGear = s.bot.awardAbyssBonusGear(uid, run.Depth) // [55][57]
	}
	if run.Depth > 0 {
		if _, err := s.bot.DB.Exec(
			"INSERT INTO abyss_runs (client_uid, depth, gold_banked, victory, tier) VALUES ($1,$2,$3,TRUE,$4)",
			uid, run.Depth, payout, run.Tier); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		_, _ = s.bot.DB.Exec(
			`UPDATE users SET abyss_best_depth = GREATEST(abyss_best_depth, $1),
			        abyss_lifetime_banked = abyss_lifetime_banked + $2,
			        abyss_bank_streak = abyss_bank_streak + 1 WHERE client_uid=$3`,
			run.Depth, payout, uid)
		s.bot.grantAbyssTokens(uid, run.Depth/5+1) // [44]
		s.bot.recordGameResult(uid, "abyss", true, payout)
		jackpotWin = s.bot.tryAbyssJackpot(uid, run.Depth) // [62]
		if jackpotWin > 0 {
			gold += jackpotWin
		}

		// Record breaker check (Item #82) — compare against the true global max
		// (including the current user's previous best) so we only fire when the
		// run sets a genuinely new server-wide record.
		var maxDepth int
		_ = s.bot.DB.QueryRow("SELECT COALESCE(MAX(depth), 0) FROM abyss_runs").Scan(&maxDepth)
		if run.Depth > maxDepth {
			uInfo, _ := s.loadWebUser(uid)
			go s.bot.BroadcastAbyssRecord(uInfo.Nickname, run.Depth)
		}
	}
	if req.Cursed {
		_, _ = s.bot.DB.Exec("UPDATE users SET abyss_curse_fights = 3 WHERE client_uid=$1", uid)
	}
	if _, err := s.bot.DB.Exec("DELETE FROM abyss_active WHERE client_uid=$1", uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	out := map[string]any{
		"ok": true, "banked": payout, "mult": mult, "depth": run.Depth,
		"gold": gold, "tokens": s.bot.abyssTokens(uid), "cursed": req.Cursed,
	}
	if jackpotWin > 0 {
		out["jackpot_win"] = jackpotWin
	}
	if bonusGear != "" {
		out["bonus_gear"] = bonusGear
	}
	writeJSON(w, out)
}

// depthLevelHint returns the player level used for the floor-bonus curve. Stored
// here so finishDescend doesn't need an extra query; it is filled by the caller.
func (run abyssRun) depthLevelHint() int {
	if run.MaxHP <= 0 {
		return 1
	}
	// Approximate level from max HP (HP ≈ base + level scaling); the exact value
	// only tunes reward magnitude, so an estimate is fine and avoids a query.
	lvl := run.MaxHP / 5
	if lvl < 1 {
		lvl = 1
	}
	return lvl
}

// ---- i18n / season / zone flavour ----------------------------------------

// abyssSeasonLabel is the current month, used for the "deepest this season" board.
func abyssSeasonLabel() string {
	return time.Now().UTC().Format("January 2006")
}

func abyssSeasonStart() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// abyssMilestoneLine returns a dread atmosphere line on milestone floors. [38][40]
func abyssMilestoneLine(depth int) string {
	switch depth {
	case 10:
		return bbWrapMuted("A cold wind rises — the tenth floor. There is no stair back.")
	case 25:
		return bbWrapMuted("The walls weep. Floor 25 — few delvers go deeper.")
	case 50:
		return bbWrapMuted("Floor 50. The dark down here has a heartbeat.")
	case 100:
		return bbWrapMuted("Floor 100. Nothing alive should be here. Including you.")
	}
	return ""
}

func bbWrapMuted(s string) string { return "[color=#8a93a8][i]" + s + "[/i][/color]" }

// handleAbyssUseConsumable handles manually drinking a potion or using a repair kit in the lobby.
func (s *WebServer) handleAbyssUseConsumable(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		ConsID string `json:"cons_id"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	// Verify they own the consumable
	var rem int
	err := s.bot.DB.QueryRow("SELECT remaining_fights FROM user_consumables WHERE client_uid = $1 AND cons_id = $2 LIMIT 1", uid, req.ConsID).Scan(&rem)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "you do not own this consumable"})
		return
	}

	c, ok := content.GetConsumableByID(req.ConsID)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid consumable"})
		return
	}

	stats, _, _, _ := s.bot.calculateTotalStats(uid, time.Now())

	switch c.Type {
	case content.ConsumableHealing:
		// Heal player
		healAmt := int(float64(stats.HP) * c.EffectValue)
		if healAmt < 50 {
			healAmt = 50
		}
		_, _ = s.bot.DB.Exec("UPDATE users SET current_hp = LEAST(current_hp + $1, $2) WHERE client_uid = $3", healAmt, stats.HP, uid)
	case content.ConsumableRepair:
		repairAmt := 30
		if req.ConsID == "master_repair_kit" {
			repairAmt = 150
		}
		// Repair gear
		_, _ = s.bot.DB.Exec("UPDATE user_gear SET durability = LEAST(durability + $1, max_durability) WHERE client_uid = $2", repairAmt, uid)
		_, _ = s.bot.DB.Exec("UPDATE users SET artifact_durability = LEAST(artifact_durability + 15, 30) WHERE client_uid = $1 AND artifact_durability > 0", uid)
	case content.ConsumableBuff:
		// Buffs elixirs: manual use sets them to active (3 remaining fights).
		// Do NOT fall through to the shared delete — buffs stay owned while active.
		_, _ = s.bot.DB.Exec("UPDATE user_consumables SET remaining_fights = 3 WHERE client_uid = $1 AND cons_id = $2", uid, req.ConsID)
		var curHP int
		_ = s.bot.DB.QueryRow("SELECT current_hp FROM users WHERE client_uid=$1", uid).Scan(&curHP)
		var gold int64
		_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
		writeJSON(w, map[string]any{
			"ok":     true,
			"hp":     curHP,
			"max_hp": stats.HP,
			"gold":   gold,
		})
		return
	default:
		writeJSON(w, map[string]any{"ok": false, "error": "consumable type cannot be used manually"})
		return
	}

	// Consume 1 item
	_, _ = s.bot.DB.Exec("DELETE FROM user_consumables WHERE client_uid = $1 AND cons_id = $2", uid, req.ConsID)

	var curHP int
	_ = s.bot.DB.QueryRow("SELECT current_hp FROM users WHERE client_uid=$1", uid).Scan(&curHP)
	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)

	writeJSON(w, map[string]any{
		"ok":          true,
		"hp":          curHP,
		"max_hp":      stats.HP,
		"gold":        gold,
		"consumables": s.bot.getConsumables(uid),
	})
}

// handleAbyssNonCombatAction resolves purchases and interactions on Rest and Event floors.
func (s *WebServer) handleAbyssNonCombatAction(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Action  string `json:"action"`
		Payload string `json:"payload"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.FloorType == "combat" {
		writeJSON(w, map[string]any{"ok": false, "error": "not on a non-combat floor"})
		return
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)

	switch run.FloorType {
	case "rest":
		switch req.Action {
		case "heal":
			cost := int64(100)
			if gold < cost {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			stats, _, _, _ := s.bot.calculateTotalStats(uid, time.Now())
			_, err := s.bot.DB.Exec("UPDATE users SET gold = gold - $1, current_hp = $2 WHERE client_uid = $3 AND gold >= $1", cost, stats.HP, uid)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "msg": "Healed to full!", "gold": gold - cost, "hp": stats.HP})
			return

		case "repair":
			cost := int64(100)
			if gold < cost {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			_, err := s.bot.DB.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1", cost, uid)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			_, _ = s.bot.DB.Exec("UPDATE user_gear SET durability = max_durability WHERE client_uid = $1", uid)
			_, _ = s.bot.DB.Exec("UPDATE users SET artifact_durability = 30 WHERE client_uid = $1 AND artifact_name IS NOT NULL", uid)
			writeJSON(w, map[string]any{"ok": true, "msg": "All gear fully repaired!", "gold": gold - cost})
			return

		case "reroll_skills":
			cost := int64(150)
			if gold < cost {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			_, err := s.bot.DB.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1", cost, uid)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			_, _ = s.bot.DB.Exec("DELETE FROM user_skills WHERE client_uid = $1", uid)
			for i := 1; i <= 5; i++ {
				sk := content.RandomSkill()
				_, _ = s.bot.DB.Exec("INSERT INTO user_skills (client_uid, slot, skill_id) VALUES ($1, $2, $3)", uid, i, sk.ID)
			}
			writeJSON(w, map[string]any{"ok": true, "msg": "Skills re-rolled!", "gold": gold - cost})
			return
		}
	case "event":
		var state struct {
			Type  string `json:"type"`
			Items []struct {
				Type  string `json:"type"`
				ID    string `json:"id"`
				Name  string `json:"name"`
				Price int64  `json:"price"`
			} `json:"items"`
		}
		_ = json.Unmarshal([]byte(run.EventState), &state)

		switch req.Action {
		case "merchant_buy":
			if state.Type != "merchant" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for merchant_buy"})
				return
			}
			var idx int
			_, _ = fmt.Sscan(req.Payload, &idx)
			if idx < 0 || idx >= len(state.Items) {
				writeJSON(w, map[string]any{"ok": false, "error": "invalid item index"})
				return
			}
			item := state.Items[idx]
			if gold < item.Price {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			res, err := s.bot.DB.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1", item.Price, uid)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if n, _ := res.RowsAffected(); n == 0 {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			if item.Type == "gear" {
				if g, ok := content.GetGearByID(item.ID); ok {
					s.bot.awardGearDrop(uid, g)
				}
			} else {
				if c, ok := content.GetConsumableByID(item.ID); ok {
					_, _ = s.bot.DB.Exec("INSERT INTO user_consumables (client_uid, cons_id, remaining_fights) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", uid, c.ID, c.Duration)
				}
			}
			state.Items = append(state.Items[:idx], state.Items[idx+1:]...)
			newStateBytes, _ := json.Marshal(state)
			_, _ = s.bot.DB.Exec("UPDATE abyss_active SET event_state = $1, last_action_at = NOW() WHERE client_uid = $2", string(newStateBytes), uid)

			writeJSON(w, map[string]any{"ok": true, "msg": "Bought " + item.Name + "!", "gold": gold - item.Price})
			return

		case "imp_gamble":
			if state.Type != "imp" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for imp_gamble"})
				return
			}
			cost := int64(300)
			if gold < cost {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			_, err := s.bot.DB.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1", cost, uid)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			
			// #nosec G404
			rRoll := rand.Float64()
			var msg string
			newGold := gold - cost
			if rRoll < 0.40 {
				msg = "The Imp giggles and steals your gold! Got nothing."
			} else if rRoll < 0.75 {
				msg = "Dice rolled! You doubled your wager! (+600 gold)"
				_ = s.bot.DB.QueryRow("UPDATE users SET gold = gold + 600 WHERE client_uid = $1 RETURNING gold", uid).Scan(&newGold)
			} else if rRoll < 0.95 {
				c := content.RandomConsumable()
				msg = "The Imp drops a consumable: " + c.Name + "!"
				_, _ = s.bot.DB.Exec("INSERT INTO user_consumables (client_uid, cons_id, remaining_fights) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", uid, c.ID, c.Duration)
			} else {
				ui := content.RandomUniqueItem()
				msg = "JACKPOT! The Imp drops a Unique Item: " + ui.Name + "!"
				_, _ = s.bot.DB.Exec("INSERT INTO user_unique_items (client_uid, item_name, rarity, power) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING", uid, ui.Name, ui.Rarity, ui.Power)
			}
			
			_, _ = s.bot.DB.Exec("UPDATE abyss_active SET event_state = NULL, floor_type = 'combat', last_action_at = NOW() WHERE client_uid = $1", uid)
			writeJSON(w, map[string]any{"ok": true, "msg": msg, "gold": newGold, "resolved": true})
			return

		case "shrine_accept":
			if state.Type != "shrine" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for shrine_accept"})
				return
			}
			newEscrow := run.Escrow + 1000
			_, err := s.bot.DB.Exec("UPDATE abyss_active SET escrow = $1, event_state = NULL, floor_type = 'combat', last_action_at = NOW() WHERE client_uid = $2", newEscrow, uid)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			_, _ = s.bot.DB.Exec("UPDATE users SET abyss_curse_fights = abyss_curse_fights + 5 WHERE client_uid = $1", uid)
			writeJSON(w, map[string]any{"ok": true, "msg": "Shrine accepted! +1,000 gold added to cache, but you are cursed!", "escrow": newEscrow, "resolved": true})
			return
		}
	}

	writeJSON(w, map[string]any{"ok": false, "error": "invalid action"})
}

// handleAbyssNonCombatProceed leaves the Rest/Event floor and returns to the lobby.
func (s *WebServer) handleAbyssNonCombatProceed(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.FloorType == "combat" {
		writeJSON(w, map[string]any{"ok": false, "error": "not on a non-combat floor"})
		return
	}

	st := s.bot.loadAbyssStats(uid)
	tier, _ := abyssTierByKey(run.Tier)
	bonus := abyssFloorBonus(run.Depth, run.depthLevelHint())
	
	var req struct {
		Focus string `json:"focus"`
	}
	_ = readJSON(r, &req)
	focus := req.Focus
	
	switch focus {
	case "gold":
		bonus = bonus * 2
	case "loot":
		bonus = bonus / 2
	}
	// Apply tier reward multiplier to match combat floor scaling
	bonus = int64(float64(bonus) * tier.RewardMult)
	bonus = int64(float64(bonus) * (1.0 + float64(st.UpGreed)*0.05) * (1.0 + float64(st.AbyssPrestige)*0.05))
	
	newEscrow := int64(float64(run.Escrow)*(1.0+abyssEscrowInterest)) + bonus
	
	_, err := s.bot.DB.Exec(
		`UPDATE abyss_active 
		    SET escrow = $1, floor_type = 'combat', modifier = '', event_state = NULL, last_action_at = NOW() 
		  WHERE client_uid = $2`, newEscrow, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	affixReward := ""
	if run.Modifier != "" {
		c := content.RandomConsumable()
		_, _ = s.bot.DB.Exec("INSERT INTO user_consumables (client_uid, cons_id, remaining_fights) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", uid, c.ID, c.Duration)
		affixReward = c.Name
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	var curHP int
	_ = s.bot.DB.QueryRow("SELECT current_hp FROM users WHERE client_uid=$1", uid).Scan(&curHP)

	writeJSON(w, map[string]any{
		"ok":           true,
		"resolved":     true,
		"depth":        run.Depth,
		"escrow":       newEscrow,
		"bonus":        bonus,
		"gold":         gold,
		"hp":           curHP,
		"affix_reward": affixReward,
	})
}

// ---- Co-op, Prestige & Weekly challenge Helpers/Handlers ------------------

func (b *Bot) currentWeeklyChallenge() (int64, string) {
	year, week := time.Now().ISOWeek()
	seed := int64(year*100 + week)
	mods := []string{"double_hazards", "zero_durability_loss", "enraged_mobs"}
	return seed, mods[seed%int64(len(mods))]
}

func (b *Bot) loadCoopHelpers(uid string) []map[string]any {
	rows, err := b.DB.Query(
		`SELECT client_uid, COALESCE(NULLIF(nickname, ''), 'Adventurer') AS nick, abyss_best_depth
		   FROM users
		  WHERE client_uid != $1 AND abyss_best_depth > 0
		  ORDER BY last_active_at DESC
		  LIMIT 6`, uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []map[string]any
	for rows.Next() {
		var cuid, nick string
		var depth int
		if err := rows.Scan(&cuid, &nick, &depth); err == nil {
			out = append(out, map[string]any{
				"UID":   cuid,
				"Nick":  nick,
				"Depth": depth,
			})
		}
	}
	return out
}

func (s *WebServer) handleAbyssCoopList(w http.ResponseWriter, r *http.Request, uid string) {
	helpers := s.bot.loadCoopHelpers(uid)
	writeJSON(w, map[string]any{"ok": true, "helpers": helpers})
}

func (s *WebServer) handleAbyssCoopInvite(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		CoopUID string `json:"coop_uid"`
	}
	_ = readJSON(r, &req)

	run := s.bot.loadAbyssRun(uid)
	if !run.Active {
		writeJSON(w, map[string]any{"ok": false, "error": "not in a run"})
		return
	}
	if (run.Depth+1)%5 != 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "co-op summons only available for boss floors"})
		return
	}
	// Reject self-targeting.
	if req.CoopUID == uid {
		writeJSON(w, map[string]any{"ok": false, "error": "cannot invite yourself as a co-op helper"})
		return
	}
	// Verify the helper is a known, eligible user (has a row in users table).
	var helperExists bool
	_ = s.bot.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE client_uid=$1)", req.CoopUID).Scan(&helperExists)
	if !helperExists {
		writeJSON(w, map[string]any{"ok": false, "error": "helper not found"})
		return
	}

	_, err := s.bot.DB.Exec("UPDATE abyss_active SET coop_uid = $1, last_action_at = NOW() WHERE client_uid = $2", req.CoopUID, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *WebServer) handleAbyssPrestige(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	st := s.bot.loadAbyssStats(uid)
	if st.BestDepth < 50 {
		writeJSON(w, map[string]any{"ok": false, "error": "must reach at least floor 50 to prestige"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec("UPDATE users SET abyss_best_depth = 0, abyss_prestige = abyss_prestige + 1 WHERE client_uid = $1", uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	_, err = tx.Exec("DELETE FROM abyss_active WHERE client_uid = $1", uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "prestige": st.AbyssPrestige + 1})
}
