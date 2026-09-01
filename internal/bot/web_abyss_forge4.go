package bot

// Advanced forge operations: batch temper with an insurance guard, the forge
// queue, bulk gem upgrade, rune
// scraping, un-attune, masterwork transfer, reforge lock (with the Eternal
// double-reforge privilege), bulk rebalance, brand/special/imbue removal and
// reroll, guided awaken, polish-all, Repair Kit II with crafting crit, socket
// relocation, fusion preview, blessed celestial fusion, recipe favorites,
// material conversion, the second daily undo purchase and forge mastery.
// Same forge shape as rounds 3-4 (web_abyss_forge2.go / web_abyss_forge3.go):
// per-uid lock, {inv_id|slot} body, one transaction, undo snapshot, guarded
// cost deduction, item_data rewrite, forge history, refreshed balances.
//
// Deliberate deviations, forced by file ownership:
//   - AB-108 (perfect corruption) is NOT here: the corrupt handler lives in
//     web_abyss_forge3.go (handleAbyssCorrupt) and needs a hidden 5% roll there.
//   - AB-117 (forge mastery) is counted here and applied by the shared
//     forgeGoldCost path so every Forge generation receives the same price.
//   - AB-125 sells the flag here; handleAbyssForgeUndo honors it and records
//     the second use independently from the base daily undo date.

import (
	"fmt"
	"strconv"

	"ts3news/internal/content"
)

// ---- Forge mastery (AB-117) ---------------------------------------------------

// forge4MasteryKey is the app_meta key counting a player's forge4 actions.
func forge4MasteryKey(uid string) string { return "abyss_forge_mastery_" + uid }

// forge4MasteryCount reads the lifetime forge4 action count.
func (b *Bot) forge4MasteryCount(uid string) int {
	var v string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", forge4MasteryKey(uid)).Scan(&v)
	n, _ := strconv.Atoi(v)
	return n
}

func forge4MasteryDiscountForCount(actions int) int {
	return min(5, max(0, actions)/50)
}

// forge4MasteryDiscount is +1% per 50 recorded actions, capped at 5%.
func (b *Bot) forge4MasteryDiscount(uid string) int {
	return forge4MasteryDiscountForCount(b.forge4MasteryCount(uid))
}

func (b *Bot) forge4MasteryAdd(uid string, actions int) {
	if actions <= 0 {
		return
	}
	_, _ = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = (COALESCE(NULLIF(app_meta.value, ''), '0')::int + $2)::text`,
		forge4MasteryKey(uid), actions)
}

// forge4GoldCost uses the shared Forge price path. forgeGoldCost applies
// mastery for every Forge generation so previews and commits stay identical.
func (s *WebServer) forge4GoldCost(uid string, base int64, r content.Rarity) int64 {
	return s.bot.forgeGoldCost(uid, base, r)
}

// forge4MasteryInfo is the response fragment describing the player's mastery.
func (b *Bot) forge4MasteryInfo(uid string) map[string]any {
	return map[string]any{"actions": b.forge4MasteryCount(uid), "discount_pct": b.forge4MasteryDiscount(uid)}
}

// forge4ItemKey identifies an item specifier as a stable string (used by the
// temper guard and the guided-awaken pending roll).
func forge4ItemKey(invID int64, slot string) string {
	if invID > 0 {
		return fmt.Sprintf("inv:%d", invID)
	}
	return "slot:" + slot
}
