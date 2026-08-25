package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ts3news/internal/content"
)

const (
	abyssRunFlagWeeklySeed = "weekly_expedition_seed"
	abyssRunFlagVaultKeys  = "vault_keys"
)

type abyssWeeklyRule struct {
	Key        string
	Label      string
	DangerMult float64
}

var abyssWeeklyRules = []abyssWeeklyRule{
	{Key: "elite_surge", Label: "Elite Surge: every enemy is enraged", DangerMult: 1.20},
	{Key: "volatile_depths", Label: "Volatile Depths: hazards trigger twice", DangerMult: 1.10},
	{Key: "iron_trial", Label: "Iron Trial: enemies gain 15% DEF", DangerMult: 1.15},
}

// abyssWeeklyExpedition is stable for the entire ISO week. Its seed is copied
// into the run at entry, so a run that crosses a weekly boundary keeps its
// original rules until it ends.
func abyssWeeklyExpedition(at time.Time) (int64, abyssWeeklyRule) {
	year, week := at.UTC().ISOWeek()
	seed := int64(year*100 + week)
	return seed, abyssWeeklyRules[seed%int64(len(abyssWeeklyRules))]
}

func abyssWeeklyRuleFromFlags(flags map[string]int64) (abyssWeeklyRule, bool) {
	seed := flags[abyssRunFlagWeeklySeed]
	if seed <= 0 {
		return abyssWeeklyRule{}, false
	}
	return abyssWeeklyRules[seed%int64(len(abyssWeeklyRules))], true
}

func resetAbyssRunFlagsInTx(
	tx *sql.Tx,
	uid string,
	weekly bool,
	kit string,
	mutation string,
	focusID int64,
	hasActiveRelic bool,
	hardcore bool,
	at time.Time,
) (abyssWeeklyRule, error) {
	flags := map[string]int64{
		abyssRunFlagBuildKit:      abyssBuildKits[normalizeAbyssBuildKit(kit)],
		abyssRunFlagSkillMutation: abyssSkillMutations[normalizeAbyssSkillMutation(mutation)],
	}
	if focusID > 0 {
		flags[abyssRunFlagFocus] = focusID
	}
	if hasActiveRelic {
		flags[abyssRunFlagRelicCharges] = 1
	}
	if hardcore {
		flags[abyssRunFlagHardcore] = 1
	}
	var rule abyssWeeklyRule
	if weekly {
		seed, selected := abyssWeeklyExpedition(at)
		flags[abyssRunFlagWeeklySeed] = seed
		rule = selected
	}
	data, err := json.Marshal(flags)
	if err != nil {
		return abyssWeeklyRule{}, fmt.Errorf("marshal run flags: %w", err)
	}
	_, err = tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssRunFlagsKey(uid), string(data))
	if err != nil {
		return abyssWeeklyRule{}, fmt.Errorf("reset run flags: %w", err)
	}
	return rule, nil
}

// floorCandidateView deliberately omits the authoritative room type. A hidden
// route cannot be uncovered by inspecting the JSON response.
type floorCandidateView struct {
	Index    int    `json:"index"`
	Label    string `json:"label"`
	Icon     string `json:"icon"`
	Revealed bool   `json:"revealed"`
	LootHint string `json:"loot_hint,omitempty"`
}

func publicFloorCandidates(candidates []floorCandidate, reveal bool, depth int) []floorCandidateView {
	views := make([]floorCandidateView, len(candidates))
	for i, candidate := range candidates {
		views[i] = floorCandidateView{
			Index: candidate.Index,
			Label: "Unexplored route",
			Icon:  "🌫️",
		}
	}
	if reveal && len(candidates) > 0 {
		candidate := candidates[0]
		views[0] = floorCandidateView{
			Index:    candidate.Index,
			Label:    candidate.Label,
			Icon:     candidate.Icon,
			Revealed: true,
			LootHint: abyssLootHint(candidate.Type, depth),
		}
	}
	return views
}

// abyssSpecialRoomForRoll keeps special-room selection testable without tying
// the room behavior to a particular random-number implementation.
func abyssSpecialRoomForRoll(roll float64) string {
	if roll < 0 || roll >= 0.20 {
		return ""
	}
	rooms := []string{
		"challenge_room", "cursed_door", "story_crossroads", "lost_explorer",
		"locked_vault", "collapsed_passage", "abyssal_garden", "cursed_elevator",
		"trap_chamber", "unstable_portal", "graveyard", "echo_floor", "bounty_board",
		abyssForgeFloorType,
	}
	index := int(roll / (0.20 / float64(len(rooms))))
	if index >= len(rooms) {
		index = len(rooms) - 1
	}
	return fmt.Sprintf(`{"type":%q}`, rooms[index])
}

// prepareAbyssEventForDepth stamps the discovery depth on every encounter and
// makes traveling-merchant inventory scale with the floor where it was found.
// Deep merchants carry the same fixed rolled stock, but its scarcity premium
// rises gradually and is persisted in state.
func prepareAbyssEventForDepth(raw string, depth int) string {
	if raw == "" {
		return raw
	}
	var state map[string]any
	if json.Unmarshal([]byte(raw), &state) != nil {
		return raw
	}
	state["depth"] = depth
	prepareAbyssTraversalEvent(state, depth)
	switch state["type"] {
	case "lost_explorer":
		index := depth % len(abyssExplorerNames)
		state["npc_id"] = index
		state["name"] = abyssExplorerName(index)
	case "abyssal_garden":
		nodes := []string{"dust", "shard", "core"}
		state["node"] = nodes[depth%len(nodes)]
	}
	if state["type"] != "merchant" {
		encoded, err := json.Marshal(state)
		if err != nil {
			return raw
		}
		return string(encoded)
	}
	items, ok := state["items"].([]any)
	if !ok {
		return raw
	}
	premium := depth
	if premium > 100 {
		premium = 100
	}
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		price, ok := item["price"].(float64)
		if ok {
			item["price"] = int64(price) * int64(100+premium) / 100
		}
	}
	// AB-34: every market visit has one opaque, flat-price mystery box. The
	// result is rolled only after purchase, so event_state never leaks it.
	state["mystery_available"] = true
	state["mystery_price"] = int64(750)
	encoded, err := json.Marshal(state)
	if err != nil {
		return raw
	}
	return string(encoded)
}

func loadAbyssRunFlagsInTx(tx *sql.Tx, uid string) (map[string]int64, error) {
	flags := map[string]int64{}
	var raw sql.NullString
	err := tx.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssRunFlagsKey(uid)).Scan(&raw)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if raw.Valid && raw.String != "" {
		if err := json.Unmarshal([]byte(raw.String), &flags); err != nil {
			return nil, err
		}
	}
	return flags, nil
}

func saveAbyssRunFlagsInTx(tx *sql.Tx, uid string, flags map[string]int64) error {
	data, err := json.Marshal(flags)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssRunFlagsKey(uid), string(data))
	return err
}

func clearAbyssSpecialRoomInTx(tx *sql.Tx, uid string) error {
	_, err := tx.Exec("UPDATE abyss_active SET event_state=NULL, last_action_at=NOW() WHERE client_uid=$1", uid)
	return err
}

func incrementAbyssMetaCounter(tx *sql.Tx, key string) (int, error) {
	var raw string
	err := tx.QueryRow(`INSERT INTO app_meta (key, value) VALUES ($1, '1')
		ON CONFLICT (key) DO UPDATE SET value = (COALESCE(NULLIF(app_meta.value, '')::int, 0) + 1)::text
		RETURNING value`, key).Scan(&raw)
	if err != nil {
		return 0, err
	}
	var count int
	_, err = fmt.Sscan(raw, &count)
	return count, err
}

func escrowExplorerKeepsake(tx *sql.Tx, uid string, depth int) error {
	gear, ok := content.GetGearByID("ABYSS_CURSED_COMPASS")
	if !ok {
		return fmt.Errorf("explorer keepsake gear missing")
	}
	data, err := json.Marshal(abyssLootGrant{Type: "gear", Gear: &gear})
	if err != nil {
		return err
	}
	label := fmt.Sprintf("%s [s:%s] (gs:%d R:%s)", gear.Name, string(gear.Slot), gear.Stats.Score(), gear.Rarity.String())
	_, err = tx.Exec("INSERT INTO abyss_escrow_loot (client_uid, item_type, label, item_data, depth) VALUES ($1,$2,$3,$4,$5)", uid, "gear", label, data, depth)
	return err
}

func abyssExplorerKeepsakeDue(rescueCount int) bool { return rescueCount == 2 }

func abyssGardenHarvestReward(harvestCount int) (amount int, greenThumb bool) {
	greenThumb = harvestCount >= 3
	amount = 2
	if greenThumb {
		amount++
	}
	return amount, greenThumb
}

// handleAbyssSpecialRoom resolves the focused run-structure encounters outside
// web_abyss.go. It returns false only when the current event is owned by the
// legacy event switch.
func (s *WebServer) handleAbyssSpecialRoom(w http.ResponseWriter, uid string, run abyssRun, action string) bool {
	var state struct {
		Type  string `json:"type"`
		Name  string `json:"name"`
		NPCID int    `json:"npc_id"`
		Node  string `json:"node"`
	}
	if json.Unmarshal([]byte(run.EventState), &state) != nil {
		return false
	}
	switch state.Type {
	case "challenge_room", "cursed_door", "story_crossroads", "lost_explorer", "locked_vault", "collapsed_passage", "abyssal_garden", abyssForgeFloorType:
	default:
		return false
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	defer func() { _ = tx.Rollback() }()

	flags, err := loadAbyssRunFlagsInTx(tx, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	msg := ""
	newEscrow := run.Escrow
	newHP := run.CurHP
	keepsake := false
	greenThumb := false

	switch state.Type {
	case "challenge_room":
		switch action {
		case "challenge_attempt":
			cost := run.MaxHP / 10
			if cost < 1 {
				cost = 1
			}
			newHP -= cost
			if newHP < 1 {
				newHP = 1
			}
			newEscrow += int64(600 + run.Depth*40)
			flags[abyssRunFlagVaultKeys]++
			msg = "⚔️ Trial conquered: cache enriched and a vault key claimed."
		case "challenge_leave":
			msg = "You leave the optional trial untouched."
		default:
			writeJSON(w, map[string]any{"ok": false, "error": "invalid challenge choice"})
			return true
		}
	case "cursed_door":
		switch action {
		case "cursed_door_enter":
			newEscrow += int64(1000 + run.Depth*50)
			flags["cursed_door_floors"] = 3
			msg = "🚪 The cursed door yields treasure, but weakens you for the next 3 fights."
		case "cursed_door_leave":
			msg = "You refuse the cursed door's bargain."
		default:
			writeJSON(w, map[string]any{"ok": false, "error": "invalid door choice"})
			return true
		}
	case "story_crossroads":
		switch action {
		case "story_guide":
			flags[abyssRunFlagVaultKeys]++
			flags[abyssRunFlagExplorerGuardFloors] = 2
			msg = "🧭 You guide the shades home. They leave a vault key and a protective map."
		case "story_plunder":
			newEscrow += int64(1400 + run.Depth*30)
			flags["cursed_door_floors"] = 2
			msg = "🗡️ You plunder the memorial. The cache grows, and its dead remember."
		default:
			writeJSON(w, map[string]any{"ok": false, "error": "invalid story choice"})
			return true
		}
	case "lost_explorer":
		switch action {
		case "explorer_rescue":
			if state.NPCID < 0 || state.NPCID >= len(abyssExplorerNames) {
				writeJSON(w, map[string]any{"ok": false, "error": "invalid explorer identity"})
				return true
			}
			explorerName := abyssExplorerName(state.NPCID)
			cost := run.MaxHP / 10
			if cost < 1 {
				cost = 1
			}
			newHP -= cost
			if newHP < 1 {
				newHP = 1
			}
			flags[abyssRunFlagVaultKeys]++
			flags[abyssRunFlagExplorerGuardFloors] = abyssExplorerSupportFloors
			flags[abyssRunFlagExplorerSupportID] = int64(state.NPCID + 1)
			rescueCount, err := incrementAbyssMetaCounter(tx, fmt.Sprintf("abyss_explorer_rescues_%s_%d", uid, state.NPCID))
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return true
			}
			msg = fmt.Sprintf("🧗 %s rescued: they fight beside you and grant +10%% DEF for the next 3 fights.", explorerName)
			if abyssExplorerKeepsakeDue(rescueCount) {
				if err := escrowExplorerKeepsake(tx, uid, run.Depth); err != nil {
					writeJSON(w, map[string]any{"ok": false, "error": "db"})
					return true
				}
				keepsake = true
				msg += " They remember you and entrust their Cursed Compass keepsake to the run cache."
			}
		case "explorer_leave":
			msg = "The explorer's lantern fades behind you."
		default:
			writeJSON(w, map[string]any{"ok": false, "error": "invalid rescue choice"})
			return true
		}
	case "locked_vault":
		switch action {
		case "vault_cache", "vault_tokens", "vault_materials":
			if flags[abyssRunFlagVaultKeys] <= 0 {
				writeJSON(w, map[string]any{"ok": false, "error": "the vault requires a key"})
				return true
			}
			flags[abyssRunFlagVaultKeys]--
			switch action {
			case "vault_cache":
				newEscrow += int64(2400 + run.Depth*100)
				msg = "🔐 Cache chosen: the vault seals a great heap into your run."
			case "vault_tokens":
				if _, err := tx.Exec("UPDATE users SET abyss_tokens=abyss_tokens+6 WHERE client_uid=$1", uid); err != nil {
					writeJSON(w, map[string]any{"ok": false, "error": "db"})
					return true
				}
				msg = "🔐 Token coffer chosen: gain 6 Abyss Tokens."
			case "vault_materials":
				if err := grantMaterialQ(tx, uid, "core", 4); err != nil {
					writeJSON(w, map[string]any{"ok": false, "error": "db"})
					return true
				}
				msg = "🔐 Forge cache chosen: gain 4 Umbral Cores."
			}
		case "vault_leave":
			msg = "The locked vault remains sealed."
		default:
			writeJSON(w, map[string]any{"ok": false, "error": "invalid vault choice"})
			return true
		}
	case "collapsed_passage":
		switch action {
		case "passage_detour":
			if err := grantMaterialQ(tx, uid, "dust", 1); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return true
			}
			msg = "🪨 The safe detour is slow, but loose stone yields 1 Abyssal Dust."
		case "passage_squeeze":
			cost := max(1, run.MaxHP/10)
			newHP = max(1, newHP-cost)
			if err := grantMaterialQ(tx, uid, "shard", 3); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return true
			}
			msg = "🧗 You force the dangerous gap, lose 10% maximum HP, and pry out 3 Void Shards."
		default:
			writeJSON(w, map[string]any{"ok": false, "error": "invalid passage choice"})
			return true
		}
	case "abyssal_garden":
		if action != "garden_harvest" {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid garden choice"})
			return true
		}
		material := state.Node
		if material != "dust" && material != "shard" && material != "core" {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid garden node"})
			return true
		}
		count, err := incrementAbyssMetaCounter(tx, fmt.Sprintf("abyss_garden_harvests_%s_%s", uid, material))
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		amount, mastered := abyssGardenHarvestReward(count)
		greenThumb = mastered
		if err := grantMaterialQ(tx, uid, material, amount); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		msg = fmt.Sprintf("🌿 You harvest %d %s from the garden (%d/3 familiarity).", amount, abyssMaterialName(material), min(count, 3))
		if count == 3 {
			msg += " Green Thumb mastered: every future node of this type yields +1 material."
		}
	case abyssForgeFloorType:
		if action != "forge_floor_leave" {
			writeJSON(w, map[string]any{"ok": false, "error": "choose a free forge action or leave the anvil"})
			return true
		}
		msg = "⚒️ You leave the Silent Anvil unused."
	}

	if _, err := tx.Exec("UPDATE users SET current_hp=$1 WHERE client_uid=$2", newHP, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	if _, err := tx.Exec("UPDATE abyss_active SET escrow=$1 WHERE client_uid=$2", newEscrow, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	if err := saveAbyssRunFlagsInTx(tx, uid, flags); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	if err := clearAbyssSpecialRoomInTx(tx, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	writeJSON(w, map[string]any{
		"ok": true, "resolved": true, "msg": msg,
		"hp": newHP, "escrow": newEscrow,
		"vault_keys":       flags[abyssRunFlagVaultKeys],
		"tokens":           s.bot.abyssTokens(uid),
		"keepsake":         keepsake,
		"green_thumb":      greenThumb,
		"materials":        s.bot.loadMaterials(uid),
		"explorer_support": abyssRescueSupportViewFromFlags(flags),
	})
	return true
}

func (b *Bot) tickAbyssRoomEffects(uid string) {
	flags := b.loadRunFlags(uid)
	changed := false
	for _, key := range []string{"cursed_door_floors", "spd_curse"} {
		if flags[key] > 0 {
			flags[key]--
			changed = true
		}
	}
	changed = tickAbyssRescueSupport(flags) || changed
	if changed {
		_ = b.saveRunFlags(uid, flags)
	}
}
