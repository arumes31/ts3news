package bot

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"

	"ts3news/internal/content"
)

// bbTagRe strips every BBCode tag, leaving plain text for labels and tooltips.
var bbTagRe = regexp.MustCompile(`\[[^\]]*\]`)

// runLootRow is the authoritative presentation model for one escrowed drop.
// Label is safe HTML produced by bbToHTML, which escapes input before restoring
// the small BBCode allowlist.
type runLootRow struct {
	EscrowID        int64         `json:"id"`
	Label           template.HTML `json:"label"`
	Depth           int           `json:"depth"`
	Title           string        `json:"title"`
	Source          string        `json:"source"`
	ItemType        string        `json:"item_type"`
	GearID          string        `json:"gear_id,omitempty"`
	Slot            string        `json:"slot,omitempty"`
	SlotIcon        string        `json:"slot_icon,omitempty"`
	Rarity          string        `json:"rarity,omitempty"`
	RarityRank      int           `json:"rarity_rank,omitempty"`
	Score           int           `json:"score,omitempty"`
	CR              float64       `json:"cr,omitempty"`
	CRDelta         float64       `json:"cr_delta,omitempty"`
	MainStat        int           `json:"main_stat,omitempty"`
	BeamClass       string        `json:"beam_class,omitempty"`
	Quality         int           `json:"quality,omitempty"`
	Foil            bool          `json:"foil,omitempty"`
	Doomed          bool          `json:"doomed,omitempty"`
	Unidentified    bool          `json:"unidentified,omitempty"`
	AlreadyOwned    bool          `json:"already_owned,omitempty"`
	SetID           string        `json:"set_id,omitempty"`
	SetCount        int           `json:"set_count,omitempty"`
	SetMax          int           `json:"set_max,omitempty"`
	Corrupted       bool          `json:"corrupted,omitempty"`
	EmptySlot       bool          `json:"empty_slot,omitempty"`
	CanEquipBest    bool          `json:"can_equip_best,omitempty"`
	EquipOnBank     bool          `json:"equip_on_bank,omitempty"`
	SmartLoot       bool          `json:"smart_loot,omitempty"`
	SmartLootReason string        `json:"smart_loot_reason,omitempty"`
	SmartLootLabel  string        `json:"smart_loot_label,omitempty"`
	SetPity         bool          `json:"set_pity,omitempty"`
	SetPityLabel    string        `json:"set_pity_label,omitempty"`
	Wishlist        bool          `json:"wishlist,omitempty"`
	WishlistLabel   string        `json:"wishlist_label,omitempty"`
	Provenance      string        `json:"provenance,omitempty"`
	CanSellJunk     bool          `json:"can_sell_junk,omitempty"`
	SellValue       int64         `json:"sell_value,omitempty"`
}

func abyssRunLootJunkQuote(grant abyssLootGrant) (int64, bool) {
	if grant.Type != "gear" || grant.Gear == nil || grant.Gear.Unidentified ||
		grant.Wishlist || grant.SetPity || grant.SmartLoot {
		return 0, false
	}
	gear := grant.Gear
	if gear.Rarity > content.RarityUncommon || gear.SetID != "" || gear.Foil ||
		gear.Special != content.EffectNone || len(gear.BonusEffects) > 0 || gear.Attuned ||
		gear.Sockets > 0 || len(gear.Gemstones) > 0 || gear.Rune != "" ||
		gear.Temper > 0 || gear.Quality > 0 || gear.Awakened || gear.Imbued != "" ||
		gear.Cursed || gear.Eldritch || gear.Insured || gear.Corrupted || gear.Embraced ||
		gear.Prismatic || gear.Reinforced > 0 || gear.Sharpened > 0 || gear.Doomed ||
		gear.Lucid || gear.AppearanceID != "" || gear.RegenAmount > 0 || gear.RegenIntervalSec > 0 {
		return 0, false
	}
	return max(gearPrice(*gear)/2, int64(1)), true
}

func abyssSetDisplayMax(setID string) int {
	switch setID {
	case "predator", "warden":
		return 6
	case "harvester":
		return 3
	default:
		return 0
	}
}

func abyssOwnedGearIDs(b *Bot, uid string) map[string]bool {
	owned := map[string]bool{}
	rows, err := b.DB.Query(`SELECT gear_id FROM user_gear WHERE client_uid=$1
		UNION SELECT gear_id FROM user_inventory WHERE client_uid=$1`, uid)
	if err != nil {
		return owned
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Printf("abyss owned gear scan failed for %s: %v", uid, err)
			return owned
		}
		owned[id] = true
	}
	if err := rows.Err(); err != nil {
		log.Printf("abyss owned gear iteration failed for %s: %v", uid, err)
	}
	return owned
}

func abyssOwnedGearSet(equipped map[content.GearSlot]content.Gear, inventory []gearView) map[string]bool {
	owned := make(map[string]bool, len(equipped)+len(inventory))
	for _, gear := range equipped {
		owned[gear.ID] = true
	}
	for _, gear := range inventory {
		owned[gear.ID] = true
	}
	return owned
}

func abyssLootMainStat(gear content.Gear, buildKit int64) int {
	switch abyssBuildNameByValue(abyssBuildKits, buildKit) {
	case "arcanist":
		return gear.Stats.INT
	case "survival":
		return gear.Stats.HP + gear.Stats.DEF
	default:
		return gear.Stats.STR
	}
}

// currentRunLootManifest returns every escrowed item oldest-first and derives
// gear presentation from the serialized grant, never from localized label text.
func (b *Bot) currentRunLootManifest(uid string, equipped map[content.GearSlot]content.Gear, owned map[string]bool) []runLootRow {
	buildKit := b.loadRunFlags(uid)[abyssRunFlagBuildKit]
	setCounts := map[string]int{}
	for _, gear := range equipped {
		if gear.SetID != "" {
			setCounts[gear.SetID]++
		}
	}
	rows, err := b.DB.Query(`SELECT id, label, depth, item_type, item_data, equip_on_bank
		FROM abyss_escrow_loot WHERE client_uid=$1 ORDER BY id`, uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	out := []runLootRow{}
	for rows.Next() {
		var row runLootRow
		var label string
		var data []byte
		if err := rows.Scan(&row.EscrowID, &label, &row.Depth, &row.ItemType, &data, &row.EquipOnBank); err != nil {
			log.Printf("abyss run loot scan failed for %s: %v", uid, err)
			return nil
		}
		row.Label = template.HTML(bbToHTML(label)) // #nosec G203 -- bbToHTML escapes before restoring allowlisted tags
		row.Title = bbTagRe.ReplaceAllString(label, "")
		if row.Depth > 0 {
			row.Source = fmt.Sprintf("Dropped floor %d", row.Depth)
		} else {
			row.Source = "Run cache"
		}
		var grant abyssLootGrant
		if json.Unmarshal(data, &grant) == nil && grant.Gear != nil {
			gear := *grant.Gear
			row.Wishlist = grant.Wishlist
			if row.Wishlist {
				row.WishlistLabel = "Wishlist guarantee"
			}
			row.SetPityLabel = abyssSetPityTag(grant.SetPitySetID)
			row.SetPity = grant.SetPity && row.SetPityLabel != ""
			row.SmartLootLabel = abyssSmartLootTag(grant.SmartLootReason)
			row.SmartLoot = grant.SmartLoot && row.SmartLootLabel != ""
			if row.SmartLoot {
				row.SmartLootReason = grant.SmartLootReason
			}
			row.Slot = string(gear.Slot)
			row.SlotIcon = content.SlotIcon(gear.Slot)
			row.Rarity = gear.Rarity.String()
			row.RarityRank = int(gear.Rarity)
			row.BeamClass = abyssBeamClass(gear.Rarity, gear.Doomed && !gear.Unidentified)
			row.Foil = gear.Foil
			row.Unidentified = gear.Unidentified
			if gear.Unidentified {
				out = append(out, row)
				continue
			}
			row.Provenance = gearProvenance(gear)
			row.GearID = gear.ID
			row.Score = gear.Stats.Score()
			row.CR = gear.CombatRating()
			row.MainStat = abyssLootMainStat(gear, buildKit)
			row.Quality = gear.Quality
			row.AlreadyOwned = owned[gear.ID]
			row.SetID = gear.SetID
			row.SetCount = setCounts[gear.SetID]
			row.SetMax = abyssSetDisplayMax(gear.SetID)
			row.Corrupted = gear.Corrupted
			row.Doomed = gear.Doomed
			row.SellValue, row.CanSellJunk = abyssRunLootJunkQuote(grant)
			row.CanSellJunk = row.CanSellJunk && !row.EquipOnBank
			current, occupied := equipped[gear.Slot]
			row.EmptySlot = !occupied
			if occupied {
				row.CRDelta = row.CR - current.CombatRating()
			} else {
				row.CRDelta = row.CR
			}
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		log.Printf("abyss run loot iteration failed for %s: %v", uid, err)
		return nil
	}
	markAbyssBestRunLoot(out)
	return out
}

func (s *WebServer) handleAbyssSellJunkLoot(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	var req struct {
		EscrowID   int64 `json:"id"`
		QuotedGold int64 `json:"quoted_gold"`
	}
	if err := readJSON(r, &req); err != nil || req.EscrowID <= 0 || req.QuotedGold <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid junk-sale quote"})
		return
	}
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var downed bool
	if err := tx.QueryRowContext(r.Context(), `SELECT downed FROM abyss_active
		WHERE client_uid=$1 FOR UPDATE`, uid).Scan(&downed); err != nil || downed {
		writeJSON(w, map[string]any{"ok": false, "error": "an active, standing run is required"})
		return
	}
	var itemType string
	var itemData []byte
	var equipOnBank bool
	if err := tx.QueryRowContext(r.Context(), `SELECT item_type,item_data,equip_on_bank
		FROM abyss_escrow_loot WHERE id=$1 AND client_uid=$2 FOR UPDATE`, req.EscrowID, uid).
		Scan(&itemType, &itemData, &equipOnBank); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "loot item not found"})
		return
	}
	var grant abyssLootGrant
	if itemType != "gear" || json.Unmarshal(itemData, &grant) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "only ordinary gear can be sold as junk"})
		return
	}
	value, eligible := abyssRunLootJunkQuote(grant)
	if !eligible || equipOnBank {
		writeJSON(w, map[string]any{"ok": false, "error": "protected or special loot cannot be sold as junk"})
		return
	}
	if req.QuotedGold != value {
		writeJSON(w, map[string]any{"ok": false, "error": "junk value changed; review the current quote", "current_value": value})
		return
	}
	result, err := tx.ExecContext(r.Context(), "DELETE FROM abyss_escrow_loot WHERE id=$1 AND client_uid=$2", req.EscrowID, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "loot item changed before sale"})
		return
	}
	var escrow int64
	if err := tx.QueryRowContext(r.Context(), `UPDATE abyss_active SET escrow=escrow+$1,last_action_at=NOW()
		WHERE client_uid=$2 RETURNING escrow`, value, uid).Scan(&escrow); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "value": value, "escrow": escrow,
		"msg": fmt.Sprintf("Junk converted into %dg of at-risk run cache.", value),
	})
}

func markAbyssBestRunLoot(rows []runLootRow) {
	best := map[string]int{}
	for i := range rows {
		row := &rows[i]
		if row.Slot == "" || row.CRDelta <= 0 {
			continue
		}
		if previous, ok := best[row.Slot]; !ok || rows[previous].CR < row.CR {
			best[row.Slot] = i
		}
	}
	for _, index := range best {
		rows[index].CanEquipBest = true
	}
}

func (b *Bot) consumeAndEquipAbyssEscrowGear(uid string, escrowID int64, gear content.Gear) error {
	data, err := json.Marshal(gear)
	if err != nil {
		return fmt.Errorf("marshal preferred Abyss gear: %w", err)
	}
	tx, err := b.DB.Begin()
	if err != nil {
		return fmt.Errorf("begin preferred Abyss equip: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.Exec("DELETE FROM abyss_escrow_loot WHERE id=$1 AND client_uid=$2 AND equip_on_bank=TRUE", escrowID, uid)
	if err != nil {
		return fmt.Errorf("consume preferred Abyss gear: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		return fmt.Errorf("consume preferred Abyss gear: escrow row changed")
	}
	if err := b.equipGear(tx, uid, gear, gear.MaxDurability, string(data)); err != nil {
		return fmt.Errorf("equip preferred Abyss gear: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit preferred Abyss gear: %w", err)
	}
	return nil
}

func (s *WebServer) handleAbyssLootManifest(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	equipped := s.bot.getEquippedItems(uid)
	recentGear, _ := s.bot.abyssRecentGearProtection(uid)
	writeJSON(w, map[string]any{
		"ok":                     true,
		"items":                  s.bot.currentRunLootManifest(uid, equipped, abyssOwnedGearIDs(s.bot, uid)),
		"recent_gear_protected":  len(recentGear),
		"duplicate_floor_window": abyssRecentGearFloorWindow,
	})
}

func (s *WebServer) handleAbyssEquipBestLoot(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	var req struct {
		EscrowID int64 `json:"id"`
	}
	if err := readJSON(r, &req); err != nil || req.EscrowID <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid loot item"})
		return
	}
	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "no active run"})
		return
	}
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(r.Context(), `SELECT id, item_data, equip_on_bank
		FROM abyss_escrow_loot WHERE client_uid=$1 AND item_type='gear' ORDER BY id FOR UPDATE`, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	type candidate struct {
		id       int64
		gear     content.Gear
		selected bool
	}
	candidates := []candidate{}
	for rows.Next() {
		var candidate candidate
		var data []byte
		var grant abyssLootGrant
		if err := rows.Scan(&candidate.id, &data, &candidate.selected); err != nil || json.Unmarshal(data, &grant) != nil || grant.Gear == nil {
			_ = rows.Close()
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		candidate.gear = *grant.Gear
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	_ = rows.Close()

	var requested *candidate
	for i := range candidates {
		if candidates[i].id == req.EscrowID {
			requested = &candidates[i]
			break
		}
	}
	if requested == nil {
		writeJSON(w, map[string]any{"ok": false, "error": "loot item is no longer available"})
		return
	}
	if requested.gear.Unidentified {
		writeJSON(w, map[string]any{"ok": false, "error": "identify this item before reserving it"})
		return
	}
	equipped := s.bot.getEquippedItems(uid)
	currentCR := 0.0
	if current, ok := equipped[requested.gear.Slot]; ok {
		currentCR = current.CombatRating()
	}
	bestID, bestCR := int64(0), currentCR
	for i := range candidates {
		candidate := &candidates[i]
		if !candidate.gear.Unidentified && candidate.gear.Slot == requested.gear.Slot && candidate.gear.CombatRating() > bestCR {
			bestID, bestCR = candidate.id, candidate.gear.CombatRating()
		}
	}
	if bestID != req.EscrowID {
		writeJSON(w, map[string]any{"ok": false, "error": "this is no longer the best upgrade for that slot"})
		return
	}
	for i := range candidates {
		candidate := &candidates[i]
		if candidate.selected && candidate.gear.Slot == requested.gear.Slot && candidate.id != requested.id {
			if _, err := tx.ExecContext(r.Context(), "UPDATE abyss_escrow_loot SET equip_on_bank=FALSE WHERE id=$1", candidate.id); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
		}
	}
	result, err := tx.ExecContext(r.Context(), "UPDATE abyss_escrow_loot SET equip_on_bank=TRUE WHERE id=$1 AND client_uid=$2", requested.id, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "loot item changed"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "id": requested.id, "slot": requested.gear.Slot})
}
