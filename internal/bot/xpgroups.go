package bot

import (
	"fmt"
	"log"

	"ts3news/internal/clientquery"
	"ts3news/internal/icons"
	"ts3news/internal/leveling"
)

// iconSizePx is the generated icon resolution. TeamSpeak group icons are 16x16.
const iconSizePx = 16

// xpGroupName is the server-group name for a level, e.g. "Peasant I".
func xpGroupName(level int) string {
	return leveling.LevelName(level)
}

// loadLevelGroups populates the in-memory level->sgid cache from the database.
func (b *Bot) loadLevelGroups() {
	rows, err := b.DB.Query("SELECT level, sgid FROM level_groups")
	if err != nil {
		log.Printf("xpgroups: loading cache failed: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var level, sgid int
		if err := rows.Scan(&level, &sgid); err == nil {
			b.xpGroups[level] = sgid
		}
	}
}

func (b *Bot) dbSaveLevelGroup(level, sgid int, iconID int64) error {
	_, err := b.DB.Exec(
		`INSERT INTO level_groups (level, sgid, icon_id) VALUES ($1, $2, $3)
		 ON CONFLICT (level) DO UPDATE SET sgid = $2, icon_id = $3`,
		level, sgid, iconID)
	return err
}

func (b *Bot) dbDeleteLevelGroup(level int) error {
	_, err := b.DB.Exec("DELETE FROM level_groups WHERE level = $1", level)
	return err
}

func (b *Bot) getUserGroupLevel(uid string) int {
	var level int
	if err := b.DB.QueryRow("SELECT group_level FROM users WHERE client_uid = $1", uid).Scan(&level); err != nil {
		return 0
	}
	return level
}

func (b *Bot) setUserGroupLevel(uid string, level int) error {
	_, err := b.DB.Exec("UPDATE users SET group_level = $2 WHERE client_uid = $1", uid, level)
	return err
}

func findGroupByName(groups []clientquery.ServerGroup, name string) (int, bool) {
	for _, g := range groups {
		if g.Name == name {
			return g.ID, true
		}
	}
	return 0, false
}

// getOrCreateLevelGroup returns the server group id for a level, creating the
// group (and uploading its generated icon) on first use. Because the ClientQuery
// servergroupadd reply does not always echo the new sgid, we re-list the groups
// and look the new group up by name.
func (b *Bot) getOrCreateLevelGroup(c *clientquery.Client, level int) (int, error) {
	if sgid, ok := b.xpGroups[level]; ok {
		return sgid, nil
	}
	name := xpGroupName(level)

	groups, err := c.ServerGroupList()
	if err != nil {
		log.Printf("xpgroups: servergrouplist failed: %v", err)
	} else {
		log.Printf("xpgroups: %d server groups visible", len(groups))
		if sgid, ok := findGroupByName(groups, name); ok {
			return b.finishGroup(c, level, sgid)
		}
	}

	// Create it; the sgid is not echoed, so re-list and look it up by name.
	if addErr := c.ServerGroupAdd(name); addErr != nil {
		log.Printf("xpgroups: servergroupadd %q: %v", name, addErr)
	}

	groups2, err := c.ServerGroupList()
	if err != nil {
		return 0, fmt.Errorf("re-list after creating %q failed: %w", name, err)
	}
	if sgid, ok := findGroupByName(groups2, name); ok {
		return b.finishGroup(c, level, sgid)
	}
	return 0, fmt.Errorf("group %q was not created or found after add", name)
}

// finishGroup caches a group and ensures it has its generated icon (idempotent:
// re-uploading the same icon and re-setting the permission are no-ops).
func (b *Bot) finishGroup(c *clientquery.Client, level, sgid int) (int, error) {
	b.xpGroups[level] = sgid

	// At 16x16 a 3-digit absolute level does not fit, so the icon shows the
	// sub-rank within the tier (1..40, matching the roman numeral in the name),
	// coloured/prestiged by tier.
	tier := leveling.TierForLevel(level)
	var iconID int64
	if png, ierr := icons.Icon(leveling.SubRank(level), tier, leveling.NumTiers, iconSizePx); ierr != nil {
		log.Printf("xpgroups: icon generation for level %d failed: %v", level, ierr)
	} else if id, uerr := c.UploadIcon(png, b.Cfg.TS3Host); uerr != nil {
		log.Printf("xpgroups: icon upload for level %d failed (group has no icon): %v", level, uerr)
	} else {
		iconID = int64(id)
		if perr := c.ServerGroupAddPerm(sgid, "i_icon_id", int(id)); perr != nil {
			log.Printf("xpgroups: setting icon perm for level %d failed: %v", level, perr)
		} else {
			log.Printf("xpgroups: icon %d set for group %q (sgid %d)", id, xpGroupName(level), sgid)
		}
	}
	if err := b.dbSaveLevelGroup(level, sgid, iconID); err != nil {
		log.Printf("xpgroups: caching level %d failed: %v", level, err)
	}
	return sgid, nil
}

// applyLevelGroup moves a user into the server group for their current level,
// removing them from their previous level group and deleting that group if empty.
func (b *Bot) applyLevelGroup(c *clientquery.Client, clid int, uid, nickname string, newLevel int) {
	oldLevel := b.getUserGroupLevel(uid)
	if newLevel == oldLevel {
		return
	}

	cldbid, err := c.ClientDBID(clid)
	if err != nil {
		log.Printf("xpgroups: cannot resolve cldbid for %s: %v", nickname, err)
		return
	}

	newSgid, err := b.getOrCreateLevelGroup(c, newLevel)
	if err != nil {
		log.Printf("xpgroups: %v", err)
		return
	}
	if err := c.AddServerGroup(newSgid, cldbid); err != nil {
		log.Printf("xpgroups: assigning %s to %q failed (needs permission): %v", nickname, xpGroupName(newLevel), err)
		return
	}
	log.Printf("xpgroups: assigned %s -> %q (level %d)", nickname, xpGroupName(newLevel), newLevel)

	// Remove the user from every OTHER level group they may still be in (robust
	// against stale state), then delete any that become empty.
	if groups, lerr := c.ServerGroupList(); lerr == nil {
		for _, g := range groups {
			lvl, ok := leveling.LevelByName(g.Name)
			if !ok || g.ID == newSgid {
				continue
			}
			_ = c.ServerGroupDelClient(g.ID, cldbid) // ignore "not a member" errors
			b.maybeDeleteEmptyLevel(c, lvl, g.ID)
		}
	}
	if err := b.setUserGroupLevel(uid, newLevel); err != nil {
		log.Printf("xpgroups: persisting level for %s failed: %v", nickname, err)
	}
}

// maybeDeleteEmptyLevel deletes a level group (and its cache entry) when it has no
// remaining members.
func (b *Bot) maybeDeleteEmptyLevel(c *clientquery.Client, level, sgid int) {
	n, err := c.ServerGroupMemberCount(sgid)
	if err != nil {
		log.Printf("xpgroups: member count for level %d failed: %v", level, err)
		return
	}
	if n > 0 {
		return
	}
	if err := c.ServerGroupDel(sgid, true); err != nil {
		log.Printf("xpgroups: deleting empty level %d failed: %v", level, err)
		return
	}
	delete(b.xpGroups, level)
	_ = b.dbDeleteLevelGroup(level)
	log.Printf("xpgroups: removed empty group %q (sgid %d)", xpGroupName(level), sgid)
}

// cleanupEmptyLevelGroups deletes any empty server group whose name is a level
// name — including orphans not in our cache (e.g. left over from a failed run).
func (b *Bot) cleanupEmptyLevelGroups(c *clientquery.Client) {
	groups, err := c.ServerGroupList()
	if err != nil {
		log.Printf("xpgroups: cleanup list failed: %v", err)
		return
	}
	for _, g := range groups {
		level, ok := leveling.LevelByName(g.Name)
		if !ok {
			continue // not an XP level group; leave it alone
		}
		b.maybeDeleteEmptyLevel(c, level, g.ID)
	}
}
