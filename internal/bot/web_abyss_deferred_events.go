package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

const abyssRunFlagDeferredReturn = "deferred_event_return"

type abyssDeferredEvent struct {
	State        string `json:"state"`
	Label        string `json:"label"`
	OriginDepth  int    `json:"origin_depth"`
	ExpiresDepth int    `json:"expires_depth"`
}

type abyssDeferredEventView struct {
	Label        string
	OriginDepth  int
	ExpiresDepth int
	FloorsLeft   int
	Available    bool
	Expired      bool
}

func abyssDeferredEventKey(uid string) string { return "abyss_deferred_event_" + uid }

func abyssDeferredEventStatus(event abyssDeferredEvent, depth int) abyssDeferredEventView {
	floorsLeft := max(0, event.ExpiresDepth-depth)
	available := event.State != "" && depth <= event.ExpiresDepth
	return abyssDeferredEventView{
		Label: event.Label, OriginDepth: event.OriginDepth, ExpiresDepth: event.ExpiresDepth,
		FloorsLeft: floorsLeft, Available: available, Expired: event.State != "" && !available,
	}
}

func (b *Bot) abyssDeferredEventView(uid string, run abyssRun) *abyssDeferredEventView {
	if !run.Active {
		return nil
	}
	var raw string
	if b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssDeferredEventKey(uid)).Scan(&raw) != nil {
		return nil
	}
	var event abyssDeferredEvent
	if json.Unmarshal([]byte(raw), &event) != nil || event.State == "" {
		return nil
	}
	view := abyssDeferredEventStatus(event, run.Depth)
	return &view
}

func markAbyssEventDeferred(raw string) (state, label string, ok bool) {
	var event map[string]any
	if json.Unmarshal([]byte(raw), &event) != nil {
		return "", "", false
	}
	if already, _ := event["deferred"].(bool); already {
		return "", "", false
	}
	typ, _ := event["type"].(string)
	if typ == "" {
		return "", "", false
	}
	event["deferred"] = true
	encoded, err := json.Marshal(event)
	if err != nil {
		return "", "", false
	}
	return string(encoded), abyssEventTypeLabel(string(encoded)), true
}

func (s *WebServer) handleAbyssDeferEventAction(w http.ResponseWriter, uid string, run abyssRun, action string) bool {
	if action != "event_defer" {
		return false
	}
	state, label, ok := markAbyssEventDeferred(run.EventState)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "this event cannot be deferred again"})
		return true
	}
	event := abyssDeferredEvent{
		State: state, Label: label, OriginDepth: run.Depth, ExpiresDepth: run.Depth + 3,
	}
	encoded, _ := json.Marshal(event)
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	defer func() { _ = tx.Rollback() }()
	var existingRaw string
	err = tx.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssDeferredEventKey(uid)).Scan(&existingRaw)
	if err == nil {
		var existing abyssDeferredEvent
		if json.Unmarshal([]byte(existingRaw), &existing) == nil && run.Depth <= existing.ExpiresDepth {
			writeJSON(w, map[string]any{"ok": false, "error": "resolve or outwalk your deferred event first"})
			return true
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1,$2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, abyssDeferredEventKey(uid), string(encoded)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	if _, err := tx.Exec("UPDATE abyss_active SET event_state=NULL, last_action_at=NOW() WHERE client_uid=$1", uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return true
	}
	writeJSON(w, map[string]any{
		"ok": true, "resolved": true,
		"msg": "⏳ " + label + " deferred — return by floor " + strconv.Itoa(event.ExpiresDepth) + " or the moment passes.",
	})
	return true
}

func (s *WebServer) handleAbyssDeferredEventClaim(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.Downed || run.FloorType != "combat" {
		writeJSON(w, map[string]any{"ok": false, "error": "return between combat floors"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var raw string
	if err := tx.QueryRow("SELECT value FROM app_meta WHERE key=$1 FOR UPDATE", abyssDeferredEventKey(uid)).Scan(&raw); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "no deferred event"})
		return
	}
	var event abyssDeferredEvent
	if json.Unmarshal([]byte(raw), &event) != nil || event.State == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid deferred event"})
		return
	}
	if run.Depth > event.ExpiresDepth {
		_, _ = tx.Exec("DELETE FROM app_meta WHERE key=$1", abyssDeferredEventKey(uid))
		if err := tx.Commit(); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		writeJSON(w, map[string]any{"ok": false, "error": "the moment has passed"})
		return
	}
	flags, err := loadAbyssRunFlagsInTx(tx, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	flags[abyssRunFlagDeferredReturn] = 1
	if err := saveAbyssRunFlagsInTx(tx, uid, flags); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec(`UPDATE abyss_active SET floor_type='event', modifier='', event_state=$1,
		last_action_at=NOW() WHERE client_uid=$2`, event.State, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec("DELETE FROM app_meta WHERE key=$1", abyssDeferredEventKey(uid)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "reload": true})
}
