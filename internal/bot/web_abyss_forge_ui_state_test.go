package bot

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestAbyssForgeHappyHourCountdown(t *testing.T) {
	tests := []struct {
		name     string
		now      time.Time
		active   bool
		startsIn int
		endsIn   int
	}{
		{name: "before", now: time.Date(2026, 8, 25, 17, 30, 0, 0, time.UTC), startsIn: 30 * 60},
		{name: "active", now: time.Date(2026, 8, 25, 18, 15, 0, 0, time.UTC), active: true, endsIn: 45 * 60},
		{name: "after", now: time.Date(2026, 8, 25, 20, 0, 0, 0, time.UTC), startsIn: 22 * 60 * 60},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			active, startsIn, endsIn := abyssForgeHappyHourCountdown(test.now)
			if active != test.active || startsIn != test.startsIn || endsIn != test.endsIn {
				t.Fatalf("countdown = (%t, %d, %d), want (%t, %d, %d)", active, startsIn, endsIn, test.active, test.startsIn, test.endsIn)
			}
		})
	}
}

func TestAbyssForgeUIPresentationContracts(t *testing.T) {
	paths := []string{
		"web.go",
		"web_abyss_forge_ui_state.go",
		"webassets/abyss.html",
		"webassets/abyss_forge_experience.html",
		"webassets/abyss_ui200.css",
	}
	var source strings.Builder
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		source.Write(data)
	}
	for _, required := range []string{
		"/api/abyss/forge/state",
		"forgeSelectedCard",
		"updateOptgroupCounts",
		"abyssForgeCommit",
		"temper_fail_stacks",
		"ab-coinflip",
		"selForgeStatBlock",
		"AB_IMBUE_DESC",
		"updateBrandProgress",
		"updateSwapCards",
		"updateInfusePreview",
		"updatePrismaticPreview",
		"updateRebalancePreview",
		"data-gemstones",
		"forgeGemPreview",
		"forgeHistoryFilter",
		"AB_RARITY_NAMES",
		"undo_available",
		"rep_to_next_discount",
		"happy_hour_ends_in_seconds",
		"AB_MAT_SOURCES",
	} {
		if !strings.Contains(source.String(), required) {
			t.Errorf("forge UI contract missing %q", required)
		}
	}
	for _, forbidden := range []string{"ab_temper_pity", "ab_forge_undo", "data-first-gem"} {
		if strings.Contains(source.String(), forbidden) {
			t.Errorf("forge UI still trusts browser-only state %q", forbidden)
		}
	}
}

func TestUnidentifiedForgeViewHidesMetadata(t *testing.T) {
	gear := content.Gear{
		ID: "secret", Name: "Secret Crown", Slot: content.SlotHead,
		Rarity: content.RarityLegendary, Stats: content.Stats{STR: 999},
		Sockets: 2, Gemstones: []string{"Ruby III"}, Quality: 5,
		KillCount: 400, MilestoneTier: 2, Unidentified: true, FoundAt: "2000-01-01T00:00:00Z",
	}
	view := toGearView(gear.Slot, gear)
	if view.Name != "Unidentified Head" || view.Rarity != "Unknown" || view.CR != 0 || view.Score != 0 {
		t.Fatalf("unidentified identity leaked: %#v", view)
	}
	if view.RarityIcon != "crystal-ball" || view.RarityColor != "#8c96aa" {
		t.Fatalf("unidentified silhouette = %q %q", view.RarityIcon, view.RarityColor)
	}
	if view.StatsJSON != "{}" || view.GemstonesJSON != "[]" || view.Sockets != 0 || len(view.Gemstones) != 0 {
		t.Fatalf("unidentified payload leaked: %#v", view)
	}
	if view.Quality != 0 || view.SetID != "" || view.Temper != 0 || view.HasSpecial || view.HasRune {
		t.Fatalf("unidentified progression leaked: %#v", view)
	}
	if view.ID != "" || view.MaxDurability != 0 || view.Durability != 0 || view.AHPrice != 0 || view.VendorPrice != 0 || view.Insured || view.XPBonusPct != 0 || view.BrokenIn {
		t.Fatalf("unidentified economy metadata leaked: %#v", view)
	}
}

func TestLoadAbyssForgeUIState(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectQuery("SELECT temper_fail_stacks").WithArgs("player").WillReturnRows(
		sqlmock.NewRows([]string{"temper_fail_stacks", "has_undo", "undo_used", "forge_rep"}).AddRow(7, true, false, 63),
	)

	state, err := (&Bot{DB: db}).loadAbyssForgeUIState("player", time.Date(2026, 8, 25, 18, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if state.TemperPityBonusPct != 35 || !state.UndoAvailable || state.UndoUsedToday {
		t.Fatalf("forge state = %#v", state)
	}
	if state.DiscountPct != 10 || state.NextDiscountRep != 75 || state.RepToNextDiscount != 12 {
		t.Fatalf("forge reputation state = %#v", state)
	}
	if !state.HappyHour || state.HappyHourEndsInSeconds != 30*60 {
		t.Fatalf("forge happy-hour state = %#v", state)
	}
	if state.AccountDiscountPct != 30 {
		t.Fatalf("account discount = %d, want 30", state.AccountDiscountPct)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
