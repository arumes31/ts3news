package bot

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/rand/v2"
	"sort"
	"time"

	"ts3news/internal/content"
)

const abyssWeeklyFeaturedDropCount = 3

type abyssWeeklyFeaturedDropItem struct {
	ID     string
	Name   string
	Slot   string
	Rarity string
}

type abyssWeeklyFeaturedDropView struct {
	Week      string
	ResetAt   time.Time
	ResetText string
	Items     []abyssWeeklyFeaturedDropItem
	IDs       map[string]bool
}

func abyssWeeklyFeaturedDrops(at time.Time) abyssWeeklyFeaturedDropView {
	resetAt, week := abyssWeeklyFeaturedWindow(at)
	catalog := content.AbyssGearCatalog()
	sort.Slice(catalog, func(i, j int) bool {
		left := sha256.Sum256([]byte(week + ":" + catalog[i].ID))
		right := sha256.Sum256([]byte(week + ":" + catalog[j].ID))
		if compared := bytes.Compare(left[:], right[:]); compared != 0 {
			return compared < 0
		}
		return catalog[i].ID < catalog[j].ID
	})

	count := min(abyssWeeklyFeaturedDropCount, len(catalog))
	view := abyssWeeklyFeaturedDropView{
		Week: week, ResetAt: resetAt, ResetText: resetAt.Format("Mon 02 Jan · 15:04 UTC"),
		Items: make([]abyssWeeklyFeaturedDropItem, 0, count),
		IDs:   make(map[string]bool, count),
	}
	for _, gear := range catalog[:count] {
		view.Items = append(view.Items, abyssWeeklyFeaturedDropItem{
			ID: gear.ID, Name: gear.Name, Slot: string(gear.Slot), Rarity: gear.Rarity.String(),
		})
		view.IDs[gear.ID] = true
	}
	return view
}

func abyssWeeklyFeaturedWindow(at time.Time) (time.Time, string) {
	at = at.UTC()
	weekdayOffset := (int(at.Weekday()) + 6) % 7
	monday := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, -weekdayOffset)
	year, week := monday.ISOWeek()
	return monday.AddDate(0, 0, 7), fmt.Sprintf("%04d-W%02d", year, week)
}

type abyssWeeklyDropRandom interface {
	Float64() float64
	IntN(int) int
}

type abyssWeeklyGameplayRandom struct{}

func (abyssWeeklyGameplayRandom) Float64() float64 { return rand.Float64() } // #nosec G404 -- gameplay loot roll
func (abyssWeeklyGameplayRandom) IntN(n int) int   { return rand.IntN(n) }   // #nosec G404 -- gameplay loot roll

func rollAbyssWeeklyFeaturedGear(
	category string,
	owned map[string]bool,
	rotation abyssWeeklyFeaturedDropView,
	source abyssWeeklyDropRandom,
) content.Gear {
	catalog := content.AbyssGearCatalog()
	candidates := weeklyFeaturedCandidates(catalog, category, owned, true)
	if len(candidates) == 0 {
		candidates = weeklyFeaturedCandidates(catalog, category, owned, false)
	}
	if len(candidates) == 0 && category != "" {
		candidates = weeklyFeaturedCandidates(catalog, "", owned, true)
		if len(candidates) == 0 {
			candidates = weeklyFeaturedCandidates(catalog, "", owned, false)
		}
	}
	if len(candidates) == 0 {
		return content.Gear{}
	}

	tickets := len(candidates)
	for _, gear := range candidates {
		if rotation.IDs[gear.ID] {
			tickets++
		}
	}
	pick := source.IntN(tickets)
	for _, gear := range candidates {
		weight := 1
		if rotation.IDs[gear.ID] {
			weight = 2
		}
		if pick < weight {
			if gear.Special == content.EffectNone {
				gear.Special = content.RandomItemEffectWithRandom(source)
			}
			return gear
		}
		pick -= weight
	}
	return candidates[len(candidates)-1]
}

func weeklyFeaturedCandidates(
	catalog []content.Gear,
	category string,
	owned map[string]bool,
	excludeOwned bool,
) []content.Gear {
	candidates := make([]content.Gear, 0, len(catalog))
	for _, gear := range catalog {
		if category != "" && !abyssGearMatchesLootCategory(gear.Slot, category) {
			continue
		}
		if excludeOwned && owned[gear.ID] {
			continue
		}
		candidates = append(candidates, gear)
	}
	return candidates
}
