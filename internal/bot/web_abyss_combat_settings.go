package bot

import "net/http"

func (s *WebServer) handleAbyssCombatSettings(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method == http.MethodGet {
		writeJSON(w, map[string]any{
			"ok":          true,
			"hold_mana":   s.bot.abyssHoldMana(uid),
			"pet_command": s.bot.loadAbyssPetCommand(uid),
		})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		HoldMana   bool    `json:"hold_mana"`
		PetCommand *string `json:"pet_command"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var command abyssPetCommand
	if req.PetCommand == nil {
		command = s.bot.loadAbyssPetCommand(uid)
	} else {
		var validCommand bool
		command, validCommand = parseAbyssPetCommand(*req.PetCommand)
		if !validCommand {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid companion command"})
			return
		}
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "combat setting could not be saved"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	holdMana := ""
	if req.HoldMana {
		holdMana = "1"
	}
	if err := setAbyssCombatOption(tx, uid, "hold_mana", holdMana); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "combat setting could not be saved"})
		return
	}
	if err := setAbyssCombatOption(tx, uid, "pet_command", string(command)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "combat setting could not be saved"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "combat setting could not be saved"})
		return
	}
	writeJSON(w, map[string]any{
		"ok":          true,
		"hold_mana":   req.HoldMana,
		"pet_command": command,
	})
}
