package bot

import (
	"database/sql"
	"fmt"
	"strings"

	"ts3news/internal/clientquery"
	"ts3news/internal/content"
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

		prefix := fmt.Sprintf("(gs:%d) %s%s", score, effCode, typeCode)
		avail := 30 - len(prefix)
		if avail <= 0 {
			return prefix[:30]
		}
		if len(name) > avail {
			name = name[:avail]
		}
		return prefix + name
	}

	// Gear
	grows, err := b.DB.Query("SELECT gear_id, slot FROM user_gear WHERE client_uid = $1", uid)
	if err == nil {
		defer func() { _ = grows.Close() }()
		for grows.Next() {
			var id string
			var slot string
			if err := grows.Scan(&id, &slot); err == nil {
				if g, ok := content.GetGearByID(id); ok {
					activeItemNames[formatGSName(g.Stats.Score(), g.Name, g.Special, "s:"+slot)] = true
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

	// Pets
	prows, err := b.DB.Query("SELECT name, mob_type, level, hp, str, def, spd FROM user_pets WHERE client_uid = $1", uid)
	if err == nil {
		defer func() { _ = prows.Close() }()
		for prows.Next() {
			var m content.Mob
			var mType string
			if err := prows.Scan(&m.Name, &mType, &m.Level, &m.Stats.HP, &m.Stats.STR, &m.Stats.DEF, &m.Stats.SPD); err == nil {
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
		isLegacy := strings.Contains(g.Name, "[slot:")
		if (isRPGRelated || isLegacy) && !activeItemNames[g.Name] {
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
