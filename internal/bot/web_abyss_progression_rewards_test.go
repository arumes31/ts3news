package bot

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestBuildAbyssProgressionPointRewards(t *testing.T) {
	t.Parallel()

	rewards := buildAbyssProgressionPointRewards(60, 250, map[string]int64{
		"normal": 100, "nightmare": 99,
	})
	if rewards.Points != 57 {
		t.Fatalf("reward points = %d, want 57", rewards.Points)
	}
	if len(rewards.Sources) != 6 {
		t.Fatalf("point sources = %d, want 6", len(rewards.Sources))
	}
	if source := rewards.Sources[0]; source.Key != "achievements" || source.Earned != abyssAchievementPointCap || source.NextReward != 0 {
		t.Fatalf("achievement source = %#v", source)
	}
	if source := rewards.Sources[1]; source.Key != "weekly_talent_xp" || source.Earned != 2 || source.Progress != 50 {
		t.Fatalf("weekly source = %#v", source)
	}
	if source := rewards.Sources[2]; source.Key != "tier_normal" || source.Earned != abyssTierMasteryPointReward || source.NextReward != 0 {
		t.Fatalf("normal mastery = %#v", source)
	}
	if source := rewards.Sources[3]; source.Key != "tier_nightmare" || source.Earned != 0 || source.Progress != 99 {
		t.Fatalf("nightmare mastery = %#v", source)
	}
}

func TestAbyssWeeklyTalentXPUsesISOWeekAndCap(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	at := time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC)
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(abyssWeeklyTalentXPKey("weekly-player", at), abyssWeeklyTalentXPPerFloor, abyssWeeklyTalentXPCap).
		WillReturnResult(sqlmock.NewResult(1, 1))
	(&Bot{DB: database}).awardAbyssWeeklyTalentXP("weekly-player", at)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssNewPlayerXPStopsAfterTenClears(t *testing.T) {
	t.Parallel()

	for _, floors := range []int64{0, 1, 9} {
		if got := abyssNewPlayerXPPercent(floors); got != 100 {
			t.Errorf("%d lifetime floors bonus = %d, want 100", floors, got)
		}
	}
	for _, floors := range []int64{-1, 10, 100} {
		if got := abyssNewPlayerXPPercent(floors); got != 0 {
			t.Errorf("%d lifetime floors bonus = %d, want 0", floors, got)
		}
	}
}

func TestAbyssNewPlayerXPReachesCombatAndEventFloors(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(source), "abyssNewPlayerXPPercent(st.LifetimeFloors)"); got != 2 {
		t.Fatalf("new-player XP applications = %d, want combat and non-combat paths", got)
	}
}
