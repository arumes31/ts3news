package bot

import (
	"fmt"
	"net/http"
	"strings"
)

func (s *WebServer) handleAbyssSpectatePage(w http.ResponseWriter, r *http.Request, _ string) {
	sessionID := strings.TrimSpace(r.URL.Query().Get("session"))
	if !validAbyssSpectatorSessionID(sessionID) {
		http.Error(w, "invalid spectator link", http.StatusBadRequest)
		return
	}
	s.render(w, "abyssSpectate", map[string]any{"Title": "Abyss Spectator", "Nav": "abyss", "SessionID": sessionID})
}

func validAbyssSpectatorSessionID(sessionID string) bool {
	if len(sessionID) != 32 {
		return false
	}
	for _, char := range sessionID {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}

func (s *WebServer) handleAbyssSpectateState(w http.ResponseWriter, r *http.Request, _ string) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	sessionID := strings.TrimSpace(r.URL.Query().Get("session"))
	if !validAbyssSpectatorSessionID(sessionID) {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid spectator link"})
		return
	}
	value, ok := s.liveCombats.Load(sessionID)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "combat is no longer live"})
		return
	}
	combat, ok := value.(*abyssLiveCombat)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "combat is unavailable"})
		return
	}
	writeJSON(w, sanitizeAbyssSpectatorSnapshot(combat.snapshotFor(combat.ownerUID)))
}

func sanitizeAbyssSpectatorSnapshot(snapshot abyssLiveSnapshot) abyssLiveSnapshot {
	snapshot.OwnerUID = ""
	snapshot.Options = nil
	snapshot.Queued = nil
	snapshot.Recommended = nil
	snapshot.Policy = abyssLivePolicy{}
	snapshot.CanConfigure = false
	snapshot.TimeBankMS = 0
	snapshot.RandomSeed = [2]uint64{}
	snapshot.RandomDraws = 0
	snapshot.Result = nil
	snapshot.Social = abyssLiveSocialSnapshot{Members: snapshot.Social.Members}

	ids := make(map[string]string, len(snapshot.Allies)+len(snapshot.Enemies))
	for i := range snapshot.Allies {
		anonymous := fmt.Sprintf("ally:%d", i+1)
		ids[snapshot.Allies[i].ID] = anonymous
		snapshot.Allies[i].ID = anonymous
		snapshot.Allies[i].IsSelf = false
		snapshot.Allies[i].Ready = false
	}
	for i := range snapshot.Enemies {
		anonymous := fmt.Sprintf("enemy:%d", i+1)
		ids[snapshot.Enemies[i].ID] = anonymous
		snapshot.Enemies[i].ID = anonymous
	}
	for i := range snapshot.Initiative {
		snapshot.Initiative[i].ID = ids[snapshot.Initiative[i].ID]
	}
	for i := range snapshot.EnemyIntents {
		snapshot.EnemyIntents[i].EnemyID = ids[snapshot.EnemyIntents[i].EnemyID]
		snapshot.EnemyIntents[i].TargetID = ids[snapshot.EnemyIntents[i].TargetID]
	}
	return snapshot
}
