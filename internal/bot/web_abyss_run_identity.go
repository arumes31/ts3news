package bot

import (
	"errors"
	"io"
	"net/http"
)

const abyssRunFlagBiomeSelectedAt = "biome_selected_at"

type abyssStoryBeatView struct {
	Depth    int    `json:"depth"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Complete bool   `json:"complete"`
	Current  bool   `json:"current"`
}

type abyssRunIdentityView struct {
	Active              bool                 `json:"active"`
	Story               bool                 `json:"story"`
	StoryComplete       bool                 `json:"story_complete"`
	StoryProgress       int                  `json:"story_progress"`
	StoryBeats          []abyssStoryBeatView `json:"story_beats"`
	Biome               *abyssBiomeContract  `json:"biome,omitempty"`
	BiomeUntil          int                  `json:"biome_until"`
	BiomeChoiceRequired bool                 `json:"biome_choice_required"`
	BiomeChoices        []abyssBiomeContract `json:"biome_choices"`
	Relics              []abyssRunRelic      `json:"relics"`
	Boons               []abyssRunBoon       `json:"boons"`
	Draft               abyssBoonDraftView   `json:"draft"`
	NextRelicDepth      int                  `json:"next_relic_depth"`
}

func abyssRunIdentityViewFrom(run abyssRun, flags map[string]int64) abyssRunIdentityView {
	view := abyssRunIdentityView{
		Active:         run.Active,
		Story:          flags[abyssRunFlagStoryCampaign] == 1,
		StoryComplete:  flags[abyssRunFlagStoryComplete] == 1,
		StoryBeats:     []abyssStoryBeatView{},
		BiomeChoices:   append([]abyssBiomeContract(nil), abyssBiomeContracts...),
		Relics:         abyssRunRelicViews(flags),
		Boons:          abyssRunBoonViews(flags),
		Draft:          abyssBoonDraftFromFlags(flags),
		NextRelicDepth: ((max(run.Depth, 0) / 4) + 1) * 4,
	}
	view.StoryProgress = min(max(run.Depth, 0), len(abyssStoryCampaign))
	for _, beat := range abyssStoryCampaign {
		view.StoryBeats = append(view.StoryBeats, abyssStoryBeatView{
			Depth: beat.Depth, Title: beat.Title, Subtitle: beat.Subtitle,
			Complete: run.Depth >= beat.Depth, Current: run.Depth+1 == beat.Depth,
		})
	}
	if contract, ok := abyssSelectedBiomeContract(flags, run.Depth+1); ok {
		copy := contract
		view.Biome = &copy
		view.BiomeUntil = int(flags[abyssRunFlagBiomeUntil])
	}
	view.BiomeChoiceRequired = run.Active && run.FloorType == "rest" &&
		flags[abyssRunFlagBiomeSelectedAt] != int64(run.Depth)
	return view
}

func (b *Bot) abyssRunIdentity(uid string, run abyssRun) abyssRunIdentityView {
	return abyssRunIdentityViewFrom(run, b.loadRunFlags(uid))
}

func (s *WebServer) handleAbyssBiomeContract(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	var request struct {
		ID int64 `json:"id"`
	}
	if err := readJSON(r, &request); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	contract, ok := abyssBiomeContractByID(request.ID)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown biome contract"})
		return
	}
	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.FloorType != "rest" {
		writeJSON(w, map[string]any{"ok": false, "error": "biome contracts are chosen at rest floors"})
		return
	}
	flags := s.bot.loadRunFlags(uid)
	flags[abyssRunFlagBiomeChoice] = contract.ID
	flags[abyssRunFlagBiomeUntil] = int64(run.Depth + 5)
	flags[abyssRunFlagBiomeSelectedAt] = int64(run.Depth)
	if err := s.bot.saveRunFlags(uid, flags); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "could not bind the biome contract"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "name": contract.Name, "until": run.Depth + 5,
		"run_identity": abyssRunIdentityViewFrom(run, flags),
	})
}

func (s *WebServer) handleAbyssBoonDraft(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	var request struct {
		ID int64 `json:"id"`
	}
	if err := readJSON(r, &request); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "no active run can receive a boon"})
		return
	}
	flags := s.bot.loadRunFlags(uid)
	draft := abyssBoonDraftFromFlags(flags)
	if !draft.Pending {
		writeJSON(w, map[string]any{"ok": false, "error": "no boon draft is pending"})
		return
	}
	var selected abyssRunBoon
	valid := false
	for _, option := range draft.Options {
		if option.ID == request.ID {
			selected = option
			valid = true
			break
		}
	}
	if !valid {
		writeJSON(w, map[string]any{"ok": false, "error": "choose one of the offered boons"})
		return
	}
	flag := abyssRunBoonFlag(selected.ID)
	flags[flag] = min(flags[flag]+1, int64(3))
	flags[abyssRunFlagBoonDraftDepth] = 0
	if err := s.bot.saveRunFlags(uid, flags); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "could not bind the boon"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "name": selected.Name, "stacks": flags[flag],
		"run_identity": abyssRunIdentityViewFrom(run, flags),
	})
}
