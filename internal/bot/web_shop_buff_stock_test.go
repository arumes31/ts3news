package bot

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestShopBuffBeyondEternalScalesOriginalStatsLinearly(t *testing.T) {
	base := content.Gear{Rarity: content.RarityCelestial, Stats: content.Stats{HP: 10000, STR: 1000, CHA: 9}}
	for _, tc := range []struct {
		tokens  int64
		hp, str int
	}{
		{1000, 10000, 1000}, {2000, 10010, 1001}, {11000, 10100, 1010}, {1500, 10005, 1000},
	} {
		got := applyShopRarityBuff(base, tc.tokens, 0)
		if got.Rarity != content.RarityEternal || got.Stats.HP != tc.hp || got.Stats.STR != tc.str || got.Stats.CHA != 9 {
			t.Fatalf("%d tokens: %+v", tc.tokens, got)
		}
	}
}

func TestShopBuffMaximumCounterCannotWrapStatsOrMakeGearCheap(t *testing.T) {
	for _, base := range content.GearAppearanceCatalog() {
		if strings.HasPrefix(base.ID, "B_") || content.IsInsanityGearID(base.ID) {
			continue
		}
		got := applyShopRarityBuff(base, math.MaxInt64, 0)
		if got.Stats.Score() < base.Stats.Score() || got.CombatRating() < base.CombatRating() || gearPrice(got) < gearPrice(base) {
			t.Fatalf("maximum counter overflow for %s: %+v", base.ID, got)
		}
	}
}

func TestShopBuffQuantityNeverRemovesAnExistingRarityPromotion(t *testing.T) {
	base := personalizedShopStock(42, "delver", shopBuffState{Rarity: 500}, nil)
	for _, count := range []int64{21, 500, 10000} {
		got := personalizedShopStock(42, "delver", shopBuffState{Rarity: 500, Quantity: count}, nil)
		if !reflect.DeepEqual(got[:len(base)], base) {
			t.Fatal("quantity rerolled existing rarity bonuses")
		}
	}
}

func TestShopBuffLargeQuantityUsesBoundedPagesAndTwentyTotalPositions(t *testing.T) {
	buffs := shopBuffState{Rarity: 1000, Quantity: 10000}
	info := shopStockPagination(42, "delver", buffs, 0)
	if info.Total != 529 || info.Pages != 3 {
		t.Fatalf("pagination: %+v", info)
	}
	promoted := 0
	seen := map[string]bool{}
	for page := int64(0); page < info.Pages; page++ {
		stock := personalizedShopStockPage(42, "delver", buffs, nil, page)
		if len(stock) > shopStockPageSize+1 {
			t.Fatal("stock page is unbounded")
		}
		for _, item := range stock {
			if item.RarityBoosted {
				promoted++
				if seen[item.ID] {
					t.Fatal("promoted offer repeated across pages")
				}
				seen[item.ID] = true
			}
		}
	}
	if promoted != 20 {
		t.Fatalf("%d boosted slots across pages, want20", promoted)
	}
	huge := shopBuffState{Rarity: math.MaxInt64, Quantity: math.MaxInt64}
	last := shopStockPagination(42, "delver", huge, math.MaxInt64)
	stock := personalizedShopStockPage(42, "delver", huge, nil, last.Page)
	if len(stock) == 0 || len(stock) > shopStockPageSize || last.Pages < 1000 {
		t.Fatal("large counters truncated bonus or broke pagination")
	}
	if len(shopRarityRolls(42, "delver")) != 20 {
		t.Fatal("large selection is unbounded")
	}
}

func TestShopBuffStockPreservesUnbuffedRotation(t *testing.T) {
	for _, seed := range []int64{0, 42, 9000} {
		got := personalizedShopStock(seed, "delver", shopBuffState{}, nil)
		if !reflect.DeepEqual(got, stockForSeed(seed, nil)) || len(got) != shopStockSize+1 {
			t.Fatal("zero buffs changed existing stock")
		}
	}
}

func TestShopBuffRaritySelectsTwentyStablePositions(t *testing.T) {
	base := content.ShopStock(42, shopStockSize)
	state := shopBuffState{Rarity: 1000}
	got := personalizedShopStock(42, "delver", state, nil)
	if !reflect.DeepEqual(got, personalizedShopStock(42, "delver", state, nil)) {
		t.Fatal("reload rerolled stock")
	}
	if !reflect.DeepEqual(got[0], stockForSeed(42, nil)[0]) {
		t.Fatal("featured relic changed")
	}
	boosted := 0
	for i, item := range got[1:] {
		if item.RarityBoosted {
			boosted++
			if item.gear.Rarity != base[i].Rarity+1 || item.Price != shopGearPrice(item.gear) || item.CR != item.gear.CombatRating() || item.ID == base[i].ID {
				t.Fatalf("promotion is not an exact, separately priced item: %+v", item)
			}
		} else if item.gear.Rarity != base[i].Rarity {
			t.Fatal("unselected position changed rarity")
		}
		if strings.HasPrefix(item.gear.ID, "B_") || content.IsInsanityGearID(item.gear.ID) {
			t.Fatal("shop pool exclusions changed")
		}
	}
	if boosted != 20 {
		t.Fatalf("promoted %d positions, want 20", boosted)
	}
	if reflect.DeepEqual(got, personalizedShopStock(42, "another-delver", state, nil)) {
		t.Fatal("different players share all rarity rolls")
	}
}

func TestShopBuffChanceGrowsWithoutRerolling(t *testing.T) {
	previous := map[int]bool{}
	for _, count := range []int64{1, 100, 500, 1000, 5000} {
		got := personalizedShopStock(42, "delver", shopBuffState{Rarity: count}, nil)
		for i, item := range got {
			if previous[i] && !item.RarityBoosted {
				t.Fatal("buying a rarity token removed a boost")
			}
			previous[i] = item.RarityBoosted
		}
	}
}

func TestShopBuffRarityContinuesBeyondOneHundredPercent(t *testing.T) {
	base := content.ShopStock(42, shopStockSize)
	for _, count := range []int64{1500, 2500} {
		stock := personalizedShopStock(42, "delver", shopBuffState{Rarity: count}, nil)
		for i, item := range stock[1:] {
			if !item.RarityBoosted {
				continue
			}
			floor := min(base[i].Rarity+content.Rarity(count/1000), content.RarityEternal)
			ceiling := min(floor+1, content.RarityEternal)
			if item.gear.Rarity < floor || item.gear.Rarity > ceiling {
				t.Fatalf("%d tokens stopped at one-tier boost", count)
			}
		}
	}
	if shopBuffViews(shopBuffState{Rarity: 2500})[0].Bonus != "250.0" {
		t.Fatal("displayed bonus capped")
	}
}

func TestShopBuffQuantityPreservesBaseStockAndFractionalChance(t *testing.T) {
	for _, count := range []int64{1, 21, 100, 1000, 2000} {
		state := shopBuffState{Quantity: count}
		got := personalizedShopStock(42, "delver", state, nil)
		base := stockForSeed(42, nil)
		if !reflect.DeepEqual(got[:len(base)], base) {
			t.Fatal("quantity rerolled base stock")
		}
		extra, floor := len(got)-len(base), int(count*shopStockSize/1000)
		if extra < floor || extra > floor+1 {
			t.Fatalf("%d tokens produced %d extras; want %d or %d", count, extra, floor, floor+1)
		}
		if !reflect.DeepEqual(got, personalizedShopStock(42, "delver", state, nil)) {
			t.Fatal("quantity changed on reload")
		}
	}
	hits := 0
	for seed := int64(0); seed < 1000; seed++ {
		if shopQuantityExtra(seed, "delver", 1) == 1 {
			hits++
		}
	}
	if hits < 25 || hits > 80 {
		t.Fatalf("fractional stock chance: %d/1000 rotations", hits)
	}
}

func TestShopBuffStockRevisionIncludesOwnerRotationAndBothCounters(t *testing.T) {
	base := shopStockRevision(42, "delver", shopBuffState{})
	for _, got := range []string{
		shopStockRevision(43, "delver", shopBuffState{}),
		shopStockRevision(42, "other", shopBuffState{}),
		shopStockRevision(42, "delver", shopBuffState{Rarity: 1}),
		shopStockRevision(42, "delver", shopBuffState{Quantity: 1}),
	} {
		if got == base {
			t.Fatal("review revision did not change")
		}
	}
}
