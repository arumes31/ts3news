package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"ts3news/internal/content"
)

const (
	abyssWishlistLimit    = 3
	abyssWishlistPityCap  = 30
	abyssWishlistPageSize = 24
)

type abyssWishlistState struct {
	GearIDs []string `json:"gear_ids"`
	Pity    int      `json:"pity"`
}

type abyssWishlistItemView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Slot       string `json:"slot"`
	Rarity     string `json:"rarity"`
	RarityRank int    `json:"rarity_rank"`
	Selected   bool   `json:"selected"`
}

type abyssWishlistView struct {
	Selected   []abyssWishlistItemView `json:"selected"`
	Candidates []abyssWishlistItemView `json:"candidates"`
	Count      int                     `json:"count"`
	Limit      int                     `json:"limit"`
	Pity       int                     `json:"pity"`
	PityCap    int                     `json:"pity_cap"`
	PityPct    int                     `json:"pity_pct"`
	Query      string                  `json:"query,omitempty"`
}

type abyssProcessedGear struct {
	Label           string
	Gear            content.Gear
	SmartLootReason string
	SetPitySetID    string
	WishlistState   abyssWishlistState
	WishlistChanged bool
	WishlistHit     bool
}

var abyssWishlistCatalog = func() map[string]content.Gear {
	catalog := content.AbyssGearCatalog()
	byID := make(map[string]content.Gear, len(catalog))
	for _, gear := range catalog {
		byID[gear.ID] = gear
	}
	return byID
}()

func abyssWishlistKey(uid string) string {
	return "abyss_loot_wishlist_" + uid
}

func normalizeAbyssWishlist(state abyssWishlistState) abyssWishlistState {
	state.Pity = min(max(state.Pity, 0), abyssWishlistPityCap)
	ids := make([]string, 0, min(len(state.GearIDs), abyssWishlistLimit))
	seen := make(map[string]bool, len(state.GearIDs))
	for _, id := range state.GearIDs {
		id = strings.TrimSpace(id)
		if seen[id] {
			continue
		}
		if _, ok := abyssWishlistCatalog[id]; !ok {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
		if len(ids) == abyssWishlistLimit {
			break
		}
	}
	state.GearIDs = ids
	if len(ids) == 0 {
		state.Pity = 0
	}
	return state
}

func abyssWishlistStatesEqual(a, b abyssWishlistState) bool {
	a = normalizeAbyssWishlist(a)
	b = normalizeAbyssWishlist(b)
	if a.Pity != b.Pity || len(a.GearIDs) != len(b.GearIDs) {
		return false
	}
	for i := range a.GearIDs {
		if a.GearIDs[i] != b.GearIDs[i] {
			return false
		}
	}
	return true
}

func toggleAbyssWishlist(state abyssWishlistState, gearID string) (abyssWishlistState, error) {
	state = normalizeAbyssWishlist(state)
	gearID = strings.TrimSpace(gearID)
	if _, ok := abyssWishlistCatalog[gearID]; !ok {
		return state, fmt.Errorf("unknown Abyss gear")
	}
	for i, id := range state.GearIDs {
		if id == gearID {
			state.GearIDs = append(state.GearIDs[:i], state.GearIDs[i+1:]...)
			state.Pity = 0
			return normalizeAbyssWishlist(state), nil
		}
	}
	if len(state.GearIDs) >= abyssWishlistLimit {
		return state, fmt.Errorf("wishlist is full")
	}
	state.GearIDs = append(state.GearIDs, gearID)
	state.Pity = 0
	return state, nil
}

func abyssWishlistItem(gear content.Gear, selected bool) abyssWishlistItemView {
	return abyssWishlistItemView{
		ID: gear.ID, Name: gear.Name, Slot: string(gear.Slot),
		Rarity: gear.Rarity.String(), RarityRank: int(gear.Rarity), Selected: selected,
	}
}

func abyssWishlistViewFor(state abyssWishlistState, query string) abyssWishlistView {
	state = normalizeAbyssWishlist(state)
	selectedIDs := make(map[string]bool, len(state.GearIDs))
	selected := make([]abyssWishlistItemView, 0, len(state.GearIDs))
	for _, id := range state.GearIDs {
		selectedIDs[id] = true
		selected = append(selected, abyssWishlistItem(abyssWishlistCatalog[id], true))
	}

	query = strings.ToLower(strings.TrimSpace(query))
	candidates := make([]abyssWishlistItemView, 0, abyssWishlistPageSize)
	for id, gear := range abyssWishlistCatalog {
		if selectedIDs[id] {
			continue
		}
		haystack := strings.ToLower(gear.Name + " " + gear.ID + " " + string(gear.Slot) + " " + gear.Rarity.String())
		if query != "" && !strings.Contains(haystack, query) {
			continue
		}
		candidates = append(candidates, abyssWishlistItem(gear, false))
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].RarityRank != candidates[j].RarityRank {
			return candidates[i].RarityRank > candidates[j].RarityRank
		}
		if candidates[i].Name != candidates[j].Name {
			return candidates[i].Name < candidates[j].Name
		}
		return candidates[i].ID < candidates[j].ID
	})
	if len(candidates) > abyssWishlistPageSize {
		candidates = candidates[:abyssWishlistPageSize]
	}
	return abyssWishlistView{
		Selected: selected, Candidates: candidates,
		Count: len(selected), Limit: abyssWishlistLimit,
		Pity: state.Pity, PityCap: abyssWishlistPityCap,
		PityPct: state.Pity * 100 / abyssWishlistPityCap, Query: query,
	}
}

func (b *Bot) loadAbyssWishlist(uid string) abyssWishlistState {
	var raw string
	err := b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssWishlistKey(uid)).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return abyssWishlistState{}
	}
	if err != nil {
		log.Printf("abyss wishlist read failed for %s: %v", uid, err)
		return abyssWishlistState{}
	}
	var state abyssWishlistState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		log.Printf("abyss wishlist decode failed for %s: %v", uid, err)
		return abyssWishlistState{}
	}
	return normalizeAbyssWishlist(state)
}

func saveAbyssWishlist(db dbOrTx, uid string, state abyssWishlistState) error {
	state = normalizeAbyssWishlist(state)
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encoding Abyss wishlist: %w", err)
	}
	_, err = db.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, abyssWishlistKey(uid), string(data))
	if err != nil {
		return fmt.Errorf("saving Abyss wishlist: %w", err)
	}
	return nil
}

type abyssWishlistRandom interface {
	IntN(int) int
}

func applyAbyssWishlist(
	gear content.Gear,
	pool content.GearDropPool,
	state abyssWishlistState,
	category string,
	allowReplacement bool,
	rng abyssWishlistRandom,
) (content.Gear, abyssWishlistState, bool) {
	state = normalizeAbyssWishlist(state)
	if pool != content.GearDropPoolAbyss || len(state.GearIDs) == 0 {
		return gear, state, false
	}
	state.Pity = min(state.Pity+1, abyssWishlistPityCap)
	if state.Pity < abyssWishlistPityCap || !allowReplacement {
		return gear, state, false
	}

	candidates := make([]content.Gear, 0, len(state.GearIDs))
	for _, id := range state.GearIDs {
		candidate := abyssWishlistCatalog[id]
		if candidate.Rarity > gear.Rarity || !abyssGearMatchesLootCategory(candidate.Slot, category) {
			continue
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return gear, state, false
	}
	replacement := candidates[rng.IntN(len(candidates))]
	replacement.Rarity = gear.Rarity
	if replacement.Special == content.EffectNone {
		replacement.Special = content.RandomItemEffect()
	}
	state.Pity = 0
	return replacement, state, true
}

func abyssWishlistLabel(hit bool) string {
	if !hit {
		return ""
	}
	return " [color=#f7c95c]★ WISHLIST GUARANTEE[/color]"
}
