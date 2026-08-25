package bot

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"testing"

	"ts3news/internal/i18n"
)

func TestAbyssTierCatalogMatchesLatestDatabaseConstraints(t *testing.T) {
	t.Parallel()

	root := abyssAAARepositoryRoot(t)
	migrations, err := filepath.Glob(filepath.Join(root, "internal", "db", "migrations", "*.up.sql"))
	if err != nil {
		t.Fatalf("find database migrations: %v", err)
	}
	sort.Strings(migrations)
	constraintPattern := regexp.MustCompile(
		`(?is)ALTER\s+TABLE\s+(abyss_active|abyss_runs|abyss_boss_kills)\s+` +
			`ADD\s+CONSTRAINT\s+\S+\s+CHECK\s*\(\s*tier\s+IN\s*\(([^)]*)\)\s*\)`,
	)
	valuePattern := regexp.MustCompile(`'([^']+)'`)
	latest := make(map[string][]string)
	for _, migration := range migrations {
		content, err := os.ReadFile(migration)
		if err != nil {
			t.Fatalf("read migration %s: %v", filepath.Base(migration), err)
		}
		for _, match := range constraintPattern.FindAllStringSubmatch(string(content), -1) {
			values := valuePattern.FindAllStringSubmatch(match[2], -1)
			tiers := make([]string, 0, len(values))
			for _, value := range values {
				tiers = append(tiers, value[1])
			}
			sort.Strings(tiers)
			latest[match[1]] = tiers
		}
	}

	want := append([]string(nil), abyssTierOrder...)
	sort.Strings(want)
	for _, table := range []string{"abyss_active", "abyss_runs", "abyss_boss_kills"} {
		if !reflect.DeepEqual(latest[table], want) {
			t.Errorf("latest %s tier constraint = %v, catalog = %v", table, latest[table], want)
		}
	}
}

func TestAbyssPageGoldenFixtures(t *testing.T) {
	if err := i18n.InitWithLocale(i18n.LocaleEnUS); err != nil {
		t.Fatalf("initialize locale bundle: %v", err)
	}
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}

	fixtures := []struct {
		name   string
		active bool
		want   string
	}{
		{name: "threshold", want: "404fe1fd1506c08cd1b1981a6a9fd91d095202d0fca2bbd5bf3dc3f33998ccfc"},
		{name: "active_run", active: true, want: "ec5775bbb9a21f676a0fbaa966ceeaeb3346e394a3d620128e9c0f9dfa25fe42"},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			var rendered bytes.Buffer
			if err := server.tmpl.ExecuteTemplate(
				&rendered,
				"abyss",
				abyssGoldenFixture(fixture.active),
			); err != nil {
				t.Fatalf("render Abyss fixture: %v", err)
			}
			stable := regexp.MustCompile(`[0-9a-f]{12}`).ReplaceAll(
				rendered.Bytes(),
				[]byte("<asset-version>"),
			)
			digest := fmt.Sprintf("%x", sha256.Sum256(stable))
			if digest != fixture.want {
				t.Fatalf("golden digest = %s, want %s", digest, fixture.want)
			}
		})
	}
}

func abyssGoldenFixture(active bool) map[string]any {
	stats := abyssStats{BestDepth: 57, Tokens: 42, LifetimeFloors: 321, LifetimeBanked: 654321}
	run := abyssRun{Tier: "normal", FloorType: "combat"}
	if active {
		run.Active = true
		run.Depth = 12
		run.Escrow = 3456
		run.Momentum = 4
		run.CurHP = 750
		run.MaxHP = 1000
	}
	return map[string]any{
		"Title": "The Abyss", "Nav": "abyss",
		"U": &webUser{
			UID: "golden-player", Nickname: "Golden Delver", Level: 100,
			Gold: 123456, AbyssTokens: 42, CurrentHP: 750, MaxHP: 1000,
		},
		"Stats": stats, "Run": run, "RegenPerSec": 0.0, "AutoFocus": "balanced",
		"Tiers": abyssTierList(stats.BestDepth), "Leaders": abyssBoards{}, "Season": "S1",
		"History": []any{}, "Achieved": []string{}, "BadgeOptions": []any{},
		"ActiveBadge": "", "ActiveBadgeName": "", "LoreList": []any{},
		"Bestiary": []any{}, "Consumables": []any{}, "DailyMod": "",
		"CommunityExpedition": map[string]any{"Week": "2026-W35", "Floors": 0, "Target": 1000},
		"Helpers": []any{}, "NextIsBoss": false, "AbyssSetPieces": 0, "AbyssSetTier": 0,
		"PredatorPieces": 0, "PredatorTier": 0, "WardenPieces": 0, "WardenTier": 0,
		"HarvesterPieces": 0, "HarvesterTier": 0, "Bounty": nil, "Shop": []any{},
		"Pacts": []any{}, "Equipped": []gearView{}, "Inventory": []gearView{},
		"LegendaryPity": 0, "DropStreak": 0, "DropStreakBonusPct": 0, "Risk": 0,
		"RunLoot": []runLootRow{}, "CanLastStand": false, "Materials": map[string]int{},
		"MaterialDefs": []any{}, "Recipes": []any{},
		"CraftQuest": map[string]int{"Done": 0, "Target": 5},
		"Sanctuary": map[string]int{}, "SanctuaryDefs": []any{}, "SanctuaryStage": 0,
		"SanctuaryStageName": "Dormant", "ProgressionTracks": []any{}, "Spec": "",
		"SpecDefs": []any{}, "ForgeHistory": []any{},
		"ForgeRep": map[string]int{"Rep": 0, "DiscountPct": 0}, "ForgeHappyHour": false,
		"ForgeCatalog": map[string]any{}, "ForgeOperations": []any{},
		"ForgeWorkbenchEnabled": false, "ForgeWorkbench": map[string]any{}, "AutoRepair": false,
		"TokenBuyGold": int64(100), "TokenSellGold": int64(50),
		"PrestigeTier": map[string]string{"Name": "", "Aura": ""},
		"CraftLegendaries": []any{}, "LBTier": "normal", "LBTiers": abyssTierList(999),
		"LastStandCost": int64(10), "NodeGates": map[string]int{}, "Checkpoints": []int{10, 20, 30, 40, 50},
		"ExpressStart": 52, "ExpressCost": int64(5200),
	}
}
