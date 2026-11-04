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
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
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

	// Every retained reference is remapped through one table so spectators
	// see consistent anonymous identifiers. Combatant IDs and entity IDs embed
	// persistent account UIDs (for example "ally:<uid>" or "pet:<uid>..."),
	// as do presentation-event actors and targets; anything not explicitly
	// mapped receives a fresh anonymous identifier rather than passing through.
	// Combatant IDs and presentation entity IDs are separate namespaces
	// ("enemy:0" can be a combatant ID while "enemy:1" is another mob's
	// entity ID), so each has its own table; lookups fall back to the other
	// namespace and finally to a fresh anonymous identifier.
	ids := make(map[string]string, len(snapshot.Allies)+len(snapshot.Enemies))
	entities := make(map[string]string, len(snapshot.Allies)+len(snapshot.Enemies))
	unknown := map[string]string{}
	fresh := func(id string) string {
		if anonymous, ok := unknown[id]; ok {
			return anonymous
		}
		anonymous := fmt.Sprintf("entity:%d", len(unknown)+1)
		unknown[id] = anonymous
		return anonymous
	}
	lookup := func(id string, primary, secondary map[string]string) string {
		if id == "" {
			return ""
		}
		if anonymous, ok := primary[id]; ok {
			return anonymous
		}
		if anonymous, ok := secondary[id]; ok {
			return anonymous
		}
		return fresh(id)
	}
	remap := func(id string) string { return lookup(id, ids, entities) }
	remapEntity := func(id string) string { return lookup(id, entities, ids) }
	snapshot.Allies = append([]abyssLiveCombatantView(nil), snapshot.Allies...)
	for i := range snapshot.Allies {
		anonymous := fmt.Sprintf("ally:%d", i+1)
		ids[snapshot.Allies[i].ID] = anonymous
		if entity := snapshot.Allies[i].EntityID; entity != "" {
			if strings.HasPrefix(entity, "pet:") {
				entities[entity] = fmt.Sprintf("pet:%d", i+1)
			} else {
				entities[entity] = anonymous
			}
		}
	}
	snapshot.Enemies = append([]abyssLiveCombatantView(nil), snapshot.Enemies...)
	for i := range snapshot.Enemies {
		anonymous := fmt.Sprintf("enemy:%d", i+1)
		ids[snapshot.Enemies[i].ID] = anonymous
		if entity := snapshot.Enemies[i].EntityID; entity != "" {
			entities[entity] = anonymous
		}
	}
	for i := range snapshot.Allies {
		snapshot.Allies[i].ID = ids[snapshot.Allies[i].ID]
		snapshot.Allies[i].EntityID = remapEntity(snapshot.Allies[i].EntityID)
		snapshot.Allies[i].IsSelf = false
		snapshot.Allies[i].Ready = false
	}
	for i := range snapshot.Enemies {
		snapshot.Enemies[i].ID = ids[snapshot.Enemies[i].ID]
		snapshot.Enemies[i].EntityID = remapEntity(snapshot.Enemies[i].EntityID)
	}
	snapshot.Initiative = append([]abyssLiveInitiativeEntry(nil), snapshot.Initiative...)
	for i := range snapshot.Initiative {
		snapshot.Initiative[i].ID = remap(snapshot.Initiative[i].ID)
	}
	snapshot.EnemyIntents = append([]abyssLiveEnemyIntent(nil), snapshot.EnemyIntents...)
	for i := range snapshot.EnemyIntents {
		snapshot.EnemyIntents[i].EnemyID = remap(snapshot.EnemyIntents[i].EnemyID)
		snapshot.EnemyIntents[i].TargetID = remap(snapshot.EnemyIntents[i].TargetID)
	}
	events := make([]abyssLivePresentationEvent, len(snapshot.PresentationEvents))
	for i, event := range snapshot.PresentationEvents {
		event.ActorID = remapEntity(event.ActorID)
		event.Targets = append([]abyssLivePresentationOutcome(nil), event.Targets...)
		for j := range event.Targets {
			event.Targets[j].TargetID = remapEntity(event.Targets[j].TargetID)
		}
		events[i] = event
	}
	if snapshot.PresentationEvents != nil {
		snapshot.PresentationEvents = events
	}
	return snapshot
}
