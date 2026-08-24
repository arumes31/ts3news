package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

func (s *WebServer) rejectDuringLiveCombat(w http.ResponseWriter, uid string) bool {
	combat, ok := s.liveCombatForUID(uid)
	if !ok || !combat.isActive() {
		return false
	}
	writeJSON(w, map[string]any{
		"ok": false, "error": "finish the active combat first",
		"state": combat.snapshotFor(uid),
	})
	return true
}

func (s *WebServer) handleAbyssCombatState(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	c, ok := s.liveCombatForUID(uid)
	if !ok {
		if snapshot, found := s.persistedAbyssLiveSnapshot(uid); found {
			writeJSON(w, snapshot)
			return
		}
		writeJSON(w, map[string]any{"ok": false, "error": "no active combat"})
		return
	}
	writeJSON(w, c.snapshotFor(uid))
}

func (s *WebServer) handleAbyssCombatAction(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	c, ok := s.liveCombatForUID(uid)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "no active combat"})
		return
	}
	var action abyssLiveAction
	if err := readJSON(r, &action); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if err := c.submit(uid, action); err != nil {
		if errors.Is(err, errAbyssLiveStale) {
			writeJSON(w, map[string]any{"ok": false, "error": "round closed", "state": c.snapshotFor(uid)})
			return
		}
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	c.persist()
	writeJSON(w, c.snapshotFor(uid))
}

func (s *WebServer) handleAbyssCombatTactics(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	c, ok := s.liveCombatForUID(uid)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "no active combat"})
		return
	}
	var req struct {
		Tactic string `json:"tactic"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if err := c.setTactic(uid, req.Tactic); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, c.snapshotFor(uid))
}

func (s *WebServer) handleAbyssCombatEvents(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	c, ok := s.liveCombatForUID(uid)
	if !ok {
		http.Error(w, "no active combat", http.StatusNotFound)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	var lastVersion int64 = -1
	for {
		snapshot := c.snapshotFor(uid)
		if snapshot.Version != lastVersion {
			data, err := json.Marshal(snapshot)
			if err != nil {
				return
			}
			if _, err := fmt.Fprintf(w, "event: combat\ndata: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
			lastVersion = snapshot.Version
			if snapshot.Phase == "complete" || snapshot.Phase == "failed" {
				return
			}
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}
