package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const abyssEntrySetupKeyPrefix = "abyss_entry_setup_"

type abyssEntrySetup struct {
	Tier         string   `json:"tier"`
	Pacts        []string `json:"pacts"`
	Start        string   `json:"start"`
	Checkpoint   int      `json:"checkpoint"`
	Kit          string   `json:"kit"`
	Mutation     string   `json:"mutation"`
	LootRule     string   `json:"loot_rule"`
	VeteranTrack string   `json:"veteran_track"`
	Focus        string   `json:"focus"`
	Expedition   bool     `json:"expedition"`
	Hardcore     bool     `json:"hardcore"`
	Hybrid       bool     `json:"hybrid"`
	Contract     string   `json:"contract"`
}

type abyssSetupYesterday struct {
	Runs       int   `json:"runs"`
	Wins       int   `json:"wins"`
	Deaths     int   `json:"deaths"`
	GoldBanked int64 `json:"gold_banked"`
	BestDepth  int   `json:"best_depth"`
}

type abyssSetupState struct {
	TierBests          map[string]int      `json:"tier_bests"`
	FloorOneRiskByTier map[string]int      `json:"floor_one_risk_by_tier"`
	FreeEntryAvailable bool                `json:"free_entry_available"`
	Jackpot            int64               `json:"jackpot"`
	Yesterday          abyssSetupYesterday `json:"yesterday"`
	LastSetup          *abyssEntrySetup    `json:"last_setup,omitempty"`
}

func abyssEntrySetupKey(uid string) string { return abyssEntrySetupKeyPrefix + uid }

func normalizeAbyssEntryFocus(value string) (string, int64, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "auto" {
		return "auto", 0, true
	}
	id, ok := abyssFocusIDs[value]
	return value, id, ok
}

func canonicalAbyssEntrySetup(setup abyssEntrySetup) abyssEntrySetup {
	if _, ok := abyssTierByKey(setup.Tier); !ok {
		setup.Tier = "normal"
	}
	if _, hasNextTier := abyssNextTier(setup.Tier); !hasNextTier {
		setup.Hybrid = false
	}
	setup.Pacts = canonicalAbyssPactRequest(setup.Pacts)
	setup.Contract, _ = normalizeAbyssContractPact(setup.Contract)
	switch setup.Start {
	case "checkpoint":
		if setup.Checkpoint < 10 || setup.Checkpoint%10 != 0 {
			setup.Start = ""
			setup.Checkpoint = 0
		}
	case "express":
		setup.Checkpoint = 0
	default:
		setup.Start = ""
		setup.Checkpoint = 0
	}
	setup.Kit = normalizeAbyssBuildKit(setup.Kit)
	setup.Mutation = normalizeAbyssSkillMutation(setup.Mutation)
	setup.LootRule = normalizeAbyssPartyLootRule(setup.LootRule)
	setup.VeteranTrack, _ = normalizeAbyssVeteranTrack(setup.VeteranTrack)
	if focus, _, ok := normalizeAbyssEntryFocus(setup.Focus); ok {
		setup.Focus = focus
	} else {
		setup.Focus = "auto"
	}
	return setup
}

func saveAbyssEntrySetup(exec dbExecQuerier, uid string, setup abyssEntrySetup) error {
	data, err := json.Marshal(canonicalAbyssEntrySetup(setup))
	if err != nil {
		return fmt.Errorf("marshal Abyss entry setup: %w", err)
	}
	if _, err := exec.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssEntrySetupKey(uid), string(data)); err != nil {
		return fmt.Errorf("save Abyss entry setup: %w", err)
	}
	return nil
}

func (b *Bot) loadAbyssEntrySetup(uid string) *abyssEntrySetup {
	var stored string
	if err := b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssEntrySetupKey(uid)).Scan(&stored); err != nil {
		return nil
	}
	var setup abyssEntrySetup
	if json.Unmarshal([]byte(stored), &setup) != nil {
		return nil
	}
	setup = canonicalAbyssEntrySetup(setup)
	return &setup
}

func (b *Bot) loadAbyssSetupState(uid string, playerCR float64) (abyssSetupState, error) {
	state := abyssSetupState{
		TierBests:          make(map[string]int, len(abyssTierOrder)),
		FloorOneRiskByTier: make(map[string]int, len(abyssTierOrder)),
		Jackpot:            b.getJackpot("abyss"),
	}
	for _, key := range abyssTierOrder {
		state.TierBests[key] = 0
		if tier, ok := abyssTierByKey(key); ok {
			state.FloorOneRiskByTier[key] = abyssRiskPct(1, tier, playerCR)
		}
	}
	rows, err := b.DB.Query(`SELECT COALESCE(tier, 'normal'), COALESCE(MAX(depth), 0)
		FROM abyss_runs WHERE client_uid=$1 GROUP BY COALESCE(tier, 'normal')`, uid)
	if err != nil {
		return state, fmt.Errorf("load Abyss tier bests: %w", err)
	}
	for rows.Next() {
		var tier string
		var depth int
		if err := rows.Scan(&tier, &depth); err != nil {
			_ = rows.Close()
			return state, fmt.Errorf("scan Abyss tier best: %w", err)
		}
		if _, known := state.TierBests[tier]; known {
			state.TierBests[tier] = max(depth, 0)
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return state, fmt.Errorf("iterate Abyss tier bests: %w", err)
	}
	_ = rows.Close()

	if err := b.DB.QueryRow(`SELECT COUNT(*), COUNT(*) FILTER (WHERE victory),
		COUNT(*) FILTER (WHERE NOT victory), COALESCE(SUM(gold_banked) FILTER (WHERE victory), 0), COALESCE(MAX(depth), 0)
		FROM abyss_runs WHERE client_uid=$1
		AND created_at >= CURRENT_DATE - INTERVAL '1 day' AND created_at < CURRENT_DATE`, uid).Scan(
		&state.Yesterday.Runs, &state.Yesterday.Wins, &state.Yesterday.Deaths,
		&state.Yesterday.GoldBanked, &state.Yesterday.BestDepth,
	); err != nil {
		return state, fmt.Errorf("load yesterday's Abyss summary: %w", err)
	}
	if err := b.DB.QueryRow(`SELECT abyss_free_entry_date IS NULL OR abyss_free_entry_date < CURRENT_DATE
		FROM users WHERE client_uid=$1`, uid).Scan(&state.FreeEntryAvailable); err != nil {
		return state, fmt.Errorf("load Abyss free-entry state: %w", err)
	}
	state.LastSetup = b.loadAbyssEntrySetup(uid)
	return state, nil
}

func (s *WebServer) handleAbyssSetupState(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	state, err := s.bot.loadAbyssSetupState(uid, s.bot.abyssPlayerCR(uid))
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "state": state})
}
