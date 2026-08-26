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
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
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
		{name: "threshold", want: "4a6e8b3faae4284e3c407f835008feb4683e17df02615f60a1220bbf9d98ab22"},
		{name: "active_run", active: true, want: "552a9280684756ae79e425adf70d00c0853122b85c2e1ff42451e26ec688e2a0"},
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
			stable = bytes.ReplaceAll(stable, []byte("\r\n"), []byte("\n"))
			digest := fmt.Sprintf("%x", sha256.Sum256(stable))
			if digest != fixture.want {
				t.Fatalf("golden digest = %s, want %s", digest, fixture.want)
			}
		})
	}
}

func TestAbyssHistoryLootEscapesMarkup(t *testing.T) {
	if err := i18n.InitWithLocale(i18n.LocaleEnUS); err != nil {
		t.Fatalf("initialize locale bundle: %v", err)
	}
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	fixture := abyssGoldenFixture(false)
	fixture["History"] = []abyssHistoryRow{{
		Depth: 4, Tier: "normal", LootCount: 1,
		Loot: []string{`<img src=x onerror="alert(1)">`},
	}}
	var rendered bytes.Buffer
	if err := server.tmpl.ExecuteTemplate(&rendered, "abyss", fixture); err != nil {
		t.Fatalf("render Abyss fixture: %v", err)
	}
	page := rendered.String()
	if strings.Contains(page, `<img src=x onerror="alert(1)">`) {
		t.Fatal("history loot rendered as executable markup")
	}
	if !strings.Contains(page, `&lt;img src=x onerror=&#34;alert(1)&#34;&gt;`) {
		t.Fatal("escaped history loot label not found")
	}
}

func abyssGoldenFixture(active bool) map[string]any {
	stats := abyssStats{BestDepth: 57, Tokens: 42, LifetimeFloors: 321, LifetimeBanked: 654321}
	run := abyssRun{Tier: "normal", FloorType: "combat"}
	campaign := abyssSeasonCampaignAt(time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC))
	seasonJourney := abyssSeasonJourneyView{
		ID: campaign.ID, Name: campaign.Name, Icon: campaign.Icon,
		Affinity: campaign.Affinity, Palette: campaign.Palette, Tagline: campaign.Tagline,
		StartLabel: campaign.Start.Format("02 Jan 2006"), EndLabel: campaign.End.Add(-time.Second).Format("02 Jan 2006"),
		CurrentWeek: campaign.CurrentWeek,
	}
	for week := 1; week <= abyssSeasonWeeks; week++ {
		progress := int64(0)
		percent := 0
		if week == 1 {
			progress = abyssSeasonWeekGoals[0]
			percent = 100
		}
		seasonJourney.Weeks = append(seasonJourney.Weeks, abyssSeasonRewardView{
			Week: week, Name: campaign.RewardWord + " " + abyssSeasonRewardNames[week-1],
			Kind: abyssSeasonRewardKinds[week-1], Goal: abyssSeasonWeekGoals[week-1],
			Progress: progress, Percent: percent,
			Available: week <= campaign.CurrentWeek, Complete: week == 1, Current: week == campaign.CurrentWeek,
		})
	}
	if active {
		run.Active = true
		run.Depth = 12
		run.Escrow = 3456
		run.Momentum = 4
		run.CurHP = 750
		run.MaxHP = 1000
	}
	bossAffinity := abyssBossAffinityForecast(run, time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC))
	elementalPreview := abyssElementalPreview(
		bossAffinity,
		map[content.GearSlot]content.Gear{
			content.SlotMainHand: {Element: content.Element(bossAffinity.WeakTo)},
		},
	)
	fixtureSkills := []content.Skill{
		{ID: "S0_1", Name: "Storm Blast", Type: content.SkillMagic, Rarity: content.RarityCommon, ManaCost: 20, CooldownRounds: 1},
		{ID: "S0_2", Name: "Wind Curse", Type: content.SkillDebuff, Rarity: content.RarityRare, ManaCost: 25, CooldownRounds: 2},
		{ID: "S0_3", Name: "Icy Heal", Type: content.SkillBuff, Rarity: content.RarityEpic, ManaCost: 30, CooldownRounds: 3},
	}
	skillPriority := abyssSkillPriorityViewForSkills(fixtureSkills, fixtureSkills)
	return map[string]any{
		"Title": "The Abyss", "Nav": "abyss",
		"U": &webUser{
			UID: "golden-player", Nickname: "Golden Delver", Level: 100,
			Gold: 123456, AbyssTokens: 42, CurrentHP: 750, MaxHP: 1000,
		},
		"Stats": stats, "Run": run, "RegenPerSec": 0.0, "AutoFocus": "balanced",
		"Tiers": abyssTierList(stats.BestDepth), "Leaders": abyssBoards{}, "Season": "S1", "SeasonJourney": seasonJourney,
		"History": []any{}, "Achievements": []abyssAchievementView{}, "BadgeOptions": []any{},
		"RunInsights": abyssRunInsightsView{}, "LongTerm": abyssLongTermView{},
		"CartographerRoute": abyssCartographerRouteView{Floors: []abyssCartographerFloorView{}},
		"EnemyForecast":     abyssEnemyForecast("golden-player", run, nil),
		"BossAffinity":      bossAffinity, "ElementalPreview": elementalPreview, "SkillPriority": skillPriority,
		"ActiveBadge": "", "ActiveBadgeName": "", "LoreList": []any{}, "LoreTotal": len(abyssLoreFragments),
		"Bestiary": []any{}, "Consumables": []any{}, "DailyMod": "",
		"CommunityExpedition": map[string]any{"Week": "2026-W35", "Floors": 0, "Target": 1000},
		"Helpers":             []any{}, "NextIsBoss": false, "AbyssSetPieces": 0, "AbyssSetTier": 0,
		"PredatorPieces": 0, "PredatorTier": 0, "WardenPieces": 0, "WardenTier": 0,
		"HarvesterPieces": 0, "HarvesterTier": 0, "Bounty": nil, "Shop": []any{},
		"Pacts": []any{}, "PactProgram": abyssPactProgramStateFromAt(nil, nil, time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)),
		"Equipped": []gearView{}, "Inventory": []gearView{},
		"LegendaryPity": 0, "FeaturedDrops": abyssWeeklyFeaturedDrops(time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)),
		"Wishlist":     abyssWishlistViewFor(abyssWishlistState{}, ""),
		"SetPityPanel": abyssSetPityPanel(nil, nil, nil),
		"DropStreak":   0, "DropStreakBonusPct": 0, "Risk": 0,
		"RunLoot": []runLootRow{}, "CanLastStand": false, "Materials": map[string]int{},
		"MaterialDefs": []any{}, "Recipes": []any{},
		"CraftQuest": map[string]int{"Done": 0, "Target": 5},
		"Sanctuary":  map[string]int{}, "SanctuaryDefs": []any{}, "SanctuaryStage": 0,
		"SanctuaryStageName": "Dormant", "ProgressionTracks": []any{}, "Spec": "",
		"SpecDefs": []any{}, "ForgeHistory": []any{},
		"ForgeRep": map[string]int{"Rep": 0, "DiscountPct": 0}, "ForgeHappyHour": false,
		"ForgeCatalog": map[string]any{}, "ForgeOperations": []any{},
		"ForgeWorkbenchEnabled": false, "ForgeWorkbench": map[string]any{}, "AutoRepair": false, "FreeID": false,
		"RepairAllCost": int64(0),
		"TokenBuyGold":  int64(100), "TokenSellGold": int64(50),
		"PrestigeTier":     map[string]string{"Name": "", "Aura": ""},
		"CraftLegendaries": []any{}, "LBTier": "normal", "LBTiers": abyssTierList(999),
		"LastStandCost": int64(10), "TalentMaxLevel": content.TalentMaxLevel,
		"NodeGates": map[string]int{}, "Checkpoints": []int{10, 20, 30, 40, 50},
		"ExpressStart": 52, "ExpressCost": int64(5200),
	}
}
