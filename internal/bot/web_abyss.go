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
	// abyssBaseDiff is the floor-1 difficulty. The Abyss is depth-driven: floor 1
	// is gentle for everyone and danger comes from how deep you push, NOT from how
	// much gear you carry. (Gear instead lets you survive deeper — that is the
	// progression.)
	abyssBaseDiff = 0.6
	// abyssDepthRamp adds this much difficulty per floor beyond the first.
	abyssDepthRamp = 0.1
	// abyssDiffSoftCap is where difficulty growth switches from linear to a gentle
	// logarithmic crawl, so very deep floors stay computationally bounded while
	// never quite flattening.
	abyssDiffSoftCap = 6.0
	// abyssMobLevelBase / abyssMobLevelRamp decouple Abyss mob level from the
	// player's exact level. Floor 1 spawns mobs at abyssMobLevelBase × the player's
	// level (well below them, so a fairly-geared delver can win), ramping toward and
	// past parity as depth grows. This is what makes DEPTH the source of danger.
	abyssMobLevelBase = 0.3
	abyssMobLevelRamp = 0.025
	// abyssMobDamageMult dampens how hard Abyss mobs hit so fights last more rounds
	// and play out tactically rather than ending in a single opening volley.
	abyssMobDamageMult = 0.6
	// abyssBossEvery forces a boss on every Nth floor; every 2nd of those (every
	// 10th floor) is a world-boss tier.
	abyssBossEvery = 5
	// abyssEscrowInterest is added to the standing cache each floor before the new
	// floor bonus, rewarding players who let it ride.
	abyssEscrowInterest = 0.005
	// abyssDayGoldCap bounds Abyss-sourced bank payouts per player per day to
	// protect the shared economy from runaway farming.
	abyssDayGoldCap = 5_000_000
	// abyssJackpotDepth is the minimum bank depth that can hit the deep-cache pot.
	abyssJackpotDepth = 25
)

// abyssEffectiveInterest returns the per-floor escrow interest rate including the
// Compounding (interest) Deep-Delver node, which adds 0.1% per level on top of the
// base let-it-ride rate.
func abyssEffectiveInterest(interestLevel int, hasLuckyCoin bool) float64 {
	rate := abyssEscrowInterest + float64(interestLevel)*0.001
	if hasLuckyCoin {
		rate *= 1.20
	}
	return rate
}

// softCap returns x unchanged up to cap, then grows logarithmically past it.
func softCap(x, cap float64) float64 {
	if x <= cap {
		return x
	}
	return cap + math.Log(1+(x-cap))
}

// abyssFloorBonusMaxPer caps the per-floor base so extremely high-level characters
// don't get runaway payouts. At the cap a full Normal descent to floor 40 accrues
// roughly 100k of cache (before tier, node, affix and pact multipliers); the cap is
// reached around level 700 and everything below scales gently with level.
const abyssFloorBonusMaxPer = 110

// abyssFloorBonus is the base escrowed gold for clearing the given floor (before
// tier and Deep-Delver multipliers). It scales with depth and level so the
// accumulated cache grows roughly quadratically with how deep you push, then flattens
// once the per-floor base hits abyssFloorBonusMaxPer.
func abyssFloorBonus(depth, level int) int64 {
	per := int64(40 + level/10)
	if per < 40 {
		per = 40
	}
	if per > abyssFloorBonusMaxPer {
		per = abyssFloorBonusMaxPer
	}
	return per * int64(depth)
}

// abyssDifficulty derives the base floor difficulty (pre-tier, pre-prestige) and
// whether a boss is forced, purely from depth. The caller layers tier and prestige
// multipliers on top.
//
// Difficulty is deliberately NOT scaled by the player's gear score: doing so made
// floor 1 instantly lethal for well-geared characters (more gear → harder floor,
// neutralising the gear). Instead the floor ramps with depth alone, so a stronger
// character simply banks deeper before the danger overtakes them.
func abyssDifficulty(depth int) (float64, bool) {
	if depth < 1 {
		depth = 1
	}
	base := abyssBaseDiff + float64(depth-1)*abyssDepthRamp
	return softCap(base, abyssDiffSoftCap), depth%abyssBossEvery == 0
}

// abyssMobLevel returns the level Abyss mobs spawn at for a given depth, decoupled
// from the player's exact level. Floor 1 is well below the delver; depth ramps it
// toward and past parity, capped so deep floors stay computationally bounded.
func abyssMobLevel(depth, playerLevel int) int {
	if depth < 1 {
		depth = 1
	}
	if playerLevel < 1 {
		playerLevel = 1
	}
	effLevel := float64(playerLevel)
	if effLevel > 300.0 {
		effLevel = 300.0
	}
	frac := abyssMobLevelBase + abyssMobLevelRamp*float64(depth-1)
	lvl := int(effLevel * frac)
	if lvl < 1 {
		lvl = 1
	}
	if ceil := playerLevel * 2; lvl > ceil {
		lvl = ceil
	}
	return lvl
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
		// Abyss drops are escrowed for the run, not granted inline by the engine.
		EscrowLoot: true,
	}, prestige, nil
}

func nullStr(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}

// gearMaxDurExpr is a SQL expression for a user_gear row's maximum durability.
// user_gear has no max_durability column, so it is read from the persisted
// item_data (Gear.MaxDurability, which has no JSON tag → key "MaxDurability"),
// falling back to the row's current durability when item_data is absent. The
// GREATEST guard guarantees a repair can never *reduce* durability: legacy gear
// without item_data is simply left unchanged instead of erroring on the missing
// column (which previously broke the Fountain event and every repair path).
const gearMaxDurExpr = `GREATEST(durability, COALESCE(NULLIF(item_data->>'MaxDurability','')::int, durability))`

// ensureGearMaxDurability backfills item_data for a user's gear rows that have
// none — legacy rows predating migration 0054, plus rows created by the base-gear
// grant paths (xp.go / auction.go) which still insert without item_data. For such
// rows gearMaxDurExpr has no MaxDurability to read and collapses to the row's
// current durability, so a "full repair" silently no-ops and the proactive
// brokenCount check reports nothing broken. The true max is taken from the static
// catalog (content.GetGearByID); procedural gear whose id is not in the catalog is
// left untouched (its rolled max was never persisted and cannot be recovered).
// Call this before any repair/broken-check so both operate on real max durability.
func (b *Bot) ensureGearMaxDurability(uid string) {
	rows, err := b.DB.Query(
		"SELECT slot, gear_id FROM user_gear WHERE client_uid = $1 AND item_data IS NULL", uid)
	if err != nil {
		return
	}
	type legacyRow struct{ slot, gearID string }
	var pending []legacyRow
	for rows.Next() {
		var r legacyRow
		if err := rows.Scan(&r.slot, &r.gearID); err == nil {
			pending = append(pending, r)
		}
	}
	_ = rows.Close()
	for _, r := range pending {
		g, ok := content.GetGearByID(r.gearID)
		if !ok || g.MaxDurability <= 0 {
			continue
		}
		data, err := json.Marshal(g)
		if err != nil {
			continue
		}
		_, _ = b.DB.Exec(
			"UPDATE user_gear SET item_data = $1 WHERE client_uid = $2 AND slot = $3 AND item_data IS NULL",
			string(data), uid, r.slot)
	}
}

// grantConsumable adds a consumable to a player's stash. If they already hold the
// same consumable its remaining_fights is increased, rather than the grant being
// silently dropped — the old `ON CONFLICT DO NOTHING` lost paid purchases (gold
// spent, nothing granted).
func (b *Bot) grantConsumable(uid, consID string, fights int) {
	if fights <= 0 {
		fights = 1
	}
	_, _ = b.DB.Exec(
		`INSERT INTO user_consumables (client_uid, cons_id, remaining_fights)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (client_uid, cons_id)
		 DO UPDATE SET remaining_fights = user_consumables.remaining_fights + EXCLUDED.remaining_fights`,
		uid, consID, fights)
}

// consumableOwned is one owned consumable stack, for the Abyss carry-cap picker.
type consumableOwned struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// abyssOwnedConsumables lists the player's consumable stacks (id, display name,
// charge count) and the total charge count, used by the carry-cap loadout picker.
func (b *Bot) abyssOwnedConsumables(uid string) ([]consumableOwned, int) {
	rows, err := b.DB.Query("SELECT cons_id, remaining_fights FROM user_consumables WHERE client_uid=$1 AND remaining_fights > 0 ORDER BY cons_id", uid)
	if err != nil {
		return nil, 0
	}
	defer func() { _ = rows.Close() }()
	var out []consumableOwned
	total := 0
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			continue
		}
		name := id
		if c, ok := content.GetConsumableByID(id); ok {
			name = c.Name
		}
		out = append(out, consumableOwned{ID: id, Name: name, Count: n})
		total += n
	}
	return out, total
}

// abyssBuildConsumableLoadout validates a player-picked loadout against what they
// own and the carry cap, returning the sanitized {cons_id: count} map (dropping
// zero entries) or a non-empty error message the picker should surface.
func abyssBuildConsumableLoadout(picked map[string]int, owned []consumableOwned, max int) (map[string]int, string) {
	ownedMap := make(map[string]int, len(owned))
	for _, o := range owned {
		ownedMap[o.ID] = o.Count
	}
	out := make(map[string]int)
	sum := 0
	for id, cnt := range picked {
		if cnt <= 0 {
			continue
		}
		have, ok := ownedMap[id]
		if !ok {
			return nil, "You don't own one of the selected consumables."
		}
		if cnt > have {
			return nil, "You selected more than you own."
		}
		out[id] = cnt
		sum += cnt
	}
	if sum < 1 {
		return nil, "Pick at least one consumable to bring."
	}
	if sum > max {
		return nil, fmt.Sprintf("You can bring at most %d (you picked %d).", max, sum)
	}
	return out, ""
}

// abyssRunLoadout returns the active run's consumable loadout and whether one is in
// force. No row or a NULL column means the run is unrestricted (entered under the
// cap), so every owned consumable is usable.
func (b *Bot) abyssRunLoadout(uid string) (map[string]int, bool) {
	var js sql.NullString
	if err := b.DB.QueryRow("SELECT consumables FROM abyss_active WHERE client_uid=$1", uid).Scan(&js); err != nil {
		return nil, false
	}
	if !js.Valid || js.String == "" {
		return nil, false
	}
	m := map[string]int{}
	if err := json.Unmarshal([]byte(js.String), &m); err != nil {
		return nil, false
	}
	return m, true
}

// abyssSpendLoadout decrements one charge of consID from the active run's loadout
// (a no-op for an unrestricted run). Serialized by the per-uid Abyss lock.
func (b *Bot) abyssSpendLoadout(uid, consID string) {
	m, restricted := b.abyssRunLoadout(uid)
	if !restricted {
		return
	}
	if _, ok := m[consID]; !ok {
		return
	}
	m[consID]--
	if m[consID] <= 0 {
		delete(m, consID)
	}
	js, _ := json.Marshal(m)
	_, _ = b.DB.Exec("UPDATE abyss_active SET consumables=$1 WHERE client_uid=$2", string(js), uid)
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

func (b *Bot) spawnEchoMob(uid string, avgLvl int) ([]content.Mob, string, int) {
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
		return nil, "", 0
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

	return []content.Mob{mob}, echoNick, echoDepth
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

	// Fold the active daily affix into the combat modifier so it actually bites in
	// the engine (previously the daily mod only touched durability + the UI banner).
	// The token-carried affixes are read inside the combat engine via FloorModifier:
	// double_hazards (applyEffects), iron_skin (mobTurn), bloodlust (userTurn).
	// enraged_mobs is wired onto the spawned mobs below; glass_cannon ramps difficulty.
	_, dailyMod := b.currentDailyChallenge()
	switch dailyMod {
	case "double_hazards", "iron_skin", "bloodlust", "execute", "vampiric_mobs":
		if !strings.Contains(u.FloorModifier, dailyMod) {
			u.FloorModifier = strings.TrimSpace(u.FloorModifier + " " + dailyMod)
		}
	}

	// Self-imposed pacts stack on top of the daily affix: fold their combat tokens
	// into the modifier (read by the engine) the same way the daily affix is folded.
	pacts := b.abyssRunPacts(uid)
	for _, tok := range abyssPactCombatTokens(pacts) {
		if !strings.Contains(u.FloorModifier, tok) {
			u.FloorModifier = strings.TrimSpace(u.FloorModifier + " " + tok)
		}
	}

	st := b.loadAbyssStats(uid)
	diff, forceBoss := abyssDifficulty(depth)
	diff *= tier.DiffMult * (1.0 + float64(prestige)*0.05) * abyssDailyDangerMult(dailyMod) * abyssPactDangerMult(pacts) // [17] prestige & tier scaling + daily affix + pacts
	worldBoss := forceBoss && depth%(abyssBossEvery*2) == 0
	// Mob level is decoupled from the player's exact level (see abyssMobLevel): the
	// custom encounters and the spawned group all key off this depth-scaled value.
	mobLevel := abyssMobLevel(depth, u.Level)

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

	// Pass 0 gear score: the zone's rarity baseline and level still set its flavour
	// difficulty, but gear no longer inflates it (that double-counted with the old
	// abyssDifficulty and made every floor brutal for geared characters).
	zone := content.GetRandomZone(u.Level, 0)
	zone.Name = zoneName

	var mobs []content.Mob
	var echoNick string
	switch modifier {
	case "watcher":
		lvlScale := 1.0 + 0.05*float64(mobLevel-1)
		effectiveDiff := 1.0 + (diff-1.0)*0.3
		bossDef := 10 + mobLevel/2
		if bossDef > 80 {
			bossDef = 80
		}
		mobs = []content.Mob{
			{
				Name:     "The Watcher",
				Type:     content.MobBoss,
				Level:    mobLevel + 2,
				Stats:    content.Stats{
					HP:  int(1500 * lvlScale * effectiveDiff),
					STR: int(40 * lvlScale * abyssMobDamageMult * effectiveDiff),
					DEF: bossDef,
					SPD: 110,
				},
				RewardXP: 250,
				Element:  content.ElementPhysical,
				Effects:  []content.MobEffect{content.EffectEnraged},
			},
		}
		logs = append(logs, "[color=#f44336]👁️ The Watcher has found you! You lingered too long in the dark, and the Stalker of the Abyss strikes from the shadows![/color]")
	case "echo_encounter":
		var echoDepth int
		mobs, echoNick, echoDepth = b.spawnEchoMob(uid, u.Level)
		if len(mobs) > 0 {
			logs = append(logs, fmt.Sprintf("[color=#9c27b0]👻 An eerie silence falls. Out of the shadows rises the Ghostly Echo of %s (Depth %d delver)![/color]", echoNick, echoDepth))
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

			lvlScale := 1.0 + 0.05*float64(mobLevel-1)
			effectiveDiff := 1.0 + (diff-1.0)*0.3
			bossDef := 10 + mobLevel/2
			if bossDef > 90 {
				bossDef = 90
			}
			mobs = []content.Mob{
				{
					Name:     bossName,
					Type:     content.MobBoss,
					Level:    mobLevel + 1,
					Stats:    content.Stats{
						HP:  int(1000 * lvlScale * effectiveDiff),
						STR: int(50 * lvlScale * abyssMobDamageMult * effectiveDiff),
						DEF: bossDef,
						SPD: 105,
					},
					RewardXP: 500,
					Element:  content.ElementPhysical,
				},
			}
			logs = append(logs, fmt.Sprintf("[color=#e91e63]💀 BOSS FLOOR! Out of the abyss rises %s![/color]", bossName))
		} else if modifier == "treasure_goblin" {
			lvlScale := 1.0 + 0.05*float64(mobLevel-1)
			effectiveDiff := 1.0 + (diff-1.0)*0.3
			gobDef := 5 + mobLevel/10
			if gobDef > 20 {
				gobDef = 20
			}
			mobs = []content.Mob{
				{
					Name:     "Hoarder Treasure Goblin",
					Type:     content.MobTreasureGoblin,
					Level:    mobLevel,
					Stats:    content.Stats{
						HP:  int(400 * lvlScale * effectiveDiff),
						STR: int(5 * lvlScale * abyssMobDamageMult * effectiveDiff),
						DEF: gobDef,
						SPD: 150,
					},
					RewardXP: 100,
					Element:  content.ElementPhysical,
				},
			}
			logs = append(logs, "[color=#ffeb3b]💰 A Treasure Goblin hoard! You corner a wealthy Treasure Goblin, but it starts sprinting towards a portal![/color]")
		} else {
			mobs = content.SpawnMobGroup(mobLevel, zone, diff*zone.Difficulty, 1, forceBoss)
		}
	}

	isBossFloor := forceBoss || worldBoss

	escalateMobs(mobs, depth, worldBoss) // [15] deeper floors → denser elites/effects
	if dailyMod == "enraged_mobs" || abyssPactsEnrage(pacts) {
		for i := range mobs {
			mobs[i].Effects = append(mobs[i].Effects, content.EffectEnraged)
		}
	}
	mobPtrs := make([]*content.Mob, len(mobs))
	for i := range mobs {
		// Dampen Abyss mob damage so floors play out over several rounds instead of
		// ending in the opening volley. HP is left intact so the fight still has
		// to be won — it just takes longer and rewards survivability.
		if mobs[i].Stats.STR > 0 {
			mobs[i].Stats.STR = int(float64(mobs[i].Stats.STR) * abyssMobDamageMult)
			if mobs[i].Stats.STR < 1 {
				mobs[i].Stats.STR = 1
			}
		}
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
			// Inherit the weekly-folded modifier so co-op allies share iron_skin /
			// bloodlust / double_hazards effects with the lead delver.
			partner.FloorModifier = u.FloorModifier
			partner.IsClone = true
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
			"INSERT INTO abyss_boss_kills (client_uid, boss_name, depth, kill_time_ms, tier) VALUES ($1, $2, $3, $4, $5)",
			uid, mobs[0].Name, depth, duration.Milliseconds(), tier.Key,
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
	if victory {
		// Override XP scaling for Abyss: just 1-20 XP per floor cleared
		rewardXP = 1 + rand.IntN(20)
		rewardXP = int(float64(rewardXP) * (1.0 + float64(st.AbyssPrestige)*0.05) * (1.0 + float64(st.UpInsight)*0.05)) // prestige + Insight node
		if lr, _ := b.awardXP(uid, "", rewardXP); lr != nil && lr.NewLevel >= PrestigeThreshold {
			b.doPrestige(uid) // [52] keep Abyss prestige consistent with the cycle
		}
	}

	// Gear wears down each floor (more on defeat), exactly like a cycle fight.
	var duraWarnings []string
	if dailyMod != "zero_durability_loss" {
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
	Level        int // player's real level, for reward scaling
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
	_ = b.DB.QueryRow("SELECT current_hp, level FROM users WHERE client_uid=$1", uid).Scan(&r.CurHP, &r.Level)
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

	_, dailyMod := s.bot.currentDailyChallenge()
	helpers := s.bot.loadCoopHelpers(uid)
	abyssSetPieces := s.bot.countEquippedAbyssGear(uid)
	_, abyssSetTier := content.AbyssSetBonus(abyssSetPieces)

	equipped := s.bot.getEquippedItems(uid)
	var slots []gearView
	for _, slot := range content.AllSlots {
		if g, ok := equipped[slot]; ok {
			slots = append(slots, toGearView(slot, g))
		}
	}
	inventory := s.bot.inventoryItems(uid)

	s.render(w, "abyss", map[string]any{
		"Title":          "The Abyss",
		"Nav":            "abyss",
		"U":              u,
		"Stats":          st,
		"Run":            run,
		"Tiers":          abyssTierList(st.BestDepth),
		"Leaders":        s.bot.abyssLeaderboards("normal"),
		"Season":         abyssSeasonLabel(),
		"History":        s.bot.abyssHistory(uid, 8),
		"Achieved":       s.bot.abyssAchievements(uid),
		"LoreList":       loreList,
		"Bestiary":       s.bot.loadAbyssBestiary(uid),
		"Consumables":    s.bot.getConsumables(uid),
		"DailyMod":      dailyMod,
		"Helpers":        helpers,
		"NextIsBoss":     run.Active && (run.Depth+1)%5 == 0,
		"AbyssSetPieces": abyssSetPieces,
		"AbyssSetTier":   abyssSetTier,
		"Bounty":         s.bot.abyssBountyStatus(uid),
		"Shop":           abyssShopCatalog,
		"Pacts":          abyssPactCatalog,
		"Equipped":       slots,
		"Inventory":      inventory,
		"LegendaryPity":  func() int {
			var pity int
			_ = s.bot.DB.QueryRow("SELECT legendary_pity FROM users WHERE client_uid=$1", uid).Scan(&pity)
			return pity
		}(),
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
		Tier        string         `json:"tier"`
		Pacts       []string       `json:"pacts"`
		Consumables map[string]int `json:"consumables"` // optional picked loadout: cons_id -> count to bring
	}
	_ = readJSON(r, &req)

	// Consumable carry cap. A player may hold more consumables than they can bring
	// into a single descent (raised by an equipped Consumable Pouch). When they're
	// over the cap they pick a loadout instead of being blocked; the unbrought ones
	// stay in their stash, just unusable this run. loadout stays nil (SQL NULL,
	// meaning "no restriction") when they're already under the cap.
	maxAllowedConsumables := 3
	equipped := s.bot.getEquippedItems(uid)
	for _, g := range equipped {
		if g.ID == "ABYSS_CONSUMABLE_POUCH" || strings.Contains(strings.ToLower(g.Name), "pouch") {
			maxAllowedConsumables = 8
			break
		}
	}
	owned, totalConsumables := s.bot.abyssOwnedConsumables(uid)
	var loadoutJSON any // nil => stored as SQL NULL (unrestricted)
	if totalConsumables > maxAllowedConsumables {
		picked, perr := abyssBuildConsumableLoadout(req.Consumables, owned, maxAllowedConsumables)
		if perr != "" {
			// Ask the client to prompt a picker; no state has changed yet.
			writeJSON(w, map[string]any{
				"ok": false, "pick_consumables": true, "error": perr,
				"consumables": owned, "max": maxAllowedConsumables, "total": totalConsumables,
			})
			return
		}
		b, _ := json.Marshal(picked)
		loadoutJSON = string(b)
	}

	tier, ok := abyssTierByKey(req.Tier)
	if !ok {
		tier = abyssTiers["normal"]
	}
	pacts := abyssValidatePacts(req.Pacts)

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
		`INSERT INTO abyss_active (client_uid, depth, escrow, tier, insured, revived, pacts, consumables, started_at, last_action_at)
		 VALUES ($1, 0, 0, $2, 0, FALSE, $3, $4, NOW(), NOW())`, uid, tier.Key, pacts, loadoutJSON); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	// Clear any loot escrow orphaned by an improperly-ended prior run.
	if _, err := tx.Exec("DELETE FROM abyss_escrow_loot WHERE client_uid=$1", uid); err != nil {
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
	var floorType string
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
			// Roll one of the mysterious-encounter types. Weighted toward the
			// merchant; the rest split the long tail of shrines, gambles and caches.
			// #nosec G404
			rEv := rand.Float64()
			if rEv < 0.34 {
				g := content.RandomGearDrop()
				c1 := content.RandomConsumable()
				c2 := content.RandomConsumable()
				
				var count1 int
				if c1.Type == content.ConsumableHealing || c1.Type == content.ConsumableRepair || c1.Type == content.ConsumableRevive {
					count1 = 1 + rand.IntN(5)
				} else {
					count1 = 1 + rand.IntN(3)
				}
				
				var count2 int
				if c2.Type == content.ConsumableHealing || c2.Type == content.ConsumableRepair || c2.Type == content.ConsumableRevive {
					count2 = 1 + rand.IntN(5)
				} else {
					count2 = 1 + rand.IntN(3)
				}
				
				var price1 int64
				if c1.Type == content.ConsumableBuff {
					price1 = int64(75 * count1)
				} else {
					price1 = int64(50 * count1)
				}
				
				var price2 int64
				if c2.Type == content.ConsumableBuff {
					price2 = int64(75 * count2)
				} else {
					price2 = int64(50 * count2)
				}
				
				name1 := c1.Name
				if count1 > 1 {
					name1 = fmt.Sprintf("%s x%d", c1.Name, count1)
				}
				
				name2 := c2.Name
				if count2 > 1 {
					name2 = fmt.Sprintf("%s x%d", c2.Name, count2)
				}
				
				eventState = fmt.Sprintf(`{"type":"merchant","items":[{"type":"gear","id":"%s","name":"%s","price":400},{"type":"cons","id":"%s","name":"%s","price":%d,"count":%d},{"type":"cons","id":"%s","name":"%s","price":%d,"count":%d}]}`, g.ID, g.Name, c1.ID, name1, price1, count1, c2.ID, name2, price2, count2)
			} else if rEv < 0.50 {
				eventState = `{"type":"imp"}`
			} else if rEv < 0.60 {
				eventState = `{"type":"shrine"}`
			} else if rEv < 0.70 {
				eventState = `{"type":"wishing_well"}`
			} else if rEv < 0.79 {
				eventState = `{"type":"gambler"}`
			} else if rEv < 0.86 {
				eventState = `{"type":"statue"}`
			} else if rEv < 0.92 {
				eventState = `{"type":"fountain"}`
			} else if rEv < 0.97 {
				eventState = `{"type":"mimic"}`
			} else {
				eventState = `{"type":"buried_cache"}`
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
		// Store NULL rather than an empty string for floors with no event payload
		// (e.g. rest floors) so the JSONB event_state column accepts the write.
		var evStateArg any
		if eventState != "" {
			evStateArg = eventState
		}
		// Update active run to rest/event floor
		_, err := s.bot.DB.Exec(
			`UPDATE abyss_active
			    SET depth=$1, floor_type=$2, modifier=$3, event_state=$4, last_action_at=NOW()
			  WHERE client_uid=$5`,
			newDepth, floorType, modifier, evStateArg, uid,
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
		_, dailyMod := s.bot.currentDailyChallenge()
		bonus = int64(float64(bonus) * abyssDailyRewardMult(dailyMod))
		bonus = int64(float64(bonus) * abyssPactRewardMult(s.bot.abyssRunPacts(uid)))

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

		hasLuckyCoin := false
		equipped := s.bot.getEquippedItems(uid)
		if _, hasCoin := equipped[content.SlotTrinket1]; hasCoin && equipped[content.SlotTrinket1].ID == "ABYSS_LUCKY_COIN" {
			hasLuckyCoin = true
		}
		newEscrow := int64(float64(escrowBefore)*(1.0+abyssEffectiveInterest(st.UpInterest, hasLuckyCoin))) + bonus // [56] interest + Compounding node
		if _, err := s.bot.DB.Exec("UPDATE abyss_active SET escrow=$1, floor_type='combat', modifier='', event_state=NULL, last_action_at=NOW() WHERE client_uid=$2", newEscrow, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		_, _ = s.bot.DB.Exec("UPDATE users SET abyss_best_depth = GREATEST(abyss_best_depth, $1) WHERE client_uid=$2", depth, uid)
		
		// Evolving Artifacts: gains level/XP on clearing floor
		if art, ok := equipped[content.SlotArtifact]; ok {
			art.GearLevel++
			switch art.GearLevel {
			case 3:
				art.Stats.HP += 100
				art.Stats.STR += 15
				art.Stats.DEF += 15
			case 5:
				art.Stats.HP += 250
				art.Stats.STR += 30
				art.Stats.DEF += 30
			}
			dataBytes, _ := json.Marshal(art)
			_, _ = s.bot.DB.Exec("UPDATE user_gear SET item_data=$1 WHERE slot='Artifact' AND client_uid=$2", string(dataBytes), uid)
		}

		out["bonus"] = bonus
		out["escrow"] = newEscrow
		// Surface any milestone newly earned this floor: depth, plus boss-kill and
		// bestiary counts (both updated during the fight that just resolved).
		var achs []string
		if ach := s.bot.checkDepthAchievements(uid, depth); ach != "" {
			achs = append(achs, ach)
		}
		if ach := s.bot.checkBossKillAchievements(uid); ach != "" {
			achs = append(achs, ach)
		}
		if ach := s.bot.checkBestiaryAchievements(uid); ach != "" {
			achs = append(achs, ach)
		}
		if len(achs) > 0 {
			out["achievement"] = strings.Join(achs, " · ")
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
			s.bot.grantConsumable(uid, c.ID, c.Duration)
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

	// Heal-to-full and the one-shot revived flag must commit together: otherwise a
	// failure after the heal would leave the player healed without consuming the
	// revival (a free heal exploit).
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec("UPDATE users SET current_hp=$1 WHERE client_uid=$2", stats.HP, uid); err != nil {
		_ = tx.Rollback()
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec("UPDATE abyss_active SET revived=TRUE, last_action_at=NOW() WHERE client_uid=$1", uid); err != nil {
		_ = tx.Rollback()
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	tier, _ := abyssTierByKey(run.Tier)
	res, err := s.bot.fightAbyssFloor(uid, run.Depth, tier, run.Modifier, "balanced")
	if err != nil {
		// Roll back the heal and the revived flag so a failed combat call doesn't
		// leave the player healed-but-unresolved or burn their one-shot revival.
		_, _ = s.bot.DB.Exec("UPDATE users SET current_hp=$1 WHERE client_uid=$2", run.CurHP, uid)
		_, _ = s.bot.DB.Exec("UPDATE abyss_active SET revived=FALSE WHERE client_uid=$1", uid)
		writeJSON(w, map[string]any{"ok": false, "error": "combat"})
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
	payout, jackpot, ferr := s.bot.forfeitAbyss(uid, run)
	if ferr != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
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
	payout, jackpot, ferr := s.bot.forfeitAbyss(uid, run)
	if ferr != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
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

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Apply the per-day gold guard inside the transaction so the cap is only
	// consumed if the gold credit and the rest of the bank commit succeed. [59]
	payout = s.bot.capAbyssDayGold(tx, uid, payout)

	var gold int64
	if payout > 0 {
		if err := tx.QueryRow("UPDATE users SET gold = gold + $1 WHERE client_uid=$2 RETURNING gold", payout, uid).Scan(&gold); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	} else {
		_ = tx.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	}

	var jackpotWin int64
	var bonusGear string
	isRecord := false

	if run.Depth > 0 {
		// Record breaker check (Item #82) — compare against the true global max
		var maxDepth int
		_ = tx.QueryRow("SELECT COALESCE(MAX(depth), 0) FROM abyss_runs").Scan(&maxDepth)
		if run.Depth > maxDepth {
			isRecord = true
		}

		if _, err := tx.Exec(
			"INSERT INTO abyss_runs (client_uid, depth, gold_banked, victory, tier) VALUES ($1,$2,$3,TRUE,$4)",
			uid, run.Depth, payout, run.Tier); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		_, _ = tx.Exec(
			`UPDATE users SET abyss_best_depth = GREATEST(abyss_best_depth, $1),
			        abyss_lifetime_banked = abyss_lifetime_banked + $2,
			        abyss_bank_streak = abyss_bank_streak + 1 WHERE client_uid=$3`,
			run.Depth, payout, uid)
	}
	if req.Cursed {
		_, _ = tx.Exec("UPDATE users SET abyss_curse_fights = 3 WHERE client_uid=$1", uid)
	}
	if _, err := tx.Exec("DELETE FROM abyss_active WHERE client_uid=$1", uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	// Post-commit side effects
	if run.Depth >= 10 {
		// Awarded only after the bank transaction commits so a rolled-back commit
		// can't hand out duplicate gear on retry. [55][57]
		bonusGear = s.bot.awardAbyssBonusGear(uid, run.Depth)
	}
	if run.Depth > 0 {
		baseTokens := run.Depth/5 + 1
		// Tribute node: +10% bank tokens per level, rounded down.
		tokenGrant := baseTokens + baseTokens*st.UpTribute/10
		s.bot.grantAbyssTokens(uid, tokenGrant) // [44] + Tribute node
		s.bot.recordGameResult(uid, "abyss", true, payout)
		jackpotWin = s.bot.tryAbyssJackpot(uid, run.Depth) // [62]
		if jackpotWin > 0 {
			gold += jackpotWin
		}
		if isRecord {
			uInfo, _ := s.loadWebUser(uid)
			go s.bot.BroadcastAbyssRecord(uInfo.Nickname, run.Depth)
		}
	}

	// Escrowed loot is now safely the player's — apply it and surface what they kept.
	// Done post-commit so a rolled-back bank can't hand out items for free.
	var escrowLoot []string
	for _, label := range s.bot.applyAbyssEscrowLoot(uid) {
		escrowLoot = append(escrowLoot, bbToHTML(label))
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
	if len(escrowLoot) > 0 {
		out["escrow_loot"] = escrowLoot
	}
	// Lifetime-banked milestone check (post-commit, so the running total is current).
	if run.Depth > 0 {
		var lifetime int64
		_ = s.bot.DB.QueryRow("SELECT abyss_lifetime_banked FROM users WHERE client_uid=$1", uid).Scan(&lifetime)
		if ach := s.bot.checkBankAchievements(uid, lifetime); ach != "" {
			out["achievement"] = ach
		}
	}
	writeJSON(w, out)
}

// depthLevelHint returns the player's real level used for the floor-bonus curve.
// loadAbyssRun populates run.Level from the users table, so rewards scale on the
// actual level rather than an HP-derived estimate (which gear/Vigor could inflate).
func (run abyssRun) depthLevelHint() int {
	if run.Level < 1 {
		return 1
	}
	return run.Level
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

	// If this run was started with a picked loadout (player carried more than the
	// carry cap), only the brought consumables are usable this descent.
	if loadout, restricted := s.bot.abyssRunLoadout(uid); restricted && loadout[req.ConsID] <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "you didn't bring this consumable on this descent"})
		return
	}

	stats, _, _, _ := s.bot.calculateTotalStats(uid, time.Now())

	switch c.Type {
	case content.ConsumableHealing:
		// A lobby heal must not bypass the downed revive flow: while downed the only
		// way back is handleAbyssRevive (which consumes the one-shot double-or-nothing).
		if run := s.bot.loadAbyssRun(uid); run.Active && run.Downed {
			writeJSON(w, map[string]any{"ok": false, "error": "cannot heal while downed — revive or concede"})
			return
		}
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
		s.bot.ensureGearMaxDurability(uid)
		_, _ = s.bot.DB.Exec("UPDATE user_gear SET durability = LEAST(durability + $1, "+gearMaxDurExpr+") WHERE client_uid = $2", repairAmt, uid)
		_, _ = s.bot.DB.Exec("UPDATE users SET artifact_durability = LEAST(artifact_durability + 15, 30) WHERE client_uid = $1 AND artifact_durability > 0", uid)
	case content.ConsumableBuff:
		// Buffs elixirs: manual use sets them to active (3 remaining fights).
		// Do NOT fall through to the shared delete — buffs stay owned while active.
		_, _ = s.bot.DB.Exec("UPDATE user_consumables SET remaining_fights = 3 WHERE client_uid = $1 AND cons_id = $2", uid, req.ConsID)
		s.bot.abyssSpendLoadout(uid, req.ConsID)
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

	// Consume 1 stacked item: decrement remaining_fights and only delete the row
	// when the last one is used, so stacked grants from grantConsumable aren't all
	// wiped by a single use.
	res, _ := s.bot.DB.Exec("UPDATE user_consumables SET remaining_fights = remaining_fights - 1 WHERE client_uid = $1 AND cons_id = $2 AND remaining_fights > 1", uid, req.ConsID)
	if n, _ := res.RowsAffected(); n == 0 {
		_, _ = s.bot.DB.Exec("DELETE FROM user_consumables WHERE client_uid = $1 AND cons_id = $2", uid, req.ConsID)
	}
	s.bot.abyssSpendLoadout(uid, req.ConsID)

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
			res, err := s.bot.DB.Exec("UPDATE users SET gold = gold - $1, current_hp = $2 WHERE client_uid = $3 AND gold >= $1", cost, stats.HP, uid)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if rows, _ := res.RowsAffected(); rows == 0 {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
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
			// Debit + repairs in one transaction so gold is never taken without the
			// gear actually being repaired.
			s.bot.ensureGearMaxDurability(uid)
			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()
			res, err := tx.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1", cost, uid)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if rows, _ := res.RowsAffected(); rows == 0 {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			if _, err := tx.Exec("UPDATE user_gear SET durability = "+gearMaxDurExpr+" WHERE client_uid = $1", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if _, err := tx.Exec("UPDATE users SET artifact_durability = 30 WHERE client_uid = $1 AND artifact_name IS NOT NULL", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "msg": "All gear fully repaired!", "gold": gold - cost})
			return

		case "reroll_skills":
			cost := int64(150)
			if gold < cost {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			// Debit + delete + re-roll in one transaction so a failure can't leave the
			// player charged with their skills deleted but not replaced.
			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()
			res, err := tx.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1", cost, uid)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if rows, _ := res.RowsAffected(); rows == 0 {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			if _, err := tx.Exec("DELETE FROM user_skills WHERE client_uid = $1", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			for i := 1; i <= 5; i++ {
				sk := content.RandomSkill()
				if _, err := tx.Exec("INSERT INTO user_skills (client_uid, slot, skill_id) VALUES ($1, $2, $3)", uid, i, sk.ID); err != nil {
					writeJSON(w, map[string]any{"ok": false, "error": "db"})
					return
				}
			}
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
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
				Count int64  `json:"count"`
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
					count := int(item.Count)
					if count <= 0 {
						count = 1
					}
					fights := c.Duration
					if fights <= 0 {
						fights = 1
					}
					s.bot.grantConsumable(uid, c.ID, fights*count)
				}
			}
			state.Items = append(state.Items[:idx], state.Items[idx+1:]...)
			newStateBytes, _ := json.Marshal(state)
			_, _ = s.bot.DB.Exec("UPDATE abyss_active SET event_state = $1, last_action_at = NOW() WHERE client_uid = $2", string(newStateBytes), uid)

			writeJSON(w, map[string]any{"ok": true, "msg": "Bought " + item.Name + "!", "gold": gold - item.Price, "event_state": string(newStateBytes)})
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
			res, err := s.bot.DB.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1", cost, uid)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			// The guarded UPDATE can match zero rows (concurrent spend); only roll
			// rewards if the wager was actually debited.
			if n, _ := res.RowsAffected(); n == 0 {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
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
				s.bot.grantConsumable(uid, c.ID, c.Duration)
			} else {
				ui := content.RandomUniqueItem()
				msg = "JACKPOT! The Imp drops a Unique Item: " + ui.Name + "!"
				_, _ = s.bot.DB.Exec("INSERT INTO user_unique_items (client_uid, item_name, rarity, power) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING", uid, ui.Name, ui.Rarity, ui.Power)
			}
			
			_, _ = s.bot.DB.Exec("UPDATE abyss_active SET event_state = NULL, last_action_at = NOW() WHERE client_uid = $1", uid)
			writeJSON(w, map[string]any{"ok": true, "msg": msg, "gold": newGold, "resolved": true})
			return

		case "shrine_accept":
			if state.Type != "shrine" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for shrine_accept"})
				return
			}
			newEscrow := run.Escrow + 1000
			// Escrow gain and the curse are the two halves of the shrine bargain; apply
			// them atomically so a player can't get the +1,000 without the hex.
			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()
			if _, err := tx.Exec("UPDATE abyss_active SET escrow = $1, event_state = NULL, last_action_at = NOW() WHERE client_uid = $2", newEscrow, uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if _, err := tx.Exec("UPDATE users SET abyss_curse_fights = abyss_curse_fights + 5 WHERE client_uid = $1", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "msg": "Shrine accepted! +1,000 gold added to cache, but you are cursed!", "escrow": newEscrow, "resolved": true})
			return

		case "well_toss":
			if state.Type != "wishing_well" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for well_toss"})
				return
			}
			const cost = int64(250)
			// The gold cost, escrow gain and event-state clear are all one bargain: run
			// them in a single transaction so a failed clear can't leave the well
			// replayable after the player already paid.
			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()
			res, err := tx.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1", cost, uid)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if n, _ := res.RowsAffected(); n == 0 {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			// #nosec G404 -- non-cryptographic reward roll
			roll := rand.Float64()
			var gain int64
			var msg string
			switch {
			case roll < 0.20:
				gain = 0
				msg = "The coin sinks without a ripple. The well keeps your gold and gives nothing."
			case roll < 0.80:
				gain = 600
				msg = "The water glows — the well blesses your cache with +600 gold!"
			default:
				gain = 1500
				msg = "✨ The well erupts with light! A jackpot blessing of +1,500 gold to your cache!"
			}
			newEscrow := run.Escrow
			if gain > 0 {
				newEscrow += gain
			}
			if _, err := tx.Exec("UPDATE abyss_active SET escrow = $1, event_state = NULL, last_action_at = NOW() WHERE client_uid = $2", newEscrow, uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "msg": msg, "gold": gold - cost, "escrow": newEscrow, "resolved": true})
			return

		case "gambler_bet":
			if state.Type != "gambler" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for gambler_bet"})
				return
			}
			const cost = int64(250)
			// Bet, payout and event-state clear run in one transaction so a failed clear
			// can't leave the draw replayable after gold already changed hands.
			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()
			res, err := tx.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1", cost, uid)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if n, _ := res.RowsAffected(); n == 0 {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			newGold := gold - cost
			var msg string
			// #nosec G404 -- non-cryptographic 50/50 card draw
			if rand.Float64() < 0.50 {
				if err := tx.QueryRow("UPDATE users SET gold = gold + 500 WHERE client_uid = $1 RETURNING gold", uid).Scan(&newGold); err != nil {
					writeJSON(w, map[string]any{"ok": false, "error": "db"})
					return
				}
				msg = "🃏 High card! The dealer pays out — you win 500 gold (net +250)!"
			} else {
				msg = "🃏 Low card. The dealer sweeps your 250 gold off the table."
			}
			if _, err := tx.Exec("UPDATE abyss_active SET event_state = NULL, last_action_at = NOW() WHERE client_uid = $1", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "msg": msg, "gold": newGold, "resolved": true})
			return

		case "statue_touch":
			if state.Type != "statue" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for statue_touch"})
				return
			}
			// A free blessing: heal to full and bless the cache. Resolves the floor so
			// it can't be farmed for repeated free heals.
			stats, _, _, _ := s.bot.calculateTotalStats(uid, time.Now())
			newEscrow := run.Escrow + 400
			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()
			if _, err := tx.Exec("UPDATE users SET current_hp = $1 WHERE client_uid = $2", stats.HP, uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if _, err := tx.Exec("UPDATE abyss_active SET escrow = $1, event_state = NULL, last_action_at = NOW() WHERE client_uid = $2", newEscrow, uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "msg": "🗿 The ancient statue radiates warmth — healed to full and +400 gold blessed into your cache.", "hp": stats.HP, "escrow": newEscrow, "resolved": true})
			return

		case "fountain_drink":
			if state.Type != "fountain" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for fountain_drink"})
				return
			}
			// Fountain of Youth: free full heal + full gear repair. Resolves the floor.
			stats, _, _, _ := s.bot.calculateTotalStats(uid, time.Now())
			s.bot.ensureGearMaxDurability(uid)
			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()
			if _, err := tx.Exec("UPDATE users SET current_hp = $1 WHERE client_uid = $2", stats.HP, uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if _, err := tx.Exec("UPDATE user_gear SET durability = "+gearMaxDurExpr+" WHERE client_uid = $1", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if _, err := tx.Exec("UPDATE users SET artifact_durability = 30 WHERE client_uid = $1 AND artifact_name IS NOT NULL", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if _, err := tx.Exec("UPDATE abyss_active SET event_state = NULL, last_action_at = NOW() WHERE client_uid = $1", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "msg": "⛲ The Fountain of Youth restores you — healed to full and all gear repaired.", "hp": stats.HP, "resolved": true})
			return

		case "mimic_open":
			if state.Type != "mimic" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for mimic_open"})
				return
			}
			// Risk/reward: the chest is often real treasure, but sometimes a mimic that
			// bites. The bite can't kill (clamped to 1 HP) — events never end a run.
			// #nosec G404 -- non-cryptographic risk roll
			if rand.Float64() < 0.60 {
				gain := int64(800 + rand.IntN(1400)) // #nosec G404
				newEscrow := run.Escrow + gain
				if _, err := s.bot.DB.Exec("UPDATE abyss_active SET escrow = $1, event_state = NULL, last_action_at = NOW() WHERE client_uid = $2", newEscrow, uid); err != nil {
					writeJSON(w, map[string]any{"ok": false, "error": "db"})
					return
				}
				writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("🎁 Real treasure! The chest spills +%d gold into your cache.", gain), "escrow": newEscrow, "resolved": true})
				return
			}
			stats, _, _, _ := s.bot.calculateTotalStats(uid, time.Now())
			var curHP int
			_ = s.bot.DB.QueryRow("SELECT current_hp FROM users WHERE client_uid=$1", uid).Scan(&curHP)
			bite := stats.HP / 4
			newHP := curHP - bite
			if newHP < 1 {
				newHP = 1
			}
			// Apply the bite and clear the event together so a failed clear can't leave
			// the chest replayable for repeated bites.
			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()
			if _, err := tx.Exec("UPDATE users SET current_hp = $1 WHERE client_uid = $2", newHP, uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if _, err := tx.Exec("UPDATE abyss_active SET event_state = NULL, last_action_at = NOW() WHERE client_uid = $1", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "msg": "🦷 IT'S A MIMIC! The chest sprouts teeth and bites you before fleeing.", "hp": newHP, "resolved": true})
			return

		case "cache_dig":
			if state.Type != "buried_cache" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for cache_dig"})
				return
			}
			// A guaranteed sealed item, rolled from the Abyss pool and dropped straight
			// into the loot escrow (recovered on bank, lost on death like all cache loot).
			g := content.RandomAbyssGearDrop()
			label := fmt.Sprintf("%s [s:%s] (gs:%d R:%s)", g.Name, string(g.Slot), g.Stats.Score(), g.Rarity.String())
			gg := g
			// Seal the loot and clear the event in one transaction so a failed clear
			// can't leave the dig replayable for infinite free items.
			data, err := json.Marshal(abyssLootGrant{Type: "gear", Gear: &gg})
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()
			if _, err := tx.Exec("INSERT INTO abyss_escrow_loot (client_uid, item_type, label, item_data) VALUES ($1,$2,$3,$4)", uid, "gear", label, data); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if _, err := tx.Exec("UPDATE abyss_active SET event_state = NULL, last_action_at = NOW() WHERE client_uid = $1", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "msg": "⛏️ You unearth a buried cache! " + label + " is sealed into your loot cache.", "resolved": true})
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
	_, dailyMod := s.bot.currentDailyChallenge()
	bonus = int64(float64(bonus) * abyssDailyRewardMult(dailyMod))
	bonus = int64(float64(bonus) * abyssPactRewardMult(s.bot.abyssRunPacts(uid)))
	
	hasLuckyCoin := false
	equipped := s.bot.getEquippedItems(uid)
	if _, hasCoin := equipped[content.SlotTrinket1]; hasCoin && equipped[content.SlotTrinket1].ID == "ABYSS_LUCKY_COIN" {
		hasLuckyCoin = true
	}
	newEscrow := int64(float64(run.Escrow)*(1.0+abyssEffectiveInterest(st.UpInterest, hasLuckyCoin))) + bonus

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
		s.bot.grantConsumable(uid, c.ID, c.Duration)
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

func (b *Bot) currentDailyChallenge() (int64, string) {
	now := time.Now().UTC()
	// Seed by calendar day (year + day-of-year) so the challenge affix rotates once
	// per day rather than once per week.
	seed := int64(now.Year()*1000 + now.YearDay())
	return seed, abyssDailyMods[seed%int64(len(abyssDailyMods))]
}

// abyssDailyMods is the rotating pool of daily challenge affixes. Each is wired
// into a concrete effect: double_hazards/enraged_mobs/glass_cannon touch combat,
// zero_durability_loss touches gear wear, and gold_rush/glass_cannon touch the
// escrow reward (see abyssDailyRewardMult / abyssDailyDangerMult).
var abyssDailyMods = []string{
	"double_hazards",
	"zero_durability_loss",
	"enraged_mobs",
	"glass_cannon",
	"gold_rush",
	"iron_skin",
	"bloodlust",
	"execute",
	"vampiric_mobs",
}

// abyssDailyRewardMult is the escrow-bonus multiplier the active daily affix
// applies to every cleared floor this week.
func abyssDailyRewardMult(dailyMod string) float64 {
	switch dailyMod {
	case "gold_rush":
		return 2.0
	case "glass_cannon":
		return 1.3
	case "iron_skin", "execute":
		// Safer floors pay a little less, keeping the risk/reward honest.
		return 0.9
	case "vampiric_mobs":
		// Tougher, drawn-out fights pay a little more.
		return 1.15
	}
	return 1.0
}

// abyssDailyDangerMult is the floor-difficulty multiplier the active daily affix
// applies to every combat floor this week.
func abyssDailyDangerMult(dailyMod string) float64 {
	if dailyMod == "glass_cannon" {
		return 1.3
	}
	return 1.0
}

// countEquippedAbyssGear returns how many Abyss-exclusive (ABYSS_) pieces the
// player currently has equipped, for the set-bonus readout on the Abyss page.
func (b *Bot) countEquippedAbyssGear(uid string) int {
	var n int
	_ = b.DB.QueryRow(`SELECT COUNT(*) FROM user_gear WHERE client_uid=$1 AND gear_id LIKE 'ABYSS\_%'`, uid).Scan(&n)
	return n
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
	// Verify the helper is eligible — same rule as loadCoopHelpers: a known user
	// who has actually descended (abyss_best_depth > 0).
	var helperExists bool
	_ = s.bot.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE client_uid=$1 AND abyss_best_depth > 0)", req.CoopUID).Scan(&helperExists)
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

	out := map[string]any{"ok": true, "prestige": st.AbyssPrestige + 1}
	if ach := s.bot.checkThresholdAchievements(uid, 1, []achTier{{1, "prestige_1"}}); ach != "" {
		out["achievement"] = ach
	}
	writeJSON(w, out)
}
