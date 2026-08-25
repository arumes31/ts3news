package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type abyssBossTollView struct {
	Available       bool
	TargetDepth     int
	Cost            int64
	Rolls           int
	Bosses          string
	ContractForfeit int64
}

func abyssBossTollExpectedValue(level, targetDepth int) (cost int64, rolls int) {
	return abyssBossTollExpectedValueForRolls(level, targetDepth, len(abyssBossNamesAtDepth(targetDepth)))
}

func abyssBossTollExpectedValueForRolls(level, targetDepth, rolls int) (cost int64, resolvedRolls int) {
	difficulty, _ := abyssDifficulty(targetDepth)
	forecast := abyssDropForecastData(max(1, difficulty), lootRarityScale(level))
	expected := forecast.Ultimate*100_000 + forecast.Title*60_000 + forecast.Unique*40_000 +
		forecast.Artifact*25_000 + forecast.Enchant*12_000 + forecast.Skill*8_000 +
		forecast.Consumable*2_000 + forecast.Gear*3_000 + forecast.Common*500
	rolls = max(1, rolls)
	cost = int64(expected * float64(rolls))
	cost = max(cost, int64(targetDepth*100))
	cost = (cost + 99) / 100 * 100
	return cost, rolls
}

func (b *Bot) abyssBossToll(uid string, run abyssRun, level int, chain abyssSecretBossChainView) abyssBossTollView {
	target := run.Depth + 1
	view := abyssBossTollView{TargetDepth: target}
	view.Available = run.Active && !run.Downed && target > 0 && target%abyssBossEvery == 0 && run.FloorType == "combat" && run.EventState == ""
	names := abyssBossNamesAtDepth(target)
	view.Bosses = strings.Join(names, " + ")
	rolls := len(names)
	if view.Available {
		if chain.Unlocked && !chain.Completed && chain.NextDepth == target && chain.Stage >= 0 && chain.Stage < len(abyssSecretBosses) {
			view.Bosses, rolls = abyssSecretBosses[chain.Stage].Name, 1
		}
		_ = b.DB.QueryRow(`SELECT boss_contract_wager FROM abyss_active
			WHERE client_uid=$1 AND boss_contract_depth=$2`, uid, target).Scan(&view.ContractForfeit)
	}
	view.Cost, view.Rolls = abyssBossTollExpectedValueForRolls(level, target, rolls)
	return view
}

func (s *WebServer) handleAbyssBossToll(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	var req struct {
		TargetDepth int   `json:"target_depth"`
		QuotedCost  int64 `json:"quoted_cost"`
	}
	if readJSON(r, &req) != nil || req.TargetDepth <= 0 || req.QuotedCost <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid boss toll quote"})
		return
	}
	_, _, secretToll := s.bot.abyssSecretBossForFloor(uid, req.TargetDepth, true)
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var depth, currentHP, level int
	var floorType string
	var eventState, pending sql.NullString
	var contractWager int64
	var contractDepth int
	err = tx.QueryRowContext(r.Context(), `SELECT active.depth,active.floor_type,active.event_state,active.pending_floor_choice,
		active.boss_contract_wager,active.boss_contract_depth,users.current_hp,users.level
		FROM abyss_active active JOIN users ON users.client_uid=active.client_uid
		WHERE active.client_uid=$1 FOR UPDATE OF active,users`, uid).
		Scan(&depth, &floorType, &eventState, &pending, &contractWager, &contractDepth, &currentHP, &level)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, map[string]any{"ok": false, "error": "not in a run"})
		return
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	target := depth + 1
	if currentHP <= 0 || floorType != "combat" || eventState.Valid || pending.Valid || target%abyssBossEvery != 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "boss toll is not available from this floor"})
		return
	}
	rolls := len(abyssBossNamesAtDepth(target))
	if secretToll && req.TargetDepth == target {
		rolls = 1
	}
	cost, rolls := abyssBossTollExpectedValueForRolls(level, target, rolls)
	if req.TargetDepth != target || req.QuotedCost != cost {
		writeJSON(w, map[string]any{"ok": false, "error": "boss toll quote changed; review the new price"})
		return
	}
	var gold int64
	err = tx.QueryRowContext(r.Context(), `UPDATE users SET gold=gold-$1,abyss_best_depth=GREATEST(abyss_best_depth,$2),abyss_win_streak=0
		WHERE client_uid=$3 AND gold>=$1 RETURNING gold`, cost, target, uid).Scan(&gold)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold for the boss toll"})
		return
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.ExecContext(r.Context(), `UPDATE abyss_active SET depth=$1,floor_type='combat',modifier='',event_state=NULL,
		pending_floor_choice=NULL,momentum=0,
		boss_contract_wager=CASE WHEN boss_contract_depth=$1 THEN 0 ELSE boss_contract_wager END,
		boss_contract_depth=CASE WHEN boss_contract_depth=$1 THEN 0 ELSE boss_contract_depth END,last_action_at=NOW()
		WHERE client_uid=$2`, target, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	flags := map[string]int64{}
	var rawFlags string
	err = tx.QueryRowContext(r.Context(), "SELECT value FROM app_meta WHERE key=$1 FOR UPDATE", abyssRunFlagsKey(uid)).Scan(&rawFlags)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if rawFlags != "" && json.Unmarshal([]byte(rawFlags), &flags) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	flags[abyssRunFlagPerfect] = 0
	flags[abyssRunFlagDeathWish] = 0
	flags[abyssRunFlagDefensiveMomentum] = 0
	if err := saveRunFlags(tx, uid, flags); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	contractForfeit := int64(0)
	if contractDepth == target {
		contractForfeit = contractWager
	}
	writeJSON(w, map[string]any{
		"ok": true, "msg": "The toll gate opens. No boss rewards were granted.",
		"depth": target, "gold": gold, "cost": cost, "loot_rolls_skipped": rolls,
		"contract_forfeit": contractForfeit,
	})
}
