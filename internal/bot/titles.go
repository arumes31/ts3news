package bot

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"ts3news/internal/clientquery"
	"ts3news/internal/content"
)

// applyTitleGroup ensures the user is in the correct TS3 server group for their
// rare title (e.g. "Overlord"), and removes them from any expired or previous ones.
func (b *Bot) applyTitleGroup(c *clientquery.Client, clid int, uid, nickname string) {
	var title sql.NullString
	var expires sql.NullTime
	err := b.DB.QueryRow("SELECT title, title_expires FROM users WHERE client_uid = $1", uid).Scan(&title, &expires)
	
	activeTitle := ""
	if err == nil && title.Valid && expires.Valid && !time.Now().After(expires.Time) {
		activeTitle = title.String
	}

	cldbid, err := c.ClientDBID(clid)
	if err != nil {
		return
	}

	// 1. If user has an active title, ensure they are in its group.
	var activeSgid int
	if activeTitle != "" {
		sgid, err := b.getOrCreateTitleGroup(c, activeTitle)
		if err == nil {
			activeSgid = sgid
			_ = c.AddServerGroup(sgid, cldbid) // ignore if already in
		}
	}

	// 2. Remove from all other groups that look like titles.
	// To avoid querying every single possible title, we list the user's groups
	// and check if they belong to any group that is NOT their active title but
	// matches a known title name pattern. 
	// For simplicity, we'll just check all server groups and see if the user is in them.
	groups, err := c.ServerGroupList()
	if err != nil {
		return
	}

	for _, g := range groups {
		// If it's a title group but not the ACTIVE one, remove the user.
		if b.isTitleGroupName(g.Name) && g.ID != activeSgid {
			_ = c.ServerGroupDelClient(g.ID, cldbid)
			// Optional: delete group if empty (similar to XP groups)
			b.maybeDeleteEmptyTitleGroup(c, g.ID, g.Name)
		}
	}
}

func (b *Bot) getOrCreateTitleGroup(c *clientquery.Client, name string) (int, error) {
	groups, err := c.ServerGroupList()
	if err == nil {
		for _, g := range groups {
			if g.Name == name {
				b.applyGroupTreePerm(c, g.ID, b.Cfg.TitleGroupShowNameInTree)
				return g.ID, nil
			}
		}
	}

	// Create title group
	if err := c.ServerGroupAdd(name); err != nil {
		return 0, err
	}

	// Re-list to find ID and show the title next to the member's name in the tree.
	groups, _ = c.ServerGroupList()
	for _, g := range groups {
		if g.Name == name {
			b.applyGroupTreePerm(c, g.ID, b.Cfg.TitleGroupShowNameInTree)
			return g.ID, nil
		}
	}
	return 0, fmt.Errorf("failed to find created title group")
}

func (b *Bot) isTitleGroupName(name string) bool {
	return content.IsTitle(name)
}

func (b *Bot) maybeDeleteEmptyTitleGroup(c *clientquery.Client, sgid int, name string) {
	n, err := c.ServerGroupMemberCount(sgid)
	if err == nil && n == 0 {
		_ = c.ServerGroupDel(sgid, true)
		log.Printf("titles: removed empty title group %q", name)
	}
}
