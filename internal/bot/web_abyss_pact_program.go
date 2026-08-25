package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	abyssPactPresetSlots   = 3
	abyssPactPresetNameMax = 32
	abyssPactMasteryRuns   = 10
)

type abyssPactPreset struct {
	Slot  int      `json:"slot"`
	Name  string   `json:"name"`
	Pacts []string `json:"pacts"`
}

type abyssPactMasteryView struct {
	Key               string  `json:"key"`
	Label             string  `json:"label"`
	Runs              int     `json:"runs"`
	RequiredRuns      int     `json:"required_runs"`
	ProgressPct       int     `json:"progress_pct"`
	Mastered          bool    `json:"mastered"`
	BaseBonusPct      int     `json:"base_bonus_pct"`
	EffectiveBonusPct float64 `json:"effective_bonus_pct"`
}

type abyssPactProgramState struct {
	Presets       []abyssPactPreset       `json:"presets"`
	Mastery       []abyssPactMasteryView  `json:"mastery"`
	MasteredCount int                     `json:"mastered_count"`
	Calendar      []abyssAffixCalendarDay `json:"calendar"`
	Featured      abyssPactFeaturedView   `json:"featured"`
	Synergies     []abyssPactSynergy      `json:"synergies"`
	WeekendPoll   abyssWeekendAffixPoll   `json:"weekend_poll"`
}

func abyssPactPresetsKey(uid string) string { return "abyss_pact_presets_" + uid }
func abyssPactMasteryKey(uid string) string { return "abyss_pact_mastery_" + uid }

func canonicalAbyssPactPreset(slot int, name string, pacts []string) (abyssPactPreset, bool) {
	if slot < 1 || slot > abyssPactPresetSlots {
		return abyssPactPreset{}, false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = fmt.Sprintf("Preset %d", slot)
	}
	if runes := []rune(name); len(runes) > abyssPactPresetNameMax {
		name = string(runes[:abyssPactPresetNameMax])
	}
	return abyssPactPreset{
		Slot:  slot,
		Name:  name,
		Pacts: canonicalAbyssPactRequest(pacts),
	}, true
}

func decodeAbyssPactPresets(stored string) []abyssPactPreset {
	var decoded []abyssPactPreset
	if json.Unmarshal([]byte(stored), &decoded) != nil {
		return nil
	}
	bySlot := make(map[int]abyssPactPreset, abyssPactPresetSlots)
	for _, preset := range decoded {
		if canonical, ok := canonicalAbyssPactPreset(preset.Slot, preset.Name, preset.Pacts); ok {
			bySlot[canonical.Slot] = canonical
		}
	}
	presets := make([]abyssPactPreset, 0, len(bySlot))
	for slot := 1; slot <= abyssPactPresetSlots; slot++ {
		if preset, ok := bySlot[slot]; ok {
			presets = append(presets, preset)
		}
	}
	return presets
}

func decodeAbyssPactMastery(stored string) map[string]int {
	decoded := make(map[string]int)
	if json.Unmarshal([]byte(stored), &decoded) != nil {
		return make(map[string]int)
	}
	mastery := make(map[string]int, len(decoded))
	for key, runs := range decoded {
		if _, known := abyssPactByKey(key); known && runs > 0 {
			mastery[key] = min(runs, 1_000_000)
		}
	}
	return mastery
}

func (b *Bot) loadAbyssPactPresets(uid string) ([]abyssPactPreset, error) {
	var stored string
	err := b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssPactPresetsKey(uid)).Scan(&stored)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load Abyss pact presets: %w", err)
	}
	return decodeAbyssPactPresets(stored), nil
}

func saveAbyssPactPresets(exec dbExecQuerier, uid string, presets []abyssPactPreset) error {
	payload, err := json.Marshal(presets)
	if err != nil {
		return fmt.Errorf("marshal Abyss pact presets: %w", err)
	}
	if _, err := exec.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssPactPresetsKey(uid), string(payload)); err != nil {
		return fmt.Errorf("save Abyss pact presets: %w", err)
	}
	return nil
}

func (b *Bot) loadAbyssPactMastery(uid string) (map[string]int, error) {
	var stored string
	err := b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssPactMasteryKey(uid)).Scan(&stored)
	if errors.Is(err, sql.ErrNoRows) {
		return make(map[string]int), nil
	}
	if err != nil {
		return nil, fmt.Errorf("load Abyss pact mastery: %w", err)
	}
	return decodeAbyssPactMastery(stored), nil
}

func advanceAbyssPactMastery(mastery map[string]int, pacts []string) map[string]int {
	if mastery == nil {
		mastery = make(map[string]int)
	}
	seen := make(map[string]bool, len(pacts))
	for _, key := range pacts {
		if _, known := abyssPactByKey(key); !known || seen[key] {
			continue
		}
		seen[key] = true
		mastery[key] = min(mastery[key]+1, 1_000_000)
	}
	return mastery
}

func incrementAbyssPactMastery(tx *sql.Tx, uid string, pacts []string) error {
	canonical := strings.Fields(abyssValidatePacts(pacts))
	if len(canonical) == 0 {
		return nil
	}
	var stored string
	err := tx.QueryRow("SELECT value FROM app_meta WHERE key=$1 FOR UPDATE", abyssPactMasteryKey(uid)).Scan(&stored)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("lock Abyss pact mastery: %w", err)
	}
	mastery := advanceAbyssPactMastery(decodeAbyssPactMastery(stored), canonical)
	payload, err := json.Marshal(mastery)
	if err != nil {
		return fmt.Errorf("marshal Abyss pact mastery: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssPactMasteryKey(uid), string(payload)); err != nil {
		return fmt.Errorf("save Abyss pact mastery: %w", err)
	}
	return nil
}

func abyssPactRewardMultWithMastery(pacts []string, mastery map[string]int) float64 {
	mult := 1.0
	for _, key := range pacts {
		pact, ok := abyssPactByKey(key)
		if !ok {
			continue
		}
		bonus := pact.Reward
		if mastery[key] >= abyssPactMasteryRuns {
			bonus *= 1.05
		}
		mult += bonus
	}
	return mult
}

func abyssPactProgramStateFrom(presets []abyssPactPreset, mastery map[string]int) abyssPactProgramState {
	return abyssPactProgramStateFromAt(presets, mastery, time.Now().UTC())
}

func abyssPactProgramStateFromAt(presets []abyssPactPreset, mastery map[string]int, at time.Time) abyssPactProgramState {
	state := abyssPactProgramState{
		Presets: presets, Mastery: make([]abyssPactMasteryView, 0, len(abyssPactCatalog)),
		Calendar: abyssAffixCalendar(at), Featured: abyssFeaturedPactAt(at), Synergies: abyssPactSynergyCatalog,
		WeekendPoll: abyssWeekendAffixPollBase(at),
	}
	for _, pact := range abyssPactCatalog {
		runs := max(0, mastery[pact.Key])
		mastered := runs >= abyssPactMasteryRuns
		progress := min(runs, abyssPactMasteryRuns) * 100 / abyssPactMasteryRuns
		effective := pact.Reward
		if mastered {
			effective *= 1.05
			state.MasteredCount++
		}
		state.Mastery = append(state.Mastery, abyssPactMasteryView{
			Key: pact.Key, Label: pact.Label, Runs: runs, RequiredRuns: abyssPactMasteryRuns,
			ProgressPct: progress, Mastered: mastered, BaseBonusPct: int(pact.Reward * 100),
			EffectiveBonusPct: float64(int(effective*1000+0.5)) / 10,
		})
	}
	return state
}

func (b *Bot) abyssPactProgramState(uid string) abyssPactProgramState {
	at := time.Now().UTC()
	presets, presetErr := b.loadAbyssPactPresets(uid)
	mastery, masteryErr := b.loadAbyssPactMastery(uid)
	if presetErr != nil {
		presets = nil
	}
	if masteryErr != nil {
		mastery = make(map[string]int)
	}
	return b.enrichAbyssPactProgramState(abyssPactProgramStateFromAt(presets, mastery, at), uid, at)
}

func (b *Bot) enrichAbyssPactProgramState(state abyssPactProgramState, uid string, at time.Time) abyssPactProgramState {
	poll, err := b.abyssWeekendAffixPoll(uid, at)
	if err != nil {
		return state
	}
	state.WeekendPoll = poll
	state.Calendar = applyAbyssWeekendAffix(state.Calendar, poll.Winner)
	return state
}

func (s *WebServer) handleAbyssPactPresets(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Action string   `json:"action"`
		Slot   int      `json:"slot"`
		Name   string   `json:"name"`
		Pacts  []string `json:"pacts"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	presets, err := s.bot.loadAbyssPactPresets(uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	mastery, err := s.bot.loadAbyssPactMastery(uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	bySlot := make(map[int]abyssPactPreset, abyssPactPresetSlots)
	for _, preset := range presets {
		bySlot[preset.Slot] = preset
	}
	switch req.Action {
	case "save":
		preset, ok := canonicalAbyssPactPreset(req.Slot, req.Name, req.Pacts)
		if !ok || len(preset.Pacts) == 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "choose a preset slot and at least one pact"})
			return
		}
		bySlot[preset.Slot] = preset
	case "delete":
		if req.Slot < 1 || req.Slot > abyssPactPresetSlots {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid preset slot"})
			return
		}
		delete(bySlot, req.Slot)
	default:
		writeJSON(w, map[string]any{"ok": false, "error": "unknown preset action"})
		return
	}
	presets = presets[:0]
	for _, preset := range bySlot {
		presets = append(presets, preset)
	}
	sort.Slice(presets, func(i, j int) bool { return presets[i].Slot < presets[j].Slot })
	if err := saveAbyssPactPresets(s.bot.DB, uid, presets); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	state := s.bot.enrichAbyssPactProgramState(abyssPactProgramStateFrom(presets, mastery), uid, time.Now().UTC())
	writeJSON(w, map[string]any{"ok": true, "state": state})
}
