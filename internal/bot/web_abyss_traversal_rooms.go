package bot

// Traversal rooms change how a delver moves through the Abyss. They live apart
// from the legacy event switch because their authoritative previews and depth
// changes form a small, cohesive subsystem.

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
)

type abyssTraversalDestination struct {
	Depth int    `json:"depth"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

type abyssTraversalEvent struct {
	Type         string                      `json:"type"`
	Depth        int                         `json:"depth"`
	Destinations []abyssTraversalDestination `json:"destinations"`
	PassChance   int                         `json:"pass_chance"`
	GhostName    string                      `json:"ghost_name"`
	DeathDepth   int                         `json:"death_depth"`
	MissedFloors []string                    `json:"missed_floors"`
}

func abyssTrapPassChance(dodge, depth int) int {
	return min(95, max(10, 50+dodge-depth))
}

func abyssPortalMissedFloors(depth int) []string {
	cycle := []string{"Combat", "Rest", "Event", "Elite combat"}
	return []string{cycle[depth%len(cycle)], cycle[(depth+1)%len(cycle)], cycle[(depth+2)%len(cycle)]}
}

func prepareAbyssTraversalEvent(state map[string]any, depth int) {
	typ, _ := state["type"].(string)
	switch typ {
	case "cursed_elevator":
		state["destinations"] = []abyssTraversalDestination{
			{Depth: depth + 2, Type: "combat", Label: "Armoured patrol · standard cache"},
			{Depth: depth + 4, Type: "rest", Label: "Distant sanctuary · no skipped-floor loot"},
		}
	case "trap_chamber":
		state["difficulty"] = depth
	case "unstable_portal":
		state["missed_floors"] = abyssPortalMissedFloors(depth)
	case "graveyard":
		names := []string{"Ada Blackglass", "Corvin Ash", "Nyra Voss", "Sable Renn", "Torren Pike"}
		state["ghost_name"] = names[depth%len(names)]
		state["death_depth"] = max(1, depth-(depth%7)-1)
	}
}

func (b *Bot) enrichAbyssTraversalEvent(uid string, state map[string]any) {
	if state["type"] != "trap_chamber" {
		return
	}
	depth, _ := state["depth"].(float64)
	state["pass_chance"] = abyssTrapPassChance(b.abyssCombatStats(uid).DGE, int(depth))
}

func (s *WebServer) handleAbyssTraversalRoom(w http.ResponseWriter, uid string, run abyssRun, action string) bool {
	var state abyssTraversalEvent
	if json.Unmarshal([]byte(run.EventState), &state) != nil {
		return false
	}
	switch state.Type {
	case "cursed_elevator", "trap_chamber", "unstable_portal", "graveyard":
	default:
		return false
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	defer func() { _ = tx.Rollback() }()

	newDepth, newHP, newEscrow := run.Depth, run.CurHP, run.Escrow
	newFloorType := "event"
	msg := ""
	reload := false

	switch state.Type {
	case "cursed_elevator":
		index := -1
		if action == "elevator_choose_0" {
			index = 0
		} else if action == "elevator_choose_1" {
			index = 1
		}
		if index < 0 || index >= len(state.Destinations) {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid elevator destination"})
			return true
		}
		destination := state.Destinations[index]
		if destination.Depth <= run.Depth || (destination.Type != "combat" && destination.Type != "rest") {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid elevator route"})
			return true
		}
		newDepth, newFloorType, reload = destination.Depth, destination.Type, true
		msg = fmt.Sprintf("🛗 The cursed elevator delivers you to floor %d: %s.", destination.Depth, destination.Label)

	case "trap_chamber":
		switch action {
		case "trap_attempt":
			passed := rand.IntN(100) < state.PassChance // #nosec G404 -- posted gameplay roll
			if passed {
				gain := int64(500 + run.Depth*35)
				newEscrow += gain
				msg = fmt.Sprintf("🪤 Clean passage at %d%% odds: +%d cache.", state.PassChance, gain)
			} else {
				damage := max(1, run.MaxHP/8)
				newHP = max(1, newHP-damage)
				msg = fmt.Sprintf("🪤 The trap catches you at %d%% pass odds: −%d HP.", state.PassChance, damage)
			}
		case "trap_detour":
			msg = "🛡️ You take the slow route and leave the trap untouched."
		default:
			writeJSON(w, map[string]any{"ok": false, "error": "invalid trap choice"})
			return true
		}

	case "unstable_portal":
		switch action {
		case "portal_enter":
			if len(state.MissedFloors) != 3 {
				writeJSON(w, map[string]any{"ok": false, "error": "invalid portal route"})
				return true
			}
			newDepth, newFloorType, reload = run.Depth+3, "combat", true
			msg = "🌀 The portal skips three floors. What you missed: " + joinAbyssLabels(state.MissedFloors) + "."
		case "portal_leave":
			msg = "You let the unstable portal collapse behind you."
		default:
			writeJSON(w, map[string]any{"ok": false, "error": "invalid portal choice"})
			return true
		}

	case "graveyard":
		switch action {
		case "graveyard_honor":
			if err := grantMaterialQ(tx, uid, "dust", 2); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return true
			}
			msg = fmt.Sprintf("🪦 You honor %s, lost at depth %d, and recover 2 Abyssal Dust.", state.GhostName, state.DeathDepth)
		case "graveyard_disturb":
			gain := int64(700 + run.Depth*40)
			newEscrow += gain
			newHP = max(1, newHP-max(1, run.MaxHP/10))
			msg = fmt.Sprintf("👻 You disturb %s's grave: +%d cache, −10%% maximum HP.", state.GhostName, gain)
		default:
			writeJSON(w, map[string]any{"ok": false, "error": "invalid graveyard choice"})
			return true
		}
	}

	if _, err := tx.Exec("UPDATE users SET current_hp=$1 WHERE client_uid=$2", newHP, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	if _, err := tx.Exec(`UPDATE abyss_active SET depth=$1, escrow=$2, floor_type=$3,
		event_state=NULL, last_action_at=NOW() WHERE client_uid=$4`, newDepth, newEscrow, newFloorType, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	if reload {
		flags, err := loadAbyssRunFlagsInTx(tx, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
		flags[abyssRunFlagEventSigils]++
		if err := saveAbyssRunFlagsInTx(tx, uid, flags); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return true
		}
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	writeJSON(w, map[string]any{
		"ok": true, "resolved": true, "reload": reload, "msg": msg,
		"depth": newDepth, "hp": newHP, "escrow": newEscrow,
		"floor_type": newFloorType, "materials": s.bot.loadMaterials(uid),
	})
	return true
}

func joinAbyssLabels(labels []string) string {
	if len(labels) == 0 {
		return "nothing the map could hold"
	}
	result := labels[0]
	for _, label := range labels[1:] {
		result += " · " + label
	}
	return result
}
