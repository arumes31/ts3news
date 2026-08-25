package bot

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssCommunityWeekendUnlocksOnlyOnWeekend(t *testing.T) {
	t.Parallel()

	saturday := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	start, end, week := abyssCommunityWeekendWindow(saturday)
	if start != time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC) || end != time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC) || week != "2026-W35" {
		t.Fatalf("window = %v .. %v, %q", start, end, week)
	}
	locked := abyssCommunityWeekendStateFrom(saturday, abyssCommunityWeekendGoal-1)
	if locked.Unlocked || locked.Active || locked.Remaining != 1 || locked.Percent != 99 {
		t.Fatalf("locked state = %#v", locked)
	}
	active := abyssCommunityWeekendStateFrom(saturday, abyssCommunityWeekendGoal)
	if !active.Unlocked || !active.Active || active.Percent != 100 {
		t.Fatalf("active state = %#v", active)
	}
	friday := abyssCommunityWeekendStateFrom(saturday.AddDate(0, 0, -1), abyssCommunityWeekendGoal)
	if !friday.Unlocked || friday.Active {
		t.Fatalf("Friday state = %#v", friday)
	}
}

func TestAbyssCommunityWeekendUsesAuthoritativeBankedGold(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	saturday := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(gold_banked\\),0\\) FROM abyss_runs").
		WithArgs(time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(abyssCommunityWeekendGoal))
	state := (&Bot{DB: db}).abyssCommunityWeekendState(saturday)
	if !state.Active {
		t.Fatalf("community state = %#v", state)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssCommunityWeekendRewardAndClientContract(t *testing.T) {
	t.Parallel()

	server, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(server), "abyssCommunityWeekendRewardMult(time.Now().UTC())") != 2 {
		t.Fatal("combat and non-combat rewards do not share the community weekend multiplier")
	}
	module, err := webAssets.ReadFile("webassets/abyss_pact_program.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"communityAffixWeekend", "community_weekend", "Community weekend goal progress"} {
		if !strings.Contains(string(module), required) {
			t.Errorf("community weekend UI is missing %q", required)
		}
	}
}
