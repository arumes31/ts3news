package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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
	hasActiveRelic bool,
	hardcore bool,
	at time.Time,
) (abyssWeeklyRule, error) {
	flags := map[string]int64{
		abyssRunFlagBuildKit:      abyssBuildKits[normalizeAbyssBuildKit(kit)],
		abyssRunFlagSkillMutation: abyssSkillMutations[normalizeAbyssSkillMutation(mutation)],
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
}

func publicFloorCandidates(candidates []floorCandidate, reveal bool) []floorCandidateView {
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
	rooms := []string{"challenge_room", "cursed_door", "story_crossroads", "lost_explorer", "locked_vault"}
	index := int(roll / (0.20 / float64(len(rooms))))
	if index >= len(rooms) {
		index = len(rooms) - 1
	}
	return fmt.Sprintf(`{"type":%q}`, rooms[index])
}

// prepareAbyssEventForDepth makes traveling-merchant inventory scale with the
// floor where it was discovered. Deep merchants carry the same fixed rolled
// stock, but its scarcity premium rises gradually and is persisted in state.
func prepareAbyssEventForDepth(raw string, depth int) string {
	if raw == "" {
		return raw
	}
	var state map[string]any
	if json.Unmarshal([]byte(raw), &state) != nil || state["type"] != "merchant" {
		return raw
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
	state["depth"] = depth
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

// handleAbyssSpecialRoom resolves the focused run-structure encounters outside
// web_abyss.go. It returns false only when the current event is owned by the
// legacy event switch.
func (s *WebServer) handleAbyssSpecialRoom(w http.ResponseWriter, uid string, run abyssRun, action string) bool {
	var state struct {
		Type string `json:"type"`
	}
	if json.Unmarshal([]byte(run.EventState), &state) != nil {
		return false
	}
	switch state.Type {
	case "challenge_room", "cursed_door", "story_crossroads", "lost_explorer", "locked_vault":
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
			flags["explorer_guard_floors"] = 2
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
			cost := run.MaxHP / 10
			if cost < 1 {
				cost = 1
			}
			newHP -= cost
			if newHP < 1 {
				newHP = 1
			}
			flags[abyssRunFlagVaultKeys]++
			flags["explorer_guard_floors"] = 3
			msg = "🧗 Explorer rescued: gain a vault key and +10% DEF for 3 fights."
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
		"vault_keys": flags[abyssRunFlagVaultKeys],
		"tokens":     s.bot.abyssTokens(uid),
	})
	return true
}

func (b *Bot) tickAbyssRoomEffects(uid string) {
	flags := b.loadRunFlags(uid)
	changed := false
	for _, key := range []string{"cursed_door_floors", "explorer_guard_floors"} {
		if flags[key] > 0 {
			flags[key]--
			changed = true
		}
	}
	if changed {
		_ = b.saveRunFlags(uid, flags)
	}
}
