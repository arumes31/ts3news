package bot

import (
	"database/sql"
	"fmt"
	"strings"

	"ts3news/internal/content"
)

// Abyss combat mechanics (docs/ABYSS_IMPROVEMENTS_300.md, group C / AB-51..75).
// Everything here is gated on the Abyss run markers (EscrowLoot / FloorModifier)
// so regular TS3 channel combat is never affected. State that must survive a
// single fight (toggles, targets) lives in app_meta so no schema change is
// needed; per-fight state lives on activeUser (see xp.go).

// abyssFightTrack accumulates per-fight damage totals that the end-of-fight
// summary reports (AB-57 thorns total, AB-66 counter-attack line).
type abyssFightTrack struct {
	thorns   int // total reflected (thorns) damage dealt to mobs
	counters int // total parry counter-attack damage dealt to mobs
}

func appendAbyssFightBreakdown(logs []string, track *abyssFightTrack) []string {
	if track == nil {
		return logs
	}
	if track.thorns > 0 {
		logs = append(logs, fmt.Sprintf("🌵 Thorns reflected: %d damage", track.thorns))
	}
	if track.counters > 0 {
		logs = append(logs, fmt.Sprintf("🤺 Parry counter-attacks: %d damage", track.counters))
	}
	return logs
}

// abyssCombatant reports whether u is fighting inside an Abyss run. Regular
// channel combat leaves both markers empty, so every mechanic below stays out
// of the non-Abyss path.
func abyssCombatant(u *UserInCombat) bool {
	return u.EscrowLoot || u.FloorModifier != ""
}

// abyssBossAlive reports whether any living boss-tier mob remains (AB-59
// stunbreak, AB-64 hold-mana, AB-70 summon interrupt).
func abyssBossAlive(mobs []*content.Mob) bool {
	for _, m := range mobs {
		if m.Stats.HP > 0 && (m.Type == content.MobBoss || m.Type == content.MobLegendary) {
			return true
		}
	}
	return false
}

// AB-67 Elemental resist: 3+ etched runes of one element grant 10% resist vs
// mobs of that element. Runes are etched on weapons/shields only (see
// handleAbyssEtchRune), so reaching 3 means a fully runed loadout.
func runeWardResist(equipped map[content.GearSlot]content.Gear, mobElement content.Element) bool {
	if mobElement == "" {
		return false
	}
	n := 0
	for _, g := range equipped {
		if g.Rune == string(mobElement) {
			n++
		}
	}
	return n >= 3
}

// AB-58 Pet focus-fire: find the alive mob matching the focus target name
// (case-insensitive substring so "Imp" matches "Summoned Imp"). Returns nil
// when no mob matches, in which case the pet falls back to a random target.
func petFocusTarget(alive []*content.Mob, focus string) *content.Mob {
	if focus == "" {
		return nil
	}
	f := strings.ToLower(focus)
	for _, m := range alive {
		if strings.Contains(strings.ToLower(m.Name), f) {
			return m
		}
	}
	return nil
}

// AB-62 Focus synergy: the auto-selected loot focus adds a matching combat
// micro-bonus (gold focus → +2% crit, etc.).
func abyssFocusMicroBonus(focus string) (critPct int, dmgMult float64, lifesteal int) {
	switch focus {
	case "gold":
		return 2, 1.0, 0
	case "xp":
		return 0, 1.02, 0
	case "loot":
		return 0, 1.0, 2
	case "materials":
		return 0, 1.02, 0
	case "tokens":
		return 2, 1.0, 0
	}
	return 0, 1.0, 0
}

// abyssCombatOption reads a per-user Abyss combat toggle from app_meta
// (key "abyss_<name>:<uid>"). Missing keys return "".
func (b *Bot) abyssCombatOption(uid, name string) string {
	var v string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", "abyss_"+name+":"+uid).Scan(&v)
	return v
}

func (b *Bot) setAbyssCombatOption(uid, name, value string) error {
	key := "abyss_" + name + ":" + uid
	if value == "" {
		_, err := b.DB.Exec("DELETE FROM app_meta WHERE key=$1", key)
		return err
	}
	_, err := b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
	                     ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, key, value)
	return err
}

// AB-64 Hold mana: when the toggle is on, the delver saves casts (and
// ultimates) for boss floors while grinding normal waves.
func (b *Bot) abyssHoldMana(uid string) bool {
	return b.abyssCombatOption(uid, "hold_mana") == "1"
}

// AB-58 Pet focus-fire target: mob name the player's pet should focus.
func (b *Bot) abyssPetFocus(uid string) string {
	return b.abyssCombatOption(uid, "pet_focus")
}

// AB-53 Weapon swap: scan the backpack for a MainHand backup whose element is
// not weak against the boss. The swap itself happens in-memory for the current
// fight only; persisting it (or a manual trigger endpoint) is web-layer work.
func (b *Bot) findBackupWeapon(uid string, bossElement content.Element, currentID string) (content.Gear, bool) {
	rows, err := b.DB.Query("SELECT gear_id, item_data FROM user_inventory WHERE client_uid=$1", uid)
	if err != nil {
		return content.Gear{}, false
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var gearID string
		var itemData sql.NullString
		if err := rows.Scan(&gearID, &itemData); err != nil {
			continue
		}
		g, ok := b.makeGear(gearID, itemData)
		if !ok || g.Slot != content.SlotMainHand || g.ID == currentID {
			continue
		}
		if getElementMult(g.Element, bossElement) >= 1.0 {
			return g, true
		}
	}
	return content.Gear{}, false
}

// AB-69 Kill-chain: a floor cleared in ≤2 rounds grants +5% speed next floor,
// stacking ×3. Stacks ride the existing consumable infrastructure: one
// user_consumables row whose remaining_fights is the stack count, so the
// standard per-fight decrement makes the bonus decay one stack per fight.
func (b *Bot) grantKillChain(uid string) int {
	stacks := 0
	_ = b.DB.QueryRow(
		`INSERT INTO user_consumables (client_uid, cons_id, remaining_fights) VALUES ($1, 'abyss_kill_chain', 1)
		 ON CONFLICT (client_uid, cons_id) DO UPDATE SET remaining_fights = LEAST(user_consumables.remaining_fights + 1, 3)
		 RETURNING remaining_fights`, uid).Scan(&stacks)
	return stacks
}

// killChainNote describes the active kill-chain bonus for stat aggregation
// notes (AB-69).
func killChainNote(stacks int) string {
	return fmt.Sprintf("⚡ Kill-chain ×%d (+%d%% speed)", stacks, stacks*5)
}
