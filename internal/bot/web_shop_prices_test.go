package bot

import (
	"math"
	"testing"

	"ts3news/internal/content"
)

func TestShopPremiumRarityPriceFloors(t *testing.T) {
	for _, tc := range []struct {
		rarity content.Rarity
		floor  int64
	}{
		{content.RarityMythic, 3_000_000},
		{content.RarityDivine, 6_000_000},
		{content.RarityCelestial, 8_000_000},
		{content.RarityEternal, 10_000_000},
	} {
		t.Run(tc.rarity.String(), func(t *testing.T) {
			gear := content.Gear{Rarity: tc.rarity}
			if got := regularShopView(gear, nil).Price; got != tc.floor {
				t.Fatalf("zero-stat shop price = %d, want rarity floor %d", got, tc.floor)
			}
			gear.Stats = content.Stats{HP: 100, STR: 20, DEF: 10}
			want := tc.floor + int64(gear.CombatRating()*1000+float64(gear.Stats.Score())*500)
			if got := regularShopView(gear, nil).Price; got != want || got <= gearPrice(gear) {
				t.Fatalf("shop price = %d, want %d including stat premium; reference value %d", got, want, gearPrice(gear))
			}
		})
	}
}

func TestShopPricingPreservesLowerTiersAndVendorValue(t *testing.T) {
	for rarity := content.RarityCommon; rarity <= content.RarityLegendary; rarity++ {
		gear := content.Gear{Name: "Eternal is a name, not a rarity", Rarity: rarity, Stats: content.Stats{STR: 100}}
		if got := regularShopView(gear, nil).Price; got != gearPrice(gear) {
			t.Fatalf("rarity %d shop price changed from %d to %d", rarity, gearPrice(gear), got)
		}
	}
	gear := content.Gear{Rarity: content.RarityEternal, Stats: content.Stats{STR: 100}}
	wantReference := int64(gear.CombatRating()*12+float64(gear.Stats.Score())*6) * (int64(gear.Rarity) + 1)
	if got := gearPrice(gear); got != wantReference {
		t.Fatalf("shop increase changed vendor reference value to %d, want %d", got, wantReference)
	}
}

func TestShopPremiumPricesCoverFeaturedAndBuffedOffers(t *testing.T) {
	for _, seed := range []int64{0, 42, 9000} {
		featured := featuredShopView(seed, nil)
		if featured.Price != regularShopView(featured.gear, nil).Price {
			t.Fatal("featured offer bypasses the shared shop rarity price")
		}
		eternals := 0
		for _, item := range personalizedShopStock(seed, "delver", shopBuffState{Rarity: 10_000}, nil) {
			if item.gear.Rarity == content.RarityEternal {
				eternals++
				if item.Price < 10_000_000 {
					t.Fatalf("buffed Eternal offer costs only %d", item.Price)
				}
			}
		}
		if eternals != 20 {
			t.Fatalf("checked %d buffed Eternal offers, want 20", eternals)
		}
	}
}

func TestShopPremiumPriceSaturatesInsteadOfOverflowing(t *testing.T) {
	gear := content.Gear{Rarity: content.RarityEternal, Stats: content.Stats{STR: 1 << 55}}
	if got := regularShopView(gear, nil).Price; got != math.MaxInt64 {
		t.Fatalf("extreme shop price = %d, want saturated %d", got, int64(math.MaxInt64))
	}
}
