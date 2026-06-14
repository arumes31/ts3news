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

func cellRow(c int) int { return c / tftCols }
func cellCol(c int) int { return c % tftCols }
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
	Units []tftUnit `json:"units"`
	Shop  []string  `json:"shop"`
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
	err := b.DB.QueryRow("SELECT data FROM tft_state WHERE client_uid=$1", uid).Scan(&raw)
	st := &tftState{}
	if err == sql.ErrNoRows || len(raw) == 0 {
		// Starter roster: two cheap units on the bench + a fresh shop.
		st.Units = []tftUnit{
			{ID: newUnitID(), Key: "brute", Star: 1, Pos: -1},
			{ID: newUnitID(), Key: "archer", Star: 1, Pos: -1},
		}
		st.Shop = rollShop()
		b.saveTFT(uid, st)
		return st
	}
	if err != nil {
		return st
	}
	_ = json.Unmarshal(raw, st)
	if len(st.Shop) != tftShopSize {
		st.Shop = rollShop()
	}
	return st
}

func (b *Bot) saveTFT(uid string, st *tftState) {
	data, _ := json.Marshal(st)
	_, _ = b.DB.Exec(
		`INSERT INTO tft_state (client_uid, data, updated_at) VALUES ($1, $2, NOW())
		 ON CONFLICT (client_uid) DO UPDATE SET data=$2, updated_at=NOW()`, uid, data)
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
		"History":   s.bot.battleHistory(uid, 12),
		"Leaders":   s.bot.gameLeaderboards("tft"),
		"Inventory": s.bot.inventoryItems(uid),
		"Traits": map[string]any{
			"warrior":  []int{2, 4, 6},
			"tank":     []int{2, 4, 6},
			"assassin": []int{2, 4, 6},
			"mage":     []int{2, 4, 6},
			"dragon":   []int{1},
			"titan":    []int{1},
		},
	})
}

// ===== Shop / board management APIs =====
func (s *WebServer) handleTFTBuy(w http.ResponseWriter, r *http.Request, uid string) {
	var req struct{ Index int `json:"index"` }
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
	res, err := s.bot.DB.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid=$2 AND gold >= $1", d.Cost, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
		return
	}
	st.Units = append(st.Units, tftUnit{ID: newUnitID(), Key: d.Key, Star: 1, Pos: -1})
	st.Shop[req.Index] = ""
	combineUnits(st)
	s.bot.saveTFT(uid, st)
	writeJSON(w, map[string]any{"ok": true, "gold": s.bot.userGold(uid)})
}

func (s *WebServer) handleTFTReroll(w http.ResponseWriter, r *http.Request, uid string) {
	res, err := s.bot.DB.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid=$2 AND gold >= $1", rerollCost, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
		return
	}
	st := s.bot.loadTFT(uid)
	st.Shop = rollShop()
	s.bot.saveTFT(uid, st)
	writeJSON(w, map[string]any{"ok": true})
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
	var req struct{ ID string `json:"id"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	st := s.bot.loadTFT(uid)
	var refund int64
	var next []tftUnit
	for _, u := range st.Units {
		if u.ID == req.ID {
			d, _ := tftDefByKey(u.Key)
			refund = int64(d.Cost * u.Star) // sell value
			continue
		}
		next = append(next, u)
	}
	st.Units = next
	s.bot.saveTFT(uid, st)
	if refund > 0 {
		_, _ = s.bot.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", refund, uid)
	}
	writeJSON(w, map[string]any{"ok": true, "gold": s.bot.userGold(uid), "refund": refund})
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
	OK      bool       `json:"ok"`
	Error   string     `json:"error,omitempty"`
	Victory bool       `json:"victory"`
	IsCreep bool       `json:"is_creep"`
	Frames  []tftFrame `json:"frames"`
	GoldWon int64      `json:"gold_won"`
	GearWon string     `json:"gear_won,omitempty"`
	Gold    int64      `json:"gold"`
}

type simUnit struct {
	id, icon, side string
	star           int
	pos            int
	hp, maxhp      int
	atk, rng       int
	cd             int
	traits         []string
	critChance     int // 0-100
	dmgRed         float64
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

	// Check if this is a creep round (based on history length)
	var histCount int
	_ = s.bot.DB.QueryRow("SELECT COUNT(*) FROM battle_history WHERE client_uid=$1", uid).Scan(&histCount)
	isCreep := histCount > 0 && histCount%3 == 0

	var enemies []*simUnit
	if isCreep {
		enemies = generateCreeps(u.Level)
	} else {
		enemies = generateEnemies(len(units), u.Level)
	}
	units = append(units, enemies...)

	// Apply synergies
	applySynergies(units)

	frames, victory := simulateTFT(units)

	// Rewards.
	res := tftCombatResult{OK: true, Victory: victory, Frames: frames, IsCreep: isCreep}
	if victory {
		res.GoldWon = int64(3 + len(enemies)*2 + u.Level/3)
		if isCreep {
			res.GoldWon *= 2 // Double gold for creep rounds
		}
		_, _ = s.bot.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", res.GoldWon, uid)
		
		// Improve gear: 45% chance to drop a gear piece. Guaranteed on Creep victory.
		dropChance := 45
		if isCreep {
			dropChance = 100
		}
		// #nosec G404
		if rand.IntN(100) < dropChance {
			g := content.RandomGearDrop()
			if _, err := s.bot.DB.Exec("INSERT INTO user_inventory (client_uid, gear_id, durability) VALUES ($1,$2,$3)", uid, g.ID, g.MaxDurability); err == nil {
				res.GearWon = g.Rarity.String() + " " + g.Name
			}
		}
	}
	// Record history.
	var gearWon any
	if res.GearWon != "" {
		gearWon = res.GearWon
	}
	mobName := fmt.Sprintf("TFT (%d enemies)", len(enemies))
	if isCreep {
		mobName = "CREEP ROUND: Golems & Wolves"
	}
	_, _ = s.bot.DB.Exec(
		"INSERT INTO battle_history (client_uid, mob_name, victory, gold_won, gear_won) VALUES ($1,$2,$3,$4,$5)",
		uid, mobName, victory, res.GoldWon, gearWon)
	s.bot.recordGameResult(uid, "tft", victory, res.GoldWon)

	res.Gold = s.bot.userGold(uid)
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
	}
}

func generateEnemies(playerCount, level int) []*simUnit {
	count := playerCount + 1
	if count > 8 {
		count = 8
	}
	// #nosec G404
	r := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	scale := 1.0 + 0.06*float64(level)
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

func generateCreeps(level int) []*simUnit {
	scale := 1.2 + 0.1*float64(level)
	var out []*simUnit
	
	// Buffed units
	out = append(out, &simUnit{
		id: "c1", icon: "🗿", side: "enemy", pos: 3, star: 1,
		hp: int(2000*scale), maxhp: int(2000*scale), atk: int(100*scale), rng: 1,
	})
	out = append(out, &simUnit{
		id: "c2", icon: "🐺", side: "enemy", pos: 10, star: 1,
		hp: int(800*scale), maxhp: int(800*scale), atk: int(150*scale), rng: 1,
	})
	out = append(out, &simUnit{
		id: "c3", icon: "🐺", side: "enemy", pos: 11, star: 1,
		hp: int(800*scale), maxhp: int(800*scale), atk: int(150*scale), rng: 1,
	})
	
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

// simulateTFT runs the board fight tick by tick, returning animation frames and
// whether the player's side won.
func simulateTFT(units []*simUnit) ([]tftFrame, bool) {
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
						dmg *= 2
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
	}

	return frames, alive("you") > 0 && alive("enemy") == 0
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
