package bot

import (
	"testing"
	"time"

	"ts3news/internal/content"
)

type featuredDropFixedRandom struct{ value int }

func (r featuredDropFixedRandom) Float64() float64 { return 1 }
func (r featuredDropFixedRandom) IntN(n int) int   { return r.value % n }

func TestAbyssWeeklyFeaturedDropsAreStableUniqueAndRotate(t *testing.T) {
	t.Parallel()

	current := abyssWeeklyFeaturedDrops(time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC))
	repeat := abyssWeeklyFeaturedDrops(time.Date(2026, time.August, 30, 23, 59, 0, 0, time.UTC))
	next := abyssWeeklyFeaturedDrops(time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC))
	if current.Week != "2026-W35" || current.ResetAt != time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("current rotation = %#v", current)
	}
	if len(current.Items) != abyssWeeklyFeaturedDropCount || len(current.IDs) != abyssWeeklyFeaturedDropCount {
		t.Fatalf("featured counts = %d items, %d IDs", len(current.Items), len(current.IDs))
	}
	for index := range current.Items {
		if current.Items[index].ID != repeat.Items[index].ID {
			t.Fatalf("same-week item %d changed from %s to %s", index, current.Items[index].ID, repeat.Items[index].ID)
		}
	}
	if next.Week == current.Week || sameFeaturedIDs(current, next) {
		t.Fatalf("next ISO week did not rotate: current=%v next=%v", current.Items, next.Items)
	}
}

func TestRollAbyssWeeklyFeaturedGearUsesExactDoubleWeight(t *testing.T) {
	t.Parallel()

	catalog := content.AbyssGearCatalog()
	if len(catalog) < 4 {
		t.Fatalf("Abyss catalog too small: %d", len(catalog))
	}
	rotation := abyssWeeklyFeaturedDropView{IDs: map[string]bool{catalog[0].ID: true}}
	tickets := len(catalog) + 1
	counts := make(map[string]int)
	for ticket := 0; ticket < tickets; ticket++ {
		gear := rollAbyssWeeklyFeaturedGear("", nil, rotation, featuredDropFixedRandom{value: ticket})
		counts[gear.ID]++
	}
	if counts[catalog[0].ID] != 2 {
		t.Fatalf("featured item tickets = %d, want 2", counts[catalog[0].ID])
	}
	for _, gear := range catalog[1:] {
		if counts[gear.ID] != 1 {
			t.Fatalf("ordinary item %s tickets = %d, want 1", gear.ID, counts[gear.ID])
		}
	}
}

func TestRollAbyssWeeklyFeaturedGearPreservesCategoryAndOwnership(t *testing.T) {
	t.Parallel()

	catalog := content.AbyssGearCatalog()
	owned := make(map[string]bool)
	for _, gear := range catalog {
		if abyssGearMatchesLootCategory(gear.Slot, "weapon") {
			owned[gear.ID] = true
		}
	}
	var availableWeapon string
	for _, gear := range catalog {
		if abyssGearMatchesLootCategory(gear.Slot, "weapon") {
			availableWeapon = gear.ID
			delete(owned, gear.ID)
			break
		}
	}
	rotation := abyssWeeklyFeaturedDropView{IDs: map[string]bool{availableWeapon: true}}
	got := rollAbyssWeeklyFeaturedGear("weapon", owned, rotation, featuredDropFixedRandom{})
	if got.ID != availableWeapon || !abyssGearMatchesLootCategory(got.Slot, "weapon") {
		t.Fatalf("category/ownership roll = %#v, want %s", got, availableWeapon)
	}
}

func sameFeaturedIDs(left, right abyssWeeklyFeaturedDropView) bool {
	if len(left.IDs) != len(right.IDs) {
		return false
	}
	for id := range left.IDs {
		if !right.IDs[id] {
			return false
		}
	}
	return true
}
