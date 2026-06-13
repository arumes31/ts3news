package bot

import (
	"log"

	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

// lootFate describes what should happen to a dropped gear piece.
type lootFate int

const (
	fateEquip   lootFate = iota // upgrade → equip it
	fateListAH                  // rare+ but not an upgrade → list on the auction house
	fateSalvage                 // sub-rare and not an upgrade → salvage into scrap
)

// decideGearFate is the rarity gate the loot system is built around.
//
// Given a freshly dropped gear piece and whatever is currently equipped in that
// slot (nil = empty slot), it decides the outcome:
//   - an upgrade (or a drop for an empty slot) is always equipped;
//   - a rare-or-better drop that is NOT an upgrade is "actioned" by listing it on
//     the auction house, never silently destroyed;
//   - only a sub-rare, non-upgrade drop is salvaged into scrap.
//
// This is the fix for rare+ drops being turned into "Looted Scrap": the rarity
// roll now gates whether an item is actioned, instead of being a flavour tag on
// raw scrap.
func decideGearFate(drop content.Gear, current *content.Gear) lootFate {
	if current == nil || drop.CombatRating() > current.CombatRating() {
		return fateEquip
	}
	if drop.Rarity >= content.RarityRare {
		return fateListAH
	}
	return fateSalvage
}

// rollLootForUser resolves the gear dropped by a single defeated mob for one
// player. It persists the outcome (equip / auction-house listing) and returns
// display notes already wrapped with the looter and mob name via
// bot.combat.looted, ready to be appended to the combat log.
func (b *Bot) rollLootForUser(uid, nickname string, mob *content.Mob) []string {
	if mob == nil || len(mob.Equipped) == 0 {
		return nil
	}

	mobLabel := mob.DisplayNameShort()
	notes := make([]string, 0, len(mob.Equipped))

	for _, drop := range mob.Equipped {
		// Re-read the slot each iteration so a second drop for the same slot
		// compares against an item we may have just equipped above.
		current := b.equippedInSlot(uid, drop.Slot)

		var detail string
		switch decideGearFate(drop, current) {
		case fateEquip:
			b.equipGear(uid, drop)
			detail = i18n.T("bot.loot.equipped", drop.Name, string(drop.Slot),
				drop.Stats.Score(), drop.CombatRating(), drop.Rarity.Color(), drop.Rarity.String())
		case fateListAH:
			b.autoListUnwantedItems(uid, drop)
			detail = i18n.T("bot.loot.listed_ah", drop.Name, string(drop.Slot),
				drop.Rarity.Color(), drop.Rarity.String())
		default: // fateSalvage
			detail = i18n.T("bot.loot.salvaged", drop.Name, string(drop.Slot), scrapValue(drop))
		}

		notes = append(notes, i18n.T("bot.combat.looted", nickname, mobLabel, detail))
	}

	return notes
}

// equippedInSlot returns the gear currently equipped in the given slot, or nil
// if the slot is empty (or the stored gear id is unknown).
func (b *Bot) equippedInSlot(uid string, slot content.GearSlot) *content.Gear {
	var id string
	if err := b.DB.QueryRow(
		"SELECT gear_id FROM user_gear WHERE client_uid=$1 AND slot=$2", uid, string(slot),
	).Scan(&id); err != nil {
		return nil
	}
	if g, ok := content.GetGearByID(id); ok {
		return &g
	}
	return nil
}

// equipGear persists a gear piece into the player's equipment, replacing
// whatever occupied the slot.
func (b *Bot) equipGear(uid string, g content.Gear) {
	if _, err := b.DB.Exec(
		`INSERT INTO user_gear (client_uid, slot, gear_id, durability)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (client_uid, slot) DO UPDATE SET gear_id = $3, durability = $4`,
		uid, string(g.Slot), g.ID, g.MaxDurability,
	); err != nil {
		log.Printf("Failed to equip gear %s for %s: %v", g.ID, uid, err)
	}
}

// scrapValue is the amount of scrap a salvaged (sub-rare, non-upgrade) gear piece
// is worth. Higher rarity and stronger stats yield more scrap.
func scrapValue(g content.Gear) int {
	v := (int(g.Rarity) + 1) * (g.Stats.Score()/10 + 1)
	if v < 1 {
		v = 1
	}
	return v
}
