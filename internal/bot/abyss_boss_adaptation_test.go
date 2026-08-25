package bot

import (
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestAbyssBossAdaptationStartsAtTenKillsAndStaysStable(t *testing.T) {
	name := "Gorgoroth the Firelord"
	if _, active := abyssBossAdaptationFor(name, abyssBossAdaptationKills-1); active {
		t.Fatal("adaptation activated before ten defeats")
	}
	first, active := abyssBossAdaptationFor(name, abyssBossAdaptationKills)
	if !active {
		t.Fatal("adaptation did not activate at ten defeats")
	}
	second, active := abyssBossAdaptationFor(name, 99)
	if !active || second != first {
		t.Fatalf("adaptation changed with kill count: first %#v, second %#v", first, second)
	}
}

func TestAbyssBossAdaptationForecastHandlesTwinsAndSecretReplacement(t *testing.T) {
	run := abyssRun{Depth: 64}
	names := abyssBossNamesAtDepth(65)
	counts := map[string]int{names[0]: 10, names[1]: 7}
	view := abyssBossAdaptationForecast(run, abyssSecretBossChainView{}, counts)
	if view.TargetDepth != 65 || len(view.Bosses) != 2 {
		t.Fatalf("twin forecast = %#v", view)
	}
	if !view.Bosses[0].Active || view.Bosses[1].Active || view.Bosses[1].Remaining != 3 {
		t.Fatalf("twin adaptation progress = %#v", view.Bosses)
	}

	chain := abyssSecretBossChainView{Unlocked: true, Stage: 1, NextDepth: 65}
	secret := abyssBossAdaptationForecast(run, chain, map[string]int{abyssSecretBosses[1].Name: 10})
	if len(secret.Bosses) != 1 || secret.Bosses[0].Name != abyssSecretBosses[1].Name || !secret.Bosses[0].Active {
		t.Fatalf("secret replacement forecast = %#v", secret)
	}
}

func TestApplyAbyssBossAdaptationsUsesRecordedBossKills(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectQuery("SELECT boss_name,COUNT").WithArgs("delver").WillReturnRows(
		sqlmock.NewRows([]string{"boss_name", "count"}).
			AddRow("Gorgoroth the Firelord", 10).
			AddRow("Malakor the Voidweaver", 9),
	)
	mobs := []content.Mob{
		{Name: "Gorgoroth the Firelord", Type: content.MobBoss},
		{Name: "Malakor the Voidweaver", Type: content.MobBoss},
		{Name: "Minion", Type: content.MobCommon},
	}
	updated, logs := (&Bot{DB: database}).applyAbyssBossAdaptations("delver", mobs)
	if len(updated[0].Effects) != 1 || len(updated[1].Effects) != 0 || len(updated[2].Effects) != 0 {
		t.Fatalf("adapted effects = %#v", updated)
	}
	if len(logs) != 1 || !strings.Contains(logs[0], "remembers 10 defeats") {
		t.Fatalf("adaptation logs = %v", logs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLiveBossAdaptationDoesNotReportExistingPersistentEffect(t *testing.T) {
	mob := &content.Mob{Type: content.MobBoss, Effects: []content.MobEffect{content.EffectArmored}}
	combat := &abyssLiveCombat{actionCounts: map[string]int{"attack": 3}}
	if logs := combat.applyLiveBossAdaptation([]*content.Mob{mob}, nil); len(logs) != 0 {
		t.Fatalf("duplicate live adaptation logs = %v", logs)
	}
	if len(mob.Effects) != 1 {
		t.Fatalf("duplicate persistent effect = %v", mob.Effects)
	}
}

func TestAbyssBossAdaptationUIContract(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss_boss_adaptation.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_boss_adaptation.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"BOSS MEMORY", "confirmed defeat", "permanent trick", ".Trick"} {
		if !strings.Contains(string(page), token) {
			t.Errorf("adaptation forecast is missing %q", token)
		}
	}
	for _, token := range []string{".ab-adaptation-forecast", ".ab-adaptation-boss.is-active", "@media (max-width: 480px)"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("adaptation styles are missing %q", token)
		}
	}
}
