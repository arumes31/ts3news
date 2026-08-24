package bot

import (
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"ts3news/internal/content"
	"ts3news/internal/leveling"
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
	// abyssDayGoldCap bounds untaxed Abyss bank payouts per player per day;
	// anything past it is still paid out, but the Abyss levies an 80% tax on
	// the excess (fed to the deep-cache jackpot) to protect the shared economy.
	abyssDayGoldCap = 5_000_000
	// abyssJackpotDepth is the minimum bank depth that can hit the deep-cache pot.
	abyssJackpotDepth = 25
	// abyssExpressGoldPerDepth is the gold cost per skipped depth for an
	// express-elevator start (#3).
	abyssExpressGoldPerDepth = 1000
	// abyssWatcherIdle is how long a run may sit idle before the Watcher Stalker
	// ambush (Item #67) forces the next floor to be a combat encounter.
	abyssWatcherIdle = 15 * time.Minute
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

// softCap returns x unchanged up to capAt, then grows logarithmically past it.
func softCap(x, capAt float64) float64 {
	if x <= capAt {
		return x
	}
	return capAt + math.Log(1+(x-capAt))
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

// abyssRiskCRScale calibrates the "Greed Meter" risk indicator: how much
// combat rating counts as one full unit of floor difficulty. Purely
// informational — abyssDifficulty itself stays deliberately un-gear-scaled
// (see its comment above), this meter never feeds back into real combat.
const abyssRiskCRScale = 1000.0

// abyssRiskPct returns a rough 0-100 risk indicator for the given floor,
// comparing its tier-scaled difficulty against the player's current equipped
// gear combat rating.
func abyssRiskPct(depth int, tier abyssTier, playerCR float64) int {
	effDiff, _ := abyssDifficulty(depth)
	effDiff *= tier.DiffMult
	pct := int(100 * effDiff / (effDiff + playerCR/abyssRiskCRScale))
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return pct
}

// abyssPlayerCR sums CombatRating() across every currently-equipped item, the
// same per-item metric already shown on the Armoury page.
func (b *Bot) abyssPlayerCR(uid string) float64 {
	var total float64
	for _, g := range b.getEquippedItems(uid) {
		total += g.CombatRating()
	}
	return total
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

// abyssMobScalars returns the level/difficulty scalars shared by the custom Abyss
// encounters (Watcher, bosses, treasure goblin): a gentle per-level growth and a
// dampened difficulty curve so these set-piece fights stay in their intended band.
func abyssMobScalars(mobLevel int, diff float64) (lvlScale, effDiff float64) {
	lvlScale = 1.0 + 0.05*float64(mobLevel-1)
	effDiff = 1.0 + (diff-1.0)*0.3
	return lvlScale, effDiff
}

// buildAbyssUser assembles a UserInCombat for the solo descent, mirroring the
// per-channel construction in the bot cycle so the engine sees an identical
// character. It does NOT auto-heal: HP carries between floors (the wound is the
// risk), and a fully-depleted character is handled by the "downed" state in the
// descend handler, not silently revived.
func (b *Bot) buildAbyssUser(uid string) (UserInCombat, int, error) {
	stats, _, _, _ := b.calculateTotalStats(uid, time.Now())

	// Skill web: allocated nodes add flat stats plus the combat %-multipliers
	// (economy keys are consumed by their own hooks in loot/bank/XP paths).
	tb := b.treeBonusFor(uid)
	stats = abyssFoldStats(stats, tb)

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

	u := UserInCombat{
		UID:         uid,
		Nickname:    nullStr(nick),
		Stats:       stats,
		Level:       lvl,
		Skills:      b.getSkills(uid),
		Ultimates:   b.getActiveUltimates(uid),
		CurrentHP:   curHP,
		RegenStacks: regen,
		Gold:        gold,
		Pets:        b.getPets(uid),
		Equipped:    b.getEquippedItems(uid),
		// Abyss drops are escrowed for the run, not granted inline by the engine.
		EscrowLoot: true,
		treeBonus:  tb,
	}
	applyAbyssRunBuild(&u, b.loadRunFlags(uid), b.loadAbyssSkillMastery(uid))
	return u, prestige, nil
}

// abyssFoldStats folds the skill-web bonus into base+gear stats the way Abyss combat
// does — flat adds first, then the combat %-multipliers. Single source of truth for
// buildAbyssUser (the live combatant) and abyssCombatStats (out-of-combat readouts),
// so the two can never drift on how the tree bonus composes.
func abyssFoldStats(base content.Stats, tb content.TreeBonus) content.Stats {
	return tb.ApplyCombatPct(base.Add(tb.Stats))
}

// abyssCombatStats returns the player's Abyss combat stats: base+gear folded with the
// skill-web bonus. Every out-of-combat HP write and max-HP readout goes through this
// so the dashboard, revives, rests and event heals agree with the max HP the descent
// actually fights at (calculateTotalStats alone omits the tree).
func (b *Bot) abyssCombatStats(uid string) content.Stats {
	stats, _, _, _ := b.calculateTotalStats(uid, time.Now())
	u := UserInCombat{Stats: abyssFoldStats(stats, b.treeBonusFor(uid))}
	applyAbyssRunBuild(&u, b.loadRunFlags(uid), nil)
	return u.Stats
}

func nullStr(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}

// applyAbyssRegen applies the real-time life-regen affix from equipped gear. It
// heals HP for the wall-clock time elapsed since the last application (tracked
// in app_meta, so no schema change), capped at max HP, and returns the new HP
// plus the combined regen rate in HP/sec that the dashboard uses for its live
// ticker. Regen accrues between page loads / out of combat.
func (b *Bot) applyAbyssRegen(uid string, equipped map[content.GearSlot]content.Gear, curHP, maxHP int) (int, float64) {
	perSec := 0.0
	for _, g := range equipped {
		if g.RegenAmount > 0 && g.RegenIntervalSec > 0 {
			perSec += float64(g.RegenAmount) / float64(g.RegenIntervalSec)
		}
	}
	// Skill-web HP-regen nodes (the 💚 regen cluster + "of Rejuvenation" enchants)
	// contribute HP/sec through the same out-of-combat regen the gear affix uses.
	if v := b.treeBonusFor(uid).Pct["hp_regen"]; v > 0 {
		perSec += v
	}
	key := "abyss_hp_regen_" + uid
	now := time.Now()
	// RFC3339Nano (not RFC3339): the CAS below advances the anchor by a sub-second
	// span for fast-regen builds; second-precision would truncate newAnchor back to
	// the stored value, so the CAS would "succeed" without advancing and re-credit the
	// same span every view. Parsing (time.Parse RFC3339) already accepts fractions.
	setTS := func(t time.Time) {
		_, _ = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, key, t.Format(time.RFC3339Nano))
	}
	// Ineligible states still anchor the clock to now. Otherwise the idle/dead
	// span accumulates and a later re-equip or revive would back-credit healing
	// for time regen was never active.
	if perSec <= 0 {
		setTS(now) // no regen gear: keep the clock current so re-equipping starts fresh
		return curHP, 0
	}
	if curHP <= 0 {
		setTS(now) // dead: gear regen must never revive; wait for a real revive
		return curHP, 0
	}
	if curHP >= maxHP {
		setTS(now) // already full: anchor the clock so regen starts fresh later
		return curHP, perSec
	}
	var tsStr string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", key).Scan(&tsStr)
	last, err := time.Parse(time.RFC3339, tsStr)
	if tsStr == "" || err != nil {
		setTS(now) // first-ever view: start the clock, heal nothing yet
		return curHP, perSec
	}
	elapsed := now.Sub(last).Seconds()
	if elapsed <= 0 {
		return curHP, perSec
	}
	heal := int(perSec * elapsed)
	if heal <= 0 {
		return curHP, perSec // too little time passed; let it accumulate
	}
	// Claim this elapsed span before healing: advance the anchor from the exact value we
	// read (tsStr) to last+consumed in a single compare-and-swap, wrapped with the heal in
	// one tx. Concurrent views all read the same tsStr, but only one CAS can match it (the
	// row lock serializes them) — the losers get 0 rows and must not also credit the span
	// (what let N open tabs heal N-fold). Advancing by only the consumed time keeps the
	// sub-second remainder accumulating; a transient heal failure rolls the anchor back
	// with it, so the span retries next view instead of being silently forfeited.
	consumed := float64(heal) / perSec
	newAnchor := last.Add(time.Duration(consumed * float64(time.Second)))
	tx, txErr := b.DB.Begin()
	if txErr != nil {
		return curHP, perSec
	}
	defer func() { _ = tx.Rollback() }()
	casRes, casErr := tx.Exec(
		"UPDATE app_meta SET value=$1 WHERE key=$2 AND value=$3", newAnchor.Format(time.RFC3339Nano), key, tsStr)
	if casErr != nil {
		return curHP, perSec
	}
	if n, _ := casRes.RowsAffected(); n == 0 {
		return curHP, perSec // another concurrent view already claimed this span
	}
	// Heal relative to the stored row so a concurrent HP change (e.g. a parallel fight)
	// isn't clobbered by a stale absolute value; the WHERE guards keep it from reviving
	// the dead or overshooting max HP.
	var newHP int
	err = tx.QueryRow(
		`UPDATE users SET current_hp = LEAST($1, current_hp + $2)
			WHERE client_uid = $3 AND current_hp > 0 AND current_hp < $1
			RETURNING current_hp`, maxHP, heal, uid).Scan(&newHP)
	if errors.Is(err, sql.ErrNoRows) {
		// Already full/dead: no HP to credit, but keep the advanced anchor (commit) so
		// the idle span isn't re-credited later.
		_ = tx.Commit()
		return curHP, perSec
	}
	if err != nil {
		return curHP, perSec // transient error: defer rolls back, span retries next view
	}
	if err := tx.Commit(); err != nil {
		return curHP, perSec
	}
	return newHP, perSec
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
			"UPDATE user_gear SET item_data = $1 WHERE client_uid = $2 AND slot = $3 AND gear_id = $4 AND item_data IS NULL",
			string(data), uid, r.slot, r.gearID)
	}
}

// grantConsumable adds a consumable to a player's stash. If they already hold the
// same consumable its remaining_fights is increased, rather than the grant being
// silently dropped — the old `ON CONFLICT DO NOTHING` lost paid purchases (gold
// spent, nothing granted).
// consumableCombineRecipes defines passive lesser→greater consumable combines:
// once a stack reaches Need after a grant, 3 lesser are auto-consumed for 1
// greater — no manual crafting UI, it just happens.
var consumableCombineRecipes = map[string]struct {
	Into string
	Need int
}{
	"small_health_potion": {"great_health_potion", 3},
	"repair_kit":          {"master_repair_kit", 3},
}

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
	b.autoCombineConsumable(uid, consID)
}

// autoCombineConsumable checks whether consID has a passive combine recipe and,
// if the player's stack now meets the threshold, consumes it (single pass, no
// recursive re-combining) for one of the greater item.
func (b *Bot) autoCombineConsumable(uid, consID string) {
	recipe, ok := consumableCombineRecipes[consID]
	if !ok {
		return
	}
	res, err := b.DB.Exec(
		"UPDATE user_consumables SET remaining_fights = remaining_fights - $1 WHERE client_uid=$2 AND cons_id=$3 AND remaining_fights >= $1",
		recipe.Need, uid, consID)
	if err != nil {
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return
	}
	_, _ = b.DB.Exec("DELETE FROM user_consumables WHERE client_uid=$1 AND cons_id=$2 AND remaining_fights<=0", uid, consID)
	_, _ = b.DB.Exec(
		`INSERT INTO user_consumables (client_uid, cons_id, remaining_fights)
		 VALUES ($1, $2, 1)
		 ON CONFLICT (client_uid, cons_id)
		 DO UPDATE SET remaining_fights = user_consumables.remaining_fights + 1`,
		uid, recipe.Into)
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
func abyssBuildConsumableLoadout(picked map[string]int, owned []consumableOwned, maxCarry int) (map[string]int, string) {
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
	if sum > maxCarry {
		return nil, fmt.Sprintf("You can bring at most %d (you picked %d).", maxCarry, sum)
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
	Timeline  []combatTimelineFrame
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
	return b.fightAbyssFloorLive(uid, depth, tier, modifier, focus, nil)
}

func (b *Bot) fightAbyssFloorLive(
	uid string,
	depth int,
	tier abyssTier,
	modifier string,
	focus string,
	live *abyssLiveCombat,
) (abyssFloorResult, error) {
	encounterRandom := combatRandomSource(defaultCombatRandomSource{})
	if live != nil {
		encounterRandom = live
	}
	u, prestige, err := b.buildAbyssUser(uid)
	if err != nil {
		return abyssFloorResult{}, err
	}
	u.LootFocus = focus
	u.FloorModifier = modifier
	u.live = live

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

	// Expansion-2 combat modifiers: momentum (#7, +2% STR per consecutive floor
	// without consumables, cap +20%), comeback (#24, +10% all stats), the Warden
	// specialization (#161, +5% all stats) and bestiary mastery (#168, +1% STR
	// per mob family with 100+ kills, cap +10%).
	frun := b.loadAbyssRun(uid)
	if frun.Active && frun.Momentum > 0 {
		mom := frun.Momentum
		if mom > 10 {
			mom = 10
		}
		u.Stats.STR += u.Stats.STR * mom * 2 / 100
		logs = append(logs, fmt.Sprintf("[color=#41c97a]🔥 Momentum ×%d: +%d%% STR (no consumables used).[/color]", mom, mom*2))
	}
	if frun.Active && frun.Comeback {
		u.Stats = u.Stats.Scaled(1.10)
		logs = append(logs, "[color=#41c97a]💪 Comeback: the Abyss pities you — +10% to all stats this run.[/color]")
	}
	if b.abyssSpec(uid) == "warden" {
		u.Stats = u.Stats.Scaled(1.05)
	}
	if fams := b.bestiaryMasteryFamilies(uid); fams > 0 {
		if fams > 10 {
			fams = 10
		}
		u.Stats.STR += u.Stats.STR * fams / 100
	}

	// 300-improvements run modifiers (docs/ABYSS_IMPROVEMENTS_300.md group A/B).
	flags := b.loadRunFlags(uid)
	weeklyRule, weeklyExpedition := abyssWeeklyRuleFromFlags(flags)
	if weeklyExpedition {
		diff *= weeklyRule.DangerMult
		logs = append(logs, "[color=#8f7cff]📅 Weekly Expedition — "+weeklyRule.Label+".[/color]")
		if weeklyRule.Key == "volatile_depths" && !strings.Contains(u.FloorModifier, "double_hazards") {
			u.FloorModifier = strings.TrimSpace(u.FloorModifier + " double_hazards")
		}
	}
	if frun.Active {
		if stacks := abyssGreedyGripStacks(frun.Depth); stacks > 0 { // AB-1 greedy grip
			u.Stats.DEF -= u.Stats.DEF * 2 * stacks / 100
			if u.Stats.DEF < 0 {
				u.Stats.DEF = 0
			}
			logs = append(logs, fmt.Sprintf("[color=#f44336]✊ Greedy grip ×%d: -%d%% DEF, but +%d%% cache interest (bank to shake it off).[/color]", stacks, 2*stacks, 2*stacks))
		}
		if pct := abyssHeavyPocketsPct(frun.Escrow); pct > 0 { // AB-2 heavy pockets
			u.Stats.SPD -= u.Stats.SPD * pct / 100
			logs = append(logs, fmt.Sprintf("[color=#f44336]🎒 Heavy pockets: the cache weighs you down, -%d%% SPD.[/color]", pct))
		}
	}
	if flags["death_wish"] == 1 { // AB-9
		diff *= 3
		logs = append(logs, "[color=#f44336]💀 DEATH WISH: this floor is 3× as deadly — and pays double.[/color]")
	}
	if flags["hybrid"] == 1 && depth%5 == 0 { // AB-25 hybrid runs
		if nt, ok := abyssNextTier(tier.Key); ok {
			diff *= nt.DiffMult
			logs = append(logs, fmt.Sprintf("[color=#9c27b0]🌀 Hybrid run: this floor surges to %s-tier danger![/color]", nt.Name))
		}
	}
	if flags["cold_muscles"] > 0 { // AB-13 cold muscles
		u.Stats = u.Stats.Scaled(0.9)
		logs = append(logs, "[color=#4a6fa5]🥶 Cold muscles: -10% stats (fresh from the elevator).[/color]")
	}
	if flags["spd_curse"] > 0 { // AB-32 cursed library SPD price
		u.Stats.SPD = u.Stats.SPD * 85 / 100
		logs = append(logs, "[color=#9c27b0]📚 The library's price: your legs feel leaden (-15% SPD).[/color]")
	}
	if flags["cursed_door_floors"] > 0 {
		u.Stats.STR = u.Stats.STR * 85 / 100
		u.Stats.DEF = u.Stats.DEF * 85 / 100
		logs = append(logs, "[color=#9c27b0]🚪 Door curse: -15% STR and DEF this fight.[/color]")
	}
	if flags["explorer_guard_floors"] > 0 {
		u.Stats.DEF += u.Stats.DEF / 10
		logs = append(logs, "[color=#41c97a]🧭 Explorer's map: +10% DEF this fight.[/color]")
	}
	if dm := flags["def_momentum"]; dm > 0 { // AB-18 defensive momentum
		if dm > 10 {
			dm = 10
		}
		u.Stats.DEF += u.Stats.DEF * int(dm) * 2 / 100
		logs = append(logs, fmt.Sprintf("[color=#41c97a]🛡️ Defensive momentum ×%d: +%d%% DEF (untouched streak).[/color]", dm, dm*2))
	}
	// AB-8 the abyss notices: idling on a floor adds +1% danger per minute.
	if frun.Active && !frun.LastActionAt.IsZero() {
		if mins := int(time.Since(frun.LastActionAt).Minutes()); mins >= 1 {
			if mins > 50 {
				mins = 50
			}
			diff *= 1.0 + float64(mins)/100.0
			logs = append(logs, fmt.Sprintf("[color=#f44336]👁️ The abyss notices your hesitation: +%d%% danger.[/color]", mins))
		}
	}

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
	biome := content.AbyssBiomeFor(depth)
	zoneName := biome.Name + " " + abyssZoneName(depth)
	diff *= biome.DiffMod
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
	zone := content.GetRandomZoneWithRandom(u.Level, 0, encounterRandom)
	zone.Name = zoneName

	var mobs []content.Mob
	var echoNick string
	switch modifier {
	case "watcher":
		lvlScale, effectiveDiff := abyssMobScalars(mobLevel, diff)
		bossDef := 10 + mobLevel/2
		if bossDef > 80 {
			bossDef = 80
		}
		mobs = []content.Mob{
			{
				Name:  "The Watcher",
				Type:  content.MobBoss,
				Level: mobLevel + 2,
				Stats: content.Stats{
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

			lvlScale, effectiveDiff := abyssMobScalars(mobLevel, diff)
			bossDef := 10 + mobLevel/2
			if bossDef > 90 {
				bossDef = 90
			}
			mobs = []content.Mob{
				{
					Name:  bossName,
					Type:  content.MobBoss,
					Level: mobLevel + 1,
					Stats: content.Stats{
						HP:  int(1000 * lvlScale * effectiveDiff),
						STR: int(50 * lvlScale * abyssMobDamageMult * effectiveDiff),
						DEF: bossDef,
						SPD: 105,
					},
					RewardXP: 500,
					Element:  content.ElementPhysical,
				},
			}
			// Boss intro card (#201): a framed banner with name and stakes.
			logs = append(logs,
				"[hr]",
				fmt.Sprintf("[center][size=12][color=#e91e63]💀 BOSS — %s[/color][/size][/center]", bossName),
				fmt.Sprintf("[center][color=#8a93a8][i]Depth %d · steel yourself — it knows you are here.[/i][/color][/center]", depth),
				"[hr]")
		} else if modifier == "treasure_goblin" {
			lvlScale, effectiveDiff := abyssMobScalars(mobLevel, diff)
			gobDef := 5 + mobLevel/10
			if gobDef > 20 {
				gobDef = 20
			}
			mobs = []content.Mob{
				{
					Name:  "Hoarder Treasure Goblin",
					Type:  content.MobTreasureGoblin,
					Level: mobLevel,
					Stats: content.Stats{
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
			mobs = content.SpawnMobGroupWithRandom(
				mobLevel,
				zone,
				diff*zone.Difficulty,
				1,
				forceBoss,
				encounterRandom,
			)
		}
	}

	isBossFloor := forceBoss || worldBoss

	escalateMobsWithRandom(mobs, depth, worldBoss, encounterRandom) // [15] deeper floors → denser elites/effects
	var enemySystemLogs []string
	mobs, enemySystemLogs = b.prepareAbyssEnemies(uid, depth, mobs, encounterRandom)
	logs = append(logs, enemySystemLogs...)
	if dailyMod == "enraged_mobs" || abyssPactsEnrage(pacts) || weeklyRule.Key == "elite_surge" {
		for i := range mobs {
			mobs[i].Effects = append(mobs[i].Effects, content.EffectEnraged)
		}
	}
	if weeklyRule.Key == "iron_trial" {
		for i := range mobs {
			mobs[i].Stats.DEF += mobs[i].Stats.DEF * 15 / 100
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

	// AB-12 experience vs killer: a permanent grudge — bonus damage vs the mob
	// families that have killed you before.
	if kb := b.killerExpBonusTenths(uid, mobs); kb > 0 {
		u.Stats.STR += u.Stats.STR * kb / 1000
		logs = append(logs, fmt.Sprintf("[color=#41c97a]🎓 Experience vs killer: +%.1f%% damage — you know these foes all too well.[/color]", float64(kb)/10))
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
			if partner.Level > u.Level {
				partner.Stats = mentorScaledStats(partner.Stats, u.Stats)
				partner.CurrentHP = min(partner.CurrentHP, partner.Stats.HP)
				logs = append(logs, fmt.Sprintf("[color=#4a6fa5]🎓 Mentor scaling: %s joins at no more than 120%% of the host's stats.[/color]", partner.Nickname))
			}
			partner.LootFocus = focus
			// Inherit the weekly-folded modifier so co-op allies share iron_skin /
			// bloodlust / double_hazards effects with the lead delver.
			partner.FloorModifier = u.FloorModifier
			partner.IsClone = true
			partner.live = live
			combatUsers = append(combatUsers, partner)
			logs = append(logs, fmt.Sprintf("[color=#4a6fa5]🔔 Co-op Ally %s has entered the fray to assist you![/color]", partner.Nickname))
		}
	}
	flagsByUID := make(map[string]map[string]int64, len(combatUsers))
	for i := range combatUsers {
		flagsByUID[combatUsers[i].UID] = b.loadRunFlags(combatUsers[i].UID)
	}
	if synergyLog, active := applyAbyssPartyBuildSynergy(combatUsers, flagsByUID); active {
		logs = append(logs, "[color=#41c97a]"+synergyLog+"[/color]")
	}

	hpBefore := u.CurrentHP
	startTime := time.Now()
	// The engine's per-kill reward is ignored here: Abyss uses its own small
	// per-floor XP payout below. We only need the win/loss outcome from combat.
	combatLogOffset := len(logs)
	resLogs, _, victory, loots, timeline := b.resolveChannelCombatDetailed(combatUsers, mobPtrs, u.Level, diff, zone)
	b.updateAbyssNemesis(uid, mobPtrs, victory)
	duration := time.Since(startTime)
	logs = append(logs, resLogs...)
	for i := range timeline {
		timeline[i].AfterLog += combatLogOffset
	}

	if victory && coopUID.Valid && coopUID.String != "" {
		helperTokens := 5
		switch abyssPartyLootRuleFromID(flags["party_loot_rule"]) {
		case "round_robin":
			helperTokens = 8
		case "need_before_greed":
			helperTokens = 6
		}
		b.grantAbyssTokens(coopUID.String, helperTokens)
		// Surface the helper's nickname, never their raw TS3 UID, in the public log.
		coopNick := coopUID.String
		_ = b.DB.QueryRow("SELECT COALESCE(NULLIF(nickname, ''), 'Adventurer') FROM users WHERE client_uid=$1", coopUID.String).Scan(&coopNick)
		logs = append(logs, fmt.Sprintf("[color=#4a6fa5]🔔 Helper %s receives %d Abyss Tokens under the %s party rule; run gear remains with the host cache.[/color]", coopNick, helperTokens, strings.ReplaceAll(abyssPartyLootRuleFromID(flags["party_loot_rule"]), "_", " ")))
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
		// AB-159: bosses have a modest chance to drop a branch-refund shard.
		// #nosec G404 -- non-cryptographic reward roll
		if encounterRandom.Float64() < 0.20 && b.grantAbyssMasteryShard(uid) {
			logs = append(logs, "🔮 The boss dropped a Mastery Shard — refund one skill-web branch for free!")
		}
	}

	// Record kills in Bestiary — use CurrentHP (live value) not Stats.HP (base max)
	var killedMobs []string
	for _, m := range mobPtrs {
		if m.CurrentHP <= 0 && m.Type != content.MobTreasureGoblin && !abyssEnemyHazard(m) {
			killedMobs = append(killedMobs, m.Name)
		}
	}
	if len(killedMobs) > 0 {
		b.recordAbyssKills(uid, killedMobs)
	}

	// Abyss floor XP stays in its deliberate small band (per-kill "kept small"):
	// a cleared floor pays the full 1-20 roll, a death still banks ~25% of it. The
	// engine also applies its own level-XP death penalty on a loss. Prestige fires
	// immediately at the cap like the cycle does.
	var rewardXP int
	{
		// #nosec G404 -- non-cryptographic reward roll
		rewardXP = 1 + encounterRandom.IntN(20)
		if !victory {
			rewardXP = (rewardXP + 3) / 4 // ~25% on death, rounds up so a death still pays >=1
		}
		rewardXP = int(float64(rewardXP) * abyssPermanentBonus(float64(st.AbyssPrestige)*0.05, 0.50) * (1.0 + float64(st.UpInsight)*0.05)) // prestige + Insight node
		if b.abyssSpec(uid) == "delver" {
			rewardXP = rewardXP * 11 / 10 // Delver specialization (#161): +10% floor XP
		}
		if focus == "xp" {
			rewardXP *= 2 // XP focus: double floor XP (loot rolls are skipped instead)
		}
		// Skill web: Void-sector xp_gain notables (single lookup — treeBonusFor
		// costs a DB read plus a full tree scan).
		treePct := b.treeBonusFor(uid).Pct
		if v := treePct["xp_gain"]; v > 0 {
			rewardXP = int(float64(rewardXP) * (1 + v))
		}
		if progressPct := abyssProgressionXPPercent(flags); progressPct > 0 {
			rewardXP = rewardXP * (100 + progressPct) / 100
			logs = append(logs, fmt.Sprintf("[color=#41c97a]🌙 Progression reserve: +%d%% floor XP.[/color]", progressPct))
		}
		// Alchemy of the Soul: converts 50% of descent XP gain into gold (Item 42).
		// The gold lands only after awardXP persists the reduced XP, so a failed
		// XP write can never mint gold on top of unspent XP (worst case is an
		// under-reward, matching the tree bonus best-effort philosophy).
		var convertedXP int
		if conv := treePct["xp_to_gold"]; conv > 0 {
			convertedXP = int(float64(rewardXP) * conv)
			rewardXP -= convertedXP
		}
		lr, xpErr := b.awardXP(uid, "", rewardXP)
		if xpErr == nil && convertedXP > 0 {
			convertedGold := int64(convertedXP)
			if _, err := b.DB.Exec("UPDATE users SET gold = gold + $2 WHERE client_uid = $1", uid, convertedGold); err == nil {
				logs = append(logs, fmt.Sprintf("✨ Alchemy of the Soul: converted %d XP into 🜲 %d Gold!", convertedXP, convertedGold))
			}
		}
		if lr != nil && lr.NewLevel >= PrestigeThreshold {
			b.doPrestige(uid) // [52] keep Abyss prestige consistent with the cycle
		}
	}

	// Gear wears down each floor (more on defeat), exactly like a cycle fight.
	var duraWarnings []string
	if dailyMod != "zero_durability_loss" {
		duraWarnings = b.applyDurabilityLoss(uid, !victory)
	}

	stats := b.abyssCombatStats(uid)
	var curHP int
	_ = b.DB.QueryRow("SELECT current_hp FROM users WHERE client_uid=$1", uid).Scan(&curHP)
	if curHP < 0 {
		curHP = 0
	}

	// End-of-fight summary (#51): a one-line recap so the long log has a TLDR.
	outcome := "☠️ Defeated"
	if victory {
		outcome = "✅ Victorious"
	}
	logs = append(logs, fmt.Sprintf("[hr][color=#8a93a8]📊 %s · %d foe(s) · fight time %d ms · HP %s → %s (%+d)[/color]",
		outcome, len(mobs), duration.Milliseconds(), FormatGoldPlain(int64(hpBefore)), FormatGoldPlain(int64(curHP)), curHP-hpBefore))

	res := abyssFloorResult{Victory: victory, RewardXP: rewardXP, Timeline: timeline, CurrentHP: curHP, MaxHP: stats.HP}
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

// bbTagReplacer maps the remaining known BBCode tokens to their safe HTML
// equivalents. Package-level: it is built once instead of per log line.
var bbTagReplacer = strings.NewReplacer(
	"[/color]", "</span>",
	"[b]", "<b>", "[/b]", "</b>",
	"[i]", "<i>", "[/i]", "</i>",
	"[center]", `<span class="ab-center">`, "[/center]", "</span>",
	"[size=12]", `<span class="ab-big">`, "[/size]", "</span>",
	"[hr]", `<span class="ab-hr"></span>`,
)

// bbToHTML converts the TeamSpeak BBCode the combat engine emits into a small,
// safe subset of HTML for the web log. The input is HTML-escaped first, so any
// player-controlled text (nicknames) cannot inject markup; only the known BBCode
// tokens are then turned back into tags.
func bbToHTML(s string) string {
	s = html.EscapeString(s)
	s = bbColorRe.ReplaceAllString(s, `<span style="color:$1">`)
	return bbTagReplacer.Replace(s)
}

// runLootRow is one escrowed drop, formatted for the right-hand loot manifest
// sidebar on the Abyss page. Label is pre-sanitised HTML from bbToHTML (which
// escapes its input first), typed template.HTML so the template doesn't
// escape it a second time and show literal <span> markup.
type runLootRow struct {
	Label template.HTML
	Depth int
	Title string // plain-text label for the hover tooltip (full, un-truncated detail)
}

// bbTagRe strips every BBCode tag, leaving plain text for title tooltips.
var bbTagRe = regexp.MustCompile(`\[[^\]]*\]`)

// currentRunLootManifest returns every item escrowed so far in the player's
// active run, oldest first, for the loot manifest sidebar.
func (b *Bot) currentRunLootManifest(uid string) []runLootRow {
	rows, err := b.DB.Query("SELECT label, depth FROM abyss_escrow_loot WHERE client_uid=$1 ORDER BY id", uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []runLootRow
	for rows.Next() {
		var label string
		var depth int
		if err := rows.Scan(&label, &depth); err == nil {
			out = append(out, runLootRow{Label: template.HTML(bbToHTML(label)), Depth: depth, Title: bbTagRe.ReplaceAllString(label, "")}) // #nosec G203 -- bbToHTML escapes first
		}
	}
	return out
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
	StartedAt    time.Time
	LastActionAt time.Time
	CoopUID      string

	// Expansion-2 run state (docs/ABYSS_IDEAS.md)
	Momentum         int  // #7 consecutive floors without consumable use
	BankLockedFloors int  // #15 floors left before banking unlocks after a Last Stand
	LastStandUsed    bool // #15 one Last Stand per run
	ReviveLocked     bool // #15 double-or-nothing not offered on this run's down
	CheckpointStart  int  // #2 run started at this checkpoint depth (rewards ×0.75)
	ExpressUntil     int  // #3 express elevator: no floor bonus until past this depth
	Comeback         bool // #24 comeback buff active (+10% stats)
}

// loadAbyssRun reads the active run plus the player's live HP so callers can tell
// whether the player is mid-fight, downed, or has no run at all.
func (b *Bot) loadAbyssRun(uid string) abyssRun {
	var r abyssRun
	var evState, coop sql.NullString
	var startedAt, lastAct time.Time
	err := b.DB.QueryRow(
		`SELECT depth, escrow, tier, insured, revived, floor_type, modifier, event_state, started_at, last_action_at, coop_uid,
		        momentum, bank_locked_floors, last_stand_used, revive_locked, checkpoint_start, express_until, comeback
		   FROM abyss_active WHERE client_uid=$1`, uid,
	).Scan(&r.Depth, &r.Escrow, &r.Tier, &r.Insured, &r.Revived, &r.FloorType, &r.Modifier, &evState, &startedAt, &lastAct, &coop,
		&r.Momentum, &r.BankLockedFloors, &r.LastStandUsed, &r.ReviveLocked, &r.CheckpointStart, &r.ExpressUntil, &r.Comeback)
	if err != nil {
		return r
	}
	r.Active = true
	if evState.Valid {
		r.EventState = evState.String
	}
	r.StartedAt = startedAt
	r.LastActionAt = lastAct
	if coop.Valid {
		r.CoopUID = coop.String
	}
	stats := b.abyssCombatStats(uid)
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
	// The dashboard must show the same stats the descent actually fights with, so
	// fold in the Abyss skill-web bonus (loadWebUser only knows base + gear).
	// Mirrors buildAbyssUser, so max HP and every stat update the moment you
	// allocate or respec the web.
	u.Stats = s.bot.abyssCombatStats(uid)
	u.MaxHP = u.Stats.HP
	u.MaxMana = 100 + u.Stats.MNA
	st := s.bot.loadAbyssStats(uid)
	run := s.bot.loadAbyssRun(uid)
	// loadWebUser clamped CurrentHP to the base (gear-only) max; re-derive it from the
	// persisted HP (captured raw) against the true combat max so a full-health delver
	// isn't shown wounded once the tree bonus raises the ceiling (nor overflowing past it).
	switch {
	case u.RawCurrentHP <= 0:
		// A downed active run must read as downed, not fake-full; only outside a run
		// does blank/zero HP fall back to full (the lobby display convention, matching
		// loadWebUser). applyAbyssRegen below also treats curHP<=0 as dead → no regen.
		if run.Active {
			u.CurrentHP = 0
		} else {
			u.CurrentHP = u.MaxHP
		}
	case u.RawCurrentHP > u.MaxHP:
		u.CurrentHP = u.MaxHP
	default:
		u.CurrentHP = u.RawCurrentHP
	}

	loreList := []map[string]any{}
	unlockedMap := make(map[int]bool)
	for _, id := range s.bot.loadUnlockedLore(uid) {
		unlockedMap[id] = true
	}
	for id := 1; id <= len(abyssLoreFragments); id++ {
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
	abyssGearBySet := s.bot.countEquippedAbyssGearBySet(uid)
	_, abyssTierBySet := content.AbyssSetBonusBySet(abyssGearBySet)
	abyssSetPieces := abyssGearBySet["abyss_legacy"]
	abyssSetTier := abyssTierBySet["abyss_legacy"]
	predatorPieces, wardenPieces := abyssGearBySet["predator"], abyssGearBySet["warden"]
	predatorTier, wardenTier := abyssTierBySet["predator"], abyssTierBySet["warden"]
	harvesterPieces, harvesterTier := abyssGearBySet["harvester"], abyssTierBySet["harvester"]

	equipped := s.bot.getEquippedItems(uid)
	// Real-time life-regen: heal for the wall-clock time elapsed from regen-affix
	// gear before rendering, and hand the live rate to the dashboard ticker.
	newHP, regenPerSec := s.bot.applyAbyssRegen(uid, equipped, u.CurrentHP, u.MaxHP)
	u.CurrentHP = newHP
	var slots []gearView
	for _, slot := range content.AllSlots {
		if g, ok := equipped[slot]; ok {
			slots = append(slots, toGearView(slot, g))
		}
	}
	inventory := s.bot.inventoryItems(uid)

	var badgeCode sql.NullString
	var dropStreak int
	var pity int
	var craftWeek sql.NullString
	var craftDone int
	var forgeRep int
	var autoRepair bool
	_ = s.bot.DB.QueryRow(
		`SELECT abyss_active_badge, abyss_drop_streak, legendary_pity, craft_quest_week, craft_quest_done, forge_rep, abyss_auto_repair
		   FROM users WHERE client_uid=$1`, uid,
	).Scan(&badgeCode, &dropStreak, &pity, &craftWeek, &craftDone, &forgeRep, &autoRepair)

	activeBadge := ""
	activeBadgeName := ""
	if badgeCode.Valid && badgeCode.String != "" {
		activeBadge = badgeCode.String
		activeBadgeName = abyssAchievementName(activeBadge)
	}
	badgeOptions := []map[string]string{}
	for _, code := range s.bot.abyssAchievementCodes(uid) {
		badgeOptions = append(badgeOptions, map[string]string{"Code": code, "Name": abyssAchievementName(code)})
	}

	dropStreakBonusPct := dropStreak * 2
	if dropStreakBonusPct > 30 {
		dropStreakBonusPct = 30
	}

	risk := 0
	if run.Active {
		if runTier, ok := abyssTierByKey(run.Tier); ok {
			risk = abyssRiskPct(run.Depth+1, runTier, s.bot.abyssPlayerCR(uid))
		}
	}

	// Per-tier leaderboard tabs (#276): ?lbtier=<key> switches the boards.
	lbTier := r.URL.Query().Get("lbtier")
	if _, ok := abyssTierByKey(lbTier); !ok {
		lbTier = "normal"
	}

	// Checkpoint depths (#2) and express start (#3) for the entry picker.
	var checkpoints []int
	for d := 10; d <= st.BestDepth; d += 10 {
		checkpoints = append(checkpoints, d)
	}
	expressStart := 0
	if st.BestDepth >= 8 {
		expressStart = st.BestDepth - 5
	}
	sanctuary := s.bot.loadSanctuary(uid)
	sanctuaryStage, sanctuaryStageName := abyssSanctuaryStage(sanctuary)

	s.render(w, "abyss", map[string]any{
		"Title":              "The Abyss",
		"Nav":                "abyss",
		"U":                  u,
		"RegenPerSec":        regenPerSec,
		"Stats":              st,
		"Run":                run,
		"AutoFocus":          s.autoSelectFocus(uid, run),
		"Tiers":              abyssTierList(st.BestDepth),
		"Leaders":            s.bot.abyssLeaderboards(lbTier),
		"Season":             abyssSeasonLabel(),
		"History":            s.bot.abyssHistory(uid, 8),
		"Achieved":           s.bot.abyssAchievements(uid),
		"BadgeOptions":       badgeOptions,
		"ActiveBadge":        activeBadge,
		"ActiveBadgeName":    activeBadgeName,
		"LoreList":           loreList,
		"Bestiary":           s.bot.loadAbyssBestiary(uid),
		"Consumables":        s.bot.getConsumables(uid),
		"DailyMod":           dailyMod,
		"CommunityExpedition": s.bot.communityExpeditionStatus(),
		"Helpers":            helpers,
		"NextIsBoss":         run.Active && (run.Depth+1)%abyssBossEvery == 0,
		"AbyssSetPieces":     abyssSetPieces,
		"AbyssSetTier":       abyssSetTier,
		"PredatorPieces":     predatorPieces,
		"PredatorTier":       predatorTier,
		"WardenPieces":       wardenPieces,
		"WardenTier":         wardenTier,
		"HarvesterPieces":    harvesterPieces,
		"HarvesterTier":      harvesterTier,
		"Bounty":             s.bot.abyssBountyStatus(uid),
		"Shop":               abyssShopCatalog,
		"Pacts":              abyssPactCatalog,
		"Equipped":           slots,
		"Inventory":          inventory,
		"LegendaryPity":      pity,
		"DropStreak":         dropStreak,
		"DropStreakBonusPct": dropStreakBonusPct,
		"Risk":               risk,
		"RunLoot":            s.bot.currentRunLootManifest(uid),
		"CanLastStand":       run.Active && !run.LastStandUsed && s.bot.abyssTokens(uid) >= abyssLastStandCost(run.Depth),

		// Expansion 2 (docs/ABYSS_IDEAS.md)
		"Materials":    s.bot.loadMaterials(uid),
		"MaterialDefs": abyssMaterials,
		"Recipes":      abyssRecipeViews(s.bot, uid),
		"CraftQuest": func() map[string]any {
			done := craftDone
			if !craftWeek.Valid || craftWeek.String != craftQuestWeek() {
				done = 0
			}
			if done > craftQuestTarget {
				done = craftQuestTarget
			}
			return map[string]any{"Done": done, "Target": craftQuestTarget}
		}(),
		"Sanctuary":      sanctuary,
		"SanctuaryDefs":  sanctuaryUpgrades,
		"SanctuaryStage": sanctuaryStage,
		"SanctuaryStageName": sanctuaryStageName,
		"ProgressionTracks": s.bot.abyssProgressionViews(uid),
		"Spec":           s.bot.abyssSpec(uid),
		"SpecDefs":       abyssSpecs,
		"ForgeHistory":   s.bot.loadForgeHistory(uid, 12),
		"ForgeRep":       map[string]int{"Rep": forgeRep, "DiscountPct": forgeDiscountPct(forgeRep)},
		"ForgeHappyHour": forgeHappyHour(),
		"AutoRepair":     autoRepair,
		"TokenBuyGold":   int64(abyssTokenBuyGold),
		"TokenSellGold":  int64(abyssTokenSellGold),
		"PrestigeTier": func() map[string]string {
			n, a := abyssPrestigeTier(st.AbyssPrestige)
			return map[string]string{"Name": n, "Aura": a}
		}(),
		"CraftLegendaries": content.LegendaryCatalog(),
		"LBTier":           lbTier,
		"LBTiers":          abyssTierList(math.MaxInt32), // full list for the board tabs: a huge depth unlocks every tier
		"LastStandCost":    abyssLastStandCost(run.Depth),
		"NodeGates":        abyssUpgradeMinDepth,
		"Checkpoints":      checkpoints,
		"ExpressStart":     expressStart,
		"ExpressCost":      int64(expressStart) * abyssExpressGoldPerDepth,
	})
}

// abyssRecipeViews resolves recipes for the template, marking discovery state.
func abyssRecipeViews(b *Bot, uid string) []map[string]any {
	known := b.knownRecipes(uid)
	out := make([]map[string]any, 0, len(craftRecipes))
	for _, r := range craftRecipes {
		cost := make([]string, 0, len(r.Cost))
		for _, m := range abyssMaterials { // stable icon order
			if n := r.Cost[m.ID]; n > 0 {
				cost = append(cost, fmt.Sprintf("%s ×%d", m.Icon, n))
			}
		}
		out = append(out, map[string]any{
			"ID": r.ID, "Name": r.Name, "Desc": r.Desc,
			"Cost":   strings.Join(cost, " "),
			"Locked": r.Secret && !known[r.ID],
		})
	}
	return out
}

// ---- APIs ----------------------------------------------------------------

// abyssCarryCapBase / abyssCarryCapPouch bound how many consumable charges a
// delver can bring into one descent, without / with an equipped Consumable Pouch.
const (
	abyssCarryCapBase  = 3
	abyssCarryCapPouch = 8
)

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
		Start       string         `json:"start"`       // "" | "checkpoint" | "express" (#2/#3)
		Checkpoint  int            `json:"checkpoint"`  // requested checkpoint depth (multiple of 10)
		Expedition  bool           `json:"expedition"`  // weekly fixed-seed rules
		Kit         string         `json:"kit"`         // starting combat identity
		Mutation    string         `json:"mutation"`    // temporary in-run skill mutation
		LootRule    string         `json:"loot_rule"`   // party reward settlement selected before entry
		VeteranTrack string        `json:"veteran_track"` // optional cosmetic challenge, unlocked at depth 50
	}
	// Reject malformed JSON outright: a garbled body would silently decode to the
	// zero-value request (Normal tier, no pacts, no loadout). An absent/empty body
	// (io.EOF) stays valid and means the defaults.
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	// Consumable carry cap. A player may hold more consumables than they can bring
	// into a single descent (raised by an equipped Consumable Pouch). When they're
	// over the cap they pick a loadout instead of being blocked; the unbrought ones
	// stay in their stash, just unusable this run. loadout stays nil (SQL NULL,
	// meaning "no restriction") when they're already under the cap.
	maxAllowedConsumables := abyssCarryCapBase
	equipped := s.bot.getEquippedItems(uid)
	for _, g := range equipped {
		if g.ID == "ABYSS_POUCH" {
			maxAllowedConsumables = abyssCarryCapPouch
			break
		}
	}
	// Quartermaster node (#158): +1 carry slot per level on top of the base/pouch cap.
	stPre := s.bot.loadAbyssStats(uid)
	maxAllowedConsumables += stPre.UpQuartermaster
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
		writeJSON(w, map[string]any{"ok": false, "error": "unknown tier"})
		return
	}
	pacts := abyssValidatePacts(req.Pacts)

	st := stPre
	if st.BestDepth < tier.MinBest {
		writeJSON(w, map[string]any{"ok": false, "error": "tier locked — reach depth " + itoa(tier.MinBest) + " first"})
		return
	}
	if req.VeteranTrack != "" && st.BestDepth < 50 {
		writeJSON(w, map[string]any{"ok": false, "error": "veteran challenge tracks unlock at best depth 50"})
		return
	}
	if req.VeteranTrack != "" {
		if track, _ := normalizeAbyssVeteranTrack(req.VeteranTrack); track == "" {
			writeJSON(w, map[string]any{"ok": false, "error": "unknown veteran challenge track"})
			return
		}
	}

	// Reject entering while a run is already active (no free heal/reset).
	if s.bot.loadAbyssRun(uid).Active {
		writeJSON(w, map[string]any{"ok": false, "error": "already in a run"})
		return
	}

	// Checkpoint (#2) / express-elevator (#3) starts. Checkpoints are every 10
	// depths already reached and cost tokens; the run's rewards are reduced ×0.75.
	// Express skips to (best−5) for gold but pays no floor bonus until the player
	// passes their record.
	startDepth, checkpointStart, expressUntil := 0, 0, 0
	var startTokenCost, startGoldCost int64
	switch req.Start {
	case "checkpoint":
		c := req.Checkpoint
		if c <= 0 || c%10 != 0 || c > st.BestDepth {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid checkpoint — pick a multiple of 10 you have already reached"})
			return
		}
		startDepth, checkpointStart = c, c
		startTokenCost = int64(c / 2)
	case "express":
		if st.BestDepth < 8 {
			writeJSON(w, map[string]any{"ok": false, "error": "the express elevator unlocks at best depth 8"})
			return
		}
		startDepth = st.BestDepth - 5
		expressUntil = st.BestDepth
		startGoldCost = int64(startDepth) * abyssExpressGoldPerDepth
	}

	// Comeback buff (#24): three deaths on the same calendar day grant +10% stats
	// on the next run, clearly labeled in the run state.
	comeback := false
	{
		// Compare against CURRENT_DATE in SQL like the writer (forfeitAbyss) does —
		// a Go-side date comparison can disagree with the DB's timezone.
		var deaths int
		_ = s.bot.DB.QueryRow("SELECT abyss_deaths_today FROM users WHERE client_uid=$1 AND abyss_deaths_date = CURRENT_DATE", uid).Scan(&deaths)
		if deaths >= 3 {
			comeback = true
		}
	}

	// Wrap gold debit, HP reset, and abyss_active creation in a single transaction
	// so a failure after charging can't leave the player paid without an active run.
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Consume the comeback buff on entry so it is single-use
	if comeback {
		if _, err := tx.Exec("UPDATE users SET abyss_deaths_today = abyss_deaths_today - 3 WHERE client_uid = $1", uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}

	// Daily free descent (#1): the first paid entry of the calendar day is waived.
	entryGold := tier.EntryGold
	freeEntry := false
	if entryGold > 0 {
		res, err := tx.Exec(`UPDATE users SET abyss_free_entry_date = CURRENT_DATE
		                      WHERE client_uid=$1 AND (abyss_free_entry_date IS NULL OR abyss_free_entry_date < CURRENT_DATE)`, uid)
		if err == nil {
			if n, _ := res.RowsAffected(); n > 0 {
				entryGold = 0
				freeEntry = true
			}
		}
	}
	if charge := entryGold + startGoldCost; charge > 0 {
		res, err := tx.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid=$2 AND gold >= $1", charge, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough gold for entry"})
			return
		}
	}
	if startTokenCost > 0 {
		res, err := tx.Exec("UPDATE users SET abyss_tokens = abyss_tokens - $1 WHERE client_uid=$2 AND abyss_tokens >= $1", startTokenCost, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough tokens for the checkpoint"})
			return
		}
	}

	// Auto-repair before the descent (#125), silently skipped if unaffordable. A
	// DB error after the gold debit aborts the whole entry transaction, so the
	// charge can never commit without the repair actually happening.
	var autoRepaired int64
	{
		var on bool
		_ = s.bot.DB.QueryRow("SELECT abyss_auto_repair FROM users WHERE client_uid=$1", uid).Scan(&on)
		if on {
			if cost := s.bot.abyssRepairAllCost(uid); cost > 0 {
				res, err := tx.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid=$2 AND gold >= $1", cost, uid)
				if err != nil {
					writeJSON(w, map[string]any{"ok": false, "error": "db"})
					return
				}
				if n, _ := res.RowsAffected(); n > 0 {
					if _, err := tx.Exec("UPDATE user_gear SET durability = "+gearMaxDurExpr+" WHERE client_uid=$1", uid); err != nil {
						writeJSON(w, map[string]any{"ok": false, "error": "db"})
						return
					}
					if _, err := tx.Exec("UPDATE users SET artifact_durability = 30 WHERE client_uid = $1 AND artifact_name IS NOT NULL", uid); err != nil {
						writeJSON(w, map[string]any{"ok": false, "error": "db"})
						return
					}
					autoRepaired = cost
				}
			}
		}
	}

	// Vigor upgrade lets a run start above the normal max (banked as current HP).
	baseStats, _, _, _ := s.bot.calculateTotalStats(uid, time.Now())
	startBuildFlags := map[string]int64{
		abyssRunFlagBuildKit:      abyssBuildKits[normalizeAbyssBuildKit(req.Kit)],
		abyssRunFlagSkillMutation: abyssSkillMutations[normalizeAbyssSkillMutation(req.Mutation)],
	}
	startUser := UserInCombat{Stats: abyssFoldStats(baseStats, s.bot.treeBonusFor(uid)), Equipped: equipped}
	applyAbyssRunBuild(&startUser, startBuildFlags, nil)
	stats := startUser.Stats
	startHP := stats.HP + stats.HP*st.UpVigor*5/100
	if _, err := tx.Exec("UPDATE users SET current_hp=$1 WHERE client_uid=$2", startHP, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec(
		`INSERT INTO abyss_active (client_uid, depth, escrow, tier, insured, revived, pacts, consumables, started_at, last_action_at,
		                           checkpoint_start, express_until, comeback)
		 VALUES ($1, $5, 0, $2, 0, FALSE, $3, $4, NOW(), NOW(), $6, $7, $8)`,
		uid, tier.Key, pacts, loadoutJSON, startDepth, checkpointStart, expressUntil, comeback); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	_, hasActiveRelic := equipped[content.SlotRelic]
	weeklyRule, err := resetAbyssRunFlagsInTx(
		tx, uid, req.Expedition, req.Kit, req.Mutation, hasActiveRelic, time.Now(),
	)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := seedAbyssSocialRunFlagsInTx(tx, uid, req.LootRule); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	entryProgression, err := seedAbyssProgressionFlagsInTx(tx, uid, req.VeteranTrack, time.Now())
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	// Clear any loot escrow orphaned by an improperly-ended prior run.
	if _, err := tx.Exec("DELETE FROM abyss_escrow_loot WHERE client_uid=$1", uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	// A fresh run always starts with no win streak, so a value left over from a prior
	// run can't seed abyssStreakBuff into this run (or regular cycle combat).
	if _, err := tx.Exec("UPDATE users SET abyss_win_streak = 0 WHERE client_uid=$1", uid); err != nil {
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
		"ok": true, "depth": startDepth, "escrow": 0, "tier": tier.Key,
		"hp": startHP, "max_hp": stats.HP, "gold": gold,
		"free_entry": freeEntry, "comeback": comeback, "auto_repaired": autoRepaired,
		"weekly_expedition": weeklyRule.Label,
		"build_summary": abyssBuildSummary(startUser, startBuildFlags),
		"rested_charges": entryProgression.RestedCharges,
		"returning_bonus": entryProgression.Returning,
		"veteran_track": entryProgression.VeteranTrack,
		"loot_rule": normalizeAbyssPartyLootRule(req.LootRule),
		"tokens": s.bot.abyssTokens(uid),
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
	var req struct {
		Interactive bool `json:"interactive"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if live, ok := s.liveCombatForUID(uid); ok && live.isActive() {
		writeJSON(w, map[string]any{"ok": false, "error": "combat already active"})
		return
	}

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

	focus := s.autoSelectFocus(uid, run)

	newDepth := run.Depth + 1
	tier, _ := abyssTierByKey(run.Tier)

	// Forced floors bypass the choice picker entirely: the Watcher Stalker
	// ambush trigger (Item #67) and boss floors are never optional.
	if !run.LastActionAt.IsZero() && time.Since(run.LastActionAt) > abyssWatcherIdle && run.Depth > 0 {
		s.commitFloor(w, uid, run, newDepth, "combat", "watcher", "", tier, focus, req.Interactive)
		return
	}
	if newDepth%abyssBossEvery == 0 {
		s.commitFloor(w, uid, run, newDepth, "combat", "", "", tier, focus, req.Interactive)
		return
	}

	// Rift peek (#35): a pre-rolled floor queue seals the next floors' fate — no
	// choice picker, the revealed type simply happens.
	if ft, ok := s.bot.popFloorQueue(uid); ok {
		modifier, eventState := rollFloorDetail(ft)
		s.commitFloor(w, uid, run, newDepth, ft, modifier, eventState, tier, focus, req.Interactive)
		return
	}

	// Event cadence: when an event is due (every 2-6 floors) force it directly rather
	// than leaving events to a random per-floor roll, so they never land back-to-back
	// nor on every floor. commitFloor re-anchors the next event.
	if s.bot.abyssEventDue(uid, newDepth) {
		modifier, eventState := rollFloorDetail("event")
		s.commitFloor(w, uid, run, newDepth, "event", modifier, eventState, tier, focus, req.Interactive)
		return
	}

	// Otherwise offer the player a choice between 2 candidate floor paths (combat/rest
	// only — events are on their own cadence above).
	candidates := rollFloorCandidates(2, false)
	pending := pendingFloorChoice{Candidates: candidates, Focus: focus, Interactive: req.Interactive}
	buf, err := json.Marshal(pending)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "internal"})
		return
	}
	if _, err := s.bot.DB.Exec("UPDATE abyss_active SET pending_floor_choice=$1, last_action_at=NOW() WHERE client_uid=$2", string(buf), uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	revealRoute := s.bot.loadAbyssStats(uid).UpCartographer > 0
	writeJSON(w, map[string]any{
		"ok": true, "choose_floor": true, "depth": newDepth,
		"options": publicFloorCandidates(candidates, revealRoute), "escrow": run.Escrow,
	})
}

// descendMultiAbort builds the partial-failure payload for a batch descend.
// Floors resolved before the failure are already persisted server-side, so
// their logs/loot plus a fresh run snapshot ride along with the error and the
// client can reconcile depth, escrow, HP and wallet instead of drifting.
func (s *WebServer) descendMultiAbort(uid, errKey string, tier abyssTier, logs, loot, dura []string, timeline []combatTimelineFrame, rewardXP int) map[string]any {
	runFinal := s.bot.loadAbyssRun(uid)
	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	return map[string]any{
		"ok": false, "error": errKey,
		"logs": logs, "loot": loot, "dura": dura, "timeline": timeline, "reward_xp": rewardXP,
		"depth": runFinal.Depth, "escrow": runFinal.Escrow,
		"hp": runFinal.CurHP, "max_hp": runFinal.MaxHP,
		"gold": gold, "tokens": s.bot.abyssTokens(uid),
		"risk": abyssRiskPct(runFinal.Depth+1, tier, s.bot.abyssPlayerCR(uid)),
	}
}

// handleAbyssDescendMulti processes a queue of 3 to 10 planned floor descents sequentially.
func (s *WebServer) handleAbyssDescendMulti(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}

	var req struct {
		Paths []string `json:"paths"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	if len(req.Paths) < 3 || len(req.Paths) > 10 {
		writeJSON(w, map[string]any{"ok": false, "error": "Invalid queue length (must be 3 to 10 floors)"})
		return
	}
	// The planned paths are preferences, validated up-front; the server owns the
	// actual floor roll inside the loop.
	for _, pt := range req.Paths {
		if pt != "combat" && pt != "rest" && pt != "event" {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid floor type in queue"})
			return
		}
	}

	var combinedLogs []string
	var combinedLoot []string
	var combinedDura []string
	var combinedTimeline []combatTimelineFrame
	var totalRewardXP int
	var gearMilestone string
	var achs []string
	var loreUnlocked bool
	var loreFragment string
	var recipeUnlocked string
	var affixReward string
	var dailyFirst bool

	runInit := s.bot.loadAbyssRun(uid)
	if !runInit.Active {
		writeJSON(w, map[string]any{"ok": false, "error": "not in a run"})
		return
	}
	tier, _ := abyssTierByKey(runInit.Tier)

	for _, pt := range req.Paths {
		run := s.bot.loadAbyssRun(uid)
		if !run.Active {
			writeJSON(w, s.descendMultiAbort(uid, "not in a run", tier, combinedLogs, combinedLoot, combinedDura, combinedTimeline, totalRewardXP))
			return
		}
		if run.Downed {
			writeJSON(w, s.descendMultiAbort(uid, "you are downed — revive or concede", tier, combinedLogs, combinedLoot, combinedDura, combinedTimeline, totalRewardXP))
			return
		}
		if run.FloorType != "combat" {
			writeJSON(w, s.descendMultiAbort(uid, "you must resolve the current floor action first", tier, combinedLogs, combinedLoot, combinedDura, combinedTimeline, totalRewardXP))
			return
		}
		focus := s.autoSelectFocus(uid, run)

		newDepth := run.Depth + 1

		// The server owns the floor roll, mirroring a single descend: forced
		// watcher/boss floors first, then any rift-peek sealed floor (#35), then a
		// weighted 2-candidate roll where the planned path is honored only if the
		// roll actually offers it. The client's plan is a preference, never an
		// override — so batch requests can't force rest floors at will.
		actualType := "combat"
		modifier := ""
		eventState := ""

		if !run.LastActionAt.IsZero() && time.Since(run.LastActionAt) > abyssWatcherIdle && run.Depth > 0 {
			modifier = "watcher"
		} else if newDepth%abyssBossEvery == 0 {
			// Boss floors are never optional.
		} else {
			if ft, ok := s.bot.popFloorQueue(uid); ok {
				actualType = ft
			} else if s.bot.abyssEventDue(uid, newDepth) {
				actualType = "event" // cadence: force the due event (every 2-6 floors)
			} else {
				candidates := rollFloorCandidates(2, false)
				actualType = candidates[0].Type
				for _, c := range candidates {
					if c.Type == pt {
						actualType = c.Type
						break
					}
				}
			}
			modifier, eventState = rollFloorDetail(actualType)
		}
		eventState = prepareAbyssEventForDepth(eventState, newDepth)

		if actualType != "combat" {
			// Stop batch at rest or event floor and let the user interact.
			var evStateArg any
			if eventState != "" {
				evStateArg = eventState
			}
			_, err := s.bot.DB.Exec(
				`UPDATE abyss_active
				    SET depth=$1, floor_type=$2, modifier=$3, event_state=$4, pending_floor_choice=NULL, last_action_at=NOW()
				  WHERE client_uid=$5`,
				newDepth, actualType, modifier, evStateArg, uid,
			)
			if err != nil {
				writeJSON(w, s.descendMultiAbort(uid, "db", tier, combinedLogs, combinedLoot, combinedDura, combinedTimeline, totalRewardXP))
				return
			}
			if actualType == "event" {
				s.bot.abyssScheduleNextEvent(uid, newDepth) // re-anchor the 2-6 floor cadence
			}
			if actualType == "rest" {
				_, _ = s.bot.DB.Exec("UPDATE users SET abyss_win_streak = 0 WHERE client_uid=$1", uid)
			}

			var gold int64
			_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)

			runFinal := s.bot.loadAbyssRun(uid)
			writeJSON(w, map[string]any{
				"ok":          true,
				"noncombat":   true,
				"floor_type":  actualType,
				"depth":       newDepth,
				"event_state": eventState,
				"escrow":      run.Escrow,
				"risk":        abyssRiskPct(newDepth+1, tier, s.bot.abyssPlayerCR(uid)),
				"gold":        gold,
				"tokens":      s.bot.abyssTokens(uid),
				"consumables": s.bot.getConsumables(uid),
				"logs":        combinedLogs,
				"loot":        combinedLoot,
				"dura":        combinedDura,
				"timeline":    combinedTimeline,
				"reward_xp":   totalRewardXP,
				"auto_focus":  s.autoSelectFocus(uid, runFinal),
			})
			return
		}

		// Normal Combat floor
		if _, err := s.bot.DB.Exec("UPDATE abyss_active SET depth=$1, modifier=$2, event_state=NULL, pending_floor_choice=NULL, last_action_at=NOW() WHERE client_uid=$3", newDepth, modifier, uid); err != nil {
			writeJSON(w, s.descendMultiAbort(uid, "db", tier, combinedLogs, combinedLoot, combinedDura, combinedTimeline, totalRewardXP))
			return
		}

		res, err := s.bot.fightAbyssFloor(uid, newDepth, tier, modifier, focus)
		if err != nil {
			_, _ = s.bot.DB.Exec("UPDATE abyss_active SET depth=$1, modifier='', event_state=NULL, last_action_at=NOW() WHERE client_uid=$2", run.Depth, uid)
			// Earlier floors in this batch already resolved and persisted — return
			// their logs/loot alongside the error so they aren't lost client-side.
			writeJSON(w, s.descendMultiAbort(uid, "combat", tier, combinedLogs, combinedLoot, combinedDura, combinedTimeline, totalRewardXP))
			return
		}

		timelineOffset := len(combinedLogs)
		if len(res.LogsHTML) > 0 {
			combinedLogs = append(combinedLogs, fmt.Sprintf("<div class='ab-batch-header'>Floor %d Combat Logs</div>", newDepth))
			timelineOffset++
			combinedLogs = append(combinedLogs, res.LogsHTML...)
		}
		for _, frame := range res.Timeline {
			frame.AfterLog += timelineOffset
			combinedTimeline = append(combinedTimeline, frame)
		}
		combinedLoot = append(combinedLoot, res.LootHTML...)
		combinedDura = append(combinedDura, res.DuraHTML...)
		totalRewardXP += res.RewardXP

		_, _ = s.bot.DB.Exec("UPDATE users SET abyss_lifetime_floors = abyss_lifetime_floors + 1 WHERE client_uid=$1", uid)

		if res.Victory {
			o := s.applyFloorVictory(uid, run, newDepth, run.Escrow, tier, modifier, focus)
			if o.DBErr {
				writeJSON(w, s.descendMultiAbort(uid, "db", tier, combinedLogs, combinedLoot, combinedDura, combinedTimeline, totalRewardXP))
				return
			}
			if o.GearMilestone != "" {
				gearMilestone = o.GearMilestone
			}
			if o.DailyFirst {
				dailyFirst = true
			}
			achs = append(achs, o.Achievements...)
			if o.LoreUnlocked {
				loreUnlocked = true
				loreFragment = o.LoreFragment
			}
			if o.RecipeUnlocked != "" {
				recipeUnlocked = o.RecipeUnlocked
			}
			if o.AffixReward != "" {
				affixReward = o.AffixReward
			}
			run.Escrow = o.NewEscrow
		} else {
			// Defeat: stop batch run
			canRevive := s.applyFloorDefeat(uid, run)

			var gold int64
			_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)

			runFinal := s.bot.loadAbyssRun(uid)
			writeJSON(w, map[string]any{
				"ok":               true,
				"victory":          false,
				"depth":            newDepth,
				"hp":               res.CurrentHP,
				"max_hp":           res.MaxHP,
				"logs":             combinedLogs,
				"loot":             combinedLoot,
				"dura":             combinedDura,
				"timeline":         combinedTimeline,
				"reward_xp":        totalRewardXP,
				"risk":             abyssRiskPct(newDepth+1, tier, s.bot.abyssPlayerCR(uid)),
				"downed":           true,
				"can_revive":       canRevive,
				"can_last_stand":   !run.LastStandUsed && s.bot.abyssTokens(uid) >= abyssLastStandCost(newDepth),
				"last_stand_cost":  abyssLastStandCost(newDepth),
				"escrow":           run.Escrow,
				"insured":          run.Insured,
				"gold":             gold,
				"tokens":           s.bot.abyssTokens(uid),
				"consumables":      s.bot.getConsumables(uid),
				"auto_focus":       s.autoSelectFocus(uid, runFinal),
			})
			return
		}
	}

	var finalGold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&finalGold)

	finalRun := s.bot.loadAbyssRun(uid)

	out := map[string]any{
		"ok":                 true,
		"victory":            true,
		"depth":              finalRun.Depth,
		"hp":                 finalRun.CurHP,
		"max_hp":             finalRun.MaxHP,
		"logs":               combinedLogs,
		"loot":               combinedLoot,
		"dura":               combinedDura,
		"timeline":           combinedTimeline,
		"reward_xp":          totalRewardXP,
		"risk":               abyssRiskPct(finalRun.Depth+1, tier, s.bot.abyssPlayerCR(uid)),
		"escrow":             finalRun.Escrow,
		"gold":               finalGold,
		"tokens":             s.bot.abyssTokens(uid),
		"consumables":        s.bot.getConsumables(uid),
		"gear_milestone":     gearMilestone,
		"lore_unlocked":      loreUnlocked,
		"lore_fragment":      loreFragment,
		"recipe_unlocked":    recipeUnlocked,
		"affix_reward":       affixReward,
		"daily":              dailyFirst,
		"auto_focus":         s.autoSelectFocus(uid, finalRun),
	}
	if len(achs) > 0 {
		out["achievement"] = strings.Join(achs, " · ")
	}
	writeJSON(w, out)
}


// floorCandidate is one offered path in the branching-floor-choice picker.
type floorCandidate struct {
	Index int    `json:"index"`
	Type  string `json:"type"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
}

// pendingFloorChoice is stored uncommitted in abyss_active.pending_floor_choice
// between the descend roll and the player's pick.
type pendingFloorChoice struct {
	Candidates  []floorCandidate `json:"candidates"`
	Focus       string           `json:"focus"`
	Interactive bool             `json:"interactive,omitempty"`
}

var floorCandidateInfo = map[string]struct{ Label, Icon string }{
	"combat": {"Press onward", "⚔️"},
	"rest":   {"Rest at a sanctuary", "🕊️"},
	"event":  {"Investigate a strange presence", "❔"},
}

// abyssEventGapMin/Max bound the gap (in floors) between event floors: events fire on
// a controlled cadence of every 2-6 floors rather than randomly on any/every floor.
const (
	abyssEventGapMin = 2
	abyssEventGapMax = 6
)

// abyssEventDue reports whether this floor depth is at or past the scheduled next event
// floor. On a fresh run (no anchor yet) it schedules the first event 2-6 floors out and
// returns false, so the opening floors aren't events.
func (b *Bot) abyssEventDue(uid string, depth int) bool {
	var s string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", "abyss_next_event_depth_"+uid).Scan(&s)
	next, err := strconv.Atoi(s)
	if s == "" || err != nil {
		b.abyssScheduleNextEvent(uid, depth)
		return false
	}
	return depth >= next
}

// abyssScheduleNextEvent anchors the next event floor to depth + a random 2-6 gap.
func (b *Bot) abyssScheduleNextEvent(uid string, depth int) {
	// #nosec G404 -- non-cryptographic cadence roll
	next := depth + abyssEventGapMin + rand.IntN(abyssEventGapMax-abyssEventGapMin+1)
	_, _ = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, "abyss_next_event_depth_"+uid, strconv.Itoa(next))
}

// rollFloorCandidates weighted-samples n distinct floor types (without replacement).
// Events are excluded unless allowEvent is set — they ride their own 2-6 floor cadence
// (see abyssEventDue) instead of appearing as a random choice on any floor.
func rollFloorCandidates(n int, allowEvent bool) []floorCandidate {
	weights := map[string]float64{"rest": 0.10, "event": 0.10, "combat": 0.80}
	remaining := []string{"rest", "combat"}
	if allowEvent {
		remaining = []string{"rest", "event", "combat"}
	}
	if n > len(remaining) {
		n = len(remaining)
	}
	chosen := make([]string, 0, n)
	for len(chosen) < n && len(remaining) > 0 {
		total := 0.0
		for _, t := range remaining {
			total += weights[t]
		}
		// #nosec G404
		r := rand.Float64() * total
		acc, pickIdx := 0.0, len(remaining)-1
		for i, t := range remaining {
			acc += weights[t]
			if r < acc {
				pickIdx = i
				break
			}
		}
		chosen = append(chosen, remaining[pickIdx])
		remaining = append(remaining[:pickIdx], remaining[pickIdx+1:]...)
	}
	out := make([]floorCandidate, len(chosen))
	for i, t := range chosen {
		info := floorCandidateInfo[t]
		out[i] = floorCandidate{Index: i, Type: t, Label: info.Label, Icon: info.Icon}
	}
	return out
}

// rollFloorDetail rolls the sub-details for an already-chosen floor type: the
// event subtype for "event" floors, or the 15% chance of a combat modifier for
// "combat" floors. Extracted unchanged from the pre-branching-choice descend roll.
func rollFloorDetail(floorType string) (modifier, eventState string) {
	switch floorType {
	case "event":
		// Special run-structure rooms are selected independently so their share can
		// grow without repeatedly compressing the legacy encounter thresholds.
		// #nosec G404 -- non-cryptographic room selection
		if special := abyssSpecialRoomForRoll(rand.Float64()); special != "" {
			eventState = special
			return
		}
		// Roll one of the mysterious-encounter types. Weighted toward the
		// merchant; the rest split the long tail of shrines, gambles and caches.
		// #nosec G404
		rEv := rand.Float64()
		if rEv < 0.34 {
			g := content.RandomGearDrop()

			// Merchant stock: two consumable stacks with size-dependent pricing.
			// Marshalled instead of Sprintf'd: a quote in an item name can't corrupt the JSON.
			type merchantItem struct {
				Type  string `json:"type"`
				ID    string `json:"id"`
				Name  string `json:"name"`
				Price int64  `json:"price"`
				Count int    `json:"count,omitempty"`
			}
			items := []merchantItem{
				// The gear piece is priced per rarity/power (gearPrice), not flat.
				{Type: "gear", ID: g.ID, Name: g.Name, Price: gearPrice(g)},
			}
			for _, c := range []content.Consumable{content.RandomConsumable(), content.RandomConsumable()} {
				var count int
				// #nosec G404 -- non-cryptographic merchant stock roll
				if c.Type == content.ConsumableHealing || c.Type == content.ConsumableRepair || c.Type == content.ConsumableRevive {
					count = 1 + rand.IntN(5)
				} else {
					count = 1 + rand.IntN(3)
				}
				price := int64(50 * count)
				if c.Type == content.ConsumableBuff {
					price = int64(75 * count)
				}
				name := c.Name
				if count > 1 {
					name = fmt.Sprintf("%s x%d", c.Name, count)
				}
				items = append(items, merchantItem{Type: "cons", ID: c.ID, Name: name, Price: price, Count: count})
			}
			stateBytes, _ := json.Marshal(map[string]any{"type": "merchant", "items": items})
			eventState = string(stateBytes)
		} else if rEv < 0.42 {
			eventState = `{"type":"imp"}`
		} else if rEv < 0.48 {
			eventState = `{"type":"shrine"}`
		} else if rEv < 0.54 {
			eventState = `{"type":"wishing_well"}`
		} else if rEv < 0.59 {
			eventState = `{"type":"gambler"}`
		} else if rEv < 0.64 {
			eventState = `{"type":"statue"}`
		} else if rEv < 0.68 {
			eventState = `{"type":"fountain"}`
		} else if rEv < 0.72 {
			eventState = `{"type":"mimic"}`
		} else if rEv < 0.75 {
			eventState = `{"type":"buried_cache"}`
		} else if rEv < 0.80 {
			eventState = `{"type":"puzzle"}` // #26
		} else if rEv < 0.84 {
			eventState = `{"type":"cursed_library"}` // #30
		} else if rEv < 0.89 {
			eventState = `{"type":"den"}` // #32
		} else if rEv < 0.92 {
			eventState = `{"type":"rift"}` // #35
		} else if rEv < 0.95 {
			eventState = `{"type":"blood_altar"}` // #41
		} else if rEv < 0.98 {
			eventState = `{"type":"alchemy_lab"}` // #43
		} else {
			// Hall of mirrors (#50): three distinct buff reflections rolled now so
			// the choice is fixed the moment the floor exists.
			elixirs := []string{"giant_strength_elixir", "iron_skin_brew", "speed_elixir", "lucky_draught", "intellect_elixir", "strength_elixir"}
			rand.Shuffle(len(elixirs), func(i, j int) { elixirs[i], elixirs[j] = elixirs[j], elixirs[i] }) // #nosec G404
			opts, _ := json.Marshal(elixirs[:3])
			eventState = fmt.Sprintf(`{"type":"mirrors","options":%s}`, string(opts))
		}
	case "combat":
		// #nosec G404
		if rand.Float64() < 0.15 {
			mods := []string{"enraged", "no_healing", "artifact_corrupted", "treasure_goblin", "echo_encounter", "fragile_cache"}
			// #nosec G404
			modifier = mods[rand.IntN(len(mods))]
		}
	}
	return
}

// handleAbyssChooseFloor commits the player's pick from the choice offered by
// handleAbyssDescend, rolls that floor's sub-details, and proceeds exactly as
// a direct descend would have.
func (s *WebServer) handleAbyssChooseFloor(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	run := s.bot.loadAbyssRun(uid)
	if !run.Active {
		writeJSON(w, map[string]any{"ok": false, "error": "not in a run"})
		return
	}
	if run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "you are downed — revive or concede"})
		return
	}

	var req struct {
		Index int `json:"index"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	var pendingRaw sql.NullString
	if err := s.bot.DB.QueryRow("SELECT pending_floor_choice FROM abyss_active WHERE client_uid=$1", uid).Scan(&pendingRaw); err != nil || !pendingRaw.Valid || pendingRaw.String == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "no floor choice pending"})
		return
	}
	var pending pendingFloorChoice
	if err := json.Unmarshal([]byte(pendingRaw.String), &pending); err != nil || req.Index < 0 || req.Index >= len(pending.Candidates) {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid choice"})
		return
	}
	chosen := pending.Candidates[req.Index]

	if _, err := s.bot.DB.Exec("UPDATE abyss_active SET pending_floor_choice=NULL WHERE client_uid=$1", uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	newDepth := run.Depth + 1
	tier, _ := abyssTierByKey(run.Tier)
	modifier, eventState := rollFloorDetail(chosen.Type)
	s.commitFloor(w, uid, run, newDepth, chosen.Type, modifier, eventState, tier, pending.Focus, pending.Interactive)
}

// commitFloor writes the resolved floor (type + rolled details) to abyss_active
// and either returns the rest/event payload or resolves combat immediately.
// Shared by the direct-descend (forced) path and the choose-floor path.
func (s *WebServer) commitFloor(w http.ResponseWriter, uid string, run abyssRun, newDepth int, floorType, modifier, eventState string, tier abyssTier, focus string, interactive bool) {
	eventState = prepareAbyssEventForDepth(eventState, newDepth)
	if floorType == "event" {
		s.bot.abyssScheduleNextEvent(uid, newDepth) // re-anchor the 2-6 floor cadence
	}
	if floorType != "combat" {
		// Store NULL rather than an empty string for floors with no event payload
		// (e.g. rest floors) so the JSONB event_state column accepts the write.
		var evStateArg any
		if eventState != "" {
			evStateArg = eventState
		}
		// pending_floor_choice=NULL clears any choice orphaned by a prior descend that
		// offered a pick the player never took (forced watcher/boss floors bypass the
		// picker), so it can't be replayed by handleAbyssChooseFloor after this commit.
		_, err := s.bot.DB.Exec(
			`UPDATE abyss_active
			    SET depth=$1, floor_type=$2, modifier=$3, event_state=$4, pending_floor_choice=NULL, last_action_at=NOW()
			  WHERE client_uid=$5`,
			newDepth, floorType, modifier, evStateArg, uid,
		)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if floorType == "rest" {
			_, _ = s.bot.DB.Exec("UPDATE users SET abyss_win_streak = 0 WHERE client_uid=$1", uid)
		}
		writeJSON(w, map[string]any{
			"ok":          true,
			"noncombat":   true,
			"floor_type":  floorType,
			"depth":       newDepth,
			"event_state": eventState,
			"escrow":      run.Escrow,
			"risk":        abyssRiskPct(newDepth+1, tier, s.bot.abyssPlayerCR(uid)),
		})
		return
	}

	// Normal Combat floor. pending_floor_choice=NULL discards any uncommitted pick
	// (forced watcher/boss descends reach here without going through the picker) so a
	// stale choice can't be reused by handleAbyssChooseFloor afterwards.
	if _, err := s.bot.DB.Exec("UPDATE abyss_active SET depth=$1, modifier=$2, event_state=NULL, pending_floor_choice=NULL, last_action_at=NOW() WHERE client_uid=$3", newDepth, modifier, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if interactive && s.abyssFeatures.enabled("live_actions", uid) {
		combat, err := s.startAbyssLiveCombat(uid, run, newDepth, tier, modifier, focus)
		if err != nil {
			_, _ = s.bot.DB.Exec("UPDATE abyss_active SET depth=$1, modifier='', event_state=NULL, last_action_at=NOW() WHERE client_uid=$2", run.Depth, uid)
			writeJSON(w, map[string]any{"ok": false, "error": "could not start live combat"})
			return
		}
		writeJSON(w, map[string]any{
			"ok": true, "live_combat": true, "session_id": combat.id,
			"depth": newDepth, "state": combat.snapshotFor(uid),
		})
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

// abyssFloorOutcome carries the per-floor victory bookkeeping results shared by
// the single-descend and batch-descend paths, so both report identical fields
// and cannot drift apart again.
type abyssFloorOutcome struct {
	Bonus          int64
	NewEscrow      int64
	ExpressSkip    bool
	SecondaryGoal  string
	GearMilestone  string
	DailyFirst     bool
	Achievements   []string
	LoreUnlocked   bool
	LoreFragment   string
	RecipeUnlocked string
	AffixReward    string
	DBErr          bool
}

// applyFloorVictory performs all victory bookkeeping for one cleared floor:
// the escrow bonus with every multiplier, interest, momentum/bank-lock ticks,
// gear XP, the daily first-descent bonus, best-depth/win-streak updates,
// artifact leveling, achievements, lore/recipe discovery and the affix
// consumable reward. Used by finishDescend and handleAbyssDescendMulti.
func (s *WebServer) applyFloorVictory(uid string, run abyssRun, depth int, escrowBefore int64, tier abyssTier, modifier, focus string) abyssFloorOutcome {
	st := s.bot.loadAbyssStats(uid)
	var o abyssFloorOutcome

	bonus := abyssFloorBonus(depth, run.depthLevelHint())
	bonus = int64(float64(bonus) * tier.RewardMult * (1.0 + float64(st.UpGreed)*0.05) * abyssPermanentBonus(float64(st.AbyssPrestige)*0.05, 0.50))
	_, dailyMod := s.bot.currentDailyChallenge()
	bonus = int64(float64(bonus) * abyssDailyRewardMult(dailyMod))
	bonus = int64(float64(bonus) * abyssPactRewardMult(s.bot.abyssRunPacts(uid)))
	runFlags := s.bot.loadRunFlags(uid)
	_, weeklyRun := abyssWeeklyRuleFromFlags(runFlags)
	if weeklyRun {
		bonus = bonus * 5 / 4
	}
	if runFlags[abyssRunFlagCatchupCharges] > 0 {
		bonus = bonus * 11 / 10
	}

	switch focus {
	case "gold":
		bonus = bonus * 2
	case "loot":
		bonus = bonus / 2
	}
	var currentHP int
	_ = s.bot.DB.QueryRow("SELECT current_hp FROM users WHERE client_uid=$1", uid).Scan(&currentHP)
	maxHP := s.bot.abyssCombatStats(uid).HP
	if modifier == "fragile_cache" {
		if maxHP > 0 && currentHP*2 >= maxHP {
			bonus = bonus * 13 / 10
			o.SecondaryGoal = "Fragile Cache preserved above 50% health: +30% cache and 1 token"
		}
	}

	// Plunderer specialization (#161): +10% escrow floor bonus.
	if s.bot.abyssSpec(uid) == "plunderer" {
		bonus = bonus * 11 / 10
	}
	// Skill web: escrow_bonus notables and the Voidheart keystone.
	if v := s.bot.treeBonusFor(uid).Pct["escrow_bonus"]; v > 0 {
		bonus = int64(float64(bonus) * (1 + v))
	}
	// Checkpoint starts (#2) trade convenience for ×0.75 rewards.
	if run.CheckpointStart > 0 {
		bonus = bonus * 3 / 4
	}
	// Express elevator (#3): no floor bonus until past the old record.
	if run.ExpressUntil > 0 && depth <= run.ExpressUntil {
		bonus = 0
		o.ExpressSkip = true
	}
	// Momentum (#7) builds each cleared floor; a Last Stand bank lock (#15)
	// ticks down one floor per victory.
	_, _ = s.bot.DB.Exec("UPDATE abyss_active SET momentum = momentum + 1, bank_locked_floors = GREATEST(bank_locked_floors - 1, 0) WHERE client_uid=$1", uid)
	// Gear XP (#108): the wielded weapon remembers its kills.
	o.GearMilestone = s.bot.tickGearXP(uid)

	if s.bot.abyssDailyFirstDescent(uid) {
		bonus = bonus * 3 / 2 // [11] daily first-descent: +50%
		s.bot.grantAbyssTokens(uid, 5)
		o.DailyFirst = true
	}

	hasLuckyCoin := false
	equipped := s.bot.getEquippedItems(uid)
	if _, hasCoin := equipped[content.SlotTrinket1]; hasCoin && equipped[content.SlotTrinket1].ID == "ABYSS_LUCKY_COIN" {
		hasLuckyCoin = true
	}
	newEscrow := int64(float64(escrowBefore)*(1.0+abyssEffectiveInterest(st.UpInterest, hasLuckyCoin))) + bonus // [56] interest + Compounding node
	if _, err := s.bot.DB.Exec("UPDATE abyss_active SET escrow=$1, floor_type='combat', modifier='', event_state=NULL, last_action_at=NOW() WHERE client_uid=$2", newEscrow, uid); err != nil {
		o.DBErr = true
		return o
	}
	if o.SecondaryGoal != "" {
		s.bot.grantAbyssTokens(uid, 1)
	}
	s.bot.advanceAbyssProgression(uid, depth, currentHP, maxHP, modifier, weeklyRun)
	s.bot.tickAbyssRoomEffects(uid)
	_, _ = s.bot.DB.Exec("UPDATE users SET abyss_best_depth = GREATEST(abyss_best_depth, $1) WHERE client_uid=$2", depth, uid)
	_, _ = s.bot.DB.Exec("UPDATE users SET abyss_win_streak = abyss_win_streak + 1 WHERE client_uid=$1", uid)

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

	o.Bonus = bonus
	o.NewEscrow = newEscrow

	// Surface any milestone newly earned this floor: depth, plus boss-kill and
	// bestiary counts (both updated during the fight that just resolved).
	if ach := s.bot.checkDepthAchievements(uid, depth); ach != "" {
		o.Achievements = append(o.Achievements, ach)
	}
	if ach := s.bot.checkBossKillAchievements(uid); ach != "" {
		o.Achievements = append(o.Achievements, ach)
	}
	if ach := s.bot.checkBestiaryAchievements(uid); ach != "" {
		o.Achievements = append(o.Achievements, ach)
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
				o.LoreUnlocked = true
				o.LoreFragment = abyssLoreFragments[fragID]
				// Recipe discovery (#104): fresh lore can carry a crafting secret.
				o.RecipeUnlocked = s.bot.discoverRandomRecipe(uid)
			}
		}
	}

	// Affix consumable reward
	if modifier != "" {
		c := content.RandomConsumable()
		s.bot.grantConsumable(uid, c.ID, c.Duration)
		o.AffixReward = c.Name
	}
	return o
}

// applyFloorDefeat rolls the one-time double-or-nothing revive offer (#15) and
// resets the win streak. Shared by finishDescend and handleAbyssDescendMulti.
// The offer is rolled once and persisted so a refresh can't reroll it.
func (s *WebServer) applyFloorDefeat(uid string, run abyssRun) (canRevive bool) {
	st := s.bot.loadAbyssStats(uid)
	canRevive = !run.Revived
	if canRevive && !run.ReviveLocked {
		offerChance := 0.45 + 0.08*float64(st.UpMercy)
		// #nosec G404 -- non-cryptographic offer roll
		if rand.Float64() >= offerChance {
			canRevive = false
			_, _ = s.bot.DB.Exec("UPDATE abyss_active SET revive_locked=TRUE WHERE client_uid=$1", uid)
		}
	} else if run.ReviveLocked {
		canRevive = false
	}
	_, _ = s.bot.DB.Exec("UPDATE users SET abyss_win_streak = 0 WHERE client_uid=$1", uid)
	return canRevive
}

// finishDescend applies the win/loss bookkeeping shared by descend and revive.
func (s *WebServer) finishDescend(w http.ResponseWriter, uid string, run abyssRun, depth int, escrowBefore int64, tier abyssTier, res abyssFloorResult, modifier string, focus string) {
	writeJSON(w, s.finishDescendData(uid, run, depth, escrowBefore, tier, res, modifier, focus))
}

func (s *WebServer) finishDescendData(uid string, run abyssRun, depth int, escrowBefore int64, tier abyssTier, res abyssFloorResult, modifier string, focus string) map[string]any {
	_, _ = s.bot.DB.Exec("UPDATE users SET abyss_lifetime_floors = abyss_lifetime_floors + 1 WHERE client_uid=$1", uid)

	out := map[string]any{
		"ok": true, "victory": res.Victory, "depth": depth,
		"hp": res.CurrentHP, "max_hp": res.MaxHP,
		"logs": res.LogsHTML, "loot": res.LootHTML, "dura": res.DuraHTML,
		"timeline": res.Timeline, "reward_xp": res.RewardXP,
		"risk": abyssRiskPct(depth+1, tier, s.bot.abyssPlayerCR(uid)),
	}

	if res.Victory {
		o := s.applyFloorVictory(uid, run, depth, escrowBefore, tier, modifier, focus)
		if o.DBErr {
			return map[string]any{"ok": false, "error": "db"}
		}
		out["bonus"] = o.Bonus
		out["escrow"] = o.NewEscrow
		if o.ExpressSkip {
			out["express_skip"] = true
		}
		if o.SecondaryGoal != "" {
			out["secondary_objective"] = o.SecondaryGoal
		}
		if o.GearMilestone != "" {
			out["gear_milestone"] = o.GearMilestone
		}
		if o.DailyFirst {
			out["daily"] = true
		}
		if len(o.Achievements) > 0 {
			out["achievement"] = strings.Join(o.Achievements, " · ")
		}
		if o.LoreUnlocked {
			out["lore_unlocked"] = true
			out["lore_fragment"] = o.LoreFragment
		}
		if o.RecipeUnlocked != "" {
			out["recipe_unlocked"] = o.RecipeUnlocked
		}
		if o.AffixReward != "" {
			out["affix_reward"] = o.AffixReward
		}
	} else {
		// Downed: hold the cache; the player must revive (if available) or concede.
		canRevive := s.applyFloorDefeat(uid, run)
		out["downed"] = true
		out["can_revive"] = canRevive
		out["can_last_stand"] = !run.LastStandUsed && s.bot.abyssTokens(uid) >= abyssLastStandCost(depth)
		out["last_stand_cost"] = abyssLastStandCost(depth)
		out["escrow"] = escrowBefore
		out["insured"] = run.Insured
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	out["gold"] = gold
	out["tokens"] = s.bot.abyssTokens(uid)
	out["consumables"] = s.bot.getConsumables(uid)

	runFinal := s.bot.loadAbyssRun(uid)
	out["auto_focus"] = s.autoSelectFocus(uid, runFinal)
	s.abyssOps.observeFloor(depth, escrowBefore, res, out)

	return out
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
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	if combat, ok := s.liveCombatForUID(uid); ok && !combat.reviveApproved() {
		writeJSON(w, map[string]any{"ok": false, "error": "party revive vote has not reached a majority"})
		return
	}

	run := s.bot.loadAbyssRun(uid)
	if !run.Active || !run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "not downed"})
		return
	}
	if run.Revived {
		writeJSON(w, map[string]any{"ok": false, "error": "revival already used"})
		return
	}
	if run.ReviveLocked {
		writeJSON(w, map[string]any{"ok": false, "error": "the dark offers no gamble this time — Last Stand or concede"})
		return
	}

	stats := s.bot.abyssCombatStats(uid)

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
	focus := s.autoSelectFocus(uid, run)
	res, err := s.bot.fightAbyssFloor(uid, run.Depth, tier, run.Modifier, focus)
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
			"loot": res.LootHTML, "dura": res.DuraHTML, "timeline": res.Timeline,
			"escrow": newEscrow, "doubled": true,
			"reward_xp": res.RewardXP, "risk": abyssRiskPct(run.Depth+1, tier, s.bot.abyssPlayerCR(uid)),
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
	payout, ferr := s.bot.forfeitAbyss(uid, run)
	if ferr != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	out := map[string]any{
		"ok": true, "revived": true, "victory": false, "depth": run.Depth,
		"hp": 0, "logs": res.LogsHTML, "loot": res.LootHTML, "dura": res.DuraHTML,
		"timeline": res.Timeline,
		"forfeited": true, "insured_refund": payout, "escrow": 0,
		"reward_xp": res.RewardXP, "risk": abyssRiskPct(run.Depth+1, tier, s.bot.abyssPlayerCR(uid)),
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
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}

	run := s.bot.loadAbyssRun(uid)
	if !run.Active || !run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "not downed"})
		return
	}
	payout, ferr := s.bot.forfeitAbyss(uid, run)
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
	writeJSON(w, out)
}

// abyssBankTokenGrant is the Abyss-token payout for banking at the given depth,
// boosted by the Tribute node (+10% per level, rounded down) and the skill web's
// token_gain notables/Voidheart keystone. Shared by the bank preview and commit
// paths so both report and grant identical amounts. [44]
func (b *Bot) abyssBankTokenGrant(uid string, depth, upTribute int) int {
	if depth <= 0 {
		return 0
	}
	tokens := depth/5 + 1
	tokens += tokens * upTribute / 10
	if v := b.treeBonusFor(uid).Pct["token_gain"]; v > 0 {
		tokens = int(float64(tokens) * (1 + v))
	}
	return tokens
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
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}

	var req struct {
		Cursed  bool `json:"cursed"`
		Preview bool `json:"preview"`
	}
	// Reject malformed JSON outright: a garbled body decoding to zero values
	// would silently turn a preview request into a real, irreversible bank
	// commit. An absent/empty body (io.EOF) stays valid and means "plain bank".
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	run := s.bot.loadAbyssRun(uid)
	if !run.Active {
		writeJSON(w, map[string]any{"ok": false, "error": "not in a run"})
		return
	}
	if run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "you are downed — revive or concede"})
		return
	}
	// Last Stand seal (#15): the exit stays shut for 2 floors after a token revive.
	if run.BankLockedFloors > 0 {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("the exit is sealed — descend %d more floor(s) first", run.BankLockedFloors)})
		return
	}

	st := s.bot.loadAbyssStats(uid)
	mult := s.bot.abyssBankMultiplier(run.Depth, st.Streak) // [2][12] depth + streak
	payout := int64(float64(run.Escrow) * mult)
	if req.Cursed && payout > 0 {
		payout = payout * 12 / 10 // [9] +20%
	}

	// Preview mode (UX-49): report the itemized payout without committing
	// anything, so the client can show a bank-confirmation breakdown first.
	if req.Preview {
		var dayGold int64
		_ = s.bot.DB.QueryRow(
			"SELECT CASE WHEN abyss_day IS NULL OR abyss_day < CURRENT_DATE THEN 0 ELSE abyss_day_gold END FROM users WHERE client_uid=$1",
			uid).Scan(&dayGold)
		capRemaining := int64(abyssDayGoldCap) - dayGold
		if capRemaining < 0 {
			capRemaining = 0
		}
		capped := payout > capRemaining
		estPayout, estTax := abyssCapTax(payout, capRemaining)
		baseTokens := s.bot.abyssBankTokenGrant(uid, run.Depth, st.UpTribute)
		var lootCount int
		_ = s.bot.DB.QueryRow("SELECT COUNT(*) FROM abyss_escrow_loot WHERE client_uid=$1", uid).Scan(&lootCount)
		writeJSON(w, map[string]any{
			"ok": true, "preview": true,
			"escrow": run.Escrow, "mult": mult, "cursed": req.Cursed,
			"payout": estPayout, "capped": capped, "cap_remaining": capRemaining, "cap_tax": estTax,
			"tokens_grant": baseTokens, "loot_count": lootCount,
			"bonus_gear_eligible": run.Depth >= 10,
			"depth":               run.Depth, "streak": st.Streak,
		})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Apply the per-day gold tax inside the transaction so the day counter and
	// the jackpot feed are only consumed if the gold credit and the rest of the
	// bank commit succeed. [59]
	payout, capTax := s.bot.taxAbyssDayGold(tx, uid, payout)

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
	// End of run: clear the per-run win streak so its combat buff (abyssStreakBuff)
	// can't leak into regular TeamSpeak-cycle fights, which read abyss_win_streak too.
	_, _ = tx.Exec("UPDATE users SET abyss_win_streak = 0 WHERE client_uid=$1", uid)
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
		s.bot.grantAbyssTokens(uid, s.bot.abyssBankTokenGrant(uid, run.Depth, st.UpTribute)) // [44] + Tribute node
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
		// Payout breakdown for the vault subtotal animation (UX-54). When the
		// daily-cap tax shrinks the payout below the raw cache, shrink the base to
		// the paid amount so the parts always sum to the payout and the bonus
		// never goes negative.
		"base":       min(run.Escrow, payout),
		"mult_bonus": max(payout-run.Escrow, 0),
	}
	if capTax > 0 {
		out["cap_tax"] = capTax
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
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}

	var req struct {
		ConsID string `json:"cons_id"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	// Verify they own the consumable, with charges left (mirror abyssOwnedConsumables).
	var rem int
	err := s.bot.DB.QueryRow("SELECT remaining_fights FROM user_consumables WHERE client_uid = $1 AND cons_id = $2 AND remaining_fights > 0 LIMIT 1", uid, req.ConsID).Scan(&rem)
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

	stats := s.bot.abyssCombatStats(uid)

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
		if req.ConsID == "repair_kit_ii" {
			repairAmt = 50
		}
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
		_, _ = s.bot.DB.Exec("UPDATE abyss_active SET momentum = 0 WHERE client_uid=$1", uid) // #7 momentum breaks on consumable use
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
	case content.ConsumableRevive:
		// Unlike handleAbyssRevive's double-or-nothing gamble, this is a plain
		// heal-and-continue — an *extra* revive beyond the normal one-per-run,
		// so it deliberately does not touch abyss_active.revived. Downed is
		// derived from CurHP<=0, so healing above 0 clears it on its own.
		run := s.bot.loadAbyssRun(uid)
		if !run.Active || !run.Downed {
			writeJSON(w, map[string]any{"ok": false, "error": "you are not downed"})
			return
		}
		_, _ = s.bot.DB.Exec("UPDATE users SET current_hp = $1 WHERE client_uid = $2", stats.HP, uid)
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
	_, _ = s.bot.DB.Exec("UPDATE abyss_active SET momentum = 0 WHERE client_uid=$1", uid) // #7 momentum breaks on consumable use

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
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}

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
		// Sanctuary upgrades (#38): permanent rest-floor perks bought with tokens.
		sanct := s.bot.loadSanctuary(uid)
		restCost := func(base int64, key string) int64 {
			c := base - base*int64(sanct[key])*25/100
			if c < 1 {
				c = 1
			}
			return c
		}
		switch req.Action {
		case "forge_station": // #113 — free full repair, unlocked by the Crafting Station upgrade
			if sanct["forge"] <= 0 {
				writeJSON(w, map[string]any{"ok": false, "error": "buy the Crafting Station sanctuary upgrade first"})
				return
			}
			s.bot.ensureGearMaxDurability(uid)
			if _, err := s.bot.DB.Exec("UPDATE user_gear SET durability = "+gearMaxDurExpr+" WHERE client_uid=$1", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "msg": "⚒️ The sanctuary's crafting station hums — all gear repaired, free of charge.", "gold": gold})
			return

		case "heal":
			cost := restCost(100, "heal")
			stats := s.bot.abyssCombatStats(uid)
			if run.CurHP >= stats.HP {
				writeJSON(w, map[string]any{"ok": false, "error": "already at full health"})
				return
			}
			if gold < cost {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			var newGold int64
			err := s.bot.DB.QueryRow("UPDATE users SET gold = gold - $1, current_hp = $2 WHERE client_uid = $3 AND gold >= $1 RETURNING gold", cost, stats.HP, uid).Scan(&newGold)
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "msg": "Healed to full!", "gold": newGold, "hp": stats.HP})
			return

		case "repair":
			cost := restCost(100, "repair")
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
			var newGold int64
			err = tx.QueryRow("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1 RETURNING gold", cost, uid).Scan(&newGold)
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			if err != nil {
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
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "msg": "All gear fully repaired!", "gold": newGold})
			return

		case "reroll_lowest_skill", "reroll_highest_skill", "reroll_highest_skill_same_tier":
			var cost int64
			switch req.Action {
			case "reroll_lowest_skill":
				cost = 100
			case "reroll_highest_skill_same_tier":
				cost = 200
			default:
				cost = 150
			}
			if gold < cost {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}

			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()

			// Load player's active skills to select target
			rows, err := tx.Query("SELECT slot, skill_id FROM user_skills WHERE client_uid = $1", uid)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			type activeSkill struct {
				slot   int
				id     string
				score  int
				rarity content.Rarity
			}
			var active []activeSkill
			for rows.Next() {
				var slot int
				var skID string
				if err := rows.Scan(&slot, &skID); err == nil {
					if sk, ok := content.GetSkillByID(skID); ok {
						score := int(sk.Rarity)*10000 + sk.Score()
						active = append(active, activeSkill{slot: slot, id: skID, score: score, rarity: sk.Rarity})
					}
				}
			}
			_ = rows.Close()

			if len(active) == 0 {
				writeJSON(w, map[string]any{"ok": false, "error": "you have no active skills to re-roll"})
				return
			}

			// Find target skill
			target := active[0]
			if req.Action == "reroll_lowest_skill" {
				for _, sk := range active {
					if sk.score < target.score {
						target = sk
					}
				}
			} else {
				// reroll_highest_skill or reroll_highest_skill_same_tier
				for _, sk := range active {
					if sk.score > target.score {
						target = sk
					}
				}
			}

			// Roll new skill, re-rolling (bounded) when the redraw lands on the
			// identical skill — a paid reroll should always change something.
			var newSk content.Skill
			for tries := 0; tries < 5; tries++ {
				if req.Action == "reroll_highest_skill_same_tier" {
					newSk = content.RandomSkillOfRarity(target.rarity)
				} else {
					newSk = content.RandomSkill()
				}
				if newSk.ID != target.id {
					break
				}
			}

			// Charge gold
			var newGold int64
			err = tx.QueryRow("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1 RETURNING gold", cost, uid).Scan(&newGold)
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}

			// Replace the single target skill
			if _, err := tx.Exec("UPDATE user_skills SET skill_id = $1 WHERE client_uid = $2 AND slot = $3", newSk.ID, uid, target.slot); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}

			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("Skill in slot %d re-rolled to %s!", target.slot, newSk.Name), "gold": newGold})
			return
		}
	case "event":
		if s.handleAbyssSpecialRoom(w, uid, run, req.Action) {
			return
		}
		var state struct {
			Type  string `json:"type"`
			Items []struct {
				Type  string `json:"type"`
				ID    string `json:"id"`
				Name  string `json:"name"`
				Price int64  `json:"price"`
				Count int64  `json:"count"`
			} `json:"items"`
			Options []string `json:"options"` // hall-of-mirrors buff choices (#50)
		}
		if err := json.Unmarshal([]byte(run.EventState), &state); err != nil {
			// Corrupt event state would make every action below report "wrong floor
			// type" and soft-lock the floor — clear it so the player can proceed.
			_, _ = s.bot.DB.Exec("UPDATE abyss_active SET event_state = NULL, last_action_at = NOW() WHERE client_uid = $1", uid)
			writeJSON(w, map[string]any{"ok": true, "resolved": true})
			return
		}

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
			// Resolve the catalog entry BEFORE any gold moves: a gear piece that no
			// longer resolves (or an unknown consumable) must reject, never silently
			// debit for nothing.
			var buyGear content.Gear
			var buyCons content.Consumable
			if item.Type == "gear" {
				g, ok := content.GetGearByID(item.ID)
				if !ok {
					writeJSON(w, map[string]any{"ok": false, "error": "that item is no longer available"})
					return
				}
				buyGear = g
			} else {
				c, ok := content.GetConsumableByID(item.ID)
				if !ok {
					writeJSON(w, map[string]any{"ok": false, "error": "that item is no longer available"})
					return
				}
				buyCons = c
			}
			if gold < item.Price {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			var newGold int64
			err := s.bot.DB.QueryRow("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1 RETURNING gold", item.Price, uid).Scan(&newGold)
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if item.Type == "gear" {
				s.bot.awardGearDrop(uid, buyGear)
			} else {
				count := int(item.Count)
				if count <= 0 {
					count = 1
				}
				fights := buyCons.Duration
				if fights <= 0 {
					fights = 1
				}
				s.bot.grantConsumable(uid, buyCons.ID, fights*count)
			}
			state.Items = append(state.Items[:idx], state.Items[idx+1:]...)
			newStateBytes, _ := json.Marshal(state)
			_, _ = s.bot.DB.Exec("UPDATE abyss_active SET event_state = $1, last_action_at = NOW() WHERE client_uid = $2", string(newStateBytes), uid)

			writeJSON(w, map[string]any{"ok": true, "msg": "Bought " + item.Name + "!", "gold": newGold, "event_state": string(newStateBytes)})
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
			// The wager debit, reward and event-state clear are one gamble: run them
			// in a single transaction (like well_toss) so a failed clear can't leave
			// the imp replayable after gold already changed hands.
			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()
			var newGold int64
			err = tx.QueryRow("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1 RETURNING gold", cost, uid).Scan(&newGold)
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}

			// #nosec G404
			rRoll := rand.Float64()
			var msg string
			var consPrize content.Consumable
			if rRoll < 0.40 {
				msg = "The Imp giggles and steals your gold! Got nothing."
			} else if rRoll < 0.75 {
				msg = "Dice rolled! You doubled your wager! (+600 gold)"
				if err := tx.QueryRow("UPDATE users SET gold = gold + 600 WHERE client_uid = $1 RETURNING gold", uid).Scan(&newGold); err != nil {
					writeJSON(w, map[string]any{"ok": false, "error": "db"})
					return
				}
			} else if rRoll < 0.95 {
				consPrize = content.RandomConsumable()
				msg = "The Imp drops a consumable: " + consPrize.Name + "!"
			} else {
				ui := content.RandomUniqueItem()
				msg = "JACKPOT! The Imp drops a Unique Item: " + ui.Name + "!"
				if _, err := tx.Exec("INSERT INTO user_unique_items (client_uid, item_name, rarity, power) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING", uid, ui.Name, ui.Rarity, ui.Power); err != nil {
					writeJSON(w, map[string]any{"ok": false, "error": "db"})
					return
				}
			}

			if _, err := tx.Exec("UPDATE abyss_active SET event_state = NULL, last_action_at = NOW() WHERE client_uid = $1", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if consPrize.ID != "" {
				s.bot.grantConsumable(uid, consPrize.ID, consPrize.Duration)
			}
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
			var newGold int64
			err = tx.QueryRow("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1 RETURNING gold", cost, uid).Scan(&newGold)
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
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
			writeJSON(w, map[string]any{"ok": true, "msg": msg, "gold": newGold, "escrow": newEscrow, "resolved": true})
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
			stats := s.bot.abyssCombatStats(uid)
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
			stats := s.bot.abyssCombatStats(uid)
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
			stats := s.bot.abyssCombatStats(uid)
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

		case "puzzle_pick": // #26 — pick 1 of 3 chests; the right one pays, wrong ones nip.
			if state.Type != "puzzle" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for puzzle_pick"})
				return
			}
			var idx int
			_, _ = fmt.Sscan(req.Payload, &idx)
			if idx < 0 || idx > 2 {
				writeJSON(w, map[string]any{"ok": false, "error": "pick chest 0, 1 or 2"})
				return
			}
			// The answer is derived, not stored, so the client-visible event state
			// can't leak it.
			correct := abyssPuzzleAnswer(uid, run.Depth, run.StartedAt)
			if idx == correct {
				gain := int64(150 * (run.Depth + 1))
				newEscrow := run.Escrow + gain
				if _, err := s.bot.DB.Exec("UPDATE abyss_active SET escrow=$1, event_state=NULL, last_action_at=NOW() WHERE client_uid=$2", newEscrow, uid); err != nil {
					writeJSON(w, map[string]any{"ok": false, "error": "db"})
					return
				}
				writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("🧩 The chest clicks open — +%d gold sealed into your cache!", gain), "escrow": newEscrow, "resolved": true})
				return
			}
			stats := s.bot.abyssCombatStats(uid)
			var curHP int
			_ = s.bot.DB.QueryRow("SELECT current_hp FROM users WHERE client_uid=$1", uid).Scan(&curHP)
			newHP := curHP - stats.HP/10
			if newHP < 1 {
				newHP = 1
			}
			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()
			if _, err := tx.Exec("UPDATE users SET current_hp=$1 WHERE client_uid=$2", newHP, uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if _, err := tx.Exec("UPDATE abyss_active SET event_state=NULL, last_action_at=NOW() WHERE client_uid=$1", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "msg": "🧩 A needle trap! Wrong chest — the right one seals itself forever.", "hp": newHP, "resolved": true})
			return

		case "library_trade": // #30 — blood for knowledge.
			if state.Type != "cursed_library" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for library_trade"})
				return
			}
			stats := s.bot.abyssCombatStats(uid)
			var curHP int
			_ = s.bot.DB.QueryRow("SELECT current_hp FROM users WHERE client_uid=$1", uid).Scan(&curHP)
			newHP := curHP - stats.HP*15/100
			if newHP < 1 {
				newHP = 1
			}
			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()
			if _, err := tx.Exec("UPDATE users SET current_hp=$1 WHERE client_uid=$2", newHP, uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			// #nosec G404 -- non-cryptographic lore roll
			fragID := 1 + rand.IntN(10)
			_, _ = tx.Exec("INSERT INTO abyss_lore_unlocked (client_uid, lore_id) VALUES ($1,$2) ON CONFLICT DO NOTHING", uid, fragID)
			if _, err := tx.Exec("UPDATE abyss_active SET event_state=NULL, last_action_at=NOW() WHERE client_uid=$1", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			elixirFights := 1
			if ec, ok := content.GetConsumableByID("intellect_elixir"); ok && ec.Duration > 0 {
				elixirFights = ec.Duration
			}
			s.bot.grantConsumable(uid, "intellect_elixir", elixirFights)
			msg := "📚 The pages drink your blood and whisper a lore fragment. An Intellect Elixir slips from the shelf."
			if recipe := s.bot.discoverRandomRecipe(uid); recipe != "" {
				msg += " 📖 Recipe discovered: " + recipe + "!"
			}
			writeJSON(w, map[string]any{"ok": true, "msg": msg, "hp": newHP, "resolved": true})
			return

		case "den_dice", "den_card", "den_wheel", "den_longshot", "den_cascade": // #32 — a den of many odds.
			if state.Type != "den" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for the gambling den"})
				return
			}
			// One game per den: pick a single wager and the den closes (was an infinite
			// gold sink). Stake, payout and the event-state clear commit together so a
			// failed clear can't leave the den replayable after gold changed hands.
			var stake, prize int64
			var winP float64
			var label string
			switch req.Action {
			case "den_dice":
				stake, prize, winP, label = 300, 600, 0.50, "🎲 Dice"
			case "den_card":
				stake, prize, winP, label = 300, 900, 0.33, "🃏 High Card"
			case "den_wheel":
				stake, prize, winP, label = 500, 4000, 0.10, "🎡 Wheel"
			case "den_longshot":
				stake, prize, winP, label = 200, 4000, 0.05, "🎯 Long Shot"
			default:
				stake, prize, winP, label = 400, 600, 0.75, "🪙 Coin Cascade"
			}
			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()
			res, err := tx.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid=$2 AND gold >= $1", stake, uid)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if n, _ := res.RowsAffected(); n == 0 {
				// Can't afford this game: leave the den open so they can pick a cheaper
				// one or proceed. (No gold moved, nothing to resolve.)
				writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
				return
			}
			newGold := gold - stake
			var msg string
			// #nosec G404 -- non-cryptographic gambling roll
			if rand.Float64() < winP {
				if err := tx.QueryRow("UPDATE users SET gold = gold + $1 WHERE client_uid=$2 RETURNING gold", prize, uid).Scan(&newGold); err != nil {
					writeJSON(w, map[string]any{"ok": false, "error": "db"})
					return
				}
				msg = fmt.Sprintf("%s: WINNER! +%d gold.", label, prize)
			} else {
				msg = fmt.Sprintf("%s: the house takes your %d gold.", label, stake)
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

		case "altar_sacrifice": // #41 — feed the altar a consumable for a 3-fight surge.
			if state.Type != "blood_altar" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for altar_sacrifice"})
				return
			}
			consID := strings.TrimSpace(req.Payload)
			var rem int
			if err := s.bot.DB.QueryRow("SELECT remaining_fights FROM user_consumables WHERE client_uid=$1 AND cons_id=$2 LIMIT 1", uid, consID).Scan(&rem); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "you do not own that consumable"})
				return
			}
			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()
			res, err := tx.Exec("UPDATE user_consumables SET remaining_fights = remaining_fights - 1 WHERE client_uid=$1 AND cons_id=$2 AND remaining_fights > 1", uid, consID)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if n, _ := res.RowsAffected(); n == 0 {
				if _, err := tx.Exec("DELETE FROM user_consumables WHERE client_uid=$1 AND cons_id=$2", uid, consID); err != nil {
					writeJSON(w, map[string]any{"ok": false, "error": "db"})
					return
				}
			}
			if _, err := tx.Exec("UPDATE abyss_active SET event_state=NULL, last_action_at=NOW() WHERE client_uid=$1", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			// The altar answers with a mighty elixir, already burning (3 fights).
			buffs := []string{"giant_strength_elixir", "iron_skin_brew", "speed_elixir"}
			pick := buffs[rand.IntN(len(buffs))] // #nosec G404
			s.bot.grantConsumable(uid, pick, 3)
			bName := pick
			if c, ok := content.GetConsumableByID(pick); ok {
				bName = c.Name
			}
			writeJSON(w, map[string]any{"ok": true, "msg": "🩸 The altar drinks deep… and answers: " + bName + " surges through you for the next 3 fights!", "resolved": true, "consumables": s.bot.getConsumables(uid)})
			return

		case "lab_combine": // #43 — two consumables in, one better one out.
			if state.Type != "alchemy_lab" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for lab_combine"})
				return
			}
			parts := strings.Split(req.Payload, ",")
			if len(parts) != 2 {
				writeJSON(w, map[string]any{"ok": false, "error": "pick two consumables"})
				return
			}
			id1, id2 := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			c1, ok1 := content.GetConsumableByID(id1)
			c2, ok2 := content.GetConsumableByID(id2)
			if !ok1 || !ok2 {
				writeJSON(w, map[string]any{"ok": false, "error": "unknown consumable"})
				return
			}
			tx, err := s.bot.DB.Begin()
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			defer func() { _ = tx.Rollback() }()
			for _, cid := range []string{id1, id2} {
				res, err := tx.Exec("UPDATE user_consumables SET remaining_fights = remaining_fights - 1 WHERE client_uid=$1 AND cons_id=$2 AND remaining_fights > 1", uid, cid)
				if err != nil {
					writeJSON(w, map[string]any{"ok": false, "error": "db"})
					return
				}
				if n, _ := res.RowsAffected(); n == 0 {
					del, err := tx.Exec("DELETE FROM user_consumables WHERE client_uid=$1 AND cons_id=$2", uid, cid)
					if err != nil {
						writeJSON(w, map[string]any{"ok": false, "error": "db"})
						return
					}
					if n, _ := del.RowsAffected(); n == 0 {
						writeJSON(w, map[string]any{"ok": false, "error": "you do not own both consumables"})
						return
					}
				}
			}
			if _, err := tx.Exec("UPDATE abyss_active SET event_state=NULL, last_action_at=NOW() WHERE client_uid=$1", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if err := tx.Commit(); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			// Result: matching types distill upward, mismatches make a random elixir.
			var resultID string
			switch {
			case c1.Type == content.ConsumableHealing && c2.Type == content.ConsumableHealing:
				resultID = "rejuvenation_potion"
			case c1.Type == content.ConsumableRepair && c2.Type == content.ConsumableRepair:
				resultID = "master_repair_kit"
			default:
				elixirs := []string{"strength_elixir", "iron_skin_brew", "speed_elixir", "lucky_draught", "intellect_elixir"}
				resultID = elixirs[rand.IntN(len(elixirs))] // #nosec G404
			}
			rName := resultID
			rFights := 1
			if c, ok := content.GetConsumableByID(resultID); ok {
				rName = c.Name
				if c.Duration > 0 {
					rFights = c.Duration
				}
			}
			s.bot.grantConsumable(uid, resultID, rFights)
			writeJSON(w, map[string]any{"ok": true, "msg": "⚗️ The mixture bubbles, flares… and settles: " + rName + "!", "resolved": true, "consumables": s.bot.getConsumables(uid)})
			return

		case "mirrors_pick": // #50 — choose one reflection, exact numbers shown client-side.
			if state.Type != "mirrors" {
				writeJSON(w, map[string]any{"ok": false, "error": "wrong floor type for mirrors_pick"})
				return
			}
			var idx int
			_, _ = fmt.Sscan(req.Payload, &idx)
			if idx < 0 || idx >= len(state.Options) {
				writeJSON(w, map[string]any{"ok": false, "error": "invalid reflection"})
				return
			}
			pick := state.Options[idx]
			c, ok := content.GetConsumableByID(pick)
			if !ok || c.Type != content.ConsumableBuff {
				writeJSON(w, map[string]any{"ok": false, "error": "the mirror shatters — invalid reflection"})
				return
			}
			if _, err := s.bot.DB.Exec("UPDATE abyss_active SET event_state=NULL, last_action_at=NOW() WHERE client_uid=$1", uid); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			s.bot.grantConsumable(uid, pick, 3) // active for the next 3 fights
			writeJSON(w, map[string]any{"ok": true, "msg": "🪞 Your reflection steps into you: " + c.Name + " burns bright for 3 fights.", "resolved": true, "consumables": s.bot.getConsumables(uid)})
			return
		}
	}

	writeJSON(w, map[string]any{"ok": false, "error": "invalid action"})
}

var puzzleSecret []byte
var puzzleSecretOnce sync.Once

func getPuzzleSecret() []byte {
	puzzleSecretOnce.Do(func() {
		const filename = "puzzle_secret.key"
		data, err := os.ReadFile(filename)
		if err == nil && len(data) >= 16 {
			puzzleSecret = data
			return
		}
		// Generate new secret
		secret := make([]byte, 32)
		_, _ = crand.Read(secret)
		_ = os.WriteFile(filename, secret, 0600)
		puzzleSecret = secret
	})
	return puzzleSecret
}

// abyssPuzzleAnswer derives the puzzle floor's correct chest (0-2) from stable
// run facts using a server-side secret key so it cannot be predicted by the client.
func abyssPuzzleAnswer(uid string, depth int, startedAt time.Time) int {
	secret := getPuzzleSecret()
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(uid))
	var depthBuf [8]byte
	binary.BigEndian.PutUint64(depthBuf[:], uint64(depth))
	_, _ = mac.Write(depthBuf[:])
	var timeBuf [8]byte
	binary.BigEndian.PutUint64(timeBuf[:], uint64(startedAt.Unix()))
	_, _ = mac.Write(timeBuf[:])
	sum := mac.Sum(nil)
	h := binary.BigEndian.Uint64(sum[:8])
	return int(h % 3)
}

// handleAbyssNonCombatProceed leaves the Rest/Event floor and returns to the lobby.
func (s *WebServer) handleAbyssNonCombatProceed(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}

	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.FloorType == "combat" {
		writeJSON(w, map[string]any{"ok": false, "error": "not on a non-combat floor"})
		return
	}

	st := s.bot.loadAbyssStats(uid)
	tier, _ := abyssTierByKey(run.Tier)
	bonus := abyssFloorBonus(run.Depth, run.depthLevelHint())
	
	focus := s.autoSelectFocus(uid, run)

	// The xp/materials/tokens focuses trade the gold floor bonus for a matching
	// reward, mirroring what they do on combat floors — never for nothing.
	focusReward := ""
	switch focus {
	case "gold":
		bonus = bonus * 2
	case "loot":
		bonus = bonus / 2
	case "xp":
		bonus = 0
		xpGain := 5 + rand.IntN(10) // #nosec G404 -- non-cryptographic reward roll
		// Skill web: apply the same xp_gain bonus combat floor XP gets.
		if v := s.bot.treeBonusFor(uid).Pct["xp_gain"]; v > 0 {
			xpGain = int(float64(xpGain) * (1 + v))
		}
		if lr, _ := s.bot.awardXP(uid, "", xpGain); lr != nil && lr.NewLevel >= PrestigeThreshold {
			s.bot.doPrestige(uid)
		}
		focusReward = fmt.Sprintf("✨ +%d XP", xpGain)
	case "materials":
		bonus = 0
		mat, n := "shard", 2+rand.IntN(3) // #nosec G404 -- non-cryptographic reward roll
		if run.Depth >= 50 {
			mat, n = "core", 1+rand.IntN(2) // #nosec G404
		}
		if s.bot.escrowAbyssLoot(uid, fmt.Sprintf("⛏️ Material Drop: %s ×%d", abyssMaterialName(mat), n), abyssLootGrant{Type: "mat", MatID: mat, MatN: n}) {
			focusReward = fmt.Sprintf("⛏️ %s ×%d sealed into the cache", abyssMaterialName(mat), n)
		}
	case "tokens":
		bonus = 0
		tks := int64(1 + rand.IntN(2)) // #nosec G404 -- non-cryptographic reward roll
		if s.bot.escrowAbyssLoot(uid, fmt.Sprintf("🜲 %d Abyss Tokens", tks), abyssLootGrant{Type: "tokens", Tokens: tks}) {
			focusReward = fmt.Sprintf("🜲 %d tokens sealed into the cache", tks)
		}
	}
	// Apply tier reward multiplier to match combat floor scaling
	bonus = int64(float64(bonus) * tier.RewardMult)
	bonus = int64(float64(bonus) * (1.0 + float64(st.UpGreed)*0.05) * abyssPermanentBonus(float64(st.AbyssPrestige)*0.05, 0.50))
	_, dailyMod := s.bot.currentDailyChallenge()
	bonus = int64(float64(bonus) * abyssDailyRewardMult(dailyMod))
	bonus = int64(float64(bonus) * abyssPactRewardMult(s.bot.abyssRunPacts(uid)))
	if _, weekly := abyssWeeklyRuleFromFlags(s.bot.loadRunFlags(uid)); weekly {
		bonus = bonus * 5 / 4
	}
	
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
		"focus_reward": focusReward,
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
// applies to every cleared floor today.
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
// applies to every combat floor today.
func abyssDailyDangerMult(dailyMod string) float64 {
	if dailyMod == "glass_cannon" {
		return 1.3
	}
	return 1.0
}

// countEquippedAbyssGearBySet buckets equipped Abyss-exclusive gear by its
// EffectiveSetID (named set, or "abyss_legacy" for untagged items) so true
// per-collection set bonuses can be computed alongside the original flat set.
func (b *Bot) countEquippedAbyssGearBySet(uid string) map[string]int {
	counts := make(map[string]int)
	rows, err := b.DB.Query("SELECT gear_id, item_data FROM user_gear WHERE client_uid=$1 AND gear_id LIKE 'ABYSS\\_%'", uid)
	if err != nil {
		return counts
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var gearID string
		var itemData sql.NullString
		if err := rows.Scan(&gearID, &itemData); err != nil {
			continue
		}
		if g, ok := b.makeGear(gearID, itemData); ok {
			counts[g.EffectiveSetID()]++
		}
	}
	return counts
}

func (b *Bot) loadCoopHelpers(uid string) []map[string]any {
	return b.loadCoopHelpersFiltered(uid, "", "")
}

func (b *Bot) loadCoopHelpersFiltered(uid, pace, difficulty string) []map[string]any {
	switch pace {
	case "fast", "standard", "deliberate":
	default:
		pace = ""
	}
	switch difficulty {
	case "normal", "nightmare", "hell", "insanity":
	default:
		difficulty = ""
	}
	rows, err := b.DB.Query(
		`SELECT client_uid, COALESCE(NULLIF(nickname, ''), 'Adventurer') AS nick, abyss_best_depth, last_seen
		   FROM users
		  WHERE client_uid != $1 AND abyss_best_depth > 0
		  ORDER BY last_seen DESC
		  LIMIT 6`, uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []map[string]any
	for rows.Next() {
		var cuid, nick string
		var depth int
		var lastSeen time.Time
		if err := rows.Scan(&cuid, &nick, &depth, &lastSeen); err == nil {
			age := time.Since(lastSeen)
			if (pace == "fast" && age > 15*time.Minute) || (pace == "standard" && age > 2*time.Hour) {
				continue
			}
			if (difficulty == "nightmare" && depth < 15) || (difficulty == "hell" && depth < 30) || (difficulty == "insanity" && depth < 50) {
				continue
			}
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
	helpers := s.bot.loadCoopHelpersFiltered(uid, r.URL.Query().Get("pace"), r.URL.Query().Get("difficulty"))
	writeJSON(w, map[string]any{"ok": true, "helpers": helpers})
}

func (s *WebServer) handleAbyssCoopInvite(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}

	var req struct {
		CoopUID string `json:"coop_uid"`
	}
	_ = readJSON(r, &req)

	run := s.bot.loadAbyssRun(uid)
	if !run.Active {
		writeJSON(w, map[string]any{"ok": false, "error": "not in a run"})
		return
	}
	if (run.Depth+1)%abyssBossEvery != 0 {
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
	// Prestiging mid-run would silently forfeit the escrow and orphan the locked
	// loot — require the player to finish or abandon the run first.
	if s.bot.loadAbyssRun(uid).Active {
		writeJSON(w, map[string]any{"ok": false, "error": "finish or abandon your run first"})
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
	ApplyPrestigeMemory(s.bot, uid)

	out := map[string]any{"ok": true, "prestige": st.AbyssPrestige + 1}
	if ach := s.bot.checkThresholdAchievements(uid, 1, []achTier{{1, "prestige_1"}}); ach != "" {
		out["achievement"] = ach
	}
	writeJSON(w, out)
}

// autoSelectFocus dynamically determines the best next-floor focus for a player based on their stats, pity, and gear status.
func (s *WebServer) autoSelectFocus(uid string, run abyssRun) string {
	nextDepth := run.Depth + 1
	if nextDepth%abyssBossEvery == 0 {
		return "loot"
	}

	var gold, tokens int64
	_ = s.bot.DB.QueryRow("SELECT gold, abyss_tokens FROM users WHERE client_uid=$1", uid).Scan(&gold, &tokens)

	// Crafting materials live in the user_materials table.
	mats := s.bot.loadMaterials(uid)
	shard, core := mats["shard"], mats["core"]

	equipped := s.bot.getEquippedItems(uid)
	lowDura := false
	equippedRows, err := s.bot.DB.Query("SELECT slot, durability FROM user_gear WHERE client_uid = $1", uid)
	if err == nil {
		defer func() { _ = equippedRows.Close() }()
		for equippedRows.Next() {
			var slot string
			var dur int
			if equippedRows.Scan(&slot, &dur) == nil {
				slotEnum := content.GearSlot(slot)
				if item, ok := equipped[slotEnum]; ok {
					if item.MaxDurability > 0 && float64(dur)/float64(item.MaxDurability) < 0.25 {
						lowDura = true
					}
				}
			}
		}
	}

	if lowDura || gold < 5000 {
		return "gold"
	}

	var legendaryPity int
	_ = s.bot.DB.QueryRow("SELECT legendary_pity FROM users WHERE client_uid=$1", uid).Scan(&legendaryPity)
	if legendaryPity >= 30 {
		return "loot"
	}

	if tokens < 15 {
		return "tokens"
	}

	if shard < 15 || core < 5 {
		return "materials"
	}

	var userXP, userLevel int
	_ = s.bot.DB.QueryRow("SELECT xp, level FROM users WHERE client_uid=$1", uid).Scan(&userXP, &userLevel)
	if userLevel < PrestigeThreshold {
		reqXP := leveling.XPForLevel(userLevel + 1)
		baseXP := leveling.XPForLevel(userLevel)
		if reqXP > baseXP && float64(reqXP-userXP)/float64(reqXP-baseXP) <= 0.15 {
			return "xp"
		}
	}

	return "balanced"
}

