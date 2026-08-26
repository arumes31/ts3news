//go:build e2e

package bot

import (
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

func TestAbyssE2EServer(t *testing.T) {
	if err := i18n.InitWithLocale(i18n.LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	var wishlistMu sync.Mutex
	wishlistState := abyssWishlistState{}
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/static/")
		if name == "" || path.Clean(name) != name {
			http.NotFound(w, r)
			return
		}
		ServeAsset(
			w,
			r,
			"webassets/"+name,
			mime.TypeByExtension(path.Ext(name)),
		)
	})
	mux.HandleFunc("/api/abyss/loot/wishlist", func(w http.ResponseWriter, r *http.Request) {
		wishlistMu.Lock()
		defer wishlistMu.Unlock()
		if r.Method == http.MethodPost {
			var request struct {
				GearID string `json:"gear_id"`
			}
			if readJSON(r, &request) != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
				return
			}
			var toggleErr error
			wishlistState, toggleErr = toggleAbyssWishlist(wishlistState, request.GearID)
			if toggleErr != nil {
				writeJSON(w, map[string]any{"ok": false, "error": toggleErr.Error()})
				return
			}
		} else if r.Method != http.MethodGet {
			http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]any{
			"ok":       true,
			"wishlist": abyssWishlistViewFor(wishlistState, r.URL.Query().Get("q")),
		})
	})
	mux.HandleFunc("/inventory", func(w http.ResponseWriter, _ *http.Request) {
		charm := content.Gear{
			ID: "TEST_CHARM", Name: "Lucky Test Charm", Slot: content.SlotCharm,
			Rarity: content.RarityEpic, MaxDurability: 60, Stats: content.Stats{HP: 80, LCK: 35},
			FoundAt: "2026-08-20T14:15:16Z", FoundDepth: 18, FoundBoss: "Gorgoroth the Firelord",
		}
		mystery := content.Gear{
			ID: "SECRET_CELESTIAL", Name: "Secret Celestial Ring", Slot: content.SlotFinger1,
			Rarity: content.RarityCelestial, MaxDurability: 90, Stats: content.Stats{INT: 900},
			Unidentified: true,
		}
		charmView := toGearView(charm.Slot, charm)
		charmView.InvID = 1
		charmView.Durability = charm.MaxDurability
		mysteryView := toGearView(mystery.Slot, mystery)
		mysteryView.InvID = 2
		if err := server.tmpl.ExecuteTemplate(w, "inventory", map[string]any{
			"Title": "Inventory", "Nav": "inventory", "EnableAbyss": true,
			"Items": []gearView{charmView, mysteryView}, "Consumables": []consumableView{},
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/armory-fixture", func(w http.ResponseWriter, _ *http.Request) {
		u := &webUser{
			UID: "e2e-armory", Nickname: "Armoury Tester", Level: 100,
			LevelName: "Eternal", CurrentHP: 123456, MaxHP: 234567, MaxMana: 4567,
			GearScore: 123456, Stats: content.Stats{HP: 234567, STR: 123456, DEF: 65432},
		}
		weapon := content.Gear{
			ID: "TEST_WEAPON", Name: "Measured Test Blade", Slot: content.SlotMainHand,
			Rarity: content.RarityLegendary, MaxDurability: 100,
			Stats: content.Stats{STR: 123456, CRT: 2345}, FoundAt: "2026-08-21T09:10:11Z",
			FoundDepth: 25, FoundBoss: "Malakor the Voidweaver",
		}
		weaponView := toGearView(weapon.Slot, weapon)
		weaponView.BrokenIn = true
		mystery := content.Gear{
			ID: "SECRET_ARMORY_CELESTIAL", Name: "Secret Armory Crown", Slot: content.SlotHead,
			Rarity: content.RarityCelestial, MaxDurability: 90,
			Stats: content.Stats{INT: 987654}, Unidentified: true,
		}
		if err := server.tmpl.ExecuteTemplate(w, "armory", map[string]any{
			"Title": "Armoury", "Nav": "armory", "EnableAbyss": true, "U": u,
			"Slots":  []gearView{weaponView, toGearView(mystery.Slot, mystery)},
			"Skills": []any{}, "Ultimates": []any{}, "Artifact": nil,
			"PlayerTitle": nil, "Pets": []any{},
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/abyss", func(w http.ResponseWriter, r *http.Request) {
		fixture := abyssGoldenFixture(r.URL.Query().Get("active") == "1")
		fixture["FreeID"] = true
		shopNow := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
		weeklyKey := abyssWeeklyInsanityCosmetic(shopNow)
		weeklyItem := abyssShopIndex[weeklyKey]
		fixture["Shop"] = []abyssShopItemView{
			{abyssShopItem: abyssShopCatalog[0], EffectiveCost: abyssShopCatalog[0].Cost},
			{
				abyssShopItem: weeklyItem, EffectiveCost: weeklyItem.Cost, Insanity: true,
				RotationWeek: abyssEconomyWeek(shopNow),
				RotationEnds: abyssWeeklyCosmeticReset(shopNow).Format("2006-01-02 15:04 UTC"),
			},
		}
		fixture["SetPityPanel"] = abyssSetPityPanelView{
			ChancePct: 25, HiddenItems: 1,
			Sets: []abyssSetPityProgressView{
				{ID: "predator", Name: "Predator", Icon: "🐺", Owned: 3, Required: 4, Percent: 75, Remaining: 1, Active: true},
				{ID: "warden", Name: "Warden", Icon: "🛡️", Owned: 1, Required: 4, Percent: 25, Remaining: 3},
			},
		}
		fixture["InsuranceCharms"] = 2
		fixture["Social"] = abyssSocialHubView{
			WeeklyBoss:  abyssWeeklyBossView{Name: "The Fixture Colossus", HP: 750_000, MaxHP: 1_000_000, Percent: 75, Multiplier: 1},
			PetFeedCost: 250,
		}
		wishlistMu.Lock()
		fixture["Wishlist"] = abyssWishlistViewFor(wishlistState, "")
		wishlistMu.Unlock()
		if r.URL.Query().Get("gear") == "1" {
			equipped := map[content.GearSlot]content.Gear{
				content.SlotMainHand: {
					ID: "TEST_BLADE", Name: "Cinder Test Blade", Slot: content.SlotMainHand,
					Element: content.ElementFire, Rune: string(content.ElementFire),
					Rarity: content.RarityEpic, MaxDurability: 100,
					Stats: content.Stats{STR: 60, INT: 20}, FoundAt: "2000-01-01T00:00:00Z",
				},
				content.SlotOffHand: {
					ID: "TEST_FOCUS", Name: "Tideglass Focus", Slot: content.SlotOffHand,
					Rarity: content.RarityEpic, MaxDurability: 80,
					Stats: content.Stats{STR: 40, INT: 60},
				},
				content.SlotFinger1: {
					ID: "TEST_RING", Name: "Prism Test Band", Slot: content.SlotFinger1,
					Rarity: content.RarityLegendary, MaxDurability: 70, Sockets: 2,
					Gemstones: []string{"Ruby", "Topaz II"}, Stats: content.Stats{DEF: 35, LCK: 12},
				},
			}
			contributions := abyssGearDamageContributions(equipped)
			views := make([]gearView, 0, len(equipped))
			for _, slot := range []content.GearSlot{content.SlotMainHand, content.SlotOffHand, content.SlotFinger1} {
				view := toGearView(slot, equipped[slot])
				view.Damage = contributions[slot]
				view.Durability = view.MaxDurability
				views = append(views, view)
			}
			fixture["Equipped"] = views
			mystery := content.Gear{
				ID: "TEST_MYSTERY", Name: "Veiled Test Relic", Slot: content.SlotCharm,
				Rarity: content.RarityEpic, MaxDurability: 60, Unidentified: true,
			}
			mysteryView := toGearView(mystery.Slot, mystery)
			mysteryView.InvID = 98
			fixture["Inventory"] = []gearView{mysteryView}
			fixture["ForgeWorkbenchEnabled"] = true
			fixture["ForgeOperations"] = abyssForgeOperations()
			fixture["ForgeWorkbench"] = abyssForgeWorkbenchData{
				SchemaVersion: 1,
				StatPresets:   map[string]map[string]int{"balanced": {"STR": 100, "INT": 100}},
				QueuePolicies: []string{"stop"},
				MaterialFlow:  map[string][]int64{},
				CraftCap:      10,
				PresetSlots:   1,
				CosmeticTheme: "apprentice",
			}
		}
		room := r.URL.Query().Get("room")
		if room == abyssForgeFloorType || room == abyssEventChainType || room == abyssCartographerEventType {
			run := fixture["Run"].(abyssRun)
			run.FloorType = "event"
			run.EventState = prepareAbyssEventForDepth(`{"type":"`+room+`"}`, run.Depth)
			fixture["Run"] = run
		}
		if err := server.tmpl.ExecuteTemplate(w, "abyss", fixture); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/abyss/plaza", func(w http.ResponseWriter, _ *http.Request) {
		plaza := abyssPlazaView{
			Catalog: make([]abyssPlazaCatalogView, 0, len(abyssPlazaCatalog)),
			Exhibits: []abyssPlazaExhibit{
				{
					Key: "obsidian_runestone", Name: "Obsidian Runestone", Tier: "Echo Court", Icon: "◆",
					Nickname: `<img src=x onerror="window.plazaInjected=true">`, GoldSpent: 2_500_000,
					AcquiredAt: "21 Aug 2026", Order: 2,
				},
			},
			Patrons: 12, Monuments: 18, GoldRetired: 42_750_000,
		}
		for _, monument := range abyssPlazaCatalog {
			plaza.Catalog = append(plaza.Catalog, abyssPlazaCatalogView{abyssPlazaMonument: monument})
		}
		if err := server.tmpl.ExecuteTemplate(w, "abyss-plaza", map[string]any{
			"Title": "Hall of Delvers", "Nav": "plaza", "EnableAbyss": true,
			"U": &webUser{UID: "plaza-e2e", Nickname: "Plaza Tester", Gold: 5_000_000}, "Plaza": plaza,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	registerAbyssTreeE2EFixture(mux, server)
	mux.HandleFunc("/abyss/ops", func(w http.ResponseWriter, _ *http.Request) {
		if err := server.tmpl.ExecuteTemplate(w, "abyssOps", map[string]any{"Title": "Abyss Operations", "Nav": "ops"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	port := strings.TrimSpace(os.Getenv("ABYSS_E2E_PORT"))
	if port == "" {
		port = "18082"
	}
	addr := "127.0.0.1:" + port
	t.Logf("Abyss E2E fixture listening on http://%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		t.Fatal(err)
	}
}
