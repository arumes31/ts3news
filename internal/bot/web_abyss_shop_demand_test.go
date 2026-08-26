package bot

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssShopDemandAdjustments(t *testing.T) {
	counts := make(map[string]int64)
	for _, item := range abyssShopCatalog {
		if abyssShopDemandEligible(item) {
			counts[item.Key] = 1
		}
	}
	counts["great_potions"] = 20
	got := abyssShopDemandAdjustments(counts)
	if got["great_potions"].Percent != 10 || got["great_potions"].Purchases != 20 {
		t.Fatalf("hot item = %+v, want 20 purchases and +10%%", got["great_potions"])
	}
	if got["repair_kits"].Percent != -10 {
		t.Fatalf("cool item adjustment = %d, want -10", got["repair_kits"].Percent)
	}
	if _, exists := got["insanity_void_aura"]; exists {
		t.Fatal("rotating cosmetics must not participate in demand pricing")
	}
	if quiet := abyssShopDemandAdjustments(map[string]int64{"great_potions": 1}); quiet["great_potions"].Percent != 0 {
		t.Fatal("a sparse market must stay at base price")
	}
}

func TestAbyssShopPricedCostLayersDemandBeforeDailyDeal(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	var deal abyssShopItem
	for _, item := range abyssShopCatalog {
		if _, selected := abyssShopEffectiveCost(item, now); selected {
			deal = item
			break
		}
	}
	marketCost := abyssShopDemandCost(deal.Cost, 10)
	got, selected := abyssShopPricedCost(deal, now, 10)
	if !selected || got != abyssDiscountedCost(marketCost) {
		t.Fatalf("layered price = %d, deal %v; want %d, true", got, selected, abyssDiscountedCost(marketCost))
	}
	if abyssShopDemandCost(30, -10) != 27 || abyssShopDemandCost(30, 10) != 33 {
		t.Fatal("demand prices must adjust base cost by exactly ten percent when integral")
	}
}

func TestAbyssShopDemandUsesPreviousCompletedUTCDays(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	now := time.Date(2026, 8, 26, 19, 30, 0, 0, time.FixedZone("local", 2*60*60))
	end := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	start := end.AddDate(0, 0, -7)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT item_key,COALESCE(SUM(purchases),0) FROM abyss_shop_demand")).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"item_key", "purchases"}).AddRow("great_potions", int64(24)))
	got := (&Bot{DB: database}).abyssShopDemand(now)
	if got["great_potions"].Purchases != 24 {
		t.Fatalf("purchases = %d, want 24", got["great_potions"].Purchases)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssShopDemandTemplateContract(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_shop_demand.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"7-day market", "previous seven completed UTC days", ".DemandSales", "exact total {{.EffectiveCost}} tokens"} {
		if !strings.Contains(string(page), token) {
			t.Fatalf("shop template missing %q", token)
		}
	}
	handler, err := os.ReadFile("web_abyss_shop.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"abyssShopDemand(now)", "abyssShopPricedCost(item, now, market.Percent)", "*req.QuotedCost != tokenCost", "recordAbyssShopDemand(item.Key, now)"} {
		if !strings.Contains(string(handler), token) {
			t.Fatalf("shop checkout missing authoritative demand contract %q", token)
		}
	}
	for _, token := range []string{".ab-shop-hot", ".ab-shop-cool", "forced-colors"} {
		if !strings.Contains(string(styles), token) {
			t.Fatalf("shop demand stylesheet missing %q", token)
		}
	}
}
