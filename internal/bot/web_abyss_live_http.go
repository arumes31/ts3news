package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
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
	c.touchMember(uid)
	writeJSON(w, c.snapshotFor(uid))
}

func (s *WebServer) handleAbyssCombatAction(w http.ResponseWriter, r *http.Request, uid string) {
	started := time.Now()
	defer func() { s.abyssOps.observeRequest(time.Since(started)) }()
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
	if err := c.persist(); err != nil {
		log.Printf("abyss live: persisting submitted action: %v", err)
		writeJSON(w, map[string]any{"ok": false, "error": "action persistence unavailable"})
		return
	}
	writeJSON(w, c.snapshotFor(uid))
}

func (s *WebServer) handleAbyssCombatReady(w http.ResponseWriter, r *http.Request, uid string) {
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
		SessionID string `json:"session_id"`
		Round     int    `json:"round"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if err := c.setReady(uid, req.SessionID, req.Round); err != nil {
		if errors.Is(err, errAbyssLiveStale) {
			writeJSON(w, map[string]any{"ok": false, "error": "round closed", "state": c.snapshotFor(uid)})
			return
		}
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if err := c.persist(); err != nil {
		log.Printf("abyss live: persisting ready state: %v", err)
		writeJSON(w, map[string]any{"ok": false, "error": "ready persistence unavailable"})
		return
	}
	c.releaseReadyRound()
	writeJSON(w, c.snapshotFor(uid))
}

func (s *WebServer) handleAbyssCombatTimeBank(w http.ResponseWriter, r *http.Request, uid string) {
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
		SessionID string `json:"session_id"`
		Round     int    `json:"round"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if err := c.spendTimeBank(uid, req.SessionID, req.Round); err != nil {
		if errors.Is(err, errAbyssLiveStale) {
			writeJSON(w, map[string]any{"ok": false, "error": "round closed", "state": c.snapshotFor(uid)})
			return
		}
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if err := c.persist(); err != nil {
		log.Printf("abyss live: persisting time-bank spend: %v", err)
		writeJSON(w, map[string]any{"ok": false, "error": "time-bank persistence unavailable"})
		return
	}
	writeJSON(w, c.snapshotFor(uid))
}

func (s *WebServer) handleAbyssCombatPauseMode(w http.ResponseWriter, r *http.Request, uid string) {
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
		Mode string `json:"mode"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if err := c.setPauseMode(uid, req.Mode); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if err := c.persist(); err != nil {
		log.Printf("abyss live: persisting pause mode: %v", err)
		writeJSON(w, map[string]any{"ok": false, "error": "pause configuration unavailable"})
		return
	}
	writeJSON(w, c.snapshotFor(uid))
}

func (s *WebServer) handleAbyssCombatPolicy(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	c, ok := s.liveCombatForUID(uid)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "no active combat"})
		return
	}
	var policy abyssLivePolicy
	if err := readJSON(r, &policy); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if err := c.setPolicy(uid, policy); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if err := c.persist(); err != nil {
		log.Printf("abyss live: persisting combat policy: %v", err)
		writeJSON(w, map[string]any{"ok": false, "error": "policy persistence unavailable"})
		return
	}
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
	s.abyssOps.sseConnections.Add(1)
	defer s.abyssOps.sseConnections.Add(-1)
	lastEventID, resume, err := abyssLiveLastEventID(r)
	if err != nil {
		http.Error(w, "invalid Last-Event-ID", http.StatusBadRequest)
		return
	}
	currentSnapshot := c.snapshotFor(uid)
	if resume && lastEventID > currentSnapshot.Version {
		http.Error(w, "Last-Event-ID is ahead of combat", http.StatusBadRequest)
		return
	}
	if !c.openMemberConnection(uid, time.Now()) {
		http.Error(w, "not a combat participant", http.StatusForbidden)
		return
	}
	defer func() {
		if c.closeMemberConnection(uid, time.Now()) {
			c.persistOrLog("granting connectivity grace")
		}
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	if !resume {
		if err := writeAbyssLiveEvent(w, flusher, currentSnapshot); err != nil {
			return
		}
		lastEventID = currentSnapshot.Version
		if currentSnapshot.Phase == "complete" || currentSnapshot.Phase == "failed" {
			return
		}
	}
	for {
		c.touchMember(uid)
		for _, snapshot := range c.eventsAfter(uid, lastEventID) {
			if err := writeAbyssLiveEvent(w, flusher, snapshot); err != nil {
				return
			}
			lastEventID = snapshot.Version
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

func abyssLiveLastEventID(r *http.Request) (int64, bool, error) {
	raw := strings.TrimSpace(r.Header.Get("Last-Event-ID"))
	if raw == "" {
		raw = strings.TrimSpace(r.URL.Query().Get("since"))
	}
	if raw == "" {
		return -1, false, nil
	}
	eventID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || eventID < 0 {
		return 0, false, fmt.Errorf("invalid event id")
	}
	return eventID, true, nil
}

func writeAbyssLiveEvent(w http.ResponseWriter, flusher http.Flusher, snapshot abyssLiveSnapshot) error {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("encoding live combat event: %w", err)
	}
	if _, err := fmt.Fprintf(w, "id: %d\nevent: combat\ndata: %s\n\n", snapshot.Version, data); err != nil {
		return fmt.Errorf("writing live combat event: %w", err)
	}
	flusher.Flush()
	return nil
}
