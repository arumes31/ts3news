package bot

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"sort"
	"strings"
	"time"

	"ts3news/internal/content"
)

type abyssTargetCraftRequest struct {
	GearID          string `json:"gear_id"`
	Slot            string `json:"slot"`
	Rarity          int    `json:"rarity"`
	SetID           string `json:"set_id"`
	Event           string `json:"event"`
	DuplicatePolicy string `json:"duplicate_policy"`
}

func abyssForgeEventRecipe(at time.Time) string {
	year, week := at.UTC().ISOWeek()
	return fmt.Sprintf("solar-%d-w%02d", year, week)
}

func abyssTargetCraftCost(rarity content.Rarity) map[string]int {
	costs := map[content.Rarity]map[string]int{
		content.RarityRare:      {"dust": 50, "shard": 5},
		content.RarityEpic:      {"dust": 100, "shard": 10},
		content.RarityLegendary: {"dust": 500, "shard": 50, "core": 10},
		content.RarityMythic:    {"dust": 800, "shard": 80, "core": 20, "prism": 1},
		content.RarityDivine:    {"dust": 1200, "shard": 120, "core": 30, "prism": 3},
		content.RarityCelestial: {"dust": 2000, "shard": 200, "core": 50, "prism": 10},
	}
	result := map[string]int{}
	for material, amount := range costs[rarity] {
		result[material] = amount
	}
	return result
}

func targetCraftCandidates(request abyssTargetCraftRequest) []content.Gear {
	rarity := content.Rarity(request.Rarity)
	candidates := content.GearByMinRarity(rarity)
	filtered := candidates[:0]
	for _, gear := range candidates {
		if gear.Rarity != rarity || (request.GearID != "" && gear.ID != request.GearID) ||
			(request.Slot != "" && string(gear.Slot) != request.Slot) ||
			(request.SetID != "" && gear.EffectiveSetID() != request.SetID) {
			continue
		}
		filtered = append(filtered, gear)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].ID < filtered[j].ID })
	return filtered
}

func (s *WebServer) ownedGearIDs(uid string) map[string]bool {
	owned := map[string]bool{}
	queries := []string{
		"SELECT gear_id FROM user_inventory WHERE client_uid=$1",
		"SELECT gear_id FROM user_gear WHERE client_uid=$1",
	}
	for _, query := range queries {
		rows, err := s.bot.DB.Query(query, uid)
		if err != nil {
			continue
		}
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				owned[id] = true
			}
		}
		_ = rows.Close()
	}
	return owned
}

func chooseTargetCraft(candidates []content.Gear, request abyssTargetCraftRequest, owned map[string]bool) (content.Gear, bool, bool) {
	if request.DuplicatePolicy == "avoid" {
		for _, candidate := range candidates {
			if !owned[candidate.ID] {
				return candidate, false, true
			}
		}
		return content.Gear{}, false, false
	}
	index := 0
	if request.Event != "" && len(candidates) > 0 {
		hash := fnv.New32a()
		_, _ = hash.Write([]byte(request.Event + "\x00" + request.Slot + "\x00" + request.SetID))
		index = int(hash.Sum32() % uint32(len(candidates)))
	}
	gear := candidates[index]
	return gear, owned[gear.ID], true
}

func (s *WebServer) handleAbyssTargetCraft(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var request abyssTargetCraftRequest
	if err := decodeBoundedForgeRequest(r, &request); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid targeted craft request"})
		return
	}
	rarity := content.Rarity(request.Rarity)
	if rarity < content.RarityRare || rarity > content.RarityCelestial {
		writeJSON(w, map[string]any{"ok": false, "error": "target rarity must be Rare through Celestial"})
		return
	}
	if request.Event != "" && request.Event != abyssForgeEventRecipe(time.Now()) {
		writeJSON(w, map[string]any{"ok": false, "error": "event recipe is not active"})
		return
	}
	switch request.DuplicatePolicy {
	case "", "allow", "avoid", "upgrade", "materials":
	default:
		writeJSON(w, map[string]any{"ok": false, "error": "unknown duplicate policy"})
		return
	}
	candidates := targetCraftCandidates(request)
	gear, duplicate, ok := chooseTargetCraft(candidates, request, s.ownedGearIDs(uid))
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "no eligible non-duplicate item matches those targets"})
		return
	}
	cost := abyssTargetCraftCost(rarity)
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if !spendMaterials(tx, uid, cost) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough materials"})
		return
	}
	result := "crafted"
	if duplicate && request.DuplicatePolicy == "materials" {
		refundMaterial, refundAmount := materialYieldForRarity(rarity)
		if err := grantMaterialQ(tx, uid, refundMaterial, refundAmount*2); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		result = "duplicate_recycled"
	} else {
		if duplicate && request.DuplicatePolicy == "upgrade" {
			gear.Stats = gear.Stats.Scaled(1.05)
			gear.Name = "Refined " + gear.Name
			result = "duplicate_upgraded"
		}
		encoded, _ := json.Marshal(gear)
		if _, err := tx.ExecContext(r.Context(),
			"INSERT INTO user_inventory (client_uid, gear_id, durability, item_data) VALUES ($1,$2,$3,$4)",
			uid, gear.ID, gear.MaxDurability, string(encoded)); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	costParts := make([]string, 0, len(cost))
	for _, material := range abyssMaterials {
		if amount := cost[material.ID]; amount > 0 {
			costParts = append(costParts, fmt.Sprintf("%d%s", amount, material.Icon))
		}
	}
	s.bot.recordForge(uid, "target craft", gear.Name, strings.Join(costParts, " "))
	writeJSON(w, map[string]any{
		"ok": true, "result": result, "item": gear, "materials": s.bot.loadMaterials(uid),
		"event": request.Event, "msg": "⚒️ Target craft complete: " + gear.Name,
	})
}
