package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"sort"

	"ts3news/internal/content"
)

// ===== Board geometry =====
// 7 columns x 4 rows. Player units occupy the bottom two rows (cells 14..27),
// enemies the top two rows (0..13) during combat.
const (
	tftCols      = 7
	tftRows      = 4
	tftCells     = tftCols * tftRows
	tftBenchSize = 8
	tftShopSize  = 5
	rerollCost   = 2
)

func cellRow(c int) int       { return c / tftCols }
func cellCol(c int) int       { return c % tftCols }
func isPlayerCell(c int) bool { return c >= tftCols*2 && c < tftCells }

// ===== Champion definitions =====
type tftDef struct {
	Key    string
	Name   string
	Icon   string
	Cost   int
	HP     int
	ATK    int
	Rng    int      // 1 = melee, 2+ = ranged
	Traits []string // e.g. "warrior", "human"
}

var tftDefs = []tftDef{
	{"brute", "Brute", "🪓", 1, 600, 55, 1, []string{"warrior", "brute"}},
	{"wolf", "Dire Wolf", "🐺", 1, 500, 60, 1, []string{"wild", "assassin"}},
	{"archer", "Archer", "🏹", 2, 450, 70, 3, []string{"scout", "ranger"}},
	{"mage", "Frost Mage", "🧙", 2, 420, 80, 3, []string{"mage", "elemental"}},
	{"knight", "Knight", "🛡️", 3, 900, 65, 1, []string{"knight", "tank"}},
	{"rogue", "Rogue", "🗡️", 3, 550, 110, 1, []string{"assassin", "rogue"}},
	{"golem", "Golem", "🗿", 4, 1300, 75, 1, []string{"tank", "golem"}},
	{"sorcerer", "Sorcerer", "🔮", 4, 600, 150, 3, []string{"mage", "mystic"}},
	{"dragon", "Dragon", "🐉", 5, 1600, 170, 2, []string{"dragon", "titan"}},
	{"titan", "Titan", "⚡", 5, 2200, 140, 1, []string{"titan", "tank"}},
}

func tftDefByKey(k string) (tftDef, bool) {
	for _, d := range tftDefs {
		if d.Key == k {
			return d, true
		}
	}
	return tftDef{}, false
}

// shop roll weighting by cost (cheaper units far more common).
var costWeights = map[int]int{1: 40, 2: 28, 3: 18, 4: 10, 5: 4}

// ===== Persistent state =====
type tftUnit struct {
	ID    string   `json:"id"`
	Key   string   `json:"key"`
	Star  int      `json:"star"`
	Pos   int      `json:"pos"`   // -1 = bench, else board cell
	Items []string `json:"items"` // list of gear IDs (from global inventory)
}

type tftState struct {
	Units       []tftUnit `json:"units"`
	Shop        []string  `json:"shop"`
	BattleGold  int       `json:"battle_gold"`
	Phase       string    `json:"phase"`        // "planning", "combat", "overtime"
	PhaseTimer  int       `json:"phase_timer"`  // seconds remaining in current phase
	RoundNumber int       `json:"round_number"` // current round (1, 2, 3, ...)
	StageNumber int       `json:"stage_number"` // current stage (1, 2, 3, ...)
}

func newUnitID() string {
	// #nosec G404 -- instance id, not security sensitive
	return fmt.Sprintf("u%08x", rand.Uint32())
}

func rollShop() []string {
	// #nosec G404
	r := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	// Build a weighted pool.
	var pool []tftDef
	for _, d := range tftDefs {
		for i := 0; i < costWeights[d.Cost]; i++ {
			pool = append(pool, d)
		}
	}
	out := make([]string, tftShopSize)
	for i := range out {
		out[i] = pool[r.IntN(len(pool))].Key
	}
	return out
}

func (b *Bot) loadTFT(uid string) *tftState {
	var raw []byte
	var battleGold, phaseTimer, roundNumber, stageNumber int
	var phase string
	err := b.DB.QueryRow("SELECT data, COALESCE(battle_gold, 0), COALESCE(phase, 'planning'), COALESCE(phase_timer, 30), COALESCE(round_number, 1), COALESCE(stage_number, 1) FROM tft_state WHERE client_uid=$1", uid).Scan(&raw, &battleGold, &phase, &phaseTimer, &roundNumber, &stageNumber)
	st := &tftState{}
	if err == sql.ErrNoRows || len(raw) == 0 {
		// Starter roster: two cheap units on the bench + a fresh shop.
		st.Units = []tftUnit{
			{ID: newUnitID(), Key: "brute", Star: 1, Pos: -1},
			{ID: newUnitID(), Key: "archer", Star: 1, Pos: -1},
		}
		st.Shop = rollShop()
		st.BattleGold = 0 // Will be set when game starts
		st.Phase = "planning"
		st.PhaseTimer = 30
		st.RoundNumber = 1
		st.StageNumber = 1
		b.saveTFT(uid, st)
		return st
	}
	if err != nil {
		return st
	}
	_ = json.Unmarshal(raw, st)
	st.BattleGold = battleGold
	st.Phase = phase
	st.PhaseTimer = phaseTimer
	st.RoundNumber = roundNumber
	st.StageNumber = stageNumber
	if len(st.Shop) != tftShopSize {
		st.Shop = rollShop()
	}
	// Ensure phase is valid
	if st.Phase != "planning" && st.Phase != "combat" && st.Phase != "overtime" {
		st.Phase = "planning"
	}
	return st
}

func (b *Bot) saveTFT(uid string, st *tftState) {
	data, _ := json.Marshal(st)
	_, _ = b.DB.Exec(
		`INSERT INTO tft_state (client_uid, data, battle_gold, phase, phase_timer, round_number, stage_number, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		 ON CONFLICT (client_uid) DO UPDATE SET
		 data=$2, battle_gold=$3, phase=$4, phase_timer=$5, round_number=$6, stage_number=$7, updated_at=NOW()`,
		uid, data, st.BattleGold, st.Phase, st.PhaseTimer, st.RoundNumber, st.StageNumber)
}

// combineUnits upgrades any 3-of-a-kind (same key + star) into one of star+1.
func combineUnits(st *tftState) {
	changed := true
	for changed {
		changed = false
		groups := map[string][]int{}
		for i, u := range st.Units {
			k := fmt.Sprintf("%s/%d", u.Key, u.Star)
			groups[k] = append(groups[k], i)
		}
		for _, idxs := range groups {
			if len(idxs) >= 3 {
				// Keep idxs[0] (upgrade it), remove idxs[1], idxs[2].
				// Transfer items from removed units to the upgraded one if possible.
				up := st.Units[idxs[0]]
				up.Star++
				for i := 1; i < 3; i++ {
					for _, itm := range st.Units[idxs[i]].Items {
						if len(up.Items) < 3 {
							up.Items = append(up.Items, itm)
						}
					}
				}

				rm := map[int]bool{idxs[1]: true, idxs[2]: true}
				var next []tftUnit
				for i, u := range st.Units {
					if rm[i] {
						continue
					}
					if i == idxs[0] {
						u = up
					}
					next = append(next, u)
				}
				st.Units = next
				changed = true
				break
			}
		}
	}
}

func benchCount(st *tftState) int {
	n := 0
	for _, u := range st.Units {
		if u.Pos < 0 {
			n++
		}
	}
	return n
}

// ===== View models for the page =====
type tftUnitView struct {
	ID     string   `json:"id"`
	Icon   string   `json:"icon"`
	Name   string   `json:"name"`
	Star   int      `json:"star"`
	Pos    int      `json:"pos"`
	HP     int      `json:"hp"`
	ATK    int      `json:"atk"`
	Items  []string `json:"items"`
	Traits []string `json:"traits"`
}

type tftShopView struct {
	Index int    `json:"index"`
	Key   string `json:"key"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Cost  int    `json:"cost"`
}

func unitView(u tftUnit) tftUnitView {
	d, _ := tftDefByKey(u.Key)
	hp, atk := starStats(d, u.Star)
	return tftUnitView{
		ID: u.ID, Icon: d.Icon, Name: d.Name, Star: u.Star, Pos: u.Pos,
		HP: hp, ATK: atk, Items: u.Items, Traits: d.Traits,
	}
}

func starStats(d tftDef, star int) (int, int) {
	mult := 1.0
	for i := 1; i < star; i++ {
		mult *= 1.8
	}
	return int(float64(d.HP) * mult), int(float64(d.ATK) * mult)
}

func (s *WebServer) handleBattlePage(w http.ResponseWriter, r *http.Request, uid string) {
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}
	st := s.bot.loadTFT(uid)

	bench := []tftUnitView{}
	board := []tftUnitView{}
	for _, un := range st.Units {
		if un.Pos < 0 {
			bench = append(bench, unitView(un))
		} else {
			board = append(board, unitView(un))
		}
	}
	shop := []tftShopView{}
	for i, k := range st.Shop {
		if d, ok := tftDefByKey(k); ok {
			shop = append(shop, tftShopView{Index: i, Key: k, Name: d.Name, Icon: d.Icon, Cost: d.Cost})
		}
	}
	s.render(w, "battle", map[string]any{
		"Title": "Auto-Battler", "Nav": "battle", "U": u,
		"BenchJSON": jsonJS(bench),
		"BoardJSON": jsonJS(board),
		"ShopJSON":  jsonJS(shop),
		"Cols":      tftCols, "Rows": tftRows, "Cells": tftCells,
		"History":     s.bot.battleHistory(uid, 12),
		"Leaders":     s.bot.gameLeaderboards("tft"),
		"Inventory":   s.bot.inventoryItems(uid),
		"BattleStats": s.bot.getBattleStats(uid),
		"BattleGold":  st.BattleGold,
		"Phase":       st.Phase,
		"PhaseTimer":  st.PhaseTimer,
		"RoundNumber": st.RoundNumber,
		"StageNumber": st.StageNumber,
		"Traits": map[string]any{
			"warrior":   []int{2, 4, 6},
			"tank":      []int{2, 4, 6},
			"assassin":  []int{2, 4, 6},
			"mage":      []int{2, 4, 6},
			"dragon":    []int{1},
			"titan":     []int{1},
			"brute":     []int{2, 4, 6},
			"wild":      []int{2, 4, 6},
			"scout":     []int{2, 4, 6},
			"ranger":    []int{2, 4, 6},
			"elemental": []int{2, 4, 6},
			"knight":    []int{2, 4, 6},
			"rogue":     []int{2, 4, 6},
			"golem":     []int{2, 4, 6},
			"mystic":    []int{2, 4, 6},
		},
	})
}

// ===== Shop / board management APIs =====
func (s *WebServer) handleTFTBuy(w http.ResponseWriter, r *http.Request, uid string) {
	var req struct {
		Index int `json:"index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	st := s.bot.loadTFT(uid)
	if req.Index < 0 || req.Index >= len(st.Shop) || st.Shop[req.Index] == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "empty slot"})
		return
	}
	if benchCount(st) >= tftBenchSize {
		writeJSON(w, map[string]any{"ok": false, "error": "bench full"})
		return
	}
	d, _ := tftDefByKey(st.Shop[req.Index])
	// Use battle gold instead of real gold
	if st.BattleGold < d.Cost {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough battle gold"})
		return
	}
	st.BattleGold -= d.Cost
	st.Units = append(st.Units, tftUnit{ID: newUnitID(), Key: d.Key, Star: 1, Pos: -1})
	st.Shop[req.Index] = ""
	combineUnits(st)
	s.bot.saveTFT(uid, st)
	writeJSON(w, map[string]any{"ok": true, "battleGold": st.BattleGold})
}

func (s *WebServer) handleTFTReroll(w http.ResponseWriter, r *http.Request, uid string) {
	st := s.bot.loadTFT(uid)
	// Use battle gold instead of real gold
	if st.BattleGold < rerollCost {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough battle gold"})
		return
	}
	st.BattleGold -= rerollCost
	st.Shop = rollShop()
	s.bot.saveTFT(uid, st)
	writeJSON(w, map[string]any{"ok": true, "battleGold": st.BattleGold})
}

func (s *WebServer) handleTFTPlace(w http.ResponseWriter, r *http.Request, uid string) {
	var req struct {
		ID  string `json:"id"`
		Pos int    `json:"pos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if req.Pos >= 0 && !isPlayerCell(req.Pos) {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid cell"})
		return
	}
	st := s.bot.loadTFT(uid)
	// Target cell occupied? swap.
	var occupant = -1
	if req.Pos >= 0 {
		for i, u := range st.Units {
			if u.Pos == req.Pos && u.ID != req.ID {
				occupant = i
			}
		}
	}
	var from = -2
	for i := range st.Units {
		if st.Units[i].ID == req.ID {
			from = st.Units[i].Pos
		}
	}
	if from == -2 {
		writeJSON(w, map[string]any{"ok": false, "error": "no unit"})
		return
	}
	for i := range st.Units {
		if st.Units[i].ID == req.ID {
			st.Units[i].Pos = req.Pos
		}
	}
	if occupant >= 0 {
		st.Units[occupant].Pos = from // swap into the mover's old spot
	}
	s.bot.saveTFT(uid, st)
	writeJSON(w, map[string]any{"ok": true})
}

func (s *WebServer) handleTFTSell(w http.ResponseWriter, r *http.Request, uid string) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	st := s.bot.loadTFT(uid)
	var refund int
	var next []tftUnit
	for _, u := range st.Units {
		if u.ID == req.ID {
			d, _ := tftDefByKey(u.Key)
			refund = d.Cost * u.Star // sell value
			continue
		}
		next = append(next, u)
	}
	st.Units = next
	// Refund to battle gold instead of real gold
	if refund > 0 {
		st.BattleGold += refund
	}
	s.bot.saveTFT(uid, st)
	writeJSON(w, map[string]any{"ok": true, "battleGold": st.BattleGold, "refund": refund})
}

// handleTFTEquip equips a gear piece from inventory onto a unit.
func (s *WebServer) handleTFTEquip(w http.ResponseWriter, r *http.Request, uid string) {
	var req struct {
		UnitID string `json:"unit_id"`
		InvID  string `json:"inv_id"` // gear_id
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	st := s.bot.loadTFT(uid)
	found := -1
	for i, u := range st.Units {
		if u.ID == req.UnitID {
			found = i
			break
		}
	}
	if found < 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "no unit"})
		return
	}
	if len(st.Units[found].Items) >= 3 {
		writeJSON(w, map[string]any{"ok": false, "error": "unit items full"})
		return
	}
	// Atomic check and remove from inventory to prevent duplication
	res, err := s.bot.DB.Exec("DELETE FROM user_inventory WHERE id IN (SELECT id FROM user_inventory WHERE client_uid=$1 AND gear_id=$2 LIMIT 1)", uid, req.InvID)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db error"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "item not in inventory"})
		return
	}

	st.Units[found].Items = append(st.Units[found].Items, req.InvID)
	s.bot.saveTFT(uid, st)
	writeJSON(w, map[string]any{"ok": true})
}

// ===== Phase Management APIs =====

// handleTFTPhaseReady marks the player as ready for combat, transitioning from planning to combat phase
func (s *WebServer) handleTFTPhaseReady(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	st := s.bot.loadTFT(uid)

	// Can only transition from planning or overtime to combat
	if st.Phase != "planning" && st.Phase != "overtime" {
		writeJSON(w, map[string]any{"ok": false, "error": "not in planning or overtime phase"})
		return
	}

	// Must have at least one unit on board
	hasUnits := false
	for _, u := range st.Units {
		if u.Pos >= 0 {
			hasUnits = true
			break
		}
	}
	if !hasUnits {
		writeJSON(w, map[string]any{"ok": false, "error": "place at least one unit on the board"})
		return
	}

	// Transition to combat phase
	st.Phase = "combat"
	st.PhaseTimer = 0 // Combat phase doesn't use timer, it's resolved immediately

	s.bot.saveTFT(uid, st)
	writeJSON(w, map[string]any{"ok": true, "phase": st.Phase})
}

// handleTFTPhaseTimer updates the phase timer (used for planning phase countdown)
func (s *WebServer) handleTFTPhaseTimer(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Timer int `json:"timer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	st := s.bot.loadTFT(uid)

	// Only update timer if in planning or overtime phase
	if st.Phase == "planning" || st.Phase == "overtime" {
		maxTimer := 30
		if st.Phase == "overtime" {
			maxTimer = 10 // Overtime has max 10 seconds
		}
		if req.Timer >= 0 && req.Timer <= maxTimer {
			st.PhaseTimer = req.Timer
			s.bot.saveTFT(uid, st)
		}
	}

	writeJSON(w, map[string]any{"ok": true, "timer": st.PhaseTimer})
}

// advanceRound advances the round/stage numbers and resets phase to planning
func (s *WebServer) advanceRound(st *tftState) {
	st.RoundNumber++

	// Stage advances every 7 rounds (1-1, 1-2, ..., 1-7, 2-1, 2-2, ...)
	if st.RoundNumber%7 == 1 {
		st.StageNumber++
	}

	// Reset to planning phase with fresh timer
	st.Phase = "planning"
	st.PhaseTimer = 30

	// Award battle gold at start of each round
	battleGoldAward := 5 + st.RoundNumber
	if st.RoundNumber%3 == 0 { // Creep round bonus
		battleGoldAward *= 2
	}
	st.BattleGold += battleGoldAward
}

func (b *Bot) userGold(uid string) int64 {
	var g int64
	_ = b.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&g)
	return g
}

// ===== Combat =====
type tftFrameUnit struct {
	ID    string `json:"id"`
	Icon  string `json:"icon"`
	Side  string `json:"side"`
	Pos   int    `json:"pos"`
	HP    int    `json:"hp"`
	MaxHP int    `json:"max_hp"`
	Star  int    `json:"star"`
}

type tftEvent struct {
	From string `json:"from"`
	To   string `json:"to"`
	Dmg  int    `json:"dmg"`
}

type tftFrame struct {
	Units  []tftFrameUnit `json:"units"`
	Events []tftEvent     `json:"events"`
}

type tftCombatResult struct {
	OK            bool       `json:"ok"`
	Error         string     `json:"error,omitempty"`
	Victory       bool       `json:"victory"`
	IsCreep       bool       `json:"is_creep"`
	Frames        []tftFrame `json:"frames"`
	GoldWon       int64      `json:"gold_won"`
	GearWon       string     `json:"gear_won,omitempty"`
	Gold          int64      `json:"gold"`
	BattleGold    int        `json:"battle_gold"`
	WaveNumber    int        `json:"wave_number"`
	HighestWave   int        `json:"highest_wave"`
	DamageDealt   int        `json:"damage_dealt"`
	TurnsSurvived int        `json:"turns_survived"`
	IsMilestone   bool       `json:"is_milestone"`
	Streak        int        `json:"streak"`
}

type simUnit struct {
	id, icon, side string
	star           int
	pos            int
	hp, maxhp      int
	atk, rng       int
	cd             int
	traits         []string
	critChance     int     // 0-100
	critDmg        float64 // crit damage multiplier
	dmgRed         float64
	lifesteal      float64 // lifesteal percentage
	damageDealt    int
}

var enemyIcons = []string{"👹", "👺", "💀", "👻", "🦂", "🕷️", "🐍", "🦅"}

func (s *WebServer) handleTFTCombat(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	u, err := s.loadWebUser(uid)
	if err != nil {
		writeJSON(w, tftCombatResult{OK: false, Error: "no character"})
		return
	}
	st := s.bot.loadTFT(uid)

	// Validate phase - combat can only happen during planning or overtime phases
	if st.Phase != "planning" && st.Phase != "overtime" {
		writeJSON(w, tftCombatResult{OK: false, Error: "combat not allowed in current phase"})
		return
	}

	// Gather the player's placed units.
	var units []*simUnit
	occupied := map[int]bool{}
	for _, un := range st.Units {
		if un.Pos < 0 {
			continue
		}
		d, ok := tftDefByKey(un.Key)
		if !ok {
			continue
		}
		hp, atk := starStats(d, un.Star)

		// Apply item bonuses
		for _, itmID := range un.Items {
			gear, ok := content.GetGearByID(itmID)
			if ok {
				hp += gear.Stats.HP
				atk += gear.Stats.STR
			}
		}

		units = append(units, &simUnit{
			id: un.ID, icon: d.Icon, side: "you", star: un.Star, pos: un.Pos,
			hp: hp, maxhp: hp, atk: atk, rng: d.Rng, cd: 0,
			traits: d.Traits, critChance: 5, dmgRed: 0,
		})
		occupied[un.Pos] = true
	}
	if len(units) == 0 {
		writeJSON(w, tftCombatResult{OK: false, Error: "place at least one unit on the board"})
		return
	}

	// Use RoundNumber from state instead of calculating from history
	roundNumber := st.RoundNumber
	stageNumber := st.StageNumber

	// Check if this is a creep round (every 3rd round)
	isCreep := roundNumber%3 == 0

	var enemies []*simUnit
	if isCreep {
		enemies = generateCreeps(u.Level, roundNumber)
	} else {
		enemies = generateEnemies(len(units), u.Level, roundNumber)
	}
	units = append(units, enemies...)

	// Apply synergies
	applySynergies(units)

	// Track round milestone (every 5 rounds)
	isMilestone := roundNumber%5 == 0

	// Award battle gold at the start of each round (base + round scaling)
	battleGoldAward := 5 + roundNumber
	if isCreep {
		battleGoldAward *= 2 // Bonus for creep rounds
	}
	st.BattleGold += battleGoldAward

	frames, victory, damageDealt, turnsSurvived := simulateTFT(units)

	// Calculate rewards with round-based scaling
	res := tftCombatResult{
		OK:            true,
		Victory:       victory,
		Frames:        frames,
		IsCreep:       isCreep,
		WaveNumber:    roundNumber,
		DamageDealt:   damageDealt,
		TurnsSurvived: turnsSurvived,
		IsMilestone:   isMilestone,
		BattleGold:    st.BattleGold,
	}

	if victory {
		// Base gold reward (real gold - end game reward only)
		baseGold := int64(3 + len(enemies)*2 + u.Level/3)

		// Round scaling: +1 gold per round
		roundBonus := int64(roundNumber)

		// Creep round bonus (2x)
		creepMultiplier := int64(1)
		if isCreep {
			creepMultiplier = 2
		}

		// Milestone bonus (every 5 rounds)
		milestoneBonus := int64(0)
		if isMilestone {
			milestoneBonus = int64(5 * (roundNumber / 5))
		}

		res.GoldWon = (baseGold + roundBonus + milestoneBonus) * creepMultiplier
		// Award real gold only as end-game reward
		_, _ = s.bot.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", res.GoldWon, uid)

		// Gear drop chance with round scaling
		// Base 45%, +1% per round, guaranteed on creep rounds
		dropChance := 45 + roundNumber
		if dropChance > 90 {
			dropChance = 90 // Cap at 90% for non-creep
		}
		if isCreep {
			dropChance = 100
		}

		// #nosec G404
		if rand.IntN(100) < dropChance {
			g := content.RandomGearDrop()
			result := s.bot.awardGearDrop(uid, g)
			res.GearWon = result.Prefix + result.ItemName
		}
	}
	// Record history with round tracking
	var gearWon any
	if res.GearWon != "" {
		gearWon = res.GearWon
	}
	// Format mob name as "Round X-Y" where X is stage, Y is round within stage
	mobName := fmt.Sprintf("Round %d-%d (%d enemies)", stageNumber, roundNumber, len(enemies))
	if isCreep {
		mobName = fmt.Sprintf("CREEP ROUND %d-%d: Golems & Wolves", stageNumber, roundNumber)
	}
	_, _ = s.bot.DB.Exec(
		`INSERT INTO battle_history (client_uid, mob_name, victory, gold_won, gear_won, wave_number, highest_wave, damage_dealt, turns_survived)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		uid, mobName, victory, res.GoldWon, gearWon, roundNumber, roundNumber, damageDealt, turnsSurvived)

	// Update battle statistics
	s.bot.updateBattleStats(uid, victory, roundNumber, damageDealt, turnsSurvived)

	s.bot.recordGameResult(uid, "tft", victory, res.GoldWon)

	res.Gold = s.bot.userGold(uid)
	res.HighestWave = roundNumber
	res.Streak = s.bot.getCurrentStreak(uid)

	// Advance to next round/stage and reset phase
	s.advanceRound(st)

	s.bot.saveTFT(uid, st) // Save battle gold state and phase
	writeJSON(w, res)
}

func applySynergies(units []*simUnit) {
	counts := map[string]int{}
	for _, u := range units {
		if u.side != "you" {
			continue
		}
		for _, t := range u.traits {
			counts[t]++
		}
	}

	for _, u := range units {
		if u.side != "you" {
			continue
		}
		// Warrior: +20/40/80 ATK
		if c := counts["warrior"]; c >= 6 {
			u.atk += 80
		} else if c >= 4 {
			u.atk += 40
		} else if c >= 2 {
			u.atk += 20
		}
		// Tank: +150/300/600 HP
		if c := counts["tank"]; c >= 6 {
			u.maxhp += 600
			u.hp += 600
		} else if c >= 4 {
			u.maxhp += 300
			u.hp += 300
		} else if c >= 2 {
			u.maxhp += 150
			u.hp += 150
		}
		// Assassin: +15/30/60% Crit Chance
		if c := counts["assassin"]; c >= 6 {
			u.critChance += 60
		} else if c >= 4 {
			u.critChance += 30
		} else if c >= 2 {
			u.critChance += 15
		}
		// Mage: +30/60/120 ATK
		if c := counts["mage"]; c >= 6 {
			u.atk += 120
		} else if c >= 4 {
			u.atk += 60
		} else if c >= 2 {
			u.atk += 30
		}
		// Dragon: 1 -> +1000 HP, +100 ATK
		if counts["dragon"] >= 1 {
			u.maxhp += 1000
			u.hp += 1000
			u.atk += 100
		}
		// Titan: 1 -> 50% DMG Red
		if counts["titan"] >= 1 {
			u.dmgRed = 0.5
		}
		// Brute: +10/25/50% Attack Speed (reduced cooldown)
		if c := counts["brute"]; c >= 6 {
			u.cd = u.cd * 50 / 100 // 50% faster attacks
		} else if c >= 4 {
			u.cd = u.cd * 75 / 100
		} else if c >= 2 {
			u.cd = u.cd * 90 / 100
		}
		// Wild: +5/12/25% Lifesteal (heal on attack)
		if c := counts["wild"]; c >= 6 {
			u.lifesteal = 0.25
		} else if c >= 4 {
			u.lifesteal = 0.12
		} else if c >= 2 {
			u.lifesteal = 0.05
		}
		// Scout: +10/25/50% Move Speed (move 2 cells per turn)
		// Implemented as bonus attack range
		if c := counts["scout"]; c >= 6 {
			u.rng += 2
		} else if c >= 4 {
			u.rng += 1
		} else if c >= 2 {
			u.rng = maxInt(u.rng, 2)
		}
		// Ranger: +15/35/60% Attack Speed
		if c := counts["ranger"]; c >= 6 {
			u.cd = u.cd * 40 / 100
		} else if c >= 4 {
			u.cd = u.cd * 65 / 100
		} else if c >= 2 {
			u.cd = u.cd * 85 / 100
		}
		// Elemental: +100/250/500 HP
		if c := counts["elemental"]; c >= 6 {
			u.maxhp += 500
			u.hp += 500
		} else if c >= 4 {
			u.maxhp += 250
			u.hp += 250
		} else if c >= 2 {
			u.maxhp += 100
			u.hp += 100
		}
		// Knight: +10/25/50% Block Chance (damage reduction)
		if c := counts["knight"]; c >= 6 {
			u.dmgRed = maxFloat(u.dmgRed, 0.5)
		} else if c >= 4 {
			u.dmgRed = maxFloat(u.dmgRed, 0.25)
		} else if c >= 2 {
			u.dmgRed = maxFloat(u.dmgRed, 0.1)
		}
		// Rogue: +20/45/80% Crit Damage
		if c := counts["rogue"]; c >= 6 {
			u.critDmg = 1.8
		} else if c >= 4 {
			u.critDmg = 1.45
		} else if c >= 2 {
			u.critDmg = 1.2
		}
		// Golem: +15/35/60% Tenacity (reduced CC - represented as flat HP)
		if c := counts["golem"]; c >= 6 {
			u.maxhp += 400
			u.hp += 400
			u.dmgRed = maxFloat(u.dmgRed, 0.2)
		} else if c >= 4 {
			u.maxhp += 200
			u.hp += 200
			u.dmgRed = maxFloat(u.dmgRed, 0.1)
		} else if c >= 2 {
			u.maxhp += 100
			u.hp += 100
		}
		// Mystic: +15/30/60% Magic Resist (damage reduction)
		if c := counts["mystic"]; c >= 6 {
			u.dmgRed = maxFloat(u.dmgRed, 0.6)
		} else if c >= 4 {
			u.dmgRed = maxFloat(u.dmgRed, 0.3)
		} else if c >= 2 {
			u.dmgRed = maxFloat(u.dmgRed, 0.15)
		}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func generateEnemies(playerCount, level int, wave int) []*simUnit {
	count := playerCount + 1
	if count > 8 {
		count = 8
	}
	// #nosec G404
	r := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))

	// Scale with both level and wave
	scale := 1.0 + 0.06*float64(level) + 0.03*float64(wave)

	var out []*simUnit
	// Place enemies across the top two rows.
	cells := []int{}
	for c := 0; c < tftCols*2; c++ {
		cells = append(cells, c)
	}
	r.Shuffle(len(cells), func(i, j int) { cells[i], cells[j] = cells[j], cells[i] })
	for i := 0; i < count && i < len(cells); i++ {
		hp := int(float64(380+r.IntN(260)) * scale)
		atk := int(float64(45+r.IntN(50)) * scale)
		rng := 1
		if r.IntN(3) == 0 {
			rng = 3
		}
		out = append(out, &simUnit{
			id:   fmt.Sprintf("e%d", i),
			icon: enemyIcons[r.IntN(len(enemyIcons))],
			side: "enemy", star: 1, pos: cells[i], hp: hp, maxhp: hp, atk: atk, rng: rng, cd: 0,
			critChance: 5,
		})
	}
	return out
}

func generateCreeps(level int, wave int) []*simUnit {
	// Creeps scale harder with wave progression
	scale := 1.2 + 0.1*float64(level) + 0.05*float64(wave)
	var out []*simUnit

	// Golem tank
	out = append(out, &simUnit{
		id: "c1", icon: "🗿", side: "enemy", pos: 3, star: 1,
		hp: int(2000 * scale), maxhp: int(2000 * scale), atk: int(100 * scale), rng: 1,
	})
	// Wolf pack
	out = append(out, &simUnit{
		id: "c2", icon: "🐺", side: "enemy", pos: 10, star: 1,
		hp: int(800 * scale), maxhp: int(800 * scale), atk: int(150 * scale), rng: 1,
	})
	out = append(out, &simUnit{
		id: "c3", icon: "🐺", side: "enemy", pos: 11, star: 1,
		hp: int(800 * scale), maxhp: int(800 * scale), atk: int(150 * scale), rng: 1,
	})

	// Add more wolves on higher waves
	if wave >= 6 {
		out = append(out, &simUnit{
			id: "c4", icon: "🐺", side: "enemy", pos: 12, star: 1,
			hp: int(800 * scale), maxhp: int(800 * scale), atk: int(150 * scale), rng: 1,
		})
	}
	if wave >= 12 {
		out = append(out, &simUnit{
			id: "c5", icon: "🐺", side: "enemy", pos: 13, star: 1,
			hp: int(800 * scale), maxhp: int(800 * scale), atk: int(150 * scale), rng: 1,
		})
	}

	return out
}

func chebyshev(a, b int) int {
	dr := cellRow(a) - cellRow(b)
	dc := cellCol(a) - cellCol(b)
	if dr < 0 {
		dr = -dr
	}
	if dc < 0 {
		dc = -dc
	}
	if dr > dc {
		return dr
	}
	return dc
}

// simulateTFT runs the board fight tick by tick, returning animation frames,
// whether the player's side won, total damage dealt, and turns survived.
func simulateTFT(units []*simUnit) ([]tftFrame, bool, int, int) {
	const maxTicks = 120
	const attackCooldown = 2

	snapshot := func() tftFrame {
		var f tftFrame
		for _, u := range units {
			if u.hp <= 0 {
				continue
			}
			f.Units = append(f.Units, tftFrameUnit{ID: u.id, Icon: u.icon, Side: u.side, Pos: u.pos, HP: u.hp, MaxHP: u.maxhp, Star: u.star})
		}
		return f
	}

	frames := []tftFrame{snapshot()}

	totalDamage := 0
	ticksSurvived := 0

	alive := func(side string) int {
		n := 0
		for _, u := range units {
			if u.hp > 0 && u.side == side {
				n++
			}
		}
		return n
	}
	occupied := func(pos, ignore int) bool {
		for i, u := range units {
			if i == ignore || u.hp <= 0 {
				continue
			}
			if u.pos == pos {
				return true
			}
		}
		return false
	}

	// #nosec G404
	r := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))

	for tick := 0; tick < maxTicks; tick++ {
		if alive("you") == 0 || alive("enemy") == 0 {
			break
		}
		var events []tftEvent

		// Deterministic-ish order: by id so frames are stable.
		order := make([]int, len(units))
		for i := range order {
			order[i] = i
		}
		sort.Slice(order, func(a, b int) bool { return units[order[a]].id < units[order[b]].id })

		for _, ui := range order {
			u := units[ui]
			if u.hp <= 0 {
				continue
			}
			// Find nearest living enemy.
			target := -1
			best := 1 << 30
			for vi, v := range units {
				if v.hp <= 0 || v.side == u.side {
					continue
				}
				d := chebyshev(u.pos, v.pos)
				if d < best {
					best, target = d, vi
				}
			}
			if target < 0 {
				continue
			}
			tgt := units[target]
			if best <= u.rng {
				if u.cd <= 0 {
					dmg := u.atk
					if r.IntN(100) < u.critChance {
						dmg = int(float64(dmg) * u.critDmg)
					}
					if tgt.dmgRed > 0 {
						dmg = int(float64(dmg) * (1.0 - tgt.dmgRed))
					}
					tgt.hp -= dmg
					if tgt.hp < 0 {
						tgt.hp = 0
					}
					events = append(events, tftEvent{From: u.id, To: tgt.id, Dmg: dmg})
					u.cd = attackCooldown

					// Lifesteal: heal on attack
					if u.lifesteal > 0 {
						heal := int(float64(dmg) * u.lifesteal)
						if heal > 0 {
							u.hp = minInt(u.hp+heal, u.maxhp)
						}
					}

					// Track damage dealt by player units
					if u.side == "you" {
						u.damageDealt += dmg
						totalDamage += dmg
					}
				}
			} else {
				// Step one cell toward the target.
				dr := sign(cellRow(tgt.pos) - cellRow(u.pos))
				dc := sign(cellCol(tgt.pos) - cellCol(u.pos))
				for _, cand := range []int{
					cellOf(cellRow(u.pos)+dr, cellCol(u.pos)+dc),
					cellOf(cellRow(u.pos)+dr, cellCol(u.pos)),
					cellOf(cellRow(u.pos), cellCol(u.pos)+dc),
				} {
					if cand >= 0 && !occupied(cand, ui) {
						u.pos = cand
						break
					}
				}
			}
			if u.cd > 0 {
				u.cd--
			}
		}
		frames = append(frames, snapshotWithEvents(units, events))
		ticksSurvived++
	}

	// Count only player-side damage
	playerDamage := 0
	for _, u := range units {
		if u.side == "you" {
			playerDamage += u.damageDealt
		}
	}

	return frames, alive("you") > 0 && alive("enemy") == 0, playerDamage, ticksSurvived
}

func snapshotWithEvents(units []*simUnit, events []tftEvent) tftFrame {
	var f tftFrame
	f.Events = events
	for _, u := range units {
		if u.hp <= 0 {
			// Still emit a final 0-hp frame so the client can fade it out.
			f.Units = append(f.Units, tftFrameUnit{ID: u.id, Icon: u.icon, Side: u.side, Pos: u.pos, HP: 0, MaxHP: u.maxhp, Star: u.star})
			continue
		}
		f.Units = append(f.Units, tftFrameUnit{ID: u.id, Icon: u.icon, Side: u.side, Pos: u.pos, HP: u.hp, MaxHP: u.maxhp, Star: u.star})
	}
	return f
}

func sign(n int) int {
	if n > 0 {
		return 1
	}
	if n < 0 {
		return -1
	}
	return 0
}

func cellOf(row, col int) int {
	if row < 0 || row >= tftRows || col < 0 || col >= tftCols {
		return -1
	}
	return row*tftCols + col
}

// updateBattleStats updates the player's cumulative battle statistics after a fight.
func (b *Bot) updateBattleStats(uid string, victory bool, wave int, damage int, turns int) {
	// Upsert battle_stats record
	_, _ = b.DB.Exec(`
		INSERT INTO battle_stats (client_uid, total_battles, total_wins, total_losses,
			current_streak, best_streak, highest_wave, total_damage, total_turns, updated_at)
		SELECT
			$1,
			COALESCE(total_battles, 0) + 1,
			COALESCE(total_wins, 0) + CASE WHEN $2 THEN 1 ELSE 0 END,
			COALESCE(total_losses, 0) + CASE WHEN $2 THEN 0 ELSE 1 END,
			CASE
				WHEN $2 THEN COALESCE(current_streak, 0) + 1
				ELSE 0
			END,
			GREATEST(COALESCE(best_streak, 0),
				CASE WHEN $2 THEN COALESCE(current_streak, 0) + 1 ELSE 0 END),
			GREATEST(COALESCE(highest_wave, 1), $3),
			COALESCE(total_damage, 0) + $4,
			COALESCE(total_turns, 0) + $5,
			NOW()
		FROM battle_stats WHERE client_uid = $1
		ON CONFLICT (client_uid) DO UPDATE SET
			total_battles = EXCLUDED.total_battles,
			total_wins = EXCLUDED.total_wins,
			total_losses = EXCLUDED.total_losses,
			current_streak = EXCLUDED.current_streak,
			best_streak = EXCLUDED.best_streak,
			highest_wave = EXCLUDED.highest_wave,
			total_damage = EXCLUDED.total_damage,
			total_turns = EXCLUDED.total_turns,
			updated_at = NOW()
	`, uid, victory, wave, damage, turns)
}

// getCurrentStreak returns the player's current win/loss streak.
// Positive values indicate win streak, negative values indicate loss streak.
func (b *Bot) getCurrentStreak(uid string) int {
	var streak int
	err := b.DB.QueryRow("SELECT current_streak FROM battle_stats WHERE client_uid = $1", uid).Scan(&streak)
	if err != nil {
		return 0
	}
	return streak
}

// getBattleStats returns the full battle statistics for a player.
func (b *Bot) getBattleStats(uid string) (stats struct {
	TotalBattles  int
	TotalWins     int
	TotalLosses   int
	CurrentStreak int
	BestStreak    int
	HighestWave   int
	TotalDamage   int
	TotalTurns    int
}) {
	row := b.DB.QueryRow(`
		SELECT COALESCE(total_battles,0), COALESCE(total_wins,0), COALESCE(total_losses,0),
			COALESCE(current_streak,0), COALESCE(best_streak,0), COALESCE(highest_wave,1),
			COALESCE(total_damage,0), COALESCE(total_turns,0)
		FROM battle_stats WHERE client_uid = $1`, uid)
	_ = row.Scan(&stats.TotalBattles, &stats.TotalWins, &stats.TotalLosses,
		&stats.CurrentStreak, &stats.BestStreak, &stats.HighestWave,
		&stats.TotalDamage, &stats.TotalTurns)
	return
}
