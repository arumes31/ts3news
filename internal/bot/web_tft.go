package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"sort"
	"time"

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
	// Board configuration
	GridType          string `json:"grid_type"`          // "hex" or "square"
	ObstaclePositions []int  `json:"obstacle_positions"` // positions that are blocked
	// Shop configuration
	XP int `json:"xp"` // player XP for shop refreshes
	// Unit upgrades (map of unit_id -> list of upgrade_ids)
	UnitUpgrades map[string][]string `json:"unit_upgrades"`
	// Combat statistics
	TotalDamageDealt   int `json:"total_damage_dealt"`
	TotalDamageTaken   int `json:"total_damage_taken"`
	TotalEnemiesKilled int `json:"total_enemies_killed"`
	CombatCount        int `json:"combat_count"`
}

// ===== Unit Abilities =====
type unitAbility struct {
	ID          string          `json:"id"`
	UnitKey     string          `json:"unit_key"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Icon        string          `json:"icon"`
	AbilityType string          `json:"ability_type"` // "active", "passive", "on_hit", "on_death"
	Cooldown    int             `json:"cooldown"`
	ManaCost    int             `json:"mana_cost"`
	Damage      int             `json:"damage"`
	DamageType  string          `json:"damage_type"` // "physical", "magic", "true"
	Range       int             `json:"range"`
	AOE         bool            `json:"aoe"`
	AOERadius   int             `json:"aoe_radius"`
	Effects     json.RawMessage `json:"effects"`
}

// ===== Unit Upgrades =====
type unitUpgrade struct {
	ID          string `json:"id"`
	UnitKey     string `json:"unit_key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Cost        int    `json:"cost"`
	StarReq     int    `json:"star_req"`
	Effect      string `json:"effect"`
	Value       int    `json:"value"`
}

// loadUnitAbilities loads all abilities for a specific unit key from the database
func (b *Bot) loadUnitAbilities(unitKey string) []unitAbility {
	rows, err := b.DB.Query(`
		SELECT id, unit_key, name, description, icon, ability_type,
		       cooldown, mana_cost, damage, damage_type, range, aoe, aoe_radius, effects
		FROM unit_abilities
		WHERE unit_key = $1
		ORDER BY id
	`, unitKey)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var abilities []unitAbility
	for rows.Next() {
		var a unitAbility
		err := rows.Scan(&a.ID, &a.UnitKey, &a.Name, &a.Description, &a.Icon, &a.AbilityType,
			&a.Cooldown, &a.ManaCost, &a.Damage, &a.DamageType, &a.Range, &a.AOE, &a.AOERadius, &a.Effects)
		if err != nil {
			continue
		}
		abilities = append(abilities, a)
	}
	return abilities
}

// loadUnitUpgrades loads all upgrades for a specific unit key from the database
func (b *Bot) loadUnitUpgrades(unitKey string) []unitUpgrade {
	rows, err := b.DB.Query(`
		SELECT id, unit_key, name, description, cost, star_req, effect, value
		FROM unit_upgrades
		WHERE unit_key = $1 OR unit_key = 'all'
		ORDER BY star_req, cost
	`, unitKey)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var upgrades []unitUpgrade
	for rows.Next() {
		var u unitUpgrade
		err := rows.Scan(&u.ID, &u.UnitKey, &u.Name, &u.Description, &u.Cost, &u.StarReq, &u.Effect, &u.Value)
		if err != nil {
			continue
		}
		upgrades = append(upgrades, u)
	}
	return upgrades
}

// ===== Synergy/Traits System =====
type traitThreshold struct {
	Count   int             `json:"count"`
	Bonus   string          `json:"bonus"`
	Effects json.RawMessage `json:"effects"`
}

type traitDefinition struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Icon        string           `json:"icon"`
	Thresholds  []traitThreshold `json:"thresholds"`
}

type activeSynergy struct {
	TraitID       string `json:"trait_id"`
	Count         int    `json:"count"`
	Threshold     int    `json:"threshold"`
	NextThreshold int    `json:"next_threshold"`
	IsActive      bool   `json:"is_active"`
	Bonus         string `json:"bonus"`
}

// loadTraitDefinitions loads all trait definitions from the database
func (b *Bot) loadTraitDefinitions() []traitDefinition {
	rows, err := b.DB.Query(`
		SELECT id, name, description, icon, thresholds
		FROM trait_definitions
		ORDER BY id
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var traits []traitDefinition
	for rows.Next() {
		var t traitDefinition
		var thresholdsJSON []byte
		err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Icon, &thresholdsJSON)
		if err != nil {
			continue
		}
		_ = json.Unmarshal(thresholdsJSON, &t.Thresholds)
		traits = append(traits, t)
	}
	return traits
}

// calculateActiveSynergies calculates which synergies are active based on units on the board
func (b *Bot) calculateActiveSynergies(st *tftState) []activeSynergy {
	// Count traits from units on the board (pos >= 0)
	traitCounts := make(map[string]int)
	for _, unit := range st.Units {
		if unit.Pos < 0 {
			continue // Skip bench units
		}
		def, ok := tftDefByKey(unit.Key)
		if !ok {
			continue
		}
		for _, trait := range def.Traits {
			traitCounts[trait]++
		}
	}

	// Load trait definitions
	traits := b.loadTraitDefinitions()

	// Calculate active synergies
	var synergies []activeSynergy
	for _, trait := range traits {
		count := traitCounts[trait.ID]
		if count == 0 {
			continue
		}

		// Find active threshold
		activeThreshold := 0
		nextThreshold := 0
		bonus := ""
		isActive := false

		for i, thresh := range trait.Thresholds {
			if count >= thresh.Count {
				activeThreshold = thresh.Count
				bonus = thresh.Bonus
				isActive = true
				// Set next threshold if there is one
				if i+1 < len(trait.Thresholds) {
					nextThreshold = trait.Thresholds[i+1].Count
				}
			} else if nextThreshold == 0 {
				nextThreshold = thresh.Count
			}
		}

		synergies = append(synergies, activeSynergy{
			TraitID:       trait.ID,
			Count:         count,
			Threshold:     activeThreshold,
			NextThreshold: nextThreshold,
			IsActive:      isActive,
			Bonus:         bonus,
		})
	}

	return synergies
}

// ===== Item System =====

// ItemStats represents the stat bonuses provided by an item
type ItemStats struct {
	HP         int     `json:"hp"`
	ATK        int     `json:"atk"`
	DEF        int     `json:"def"`
	MDEF       int     `json:"mdef"`
	SPD        int     `json:"spd"`
	CritChance float64 `json:"crit_chance"`
	CritDamage float64 `json:"crit_damage"`
	Lifesteal  float64 `json:"lifesteal"`
	Dodge      float64 `json:"dodge"`
}

// ItemEffect represents a special effect triggered by an item
type ItemEffect struct {
	Type     string  `json:"type"`     // "on_hit", "on_kill", "passive", "active"
	Trigger  string  `json:"trigger"`  // "attack", "damage_taken", "kill", etc.
	Chance   float64 `json:"chance"`   // 0-100
	Cooldown int     `json:"cooldown"` // Ticks between triggers
	Effect   string  `json:"effect"`   // "damage", "heal", "stun", "buff", etc.
	Value    int     `json:"value"`    // Effect strength
	Duration int     `json:"duration"` // Duration in ticks
}

// ItemComponent represents a basic item component
type ItemComponent struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Tier        int       `json:"tier"` // 1-3 (basic, advanced, rare)
	Stats       ItemStats `json:"stats"`
}

// CraftedItem represents a crafted item with special effects
type CraftedItem struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Icon        string       `json:"icon"`
	Tier        int          `json:"tier"`       // 4-5 (epic, legendary)
	Components  []string     `json:"components"` // Component IDs used
	Stats       ItemStats    `json:"stats"`
	Effects     []ItemEffect `json:"effects"`
}

// CraftingRecipe defines how to craft an item
type CraftingRecipe struct {
	ID         string   `json:"id"`
	ResultID   string   `json:"result_id"`  // Crafted item ID
	Components []string `json:"components"` // Required component IDs
	GoldCost   int      `json:"gold_cost"`  // Crafting cost
}

// PlayerItem represents an item instance owned by a player
type PlayerItem struct {
	ID         string `json:"id"`
	ClientUID  string `json:"client_uid"`
	ItemID     string `json:"item_id"`
	ItemType   string `json:"item_type"`   // "component" or "crafted"
	EquippedTo string `json:"equipped_to"` // Unit ID if equipped
	CreatedAt  string `json:"created_at"`
}

// loadItemComponents loads all item components from the database
func (b *Bot) loadItemComponents() []ItemComponent {
	rows, err := b.DB.Query(`
		SELECT id, name, description, icon, tier, stats
		FROM item_components
		ORDER BY tier, id
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var components []ItemComponent
	for rows.Next() {
		var c ItemComponent
		var statsJSON []byte
		err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.Icon, &c.Tier, &statsJSON)
		if err != nil {
			continue
		}
		_ = json.Unmarshal(statsJSON, &c.Stats)
		components = append(components, c)
	}
	return components
}

// loadCraftedItems loads all crafted items from the database
func (b *Bot) loadCraftedItems() []CraftedItem {
	rows, err := b.DB.Query(`
		SELECT id, name, description, icon, tier, components, stats, effects
		FROM crafted_items
		ORDER BY tier, id
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var items []CraftedItem
	for rows.Next() {
		var item CraftedItem
		var componentsJSON, statsJSON, effectsJSON []byte
		err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Icon, &item.Tier,
			&componentsJSON, &statsJSON, &effectsJSON)
		if err != nil {
			continue
		}
		_ = json.Unmarshal(componentsJSON, &item.Components)
		_ = json.Unmarshal(statsJSON, &item.Stats)
		_ = json.Unmarshal(effectsJSON, &item.Effects)
		items = append(items, item)
	}
	return items
}

// loadCraftingRecipes loads all crafting recipes from the database
func (b *Bot) loadCraftingRecipes() []CraftingRecipe {
	rows, err := b.DB.Query(`
		SELECT id, result_id, components, gold_cost
		FROM crafting_recipes
		ORDER BY id
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var recipes []CraftingRecipe
	for rows.Next() {
		var r CraftingRecipe
		var componentsJSON []byte
		err := rows.Scan(&r.ID, &r.ResultID, &componentsJSON, &r.GoldCost)
		if err != nil {
			continue
		}
		_ = json.Unmarshal(componentsJSON, &r.Components)
		recipes = append(recipes, r)
	}
	return recipes
}

// loadPlayerItems loads all items owned by a player
func (b *Bot) loadPlayerItems(uid string) []PlayerItem {
	rows, err := b.DB.Query(`
		SELECT id, client_uid, item_id, item_type, COALESCE(equipped_to, ''), created_at
		FROM player_items
		WHERE client_uid=$1
		ORDER BY created_at DESC
	`, uid)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var items []PlayerItem
	for rows.Next() {
		var item PlayerItem
		err := rows.Scan(&item.ID, &item.ClientUID, &item.ItemID, &item.ItemType, &item.EquippedTo, &item.CreatedAt)
		if err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

// craftItem crafts an item using a recipe
func (b *Bot) craftItem(uid string, recipeID string) error {
	// Load the recipe
	recipes := b.loadCraftingRecipes()
	var recipe *CraftingRecipe
	for i := range recipes {
		if recipes[i].ID == recipeID {
			recipe = &recipes[i]
			break
		}
	}
	if recipe == nil {
		return fmt.Errorf("recipe not found")
	}

	// Check if player has all required components
	playerItems := b.loadPlayerItems(uid)
	componentCounts := make(map[string]int)
	for _, item := range playerItems {
		if item.ItemType == "component" && item.EquippedTo == "" {
			componentCounts[item.ItemID]++
		}
	}

	// Check if we have enough of each component
	requiredCounts := make(map[string]int)
	for _, compID := range recipe.Components {
		requiredCounts[compID]++
	}

	for compID, required := range requiredCounts {
		if componentCounts[compID] < required {
			return fmt.Errorf("insufficient components")
		}
	}

	// Check if player has enough gold
	gold := b.userGold(uid)
	if gold < int64(recipe.GoldCost) {
		return fmt.Errorf("insufficient gold")
	}

	// Deduct gold
	_, err := b.DB.Exec("UPDATE users SET gold = gold - $1 WHERE uid=$2", recipe.GoldCost, uid)
	if err != nil {
		return fmt.Errorf("failed to deduct gold")
	}

	// Remove components from inventory
	componentsToRemove := make(map[string]int)
	for _, compID := range recipe.Components {
		componentsToRemove[compID]++
	}

	for compID, count := range componentsToRemove {
		for i := 0; i < count; i++ {
			_, err := b.DB.Exec(`
				DELETE FROM player_items
				WHERE id = (
					SELECT id FROM player_items
					WHERE client_uid=$1 AND item_id=$2 AND item_type='component' AND equipped_to=''
					LIMIT 1
				)
			`, uid, compID)
			if err != nil {
				return fmt.Errorf("failed to remove component")
			}
		}
	}

	// Add crafted item to inventory
	itemID := fmt.Sprintf("item_%08x", rand.Uint32())
	_, err = b.DB.Exec(`
		INSERT INTO player_items (id, client_uid, item_id, item_type, equipped_to)
		VALUES ($1, $2, $3, 'crafted', '')
	`, itemID, uid, recipe.ResultID)
	if err != nil {
		return fmt.Errorf("failed to add crafted item")
	}

	return nil
}

// equipItem equips an item to a unit
func (b *Bot) equipItem(uid string, playerItemID string, unitID string) error {
	// Load player items
	items := b.loadPlayerItems(uid)
	var playerItem *PlayerItem
	for i := range items {
		if items[i].ID == playerItemID {
			playerItem = &items[i]
			break
		}
	}
	if playerItem == nil {
		return fmt.Errorf("item not found")
	}

	// Check if item is already equipped
	if playerItem.EquippedTo != "" {
		return fmt.Errorf("item is already equipped")
	}

	// Load TFT state to check unit
	st := b.loadTFT(uid)
	var unit *tftUnit
	for i := range st.Units {
		if st.Units[i].ID == unitID {
			unit = &st.Units[i]
			break
		}
	}
	if unit == nil {
		return fmt.Errorf("unit not found")
	}

	// Check if unit already has 3 items (max)
	if len(unit.Items) >= 3 {
		return fmt.Errorf("unit already has maximum items")
	}

	// Equip the item
	_, err := b.DB.Exec(`
		UPDATE player_items SET equipped_to=$1 WHERE id=$2
	`, unitID, playerItemID)
	if err != nil {
		return fmt.Errorf("failed to equip item")
	}

	// Update unit's item list
	unit.Items = append(unit.Items, playerItem.ItemID)
	b.saveTFT(uid, st)

	return nil
}

// unequipItem unequips an item from a unit
func (b *Bot) unequipItem(uid string, playerItemID string) error {
	// Load player items
	items := b.loadPlayerItems(uid)
	var playerItem *PlayerItem
	for i := range items {
		if items[i].ID == playerItemID {
			playerItem = &items[i]
			break
		}
	}
	if playerItem == nil {
		return fmt.Errorf("item not found")
	}

	// Check if item is equipped
	if playerItem.EquippedTo == "" {
		return fmt.Errorf("item is not equipped")
	}

	// Load TFT state to update unit
	st := b.loadTFT(uid)
	for i := range st.Units {
		if st.Units[i].ID == playerItem.EquippedTo {
			// Remove item from unit's list
			newItems := make([]string, 0)
			for _, itemID := range st.Units[i].Items {
				if itemID != playerItem.ItemID {
					newItems = append(newItems, itemID)
				}
			}
			st.Units[i].Items = newItems
			break
		}
	}
	b.saveTFT(uid, st)

	// Unequip the item
	_, err := b.DB.Exec(`
		UPDATE player_items SET equipped_to='' WHERE id=$1
	`, playerItemID)
	if err != nil {
		return fmt.Errorf("failed to unequip item")
	}

	return nil
}

// sellItem sells an item for gold
func (b *Bot) sellItem(uid string, playerItemID string) error {
	// Load player items
	items := b.loadPlayerItems(uid)
	var playerItem *PlayerItem
	for i := range items {
		if items[i].ID == playerItemID {
			playerItem = &items[i]
			break
		}
	}
	if playerItem == nil {
		return fmt.Errorf("item not found")
	}

	// Check if item is equipped
	if playerItem.EquippedTo != "" {
		return fmt.Errorf("cannot sell equipped item")
	}

	// Determine sell value based on item tier
	var sellValue int
	if playerItem.ItemType == "component" {
		components := b.loadItemComponents()
		for _, c := range components {
			if c.ID == playerItem.ItemID {
				sellValue = c.Tier * 10 // 10 gold per tier
				break
			}
		}
	} else {
		craftedItems := b.loadCraftedItems()
		for _, item := range craftedItems {
			if item.ID == playerItem.ItemID {
				sellValue = item.Tier * 25 // 25 gold per tier
				break
			}
		}
	}

	// Add gold to player
	_, err := b.DB.Exec("UPDATE users SET gold = gold + $1 WHERE uid=$2", sellValue, uid)
	if err != nil {
		return fmt.Errorf("failed to add gold")
	}

	// Remove item from inventory
	_, err = b.DB.Exec("DELETE FROM player_items WHERE id=$1", playerItemID)
	if err != nil {
		return fmt.Errorf("failed to remove item")
	}

	return nil
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
	var battleGold, phaseTimer, roundNumber, stageNumber, xp int
	var phase string
	err := b.DB.QueryRow("SELECT data, COALESCE(battle_gold, 0), COALESCE(phase, 'planning'), COALESCE(phase_timer, 30), COALESCE(round_number, 1), COALESCE(stage_number, 1), COALESCE(xp, 0) FROM tft_state WHERE client_uid=$1", uid).Scan(&raw, &battleGold, &phase, &phaseTimer, &roundNumber, &stageNumber, &xp)
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
		st.XP = 0 // Start with no XP
		// Initialize board configuration
		st.GridType = "hex"                         // Default to hex grid
		st.ObstaclePositions = []int{5, 12, 18}     // Example obstacle positions
		st.UnitUpgrades = make(map[string][]string) // Initialize unit upgrades
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
	st.XP = xp
	if len(st.Shop) != tftShopSize {
		st.Shop = rollShop()
	}
	// Ensure phase is valid
	if st.Phase != "planning" && st.Phase != "combat" && st.Phase != "overtime" {
		st.Phase = "planning"
	}
	// Initialize UnitUpgrades if nil
	if st.UnitUpgrades == nil {
		st.UnitUpgrades = make(map[string][]string)
	}
	return st
}

func (b *Bot) saveTFT(uid string, st *tftState) {
	data, _ := json.Marshal(st)
	_, _ = b.DB.Exec(
		`INSERT INTO tft_state (client_uid, data, battle_gold, phase, phase_timer, round_number, stage_number, xp, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		 ON CONFLICT (client_uid) DO UPDATE SET
		 data=$2, battle_gold=$3, phase=$4, phase_timer=$5, round_number=$6, stage_number=$7, xp=$8, updated_at=NOW()`,
		uid, data, st.BattleGold, st.Phase, st.PhaseTimer, st.RoundNumber, st.StageNumber, st.XP)
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
		"History":           s.bot.battleHistory(uid, 12),
		"Leaders":           s.bot.gameLeaderboards("tft"),
		"Inventory":         s.bot.inventoryItems(uid),
		"BattleStats":       s.bot.getBattleStats(uid),
		"BattleGold":        st.BattleGold,
		"Phase":             st.Phase,
		"PhaseTimer":        st.PhaseTimer,
		"RoundNumber":       st.RoundNumber,
		"StageNumber":       st.StageNumber,
		"GridType":          st.GridType,
		"ObstaclePositions": jsonJS(st.ObstaclePositions),
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

// handleTFTGetXP retrieves current XP for the player
func (s *WebServer) handleTFTGetXP(w http.ResponseWriter, r *http.Request, uid string) {
	st := s.bot.loadTFT(uid)
	writeJSON(w, map[string]any{"ok": true, "xp": st.XP})
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

// handleTFTRefreshShop refreshes the shop using XP
func (s *WebServer) handleTFTRefreshShop(w http.ResponseWriter, r *http.Request, uid string) {
	st := s.bot.loadTFT(uid)

	// XP cost for refresh (2 XP)
	if st.XP < 2 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough XP"})
		return
	}

	st.XP -= 2
	st.Shop = rollShop()
	s.bot.saveTFT(uid, st)
	writeJSON(w, map[string]any{"ok": true, "xp": st.XP, "shop": st.Shop})
}

// handleTFTUnitInfo returns detailed information about a unit including abilities and upgrades
func (s *WebServer) handleTFTUnitInfo(w http.ResponseWriter, r *http.Request, uid string) {
	var req struct {
		UnitKey string `json:"unit_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	// Load unit definition
	def, ok := tftDefByKey(req.UnitKey)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unit not found"})
		return
	}

	// Load abilities and upgrades
	abilities := s.bot.loadUnitAbilities(req.UnitKey)
	upgrades := s.bot.loadUnitUpgrades(req.UnitKey)

	// Get player's unit upgrades
	st := s.bot.loadTFT(uid)
	playerUpgrades := make(map[string][]string)
	if st.UnitUpgrades != nil {
		playerUpgrades = st.UnitUpgrades
	}

	writeJSON(w, map[string]any{
		"ok": true,
		"unit": map[string]any{
			"key":    def.Key,
			"name":   def.Name,
			"icon":   def.Icon,
			"cost":   def.Cost,
			"hp":     def.HP,
			"atk":    def.ATK,
			"range":  def.Rng,
			"traits": def.Traits,
		},
		"abilities":       abilities,
		"upgrades":        upgrades,
		"player_upgrades": playerUpgrades,
	})
}

// handleTFTUpgradeUnit upgrades a unit with a specific upgrade
func (s *WebServer) handleTFTUpgradeUnit(w http.ResponseWriter, r *http.Request, uid string) {
	var req struct {
		UnitID    string `json:"unit_id"`
		UpgradeID string `json:"upgrade_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	st := s.bot.loadTFT(uid)

	// Find the unit
	var unit *tftUnit
	for i := range st.Units {
		if st.Units[i].ID == req.UnitID {
			unit = &st.Units[i]
			break
		}
	}
	if unit == nil {
		writeJSON(w, map[string]any{"ok": false, "error": "unit not found"})
		return
	}

	// Load upgrade definition
	upgrades := s.bot.loadUnitUpgrades(unit.Key)
	var upgrade *unitUpgrade
	for i := range upgrades {
		if upgrades[i].ID == req.UpgradeID {
			upgrade = &upgrades[i]
			break
		}
	}
	if upgrade == nil {
		writeJSON(w, map[string]any{"ok": false, "error": "upgrade not found"})
		return
	}

	// Check star requirement
	if unit.Star < upgrade.StarReq {
		writeJSON(w, map[string]any{"ok": false, "error": "unit star level too low"})
		return
	}

	// Check if already has this upgrade
	if st.UnitUpgrades == nil {
		st.UnitUpgrades = make(map[string][]string)
	}
	for _, upID := range st.UnitUpgrades[req.UnitID] {
		if upID == req.UpgradeID {
			writeJSON(w, map[string]any{"ok": false, "error": "upgrade already applied"})
			return
		}
	}

	// Check gold cost
	if st.BattleGold < upgrade.Cost {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough battle gold"})
		return
	}

	// Apply upgrade
	st.BattleGold -= upgrade.Cost
	st.UnitUpgrades[req.UnitID] = append(st.UnitUpgrades[req.UnitID], req.UpgradeID)
	s.bot.saveTFT(uid, st)

	writeJSON(w, map[string]any{
		"ok":          true,
		"battle_gold": st.BattleGold,
		"upgrades":    st.UnitUpgrades[req.UnitID],
	})
}

// handleTFTSynergies returns the current active synergies
func (s *WebServer) handleTFTSynergies(w http.ResponseWriter, r *http.Request, uid string) {
	st := s.bot.loadTFT(uid)
	synergies := s.bot.calculateActiveSynergies(st)
	writeJSON(w, map[string]any{
		"ok":        true,
		"synergies": synergies,
	})
}

// ===== Item System API Handlers =====

// handleTFTItems returns the player's item inventory
func (s *WebServer) handleTFTItems(w http.ResponseWriter, r *http.Request, uid string) {
	items := s.bot.loadPlayerItems(uid)
	components := s.bot.loadItemComponents()
	craftedItems := s.bot.loadCraftedItems()

	// Build item details
	type itemDetail struct {
		PlayerItem
		Name        string       `json:"name"`
		Description string       `json:"description"`
		Icon        string       `json:"icon"`
		Tier        int          `json:"tier"`
		Stats       ItemStats    `json:"stats"`
		Effects     []ItemEffect `json:"effects,omitempty"`
	}

	var details []itemDetail
	for _, item := range items {
		var detail itemDetail
		detail.PlayerItem = item

		if item.ItemType == "component" {
			for _, c := range components {
				if c.ID == item.ItemID {
					detail.Name = c.Name
					detail.Description = c.Description
					detail.Icon = c.Icon
					detail.Tier = c.Tier
					detail.Stats = c.Stats
					break
				}
			}
		} else {
			for _, ci := range craftedItems {
				if ci.ID == item.ItemID {
					detail.Name = ci.Name
					detail.Description = ci.Description
					detail.Icon = ci.Icon
					detail.Tier = ci.Tier
					detail.Stats = ci.Stats
					detail.Effects = ci.Effects
					break
				}
			}
		}
		details = append(details, detail)
	}

	writeJSON(w, map[string]any{
		"ok":    true,
		"items": details,
	})
}

// handleTFTCraftItem crafts an item using a recipe
func (s *WebServer) handleTFTCraftItem(w http.ResponseWriter, r *http.Request, uid string) {
	var req struct {
		RecipeID string `json:"recipe_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	err := s.bot.craftItem(uid, req.RecipeID)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	writeJSON(w, map[string]any{"ok": true})
}

// handleTFTEquipItem equips an item to a unit
func (s *WebServer) handleTFTEquipItem(w http.ResponseWriter, r *http.Request, uid string) {
	var req struct {
		PlayerItemID string `json:"player_item_id"`
		UnitID       string `json:"unit_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	err := s.bot.equipItem(uid, req.PlayerItemID, req.UnitID)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	writeJSON(w, map[string]any{"ok": true})
}

// handleTFTUnequipItem unequips an item from a unit
func (s *WebServer) handleTFTUnequipItem(w http.ResponseWriter, r *http.Request, uid string) {
	var req struct {
		PlayerItemID string `json:"player_item_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	err := s.bot.unequipItem(uid, req.PlayerItemID)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	writeJSON(w, map[string]any{"ok": true})
}

// handleTFTSellItem sells an item for gold
func (s *WebServer) handleTFTSellItem(w http.ResponseWriter, r *http.Request, uid string) {
	var req struct {
		PlayerItemID string `json:"player_item_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	err := s.bot.sellItem(uid, req.PlayerItemID)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	writeJSON(w, map[string]any{"ok": true})
}

// handleTFTRecipes returns all available crafting recipes
func (s *WebServer) handleTFTRecipes(w http.ResponseWriter, r *http.Request, uid string) {
	recipes := s.bot.loadCraftingRecipes()
	components := s.bot.loadItemComponents()
	craftedItems := s.bot.loadCraftedItems()

	// Build recipe details
	type recipeDetail struct {
		CraftingRecipe
		ResultName       string    `json:"result_name"`
		ResultIcon       string    `json:"result_icon"`
		ResultTier       int       `json:"result_tier"`
		ResultStats      ItemStats `json:"result_stats"`
		ComponentDetails []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Icon string `json:"icon"`
		} `json:"component_details"`
	}

	var details []recipeDetail
	for _, recipe := range recipes {
		var detail recipeDetail
		detail.CraftingRecipe = recipe

		// Find result item
		for _, ci := range craftedItems {
			if ci.ID == recipe.ResultID {
				detail.ResultName = ci.Name
				detail.ResultIcon = ci.Icon
				detail.ResultTier = ci.Tier
				detail.ResultStats = ci.Stats
				break
			}
		}

		// Find component details
		for _, compID := range recipe.Components {
			for _, c := range components {
				if c.ID == compID {
					detail.ComponentDetails = append(detail.ComponentDetails, struct {
						ID   string `json:"id"`
						Name string `json:"name"`
						Icon string `json:"icon"`
					}{
						ID:   c.ID,
						Name: c.Name,
						Icon: c.Icon,
					})
					break
				}
			}
		}
		details = append(details, detail)
	}

	writeJSON(w, map[string]any{
		"ok":      true,
		"recipes": details,
	})
}

// ===== UI/UX System API Handlers =====

// PlayerSettings represents player UI/UX preferences
type PlayerSettings struct {
	SoundEnabled      bool `json:"sound_enabled"`
	MusicVolume       int  `json:"music_volume"`
	AnimationsEnabled bool `json:"animations_enabled"`
	ParticlesEnabled  bool `json:"particles_enabled"`
	AutoCombat        bool `json:"auto_combat"`
	CombatSpeed       int  `json:"combat_speed"`
}

// loadPlayerSettings loads player settings from the database
func (b *Bot) loadPlayerSettings(uid string) PlayerSettings {
	var settingsJSON []byte
	err := b.DB.QueryRow("SELECT settings FROM player_settings WHERE client_uid=$1", uid).Scan(&settingsJSON)
	if err != nil {
		// Return default settings
		return PlayerSettings{
			SoundEnabled:      true,
			MusicVolume:       50,
			AnimationsEnabled: true,
			ParticlesEnabled:  true,
			AutoCombat:        true,
			CombatSpeed:       1,
		}
	}

	var settings PlayerSettings
	_ = json.Unmarshal(settingsJSON, &settings)
	return settings
}

// savePlayerSettings saves player settings to the database
func (b *Bot) savePlayerSettings(uid string, settings PlayerSettings) error {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	_, err = b.DB.Exec(`
		INSERT INTO player_settings (client_uid, settings, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (client_uid) DO UPDATE SET settings=$2, updated_at=NOW()
	`, uid, settingsJSON)
	return err
}

// handleTFTSettings returns the player's settings
func (s *WebServer) handleTFTSettings(w http.ResponseWriter, r *http.Request, uid string) {
	settings := s.bot.loadPlayerSettings(uid)
	writeJSON(w, map[string]any{
		"ok":       true,
		"settings": settings,
	})
}

// handleTFTSaveSettings saves the player's settings
func (s *WebServer) handleTFTSaveSettings(w http.ResponseWriter, r *http.Request, uid string) {
	var settings PlayerSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	// Validate settings
	if settings.MusicVolume < 0 || settings.MusicVolume > 100 {
		settings.MusicVolume = 50
	}
	if settings.CombatSpeed < 1 || settings.CombatSpeed > 4 {
		settings.CombatSpeed = 1
	}

	err := s.bot.savePlayerSettings(uid, settings)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "failed to save settings"})
		return
	}

	writeJSON(w, map[string]any{"ok": true})
}

// handleTFTScoreboard returns the current scoreboard data
func (s *WebServer) handleTFTScoreboard(w http.ResponseWriter, r *http.Request, uid string) {
	// Get top players from battle history
	rows, err := s.bot.DB.Query(`
		SELECT u.username, COALESCE(SUM(bh.damage_dealt), 0) as total_damage,
		       COUNT(CASE WHEN bh.victory THEN 1 END) as wins,
		       MAX(bh.wave_number) as highest_wave
		FROM users u
		LEFT JOIN battle_history bh ON u.client_uid = bh.client_uid
		WHERE u.client_uid IN (
			SELECT DISTINCT client_uid FROM battle_history
		)
		GROUP BY u.username
		ORDER BY total_damage DESC
		LIMIT 10
	`)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "failed to load scoreboard"})
		return
	}
	defer rows.Close()

	type playerScore struct {
		Rank        int    `json:"rank"`
		Name        string `json:"name"`
		TotalDamage int    `json:"total_damage"`
		Wins        int    `json:"wins"`
		HighestWave int    `json:"highest_wave"`
	}

	var players []playerScore
	rank := 1
	for rows.Next() {
		var p playerScore
		err := rows.Scan(&p.Name, &p.TotalDamage, &p.Wins, &p.HighestWave)
		if err != nil {
			continue
		}
		p.Rank = rank
		rank++
		players = append(players, p)
	}

	writeJSON(w, map[string]any{
		"ok":      true,
		"players": players,
	})
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
	st := s.bot.loadTFT(uid)

	if req.Pos >= 0 && !isPlayerCell(req.Pos) {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid cell"})
		return
	}

	// Check if the position is an obstacle (blocked position)
	for _, obs := range st.ObstaclePositions {
		if req.Pos == obs {
			writeJSON(w, map[string]any{"ok": false, "error": "position blocked by obstacle"})
			return
		}
	}
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

	// Apply streak bonus (10% per streak, max 50%)
	if st.RoundNumber%5 == 0 {
		streakBonus := minInt(st.RoundNumber/5, 5) * 10
		battleGoldAward += battleGoldAward * streakBonus / 100
		st.BattleGold += battleGoldAward
	}

	// Apply interest (5 gold for every 10 battle gold, max 50)
	interest := st.BattleGold / 10 * 5
	if interest > 50 {
		interest = 50
	}
	st.BattleGold += interest

	// Enforce gold cap (max 100 battle gold)
	if st.BattleGold > 100 {
		st.BattleGold = 100
	}
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
	From       string     `json:"from"`
	To         string     `json:"to"`
	Dmg        int        `json:"dmg"`
	DamageType DamageType `json:"damage_type,omitempty"`
	IsCrit     bool       `json:"is_crit,omitempty"`
	IsKill     bool       `json:"is_kill,omitempty"`
	Effect     string     `json:"effect,omitempty"` // Status effect applied
	Value      int        `json:"value,omitempty"`  // Effect value
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

	// Enhanced combat stats
	def     int // Physical defense
	mdef    int // Magic defense
	spd     int // Speed (affects movement and attack speed)
	mana    int // Current mana
	maxMana int // Max mana for abilities

	// Status effects (ticks remaining)
	stunned int // Cannot move or attack
	slowed  int // Movement speed reduced
	burning int // Takes damage over time
	frozen  int // Cannot move, takes extra damage

	// Item effects
	items       []string     // Item IDs equipped
	itemEffects []ItemEffect // Active item effects

	// Combat tracking
	kills       int // Number of kills
	damageTaken int // Total damage taken
	healingDone int // Total healing done
}

// StatusEffect represents a status effect applied to a unit
type StatusEffect struct {
	Type     string `json:"type"`     // "stun", "slow", "burn", "freeze"
	Duration int    `json:"duration"` // Ticks remaining
	Value    int    `json:"value"`    // Effect strength (for DOT)
}

// DamageType represents the type of damage dealt
type DamageType string

const (
	DamageTypePhysical DamageType = "physical"
	DamageTypeMagic    DamageType = "magic"
	DamageTypeTrue     DamageType = "true"
)

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

	// Award XP for combat
	xpEarned := 1 + roundNumber/5 // More XP for later rounds
	if victory {
		xpEarned += 2 // Bonus XP for victory
	}
	_, _ = s.bot.DB.Exec("UPDATE users SET xp = xp + $1 WHERE client_uid=$2", xpEarned, uid)

	// Also award XP to tftState for shop refreshes
	st.XP += xpEarned

	s.bot.recordGameResult(uid, "tft", victory, res.GoldWon)

	res.Gold = s.bot.userGold(uid)
	res.HighestWave = roundNumber
	res.Streak = s.bot.getCurrentStreak(uid)

	// Advance to next round/stage and reset phase
	s.advanceRound(st)

	s.bot.saveTFT(uid, st) // Save battle gold state and phase

	// Save combat log
	s.saveCombatLog(uid, st, victory, damageDealt, turnsSurvived, frames)

	writeJSON(w, res)
}

// saveCombatLog saves combat statistics to the database
func (s *WebServer) saveCombatLog(uid string, st *tftState, victory bool, damageDealt int, turnsSurvived int, frames []tftFrame) {
	// Count units survived and enemies killed
	unitsSurvived := 0
	enemiesKilled := 0

	for _, u := range st.Units {
		if u.Pos >= 0 { // On board
			unitsSurvived++
		}
	}

	// Estimate enemies killed based on damage dealt
	// This is a rough estimate since we don't track individual enemy deaths
	enemiesKilled = damageDealt / 500 // Assume ~500 HP per enemy on average

	// Update combat statistics
	st.TotalDamageDealt += damageDealt
	st.TotalDamageTaken += turnsSurvived * 10 // Estimate damage taken
	st.TotalEnemiesKilled += enemiesKilled
	st.CombatCount++

	// Serialize frames to JSON
	framesJSON, err := json.Marshal(frames)
	if err != nil {
		framesJSON = []byte("[]")
	}

	// Insert combat log
	logID := fmt.Sprintf("combat_%d_%d_%d", st.StageNumber, st.RoundNumber, time.Now().Unix())
	_, _ = s.bot.DB.Exec(`
		INSERT INTO combat_logs (id, client_uid, round_number, stage_number, won, damage_dealt, damage_taken, units_survived, enemies_killed, frames)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, logID, uid, st.RoundNumber, st.StageNumber, victory, damageDealt, turnsSurvived*10, unitsSurvived, enemiesKilled, framesJSON)
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

// calculateDamage calculates damage with defense reduction
func calculateDamage(atk int, def int, damageType DamageType) int {
	if damageType == DamageTypeTrue {
		return atk // True damage ignores defense
	}

	// Defense reduces damage: damage = atk * (100 / (100 + def))
	reduction := 100.0 / (100.0 + float64(def))
	return int(float64(atk) * reduction)
}

// applyStatusEffect applies a status effect to a unit
func applyStatusEffect(u *simUnit, effectType string, duration int, value int) {
	switch effectType {
	case "stun":
		u.stunned = maxInt(u.stunned, duration)
	case "slow":
		u.slowed = maxInt(u.slowed, duration)
	case "burn":
		u.burning = maxInt(u.burning, duration)
	case "freeze":
		u.frozen = maxInt(u.frozen, duration)
	}
}

// processStatusEffects processes status effects at the start of each tick
func processStatusEffects(units []*simUnit) []tftEvent {
	var events []tftEvent

	for _, u := range units {
		if u.hp <= 0 {
			continue
		}

		// Process burning damage
		if u.burning > 0 {
			burnDmg := 20 // Base burn damage
			u.hp -= burnDmg
			u.damageTaken += burnDmg
			if u.hp < 0 {
				u.hp = 0
			}
			events = append(events, tftEvent{
				From:       "burn",
				To:         u.id,
				Dmg:        burnDmg,
				DamageType: DamageTypeMagic,
				Effect:     "burn",
			})
			u.burning--
		}

		// Decrement other status effects
		if u.stunned > 0 {
			u.stunned--
		}
		if u.slowed > 0 {
			u.slowed--
		}
		if u.frozen > 0 {
			u.frozen--
		}
	}

	return events
}

// triggerItemEffects triggers item effects based on the event type
func triggerItemEffects(u *simUnit, target *simUnit, eventType string, r *rand.Rand) []tftEvent {
	var events []tftEvent

	for _, effect := range u.itemEffects {
		if effect.Trigger != eventType {
			continue
		}

		// Check chance
		if r.IntN(100) >= int(effect.Chance) {
			continue
		}

		// Apply effect
		switch effect.Effect {
		case "damage":
			target.hp -= effect.Value
			target.damageTaken += effect.Value
			if target.hp < 0 {
				target.hp = 0
			}
			events = append(events, tftEvent{
				From:   u.id,
				To:     target.id,
				Dmg:    effect.Value,
				Effect: "item_damage",
			})

		case "heal":
			heal := effect.Value
			u.hp = minInt(u.hp+heal, u.maxhp)
			u.healingDone += heal
			events = append(events, tftEvent{
				From:   u.id,
				To:     u.id,
				Dmg:    -heal, // Negative damage = healing
				Effect: "item_heal",
			})

		case "heal_percent":
			heal := u.maxhp * effect.Value / 100
			u.hp = minInt(u.hp+heal, u.maxhp)
			u.healingDone += heal
			events = append(events, tftEvent{
				From:   u.id,
				To:     u.id,
				Dmg:    -heal,
				Effect: "item_heal_percent",
			})

		case "buff":
			u.atk += effect.Value
			events = append(events, tftEvent{
				From:   u.id,
				To:     u.id,
				Effect: "item_buff_atk",
				Value:  effect.Value,
			})

		case "buff_def":
			u.def += effect.Value
			events = append(events, tftEvent{
				From:   u.id,
				To:     u.id,
				Effect: "item_buff_def",
				Value:  effect.Value,
			})

		case "stun":
			applyStatusEffect(target, "stun", effect.Duration, 0)
			events = append(events, tftEvent{
				From:   u.id,
				To:     target.id,
				Effect: "item_stun",
				Value:  effect.Duration,
			})

		case "slow":
			applyStatusEffect(target, "slow", effect.Duration, 0)
			events = append(events, tftEvent{
				From:   u.id,
				To:     target.id,
				Effect: "item_slow",
				Value:  effect.Duration,
			})

		case "burn":
			applyStatusEffect(target, "burn", effect.Duration, effect.Value)
			events = append(events, tftEvent{
				From:   u.id,
				To:     target.id,
				Effect: "item_burn",
				Value:  effect.Duration,
			})

		case "true_damage":
			target.hp -= effect.Value
			target.damageTaken += effect.Value
			if target.hp < 0 {
				target.hp = 0
			}
			events = append(events, tftEvent{
				From:       u.id,
				To:         target.id,
				Dmg:        effect.Value,
				DamageType: DamageTypeTrue,
				Effect:     "item_true_damage",
			})

		case "magic_damage":
			dmg := calculateDamage(effect.Value, target.mdef, DamageTypeMagic)
			target.hp -= dmg
			target.damageTaken += dmg
			if target.hp < 0 {
				target.hp = 0
			}
			events = append(events, tftEvent{
				From:       u.id,
				To:         target.id,
				Dmg:        dmg,
				DamageType: DamageTypeMagic,
				Effect:     "item_magic_damage",
			})
		}
	}

	return events
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

		// Process status effects at the start of each tick
		statusEvents := processStatusEffects(units)
		events = append(events, statusEvents...)

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

			// Skip if stunned or frozen
			if u.stunned > 0 || u.frozen > 0 {
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
					// Calculate base damage with defense
					dmg := calculateDamage(u.atk, tgt.def, DamageTypePhysical)

					// Check for critical hit
					isCrit := false
					if r.IntN(100) < u.critChance {
						dmg = int(float64(dmg) * u.critDmg)
						isCrit = true
					}

					// Apply damage reduction from traits
					if tgt.dmgRed > 0 {
						dmg = int(float64(dmg) * (1.0 - tgt.dmgRed))
					}

					// Frozen targets take 25% extra damage
					if tgt.frozen > 0 {
						dmg = int(float64(dmg) * 1.25)
					}

					tgt.hp -= dmg
					tgt.damageTaken += dmg
					if tgt.hp < 0 {
						tgt.hp = 0
					}

					// Check if target died
					isKill := tgt.hp == 0
					if isKill {
						u.kills++
						// Trigger on_kill item effects
						killEvents := triggerItemEffects(u, tgt, "kill", r)
						events = append(events, killEvents...)
					}

					events = append(events, tftEvent{
						From:       u.id,
						To:         tgt.id,
						Dmg:        dmg,
						DamageType: DamageTypePhysical,
						IsCrit:     isCrit,
						IsKill:     isKill,
					})
					u.cd = attackCooldown

					// Lifesteal: heal on attack
					if u.lifesteal > 0 {
						heal := int(float64(dmg) * u.lifesteal)
						if heal > 0 {
							u.hp = minInt(u.hp+heal, u.maxhp)
							u.healingDone += heal
						}
					}

					// Trigger on_hit item effects
					hitEvents := triggerItemEffects(u, tgt, "attack", r)
					events = append(events, hitEvents...)

					// Track damage dealt by player units
					if u.side == "you" {
						u.damageDealt += dmg
						totalDamage += dmg
					}
				}
			} else {
				// Movement: check if slowed (move half as often)
				canMove := true
				if u.slowed > 0 && tick%2 == 0 {
					canMove = false
				}

				if canMove {
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

// ===== Augment System =====

// Augment represents an augment definition
type Augment struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tier        int    `json:"tier"`
	Type        string `json:"type"`
	EffectData  string `json:"effect_data"`
	Icon        string `json:"icon"`
}

// AugmentOffer represents a current augment offer for selection
type AugmentOffer struct {
	ID         string `json:"id"`
	Key        string `json:"key"`
	Name       string `json:"name"`
	Desc       string `json:"description"`
	Tier       int    `json:"tier"`
	Type       string `json:"type"`
	Icon       string `json:"icon"`
	OfferIndex int    `json:"offer_index"`
}

// AugmentState tracks player augment state
type AugmentState struct {
	RerollCount    int `json:"reroll_count"`
	LastOfferStage int `json:"last_offer_stage"`
	LastOfferRound int `json:"last_offer_round"`
	Selected       int `json:"augments_selected"`
}

// loadAugments loads all augment definitions from the database
func (b *Bot) loadAugments() []Augment {
	var augments []Augment
	rows, err := b.DB.Query(`
		SELECT id, key, name, description, tier, type, effect_data, icon
		FROM tft_augments ORDER BY tier, name
	`)
	if err != nil {
		return augments
	}
	defer rows.Close()

	for rows.Next() {
		var a Augment
		if err := rows.Scan(&a.ID, &a.Key, &a.Name, &a.Description, &a.Tier, &a.Type, &a.EffectData, &a.Icon); err == nil {
			augments = append(augments, a)
		}
	}
	return augments
}

// loadAugmentsByTier loads augments filtered by tier
func (b *Bot) loadAugmentsByTier(tier int) []Augment {
	var augments []Augment
	rows, err := b.DB.Query(`
		SELECT id, key, name, description, tier, type, effect_data, icon
		FROM tft_augments WHERE tier = $1 ORDER BY RANDOM()
	`, tier)
	if err != nil {
		return augments
	}
	defer rows.Close()

	for rows.Next() {
		var a Augment
		if err := rows.Scan(&a.ID, &a.Key, &a.Name, &a.Description, &a.Tier, &a.Type, &a.EffectData, &a.Icon); err == nil {
			augments = append(augments, a)
		}
	}
	return augments
}

// getAugmentState returns the player's augment state
func (b *Bot) getAugmentState(uid string) AugmentState {
	var state AugmentState
	err := b.DB.QueryRow(`
		SELECT COALESCE(reroll_count, 0), COALESCE(last_offer_stage, 0),
		       COALESCE(last_offer_round, 0), COALESCE(augments_selected, 0)
		FROM tft_augment_state WHERE user_id = $1
	`, uid).Scan(&state.RerollCount, &state.LastOfferStage, &state.LastOfferRound, &state.Selected)
	if err != nil {
		return AugmentState{}
	}
	return state
}

// generateAugmentOffers generates 3 random augment offers for the current stage
func (b *Bot) generateAugmentOffers(uid string, stage, round int) []AugmentOffer {
	// Determine tier based on stage
	tier := 1
	if stage >= 4 {
		tier = 3
	} else if stage >= 3 {
		tier = 2
	}

	// Get augments for this tier
	augments := b.loadAugmentsByTier(tier)
	if len(augments) < 3 {
		// Fallback to all augments if not enough in tier
		augments = b.loadAugments()
	}

	// Shuffle and pick 3
	rand.Shuffle(len(augments), func(i, j int) {
		augments[i], augments[j] = augments[j], augments[i]
	})

	var offers []AugmentOffer
	limit := 3
	if len(augments) < limit {
		limit = len(augments)
	}

	for i := 0; i < limit; i++ {
		offers = append(offers, AugmentOffer{
			ID:         augments[i].ID,
			Key:        augments[i].Key,
			Name:       augments[i].Name,
			Desc:       augments[i].Description,
			Tier:       augments[i].Tier,
			Type:       augments[i].Type,
			Icon:       augments[i].Icon,
			OfferIndex: i,
		})

		// Save offer to database
		_, _ = b.DB.Exec(`
			INSERT INTO tft_augment_offers (user_id, offer_index, augment_id, stage, round)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (user_id, offer_index, stage, round) DO UPDATE
			SET augment_id = EXCLUDED.augment_id
		`, uid, i, augments[i].ID, stage, round)
	}

	// Update augment state
	_, _ = b.DB.Exec(`
		INSERT INTO tft_augment_state (user_id, last_offer_stage, last_offer_round)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE
		SET last_offer_stage = EXCLUDED.last_offer_stage, last_offer_round = EXCLUDED.last_offer_round
	`, uid, stage, round)

	return offers
}

// getRerollCost calculates the cost for rerolling augments
func getRerollCost(rerollCount int) int {
	if rerollCount == 0 {
		return 0 // First reroll is free
	}
	return 5 * rerollCount // Subsequent rerolls cost gold
}

// handleTFTAugments returns the current augment offers
func (s *WebServer) handleTFTAugments(w http.ResponseWriter, r *http.Request, uid string) {
	state := s.bot.getAugmentState(uid)
	st := s.bot.loadTFT(uid)

	// Check if we need to generate new offers
	if state.LastOfferStage != st.StageNumber || state.LastOfferRound != st.RoundNumber {
		offers := s.bot.generateAugmentOffers(uid, st.StageNumber, st.RoundNumber)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"offers":     offers,
			"rerollCost": getRerollCost(state.RerollCount),
			"canReroll":  true,
		}); err != nil {
			http.Error(w, "Failed to encode response", 500)
		}
		return
	}

	// Load existing offers
	rows, err := s.bot.DB.Query(`
		SELECT o.offer_index, a.id, a.key, a.name, a.description, a.tier, a.type, a.icon
		FROM tft_augment_offers o
		JOIN tft_augments a ON o.augment_id = a.id
		WHERE o.user_id = $1 AND o.stage = $2 AND o.round = $3
		ORDER BY o.offer_index
	`, uid, st.StageNumber, st.RoundNumber)
	if err != nil {
		http.Error(w, "Failed to load offers", 500)
		return
	}
	defer rows.Close()

	var offers []AugmentOffer
	for rows.Next() {
		var o AugmentOffer
		if err := rows.Scan(&o.OfferIndex, &o.ID, &o.Key, &o.Name, &o.Desc, &o.Tier, &o.Type, &o.Icon); err == nil {
			offers = append(offers, o)
		}
	}

	if len(offers) == 0 {
		offers = s.bot.generateAugmentOffers(uid, st.StageNumber, st.RoundNumber)
	}

	if err := json.NewEncoder(w).Encode(map[string]any{
		"offers":     offers,
		"rerollCost": getRerollCost(state.RerollCount),
		"canReroll":  true,
	}); err != nil {
		http.Error(w, "Failed to encode response", 500)
	}
}

// handleTFTSelectAugment processes augment selection
func (s *WebServer) handleTFTSelectAugment(w http.ResponseWriter, r *http.Request, uid string) {
	var req struct {
		AugmentID string `json:"augmentId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	// Get augment details
	var augment Augment
	err := s.bot.DB.QueryRow(`
		SELECT id, key, name, description, tier, type, effect_data, icon
		FROM tft_augments WHERE id = $1
	`, req.AugmentID).Scan(&augment.ID, &augment.Key, &augment.Name, &augment.Description, &augment.Tier, &augment.Type, &augment.EffectData, &augment.Icon)
	if err != nil {
		http.Error(w, "Augment not found", 404)
		return
	}

	st := s.bot.loadTFT(uid)

	// Save player augment selection
	_, err = s.bot.DB.Exec(`
		INSERT INTO tft_player_augments (user_id, augment_id, game_id)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`, uid, augment.ID, fmt.Sprintf("%d-%d", st.StageNumber, st.RoundNumber))
	if err != nil {
		http.Error(w, "Failed to save augment", 500)
		return
	}

	// Apply immediate effects
	s.bot.applyAugmentEffect(uid, augment, st)

	// Update augment state
	_, _ = s.bot.DB.Exec(`
		UPDATE tft_augment_state SET augments_selected = augments_selected + 1
		WHERE user_id = $1
	`, uid)

	// Clear offers after selection
	_, _ = s.bot.DB.Exec(`
		DELETE FROM tft_augment_offers WHERE user_id = $1
	`, uid)

	if err := json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"augment": augment,
	}); err != nil {
		http.Error(w, "Failed to encode response", 500)
	}
}

// handleTFTRerollAugments rerolls the current augment offers
func (s *WebServer) handleTFTRerollAugments(w http.ResponseWriter, r *http.Request, uid string) {
	state := s.bot.getAugmentState(uid)
	st := s.bot.loadTFT(uid)
	rerollCost := getRerollCost(state.RerollCount)

	// Check if player can afford reroll
	if rerollCost > 0 && st.BattleGold < rerollCost {
		http.Error(w, "Not enough gold", 400)
		return
	}

	// Deduct gold if not free
	if rerollCost > 0 {
		st.BattleGold -= rerollCost
		s.bot.saveTFT(uid, st)
	}

	// Update reroll count
	_, _ = s.bot.DB.Exec(`
		INSERT INTO tft_augment_state (user_id, reroll_count)
		VALUES ($1, 1)
		ON CONFLICT (user_id) DO UPDATE
		SET reroll_count = tft_augment_state.reroll_count + 1
	`, uid)

	// Generate new offers
	newState := s.bot.getAugmentState(uid)
	offers := s.bot.generateAugmentOffers(uid, st.StageNumber, st.RoundNumber)

	if err := json.NewEncoder(w).Encode(map[string]any{
		"offers":     offers,
		"rerollCost": getRerollCost(newState.RerollCount),
		"canReroll":  true,
		"gold":       st.BattleGold,
	}); err != nil {
		http.Error(w, "Failed to encode response", 500)
	}
}

// applyAugmentEffect applies the effect of an augment
func (b *Bot) applyAugmentEffect(uid string, augment Augment, st *tftState) {
	var effect struct {
		Type   string `json:"type"`
		Effect string `json:"effect"`
		Value  int    `json:"value"`
	}
	if err := json.Unmarshal([]byte(augment.EffectData), &effect); err != nil {
		return
	}

	switch effect.Type {
	case "immediate":
		switch effect.Effect {
		case "grant_gold":
			st.BattleGold += effect.Value
			b.saveTFT(uid, st)
		case "grant_xp":
			st.XP += effect.Value
			b.saveTFT(uid, st)
		case "heal_player":
			// Would need to track player HP - placeholder
		}
	case "passive":
		// Passive effects are tracked in the database and applied during relevant game actions
	}
}

// getPlayerAugments returns all augments selected by a player
func (b *Bot) getPlayerAugments(uid string) []Augment {
	var augments []Augment
	rows, err := b.DB.Query(`
		SELECT a.id, a.key, a.name, a.description, a.tier, a.type, a.effect_data, a.icon
		FROM tft_player_augments pa
		JOIN tft_augments a ON pa.augment_id = a.id
		WHERE pa.user_id = $1
		ORDER BY pa.selected_at
	`, uid)
	if err != nil {
		return augments
	}
	defer rows.Close()

	for rows.Next() {
		var a Augment
		if err := rows.Scan(&a.ID, &a.Key, &a.Name, &a.Description, &a.Tier, &a.Type, &a.EffectData, &a.Icon); err == nil {
			augments = append(augments, a)
		}
	}
	return augments
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
