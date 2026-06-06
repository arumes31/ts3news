package bot

import (
	"database/sql"
	"fmt"
	"strings"

	"ts3news/internal/clientquery"
	"ts3news/internal/content"
)

// syncLootGroups ensures the user is in the correct TS3 server groups for their
// currently equipped gear and artifacts.
func (b *Bot) syncLootGroups(c *clientquery.Client, clid int, uid string) {
	cldbid, err := c.ClientDBID(clid)
	if err != nil {
		return
	}

	// 1. Get all active items for this user
	activeItemNames := map[string]bool{}

	// Helper to format group names with 30-char limit: "(gs:XXXX) Name..."
	formatGSName := func(score int, name string) string {
		prefix := fmt.Sprintf("(gs:%d) ", score)
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
	grows, err := b.DB.Query("SELECT gear_id FROM user_gear WHERE client_uid = $1", uid)
	if err == nil {
		defer func() { _ = grows.Close() }()
		for grows.Next() {
			var id string
			if err := grows.Scan(&id); err == nil {
				if g, ok := content.GetGearByID(id); ok {
					activeItemNames[formatGSName(g.Stats.Score(), g.Name)] = true
				}
			}
		}
	}

	// Artifact
	var aName sql.NullString
	if err := b.DB.QueryRow("SELECT artifact_name FROM users WHERE client_uid = $1", uid).Scan(&aName); err == nil && aName.Valid && aName.String != "" {
		if art, ok := content.GetArtifactByName(aName.String); ok {
			activeItemNames[formatGSName(art.Score(), art.Name)] = true
		}
	}

	// Skills
	srows, err := b.DB.Query("SELECT skill_id FROM user_skills WHERE client_uid = $1", uid)
	if err == nil {
		defer func() { _ = srows.Close() }()
		for srows.Next() {
			var id string
			if err := srows.Scan(&id); err == nil {
				if s, ok := content.GetSkillByID(id); ok {
					activeItemNames[formatGSName(s.Score(), s.Name)] = true
				}
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

	// 3. Remove from groups that are no longer active loot or skills
	groups, err := c.ServerGroupList()
	if err != nil {
		return
	}

	for _, g := range groups {
		isRPGRelated := strings.Contains(g.Name, "(gs:")
		if isRPGRelated && !activeItemNames[g.Name] {
			_ = c.ServerGroupDelClient(g.ID, cldbid)
			b.maybeDeleteEmptyTitleGroup(c, g.ID, g.Name) // reuse helper
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

	// Create group
	if err := c.ServerGroupAdd(name); err != nil {
		return 0, err
	}

	// Re-list to find ID
	groups, _ = c.ServerGroupList()
	for _, g := range groups {
		if g.Name == name {
			return g.ID, nil
		}
	}
	return 0, fmt.Errorf("failed to find created item group")
}
