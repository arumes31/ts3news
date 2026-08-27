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
	fontSize := "m"
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
	mux.HandleFunc("/api/abyss/preferences/font-size", func(w http.ResponseWriter, r *http.Request) {
		wishlistMu.Lock()
		defer wishlistMu.Unlock()
		if r.Method == http.MethodPost {
			var request struct {
				FontSize string `json:"font_size"`
			}
			if readJSON(r, &request) != nil || normalizeAbyssFontSize(request.FontSize) != request.FontSize {
				writeJSON(w, map[string]any{"ok": false, "error": "invalid font size"})
				return
			}
			fontSize = request.FontSize
		} else if r.Method != http.MethodGet {
			http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]any{"ok": true, "font_size": fontSize})
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
		buybackView := toGearView(charm.Slot, charm)
		buybackView.Durability = 41
		if err := server.tmpl.ExecuteTemplate(w, "inventory", map[string]any{
			"Title": "Inventory", "Nav": "inventory", "EnableAbyss": true,
			"U":     &webUser{UID: "inventory-e2e", Nickname: "Inventory Tester", Gold: 25_000},
			"Items": []gearView{charmView, mysteryView}, "Consumables": []consumableView{},
			"Buybacks": []vendorBuybackView{{
				gearView: buybackView, BuybackID: 77, BuybackCost: 11_000,
				SaleValue: 10_000, SoldAt: "25 Aug · 18:30 UTC",
			}},
			"Pouch": abyssPouchView{
				Owned: true, Equipped: true, Level: 1, MaxLevel: 3, StackCap: 6, CarryCap: 9,
				NextStack: 7, NextCarry: 10, NextLevel: 2, NextCost: 1_000_000, CanUpgrade: true,
				StatusLabel: "Equipped · carry expansion active.",
			},
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
		skill, _ := content.GetSkillByID("S_EQ")
		if err := server.tmpl.ExecuteTemplate(w, "armory", map[string]any{
			"Title": "Armoury", "Nav": "armory", "EnableAbyss": true, "U": u,
			"Slots":  []gearView{weaponView, toGearView(mystery.Slot, mystery)},
			"Skills": []content.Skill{skill}, "Ultimates": []any{}, "Artifact": nil,
			"PlayerTitle": nil, "Pets": []any{},
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/abyss", func(w http.ResponseWriter, r *http.Request) {
		fixture := abyssGoldenFixture(r.URL.Query().Get("active") == "1")
		if chronicle := r.URL.Query().Get("chronicle"); chronicle != "" {
			run := fixture["Run"].(abyssRun)
			flags := map[string]int64{
				abyssRunFlagStoryCampaign: 1,
				abyssRunRelicFlag(1):      1,
				abyssRunBoonFlag(2):       2,
			}
			if chronicle == "draft" {
				run.Depth = 5
				run.FloorType = "combat"
				flags[abyssRunFlagBoonDraftDepth] = 5
			} else if chronicle == "rest" {
				run.Depth = 6
				run.FloorType = "rest"
			}
			fixture["Run"] = run
			fixture["RunIdentity"] = abyssRunIdentityViewFrom(run, flags)
		}
		if r.URL.Query().Get("endless") == "1" {
			retention := fixture["Retention"].(abyssRetentionView)
			retention.Endless = abyssEndlessProgram(175, map[string]bool{})
			fixture["Retention"] = retention
		}
		fixture["Competition"] = abyssCompetitionView{
			Tier: "normal", Period: "season", PeriodLabel: "2026-S3",
			Builds: []string{"initiate", "delver", "plunderer", "warden"},
			Boards: []abyssCompetitionBoard{
				{Key: "depth", Title: "Deepest descents", TieBreak: "Depth, then banked gold."},
				{Key: "speed", Title: "Depth 20 speedruns", TieBreak: "Duration, then depth."},
				{Key: "economy", Title: "Weekly vault", TieBreak: "Gold, then depth."},
				{Key: "pact", Title: "Pact survival", TieBreak: "Multiplier, then depth."},
				{Key: "bestiary", Title: "Bestiary families", TieBreak: "Kills, then family."},
				{Key: "shame", Title: "Hall of shame", TieBreak: "Depth, then date."},
				{Key: "streak", Title: "Bank streaks", TieBreak: "Streak, then gold."},
				{Key: "pets", Title: "Companion power", TieBreak: "Power, then level."},
			},
			Wagers: []abyssCompetitionWagerView{{Bracket: 1_000, Fee: 1_000, Entrants: 7, Pool: 7_000}},
		}
		fixture["Tiers"] = abyssTierListWithRates(999, []abyssTierRateView{{Tier: "normal", Wins: 7, Runs: 10, Percent: 70}})
		fixture["FreeID"] = true
		shopNow := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
		weeklyKey := abyssWeeklyInsanityCosmetic(shopNow)
		weeklyItem := abyssShopIndex[weeklyKey]
		shopItems := []abyssShopItemView{
			{
				abyssShopItem: abyssShopCatalog[0], EffectiveCost: abyssShopDemandCost(abyssShopCatalog[0].Cost, 10),
				DemandPct: 10, DemandSales: 24,
			},
			{
				abyssShopItem: weeklyItem, EffectiveCost: weeklyItem.Cost, Insanity: true,
				RotationWeek: abyssEconomyWeek(shopNow),
				RotationEnds: abyssWeeklyCosmeticReset(shopNow).Format("2006-01-02 15:04 UTC"),
			},
		}
		fixture["Shop"] = shopItems
		flashItem := shopItems[0]
		flashItem.HappyAccident = true
		fixture["ShopProgram"] = abyssShopProgramView{
			Gold: 123_500, Tokens: 572, LoyaltyPunches: 0, LoyaltyTarget: abyssShopLoyaltyPurchases,
			GiftFeeGold: abyssShopGiftFeeGold, Bundles: abyssShopBundles, Flash: &flashItem,
			FlashEnds: "22 Aug 00:00 UTC", SeasonExchangeCost: abyssSeasonExchangeCost,
			SeasonExchangeName: "Ember Legacy Cache",
			Materials: []abyssShopCurrencyMaterial{
				{ID: "dust", Name: "Abyssal Dust", Icon: "🌫️", Count: 40},
				{ID: "shard", Name: "Void Shard", Icon: "🔷", Count: 12},
				{ID: "core", Name: "Umbral Core", Icon: "🟣", Count: 4},
				{ID: "prism", Name: "Eldritch Prism", Icon: "💠", Count: 1},
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
		fixture["BestKill"] = abyssBestKillView{
			Available: true, Boss: "The Fixture Colossus", Depth: 57,
			KillTimeMS: 12_300, KillTime: "12.3s", Tier: "normal", TierName: "Normal", KilledAt: "2026-08-25",
		}
		fixture["BossCosmetics"] = abyssBossCosmeticCollectionView{
			Items: []abyssBossCosmeticView{{abyssBossCosmetic: abyssBossCosmeticCatalog[0], Owned: true}},
			Owned: 1, Total: len(abyssBossCosmeticCatalog), Rates: "Normal 2% · Nightmare 4% · Hell 7% · Insanity 12%",
		}
		fixture["Social"] = abyssSocialHubView{
			SecondPetUnlocked: true,
			Pets: []abyssSocialPetView{
				{
					ID: 101, Name: "Ember", Type: string(content.MobElite), Level: 18,
					HP: 4200, MaxHP: 5000, STR: 860, DEF: 620, SPD: 740, Loyalty: 82,
					ActiveSlot: 1, Mood: "content", MoodIcon: "😊", MoodPct: 2,
					Equipment: "Collar: Ember Chain · +20 STR · Charm: empty", Ability: "Predator's Rush", Class: "damage",
					XP: 640, XPNext: 1000, LoyaltyPct: 8, FusionRank: 1, BarkStyle: "gentle",
					Cosmetics: []abyssPetCosmeticView{
						{abyssPetCosmetic: abyssPetCosmetics[0], Owned: true, Selected: true},
						{abyssPetCosmetic: abyssPetCosmetics[1]},
					},
				},
				{
					ID: 102, Name: "Cinder", Type: string(content.MobElite), Level: 12,
					HP: 3100, MaxHP: 3100, STR: 580, DEF: 440, SPD: 510, Loyalty: 70,
					Mood: "content", MoodIcon: "😊", MoodPct: 2, Equipment: "Collar: empty · Charm: empty",
					Ability: "Reserve", Class: "damage", XP: 220, XPNext: 700, LoyaltyPct: 7, BarkStyle: "bold",
					Cosmetics: []abyssPetCosmeticView{
						{abyssPetCosmetic: abyssPetCosmetics[0]},
						{abyssPetCosmetic: abyssPetCosmetics[1]},
					},
				},
				{
					ID: 103, Name: "Spark", Type: string(content.MobElite), Level: 7,
					HP: 1800, MaxHP: 1800, STR: 320, DEF: 260, SPD: 300, Loyalty: 61,
					Mood: "content", MoodIcon: "😊", MoodPct: 2, Equipment: "Collar: empty · Charm: empty",
					Ability: "Reserve", Class: "damage", XPNext: 400, LoyaltyPct: 1, BarkStyle: "quiet",
					Cosmetics: []abyssPetCosmeticView{{abyssPetCosmetic: abyssPetCosmetics[0]}},
				},
			},
			Memorials:       []abyssMemorialView{{ID: 91, Name: "Ash", Type: string(content.MobMiniboss), Level: 9, Loyalty: 66, When: "Aug 20 12:00"}},
			WeeklyBoss:      abyssWeeklyBossView{Name: "The Fixture Colossus", HP: 750_000, MaxHP: 1_000_000, Percent: 75, Multiplier: 1},
			PetFeedOptions:  []consumableOwned{{ID: "small_health_potion", Name: "Small Health Potion", Count: 2}},
			PetPowerLeaders: []abyssPetPowerView{{Rank: 1, Nick: "Fixture Delver", Name: "Ember", Power: 7220}},
		}
		fixture["Bestiary"] = []abyssBestiaryRow{{
			MobName: "Fixture Stalker", Family: string(content.MobElite), Kills: 25,
			Milestone: "Studied", NextMilestone: 50, KillsToNext: 25,
			CapturePct: abyssBestiaryCapturePercent(string(content.MobElite)),
		}}
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
	mux.HandleFunc("/ah", func(w http.ResponseWriter, _ *http.Request) {
		listing := ahListingView{
			ID: "listing-history", ItemType: "gear", ItemID: "TEST_BLADE", Icon: "⚔", Name: "Cinder Test Blade",
			Price: 1600, Seller: "Market Tester", Listed: "Aug 25", Rarity: "Epic", RarityColor: "#b56cff",
			PriceHistory: buildAHPriceHistory([]int64{900, 1200, 1000, 1600}),
		}
		if err := server.tmpl.ExecuteTemplate(w, "ah", map[string]any{
			"Title": "Auction House", "Nav": "ah", "EnableAbyss": true,
			"U":      &webUser{UID: "ah-e2e", Nickname: "Market Tester", Gold: 25_000},
			"Active": []ahListingView{listing}, "Mine": []ahListingView{}, "History": []ahHistoryView{},
			"Sellable": []gearView{}, "Economy": abyssAHEconomyView{},
			"SearchQuery": "", "UpgradesOnly": false, "InsanityOnly": false,
			"CurrentPage": 1, "TotalPages": 1, "TotalCount": 1, "PrevPage": 1, "NextPage": 1,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	const spectatorSession = "0123456789abcdef0123456789abcdef"
	mux.HandleFunc("/abyss/spectate", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("session") != spectatorSession {
			http.Error(w, "invalid spectator link", http.StatusBadRequest)
			return
		}
		if err := server.tmpl.ExecuteTemplate(w, "abyssSpectate", map[string]any{
			"Title": "Abyss Spectator", "Nav": "abyss", "SessionID": spectatorSession,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/api/abyss/spectate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Get("session") != spectatorSession {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid spectator link"})
			return
		}
		writeJSON(w, map[string]any{
			"ok": true, "phase": "active", "round": 7,
			"allies": []map[string]any{
				{"id": "ally:1", "name": "Fixture Delver", "hp": 72_500, "max_hp": 100_000},
				{"id": "ally:2", "name": "Summoned Frost Lich", "hp": 18_400, "max_hp": 22_000},
			},
			"enemies": []map[string]any{
				{"id": "enemy:1", "name": `<img src=x onerror="window.spectatorInjected=true">`, "hp": 31_250, "max_hp": 125_000},
				{"id": "enemy:2", "name": "Ashen Gatekeeper", "hp": 48_000, "max_hp": 60_000},
			},
			"recent_logs": []string{
				"Fixture Delver strikes for 12,500.",
				`<svg onload="window.spectatorLogInjected=true"> hostile log payload`,
			},
		})
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
