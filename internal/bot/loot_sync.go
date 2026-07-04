package bot

import (
	"database/sql"
	"fmt"
	"strings"

	"ts3news/internal/clientquery"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
	)

	// syncLootGroups ensures the user is in the correct TS3 server groups for their
	// currently equipped gear, artifacts, skills, and pets.
	func (b *Bot) syncLootGroups(c *clientquery.Client, clid int, uid string) {
	cldbid, err := c.ClientDBID(clid)
	if err != nil {
		return
	}

	// 1. Get all active items and pets for this user
	activeItemNames := map[string]bool{}

	// Helper to format group names with 30-char limit: "(gs:XXXX)[E] [type:X] Name..."
	formatGSName := func(score int, name string, effect content.ItemEffect, itemType string) string {
		effCode := ""
		if effect != content.EffectNone {
			mapping := map[content.ItemEffect]string{
				content.EffectThorns:         "T",
				content.EffectVampiric:       "V",
				content.EffectBerserk:        "B",
				content.EffectLucky:          "L",
				content.EffectTreasureHunter: "H",
				content.EffectQuick:          "Q",
				content.EffectBulwark:        "W",
				content.EffectRadiant:        "R",
				content.EffectFragile:        "F",
				content.EffectSteady:         "S",
				content.EffectMindControl:    "M",
				content.EffectRegenStack:     "G",
			}
			if code, ok := mapping[effect]; ok {
				effCode = "[" + code + "] "
			}
		}

		// Add type information
		typeCode := "[" + itemType + "] "

		groupName := i18n.T("loot_sync.group_name", score, effCode, typeCode)
		avail := 30 - len(groupName)
		if avail <= 0 {
			return groupName[:30]
		}
		if len(name) > avail {
			name = name[:avail]
		}
		return groupName + name
	}

	// Gear. makeGear overlays the per-item item_data JSON (Abyss upgrades, gems,
	// runes, tier ascensions) so groups reflect the piece as worn, not the base
	// catalog entry.
	grows, err := b.DB.Query("SELECT gear_id, slot, item_data FROM user_gear WHERE client_uid = $1", uid)
	if err == nil {
		defer func() { _ = grows.Close() }()
		for grows.Next() {
			var id string
			var slot string
			var itemData sql.NullString
			if err := grows.Scan(&id, &slot, &itemData); err == nil {
				if g, ok := b.makeGear(id, itemData); ok {
					name := g.Name
					if g.GearLevel > 0 {
						name = fmt.Sprintf("%s +%d", name, g.GearLevel)
					}
					activeItemNames[formatGSName(g.Stats.Score(), name, g.Special, "slot:"+slot)] = true
				}
			}
		}
	}

	// Artifact
	var aName sql.NullString
	if err := b.DB.QueryRow("SELECT artifact_name FROM users WHERE client_uid = $1", uid).Scan(&aName); err == nil && aName.Valid && aName.String != "" {
		if art, ok := content.GetArtifactByName(aName.String); ok {
			activeItemNames[formatGSName(art.Score(), art.Name, art.Special, "artifact")] = true
		}
	}

	// Skills
	srows, err := b.DB.Query("SELECT slot, skill_id FROM user_skills WHERE client_uid = $1", uid)
	if err == nil {
		defer func() { _ = srows.Close() }()
		for srows.Next() {
			var slot int
			var id string
			if err := srows.Scan(&slot, &id); err == nil {
				if s, ok := content.GetSkillByID(id); ok {
					activeItemNames[formatGSName(s.Score(), s.Name, s.Special, fmt.Sprintf("skill:%d", slot))] = true
				}
			}
		}
	}

	// Ultimate Skills
	var ultimateID sql.NullString
	if err := b.DB.QueryRow("SELECT ultimate_skill_id FROM users WHERE client_uid = $1", uid).Scan(&ultimateID); err == nil && ultimateID.Valid {
		if us, ok := content.GetUltimateSkillByID(ultimateID.String); ok {
			// Ultimate skills don't have a Score() method like gear, so we use 0 for the score
			activeItemNames[formatGSName(0, us.Name, content.EffectNone, "ultimate")] = true
		}
	}

	// Pets (captured via the Mind Control effect); max_hp feeds Mob.Score so the
	// group gs matches what the armoury shows.
	prows, err := b.DB.Query("SELECT name, mob_type, level, hp, max_hp, str, def, spd FROM user_pets WHERE client_uid = $1", uid)
	if err == nil {
		defer func() { _ = prows.Close() }()
		for prows.Next() {
			var m content.Mob
			var mType string
			if err := prows.Scan(&m.Name, &mType, &m.Level, &m.Stats.HP, &m.MaxHP, &m.Stats.STR, &m.Stats.DEF, &m.Stats.SPD); err == nil {
				m.Type = content.MobType(mType)
				activeItemNames[formatGSName(m.Score(), "Pet "+m.Name, content.EffectNone, "pet")] = true
			}
		}
	}

	// 2. Ensure user is in these groups
	for name := range activeItemNames {
		sgid, err := b.getOrCreateItemGroup(c, name)
		if err == nil {
			_ = c.AddServerGroup(sgid, cldbid)
		}
	}

	// 3. Remove from groups that are no longer active
	groups, err := c.ServerGroupList()
	if err != nil {
		return
	}

	for _, g := range groups {
		isRPGRelated := strings.Contains(g.Name, "(gs:")
		if isRPGRelated && !activeItemNames[g.Name] {
			_ = c.ServerGroupDelClient(g.ID, cldbid)
			b.maybeDeleteEmptyTitleGroup(c, g.ID, g.Name)
		}
	}
}

func (b *Bot) getOrCreateItemGroup(c *clientquery.Client, name string) (int, error) {
	groups, err := c.ServerGroupList()
	if err == nil {
		for _, g := range groups {
			if g.Name == name {
				return g.ID, nil
			}
		}
	}

	if err := c.ServerGroupAdd(name); err != nil {
		return 0, err
	}

	groups, _ = c.ServerGroupList()
	for _, g := range groups {
		if g.Name == name {
			return g.ID, nil
		}
	}
	return 0, fmt.Errorf("failed to find created group")
}
