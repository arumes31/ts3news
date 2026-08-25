package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"sort"
	"strings"
	"time"

	"ts3news/internal/clientquery"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
	"ts3news/internal/leveling"
)

// XP-modifier tuning constants.
const (
	critChance          = 0.05
	critMult            = 3.0
	partyMult           = 1.25
	serverMultPerUser   = 0.05
	serverMultCap       = 2.0
	noGamePenalty       = 0.5
	dailyLoginXP        = 5
	lootBoxEveryLevels  = 25
	lootBoxMin          = 50
	lootBoxMax          = 500
	slothGraceDays      = 7
	slothDailyDecay     = 0.02
	artifactChance      = 0.01
	titleChance         = 0.01
	gearChance          = 0.10
	consChance          = 0.1
	enchChance          = 0.02
	skillChance         = 0.05
	uniqueItemChance    = 0.01  // 1% chance per loot roll
	ultimateSkillChance = 0.005 // 0.5% chance per loot roll
	duraLossPerFight    = 1
	duraLossPenalty     = 3
	occupiedSlotRare    = 0.1
	deathXPPenalty      = 0.05 // 5% XP loss on death
)

// UserInCombat is a character's resolved combat-ready state for one fight:
// stats, equipped gear, skills, pets, and per-fight modifiers.
type UserInCombat struct {
	UID           string
	Nickname      string
	CLID          int
	Level         int
	Stats         content.Stats
	Skills        []content.Skill
	Ultimates     []*content.UltimateSkill // up to maxActiveUltimates, no duplicates
	CurrentHP     int
	DamageTaken   int // per-fight incoming/self damage; used by authoritative Abyss perfect-run tracking
	RegenStacks   int
	Gold          int64
	Pets          []*content.Mob
	Equipped      map[content.GearSlot]content.Gear
	Position      content.Position
	STRMod        float64
	DEFMod        float64
	SPDMod        float64
	LootFocus     string // Auto-selected per floor: "balanced", "gold", "loot", "xp", "materials" or "tokens"
	FloorModifier string
	IsClone       bool // If true, DB updates are skipped (for co-op)
	// EscrowLoot suppresses inline loot application in the combat engine. The Abyss
	// sets this so drops are not granted mid-run; instead they are rolled into the
	// run's loot escrow (locked until banked, lost on death). See web_abyss_loot.go.
	EscrowLoot bool
	// treeBonus carries the Abyss skill-web bonus into combat. Only buildAbyssUser
	// populates it; regular channel fights leave it zero so the Abyss tree never
	// leaks into non-Abyss combat (and treeBonusFor's DB cost stays off that path).
	treeBonus content.TreeBonus
	// killerExp is an Abyss-only per-mob-tier damage bonus in tenths of a
	// percent. Keeping it on the combatant lets mixed waves apply grudges only
	// to the matching target family.
	killerExp map[string]int
	// live is set only for interactive Abyss fights. It pauses the shared combat
	// engine at player phases and supplies the party's submitted actions.
	live *abyssLiveCombat
	// petHealEnabled carries the owner's per-companion autoskill preference into
	// the fight snapshot. Missing entries preserve the legacy enabled behavior.
	petHealEnabled map[string]bool
	// abyssSupport is a run-scoped rescued delver. It is deliberately not a
	// UserInCombat or persisted pet: it can assist and appear in the live ally
	// roster without receiving player actions, loot, HP writes, or pet progression.
	abyssSupport *abyssRescueSupport
}

func abyssKillerDamage(base int, user *UserInCombat, mob *content.Mob) int {
	if base <= 0 || user == nil || mob == nil {
		return base
	}
	bonus := min(max(user.killerExp[string(mob.Type)], 0), abyssKillerExpCap)
	damage := base + base*bonus/1000
	if familyBonus := user.treeBonus.Pct["bestiary_damage_"+strings.ToLower(string(mob.Type))]; familyBonus > 0 {
		damage += int(float64(damage) * familyBonus)
	}
	return damage
}

type activeUser struct {
	u                   *UserInCombat
	effects             []content.ItemEffect
	lastSkillID         string
	skillRepeatCount    int
	skillCooldowns      map[string]int
	Stunned             bool // scripted boss-phase stun: skips this user's next turn
	CurrentMana         int
	MaxMana             int
	petCooldowns        map[int]int // Independent active-pet ability cooldowns by formation index.
	petCaptureAttempted bool        // Avoid repeated full-stable capture offers or failure spam per fight.
	// Skill-web bonus, loaded once per fight: treeBonusFor hits the DB and
	// scans the full 1000-node tree, so per-turn lookups must use this cache.
	treeBonus content.TreeBonus

	// --- Abyss combat mechanics (docs/ABYSS_IMPROVEMENTS_300.md, group C) ---
	// All of these are only consulted on paths gated by abyssCombatant(u), so
	// regular TS3 channel combat never sees them.
	lastCastElement   content.Element       // AB-51 element of the previous cast (elemental combo)
	stunbreakUsed     bool                  // AB-59 one free stun cleanse per boss fight
	stunbrokenRound   int                   // AB-59 round the stunbreak fired (acts at 50%)
	parryCount        int                   // AB-56 parries this fight (3 grant Stealth)
	stealthUntilRound int                   // AB-56 granted stealth: mobs skip up to this round
	fumbled           bool                  // AB-72 next hit gets +10% crit (embarrassed rage)
	weaponSwapped     bool                  // AB-53 once-per-fight mid-boss weapon swap
	petFocus          string                // AB-58 pet focus-fire target (mob name)
	petFocusLogged    bool                  // AB-58 one-time focus-fire log
	holdMana          bool                  // AB-64 hold-mana toggle (save casts for bosses)
	holdManaLogged    bool                  // AB-64 one-time hold-mana log
	lastAttackers     map[*content.Mob]bool // AB-68 mobs that targeted this user last mobTurn
	lastUltRound      int                   // AB-70 round an ultimate was last fired
	execFlourished    bool                  // AB-55 one-time Executioner+Execute flourish log
	cursedMercyLogged bool                  // AB-54 one-time cursed mercy log
	runeWardLogged    bool                  // AB-67 one-time rune-ward resist log
	defRuneLogged     bool                  // AB-84 one-time defensive-rune resist log
	petNervousLogged  map[*content.Mob]bool // AB-73 low-loyalty foreshadowing, once per pet per fight
	defendingRound    int                   // live combat: DEF boost remains through one enemy phase
	potionCooldown    int                   // shared cooldown for powerful live consumables
	relicCharges      int                   // run-bound active relic uses remaining
}

// cycleContext holds per-cycle shared facts used by the XP modifiers.
type cycleContext struct {
	onlineNormal       int
	channelNormalCount map[int]int
	onlineNicks        map[string]bool
	today              time.Time
}

func (b *Bot) buildCycleContext(clients []clientquery.ClientInfo) cycleContext {
	online := map[string]bool{}
	chans := map[int]int{}
	normal := 0
	for _, cl := range clients {
		if cl.Type == 0 {
			normal++
			online[strings.ToLower(cl.Nickname)] = true
			if cl.CID >= 0 {
				chans[cl.CID]++
			}
		}
	}
	return cycleContext{
		onlineNormal:       normal,
		channelNormalCount: chans,
		onlineNicks:        online,
		today:              time.Now(),
	}
}

// processUserXP applies all XP gains for one user this cycle.
func (b *Bot) processUserXP(uid, nickname string, cid, base int, hasGame bool, ctx cycleContext) (*levelResult, []string, string) {
	var notes []string
	delta := 0

	if b.Cfg.EnableXPModifiers {
		b.ensureUserHasGear(uid)

		if b.dailyLoginDue(uid, ctx.today) {
			delta += dailyLoginXP
			notes = append(notes, i18n.T("bot.flavour.daily_login", dailyLoginXP))
			b.setLastLogin(uid, ctx.today)
		}
	}

	stats, mult, _, mnotes := b.calculateTotalStats(uid, ctx.today)
	notes = append(notes, mnotes...)

	// Intelligence bonus
	if stats.INT > 0 {
		intMult := 1.0 + float64(stats.INT)/1000.0
		mult *= intMult
		notes = append(notes, i18n.T("bot.flavour.int_bonus", intMult))
	}

	// Flavour stats
	if stats.STN > 50 {
		notes = append(notes, i18n.T("bot.flavour.smell_terrible"))
	}
	if stats.CHA > 100 {
		notes = append(notes, i18n.T("bot.flavour.charming"))
	}
	if stats.SHN > 50 {
		notes = append(notes, i18n.T("bot.flavour.glowing"))
	}

	awardMult := b.computeMiscMult(uid, nickname, cid, ctx)
	if !hasGame {
		mult *= noGamePenalty
		notes = append(notes, i18n.T("bot.flavour.no_game_penalty"))
	}
	award := 0
	if base > 0 {
		award = int(math.Round(float64(base) * mult * awardMult))
		if award < 1 {
			award = 1
		}
	} else {
		// Penalty should NOT be subject to positive multipliers (streak, etc.)
		award = base // base is already negative here
		var curXP, curLevel int
		_ = b.DB.QueryRow("SELECT xp, level FROM users WHERE client_uid=$1", uid).Scan(&curXP, &curLevel)

		baseXP := leveling.XPForLevel(curLevel)
		levelProgress := curXP - baseXP

		maxLoss := -levelProgress
		if maxLoss > -10 {
			maxLoss = -10 // minimum 10 xp loss
		}

		if award < maxLoss {
			award = maxLoss
		}
	}
	delta += award

	lr, err := b.awardXP(uid, nickname, delta)
	if err != nil {
		log.Printf("processUserXP: awardXP failed for %s: %v", nickname, err)
		return &levelResult{}, notes, ""
	}

	if b.Cfg.EnableXPModifiers {
		if lr.NewLevel/lootBoxEveryLevels > lr.OldLevel/lootBoxEveryLevels {
			// Milestone Reward!
			g := content.RandomGearDrop()
			if g.Rarity < content.RarityEpic {
				g.Rarity = content.RarityEpic
			}
			_ = b.awardGearDrop(uid, g)

			c := content.RandomConsumable()
			_, _ = b.DB.Exec("INSERT INTO user_consumables (client_uid, cons_id, remaining_fights) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", uid, c.ID, c.Duration)

			notes = append(notes, fmt.Sprintf("🎁 Level %d Milestone Reached! You found a %s and a %s!", (lr.NewLevel/lootBoxEveryLevels)*lootBoxEveryLevels, g.Name, c.Name))
		}
	}

	return lr, notes, ""
}

func (b *Bot) computeMiscMult(uid, _ string, cid int, ctx cycleContext) float64 {
	if !b.Cfg.EnableXPModifiers {
		return 1.0
	}
	mult := 1.0
	if streak := b.updateStreak(uid, ctx.today); streakMultiplier(streak) > 1 {
		mult *= streakMultiplier(streak)
	}
	// #nosec G404
	if rand.Float64() < critChance { // #nosec G404
		mult *= critMult
	}
	if sv := serverMultiplier(ctx.onlineNormal); sv > 1 {
		mult *= sv
	}
	if cid >= 0 && ctx.channelNormalCount[cid] > 1 {
		mult *= partyMult
	}

	// Group size XP penalty: Solo players get 100% XP. Groups of 2-4 get a 10% penalty.
	// Groups of 5+ get an additional 5% penalty per extra member (min 50%).
	groupSize := ctx.channelNormalCount[cid]
	if groupSize >= 2 {
		var groupPenalty float64
		if groupSize <= 4 {
			groupPenalty = 0.9 // 10% penalty for small groups
		} else {
			groupPenalty = 0.9 - float64(groupSize-4)*0.05
			if groupPenalty < 0.5 {
				groupPenalty = 0.5
			}
		}
		mult *= groupPenalty
	}

	return mult
}

func (b *Bot) getPets(uid string) []*content.Mob {
	rows, err := b.DB.Query(`SELECT p.name,p.mob_type,p.level,p.hp,p.max_hp,p.str,p.def,p.spd,p.loyalty,p.autoskills::text
		FROM user_pets p JOIN users u ON u.client_uid=p.client_uid
		WHERE p.client_uid=$1 AND (p.active_slot=1 OR (p.active_slot=2 AND u.abyss_prestige>=2))
		ORDER BY p.active_slot`, uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []*content.Mob
	for rows.Next() {
		var m content.Mob
		var mType string
		var maxHP int
		var rawProfile string
		if err := rows.Scan(&m.Name, &mType, &m.Level, &m.Stats.HP, &maxHP, &m.Stats.STR, &m.Stats.DEF, &m.Stats.SPD, &m.Loyalty, &rawProfile); err == nil {
			profile := decodeAbyssPetProfile(rawProfile)
			if profile.busy(time.Now()) {
				continue
			}
			m.Type = content.MobType(mType)
			m.MaxHP = maxHP
			m.PetShiny = profile.Shiny
			m.PetBoss = profile.BossVariant
			m.PetBark = profile.BarkStyle
			out = append(out, &m)
		}
	}
	if rows.Err() != nil || len(out) == 0 {
		return out
	}
	_ = rows.Close()
	petGearStats := abyssPetGearStats(b.getEquippedItems(uid))
	for _, pet := range out {
		applyAbyssPetGear(pet, petGearStats)
		applyAbyssPetClass(pet)
		_, _, moodPct := abyssPetMood(pet.Stats.HP, pet.MaxHP, pet.Loyalty)
		combatPct := moodPct + abyssPetLoyaltyBonusPct(pet.Loyalty)
		pet.Stats.STR = abyssPetMoodScale(pet.Stats.STR, combatPct)
		pet.Stats.DEF = abyssPetMoodScale(pet.Stats.DEF, combatPct)
		pet.Stats.SPD = abyssPetMoodScale(pet.Stats.SPD, combatPct)
	}
	return out
}

func (b *Bot) loadPetHealSettings(uid string) map[string]bool {
	rows, err := b.DB.Query(`SELECT name,COALESCE((autoskills->>'heal')::boolean,TRUE)
		FROM user_pets WHERE client_uid=$1 AND active_slot>0`, uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	settings := map[string]bool{}
	for rows.Next() {
		var name string
		var enabled bool
		if rows.Scan(&name, &enabled) != nil {
			return nil
		}
		settings[name] = enabled
	}
	return settings
}

func abyssPetAutoskillEnabled(settings map[string]bool, petName string) bool {
	if settings == nil {
		return true
	}
	enabled, exists := settings[petName]
	return !exists || enabled
}

func (b *Bot) deletePet(uid, name string) {
	_, _ = b.DB.Exec(`WITH fallen AS (
		DELETE FROM user_pets WHERE client_uid=$1 AND name=$2
		RETURNING client_uid,name,mob_type,level,loyalty
	) INSERT INTO abyss_pet_memorials (client_uid,name,mob_type,level,loyalty)
	SELECT client_uid,name,mob_type,level,loyalty FROM fallen`, uid, name)
}

func (b *Bot) updatePetState(uid string, pet *content.Mob) {
	if pet == nil || pet.Stats.HP <= 0 {
		if pet != nil {
			b.deletePet(uid, pet.Name)
		}
		return
	}
	_, _ = b.DB.Exec("UPDATE user_pets SET hp=$1, loyalty=$2 WHERE client_uid=$3 AND name=$4", pet.Stats.HP, pet.Loyalty, uid, pet.Name)
}

func (b *Bot) checkUserRevive(u *UserInCombat, logs *[]string) bool {
	if u.CurrentHP > 0 {
		return false
	}

	// 1. Check Consumables
	cons := b.getConsumables(u.UID)
	for _, c := range cons {
		if c.Type == content.ConsumableRevive {
			u.CurrentHP = u.Stats.HP / 2
			*logs = append(*logs, i18n.T("bot.combat.revived_item", u.Nickname, c.ID))
			_, _ = b.DB.Exec("DELETE FROM user_consumables WHERE client_uid = $1 AND cons_id = $2", u.UID, c.ID)
			return true
		}
	}
	// 2. Check Item Effects (Phoenix)
	_, _, _, _, effects := b.activeLootMult(u.UID, time.Now())
	for _, eff := range effects {
		if eff == content.EffectPhoenix {
			u.CurrentHP = u.Stats.HP / 2
			*logs = append(*logs, i18n.T("bot.combat.revived_phoenix", u.Nickname))
			return true
		}
	}
	return false
}

func getElementMult(attacker, defender content.Element) float64 {
	// Fire > Air > Earth > Water > Fire
	switch attacker {
	case content.ElementFire:
		if defender == content.ElementAir {
			return 2.0
		}
		if defender == content.ElementWater {
			return 0.5
		}
	case content.ElementAir:
		if defender == content.ElementEarth {
			return 2.0
		}
		if defender == content.ElementFire {
			return 0.5
		}
	case content.ElementEarth:
		if defender == content.ElementWater {
			return 2.0
		}
		if defender == content.ElementAir {
			return 0.5
		}
	case content.ElementWater:
		if defender == content.ElementFire {
			return 2.0
		}
		if defender == content.ElementEarth {
			return 0.5
		}
	}
	return 1.0
}

// LootResult is one item/gold grant from a resolved fight, ready to log and
// (optionally) poke the recipient about.
type LootResult struct {
	UID      string
	Note     string
	Poke     string
	PityProc bool
}

// combatTimelineFrame is an authoritative snapshot taken by the combat engine
// after a group of log lines. The Abyss web UI replays these frames alongside
// those lines so HP/mana animation never has to infer state from translated text.
type combatTimelineFrame struct {
	AfterLog int    `json:"after_log"`
	Round    int    `json:"round,omitempty"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"max_hp"`
	Mana     int    `json:"mana"`
	MaxMana  int    `json:"max_mana"`
	EnemyHP  int    `json:"enemy_hp"`
	EnemyMax int    `json:"enemy_max"`
	PetName  string `json:"pet_name,omitempty"`
	PetHP    int    `json:"pet_hp,omitempty"`
	PetMax   int    `json:"pet_max,omitempty"`
	Side     string `json:"side,omitempty"`
	Actions  int    `json:"actions,omitempty"`
}

func appendCombatTimelineFrame(frames *[]combatTimelineFrame, afterLog, round int, users []activeUser, mobs []*content.Mob) {
	if len(users) == 0 || users[0].u == nil {
		return
	}
	enemyHP, enemyMax := 0, 0
	for _, mob := range mobs {
		if mob == nil {
			continue
		}
		enemyHP += max(0, mob.Stats.HP)
		enemyMax += max(0, mob.MaxHP)
	}
	frame := combatTimelineFrame{
		AfterLog: afterLog,
		Round:    round,
		HP:       max(0, users[0].u.CurrentHP),
		MaxHP:    max(0, users[0].u.Stats.HP),
		Mana:     max(0, users[0].CurrentMana),
		MaxMana:  max(0, users[0].MaxMana),
		EnemyHP:  enemyHP,
		EnemyMax: enemyMax,
	}
	for _, pet := range users[0].u.Pets {
		if pet == nil || pet.MaxHP <= 0 {
			continue
		}
		frame.PetName = pet.Name
		frame.PetHP = max(0, min(pet.Stats.HP, pet.MaxHP))
		frame.PetMax = pet.MaxHP
		break
	}
	if n := len(*frames); n > 0 && (*frames)[n-1].AfterLog == afterLog {
		(*frames)[n-1] = frame
		return
	}
	*frames = append(*frames, frame)
}

func markCombatTimelineExchange(frames *[]combatTimelineFrame, side string, actionLogs int) {
	if len(*frames) == 0 {
		return
	}
	frame := &(*frames)[len(*frames)-1]
	frame.Side = side
	frame.Actions = min(6, max(1, actionLogs))
}

// ambushDamageCapPct bounds how much of a player's max HP a surprise round (mobs
// acting before any player has moved) may strip. An ambush can wound but must
// never be a guaranteed kill — combined with the "never below 1 HP during the
// ambush" clamp in mobTurn, every player is guaranteed to act at least once. This
// applies to all combat modes (the channel cycle and the Abyss alike).
const ambushDamageCapPct = 0.5

// bossResistCap is the most boss damage a character can ever shrug off via the
// auto-scaling boss resistance below.
const bossResistCap = 0.80

// bossResist is an automatic, derived "boss resistance" that mitigates damage from
// boss-tier enemies, scaling with the character's level and gear score. Bosses
// should be a war of attrition every player gets to fight, not a coin-flip
// one-shot. It is purely derived (no gear slot / no DB column) so it grows with
// progression on its own, and is hard-capped so bosses still hurt. Applies in all
// combat modes.
func bossResist(level, gearScore int) float64 {
	r := float64(level)*0.0006 + float64(gearScore)*0.00002
	if r < 0 {
		r = 0
	}
	if r > bossResistCap {
		r = bossResistCap
	}
	return r
}

func (b *Bot) resolveChannelCombatDetailed(users []UserInCombat, initialMobs []*content.Mob, avgLvl int, diffFactor float64, zone content.Zone) ([]string, int, bool, []LootResult, []combatTimelineFrame) {
	return b.resolveChannelCombatDetailedWithRandom(
		users,
		initialMobs,
		avgLvl,
		diffFactor,
		zone,
		combatRandomForUsers(users),
	)
}

func (b *Bot) resolveChannelCombatDetailedWithRandom(
	users []UserInCombat,
	initialMobs []*content.Mob,
	avgLvl int,
	diffFactor float64,
	zone content.Zone,
	random combatRandomSource,
) ([]string, int, bool, []LootResult, []combatTimelineFrame) {
	rand := random
	var logs []string
	var loots []LootResult
	var timeline []combatTimelineFrame
	victory := false
	var totalUserDamage, totalMobDamage, killedXP int

	isAbyss := false
	for _, uc := range users {
		if uc.EscrowLoot {
			isAbyss = true
			break
		}
	}

	floorMod := ""
	for _, uc := range users {
		if uc.FloorModifier != "" {
			floorMod = uc.FloorModifier
			break
		}
	}

	// Determine number of waves (1-3)
	// #nosec G404
	waves := 1
	// #nosec G404
	if rand.Float64() < 0.2 {
		waves = 2
	}
	// #nosec G404
	if rand.Float64() < 0.05 {
		waves = 3
	}

	// Later waves spawn at the *encounter's* level, not the party's. In normal
	// combat the initial mobs are already spawned at avgLvl so this is a no-op, but
	// the Abyss spawns a decoupled, depth-scaled mob level — without this, waves 2/3
	// would ignore that and spawn at the (much higher) player level, undoing the
	// Abyss mob-level decoupling and producing instant-kill follow-up waves.
	spawnLvl := avgLvl
	if len(initialMobs) > 0 {
		sum := 0
		for _, m := range initialMobs {
			sum += m.Level
		}
		spawnLvl = sum / len(initialMobs)
		if spawnLvl < 1 {
			spawnLvl = 1
		}
	}

	phaseOnce := make(map[string]bool)
	var track *abyssFightTrack
	if isAbyss {
		track = &abyssFightTrack{}
	}
	activeUsers := make([]activeUser, len(users))
	for i := range users {
		_, _, _, _, effects := b.activeLootMult(users[i].UID, time.Now())
		activeUsers[i] = activeUser{
			u: &users[i], effects: effects, treeBonus: users[i].treeBonus,
			skillCooldowns: map[string]int{}, petCooldowns: map[int]int{},
			petNervousLogged: map[*content.Mob]bool{},
		}
		activeUsers[i].u.STRMod = 1.0
		activeUsers[i].u.DEFMod = 1.0
		activeUsers[i].u.SPDMod = 1.0
		activeUsers[i].MaxMana = 100 + users[i].Stats.MNA
		activeUsers[i].CurrentMana = activeUsers[i].MaxMana
		// Abyss-only combat options (AB-58 pet focus-fire, AB-64 hold mana),
		// persisted in app_meta so no schema change is needed.
		if abyssCombatant(&users[i]) {
			activeUsers[i].holdMana = b.abyssHoldMana(users[i].UID)
			activeUsers[i].petFocus = b.abyssPetFocus(users[i].UID)
			activeUsers[i].relicCharges = int(b.loadRunFlags(users[i].UID)[abyssRunFlagRelicCharges])
		}
	}
	totalRounds := 0

	for w := 1; w <= waves; w++ {
		var currentMobs []*content.Mob
		if w == 1 {
			// Deep copy initial mobs
			currentMobs = make([]*content.Mob, len(initialMobs))
			for i, m := range initialMobs {
				currentMobs[i] = m.Clone()
				currentMobs[i].STRMod = 1.0
				currentMobs[i].DEFMod = 1.0
				currentMobs[i].SPDMod = 1.0
				// Some manually-built mobs (e.g. Abyss scripted bosses) ship without
				// MaxHP set; seed it from base HP so phase-percentage checks are valid.
				if currentMobs[i].MaxHP <= 0 {
					currentMobs[i].MaxHP = currentMobs[i].Stats.HP
					currentMobs[i].CurrentHP = currentMobs[i].MaxHP
				}
				if floorMod == "enraged" {
					currentMobs[i].Effects = append(currentMobs[i].Effects, content.EffectEnraged)
				}
			}
		} else {
			// Spawn new wave
			logs = append(logs, i18n.T("bot.combat.wave_approaches", w))
			newMobs := content.SpawnMobGroupWithRandom(
				spawnLvl,
				zone,
				diffFactor*zone.Difficulty,
				len(users),
				w == 3,
				rand,
			)
			currentMobs = make([]*content.Mob, len(newMobs))
			for i := range newMobs {
				currentMobs[i] = (&newMobs[i]).Clone()
				currentMobs[i].STRMod = 1.0
				currentMobs[i].DEFMod = 1.0
				currentMobs[i].SPDMod = 1.0
				if floorMod == "enraged" {
					currentMobs[i].Effects = append(currentMobs[i].Effects, content.EffectEnraged)
				}
				initialMobs = append(initialMobs, currentMobs[i]) // track for rewards
			}
		}

		// Initialize wave header (rarity-coloured enemy names + wave countdown)
		mobCounts := make(map[string]int)
		mobTypes := make(map[string]content.MobType)
		totalEnemyCR := 0
		for _, m := range currentMobs {
			dn := m.DisplayName()
			if hasAbyssFloorModifier(floorMod, "darkness") {
				dn = m.DisplayNameShort()
			}
			mobCounts[dn]++
			mobTypes[dn] = m.Type
			totalEnemyCR += m.Score()
		}
		// Iterate in sorted key order so the wave header is stable across runs
		// (Go map iteration order is randomized).
		mobNames := make([]string, 0, len(mobCounts))
		for name := range mobCounts {
			mobNames = append(mobNames, name)
		}
		sort.Strings(mobNames)
		var enemyNames []string
		for _, name := range mobNames {
			count := mobCounts[name]
			display := colorMobName(name, mobTypes[name])
			if count > 1 {
				enemyNames = append(enemyNames, i18n.T("bot.combat.enemy_count", count, display))
			} else {
				enemyNames = append(enemyNames, display)
			}
		}
		// On the opening wave, set the scene with a deterministic zone lore line.
		if w == 1 {
			if lore := zoneLore(zone.Name); lore != "" {
				logs = append(logs, lore)
			}
		}
		logs = append(logs, i18n.T("bot.combat.wave_header", w, totalEnemyCR, strings.Join(enemyNames, ", "), waves))
		appendCombatTimelineFrame(&timeline, len(logs), 0, activeUsers, currentMobs)

		// Reset SPD for any stunned mobs from previous round/waves
		for _, m := range currentMobs {
			if m.Stats.SPD == 0 {
				m.Stats.SPD = 10
			}
		}

		// Fight the wave
		waveVictory := false
		// #nosec G404
		playerStarts := rand.IntN(2) == 0 // #nosec G404
		if !playerStarts {
			logs = append(logs, i18n.T("bot.combat.ambush"))
		}

		maxRounds := 10
		enrageRound := combatBossEnrageRound(isAbyss)
		if isAbyss {
			maxRounds = 40
		}

		// AB-70: per-wave boss summon telegraphs (mob → round the channel began).
		summonTelegraph := make(map[*content.Mob]int)
		stormSide := ""

		for r := 1; r <= maxRounds; r++ {
			totalRounds++
			intensify := 1.0 + float64(r-1)*0.15
			fatigueMult := 1.0
			if r > 5 {
				fatigueMult = 1.0 - float64(r-5)*0.1
				if fatigueMult < 0.1 {
					fatigueMult = 0.1
				}
			}
			healPenalty := 1.0
			if r >= 6 {
				healPenalty = 1.0 - float64(r-5)*0.2
			}
			if healPenalty < 0 {
				healPenalty = 0
			}

			// AB-36: storm rooms announce the side selected by the authoritative
			// combat RNG one full planning round before the strike lands.
			if hasAbyssFloorModifier(floorMod, "storm_floor") {
				if stormSide != "" {
					partyDamage, enemyDamage := strikeAbyssStorm(stormSide, activeUsers, currentMobs)
					totalMobDamage += partyDamage
					totalUserDamage += enemyDamage
					logs = append(logs, abyssStormImpactLog(stormSide, partyDamage+enemyDamage))
					for i := range activeUsers {
						if activeUsers[i].u != nil && activeUsers[i].u.CurrentHP <= 0 {
							_ = b.checkUserRevive(activeUsers[i].u, &logs)
						}
					}
					appendCombatTimelineFrame(&timeline, len(logs), r, activeUsers, currentMobs)
					aliveUsers := 0
					for i := range activeUsers {
						if activeUsers[i].u != nil && activeUsers[i].u.CurrentHP > 0 {
							aliveUsers++
						}
					}
					if aliveUsers == 0 {
						break
					}
					if len(b.getAliveMobs(currentMobs)) == 0 {
						waveVictory = true
						break
					}
				}
				stormSide = nextAbyssStormSide(rand)
				telegraph := abyssStormTelegraph(stormSide)
				logs = append(logs, telegraph)
				if liveCombat := abyssLiveCombatFor(activeUsers); liveCombat != nil {
					liveCombat.setHazardTelegraph(telegraph)
				}
			}

			// AB-75 Desperation: from round 25 both sides gain stacking +5%
			// damage per round. Only reachable in the Abyss (normal combat caps
			// at 10 rounds), so no extra gating is needed.
			despMult := 1.0
			if r > 25 {
				despMult = 1.0 + float64(r-25)*0.05
				if r == 26 {
					logs = append(logs, "😤 Desperation sets in — both sides fight +5% harder each round!")
				}
			}

			// AB-61 Enrage warning (combat-log side; the boss-card border flash
			// is web UI): call out the coming enrage 2 rounds ahead.
			if isAbyss && r == enrageRound-2 && !phaseOnce["enrage_warn"] {
				for _, m := range currentMobs {
					if m.Stats.HP > 0 && m.Type == content.MobBoss {
						phaseOnce["enrage_warn"] = true
						logs = append(logs, fmt.Sprintf("⚠️ %s is winding up — ENRAGE in 2 rounds!", m.Name))
						break
					}
				}
			}

			// AB-70 Boss summon interrupt: below 50% HP an Abyss boss channels a
			// summoning ritual for one round (skipping its attack); an ultimate
			// fired during the channel cancels the summon.
			if isAbyss {
				for _, m := range currentMobs {
					if m.Stats.HP <= 0 || m.Type != content.MobBoss {
						continue
					}
					if tr, ok := summonTelegraph[m]; ok {
						if tr >= r {
							continue
						}
						delete(summonTelegraph, m)
						interrupted := false
						for i := range activeUsers {
							if activeUsers[i].lastUltRound >= tr {
								interrupted = true
								break
							}
						}
						if interrupted {
							logs = append(logs, fmt.Sprintf("⚡ %s's summoning ritual is INTERRUPTED by the ultimate!", m.Name))
						} else {
							choreography := abyssBossSummonFor(m.Name)
							newMob := content.SpawnMobWithRandom(
								spawnLvl,
								false,
								diffFactor*zone.Difficulty*0.7,
								rand,
							)
							newMob.Name = choreography.AddPrefix + " " + newMob.Name
							add := newMob.Clone()
							add.STRMod, add.DEFMod, add.SPDMod = 1.0, 1.0, 1.0
							if add.MaxHP <= 0 {
								add.MaxHP = add.Stats.HP
								add.CurrentHP = add.MaxHP
							}
							currentMobs = append(currentMobs, add)
							initialMobs = append(initialMobs, add) // track for rewards
							logs = append(logs, choreography.Arrival)
						}
						continue
					}
					if m.MaxHP > 0 && m.Stats.HP*2 < m.MaxHP && !phaseOnce["summon:"+m.Name] {
						phaseOnce["summon:"+m.Name] = true
						summonTelegraph[m] = r
						m.Stats.SPD = 0 // channelling: the boss skips this round's attack
						logs = append(logs, abyssBossSummonFor(m.Name).Telegraph)
					}
				}
			}

			// Scripted Boss Phases and Soft-Enrage (Item #63, #69, #70)
			for _, m := range currentMobs {
				if m.Stats.HP > 0 {
					// Check soft-enrage at the advertised threshold.
					if combatBossShouldEnrage(r, isAbyss) && m.Type == content.MobBoss {
						hasEnraged := false
						for _, eff := range m.Effects {
							if eff == content.EffectEnraged {
								hasEnraged = true
								break
							}
						}
						if !hasEnraged {
							m.Effects = append(m.Effects, content.EffectEnraged)
							logs = append(logs, fmt.Sprintf("[color=#f44336]⏳ The Abyss closes in! %s becomes ENRAGED! (Double damage)[/color]", m.Name))
						}
					}

					// Custom Boss Scripted Phases — use the live combat HP field
					// (m.Stats.HP, which damage decrements) against MaxHP, not the
					// spawn-time CurrentHP snapshot which never updates here.
					hpPct := float64(m.Stats.HP) / float64(m.MaxHP)
					if m.Name == "Gorgoroth the Firelord" {
						if hpPct < 0.50 {
							hasEnraged := false
							for _, eff := range m.Effects {
								if eff == content.EffectEnraged {
									hasEnraged = true
									break
								}
							}
							if !hasEnraged {
								m.Effects = append(m.Effects, content.EffectEnraged)
								m.Element = content.ElementFire
								logs = append(logs, "[color=#ff3333]🔥 Gorgoroth bellows in fury as his blood boils, wrapping himself in roaring flames! (Gains Enraged & Element shifted to Fire)[/color]")
							}
						}
					} else if m.Name == "Malakor the Voidweaver" {
						if hpPct < 0.50 {
							hasArmored := false
							for _, eff := range m.Effects {
								if eff == content.EffectArmored {
									hasArmored = true
									break
								}
							}
							if !hasArmored {
								// Purge debuffs only; preserve beneficial effects (Regen, etc.)
								var kept []content.MobEffect
								for _, eff := range m.Effects {
									if eff == content.EffectRegen || eff == content.EffectArmored {
										kept = append(kept, eff)
									}
								}
								m.Effects = append(kept, content.EffectArmored)
								logs = append(logs, "[color=#9c27b0]🔮 Malakor wraps himself in a shimmering Void Barrier! (Active debuffs purged, gains Armored)[/color]")
							}
						}
					} else if m.Name == "Azazoth the Slumbering Eye" {
						if hpPct < 0.50 {
							if !phaseOnce["azazoth_stun"] {
								phaseOnce["azazoth_stun"] = true
								for i := range activeUsers {
									activeUsers[i].Stunned = true
								}
								logs = append(logs, "[color=#ffeb3b]👁️ Azazoth opens his slumbering eye, releasing a hypnotic pulse that dazes all delvers! (Skip next turn)[/color]")
							}
						}
					} else if m.Name == "Abyssus, Heart of the Void" {
						if hpPct < 0.75 {
							hasEnraged := false
							for _, eff := range m.Effects {
								if eff == content.EffectEnraged {
									hasEnraged = true
									break
								}
							}
							if !hasEnraged {
								m.Effects = append(m.Effects, content.EffectEnraged)
								m.Element = content.ElementFire
								logs = append(logs, "[color=#ff3333]🔥 Abyssus bellows in fury, wrapping himself in roaring flames! (Gains Enraged & Element shifted to Fire)[/color]")
							}
						}
						if hpPct < 0.50 {
							hasArmored := false
							for _, eff := range m.Effects {
								if eff == content.EffectArmored {
									hasArmored = true
									break
								}
							}
							if !hasArmored {
								m.Effects = append(m.Effects, content.EffectArmored)
								logs = append(logs, "[color=#9c27b0]🔮 Abyssus channels Void shield! (Gains Armored)[/color]")
							}
						}
						if hpPct < 0.25 {
							if !phaseOnce["abyssus_stun"] {
								phaseOnce["abyssus_stun"] = true
								for i := range activeUsers {
									activeUsers[i].Stunned = true
								}
								logs = append(logs, "[color=#ffeb3b]👁️ Abyssus releases a cataclysmic sleep shockwave! (All delvers stunned)[/color]")
							}
						}
					}
				}
			}

			b.applyEffects(activeUsers, currentMobs, zone, r, intensify, healPenalty, &logs)
			appendCombatTimelineFrame(&timeline, len(logs), r, activeUsers, currentMobs)
			liveActions := map[string]abyssLiveAction{}
			liveCombat := abyssLiveCombatFor(activeUsers)
			if liveCombat != nil {
				liveActions = liveCombat.awaitActions(r, activeUsers, currentMobs, logs, playerStarts)
			}
			resolutionStarted := time.Now()
			observeLiveResolution := func() {
				if liveCombat != nil {
					liveCombat.server.abyssOps.observeResolution(time.Since(resolutionStarted))
					liveCombat = nil
				}
			}

			if playerStarts {
				userLogStart := len(logs)
				b.userTurn(activeUsers, &currentMobs, zone, intensify*fatigueMult*despMult, healPenalty, &logs, &totalUserDamage, &totalMobDamage, avgLvl, diffFactor, users, &loots, r, track, liveActions, rand)
				appendCombatTimelineFrame(&timeline, len(logs), r, activeUsers, currentMobs)
				markCombatTimelineExchange(&timeline, "player", len(logs)-userLogStart)
				if len(b.getAliveMobs(currentMobs)) == 0 {
					observeLiveResolution()
					waveVictory = true
					break
				}
				mobLogStart := len(logs)
				b.mobTurn(activeUsers, currentMobs, zone, intensify*despMult, &logs, &totalMobDamage, &totalUserDamage, r, false, track, rand)
				appendCombatTimelineFrame(&timeline, len(logs), r, activeUsers, currentMobs)
				markCombatTimelineExchange(&timeline, "enemy", len(logs)-mobLogStart)
				observeLiveResolution()
			} else {
				// The opening round of an enemy-first wave is the ambush: soften it so
				// it can't one-shot a player before they ever act.
				mobLogStart := len(logs)
				b.mobTurn(activeUsers, currentMobs, zone, intensify*despMult, &logs, &totalMobDamage, &totalUserDamage, r, r == 1, track, rand)
				appendCombatTimelineFrame(&timeline, len(logs), r, activeUsers, currentMobs)
				markCombatTimelineExchange(&timeline, "enemy", len(logs)-mobLogStart)
				aliveUsers := 0
				for _, u := range users {
					if u.CurrentHP > 0 {
						aliveUsers++
					}
				}
				if aliveUsers == 0 {
					observeLiveResolution()
					break
				}
				userLogStart := len(logs)
				b.userTurn(activeUsers, &currentMobs, zone, intensify*fatigueMult*despMult, healPenalty, &logs, &totalUserDamage, &totalMobDamage, avgLvl, diffFactor, users, &loots, r, track, liveActions, rand)
				appendCombatTimelineFrame(&timeline, len(logs), r, activeUsers, currentMobs)
				markCombatTimelineExchange(&timeline, "player", len(logs)-userLogStart)
				observeLiveResolution()
				if len(b.getAliveMobs(currentMobs)) == 0 {
					waveVictory = true
					break
				}
			}

			// AB-24 Anti-stall: from round 31, unavoidable fatigue burns both
			// sides for an additional 1% of max HP per overtime round. Resolve it
			// simultaneously after both turns so initiative cannot evade the tax.
			if isAbyss && r > 30 {
				userFatigue, mobFatigue := 0, 0
				for i := range activeUsers {
					u := activeUsers[i].u
					if u == nil || u.CurrentHP <= 0 {
						continue
					}
					damage := min(abyssFatigueDamage(u.Stats.HP, r), u.CurrentHP)
					u.CurrentHP -= damage
					u.DamageTaken += damage
					userFatigue += damage
					if u.CurrentHP <= 0 {
						_ = b.checkUserRevive(u, &logs)
					}
				}
				for _, mob := range currentMobs {
					if mob == nil || mob.Stats.HP <= 0 {
						continue
					}
					damage := min(abyssFatigueDamage(mob.MaxHP, r), mob.Stats.HP)
					mob.Stats.HP -= damage
					mobFatigue += damage
				}
				totalMobDamage += userFatigue
				totalUserDamage += mobFatigue
				logs = append(logs, fmt.Sprintf("🥀 Overtime fatigue ×%d sears both sides — party %d, enemies %d.", r-30, userFatigue, mobFatigue))
				appendCombatTimelineFrame(&timeline, len(logs), r, activeUsers, currentMobs)
				aliveUsers := 0
				for i := range activeUsers {
					if activeUsers[i].u != nil && activeUsers[i].u.CurrentHP > 0 {
						aliveUsers++
					}
				}
				if aliveUsers == 0 {
					break
				}
				if len(b.getAliveMobs(currentMobs)) == 0 {
					waveVictory = true
					break
				}
			}

			for i := range activeUsers {
				for _, us := range activeUsers[i].u.Ultimates {
					if us.CurrentCooldown > 0 {
						us.CurrentCooldown--
					}
				}
				logs = append(logs, tickAbyssPetAbilityCooldowns(&activeUsers[i])...)
				if activeUsers[i].potionCooldown > 0 {
					activeUsers[i].potionCooldown--
				}
				for skillID, cooldown := range activeUsers[i].skillCooldowns {
					if cooldown <= 1 {
						delete(activeUsers[i].skillCooldowns, skillID)
					} else {
						activeUsers[i].skillCooldowns[skillID] = cooldown - 1
					}
				}
			}
			for _, mob := range currentMobs {
				if mob.StunRounds <= 0 {
					continue
				}
				mob.StunRounds--
				if mob.StunRounds == 0 && mob.PreStunSPD > 0 {
					mob.Stats.SPD = mob.PreStunSPD
					mob.PreStunSPD = 0
				}
			}

			aliveUsers := 0
			for _, u := range users {
				if u.CurrentHP > 0 {
					aliveUsers++
				}
			}
			if aliveUsers == 0 {
				break
			}
		}

		// Per-kill XP: bank the RewardXP of every mob that died this wave, so a
		// wipe still credits the kills that landed (a lost fight keeps 25% below).
		for _, m := range currentMobs {
			if m.Stats.HP <= 0 {
				killedXP += m.RewardXP
			}
		}

		if !waveVictory {
			victory = false
			break
		}
		if w == waves {
			victory = true
		}
	}

	var finalAwardedXP int
	logs, finalAwardedXP, victory = b.distributeRewards(users, activeUsers, victory, totalUserDamage, totalMobDamage, killedXP, initialMobs, nil, zone, logs, avgLvl, track, rand)

	// AB-69 Kill-chain: clearing the floor in ≤2 rounds grants +5% speed next
	// floor, stacking ×3. Granted after distributeRewards so the standard
	// per-fight consumable decrement doesn't burn the fresh stack immediately.
	if victory && isAbyss && totalRounds >= 1 && totalRounds <= 2 {
		for i := range users {
			if users[i].IsClone {
				continue
			}
			if stacks := b.grantKillChain(users[i].UID); stacks > 0 {
				logs = append(logs, fmt.Sprintf("⚡ Kill-chain! %s cleared the floor in %d round(s) — +%d%% speed next floor!", users[i].Nickname, totalRounds, stacks*5))
			}
		}
	}
	return logs, finalAwardedXP, victory, loots, timeline
}

func (b *Bot) resolveChannelCombat(users []UserInCombat, initialMobs []*content.Mob, avgLvl int, diffFactor float64, zone content.Zone) ([]string, int, bool, []LootResult) {
	logs, xp, victory, loots, _ := b.resolveChannelCombatDetailed(users, initialMobs, avgLvl, diffFactor, zone)
	return logs, xp, victory, loots
}

func (b *Bot) applyEffects(activeUsers []activeUser, mobs []*content.Mob, zone content.Zone, round int, intensify, healPenalty float64, logs *[]string) {
	doubleHazards := false
	isAbyss := false
	for _, au := range activeUsers {
		isAbyss = isAbyss || au.u != nil && abyssCombatant(au.u)
		if strings.Contains(au.u.FloorModifier, "double_hazards") {
			doubleHazards = true
		}
	}
	for _, eff := range zone.Effects {
		if eff.Type == content.ZoneHazard {
			dmg := int(eff.Power * 25 * intensify)
			if doubleHazards {
				dmg *= 2
			}
			if dmg < 1 {
				dmg = 1
			}
			for i := range activeUsers {
				u := activeUsers[i].u
				hasCleanse := false
				for _, ueff := range activeUsers[i].effects {
					if ueff == content.EffectCleanse {
						hasCleanse = true
						break
					}
				}
				if hasCleanse {
					if round == 1 {
						*logs = append(*logs, i18n.T("bot.combat.hazard_cleansed", u.Nickname, eff.Name))
					}
					continue
				}
				u.DamageTaken += max(dmg, 0)
				u.CurrentHP -= dmg
				if u.CurrentHP <= 0 {
					u.CurrentHP = 0
					if !b.checkUserRevive(u, logs) {
						*logs = append(*logs, i18n.T("bot.combat.hazard_slain", u.Nickname, eff.Name))
					}
				}
			}
			for _, m := range mobs {
				m.Stats.HP -= dmg
			}
			if round == 1 {
				*logs = append(*logs, i18n.T("bot.combat.hazard_active", eff.Name))
			}
		}
	}

	for i := range mobs {
		m := mobs[i]
		if m.Stats.HP <= 0 {
			continue
		}
		// Improvement 4: Status Effect Stacking
		poisonStacks := 0
		regenStacks := 0
		for _, eff := range m.Effects {
			if eff == content.EffectPoisoned {
				poisonStacks++
			}
			if eff == content.EffectRegen {
				regenStacks++
			}
		}

		if poisonStacks > 0 {
			delta := int(float64(m.Stats.HP/20) * float64(poisonStacks) * intensify)
			if delta < 1 {
				delta = 1
			}
			m.Stats.HP -= delta
			if round%3 == 0 {
				line := i18n.T("bot.combat.poison_damage", m.Name, delta, poisonStacks)
				*logs = append(*logs, markAbyssDoTLog(line, isAbyss))
			}
		}
		if regenStacks > 0 {
			delta := int(float64(m.Stats.HP/20) * float64(regenStacks) * healPenalty)
			if delta < 1 {
				delta = 1
			}
			m.Stats.HP += delta
		}
	}

	for _, au := range activeUsers {
		u := au.u
		if u.CurrentHP <= 0 {
			continue
		}

		// Interactive fights let the submitted action/tactic engine decide whether
		// a potion is worth spending; the legacy engine retains its automatic use.
		if u.live == nil && u.CurrentHP < u.Stats.HP/2 && healPenalty > 0 {
			cons := b.getConsumables(u.UID)
			for _, c := range cons {
				if c.Type == content.ConsumableHealing {
					healAmt := int(float64(u.Stats.HP) * c.EffectValue)
					u.CurrentHP += healAmt
					if u.CurrentHP > u.Stats.HP {
						u.CurrentHP = u.Stats.HP
					}
					*logs = append(*logs, i18n.T("bot.combat.consumable_used", u.Nickname, c.Name, healAmt, c.EffectValue*100))
					// Consume the item
					_, _ = b.DB.Exec("DELETE FROM user_consumables WHERE ctid IN (SELECT ctid FROM user_consumables WHERE client_uid = $1 AND cons_id = $2 LIMIT 1)", u.UID, c.ID)
					break // Only use one potion per round
				}
			}
		}

		// Passive Regen Stacks
		if u.RegenStacks > 0 {
			heal := int(float64(u.RegenStacks*2) * healPenalty)
			u.CurrentHP += heal
			if u.CurrentHP > u.Stats.HP {
				u.CurrentHP = u.Stats.HP
			}
		}
		// Pets Regen
		for _, p := range u.Pets {
			if p.Stats.HP > 0 {
				p.Stats.HP += int(float64(p.Level*2) * healPenalty)
			}
		}
	}
}

// randomLootEligibleUser picks a random non-clone user to receive kill loot so
// co-op clones never trigger DB-persisting loot rolls (gear, gold, consumables,
// pity). Returns nil if no eligible recipient exists.
func randomLootEligibleUser(users []UserInCombat, source combatRandomSource) *UserInCombat {
	var eligible []*UserInCombat
	for i := range users {
		if !users[i].IsClone {
			eligible = append(eligible, &users[i])
		}
	}
	if len(eligible) == 0 {
		return nil
	}
	// #nosec G404 -- non-cryptographic loot recipient selection
	return eligible[source.IntN(len(eligible))]
}

func canUseHeldManaAbility(holdMana, bossPresent, manuallySelected bool) bool {
	return !holdMana || bossPresent || manuallySelected
}

func (b *Bot) userTurn(activeUsers []activeUser, mobs *[]*content.Mob, zone content.Zone, intensify, healPenalty float64, logs *[]string, totalUserDamage, totalMobDamage *int, avgLvl int, diffFactor float64, originalUsers []UserInCombat, loots *[]LootResult, round int, track *abyssFightTrack, liveActions map[string]abyssLiveAction, rand combatRandomSource) {
	previousLiveSkillTarget := ""
	for i := range activeUsers {
		au := &activeUsers[i]
		u := au.u
		if u.CurrentHP <= 0 {
			continue
		}
		liveAction, isLiveAction := liveActions[u.UID]
		comboFollowup := isLiveAction && liveSkillComboFollowup(previousLiveSkillTarget, liveAction)
		if isLiveAction && liveAction.Kind == "skill" && strings.HasPrefix(liveAction.TargetID, "enemy:") {
			previousLiveSkillTarget = liveAction.TargetID
		} else {
			previousLiveSkillTarget = ""
		}
		if au.defendingRound > 0 && au.defendingRound < round {
			u.DEFMod /= 1.5
			au.defendingRound = 0
		}
		bossPresent := abyssBossAlive(*mobs)
		// Scripted boss-phase stun: consume the flag and skip this turn.
		if au.Stunned {
			if au.treeBonus.Pct["stun_immunity"] > 0 {
				au.Stunned = false
				*logs = append(*logs, fmt.Sprintf("🛡️ %s resists the stun due to Stun Immunity!", u.Nickname))
			} else if abyssCombatant(u) && bossPresent && !au.stunbreakUsed {
				// AB-59 Stunbreak: one free stun cleanse per boss fight, acting
				// at 50% effectiveness this round (applied at dmgMult below).
				au.Stunned = false
				au.stunbreakUsed = true
				au.stunbrokenRound = round
				*logs = append(*logs, fmt.Sprintf("💪 %s breaks free of the stun through sheer will! (Stunbreak — acting at 50%% effectiveness)", u.Nickname))
			} else {
				au.Stunned = false
				*logs = append(*logs, i18n.T("bot.combat.stunned", u.Nickname))
				continue
			}
		}
		// Skill stuns zero SPD; stun immunity covers this path too so the
		// user still acts (SPD recovery happens either way).
		if u.Stats.SPD == 0 {
			u.Stats.SPD = 10
			if au.treeBonus.Pct["stun_immunity"] > 0 {
				*logs = append(*logs, fmt.Sprintf("🛡️ %s resists the stun due to Stun Immunity!", u.Nickname))
			} else if abyssCombatant(u) && bossPresent && !au.stunbreakUsed {
				// AB-59 Stunbreak (skill-stun path).
				au.stunbreakUsed = true
				au.stunbrokenRound = round
				*logs = append(*logs, fmt.Sprintf("💪 %s breaks free of the stun through sheer will! (Stunbreak — acting at 50%% effectiveness)", u.Nickname))
			} else {
				*logs = append(*logs, i18n.T("bot.combat.stunned", u.Nickname))
				continue
			}
		}

		// Mana regeneration: base 10 + 5% of flat MNA stat per round
		regen := 10 + u.Stats.MNA/20
		au.CurrentMana += regen
		if au.CurrentMana > au.MaxMana {
			au.CurrentMana = au.MaxMana
		}

		// Check for cursed gear
		isCursed := false
		for _, g := range u.Equipped {
			if g.Cursed {
				isCursed = true
				break
			}
		}
		if isCursed {
			// AB-54 Cursed mercy rule: the HP drain pauses below 20% HP.
			if abyssCombatant(u) && u.CurrentHP*5 < u.Stats.HP {
				if !au.cursedMercyLogged {
					au.cursedMercyLogged = true
					*logs = append(*logs, fmt.Sprintf("😮‍💨 The curse relents — %s is below 20%% HP, so the drain pauses (Cursed Mercy).", u.Nickname))
				}
			} else {
				loss := int(float64(u.Stats.HP) * 0.02)
				if loss < 1 {
					loss = 1
				}
				u.DamageTaken += max(loss, 0)
				u.CurrentHP -= loss
				*logs = append(*logs, fmt.Sprintf("💀 Cursed weapon drains %d HP from %s!", loss, u.Nickname))
				if u.CurrentHP <= 0 {
					u.CurrentHP = 0
					*logs = append(*logs, fmt.Sprintf("💀 %s has succumbed to their cursed weapon's corruption!", u.Nickname))
					continue
				}
			}
		}

		if isLiveAction && liveAction.Kind == "defend" {
			u.DEFMod *= 1.5
			au.defendingRound = round
			*logs = append(*logs, fmt.Sprintf("🛡️ %s braces for the enemy assault (+50%% DEF).", u.Nickname))
			charged := false
			for _, ultimate := range u.Ultimates {
				if ultimate != nil && ultimate.CurrentCooldown > 0 {
					ultimate.CurrentCooldown--
					charged = true
				}
			}
			if charged {
				*logs = append(*logs, fmt.Sprintf("🌟 Tactical charge: %s's ultimate cooldowns advance by 1.", u.Nickname))
			}
			continue
		}
		if isLiveAction && liveAction.Kind == "item" {
			if au.potionCooldown > 0 {
				*logs = append(*logs, fmt.Sprintf("🧪 %s's potion belt is still recovering.", u.Nickname))
				continue
			}
			target := liveAllyFromTarget(liveAction.TargetID, activeUsers)
			if !b.useLiveConsumable(au, target, liveAction.AbilityID, logs) {
				*logs = append(*logs, fmt.Sprintf("⚠️ %s's queued item is no longer available.", u.Nickname))
			} else {
				au.potionCooldown = 2
			}
			continue
		}
		if isLiveAction && liveAction.Kind == "relic" {
			if !b.useAbyssActiveRelic(au, round, logs) {
				*logs = append(*logs, fmt.Sprintf("⚠️ %s's relic has no charge remaining.", u.Nickname))
			}
			continue
		}
		if isLiveAction && liveAction.Kind == "companion" {
			target := liveMobFromTarget(liveAction.TargetID, *mobs)
			if target == nil || len(u.Pets) == 0 {
				*logs = append(*logs, fmt.Sprintf("⚠️ %s's companion command has no valid target.", u.Nickname))
				continue
			}
			au.petFocus = target.Name
			au.petFocusLogged = false
			*logs = append(*logs, fmt.Sprintf("🐾 %s commands their companion to focus %s.", u.Nickname, target.Name))
			continue
		}

		// Check for sentient weapon
		hasSentient := false
		sentientName := ""
		if mh, ok := u.Equipped[content.SlotMainHand]; ok && (mh.Eldritch || mh.Rarity >= content.RarityLegendary) {
			hasSentient = true
			sentientName = mh.Name
		}

		// #nosec G404 -- non-cryptographic flavour-text roll
		if u.CurrentHP < u.Stats.HP/3 && hasSentient && rand.Float64() < 0.3 {
			dialogues := []string{
				"The weapon whispers: 'Do not fail me now... there are still souls to consume...'",
				"The sentient weapon thrums with urgent energy: 'Stand strong, mortal!'",
			}
			// #nosec G404 -- non-cryptographic flavour-text roll
			*logs = append(*logs, fmt.Sprintf("💬 [%s]: %s", sentientName, dialogues[rand.IntN(len(dialogues))]))
		}

		// Zone Buff check
		uSTR := int(float64(u.Stats.STR) * u.STRMod)
		for _, eff := range zone.Effects {
			if eff.Type == content.ZoneBuff {
				uSTR = int(float64(uSTR) * (1.0 + eff.Power))
			}
		}

		// Momentum check (from simulation): 10% chance for 10% STR boost
		// #nosec G404
		if rand.Float64() < 0.1 {
			uSTR = int(float64(uSTR) * 1.1)
		}

		var lifesteal int
		var multiStrike int
		var mindControlLevel int
		var extraHits = 1

		var tName sql.NullString
		_ = b.DB.QueryRow("SELECT title FROM users WHERE client_uid=$1", u.UID).Scan(&tName)
		if tName.Valid {
			if t, ok := content.GetTitleByName(tName.String); ok {
				lifesteal = t.Lifesteal
				multiStrike = t.MultiStrike
			}
		}

		// Calculate Mind Control Level
		rows, _ := b.DB.Query("SELECT gear_id FROM user_gear WHERE client_uid = $1", u.UID)
		if rows != nil {
			for rows.Next() {
				var gid string
				if err := rows.Scan(&gid); err == nil {
					if g, ok := content.GetGearByID(gid); ok && g.Special == content.EffectMindControl {
						mindControlLevel += int(g.Rarity) + 1
					}
				}
			}
			_ = rows.Close()
		}
		for _, s := range u.Skills {
			if s.Special == content.EffectMindControl {
				mindControlLevel += int(s.Rarity) + 1
			}
		}

		for _, eff := range au.effects {
			if eff == content.EffectVampiric {
				lifesteal += 5
			}
		}

		// AB-62 Focus synergy: the auto-selected loot focus adds a matching
		// combat micro-bonus (gold focus → +2% crit, etc.).
		focusCrit := 0
		focusDmg := 1.0
		if abyssCombatant(u) {
			var focusLifesteal int
			focusCrit, focusDmg, focusLifesteal = abyssFocusMicroBonus(u.LootFocus)
			lifesteal += focusLifesteal
		}

		// Skill web: Souldrinker converts defensive stats into lifesteal —
		// +v of DEF as lifesteal % (v=0.01 → DEF 400 grants +4%), capped so
		// stacked-DEF builds can't become unkillable.
		if v := au.treeBonus.Pct["def_to_lifesteal"]; v > 0 {
			ls := int(float64(u.Stats.DEF) * v)
			if ls > 15 {
				ls = 15
			}
			lifesteal += ls
		}

		// #nosec G404
		if multiStrike > 0 && rand.IntN(100) < multiStrike { // #nosec G404
			extraHits = 2
			*logs = append(*logs, i18n.T("bot.combat.double_attack", u.Nickname))
		}

		for h := 0; h < extraHits; h++ {
			aliveMobs := b.getAliveMobs(*mobs)
			if len(aliveMobs) == 0 {
				break
			}
			liveHitKind := liveAction.Kind
			if isLiveAction && h > 0 && liveHitKind == "skill" {
				liveHitKind = "attack"
			}
			var target *content.Mob
			if isLiveAction {
				target = liveMobFromTarget(liveAction.TargetID, *mobs)
				isAllySkill := liveHitKind == "skill" && strings.HasPrefix(liveAction.TargetID, "ally:")
				if target == nil && !isAllySkill && liveHitKind != "attack" {
					*logs = append(*logs, fmt.Sprintf("⚠️ %s's target is no longer valid; the action fizzles.", u.Nickname))
					break
				}
			}
			if target == nil {
				if isLiveAction {
					target = aliveMobs[0]
				} else {
					// #nosec G404 -- legacy combat target selection
					target = aliveMobs[rand.IntN(len(aliveMobs))] // #nosec G404
				}
			}

			dmgMult := focusDmg
			if comboFollowup {
				dmgMult *= 1.15
				*logs = append(*logs, fmt.Sprintf("🔗 %s follows the party's skill order on the same target — combo +15%%!", u.Nickname))
			}
			ignoreDef := 0.0
			skipDamage := false
			// AB-59: a stunbroken delver acts at 50% effectiveness this round.
			if au.stunbrokenRound == round {
				dmgMult *= 0.5
			}
			for _, eff := range au.effects {
				if eff == content.EffectBerserk && u.CurrentHP < u.Stats.HP/2 {
					dmgMult += 0.2
				}
				if eff == content.EffectFragile {
					dmgMult += 0.3
				}
			}

			// Spell cost and cast check
			st := b.loadAbyssStats(u.UID)
			spellCostFor := func(base int) int {
				if base <= 0 {
					base = 20
				}
				if chest, ok := u.Equipped[content.SlotChest]; ok && chest.ID == "ABYSS_ARCHMAGE_ROBES" {
					base -= 5
				}
				base -= st.UpInsight * 2
				if v := au.treeBonus.Pct["skill_mana_cost"]; v > 0 {
					base = int(float64(base) * (1 - v))
				}
				if base < 5 {
					base = 5
				}
				return base
			}
			spellCost := spellCostFor(20)

			// AB-64 Hold mana: with the toggle on, save casts for boss floors
			// while grinding normal waves.
			holdCast := au.holdMana && !bossPresent
			if holdCast && !au.holdManaLogged {
				au.holdManaLogged = true
				*logs = append(*logs, fmt.Sprintf("🔷 %s holds their mana, saving their casts for the boss.", u.Nickname))
			}

			var selectedSkill *content.Skill
			manuallySelectedSkill := false
			if isLiveAction && liveHitKind == "skill" {
				if au.skillCooldowns[liveAction.AbilityID] == 0 {
					selectedSkill = findLiveSkill(u, liveAction.AbilityID)
				}
				manuallySelectedSkill = selectedSkill != nil
			} else if !isLiveAction && !holdCast && len(u.Skills) > 0 && au.CurrentMana >= spellCost && rand.Float64() < 0.3 { // #nosec G404
				available := make([]int, 0, len(u.Skills))
				for skillIndex := range u.Skills {
					if au.skillCooldowns[u.Skills[skillIndex].ID] == 0 {
						available = append(available, skillIndex)
					}
				}
				if len(available) > 0 {
					// #nosec G404 -- legacy automatic skill selection
					selectedSkill = &u.Skills[available[rand.IntN(len(available))]] // #nosec G404
				}
			}
			if selectedSkill != nil {
				spellCost = spellCostFor(selectedSkill.ManaCost)
			}
			if selectedSkill != nil && canUseHeldManaAbility(au.holdMana, bossPresent, manuallySelectedSkill) && au.CurrentMana >= spellCost {
				s := *selectedSkill
				// AB-52 Mana overflow: casting at full mana overcharges the spell +15%.
				overcharged := abyssCombatant(u) && au.CurrentMana >= au.MaxMana
				au.CurrentMana -= spellCost
				if abyssCombatant(u) {
					if mastery := b.recordAbyssSkillUse(u.UID, s.ID); mastery > 0 && mastery%25 == 0 && mastery <= 100 {
						*logs = append(*logs, fmt.Sprintf("🏅 %s mastery reached %d casts — its future runs gain +5%% power.", s.Name, mastery))
					}
				}

				// Spell Power scaling: +1% damage multiplier per 1 INT
				spellPowerMult := 1.0 + float64(u.Stats.INT)*0.01

				// Mage offhand / battery / shadow orb boosts spell power by +15%
				if oh, ok := u.Equipped[content.SlotOffHand]; ok && (strings.Contains(strings.ToLower(oh.Name), "orb") || strings.Contains(strings.ToLower(oh.Name), "battery")) {
					spellPowerMult *= 1.15
				}
				if t2, ok := u.Equipped[content.SlotTrinket2]; ok && (strings.Contains(strings.ToLower(t2.Name), "orb") || strings.Contains(strings.ToLower(t2.Name), "battery")) {
					spellPowerMult *= 1.15
				}

				castElement := abyssSkillElement(s, u.Equipped)
				repeatCount := 0
				if au.lastSkillID == s.ID {
					repeatCount = au.skillRepeatCount + 1
				}
				skillModifiers := calculateAbyssSkillModifiers(abyssSkillModifierContext{
					Skill: s, TreeBonus: au.treeBonus, Element: castElement,
					PreviousElement: au.lastCastElement, CurrentHP: u.CurrentHP,
					MaxHP: u.Stats.HP, PartySize: len(activeUsers), Round: round,
					RepeatCount: repeatCount,
				})

				dmgMult *= s.Power * spellPowerMult * skillModifiers.DamageMultiplier
				// Pure support skills (Power 0, e.g. Arcane Shield) only heal —
				// flag the hit so it deals no mob damage below.
				if s.Power == 0 {
					skipDamage = true
				}
				ignoreDef = skillModifiers.IgnoreDefense
				*logs = append(*logs, fmt.Sprintf("✨ %s cast %s (cost: %d Mana, Remaining: %d/%d). Spell Power: +%d%%!", u.Nickname, s.Name, spellCost, au.CurrentMana, au.MaxMana, int(float64(u.Stats.INT)*spellPowerMult)))
				if summary := abyssModifierSummary(skillModifiers.Active); summary != "" {
					*logs = append(*logs, "🌐 "+summary)
				}

				if s.HealPercent > 0 {
					healTarget := u
					if isLiveAction {
						if selected := liveAllyFromTarget(liveAction.TargetID, activeUsers); selected != nil {
							healTarget = selected
						}
					}
					healAmount := int(float64(healTarget.Stats.HP) * s.HealPercent * healPenalty * skillModifiers.HealingMultiplier)
					if healAmount < 0 {
						healAmount = 0
					}
					healTarget.CurrentHP += healAmount
					if healTarget.CurrentHP > healTarget.Stats.HP {
						healTarget.CurrentHP = healTarget.Stats.HP
					}
					*logs = append(*logs, fmt.Sprintf("💚 %s restores %d HP to %s (+%d%% of max HP) with %s!", u.Nickname, healAmount, healTarget.Nickname, int(s.HealPercent*100), s.Name))
				}

				// Combo System (Improvement 6)
				if au.lastSkillID != "" && au.lastSkillID == s.ID {
					dmgMult *= 1.25
					*logs = append(*logs, i18n.T("bot.combat.combo", s.Name))
				}
				au.lastSkillID = s.ID
				au.skillRepeatCount = repeatCount
				if skillModifiers.CooldownRounds > 0 {
					au.skillCooldowns[s.ID] = skillModifiers.CooldownRounds
				}

				// AB-52 Mana overflow: the overcharged spell lands +15% harder.
				if overcharged {
					dmgMult *= 1.15
					*logs = append(*logs, fmt.Sprintf("🔷 Mana overflow! %s's %s is overcharged +15%%!", u.Nickname, s.Name))
				}

				// AB-51 Elemental combo: two same-element skills in a row boost
				// the second +10%. Skills carry no element of their own, so the
				// cast element is the channelled weapon element at cast time.
				if reaction := abyssElementReaction(au.lastCastElement, castElement); abyssCombatant(u) && reaction != "" {
					dmgMult *= 1.20
					*logs = append(*logs, fmt.Sprintf("⚗️ %s reaction! %s combines %s and %s — +20%%!", reaction, u.Nickname, au.lastCastElement, castElement))
				} else if abyssCombatant(u) && au.lastCastElement != "" && au.lastCastElement == castElement {
					dmgMult *= 1.10
					*logs = append(*logs, fmt.Sprintf("🔥 Elemental combo! %s channels back-to-back %s magic — +10%%!", u.Nickname, castElement))
				}
				au.lastCastElement = castElement

				// #nosec G404
				if skillModifiers.StunChance > 0 && rand.Float64() < skillModifiers.StunChance { // #nosec G404
					*logs = append(*logs, i18n.T("bot.combat.stunned", target.Name))
					if target.StunRounds == 0 {
						target.PreStunSPD = target.Stats.SPD
					}
					target.Stats.SPD = 0
					target.StunRounds = max(target.StunRounds, max(1, skillModifiers.EffectRounds))
				}
			} else {
				au.lastSkillID = "" // Reset combo if no skill used
				au.skillRepeatCount = 0
				au.lastCastElement = ""
			}

			// Pure support cast (e.g. Arcane Shield): the heal already applied
			// above, so skip the rest of the attack resolution — no mob damage.
			if skipDamage {
				continue
			}

			// AB-53 Weapon swap mid-boss: when the MainHand element is weak
			// against the boss and the backpack holds a better matchup, swap
			// once per fight at the cost of the next action (1-round penalty).
			// The swap is in-memory for this fight; a manual trigger/persist
			// endpoint is web-layer work (noted in the group-C report).
			if abyssCombatant(u) && !au.weaponSwapped && (target.Type == content.MobBoss || target.Type == content.MobLegendary) {
				if mh, ok := u.Equipped[content.SlotMainHand]; ok && getElementMult(mh.Element, target.Element) < 1.0 {
					if backup, found := b.findBackupWeapon(u.UID, target.Element, mh.ID); found {
						u.Equipped[content.SlotMainHand] = backup
						au.weaponSwapped = true
						au.Stunned = true // 1-round penalty: the swap costs the next action
						*logs = append(*logs, fmt.Sprintf("🔄 %s swaps %s for %s mid-fight! (Loses their next action)", u.Nickname, mh.Name, backup.Name))
					}
				}
			}

			// Elemental System (Improvement 1)
			// Determine user's active element from MainHand
			userElement := content.ElementPhysical
			if mh, ok := u.Equipped[content.SlotMainHand]; ok {
				userElement = mh.Element
			}
			elementMult := getElementMult(userElement, target.Element)
			if elementMult > 1.0 {
				*logs = append(*logs, i18n.T("bot.combat.element_effective", userElement, target.Element))
			} else if elementMult < 1.0 {
				*logs = append(*logs, i18n.T("bot.combat.element_weak", userElement, target.Element))
			}
			dmgMult *= elementMult

			// Position Bonus (Improvement 2)
			if u.Position == content.PositionBackline {
				dmgMult *= 1.10 // 10% damage bonus for backline
			}

			// AB-68 Backstab: backline weapons deal +8% to mobs that didn't
			// target this delver last round.
			if abyssCombatant(u) && u.Position == content.PositionBackline && (au.lastAttackers == nil || !au.lastAttackers[target]) {
				dmgMult *= 1.08
				*logs = append(*logs, fmt.Sprintf("🗡️ %s strikes %s from an unseen angle! (Backstab +8%%)", u.Nickname, target.Name))
			}

			// Ultimate Skill activation: at most one fires per turn — the strongest
			// ready one — so stacking 3 ultimates widens uptime, not burst.
			var readyUlt *content.UltimateSkill
			manuallySelectedUltimate := false
			if isLiveAction && liveAction.Kind == "ultimate" {
				candidate := findLiveUltimate(u, liveAction.AbilityID)
				if candidate != nil && candidate.CurrentCooldown == 0 {
					readyUlt = candidate
					manuallySelectedUltimate = true
				}
			} else if !isLiveAction {
				for _, us := range u.Ultimates {
					if us.CurrentCooldown == 0 && (readyUlt == nil || us.Power > readyUlt.Power) {
						readyUlt = us
					}
				}
			}
			if readyUlt != nil && canUseHeldManaAbility(au.holdMana, bossPresent, manuallySelectedUltimate) {
				ultMult := readyUlt.Power
				if bonus := au.treeBonus.Pct["ult_damage"]; bonus > 0 {
					ultMult *= (1.0 + bonus)
				}
				dmgMult *= ultMult
				*logs = append(*logs, i18n.T("bot.combat.ultimate_activation", readyUlt.Name))
				au.lastUltRound = round // AB-70: interrupt window for boss summon telegraphs

				cooldownVal := readyUlt.CooldownRounds
				if red := clampRecovery(au.treeBonus.Pct["ult_cooldown"] + au.treeBonus.Pct["ultimate_charge"]); red > 0 {
					cooldownVal = int(float64(cooldownVal) * (1.0 - red))
					if cooldownVal < 2 {
						cooldownVal = 2
					}
				}
				readyUlt.CurrentCooldown = cooldownVal
			}

			// Weapon Scopes for Rangers: check if ranger/scope equipped
			hasScope := false
			for _, g := range u.Equipped {
				if strings.Contains(strings.ToLower(g.Name), "scope") {
					hasScope = true
					break
				}
			}
			if hasScope {
				dmgMult *= 1.15
			}
			if isLiveAction && h == 0 {
				weakpoint := resolveAbyssBossWeakpoint(liveAction.Weakpoint, target)
				dmgMult *= weakpoint.DamageMultiplier
				if weakpoint.Silence {
					silenceAbyssBoss(target)
				}
				if weakpoint.Log != "" {
					*logs = append(*logs, weakpoint.Log)
				}
			}

			effDef := float64(target.Stats.DEF) * target.DEFMod * (1.0 - ignoreDef)
			dmg := int((float64(uSTR)*dmgMult - effDef) * intensify)

			// Percentage-Based Damage Floor (15% of STR) to prevent DEF stalemates
			minDmg := int(float64(uSTR) * 0.15 * intensify)
			if dmg < minDmg {
				dmg = minDmg
			}
			if dmg < 1 {
				dmg = 1
			}

			// Abyss crit & fumble (AB-72 fumble recovery, AB-62 focus crit bonus).
			// The CRT stat is displayed as "Crit %" in the armory but was never
			// rolled in combat — the Abyss path now rolls it (×2 damage, capped).
			if abyssCombatant(u) {
				// #nosec G404 -- non-cryptographic combat roll
				if rand.Float64() < 0.03 {
					// Fumble: half damage, but the next hit gets +10% crit.
					dmg = dmg / 2
					if dmg < 1 {
						dmg = 1
					}
					au.fumbled = true
					*logs = append(*logs, fmt.Sprintf("😳 %s fumbles their attack! (Half damage — the next hit is fueled by embarrassed rage)", u.Nickname))
				} else {
					critPct := u.Stats.CRT + focusCrit
					if au.fumbled {
						critPct += 10 // AB-72 embarrassed rage
						au.fumbled = false
					}
					if critPct > 50 {
						critPct = 50
					}
					// #nosec G404 -- non-cryptographic combat roll
					if critPct > 0 && rand.IntN(100) < critPct {
						dmg *= 2
						*logs = append(*logs, fmt.Sprintf("💥 CRITICAL HIT! %s lands a devastating blow on %s!", u.Nickname, target.Name))
					}
				}
			}

			// Daily affix: Execute — strikes land 50% harder on targets below 30% HP.
			executeAffix := strings.Contains(u.FloorModifier, "execute") && target.MaxHP > 0 && target.Stats.HP*10 < target.MaxHP*3
			if executeAffix {
				dmg = dmg * 3 / 2
			}

			// Executioner affix: +25% damage to targets below 30% HP (stacks with
			// the Execute daily affix).
			executioner := false
			if target.MaxHP > 0 && target.Stats.HP*10 < target.MaxHP*3 {
				for _, eff := range au.effects {
					if eff == content.EffectExecutioner {
						dmg = dmg * 5 / 4
						executioner = true
						break
					}
				}
			}

			// AB-55: Executioner + Execute-affix days stack with a special log
			// flourish and an extra +5%.
			if executeAffix && executioner {
				dmg = dmg * 21 / 20
				if !au.execFlourished {
					au.execFlourished = true
					*logs = append(*logs, fmt.Sprintf("⚔️ EXECUTIONER'S FLOURISH! %s's blade sings on Execute day — both bonuses stack with an extra +5%%!", u.Nickname))
				}
			}

			killerBaseDamage := dmg
			dmg = abyssKillerDamage(killerBaseDamage, u, target)
			remainingHP := target.Stats.HP
			overkill := max(0, dmg-remainingHP)
			massiveOverkill := abyssOverkillHit(dmg, remainingHP)
			target.Stats.HP -= dmg
			applyAbyssBreakDamage(target, dmg, logs)
			*totalUserDamage += dmg

			// #nosec G404 -- non-cryptographic flavour-text roll
			if hasSentient && rand.Float64() < 0.25 {
				var sentientLog string
				if dmg > int(float64(uSTR)*0.8) {
					sentientLog = "The sentient weapon laughs: 'A fine strike! Feast on their pain!'"
				} else {
					sentientLog = "The sentient weapon whispers: 'Make it bleed more...'"
				}
				*logs = append(*logs, fmt.Sprintf("💬 [%s]: %s", sentientName, sentientLog))
			}

			// Chain Attack Logic for groups (3+ players)
			// #nosec G404
			if len(originalUsers) >= 3 && rand.Float64() < 0.3 { // #nosec G404
				others := b.getAliveMobs(*mobs)
				if len(others) > 1 {
					var chainTarget *content.Mob
					for _, xm := range others {
						if xm != target {
							chainTarget = xm
							break
						}
					}
					if chainTarget != nil {
						chainDmg := killerBaseDamage / 2
						if chainDmg < 1 {
							chainDmg = 1
						}
						chainDmg = abyssKillerDamage(chainDmg, u, chainTarget)
						chainTarget.Stats.HP -= chainDmg
						applyAbyssBreakDamage(chainTarget, chainDmg, logs)
						*totalUserDamage += chainDmg
					}
				}
			}

			// Mind Control Logic (Scale with level). At the three-pet cap, preserve
			// a successful capture as a restart-safe decision instead of silently
			// discarding it or overwriting an existing companion.
			captureLimit := abyssPetCaptureLimitWithBonus(mindControlLevel, int(au.treeBonus.Pct["pet_cap"]))
			if abyssCanAttemptPetCaptureAtLimit(len(u.Pets), captureLimit, au.petCaptureAttempted) &&
				target.Stats.HP > 0 && float64(target.Stats.HP) < float64(target.Level*20)*0.2 {
				// #nosec G404
				if rand.Float64() < abyssPetCaptureChance(target.Type) { // #nosec G404
					captured := false
					candidate := *target
					candidate.PetBoss = target.Type == content.MobBoss
					candidate.PetShiny = rand.Float64() < 0.01 // #nosec G404 -- cosmetic rarity roll
					abyssMindControlCapture(&candidate)
					result, err := b.persistAbyssPetCapture(u.UID, &candidate, captureLimit)
					switch {
					case err != nil:
						au.petCaptureAttempted = true
						*logs = append(*logs, "⚠️ The stable could not preserve this capture; the enemy breaks free.")
					case result == abyssPetCapturePreserved:
						au.petCaptureAttempted = true
						*logs = append(*logs, "🐾 Your stable already has a captured companion awaiting a decision.")
					case result == abyssPetCapturePending:
						au.petCaptureAttempted = true
						*target = candidate
						captured = true
						*logs = append(*logs, fmt.Sprintf("🐾 Stable full — %s is secured. Choose a companion to release after combat.", target.Name))
					case result == abyssPetCaptureRecruited:
						*target = candidate
						u.Pets = append(u.Pets, target)
						captured = true
					case result == abyssPetCaptureFull:
						au.petCaptureAttempted = true
						*logs = append(*logs, "🐾 Your current Mind Control bond cannot hold another companion.")
					}
					if captured {
						*logs = append(*logs, i18n.T("bot.combat.captive", target.Name))
						newMobs := make([]*content.Mob, 0, len(*mobs)-1)
						for _, xm := range *mobs {
							if xm != target {
								newMobs = append(newMobs, xm)
							}
						}
						*mobs = newMobs
					}
				}
			}

			if lifesteal > 0 {
				heal := int(float64(dmg) * float64(lifesteal) / 100.0 * healPenalty)
				if heal > 0 {
					beforeHeal := u.CurrentHP
					u.CurrentHP += heal
					if u.CurrentHP > u.Stats.HP {
						u.CurrentHP = u.Stats.HP
					}
					if restored := u.CurrentHP - beforeHeal; restored > 0 {
						*logs = append(*logs, fmt.Sprintf("[color=#41c97a]💚 %s lifesteal: +%d HP.[/color]", u.Nickname, restored))
					}
				}
			}

			if target.Stats.HP <= 0 {
				defeatLog := i18n.T("bot.combat.defeated", target.Name, u.Nickname)
				*logs = append(*logs, markAbyssOverkillLog(defeatLog, abyssCombatant(u) && massiveOverkill))
				// #nosec G404 -- non-cryptographic flavour-text roll
				if hasSentient && rand.Float64() < 0.4 {
					*logs = append(*logs, fmt.Sprintf("💬 [%s]: 'Their soul is ours now!'", sentientName))
				}
				// Weekly affix: Bloodlust heals the slayer for 20% of max HP on a kill.
				if strings.Contains(u.FloorModifier, "bloodlust") {
					u.CurrentHP += u.Stats.HP / 5
					if u.CurrentHP > u.Stats.HP {
						u.CurrentHP = u.Stats.HP
					}
				}
				// Award loot for every mob defeated, regardless of final outcome.
				// Clones (co-op helpers) are excluded so loot never persists for them.
				if winner := randomLootEligibleUser(originalUsers, rand); winner != nil {
					b.awardCombatLoot(winner, *target, zone, logs, loots)
				}
				b.handleDeathEffects(target, mobs, logs, avgLvl, diffFactor, activeUsers, rand)
			}
			if isLiveAction && overkill > 1 && liveHitKind != "item" {
				cleaveTarget := lowestHealthMobExcept(*mobs, target)
				if cleaveTarget != nil {
					cleaveDamage := max(1, max(0, killerBaseDamage-remainingHP)/2)
					cleaveDamage = abyssKillerDamage(cleaveDamage, u, cleaveTarget)
					cleaveOverkill := abyssOverkillHit(cleaveDamage, cleaveTarget.Stats.HP)
					cleaveTarget.Stats.HP -= cleaveDamage
					applyAbyssBreakDamage(cleaveTarget, cleaveDamage, logs)
					*totalUserDamage += cleaveDamage
					*logs = append(*logs, fmt.Sprintf("🪓 %s's overkill cleaves %s for %d damage!", u.Nickname, cleaveTarget.Name, cleaveDamage))
					if cleaveTarget.Stats.HP <= 0 {
						defeatLog := i18n.T("bot.combat.defeated", cleaveTarget.Name, u.Nickname)
						*logs = append(*logs, markAbyssOverkillLog(defeatLog, abyssCombatant(u) && cleaveOverkill))
						if winner := randomLootEligibleUser(originalUsers, rand); winner != nil {
							b.awardCombatLoot(winner, *cleaveTarget, zone, logs, loots)
						}
						b.handleDeathEffects(cleaveTarget, mobs, logs, avgLvl, diffFactor, activeUsers, rand)
					}
				}
			}
			if len(b.getAliveMobs(*mobs)) == 0 {
				break
			}
		}

		// Pet actions: each active formation slot carries one visible ability with
		// an independent cooldown, then falls back to its normal attack.
		for petIdx, p := range u.Pets {
			if p.Stats.HP <= 0 {
				continue
			}
			if abyssCombatant(u) && abyssPetNervous(p.Loyalty) && !au.petNervousLogged[p] {
				au.petNervousLogged[p] = true
				*logs = append(*logs, fmt.Sprintf("🐾 %s hangs back, eyes darting toward the exit. (Loyalty %d%% — betrayal risk)", p.Name, p.Loyalty))
			}

			// Loyalty steadily suppresses the base betrayal risk; the skill-web
			// reduction remains additive on top of that bond.
			betrayalChance := abyssPetBetrayalChance(p.Loyalty, au.treeBonus.Pct["pet_betrayal_reduce"])
			if rand.Float64() < betrayalChance { // #nosec G404
				p.Loyalty = max(0, p.Loyalty-5)
				// #nosec G404
				targetAU := activeUsers[rand.IntN(len(activeUsers))] // #nosec G404
				target := targetAU.u
				if target.CurrentHP > 0 {
					pdmg := int(float64(p.Stats.STR-target.Stats.DEF) * intensify)
					if pdmg < 1 {
						pdmg = 1
					}
					target.DamageTaken += pdmg
					target.CurrentHP -= pdmg
					*logs = append(*logs, i18n.T("bot.combat.rogue_pet_bite", p.Name, target.Nickname, pdmg))
					*totalMobDamage += pdmg
					if target.CurrentHP <= 0 {
						target.CurrentHP = 0
						if !b.checkUserRevive(target, logs) {
							*logs = append(*logs, i18n.T("bot.combat.slain_by_pet", target.Nickname, p.Name))
						}
					}
					continue
				}
			}

			ability, hasAbility := abyssPetAbilityForClass(petIdx+1, p.PetClass)
			abilityReady := hasAbility && au.petCooldowns[petIdx] == 0
			if abilityReady && ability.Kind == "heal" && abyssPetAutoskillEnabled(u.petHealEnabled, p.Name) {
				var bestTarget *UserInCombat
				lowestHPPct := 1.0
				for k := range activeUsers {
					targetU := activeUsers[k].u
					if targetU.CurrentHP > 0 && targetU.CurrentHP < targetU.Stats.HP {
						pct := float64(targetU.CurrentHP) / float64(targetU.Stats.HP)
						if pct < lowestHPPct {
							lowestHPPct = pct
							bestTarget = targetU
						}
					}
				}
				if bestTarget != nil {
					healAmt := int(float64(bestTarget.Stats.HP)*ability.PowerScale) + p.Level*3
					if healAmt < 10 {
						healAmt = 10
					}
					bestTarget.CurrentHP += healAmt
					if bestTarget.CurrentHP > bestTarget.Stats.HP {
						healAmt -= (bestTarget.CurrentHP - bestTarget.Stats.HP)
						bestTarget.CurrentHP = bestTarget.Stats.HP
					}
					setAbyssPetAbilityCooldown(au, petIdx, ability.Cooldown)
					*logs = append(*logs, fmt.Sprintf("✨ [color=#4caf50]%s's Pet %s casts %s on %s, restoring %d HP! (Cooldown: %d rounds)[/color]", u.Nickname, p.Name, ability.Name, bestTarget.Nickname, healAmt, ability.Cooldown))
					if bark := abyssPetBark(p.PetBark, p.Name, "heal"); bark != "" {
						*logs = append(*logs, bark)
					}
					continue
				}
			}

			aliveMobs := b.getAliveMobs(*mobs)
			if len(aliveMobs) == 0 {
				break
			}
			ptarget := petFocusTarget(aliveMobs, au.petFocus)
			if ptarget != nil && !au.petFocusLogged {
				au.petFocusLogged = true
				*logs = append(*logs, fmt.Sprintf("🎯 %s focuses %s on %s.", u.Nickname, p.Name, ptarget.Name))
			}
			if ptarget == nil {
				// #nosec G404 -- legacy random pet targeting fallback
				ptarget = aliveMobs[rand.IntN(len(aliveMobs))] // #nosec G404
			}
			petDmgMult := 1.0
			if bonus := au.treeBonus.Pct["pet_damage_pct"]; bonus > 0 {
				petDmgMult += bonus
			}
			petDmgMult *= abyssTreeActionMultiplier(au.treeBonus, "companion_skill_power")
			usesAttackAbility := abilityReady && ability.Kind == "attack"
			if usesAttackAbility {
				petDmgMult *= ability.PowerScale
			}
			pdmg := int(float64(p.Stats.STR-ptarget.Stats.DEF) * petDmgMult * intensify)
			if pdmg < 1 {
				pdmg = 1
			}
			pdmg = abyssKillerDamage(pdmg, u, ptarget)
			petOverkill := abyssOverkillHit(pdmg, ptarget.Stats.HP)
			ptarget.Stats.HP -= pdmg
			applyAbyssBreakDamage(ptarget, pdmg, logs)
			*totalUserDamage += pdmg
			if usesAttackAbility {
				setAbyssPetAbilityCooldown(au, petIdx, ability.Cooldown)
				*logs = append(*logs, fmt.Sprintf("🦷 %s uses %s on %s for %d damage! (Cooldown: %d rounds)", p.Name, ability.Name, ptarget.Name, pdmg, ability.Cooldown))
			}
			if ptarget.Stats.HP <= 0 {
				killLog := i18n.T("bot.combat.killed_by_pet", ptarget.Name, p.Name)
				*logs = append(*logs, markAbyssOverkillLog(killLog, abyssCombatant(u) && petOverkill))
				if bark := abyssPetBark(p.PetBark, p.Name, "kill"); bark != "" {
					*logs = append(*logs, bark)
				}
				// Clones (co-op helpers) are excluded so loot never persists for them.
				if winner := randomLootEligibleUser(originalUsers, rand); winner != nil {
					b.awardCombatLoot(winner, *ptarget, zone, logs, loots)
				}
				b.handleDeathEffects(ptarget, mobs, logs, avgLvl, diffFactor, activeUsers, rand)
			}
		}

		if len(b.getAliveMobs(*mobs)) == 0 {
			break
		}
	}

	// A rescued delver is one server-owned party action per round, not one
	// action per player. Resolve it after every real player and pet has acted.
	b.applyAbyssRescueSupportTurn(activeUsers, mobs, zone, intensify, logs, totalUserDamage, avgLvl, diffFactor, originalUsers, loots, rand)
}

func (b *Bot) mobTurn(activeUsers []activeUser, mobs []*content.Mob, zone content.Zone, intensify float64, logs *[]string, totalMobDamage, totalUserDamage *int, round int, ambush bool, track *abyssFightTrack, rand combatRandomSource) {
	livePlans := map[int]abyssLiveEnemyPlan{}
	if live := abyssLiveCombatFor(activeUsers); live != nil {
		livePlans = live.enemyPlansForRound(round)
	}
	for i := range activeUsers {
		activeUsers[i].lastAttackers = make(map[*content.Mob]bool)
	}
	// Ambush softening (all modes): track a per-target damage budget so a surprise
	// round can strip at most ambushDamageCapPct of each player's max HP, and clamp
	// it below so the ambush can never land the killing blow. Guarantees every
	// player gets to act at least once. Keyed by *UserInCombat so co-op partners
	// each get their own budget.
	var ambushBudget map[*UserInCombat]int
	if ambush {
		ambushBudget = make(map[*UserInCombat]int, len(activeUsers))
		for _, au := range activeUsers {
			budget := int(float64(au.u.Stats.HP) * ambushDamageCapPct)
			if budget < 1 {
				budget = 1
			}
			ambushBudget[au.u] = budget
		}
	}
	for mobIndex, m := range mobs {
		if m.Stats.HP <= 0 || m.Stats.SPD == 0 {
			if m.Stats.SPD == 0 {
				m.Stats.SPD = 10
				if m.MaxBreak > 0 && m.Break <= 0 {
					m.Break = m.MaxBreak
				}
			} // recover
			continue
		}

		plan, planned := livePlans[mobIndex]
		var targetAU *activeUser
		if planned {
			for i := range activeUsers {
				if activeUsers[i].u != nil && activeUsers[i].u.UID == plan.TargetUID && activeUsers[i].u.CurrentHP > 0 {
					targetAU = &activeUsers[i]
					break
				}
			}
		}
		if targetAU == nil {
			potentialTargets := make([]int, 0, len(activeUsers))
			for i := range activeUsers {
				if activeUsers[i].u != nil && activeUsers[i].u.CurrentHP > 0 {
					potentialTargets = append(potentialTargets, i)
				}
			}
			if len(potentialTargets) == 0 {
				continue
			}
			// #nosec G404 -- legacy combat target selection
			targetAU = &activeUsers[potentialTargets[rand.IntN(len(potentialTargets))]] // #nosec G404
		}
		target := targetAU.u
		targetAU.lastAttackers[m] = true

		// Physical Evasion for Backline
		if target.Position == content.PositionBackline && m.Element == content.ElementPhysical {
			// #nosec G404
			if rand.Float64() < 0.5 { // 50% extra miss chance for physical mobs vs backline
				*logs = append(*logs, i18n.T("bot.combat.stealth_shadow", target.Nickname, m.Name))
				continue
			}
		}

		// Task 60: Stealth check - skip first round mob attacks
		hasStealth := false
		for _, eff := range targetAU.effects {
			if eff == content.EffectStealth {
				hasStealth = true
				break
			}
		}
		if (round == 1 && hasStealth) || targetAU.stealthUntilRound >= round {
			continue
		}

		// Task 63: Parry check - 10% chance to take 0 damage and counter
		hasParry := false
		for _, eff := range targetAU.effects {
			if eff == content.EffectParry {
				hasParry = true
				break
			}
		}
		// #nosec G404
		if hasParry && rand.IntN(100) < 10 { // #nosec G404
			*logs = append(*logs, i18n.T("bot.combat.parried", target.Nickname, m.Name))
			counterDmg := int(float64(target.Stats.STR) * 0.5 * intensify)
			if counterDmg < 1 {
				counterDmg = 1
			}
			counterDmg = abyssKillerDamage(counterDmg, target, m)
			m.Stats.HP -= counterDmg
			*totalUserDamage += counterDmg
			if track != nil {
				track.counters += counterDmg
			}
			if abyssCombatant(target) {
				targetAU.parryCount++
				if targetAU.parryCount == 3 {
					targetAU.stealthUntilRound = round + 1
					*logs = append(*logs, fmt.Sprintf("🌫️ Parry mastery! %s vanishes into Stealth for the next round.", target.Nickname))
				}
			}
			continue
		}

		// #nosec G404
		// Dodge check - capped at 25%
		dodgeChance := target.Stats.DGE
		if dodgeChance > 25 {
			dodgeChance = 25
		}
		if rand.IntN(100) < dodgeChance { // #nosec G404
			continue
		} // #nosec G404

		dmgMult := 1.0
		spellIndex := -1
		silenced := consumeAbyssBossSilence(m)
		if silenced {
			*logs = append(*logs, fmt.Sprintf("🔇 %s's spell fails; the arms weakpoint remains disabled!", m.Name))
		} else if planned {
			spellIndex = plan.SpellIndex
		} else if len(m.Spells) > 0 && rand.Float64() < 0.2 { // #nosec G404
			// #nosec G404 -- legacy combat spell selection
			spellIndex = rand.IntN(len(m.Spells))
		}
		if spellIndex >= 0 && spellIndex < len(m.Spells) {
			s := m.Spells[spellIndex]
			dmgMult = s.Power
			*logs = append(*logs, i18n.T("bot.combat.cast_spell", m.Name, s.Name))
		}

		// Elemental System (Improvement 1)
		targetElement := content.ElementPhysical
		// Determine user's defensive element from Chest/OffHand
		if ch, ok := target.Equipped[content.SlotChest]; ok {
			targetElement = ch.Element
		}
		elementMult := getElementMult(m.Element, targetElement)
		dmgMult *= elementMult

		// Treasure Goblin Flee Logic
		if m.Type == content.MobTreasureGoblin && round >= 3 {
			flee := planned && plan.Intent.Kind == "flee"
			if !planned {
				flee = rand.Float64() < 0.3 // #nosec G404
			}
			if flee {
				*logs = append(*logs, i18n.T("bot.combat.goblin_flee"))
				m.Stats.HP = 0 // Remove from combat
				continue
			}
		}

		mSTR := int(float64(m.Stats.STR) * m.STRMod)
		// Zone Debuff check
		for _, eff := range zone.Effects {
			if eff.Type == content.ZoneDebuff {
				mSTR = int(float64(mSTR) * (1.0 - eff.Power))
			}
		}

		for _, eff := range m.Effects {
			switch eff {
			case content.EffectEnraged:
				mSTR = int(float64(mSTR) * 1.5)
			case content.EffectWeakened:
				mSTR = int(float64(mSTR) * 0.5)
			}
		}

		dmg := int((float64(mSTR)*dmgMult - float64(target.Stats.DEF)*target.DEFMod) * intensify)

		// Armor Plating chest attachments: flat damage reduction
		if chest, ok := target.Equipped[content.SlotChest]; ok && (strings.Contains(strings.ToLower(chest.Name), "plated") || strings.Contains(strings.ToLower(chest.Name), "plate") || strings.Contains(strings.ToLower(chest.Name), "aegis")) {
			dmg -= 30
		}

		// Frontline Defense Bonus (Improvement 2)
		if target.Position == content.PositionFrontline {
			dmg = int(float64(dmg) * 0.9) // 10% damage reduction for frontline
		}

		// Percentage-Based Damage Floor (15% of STR)
		minDmg := int(float64(mSTR) * 0.15 * intensify)
		if dmg < minDmg {
			dmg = minDmg
		}
		if dmg < 1 {
			dmg = 1
		}

		for _, eff := range m.Effects {
			// #nosec G404
			if eff == content.EffectBlinded && rand.Float64() < 0.5 {
				dmg = 0
			} // #nosec G404
		}

		// Boss resistance: shrink hits from boss-tier enemies by the target's
		// auto-scaling resistance so a boss can't erase a full-HP, well-geared
		// character in a single blow. Trash mobs are unaffected.
		if m.Type == content.MobBoss || m.Type == content.MobLegendary {
			if resist := bossResist(target.Level, target.Stats.Score()); resist > 0 {
				dmg = int(float64(dmg) * (1.0 - resist))
				if dmg < 0 {
					dmg = 0
				}
			}
		}
		if abyssCombatant(target) && runeWardResist(target.Equipped, m.Element) {
			dmg = dmg * 9 / 10
			if !targetAU.runeWardLogged {
				targetAU.runeWardLogged = true
				*logs = append(*logs, fmt.Sprintf("🔷 %s's three-rune ward resists 10%% %s damage.", target.Nickname, m.Element))
			}
		}
		if resist := defensiveRuneResistPct(target.Equipped, m.Element); abyssCombatant(target) && resist > 0 {
			dmg = dmg * (100 - resist) / 100
			if !targetAU.defRuneLogged {
				targetAU.defRuneLogged = true
				*logs = append(*logs, fmt.Sprintf("🛡️ %s's etched wards resist %d%% %s damage.", target.Nickname, resist, m.Element))
			}
		}

		// Ambush cap: limit total surprise-round damage to this target and never let
		// it reduce them below 1 HP, so a dense enemy group can't erase a full-HP
		// character before their first action.
		if ambush {
			if budget, ok := ambushBudget[target]; ok {
				if dmg > budget {
					dmg = budget
				}
				ambushBudget[target] = budget - dmg
			}
			if dmg >= target.CurrentHP {
				dmg = target.CurrentHP - 1
			}
			if dmg < 0 {
				dmg = 0
			}
		}

		// Daily affix: Iron Skin shaves 30% off every hit a delver takes.
		if strings.Contains(target.FloorModifier, "iron_skin") {
			dmg = dmg * 7 / 10
		}

		target.DamageTaken += max(dmg, 0)
		target.CurrentHP -= dmg
		*totalMobDamage += dmg

		// Daily affix: Vampiric mobs heal for 15% of the damage they deal.
		if dmg > 0 && strings.Contains(target.FloorModifier, "vampiric_mobs") {
			m.Stats.HP += dmg * 15 / 100
			if m.MaxHP > 0 && m.Stats.HP > m.MaxHP {
				m.Stats.HP = m.MaxHP
			}
		}

		// Check Revival
		if target.CurrentHP <= 0 {
			if !b.checkUserRevive(target, logs) {
				*logs = append(*logs, i18n.T("bot.combat.slain_by_mob", target.Nickname, m.Name))
			}
		}

		// Check for Spike Enamels on shield
		hasSpikes := false
		if oh, ok := target.Equipped[content.SlotOffHand]; ok && (strings.Contains(strings.ToLower(oh.Name), "spike") || strings.Contains(strings.ToLower(oh.Name), "gorgon") || strings.Contains(strings.ToLower(oh.Name), "aegis")) {
			hasSpikes = true
		}
		for _, eff := range targetAU.effects {
			if eff == content.EffectThorns && dmg > 0 {
				reflect := dmg / 10
				if hasSpikes {
					reflect = dmg * 3 / 10 // Thorns boosted to 30% with Spikes/shield!
				}
				if reflect < 1 {
					reflect = 1
				}
				reflect = abyssKillerDamage(reflect, target, m)
				m.Stats.HP -= reflect
				*totalUserDamage += reflect
				if track != nil {
					track.thorns += reflect
				}
			}
		}
	}
}

func (b *Bot) distributeRewards(users []UserInCombat, aus []activeUser, victory bool, totalUserDamage, totalMobDamage, killedXP int, initialMobs []*content.Mob, _ []*content.Mob, zone content.Zone, logs []string, avgLvl int, track *abyssFightTrack, rand combatRandomSource) ([]string, int, bool) {
	// Summarize Combat — centred header plus visual damage-share bars.
	totalDamage := totalUserDamage + totalMobDamage
	logs = append(logs, hr())
	logs = append(logs, centerHeader(i18n.T("bot.combat.summary_title")))
	logs = append(logs, i18n.T("bot.combat.summary_party", colorHeal(totalUserDamage), damageBar(totalUserDamage, totalDamage)))
	logs = append(logs, i18n.T("bot.combat.summary_mobs", colorDmg(totalMobDamage), damageBar(totalMobDamage, totalDamage)))
	logs = appendAbyssFightBreakdown(logs, track)

	// Update pity, quests, consumables AND persistent stats
	for i := range users {
		u := &users[i]
		// Save pets state
		if !u.IsClone {
			for _, p := range u.Pets {
				if victory && p != nil && p.Stats.HP > 0 {
					p.Loyalty = min(100, p.Loyalty+1)
				}
				b.updatePetState(u.UID, p)
			}
		}

		finalXP := 0
		if victory {
			if !u.IsClone {
				_, _ = b.DB.Exec("UPDATE users SET consecutive_losses = 0 WHERE client_uid = $1", u.UID)
				b.updateQuest(u.UID, "mobs_killed", len(initialMobs))
			}

			// Regen Stacks logic
			hasRegEffect := false
			_, _, _, _, effects := b.activeLootMult(u.UID, time.Now())
			for _, eff := range effects {
				if eff == content.EffectRegenStack {
					hasRegEffect = true
				}
			}
			if hasRegEffect {
				u.RegenStacks++
			}

		} else {
			if !u.IsClone {
				_, _ = b.DB.Exec("UPDATE users SET consecutive_losses = consecutive_losses + 1 WHERE client_uid = $1", u.UID)
				// Death Penalty: 25% of the XP required for the current level
				var curXP, curLevel int
				_ = b.DB.QueryRow("SELECT xp, level FROM users WHERE client_uid=$1", u.UID).Scan(&curXP, &curLevel)

				baseXP := leveling.XPForLevel(curLevel)
				nextXP := leveling.XPForLevel(curLevel + 1)
				levelSize := nextXP - baseXP

				penalty := int(float64(levelSize) * 0.25)
				if penalty < 10 {
					penalty = 10
				}

				finalXP -= penalty

				// Increase jackpot by 1% of lost XP value
				b.incrementJackpot("global", int64(penalty))

				logs = append(logs, deathPenaltyLine(u.Nickname, penalty))
			}
			u.CurrentHP = 0   // dead
			u.RegenStacks = 0 // lose stacks on death
		}

		// Gold Drop logic
		goldDrop := 0
		if victory {
			// Economic Inflation (Improvement 44)
			var totalGold int64
			_ = b.DB.QueryRow("SELECT SUM(gold) FROM users").Scan(&totalGold)
			inflationMult := 1.0
			if totalGold > 10000000 { // 10M Gold threshold
				inflationMult = 1.0 / (1.0 + float64(totalGold-10000000)/5000000.0)
			}

			for _, m := range initialMobs {
				baseGold := m.Level * 2
				switch m.Type {
				case content.MobBoss, content.MobLegendary:
					baseGold = m.Level * 10
				case content.MobTreasureGoblin:
					baseGold = m.Level * 25
				}
				// #nosec G404
				goldDrop += int(float64(baseGold) * (0.8 + rand.Float64()*0.4) * inflationMult)
			}

			// First Win of the Day Bonus
			if !u.IsClone {
				var lastWin sql.NullTime
				_ = b.DB.QueryRow("SELECT last_win FROM users WHERE client_uid=$1", u.UID).Scan(&lastWin)

				if !lastWin.Valid || lastWin.Time.Before(time.Now().Add(-24*time.Hour)) {
					// 5x Gold and XP multiplier for First Win
					goldDrop *= 5
					finalXP *= 5
					logs = append(logs, "🌟 FIRST WIN OF THE DAY! (5x Gold & XP)")
					_, _ = b.DB.Exec("UPDATE users SET last_win=NOW() WHERE client_uid=$1", u.UID)
				}
			}

			// VIP Gold Bonus
			vip, _ := b.getVIP(u.UID)
			if vip.Bonus > 0 {
				goldDrop = int(float64(goldDrop) * (1.0 + float64(vip.Bonus)/100.0))
			}

			u.Gold += int64(goldDrop)
		}

		if !u.IsClone {
			// Save per-ultimate cooldown state
			for _, us := range u.Ultimates {
				_, _ = b.DB.Exec("UPDATE user_ultimate_skills SET current_cooldown = $3 WHERE client_uid = $1 AND ultimate_id = $2", u.UID, us.ID, us.CurrentCooldown)
			}

			_, _ = b.DB.Exec("UPDATE users SET current_hp = $2, regen_stacks = $3, gold = users.gold + $4 WHERE client_uid = $1", u.UID, u.CurrentHP, u.RegenStacks, int64(goldDrop))

			// Skill web: Alchemist's Ritual — a lucky fight burns no
			// consumable charges.
			savePct := 0.0
			for k := range aus {
				if aus[k].u != nil && aus[k].u.UID == u.UID {
					savePct = aus[k].treeBonus.Pct["consumable_save"]
					break
				}
			}
			// #nosec G404 -- non-cryptographic charge-save roll
			if savePct <= 0 || rand.Float64() >= savePct {
				_, _ = b.DB.Exec("UPDATE user_consumables SET remaining_fights = remaining_fights - 1 WHERE client_uid = $1", u.UID)
				_, _ = b.DB.Exec("DELETE FROM user_consumables WHERE client_uid = $1 AND remaining_fights < 0", u.UID)
			}
		}

		if finalXP > 0 {
			// Improvement 24: Dynamic Level Scaling
			// Penalize high level players in low level zones
			if u.Level > avgLvl+20 {
				penalty := float64(u.Level-(avgLvl+20)) * 0.1
				if penalty > 1.0 {
					penalty = 1.0
				}
				finalXP = int(float64(finalXP) * (1.0 - penalty))
				if finalXP < 0 {
					finalXP = 0
				}
			}

			// Apply gear XP multipliers to combat rewards
			mult, _, _, _, _ := b.activeLootMult(u.UID, time.Now())
			if mult > 1.0 {
				finalXP = int(float64(finalXP) * mult)
			}
		}
		if finalXP != 0 && !u.IsClone {
			_, _ = b.awardXP(u.UID, "", finalXP)
		}
	}

	if victory {
		logs = append(logs, i18n.T("bot.combat.victory", len(initialMobs), zone.Name))

		// 1% chance for global jackpot on victory
		// #nosec G404
		if rand.Float64() < 0.01 {
			// Find a winner among participants
			// #nosec G404
			winner := users[rand.IntN(len(users))]
			jackpot := b.claimJackpot(winner.UID, "global")
			if jackpot > 0 {
				logs = append(logs, "🔥 GLOBAL JACKPOT WIN! "+winner.Nickname+" won "+FormatGold(jackpot)+"!")
			}
		}

		// Round-half-up so a small kill total still credits landed kills instead
		// of truncating the per-user share to zero.
		rc := realUserCount(users)
		return logs, (killedXP + rc/2) / rc, true
	}
	logs = append(logs, i18n.T("bot.combat.defeat", zone.Name))
	// Per-kill XP with a death tax: a lost fight still banks 25% of the XP from
	// the mobs that were actually killed (the other 75% is forfeit). Round-half-up
	// so the quartered share doesn't truncate small totals away to nothing.
	d := 4 * realUserCount(users)
	return logs, (killedXP + d/2) / d, false
}

// realUserCount counts non-clone participants. Co-op clones must not dilute the
// combat XP share, since only real delvers actually receive it. Always >= 1.
func realUserCount(users []UserInCombat) int {
	n := 0
	for i := range users {
		if !users[i].IsClone {
			n++
		}
	}
	if n < 1 {
		n = 1
	}
	return n
}

func (b *Bot) getAliveMobs(mobs []*content.Mob) []*content.Mob {
	var out []*content.Mob
	for _, m := range mobs {
		if m.Stats.HP > 0 {
			out = append(out, m)
		}
	}
	return out
}

func (b *Bot) handleDeathEffects(m *content.Mob, mobs *[]*content.Mob, logs *[]string, avgLvl int, diffFactor float64, users []activeUser, rand combatRandomSource) {
	if m.DeathEffect == nil {
		return
	}

	*logs = append(*logs, i18n.T("bot.combat.death_trigger", m.Name, m.DeathEffect.Type, m.DeathEffect.Name))

	switch m.DeathEffect.Type {
	case content.DeathSummon:
		count := 1
		if m.Type == content.MobCommon {
			count = 3
		} // Trash mobs summon hordes
		for i := 0; i < count; i++ {
			// Summoned mobs are lower tier
			lvl := avgLvl - 5
			if lvl < 1 {
				lvl = 1
			}
			newMob := content.SpawnMobWithRandom(lvl, false, diffFactor*0.7, rand)
			newMob.Name = "Summoned " + newMob.Name
			*mobs = append(*mobs, &newMob)
		}
		*logs = append(*logs, i18n.N("bot.combat.reinforcements", count))

	case content.DeathExplosion:
		dmg := m.Level * 10
		*logs = append(*logs, i18n.T("bot.combat.explosion_damage", dmg))
		for i := range users {
			target := users[i].u
			if target.CurrentHP <= 0 {
				continue
			}
			target.DamageTaken += max(dmg, 0)
			target.CurrentHP -= dmg
			if target.CurrentHP <= 0 {
				target.CurrentHP = 0
				if !b.checkUserRevive(target, logs) {
					*logs = append(*logs, i18n.T("bot.combat.slain_by_explosion", target.Nickname))
				}
			}
		}

	case content.DeathCurse:
		for i := range users {
			users[i].u.Stats.STR -= 10
			users[i].u.Stats.DEF -= 5
		}
		*logs = append(*logs, i18n.T("bot.combat.curse_weakens"))

	case content.DeathXP:
		*logs = append(*logs, i18n.T("bot.combat.bonus_xp_pulse"))

	case content.DeathLoot:
		*logs = append(*logs, i18n.T("bot.combat.shiny_items"))
	}
}

func (b *Bot) getConsumables(uid string) []content.Consumable {
	rows, err := b.DB.Query("SELECT cons_id, remaining_fights FROM user_consumables WHERE client_uid = $1", uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	loadout, restricted := b.abyssRunLoadout(uid)

	var out []content.Consumable
	for rows.Next() {
		var id string
		var rem int
		if err := rows.Scan(&id, &rem); err == nil {
			if c, ok := content.GetConsumableByID(id); ok {
				if restricted {
					count, ok := loadout[id]
					if !ok || count <= 0 {
						continue
					}
					c.Duration = count
				} else {
					c.Duration = rem
				}
				out = append(out, c)
			}
		}
	}
	return out
}

func (b *Bot) updateQuest(uid, qType string, progress int) {
	_, _ = b.DB.Exec(`INSERT INTO user_quests (client_uid, quest_type, progress, total_earned) 
	                  VALUES ($1, $2, $3, $3) 
	                  ON CONFLICT (client_uid, quest_type) 
	                  DO UPDATE SET progress = user_quests.progress + $3, total_earned = user_quests.total_earned + $3`,
		uid, qType, progress)
}

func streakMultiplier(streak int) float64 {
	switch {
	case streak >= 7:
		return 2.0
	case streak >= 5:
		return 1.5
	case streak >= 3:
		return 1.25
	default:
		return 1.0
	}
}

func serverMultiplier(onlineNormal int) float64 {
	humans := onlineNormal - 1
	if humans < 1 {
		humans = 1
	}
	// Simulation-tuned base: 1.5x for any human presence
	m := 1.5 + serverMultPerUser*float64(humans-1)
	if m > serverMultCap {
		m = serverMultCap
	}
	return m
}

func (b *Bot) updateStreak(uid string, today time.Time) int {
	var last sql.NullTime
	var streak int
	if err := b.DB.QueryRow("SELECT last_poke_date, streak_days FROM users WHERE client_uid=$1", uid).Scan(&last, &streak); err != nil {
		return 0
	}
	if last.Valid && sameDay(last.Time, today) {
		return streak
	}
	if last.Valid && sameDay(last.Time, today.AddDate(0, 0, -1)) {
		streak++
	} else {
		streak = 1
	}
	_, _ = b.DB.Exec("UPDATE users SET streak_days=$2, last_poke_date=$3 WHERE client_uid=$1", uid, streak, today)
	return streak
}

func (b *Bot) dailyLoginDue(uid string, today time.Time) bool {
	var last sql.NullTime
	if err := b.DB.QueryRow("SELECT last_login_date FROM users WHERE client_uid=$1", uid).Scan(&last); err != nil {
		return false
	}
	return !last.Valid || !sameDay(last.Time, today)
}

func (b *Bot) setLastLogin(uid string, today time.Time) {
	_, _ = b.DB.Exec("UPDATE users SET last_login_date=$2 WHERE client_uid=$1", uid, today)
}

func (b *Bot) ensureUserHasGear(uid string) {
	var count int
	// Count gear, skills, and check artifact
	_ = b.DB.QueryRow(`
		SELECT 
			(SELECT COUNT(*) FROM user_gear WHERE client_uid = $1) + 
			(SELECT COUNT(*) FROM user_skills WHERE client_uid = $1) + 
			(CASE WHEN artifact_name IS NOT NULL AND artifact_durability > 0 THEN 1 ELSE 0 END)
		FROM users WHERE client_uid = $1`, uid).Scan(&count)

	if count > 5 {
		return
	}

	// Get currently equipped slots
	rows, err := b.DB.Query("SELECT slot FROM user_gear WHERE client_uid = $1", uid)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()

	equipped := make(map[string]bool)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err == nil {
			equipped[s] = true
		}
	}

	// Fill ALL empty slots with Novice gear
	for _, slot := range content.AllSlots {
		slotStr := string(slot)
		if !equipped[slotStr] {
			gearID := fmt.Sprintf("B_%s", slotStr)
			if gear, ok := content.GetGearByID(gearID); ok {
				_, _ = b.DB.Exec("INSERT INTO user_gear (client_uid, slot, gear_id, durability) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
					uid, slotStr, gear.ID, gear.MaxDurability)
			}
		}
	}

	// Give 2 Random "Better" Items (Head Start)
	for i := 0; i < 2; i++ {
		g := content.RandomGearDrop()
		// Only give if it's actually an improvement or filling a low-tier slot
		if b.shouldEquip(uid, g) {
			_, _ = b.DB.Exec(`INSERT INTO user_gear (client_uid, slot, gear_id, durability) 
			                  VALUES ($1, $2, $3, $4) 
			                  ON CONFLICT (client_uid, slot) DO UPDATE SET gear_id = $3, durability = $4`,
				uid, string(g.Slot), g.ID, g.MaxDurability)
		}
	}

	// Also give Novice Skills if empty
	var skillCount int
	_ = b.DB.QueryRow("SELECT COUNT(*) FROM user_skills WHERE client_uid = $1", uid).Scan(&skillCount)
	if skillCount == 0 {
		_, _ = b.DB.Exec("INSERT INTO user_skills (client_uid, slot, skill_id) VALUES ($1, 1, 'S0_1'), ($1, 2, 'S0_2')", uid)
	}
}

func (b *Bot) applyDurabilityLoss(uid string, defeat bool) []string {
	var warnings []string
	if b.abyssRepairSubscriptionActive(uid, time.Now()) {
		return []string{"🛠️ Repair subscription covered all durability loss."}
	}
	var stats content.Stats
	var effects []content.ItemEffect
	_, stats, _, _, effects = b.activeLootMult(uid, time.Now())

	// Check for repair consumables and apply before durability loss
	consRows, err := b.DB.Query("SELECT cons_id FROM user_consumables WHERE client_uid = $1 AND cons_id IN ('P6','P7')", uid)
	if err == nil {
		var repairIDs []string
		for consRows.Next() {
			var cid string
			if err := consRows.Scan(&cid); err == nil {
				repairIDs = append(repairIDs, cid)
			}
		}
		_ = consRows.Close()

		for _, cid := range repairIDs {
			var repairAmt int
			switch cid {
			case "P6":
				repairAmt = 30
			case "P7":
				repairAmt = 75
			}

			// Only repair if there's actually broken gear
			b.ensureGearMaxDurability(uid)
			var brokenCount int
			_ = b.DB.QueryRow("SELECT COUNT(*) FROM user_gear WHERE client_uid = $1 AND durability < "+gearMaxDurExpr, uid).Scan(&brokenCount)
			if brokenCount > 0 {
				// Apply repair to all damaged gear (spread evenly)
				_, _ = b.DB.Exec("UPDATE user_gear SET durability = LEAST(durability + $2, "+gearMaxDurExpr+") WHERE client_uid = $1 AND durability < "+gearMaxDurExpr, uid, repairAmt/brokenCount)
				// Also repair artifact
				_, _ = b.DB.Exec("UPDATE users SET artifact_durability = LEAST(artifact_durability + 15, 30) WHERE client_uid = $1 AND artifact_durability > 0 AND artifact_durability < 30", uid)
				// Consume one repair kit
				_, _ = b.DB.Exec("DELETE FROM user_consumables WHERE ctid IN (SELECT ctid FROM user_consumables WHERE client_uid = $1 AND cons_id = $2 LIMIT 1)", uid, cid)
			}
		}
	}

	// Fragile check
	lossMult := 1
	for _, eff := range effects {
		if eff == content.EffectFragile {
			lossMult = 2
		}
	}

	// #nosec G404
	if rand.IntN(100) < stats.STA {
		return warnings
	} // #nosec G404

	baseLoss := duraLossPerFight * lossMult
	if defeat {
		baseLoss = duraLossPenalty * lossMult
	}

	// XP items lose durability faster: the higher the XP multiplier, the faster they decay.
	// This ensures powerful XP-boosting gear is a tradeoff (more XP but shorter lifespan).
	grows, gerr := b.DB.Query("SELECT gear_id, enchantment_id, item_data FROM user_gear WHERE client_uid = $1", uid)
	if gerr == nil {
		type gearLoss struct {
			gearID string
			loss   int
		}
		var losses []gearLoss
		for grows.Next() {
			var gearID string
			var enchID sql.NullString
			var itemData sql.NullString
			if grows.Scan(&gearID, &enchID, &itemData) == nil {
				itemLoss := baseLoss
				if gear, ok := b.makeGear(gearID, itemData); ok {
					if gear.Insured {
						itemLoss = 0
					} else if gear.XPMultiplier > 1.0 {
						xpPenalty := int((gear.XPMultiplier - 1.0) * 10) // e.g. 1.30x = +3 extra loss
						itemLoss += xpPenalty
					}
				}
				if enchID.Valid && enchID.String != "" {
					if ench, ok := content.GetEnchantmentByID(enchID.String); ok {
						// Enchantments with XP multiplier also increase durability loss
						if ench.XPMultiplier > 1.0 {
							xpPenalty := int((ench.XPMultiplier - 1.0) * 10)
							itemLoss += xpPenalty
						}
					}
				}
				losses = append(losses, gearLoss{gearID: gearID, loss: itemLoss})
			}
		}
		_ = grows.Close()

		// Apply individual durability losses
		for _, gl := range losses {
			var oldDura int
			_ = b.DB.QueryRow("SELECT durability FROM user_gear WHERE client_uid = $1 AND gear_id = $2", uid, gl.gearID).Scan(&oldDura)

			_, _ = b.DB.Exec("UPDATE user_gear SET durability = durability - $2 WHERE client_uid = $1 AND gear_id = $3", uid, gl.loss, gl.gearID)

			if gl.loss > baseLoss*2 && gl.loss >= 10 {
				if gear, ok := content.GetGearByID(gl.gearID); ok {
					warnings = append(warnings, fmt.Sprintf("⚠️ Your %s took heavy damage (-%d durability)!", gear.Name, gl.loss))
				}
			}

			if oldDura > 0 && oldDura-gl.loss <= 0 {
				if gear, ok := content.GetGearByID(gl.gearID); ok {
					warnings = append(warnings, fmt.Sprintf("💥 Your %s shattered into pieces!", gear.Name))
				}
			} else if oldDura > 10 && oldDura-gl.loss <= 10 {
				if gear, ok := content.GetGearByID(gl.gearID); ok {
					warnings = append(warnings, fmt.Sprintf("⚠️ Your %s is badly damaged and will break soon!", gear.Name))
				}
			}
		}
	} else {
		// Fallback: uniform loss if query fails
		_, _ = b.DB.Exec("UPDATE user_gear SET durability = durability - $2 WHERE client_uid = $1", uid, baseLoss)
	}

	_, _ = b.DB.Exec("DELETE FROM user_gear WHERE client_uid = $1 AND durability <= 0", uid)

	// Artifact break check
	var oldArtDura int
	var artName sql.NullString
	_ = b.DB.QueryRow("SELECT artifact_durability, artifact_name FROM users WHERE client_uid = $1", uid).Scan(&oldArtDura, &artName)

	_, _ = b.DB.Exec("UPDATE users SET artifact_durability = artifact_durability - $2 WHERE client_uid = $1 AND artifact_durability > 0", uid, baseLoss)

	if oldArtDura > 0 && oldArtDura-baseLoss <= 0 && artName.Valid && artName.String != "" {
		warnings = append(warnings, fmt.Sprintf("💥 Your %s shattered into pieces!", artName.String))
	} else if oldArtDura > 10 && oldArtDura-baseLoss <= 10 && artName.Valid && artName.String != "" {
		warnings = append(warnings, fmt.Sprintf("⚠️ Your %s is badly damaged and will break soon!", artName.String))
	}

	_, _ = b.DB.Exec("UPDATE users SET artifact_mult=1, artifact_name=NULL, artifact_durability=0 WHERE client_uid=$1 AND artifact_durability <= 0 AND artifact_name IS NOT NULL", uid)

	return warnings
}

func (b *Bot) calculateTotalStats(uid string, today time.Time) (content.Stats, float64, int, []string) {
	var level, prestige int
	_ = b.DB.QueryRow("SELECT level, prestige FROM users WHERE client_uid=$1", uid).Scan(&level, &prestige)
	base := content.Stats{
		HP: 100 + level*5, STR: 10 + level, DEF: 5 + level/2, SPD: 10 + level, LCK: level / 5,
		INT: level / 10, STA: level / 10, CRT: 5 + level/50, DGE: 5 + level/50,
	}

	// Apply Prestige Bonus
	if prestige > 0 {
		pMult := 1.0 + (float64(prestige) * prestigeStatBonus)
		base.HP = int(float64(base.HP) * pMult)
		base.STR = int(float64(base.STR) * pMult)
		base.DEF = int(float64(base.DEF) * pMult)
		base.SPD = int(float64(base.SPD) * pMult)
	}

	mult, lootStats, gearScore, notes, effects := b.activeLootMult(uid, today)
	totalStats := base.Add(lootStats)

	// Apply effects to stats
	for _, eff := range effects {
		switch eff {
		case content.EffectLucky:
			totalStats.LCK = int(float64(totalStats.LCK) * 1.1)
		case content.EffectQuick:
			totalStats.SPD = int(float64(totalStats.SPD) * 1.1)
		case content.EffectBulwark:
			totalStats.DEF = int(float64(totalStats.DEF) * 1.1)
		case content.EffectFocused:
			totalStats.CRT = int(float64(totalStats.CRT) * 1.1)
		case content.EffectRadiant:
			mult *= 1.1
		}
	}

	return totalStats, mult, gearScore, notes
}

func (b *Bot) activeLootMult(uid string, today time.Time) (float64, content.Stats, int, []string, []content.ItemEffect) {
	mult := 1.0
	var stats content.Stats
	var notes []string
	var effects []content.ItemEffect
	var gearScore int

	var title sql.NullString
	var tMult sql.NullFloat64
	var tExp sql.NullTime
	if err := b.DB.QueryRow("SELECT title, title_mult, title_expires FROM users WHERE client_uid=$1", uid).Scan(&title, &tMult, &tExp); err == nil {
		if tExp.Valid && !today.After(tExp.Time) && title.Valid {
			mult *= tMult.Float64
			notes = append(notes, i18n.T("bot.loot.multiplier_simple", title.String, tMult.Float64))
			if t, ok := content.GetTitleByName(title.String); ok {
				stats = stats.Add(t.Stats)
			}
		} else if title.Valid {
			_, _ = b.DB.Exec("UPDATE users SET title=NULL, title_mult=NULL, title_expires=NULL WHERE client_uid=$1", uid)
		}
	}
	var aMult sql.NullFloat64
	var aName sql.NullString
	var aDura int
	if err := b.DB.QueryRow("SELECT artifact_mult, artifact_name, artifact_durability FROM users WHERE client_uid=$1", uid).Scan(&aMult, &aName, &aDura); err == nil {
		if aName.Valid && aName.String != "" && aDura > 0 {
			mult *= aMult.Float64
			notes = append(notes, i18n.T("bot.loot.multiplier_with_dur", aName.String, aMult.Float64, int(aDura)))
			if art, ok := content.GetArtifactByName(aName.String); ok {
				stats = stats.Add(art.Stats)
				gearScore += art.Stats.Score()
				if art.Special != content.EffectNone {
					effects = append(effects, art.Special)
				}
			}
		}
	}
	// Calculate gear XP multiplier
	// Only Rare+ items provide XP bonuses (Common/Uncommon have 1.0-1.05x)
	// Max possible from gear: 30 slots × 1.30x = ~2600x (capped by rarity distribution)
	abyssSetCounts := make(map[string]int)
	equippedGear := make(map[content.GearSlot]content.Gear)
	rows, err := b.DB.Query("SELECT slot, gear_id, durability, enchantment_id, item_data FROM user_gear WHERE client_uid = $1", uid)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var slot, gearID string
			var dura int
			var enchID sql.NullString
			var itemData sql.NullString
			if err := rows.Scan(&slot, &gearID, &dura, &enchID, &itemData); err == nil {
				if gear, ok := b.makeGear(gearID, itemData); ok {
					equippedSlot := content.GearSlot(slot)
					if content.IsPetGearSlot(equippedSlot) {
						continue
					}
					equippedGear[equippedSlot] = gear
					if content.IsAbyssGearID(gearID) {
						abyssSetCounts[gear.EffectiveSetID()]++
					}
					// Define which slots can have high XP multipliers (more than 20%)
					highXPSlots := map[content.GearSlot]bool{
						content.SlotMainHand: true,
						content.SlotChest:    true,
						content.SlotHead:     true,
						content.SlotLegs:     true,
						content.SlotFeet:     true,
						content.SlotFinger1:  true,
					}

					// Apply XP multiplier based on slot and rarity restrictions
					xpMultiplier := 1.0
					if gear.Rarity >= content.RarityRare {
						if highXPSlots[gear.Slot] {
							// High XP slots can have full multiplier
							xpMultiplier = gear.XPMultiplier
						} else {
							// Other slots limited to max 1-2% XP bonus
							if gear.XPMultiplier > 1.02 {
								xpMultiplier = 1.02
							} else {
								xpMultiplier = gear.XPMultiplier
							}
						}
						mult *= xpMultiplier
					}

					// Only show gear with XP multiplier in notes, but without durability
					if xpMultiplier > 1.0 {
						notes = append(notes, i18n.T("bot.loot.multiplier_simple", gear.Name, xpMultiplier))
					}

					stats = stats.Add(gear.Stats)
					gearScore += gear.Stats.Score()
					if bonus := sentimentalValueBonus(gear, today); bonus != (content.Stats{}) {
						stats = stats.Add(bonus)
						gearScore += bonus.Score()
						notes = append(notes, fmt.Sprintf("💛 Broken in: %s (+1%% stats)", gear.Name))
					}
					if gear.Special != content.EffectNone {
						effects = append(effects, gear.Special)
					}
					// High-tier gear (Mythic/Divine) can carry extra bonus affixes.
					for _, be := range gear.BonusEffects {
						if be != content.EffectNone {
							effects = append(effects, be)
						}
					}

					if enchID.Valid && enchID.String != "" {
						if ench, ok := content.GetEnchantmentByID(enchID.String); ok {
							// Apply doubled stats at runtime (Unstable Enchantments mechanic)
							eStats := ench.Stats
							eStats.STR *= 2
							eStats.SPD *= 2
							stats = stats.Add(eStats)
							gearScore += eStats.Score()
							mult *= ench.XPMultiplier // Apply enchantment XP penalty
							if ench.Special != content.EffectNone {
								effects = append(effects, ench.Special)
							}

							eName := ench.Name
							if ench.Special != content.EffectNone {
								eName = i18n.T("bot.loot.pool_prefix", ench.Special, eName)
							}
							notes = append(notes, i18n.T("bot.loot.pool_prefix_xp", eName, gear.Name, ench.XPMultiplier))
						}
					}
				}
			}
		}
	}
	if resonanceBonus, resonance := abyssGemResonanceBonus(equippedGear); resonanceBonus != (content.Stats{}) {
		stats = stats.Add(resonanceBonus)
		gearScore += resonanceBonus.Score()
		families := make([]string, 0, len(resonance))
		for family, count := range resonance {
			if count >= 3 {
				families = append(families, fmt.Sprintf("%s ×%d", family, count))
			}
		}
		sort.Strings(families)
		notes = append(notes, "💎 Gem Resonance: "+strings.Join(families, ", "))
	}

	// Abyss set bonus: equipping multiple ABYSS_ pieces grants cumulative stat
	// tiers, applied here so the set(s) work everywhere the character fights.
	// Named sets (predator/warden) and the flat legacy set for untagged items
	// are all folded together — see content.AbyssSetBonusBySet.
	if bonus, reachedBySet := content.AbyssSetBonusBySet(abyssSetCounts); len(reachedBySet) > 0 {
		stats = stats.Add(bonus)
		gearScore += bonus.Score()
		for setID, tier := range reachedBySet {
			notes = append(notes, fmt.Sprintf("🕳️ Abyss Set: %s (%d pieces)", setID, tier))
		}
	}

	// Abyss win-streak buff: consecutive floor wins within the current run grant a
	// small stacking combat bonus (see abyssStreakBuff), applied wherever the
	// character fights so it's live during real Abyss combat.
	var winStreak int
	_ = b.DB.QueryRow("SELECT abyss_win_streak FROM users WHERE client_uid=$1", uid).Scan(&winStreak)
	if streakBonus := abyssStreakBuff(winStreak); streakBonus != (content.Stats{}) {
		stats = stats.Add(streakBonus)
		gearScore += streakBonus.Score()
		notes = append(notes, fmt.Sprintf("🔥 Abyss Streak (%d wins)", winStreak))
	}

	// Skills also provide effects
	srows, err := b.DB.Query("SELECT skill_id FROM user_skills WHERE client_uid = $1", uid)
	if err == nil {
		defer func() { _ = srows.Close() }()
		for srows.Next() {
			var sid string
			if err := srows.Scan(&sid); err == nil {
				if s, ok := content.GetSkillByID(sid); ok {
					if s.Special != content.EffectNone {
						effects = append(effects, s.Special)
					}
				}
			}
		}
	}

	// Active ultimates also provide their effects
	for _, us := range b.getActiveUltimates(uid) {
		if us.Special != content.EffectNone {
			effects = append(effects, us.Special)
		}
	}

	// Apply active elixir buffs
	crows, err := b.DB.Query("SELECT cons_id, remaining_fights FROM user_consumables WHERE client_uid = $1 AND remaining_fights > 0", uid)
	if err == nil {
		defer func() { _ = crows.Close() }()
		for crows.Next() {
			var cid string
			var rem int
			if err := crows.Scan(&cid, &rem); err == nil {
				switch cid {
				case "strength_elixir":
					stats.STR += 15
					notes = append(notes, i18n.T("bot.loot.multiplier_simple", "Strength Elixir", 1.0))
				case "iron_skin_brew":
					stats.DEF += 10
					notes = append(notes, i18n.T("bot.loot.multiplier_simple", "Iron Skin Brew", 1.0))
				case "speed_elixir":
					stats.SPD += 25
					notes = append(notes, i18n.T("bot.loot.multiplier_simple", "Speed Elixir", 1.0))
				case "intellect_elixir":
					stats.INT += 20
					notes = append(notes, i18n.T("bot.loot.multiplier_simple", "Intellect Elixir", 1.0))
				case "lucky_draught":
					stats.LCK += 20
					notes = append(notes, i18n.T("bot.loot.multiplier_simple", "Lucky Draught", 1.0))
				case "giant_strength_elixir":
					stats.STR += 40
					notes = append(notes, i18n.T("bot.loot.multiplier_simple", "Giant Strength Elixir", 1.0))
				}
			}
		}
	}

	return mult, stats, gearScore, notes, effects
}

func (b *Bot) rollLootForUser(uid string, mob content.Mob, zoneDifficulty float64, focus string) (string, string) {
	var results []string
	var pokes []string
	count := 1
	if mob.Type == content.MobBoss {
		count = 3
		// Guaranteed consumable
		c := content.RandomConsumable()
		b.grantConsumable(uid, c.ID, c.Duration)
		results = append(results, i18n.T("bot.loot.item", c.Name, c.ID))
	}
	if mob.Type == content.MobLegendary {
		count = 5
		c := content.RandomConsumable()
		b.grantConsumable(uid, c.ID, c.Duration)
		results = append(results, i18n.T("bot.loot.item", c.Name, c.ID))
	}
	if mob.Type == content.MobTreasureGoblin {
		count = 2
	}

	// Double Loot Title check
	var tName sql.NullString
	_ = b.DB.QueryRow("SELECT title FROM users WHERE client_uid=$1", uid).Scan(&tName)

	vip, _ := b.getVIP(uid)

	// Effect check
	_, _, _, _, effects := b.activeLootMult(uid, time.Now())
	lootFindBonus := 0.0
	for _, eff := range effects {
		if eff == content.EffectTreasureHunter {
			lootFindBonus += 0.05
		}
	}

	// Loot Quality Multiplier: Higher difficulty = better chance for Rares
	qualityMult := zoneDifficulty
	if qualityMult < 1.0 {
		qualityMult = 1.0
	}

	if focus == "loot" {
		qualityMult *= 1.2
		lootFindBonus += 0.50
	}

	// Low-level / low-difficulty content drops fewer high-rarity items: the rare
	// tiers (ultimate, title, unique, artifact, enchant) are scaled down until the
	// fought level catches up. Common gear/consumable odds are left untouched.
	rareScale := lootRarityScale(mob.Level)

	if tName.Valid {
		if t, ok := content.GetTitleByName(tName.String); ok && t.DoubleLoot {
			count *= 2
		}
	}

	var ultPity, artPity int
	_ = b.DB.QueryRow("SELECT ultimate_pity, artifact_pity FROM users WHERE client_uid=$1", uid).Scan(&ultPity, &artPity)
	ultDropped, artDropped := false, false

	for i := 0; i < count; i++ {
		// #nosec G404
		r := rand.Float64() - lootFindBonus // #nosec G404

		// Gold focus is handled before the generic treasure-goblin branch so the
		// richer goblin payout below is actually reachable (the goblin branch's
		// own `continue` would otherwise skip it).
		if focus == "gold" {
			// If gold focus, skip item rolls but always award a gold bonus.
			// Treasure goblins get an even richer payout.
			if mob.Type == content.MobTreasureGoblin {
				// #nosec G404 -- non-cryptographic reward roll
				gold := int64(1000 + rand.IntN(2000))
				if vip.Bonus > 0 {
					gold = int64(float64(gold) * (1.0 + float64(vip.Bonus)/100.0))
				}
				_, _ = b.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid = $2", gold, uid)
				results = append(results, fmt.Sprintf("💰 %d gold", gold))
			} else {
				// Standard gold reward for non-goblin mobs in gold-focus mode
				// #nosec G404 -- non-cryptographic reward roll
				baseGold := int64(10 + rand.IntN(mob.RewardXP/2+10))
				if vip.Bonus > 0 {
					baseGold = int64(float64(baseGold) * (1.0 + float64(vip.Bonus)/100.0))
				}
				_, _ = b.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid = $2", baseGold, uid)
				results = append(results, fmt.Sprintf("💰 %d gold", baseGold))
			}
			continue
		}

		if mob.Type == content.MobTreasureGoblin {
			gold := int64(1000 + rand.IntN(2000)) // #nosec G404 - non-cryptographic gold calculation
			if vip.Bonus > 0 {
				gold = int64(float64(gold) * (1.0 + float64(vip.Bonus)/100.0))
			}
			_, _ = b.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid = $2", gold, uid)
			results = append(results, fmt.Sprintf("💰 %d gold", gold))
			continue
		}

		lootFound := false
		// ... rest of loop ...
		// Checks ordered by ascending threshold so smaller chances are evaluated first
		// Thresholds: title=0.005, ultimateSkill=0.005, uniqueItem=0.01, artifact=0.01, ench=0.02, skill=0.05, cons=0.1, gear=0.10

		effUltChance := ultimateSkillChance*qualityMult*rareScale + float64(ultPity)*0.001 // 0.1% extra per pity point
		effArtChance := artifactChance*qualityMult*rareScale + float64(artPity)*0.002      // 0.2% extra per pity point

		if r < effUltChance {
			// Ultimate skill drop (0.5%)
			us := content.RandomUltimateSkill()
			var exists bool
			_ = b.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM user_ultimate_skills WHERE client_uid=$1 AND ultimate_id=$2)", uid, us.ID).Scan(&exists)
			if !exists {
				_, _ = b.DB.Exec("INSERT INTO user_ultimate_skills (client_uid, ultimate_id) VALUES ($1, $2)", uid, us.ID)
				_, _ = b.DB.Exec("UPDATE users SET ultimate_skills_count = ultimate_skills_count + 1 WHERE client_uid=$1", uid)
				if b.activateUltimateIfSlotFree(uid, us.ID) {
					results = append(results, i18n.T("bot.loot.ultimate_equipped", us.Name))
				} else {
					results = append(results, i18n.T("bot.loot.ultimate_collected", us.Name))
				}
				if us.Rarity >= content.RarityLegendary {
					pokes = append(pokes, i18n.T("bot.loot.major_ultimate", us.Name))
				}
			} else {
				// Duplicate Ultimates -> List on AH
				b.autoListUnwantedItems(uid, us)
				results = append(results, i18n.T("bot.loot.duplicate_ultimate_ah", us.Name))
			}
			lootFound = true
			ultDropped = true
		} else if r < titleChance*qualityMult*rareScale {
			t := content.RandomTitle()
			// Only credit the drop if it actually landed: the WHERE guard rejects the
			// grant when an unexpired title is already equipped, so crediting it
			// unconditionally would tell the player they got a title they never received.
			if res, execErr := b.DB.Exec("UPDATE users SET title=$2, title_mult=$3, title_expires=NOW() + INTERVAL '7 days', title_source='xp' WHERE client_uid=$1 AND (title IS NULL OR title_expires < NOW())", uid, t.Name, t.XPMultiplier); execErr == nil {
				// res is nil when Exec errors, so only read RowsAffected on success.
				if n, _ := res.RowsAffected(); n > 0 {
					results = append(results, i18n.T("bot.loot.title", t.Name, t.Name))
					lootFound = true
				}
			}
		} else if r < uniqueItemChance*qualityMult*rareScale {
			// Unique item drop (1%)
			ui := content.RandomUniqueItem()
			var exists bool
			_ = b.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM user_unique_items WHERE client_uid=$1 AND item_name=$2)", uid, ui.Name).Scan(&exists)
			if !exists {
				_, _ = b.DB.Exec("INSERT INTO user_unique_items (client_uid, item_name, rarity, power) VALUES ($1, $2, $3, $4)", uid, ui.Name, ui.Rarity, ui.Power)
				_, _ = b.DB.Exec("UPDATE users SET unique_items_count = unique_items_count + 1 WHERE client_uid=$1", uid)
				results = append(results, i18n.T("bot.loot.unique", ui.Name, ui.Name, ui.Rarity.String()))
				if ui.Rarity >= content.RarityLegendary {
					pokes = append(pokes, i18n.T("bot.loot.unique_drop", ui.Name))
				}
			} else {
				// Duplicate Uniques -> List on AH
				b.autoListUnwantedItems(uid, ui)
				results = append(results, i18n.T("bot.loot.duplicate_unique_ah", ui.Name))
			}
			lootFound = true
		} else if r < effArtChance {
			a := content.RandomArtifact()
			a.Stats.HP = int(float64(a.Stats.HP) * zoneDifficulty)
			a.Stats.STR = int(float64(a.Stats.STR) * zoneDifficulty)
			a.Stats.DEF = int(float64(a.Stats.DEF) * zoneDifficulty)
			_, _ = b.DB.Exec("UPDATE users SET artifact_mult=$2, artifact_name=$3, artifact_durability=$4 WHERE client_uid=$1", uid, a.Mult, a.Name, a.MaxDurability)
			results = append(results, i18n.T("bot.loot.artifact", a.Name, a.Name))
			pokes = append(pokes, i18n.T("bot.loot.artifact_found", a.Name))
			lootFound = true
			artDropped = true
		} else if r < enchChance*qualityMult*rareScale {
			ench := content.RandomEnchantment()
			ench.Stats.STR = int(float64(ench.Stats.STR) * zoneDifficulty)
			ench.Stats.SPD = int(float64(ench.Stats.SPD) * zoneDifficulty)
			if slot, ok := b.applyEnchantment(uid, ench); ok {
				results = append(results, i18n.T("bot.loot.enchanted", slot, ench.Name, ench.Name))
			} else {
				// Unwanted Enchantments -> List on AH
				b.autoListUnwantedItems(uid, ench)
				results = append(results, i18n.T("bot.loot.unwanted_enchant_ah", ench.Name))
			}
			lootFound = true
		} else if r < skillChance*qualityMult {
			s := content.RandomSkill()
			s.Power *= zoneDifficulty
			if slot, ok := b.equipSkill(uid, s); ok {
				results = append(results, i18n.T("bot.loot.learned_skill", s.Name, s.Name, slot))
			} else {
				// Unwanted Skills -> List on AH
				b.autoListUnwantedItems(uid, s)
				results = append(results, i18n.T("bot.loot.unwanted_skill_ah", s.Name))
			}
			lootFound = true
		} else if r < consChance*qualityMult {
			c := content.RandomConsumable()
			b.grantConsumable(uid, c.ID, c.Duration)
			results = append(results, i18n.T("bot.loot.item", c.Name, c.ID))
			lootFound = true
		} else if r < gearChance*qualityMult {
			g := content.RandomGearDrop()
			// #nosec G404 -- non-cryptographic loot roll
			if b.loadAbyssRun(uid).Active && rand.Float64() < 0.20 {
				g = content.RandomAbyssGearDrop()
			}
			g.Stats.HP = int(float64(g.Stats.HP) * zoneDifficulty)
			g.Stats.STR = int(float64(g.Stats.STR) * zoneDifficulty)
			g.Stats.DEF = int(float64(g.Stats.DEF) * zoneDifficulty)
			g.Stats.SPD = int(float64(g.Stats.SPD) * zoneDifficulty)

			result := b.awardGearDrop(uid, g)
			results = append(results, fmt.Sprintf("%s%s [s:%s] (gs:%d CR:%.1f R:[color=%s]%s[/color])",
				result.Prefix, g.Name, string(g.Slot), g.Stats.Score(), g.CombatRating(), g.Rarity.Color(), g.Rarity.String()))

			if result.Action == "equipped" && g.Rarity >= content.RarityLegendary {
				pokes = append(pokes, i18n.T("bot.loot.legendary_equipped", g.Name))
			}
			lootFound = true
		}

		// 100% Drop Guarantee: If nothing else found, drop a Common item
		if !lootFound && focus != "gold" {
			// #nosec G404
			if rand.Float64() < 0.7 { // #nosec G404
				// Drop a basic common gear
				g := content.RandomStarterGear()
				if b.shouldEquip(uid, g) {
					_ = b.equipGear(b.DB, uid, g, g.MaxDurability, nil)
					results = append(results, i18n.T("bot.loot.found", g.Name, string(g.Slot), g.Stats.Score(), g.CombatRating(), g.Rarity.String()))
				} else {
					b.autoListUnwantedItems(uid, g)
					results = append(results, i18n.T("bot.loot.listed_ah", g.Name, string(g.Slot), g.Rarity.Color(), g.Rarity.String()))
				}
			} else {
				results = append(results, i18n.T("bot.loot.small_health_potion"))
				_, _ = b.DB.Exec("INSERT INTO user_consumables (client_uid, cons_id, remaining_fights) VALUES ($1, 'P1', 0) ON CONFLICT DO NOTHING", uid)
			}
		}
	}

	// Gold-focus rolls skip every item roll, so they must not advance pity (which
	// would otherwise inflate ultimate/artifact odds for free).
	if focus != "gold" {
		if ultDropped {
			ultPity = 0
		} else {
			ultPity += count
		}
		if artDropped {
			artPity = 0
		} else {
			artPity += count
		}
		_, _ = b.DB.Exec("UPDATE users SET ultimate_pity=$2, artifact_pity=$3 WHERE client_uid=$1", uid, ultPity, artPity)
	}

	resStr := ""
	if len(results) > 0 {
		resStr = strings.Join(results, ", ")
	}
	pokeStr := ""
	if len(pokes) > 0 {
		pokeStr = strings.Join(pokes, " ")
	}
	return resStr, pokeStr
}

func (b *Bot) equipSkill(uid string, newSkill content.Skill) (int, bool) {
	// Check for Title-based extra slots
	extraSlots := 0
	var tName sql.NullString
	_ = b.DB.QueryRow("SELECT title FROM users WHERE client_uid=$1", uid).Scan(&tName)
	if tName.Valid {
		if t, ok := content.GetTitleByName(tName.String); ok {
			extraSlots = t.ExtraSkills
		}
	}
	maxSlots := 5 + extraSlots

	// Find slot to replace (empty first, then lowest rarity)
	rows, err := b.DB.Query("SELECT slot, skill_id FROM user_skills WHERE client_uid = $1", uid)
	if err != nil {
		return 0, false
	}
	defer func() { _ = rows.Close() }()

	slots := make(map[int]string)
	for rows.Next() {
		var s int
		var id string
		if err := rows.Scan(&s, &id); err == nil {
			slots[s] = id
		}
	}

	// 1. Empty slot
	for i := 1; i <= maxSlots; i++ {
		if _, ok := slots[i]; !ok {
			_, _ = b.DB.Exec("INSERT INTO user_skills (client_uid, slot, skill_id) VALUES ($1, $2, $3)", uid, i, newSkill.ID)
			return i, true
		}
	}

	// 2. Replace lowest rarity if new one is better
	for i := 1; i <= maxSlots; i++ {
		if curID := slots[i]; curID != "" {
			if cur, ok := content.GetSkillByID(curID); ok {
				if newSkill.Rarity > cur.Rarity {
					_, _ = b.DB.Exec("UPDATE user_skills SET skill_id = $3 WHERE client_uid = $1 AND slot = $2", uid, i, newSkill.ID)
					return i, true
				}
			}
		}
	}

	return 0, false
}

func (b *Bot) getSkills(uid string) []content.Skill {
	rows, err := b.DB.Query("SELECT skill_id FROM user_skills WHERE client_uid = $1", uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []content.Skill
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			if s, ok := content.GetSkillByID(id); ok {
				out = append(out, s)
			}
		}
	}

	// Append active skills from the Abyss Skill Tree (passive web)
	if alloc, err := b.loadTreeAllocated(uid); err == nil {
		hasEQ := false
		hasAS := false
		for _, nid := range alloc {
			switch nid {
			case content.NodeSkillEarthquake:
				hasEQ = true
			case content.NodeSkillArcaneShield:
				hasAS = true
			}
		}
		if hasEQ {
			if s, ok := content.GetSkillByID("S_EQ"); ok {
				out = append(out, s)
			}
		}
		if hasAS {
			if s, ok := content.GetSkillByID("S_AS"); ok {
				out = append(out, s)
			}
		}
	}

	return out
}

// maxActiveUltimates caps how many distinct ultimates a player can run at once.
const maxActiveUltimates = 3

// getActiveUltimates returns the player's active ultimates (up to
// maxActiveUltimates, legacy/oldest first) with their live cooldowns.
func (b *Bot) getActiveUltimates(uid string) []*content.UltimateSkill {
	rows, err := b.DB.Query(
		"SELECT ultimate_id, current_cooldown FROM user_ultimate_skills WHERE client_uid=$1 AND active ORDER BY obtained, ultimate_id LIMIT $2",
		uid, maxActiveUltimates)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []*content.UltimateSkill
	for rows.Next() {
		var id string
		var cooldown int
		if err := rows.Scan(&id, &cooldown); err == nil {
			if us, ok := content.GetUltimateSkillByID(id); ok {
				us.CurrentCooldown = cooldown
				out = append(out, &us)
			}
		}
	}
	return out
}

// activateUltimateIfSlotFree flips a freshly-obtained ultimate to active when
// the player runs fewer than maxActiveUltimates. Returns true if activated.
func (b *Bot) activateUltimateIfSlotFree(uid, ultID string) bool {
	res, err := b.DB.Exec(
		`UPDATE user_ultimate_skills SET active = TRUE, current_cooldown = 0
		 WHERE client_uid=$1 AND ultimate_id=$2 AND NOT active
		 AND (SELECT COUNT(*) FROM user_ultimate_skills WHERE client_uid=$1 AND active) < $3`,
		uid, ultID, maxActiveUltimates)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

func (b *Bot) applyEnchantment(uid string, ench content.Enchantment) (string, bool) {
	rows, err := b.DB.Query("SELECT slot, enchantment_id FROM user_gear WHERE client_uid = $1", uid)
	if err != nil {
		return "", false
	}
	defer func() { _ = rows.Close() }()
	type slotInfo struct{ slot, enchID string }
	var slots []slotInfo
	for rows.Next() {
		var s slotInfo
		var e sql.NullString
		if err := rows.Scan(&s.slot, &e); err == nil {
			if e.Valid {
				s.enchID = e.String
			}
			slots = append(slots, s)
		}
	}
	if len(slots) == 0 {
		return "", false
	}
	// #nosec G404
	target := slots[rand.IntN(len(slots))] // #nosec G404

	// Improvement 39: Unstable Enchantments
	// #nosec G404
	if rand.Float64() < 0.05 {
		// 5% chance to break item
		_, _ = b.DB.Exec("DELETE FROM user_gear WHERE client_uid = $1 AND slot = $2", uid, target.slot)
		return target.slot, false
	}

	// 95% chance for success + double stats boost
	ench.Stats.STR *= 2
	ench.Stats.SPD *= 2

	if target.enchID != "" {
		if cur, ok := content.GetEnchantmentByID(target.enchID); ok {
			if ench.Rarity < cur.Rarity {
				return "", false
			}
		}
	}
	_, _ = b.DB.Exec("UPDATE user_gear SET enchantment_id = $3, durability = durability + $4 WHERE client_uid = $1 AND slot = $2", uid, target.slot, ench.ID, ench.DuraBonus)
	return target.slot, true
}

func (b *Bot) shouldEquip(uid string, newGear content.Gear) bool {
	var currentID string
	var itemData sql.NullString
	err := b.DB.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE client_uid=$1 AND slot=$2", uid, string(newGear.Slot)).Scan(&currentID, &itemData)
	if err == sql.ErrNoRows {
		return true
	}
	var cur content.Gear
	hasGear := false
	if itemData.Valid && itemData.String != "" {
		if err := json.Unmarshal([]byte(itemData.String), &cur); err == nil {
			hasGear = true
		}
	}
	if !hasGear {
		if c, ok := content.GetGearByID(currentID); ok {
			cur = c
			hasGear = true
		}
	}
	if hasGear {
		// Prioritize XP Multiplier first for faster progression
		if newGear.XPMultiplier > cur.XPMultiplier {
			return true
		}
		// Equip if higher rarity OR if CombatRating is better (replaces stale gear with fresh durability)
		return newGear.Rarity > cur.Rarity || newGear.CombatRating() > cur.CombatRating()
	}
	return true
}

func (b *Bot) awardXP(uid, nickname string, awarded int) (*levelResult, error) {
	var curXP, curLevel int
	err := b.DB.QueryRow("SELECT xp, level FROM users WHERE client_uid = $1", uid).Scan(&curXP, &curLevel)
	if err == sql.ErrNoRows {
		curXP, curLevel = 0, 1
	} else if err != nil {
		return nil, err
	}
	total := curXP + awarded
	if total < 0 {
		total = 0
	}
	newLevel := leveling.LevelForXP(total)

	if nickname != "" {
		_, err = b.DB.Exec(`INSERT INTO users (client_uid, nickname, xp, level, last_seen) VALUES ($1, $2, $3, $4, NOW()) ON CONFLICT (client_uid) DO UPDATE SET xp = $3, level = $4, nickname = $2, last_seen = NOW()`, uid, nickname, total, newLevel)
	} else {
		_, err = b.DB.Exec(`UPDATE users SET xp = $2, level = $3, last_seen = NOW() WHERE client_uid = $1`, uid, total, newLevel)
	}
	return &levelResult{OldLevel: curLevel, NewLevel: newLevel, TotalXP: total, Awarded: awarded}, err
}

func (b *Bot) slothDecay(_ *clientquery.Client, today time.Time) {
	cutoff := today.AddDate(0, 0, -slothGraceDays)
	rows, err := b.DB.Query(`SELECT client_uid, nickname, xp, level, last_seen FROM users WHERE last_seen < $1`, cutoff)
	if err != nil {
		return
	}
	type decayRow struct {
		uid, nick string
		xp, level int
	}
	var batch []decayRow
	for rows.Next() {
		var d decayRow
		if err := rows.Scan(&d.uid, &d.nick, &d.xp, &d.level); err == nil {
			batch = append(batch, d)
		}
	}
	_ = rows.Close()
	for _, d := range batch {
		newXP := int(float64(d.xp) * (1.0 - slothDailyDecay))
		if newXP < 0 {
			newXP = 0
		}
		newLevel := leveling.LevelForXP(newXP)
		_, _ = b.DB.Exec("UPDATE users SET xp=$2, level=$3 WHERE client_uid=$1", d.uid, newXP, newLevel)
	}
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
