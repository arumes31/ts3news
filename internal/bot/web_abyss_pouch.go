package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
)

const abyssPouchMaxLevel = 3

var (
	errAbyssPouchMissing = errors.New("own a Consumable Pouch before tailoring it")
	errAbyssPouchMaxed   = errors.New("Consumable Pouch is already fully tailored")
	errAbyssPouchFunds   = errors.New("not enough gold")
)

var abyssPouchUpgradeCosts = [...]int64{250_000, 1_000_000, 5_000_000}

type abyssPouchView struct {
	Owned       bool
	Equipped    bool
	Level       int
	MaxLevel    int
	StackCap    int
	CarryCap    int
	NextStack   int
	NextCarry   int
	NextLevel   int
	NextCost    int64
	CanUpgrade  bool
	StatusLabel string
}

func abyssPouchCaps(level int) (stack, carry int) {
	level = min(max(level, 0), abyssPouchMaxLevel)
	return abyssConsumableStackCapBase + level, abyssCarryCapPouch + level
}

func (b *Bot) abyssPouchLevel(uid string) int {
	var level int
	if err := b.DB.QueryRow("SELECT level FROM abyss_consumable_pouches WHERE client_uid=$1", uid).Scan(&level); err != nil {
		return 0
	}
	return min(max(level, 0), abyssPouchMaxLevel)
}

func (b *Bot) abyssConsumableStackLimit(uid string) int {
	stack, _ := abyssPouchCaps(b.abyssPouchLevel(uid))
	return stack
}

func (b *Bot) abyssPouchProgress(uid string) abyssPouchView {
	view := abyssPouchView{MaxLevel: abyssPouchMaxLevel}
	_ = b.DB.QueryRow(`SELECT
		EXISTS(SELECT 1 FROM user_gear WHERE client_uid=$1 AND gear_id='ABYSS_POUCH'),
		EXISTS(SELECT 1 FROM user_inventory WHERE client_uid=$1 AND gear_id='ABYSS_POUCH')`, uid).
		Scan(&view.Equipped, &view.Owned)
	view.Owned = view.Owned || view.Equipped
	view.Level = b.abyssPouchLevel(uid)
	view.StackCap, view.CarryCap = abyssPouchCaps(view.Level)
	view.NextStack, view.NextCarry = view.StackCap, view.CarryCap
	switch {
	case !view.Owned:
		view.StatusLabel = "Find a Consumable Pouch in Abyss loot to unlock tailoring."
	case view.Level >= abyssPouchMaxLevel:
		view.StatusLabel = "Masterwork tailoring complete."
	default:
		view.NextCost = abyssPouchUpgradeCosts[view.Level]
		view.NextLevel = view.Level + 1
		view.NextStack, view.NextCarry = abyssPouchCaps(view.Level + 1)
		view.CanUpgrade = true
		if view.Equipped {
			view.StatusLabel = "Equipped · carry expansion active."
		} else {
			view.StatusLabel = "Owned · equip the pouch to activate its carry expansion."
		}
	}
	return view
}

func lockOwnedAbyssPouch(tx *sql.Tx, uid string) error {
	var found int
	err := tx.QueryRow(`SELECT 1 FROM user_gear
		WHERE client_uid=$1 AND gear_id='ABYSS_POUCH' LIMIT 1 FOR UPDATE`, uid).Scan(&found)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	err = tx.QueryRow(`SELECT 1 FROM user_inventory
		WHERE client_uid=$1 AND gear_id='ABYSS_POUCH' LIMIT 1 FOR UPDATE`, uid).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return errAbyssPouchMissing
	}
	return err
}

func upgradeAbyssPouch(tx *sql.Tx, uid string) (level int, gold int64, err error) {
	if err := lockOwnedAbyssPouch(tx, uid); err != nil {
		return 0, 0, err
	}
	if _, err := tx.Exec(`INSERT INTO abyss_consumable_pouches (client_uid,level)
		VALUES ($1,0) ON CONFLICT (client_uid) DO NOTHING`, uid); err != nil {
		return 0, 0, err
	}
	if err := tx.QueryRow("SELECT level FROM abyss_consumable_pouches WHERE client_uid=$1 FOR UPDATE", uid).
		Scan(&level); err != nil {
		return 0, 0, err
	}
	if level >= abyssPouchMaxLevel {
		return 0, 0, errAbyssPouchMaxed
	}
	cost := abyssPouchUpgradeCosts[level]
	if err := tx.QueryRow(`UPDATE users SET gold=gold-$1
		WHERE client_uid=$2 AND gold >= $1 RETURNING gold`, cost, uid).Scan(&gold); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, errAbyssPouchFunds
		}
		return 0, 0, err
	}
	level++
	if _, err := tx.Exec(`UPDATE abyss_consumable_pouches
		SET level=$1,upgraded_at=NOW() WHERE client_uid=$2`, level, uid); err != nil {
		return 0, 0, err
	}
	return level, gold, nil
}

func (s *WebServer) handleAbyssPouchUpgrade(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	level, gold, err := upgradeAbyssPouch(tx, uid)
	if errors.Is(err, errAbyssPouchMissing) || errors.Is(err, errAbyssPouchMaxed) || errors.Is(err, errAbyssPouchFunds) {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	stack, carry := abyssPouchCaps(level)
	nextCost := int64(0)
	if level < abyssPouchMaxLevel {
		nextCost = abyssPouchUpgradeCosts[level]
	}
	writeJSON(w, map[string]any{
		"ok": true, "level": level, "gold": gold, "stack_cap": stack, "carry_cap": carry,
		"next_cost": nextCost,
		"msg":       fmt.Sprintf("Consumable Pouch tailored to rank %d.", level),
	})
}
