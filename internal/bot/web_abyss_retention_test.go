package bot

import (
	"database/sql"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestEnrichAbyssSeasonJournalBuildsFiftyObjectivesAndFinale(t *testing.T) {
	t.Parallel()
	campaign := abyssSeasonCampaignAt(abyssSeasonAnchor.Add(10*abyssSeasonWeek - time.Second))
	var progress [abyssSeasonWeeks]int64
	copy(progress[:], abyssSeasonWeekGoals)
	owned := map[string]bool{abyssSeasonJournalFinaleKey(campaign): true}
	view := abyssSeasonJourneyView{}

	enrichAbyssSeasonJournal(&view, campaign, progress, owned)

	if len(view.Objectives) != 50 || view.ObjectiveTotal != 50 || view.ObjectivesDone != 50 {
		t.Fatalf("journal totals = %d/%d with %d rows", view.ObjectivesDone, view.ObjectiveTotal, len(view.Objectives))
	}
	if !view.FinaleUnlocked || !view.FinaleClaimed || !strings.Contains(view.FinaleName, "Chronicle Mantle") {
		t.Fatalf("finale = unlocked:%v claimed:%v name:%q", view.FinaleUnlocked, view.FinaleClaimed, view.FinaleName)
	}
	for _, objective := range view.Objectives {
		if objective.ID == "" || objective.Goal <= 0 || objective.Percent != 100 || !objective.Complete {
			t.Fatalf("invalid completed objective: %+v", objective)
		}
	}
}

func TestAbyssLoginCalendarAtUsesBoundedCycleAndClaimState(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, time.August, 25, 17, 30, 0, 0, time.FixedZone("test", 3*60*60))
	preview := abyssLoginCalendarAt(at, map[string]bool{})
	claimed := map[string]bool{preview.Days[0].Date: true, preview.Days[preview.CurrentDay-1].Date: true}
	view := abyssLoginCalendarAt(at, claimed)

	if len(view.Days) != abyssLoginCalendarDays || view.CurrentDay < 1 || view.CurrentDay > abyssLoginCalendarDays {
		t.Fatalf("calendar has %d days and current day %d", len(view.Days), view.CurrentDay)
	}
	if view.Claimed != 2 || !view.TodayClaimed || view.TodayReward == "" {
		t.Fatalf("calendar claim state = %+v", view)
	}
	current := 0
	for _, day := range view.Days {
		if day.Current {
			current++
			if day.Date != utcDay(at).Format("2006-01-02") {
				t.Fatalf("current date = %s", day.Date)
			}
		}
	}
	if current != 1 {
		t.Fatalf("current calendar cells = %d", current)
	}
	if gold, tokens := abyssLoginReward(7); gold != 0 || tokens != 1 {
		t.Fatalf("day 7 reward = %dg/%d tokens", gold, tokens)
	}
	if gold, tokens := abyssLoginReward(28); gold != 0 || tokens != 4 {
		t.Fatalf("day 28 reward = %dg/%d tokens", gold, tokens)
	}
}

func TestAbyssLoginClaimedDatesBoundsQueryToActiveCycle(t *testing.T) {
	t.Parallel()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	at := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	calendar := abyssLoginCalendarAt(at, map[string]bool{})
	prefix := abyssLoginClaimPrefix("delver")
	mock.ExpectQuery("SELECT key FROM app_meta WHERE key >=").
		WithArgs(prefix+calendar.Days[0].Date, prefix+calendar.Days[len(calendar.Days)-1].Date).
		WillReturnRows(sqlmock.NewRows([]string{"key"}).
			AddRow(prefix + calendar.Days[0].Date).
			AddRow(prefix + calendar.Days[calendar.CurrentDay-1].Date))

	bot := &Bot{DB: database}
	claimed, err := bot.abyssLoginClaimedDates(t.Context(), "delver", at)
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 2 || !claimed[calendar.Days[0].Date] || !claimed[calendar.Days[calendar.CurrentDay-1].Date] {
		t.Fatalf("claimed dates = %#v", claimed)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssWeeklyDigestFromMetricsReportsTrendAndNearRecord(t *testing.T) {
	t.Parallel()
	view := abyssWeeklyDigestFromMetrics(abyssDigestMetrics{
		Depth: 48, Gold: 70_000, Runs: 5, Floors: 32, PreviousDepth: 44,
		PreviousGold: 50_000, PreviousRuns: 7, PreviousFloors: 25,
	}, 50, 5)
	if view.DepthTrend != 4 || view.GoldPerDay != 14_000 || view.GoldTrend != 20_000 ||
		view.RunsTrend != -2 || view.FloorsTrend != 7 || view.NearRecord != "2 floors from your record" {
		t.Fatalf("digest = %+v", view)
	}
}

func TestAbyssEndlessProgramGrantsOnlyDepthCosmetics(t *testing.T) {
	t.Parallel()
	owned := map[string]bool{abyssEndlessCosmeticKey(125): true}
	view := abyssEndlessProgram(175, owned)
	if !view.Unlocked || view.Renown != 15 || len(view.Rewards) != 6 || view.NextDepth != 200 {
		t.Fatalf("endless view = %+v", view)
	}
	for _, reward := range view.Rewards {
		if reward.Key != abyssEndlessCosmeticKey(reward.Depth) || !strings.Contains(reward.Name, "Endless") {
			t.Fatalf("reward is not a cosmetic milestone: %+v", reward)
		}
	}
	if !view.Rewards[0].Owned || !view.Rewards[2].Available || view.Rewards[3].Available {
		t.Fatalf("endless availability = %+v", view.Rewards)
	}
	for _, depth := range []int{0, 100, 124, 126, abyssEndlessMaxClaimDepth + abyssEndlessRewardStep} {
		if validAbyssEndlessRewardDepth(depth) {
			t.Errorf("depth %d unexpectedly valid", depth)
		}
	}
	for _, depth := range []int{125, 150, abyssEndlessMaxClaimDepth} {
		if !validAbyssEndlessRewardDepth(depth) {
			t.Errorf("depth %d unexpectedly invalid", depth)
		}
	}
}

func TestClaimAbyssLoginIsAtomicAndIdempotent(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		inserted    int64
		wantAlready bool
	}{
		{name: "first claim", inserted: 1},
		{name: "repeat claim", wantAlready: true},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			at := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
			calendar := abyssLoginCalendarAt(at, map[string]bool{})
			rewardGold, rewardTokens := abyssLoginReward(calendar.CurrentDay)
			if test.wantAlready {
				rewardGold, rewardTokens = 0, 0
			}
			mock.ExpectBegin()
			mock.ExpectExec("INSERT INTO app_meta").
				WithArgs(abyssLoginClaimKey("delver", at), "day:"+itoa(calendar.CurrentDay)).
				WillReturnResult(sqlmock.NewResult(0, test.inserted))
			mock.ExpectQuery(regexp.QuoteMeta("UPDATE users")).
				WithArgs(rewardGold, rewardTokens, "delver").
				WillReturnRows(sqlmock.NewRows([]string{"gold", "abyss_tokens"}).AddRow(10_000+rewardGold, 12+rewardTokens))
			mock.ExpectCommit()

			server := &WebServer{bot: &Bot{DB: database}}
			result, err := server.claimAbyssLogin(t.Context(), "delver", at)
			if err != nil {
				t.Fatal(err)
			}
			if result.AlreadyClaimed != test.wantAlready {
				t.Fatalf("already claimed = %v", result.AlreadyClaimed)
			}
			if test.wantAlready && (result.RewardGold != 0 || result.RewardTokens != 0) {
				t.Fatalf("repeat reward = %+v", result)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestClaimAbyssLoginRollsBackOnRewardFailure(t *testing.T) {
	t.Parallel()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	at := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	calendar := abyssLoginCalendarAt(at, map[string]bool{})
	rewardGold, rewardTokens := abyssLoginReward(calendar.CurrentDay)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(abyssLoginClaimKey("delver", at), "day:"+itoa(calendar.CurrentDay)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE users")).
		WithArgs(rewardGold, rewardTokens, "delver").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	server := &WebServer{bot: &Bot{DB: database}}
	if _, err := server.claimAbyssLogin(t.Context(), "delver", at); err == nil {
		t.Fatal("reward failure unexpectedly committed")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssRetentionAssetsAndRoutesAreWired(t *testing.T) {
	t.Parallel()
	partial, err := webAssets.ReadFile("webassets/abyss_retention.html")
	if err != nil {
		t.Fatal(err)
	}
	stylesheet, err := webAssets.ReadFile("webassets/abyss_retention.css")
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	routes, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name string
		body string
		want []string
	}{
		{name: "partial", body: string(partial), want: []string{
			`define "abyssRetention"`, "Daily descent calendar", "Seven-day run digest", "Endless renown",
			"/api/abyss/retention/login", "/api/abyss/retention/endless",
		}},
		{name: "stylesheet", body: string(stylesheet), want: []string{".ab-login-days", ".ab-weekly-digest", ".ab-endless-rail", "@media (max-width: 520px)"}},
		{name: "page", body: string(page), want: []string{"/static/abyss_retention.css", `template "abyssRetention"`, `template "abyssRetentionJS"`}},
		{name: "routes", body: string(routes), want: []string{"/api/abyss/season/journal/finale", "/api/abyss/retention/login", "/api/abyss/retention/endless"}},
	}
	for _, check := range checks {
		for _, want := range check.want {
			if !strings.Contains(check.body, want) {
				t.Errorf("%s is missing %q", check.name, want)
			}
		}
	}
}
