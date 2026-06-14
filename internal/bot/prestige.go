package bot

import (
	"log"
	"strconv"
	"strings"

	"ts3news/internal/clientquery"
	"ts3news/internal/i18n"
	"ts3news/internal/icons"
	"ts3news/internal/leveling"
	)

	const prestigeStatBonus = 0.15 // +15% permanent stat boost per prestige level
	const PrestigeThreshold = 9999 // Level required to prestige (was 10000)

	// doPrestige increments a user's prestige and resets their level/xp to the start,
	// returning the new prestige number.
	func (b *Bot) doPrestige(uid string) int {
	var p int
	_ = b.DB.QueryRow(
		"UPDATE users SET prestige = prestige + 1, xp = 0, level = 1 WHERE client_uid = $1 RETURNING prestige",
		uid).Scan(&p)
	return p
	}

	func prestigeGroupName(p int) string { return i18n.T("prestige.group_name", p) }

func prestigeFromGroupName(name string) (int, bool) {
	if rest, ok := strings.CutPrefix(name, "Prestige "); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(rest)); err == nil {
			return n, true
		}
	}
	return 0, false
}

// applyPrestigeGroup grants the user the server group for their current prestige
// (creating it with a generated prestige icon if needed) and removes any lower
// prestige groups, deleting them when empty.
func (b *Bot) applyPrestigeGroup(c *clientquery.Client, clid int, uid, nickname string, prestige int) {
	if prestige < 1 {
		return
	}
	cldbid, err := c.ClientDBID(clid)
	if err != nil {
		log.Printf("prestige: cannot resolve cldbid for %s: %v", nickname, err)
		return
	}

	name := prestigeGroupName(prestige)
	groups, err := c.ServerGroupList()
	if err != nil {
		log.Printf("prestige: servergrouplist failed: %v", err)
		return
	}

	var sgid int
	found := false
	for _, g := range groups {
		if g.Name == name {
			sgid = g.ID
			found = true
			if g.IconID == 0 {
				// Re-upload icon if missing
				if png, ierr := icons.Icon(prestige, leveling.NumTiers, leveling.NumTiers, iconSizePx); ierr == nil {
					if id, uerr := c.UploadIcon(png, b.Cfg.TS3Host); uerr == nil {
						_ = c.ServerGroupAddPerm(sgid, "i_icon_id", int(id))
					}
				}
			}
			break
		}
	}

	if !found {
		if err := c.ServerGroupAdd(name); err != nil {
			log.Printf("prestige: servergroupadd %q: %v", name, err)
		}
		sgid, ok := b.findServerGroupByName(c, name)
		if !ok {
			log.Printf("prestige: group %q was not created", name)
			return
		}
		// Prestige icon: the prestige number on the top-tier (prestige) colour.
		if png, ierr := icons.Icon(prestige, leveling.NumTiers, leveling.NumTiers, iconSizePx); ierr == nil {
			if id, uerr := c.UploadIcon(png, b.Cfg.TS3Host); uerr == nil {
				if perr := c.ServerGroupAddPerm(sgid, "i_icon_id", int(id)); perr == nil {
					log.Printf("prestige: icon %d set for %q (sgid %d)", id, name, sgid)
				}
			}
		}
	}

	if err := c.AddServerGroup(sgid, cldbid); err != nil {
		log.Printf("prestige: assigning %s to %q failed: %v", nickname, name, err)
		return
	}
	log.Printf("prestige: %s reached %q", nickname, name)

	// Remove lower prestige groups from this user; delete them if left empty.
	if groups, lerr := c.ServerGroupList(); lerr == nil {
		for _, g := range groups {
			if pv, isP := prestigeFromGroupName(g.Name); isP && pv < prestige {
				_ = c.ServerGroupDelClient(g.ID, cldbid)
				if n, e := c.ServerGroupMemberCount(g.ID); e == nil && n == 0 {
					_ = c.ServerGroupDel(g.ID, true)
				}
			}
		}
	}
}

func (b *Bot) findServerGroupByName(c *clientquery.Client, name string) (int, bool) {
	groups, err := c.ServerGroupList()
	if err != nil {
		return 0, false
	}
	if sg, ok := findGroupAndIconByName(groups, name); ok {
		return sg.ID, true
	}
	return 0, false
}
