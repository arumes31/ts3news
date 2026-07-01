package bot

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"ts3news/internal/clientquery"
	"ts3news/internal/icons"
	"ts3news/internal/leveling"
)

func (b *Bot) applyMilestones(c *clientquery.Client, clid int, nickname string, lr *levelResult) {
	if lr == nil { return }
	crossed := leveling.MilestonesCrossed(lr.OldLevel, lr.NewLevel, b.levelGroups)
	if len(crossed) == 0 { return }

	cldbid, err := c.ClientDBID(clid)
	if err != nil { return }

	for _, sgid := range crossed {
		if err := c.AddServerGroup(sgid, cldbid); err != nil {
			log.Printf("milestone: failed to add group %d to %s: %v", sgid, nickname, err)
		} else {
			log.Printf("milestone: %s reached level and earned group %d", nickname, sgid)
		}
	}
}

// iconSizePx is the generated icon resolution. TeamSpeak group icons are 16x16.
const iconSizePx = 16

// maxGroupNameLen is TeamSpeak's server-group name limit (longer names fail with
// error 1541 "invalid parameter size").
const maxGroupNameLen = 30

// xpGroupName is the server-group name for a level. For named tiers it uses the
// flavour name ("Peasant I"); for the long procedural "infinite tier" names that
// would exceed the TS3 limit it falls back to a compact, unique "Lvl N" form.
func xpGroupName(level int) string {
	n := leveling.LevelName(level)
	if len([]rune(n)) <= maxGroupNameLen {
		return n
	}
	return fmt.Sprintf("Lvl %d", level)
}

// levelFromGroupName maps a server-group name back to its level (the inverse of
// xpGroupName), recognising both the flavour names and the compact "Lvl N" form.
func levelFromGroupName(name string) (int, bool) {
	if l, ok := leveling.LevelByName(name); ok {
		return l, true
	}
	if rest, ok := strings.CutPrefix(name, "Lvl "); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(rest)); err == nil {
			return n, true
		}
	}
	return 0, false
}

// loadLevelGroups populates the in-memory level->sgid cache from the database.
func (b *Bot) loadLevelGroups() {
	rows, err := b.DB.Query("SELECT level, sgid FROM level_groups")
	if err != nil {
		log.Printf("xpgroups: loading cache failed: %v", err)
		return
	}
	defer func() { _ = rows.Close() }()
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

func (b *Bot) getOrCreateLevelGroup(c *clientquery.Client, level int) (int, error) {
	name := xpGroupName(level)

	groups, err := c.ServerGroupList()
	if err != nil {
		log.Printf("xpgroups: servergrouplist failed: %v", err)
	} else {
		log.Printf("xpgroups: %d server groups visible", len(groups))
		if sg, ok := findGroupAndIconByName(groups, name); ok {
			// If group exists but icon is missing, finish it (re-upload icon)
			if sg.IconID == 0 {
				return b.finishGroup(c, level, sg.ID)
			}
			b.xpGroups[level] = sg.ID
			return sg.ID, nil
		}
	}

	if sgid, ok := b.xpGroups[level]; ok {
		return sgid, nil
	}

	// Create it; the sgid is not echoed, so re-list and look it up by name.
	if addErr := c.ServerGroupAdd(name); addErr != nil {
		log.Printf("xpgroups: servergroupadd %q: %v", name, addErr)
	}

	groups2, err := c.ServerGroupList()
	if err != nil {
		return 0, fmt.Errorf("re-list after creating %q failed: %w", name, err)
	}
	if sg, ok := findGroupAndIconByName(groups2, name); ok {
		return b.finishGroup(c, level, sg.ID)
	}
	return 0, fmt.Errorf("group %q was not created or found after add", name)
}

func findGroupAndIconByName(groups []clientquery.ServerGroup, name string) (clientquery.ServerGroup, bool) {
	for _, g := range groups {
		if g.Name == name {
			return g, true
		}
	}
	return clientquery.ServerGroup{}, false
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
			lvl, ok := levelFromGroupName(g.Name)
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

// applyAbyssMilestones checks depth milestones (10, 25, 50, 100) and automatically
// grants the corresponding TS3 server groups configured for achievements.
func (b *Bot) applyAbyssMilestones(c *clientquery.Client, clid int, uid, nickname string, depth int) {
	if !b.Cfg.XPServerGroups {
		return
	}
	cldbid, err := c.ClientDBID(clid)
	if err != nil {
		return
	}
	milestones := []struct {
		Floor int
		Name  string
	}{
		{10, "Threshold Breaker"},
		{25, "Deep Diver"},
		{50, "Abyssal Veteran"},
		{100, "Voidwalker"},
	}
	for _, m := range milestones {
		if depth < m.Floor {
			continue
		}
		sgid, err := b.getOrCreateTitleGroup(c, m.Name)
		if err != nil {
			log.Printf("abyss milestone: failed to create title group %q for %s: %v", m.Name, nickname, err)
			continue
		}
		if err := c.AddServerGroup(sgid, cldbid); err != nil {
			// A permanent milestone group is re-granted every cycle; once the client
			// already holds it, TeamSpeak returns "duplicate entry" (id 2561). That's
			// the expected steady state, not a failure — skip it quietly.
			var cerr *clientquery.CommandError
			if errors.As(err, &cerr) && cerr.ID == clientquery.ErrDuplicateEntry {
				continue
			}
			log.Printf("abyss milestone: granting %q (sgid %d) to %s failed: %v", m.Name, sgid, nickname, err)
		} else {
			log.Printf("abyss milestone: %s reached floor %d, granted %q", nickname, m.Floor, m.Name)
		}
	}
}

