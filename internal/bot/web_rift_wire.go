package bot

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"ts3news/internal/rift"
)

// riftRetainedFields contains entire sections replaced whenever their content
// changes. Dynamic omitted fields are never recovered from an older snapshot.
func riftRetainedFields(r *rift.Run) map[string]any {
	return map[string]any{
		"build": r.Build, "encounter_plan": r.EncounterPlan, "level": r.Level,
		"mission_history": r.History, "region_versions": r.RegionVersions, "region_records": r.RegionRecords,
		"objective_history": r.ObjectiveHistory, "completed_levels": r.CompletedLevels, "attempt_history": r.AttemptHistory,
		"banked_items": r.BankedItems, "banked_loot": r.BankedLoot, "past_expeditions": r.PastExpeditions,
		"room_baseline": r.RoomBaseline, "last_encounter": r.LastEncounter, "last_objectives": r.LastObjectives,
	}
}

// The nil shadow fields suppress only the explicitly retained sections. Run is
// read-only here: database persistence always keeps the complete canonical state.
type riftLeanRun struct {
	*rift.Run
	Build            any `json:"build,omitempty"`
	EncounterPlan    any `json:"encounter_plan,omitempty"`
	Level            any `json:"level,omitempty"`
	History          any `json:"mission_history,omitempty"`
	RegionVersions   any `json:"region_versions,omitempty"`
	RegionRecords    any `json:"region_records,omitempty"`
	ObjectiveHistory any `json:"objective_history,omitempty"`
	CompletedLevels  any `json:"completed_levels,omitempty"`
	AttemptHistory   any `json:"attempt_history,omitempty"`
	BankedItems      any `json:"banked_items,omitempty"`
	BankedLoot       any `json:"banked_loot,omitempty"`
	PastExpeditions  any `json:"past_expeditions,omitempty"`
	RoomBaseline     any `json:"room_baseline,omitempty"`
	LastEncounter    any `json:"last_encounter,omitempty"`
	LastObjectives   any `json:"last_objectives,omitempty"`
}

func riftWireSnapshot(run *rift.Run, action, base string) (map[string]any, error) {
	response := map[string]any{"ok": true, "run": run, "snapshot_kind": "full", "snapshot_base": ""}
	if run == nil {
		return response, nil
	}
	fields := riftRetainedFields(run)
	encoded, err := json.Marshal(struct {
		ID     string
		Fields map[string]any
	}{run.ID, fields})
	if err != nil {
		return nil, err
	}
	token := fmt.Sprintf("%x", sha256.Sum256(encoded))
	response["snapshot_base"] = token
	if action == "step" && run.Status == "fighting" && base == token {
		response["snapshot_kind"] = "lean-v1"
		response["run"] = riftLeanRun{Run: run}
	}
	return response, nil
}

func riftHTTPResponse(r *http.Request, run *rift.Run, action string) (map[string]any, error) {
	if r.Header.Get("X-Rift-Snapshot") != "lean-v1" {
		return map[string]any{"ok": true, "run": run}, nil
	}
	return riftWireSnapshot(run, action, r.Header.Get("X-Rift-Snapshot-Base"))
}
func writeRiftSnapshot(w http.ResponseWriter, r *http.Request, action string, run *rift.Run) {
	response, err := riftHTTPResponse(r, run, action)
	if err != nil {
		riftFailure(w, r, err)
		return
	}
	writeJSON(w, response)
}
