package bot

import (
	"database/sql"
	"errors"
	"net/http"
)

const abyssBossContractPayoutMultiplier = int64(2)

var abyssBossContractWagers = map[int64]bool{1: true, 3: true, 5: true}

type abyssBossContractView struct {
	RunActive   bool
	Downed      bool
	Wager       int64
	TargetDepth int
	Payout      int64
}

func abyssNextNaturalBossDepth(depth int) int {
	return (max(0, depth)/abyssBossEvery + 1) * abyssBossEvery
}

func (b *Bot) abyssBossContract(uid string, run abyssRun) abyssBossContractView {
	view := abyssBossContractView{RunActive: run.Active, Downed: run.Downed}
	if !run.Active {
		return view
	}
	_ = b.DB.QueryRow(`SELECT boss_contract_wager,boss_contract_depth FROM abyss_active WHERE client_uid=$1`, uid).
		Scan(&view.Wager, &view.TargetDepth)
	if view.Wager > 0 {
		view.Payout = view.Wager * abyssBossContractPayoutMultiplier
	} else {
		view.TargetDepth = abyssNextNaturalBossDepth(run.Depth)
	}
	return view
}

func claimAbyssBossContractTx(tx *sql.Tx, uid string, depth int) (int64, error) {
	var wager int64
	var target int
	err := tx.QueryRow(`SELECT boss_contract_wager,boss_contract_depth FROM abyss_active
		WHERE client_uid=$1 FOR UPDATE`, uid).Scan(&wager, &target)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if wager <= 0 || target != depth {
		return 0, nil
	}
	if _, err := tx.Exec(`UPDATE abyss_active SET boss_contract_wager=0,boss_contract_depth=0
		WHERE client_uid=$1`, uid); err != nil {
		return 0, err
	}
	return wager * abyssBossContractPayoutMultiplier, nil
}

func (b *Bot) forfeitAbyssBossContract(uid string, depth int) int64 {
	var wager int64
	err := b.DB.QueryRow(`WITH contract AS (
		SELECT client_uid,boss_contract_wager FROM abyss_active
		WHERE client_uid=$1 AND boss_contract_depth=$2 AND boss_contract_wager>0 FOR UPDATE
	) UPDATE abyss_active active SET boss_contract_wager=0,boss_contract_depth=0
		FROM contract WHERE active.client_uid=contract.client_uid
		RETURNING contract.boss_contract_wager`, uid, depth).Scan(&wager)
	if err != nil {
		return 0
	}
	return wager
}

func (s *WebServer) handleAbyssBossContract(w http.ResponseWriter, r *http.Request, uid string) {
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
		Wager int64 `json:"wager"`
	}
	if readJSON(r, &req) != nil || !abyssBossContractWagers[req.Wager] {
		writeJSON(w, map[string]any{"ok": false, "error": "choose a 1, 3, or 5 Boss Token wager"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var depth, existingDepth, currentHP int
	var existingWager int64
	if err := tx.QueryRow(`SELECT active.depth,active.boss_contract_wager,active.boss_contract_depth,users.current_hp
		FROM abyss_active active JOIN users ON users.client_uid=active.client_uid
		WHERE active.client_uid=$1 FOR UPDATE OF active,users`, uid).Scan(&depth, &existingWager, &existingDepth, &currentHP); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "start a run before declaring a boss contract"})
		return
	}
	if currentHP <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "revive or concede before declaring a boss contract"})
		return
	}
	if existingWager > 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "a boss contract is already active"})
		return
	}
	var balance int64
	err = tx.QueryRow(`UPDATE users SET abyss_boss_tokens=abyss_boss_tokens-$1
		WHERE client_uid=$2 AND abyss_boss_tokens>=$1 RETURNING abyss_boss_tokens`, req.Wager, uid).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Boss Tokens"})
		return
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	target := abyssNextNaturalBossDepth(depth)
	if _, err := tx.Exec(`UPDATE abyss_active SET boss_contract_wager=$1,boss_contract_depth=$2,last_action_at=NOW()
		WHERE client_uid=$3`, req.Wager, target, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "msg": "Boss contract sealed.", "boss_tokens": balance,
		"wager": req.Wager, "target_depth": target, "payout": req.Wager * abyssBossContractPayoutMultiplier,
	})
}
