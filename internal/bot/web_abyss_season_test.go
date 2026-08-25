package bot

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssSeasonCampaignAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		at   time.Time
		want string
		week int
	}{
		{name: "anchor", at: abyssSeasonAnchor, want: "Frostbound Vigil", week: 1},
		{name: "last instant", at: abyssSeasonAnchor.Add(10*abyssSeasonWeek - time.Second), want: "Frostbound Vigil", week: 10},
		{name: "next campaign", at: abyssSeasonAnchor.Add(10 * abyssSeasonWeek), want: "Verdant Reawakening", week: 1},
		{name: "current ember rotation", at: time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC), want: "Ember Descent", week: 4},
		{name: "before anchor", at: abyssSeasonAnchor.Add(-time.Second), want: "Ember Descent", week: 10},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			campaign := abyssSeasonCampaignAt(test.at)
			if campaign.Name != test.want || campaign.CurrentWeek != test.week {
				t.Fatalf("campaign = %q week %d, want %q week %d", campaign.Name, campaign.CurrentWeek, test.want, test.week)
			}
			if test.at.Before(campaign.Start) || !test.at.Before(campaign.End) {
				t.Fatalf("%s is outside campaign range [%s, %s)", test.at, campaign.Start, campaign.End)
			}
		})
	}
}

func TestAbyssSeasonJourneyAggregatesWeeklyProgress(t *testing.T) {
	t.Parallel()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	at := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	campaign := abyssSeasonCampaignAt(at)
	mock.ExpectQuery("SELECT FLOOR").WithArgs("delver", campaign.Start, campaign.End).
		WillReturnRows(sqlmock.NewRows([]string{"week_index", "floors"}).AddRow(0, 7).AddRow(3, 12))

	bot := &Bot{DB: database}
	owned := map[string]bool{
		abyssSeasonCosmeticKey(campaign, 1):        true,
		abyssSeasonPremiumEntitlementKey(campaign): true,
		abyssSeasonPremiumCosmeticKey(campaign, 1): true,
	}
	view, err := bot.abyssSeasonJourney(t.Context(), "delver", at, owned)
	if err != nil {
		t.Fatal(err)
	}
	if view.Name != "Ember Descent" || view.CurrentWeek != 4 || view.Claimed != 1 || view.PremiumClaimed != 1 || !view.PremiumUnlocked {
		t.Fatalf("unexpected journey header: %+v", view)
	}
	if got := view.Weeks[0]; got.Progress != 7 || !got.Complete || !got.Claimed {
		t.Fatalf("week 1 = %+v", got)
	}
	if got := view.Weeks[0]; !got.PremiumClaimed || got.PremiumName != "Ember Gilded Scout Sigil" {
		t.Fatalf("week 1 premium reward = %+v", got)
	}
	if got := view.Weeks[3]; got.Progress != 12 || !got.Complete || !got.Available || !got.Current {
		t.Fatalf("week 4 = %+v", got)
	}
	if got := view.Weeks[4]; got.Available {
		t.Fatalf("future week unexpectedly available: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAbyssSeasonPremiumUnlockIsAtomicAndIdempotent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		inserted     int64
		updateErr    error
		commitErr    error
		wantOK       bool
		wantAlready  bool
		wantRollback bool
	}{
		{name: "unlock", inserted: 1, wantOK: true},
		{name: "already unlocked", wantOK: true, wantAlready: true},
		{name: "insufficient tokens rolls back entitlement", inserted: 1, updateErr: sql.ErrNoRows, wantRollback: true},
		{name: "commit failure reports no unlock", inserted: 1, commitErr: sql.ErrTxDone},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()

			campaign := abyssSeasonCampaignAt(time.Now().UTC())
			mock.ExpectBegin()
			mock.ExpectExec("INSERT INTO abyss_shop_cosmetics").
				WithArgs("delver", abyssSeasonPremiumEntitlementKey(campaign)).
				WillReturnResult(sqlmock.NewResult(0, test.inserted))
			if test.inserted > 0 {
				query := mock.ExpectQuery(regexp.QuoteMeta("UPDATE users SET abyss_tokens=abyss_tokens-$1")).
					WithArgs(abyssSeasonPremiumUnlockCost, "delver")
				if test.updateErr != nil {
					query.WillReturnError(test.updateErr)
				} else {
					query.WillReturnRows(sqlmock.NewRows([]string{"abyss_tokens"}).AddRow(62))
				}
			}
			if test.wantRollback {
				mock.ExpectRollback()
			} else {
				commit := mock.ExpectCommit()
				if test.commitErr != nil {
					commit.WillReturnError(test.commitErr)
				}
			}

			server := &WebServer{bot: &Bot{DB: database}}
			request := httptest.NewRequest("POST", "/api/abyss/season/premium/unlock", nil)
			response := httptest.NewRecorder()
			server.handleAbyssSeasonPremiumUnlock(response, request, "delver")

			var body struct {
				OK              bool `json:"ok"`
				AlreadyUnlocked bool `json:"already_unlocked"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.OK != test.wantOK || body.AlreadyUnlocked != test.wantAlready {
				t.Fatalf("response = %s", response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHandleAbyssSeasonPremiumClaimRequiresEntitlement(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		entitled  bool
		inserted  int64
		wantOK    bool
		wantOwned bool
	}{
		{name: "locked", entitled: false},
		{name: "claim", entitled: true, inserted: 1, wantOK: true},
		{name: "idempotent", entitled: true, wantOK: true, wantOwned: true},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()

			campaign := abyssSeasonCampaignAt(time.Now().UTC())
			mock.ExpectQuery("SELECT FLOOR").WithArgs("delver", campaign.Start, campaign.End).
				WillReturnRows(sqlmock.NewRows([]string{"week_index", "floors"}).AddRow(0, int64(5)))
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT EXISTS").WithArgs("delver", abyssSeasonPremiumEntitlementKey(campaign)).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(test.entitled))
			if test.entitled {
				mock.ExpectExec("INSERT INTO abyss_shop_cosmetics").
					WithArgs("delver", abyssSeasonPremiumCosmeticKey(campaign, 1)).
					WillReturnResult(sqlmock.NewResult(0, test.inserted))
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}

			server := &WebServer{bot: &Bot{DB: database}}
			request := httptest.NewRequest("POST", "/api/abyss/season/premium/claim", bytes.NewBufferString(`{"week":1}`))
			response := httptest.NewRecorder()
			server.handleAbyssSeasonPremiumClaim(response, request, "delver")

			var body struct {
				OK           bool `json:"ok"`
				AlreadyOwned bool `json:"already_owned"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.OK != test.wantOK || body.AlreadyOwned != test.wantOwned {
				t.Fatalf("response = %s", response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHandleAbyssSeasonClaim(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		floors     int64
		inserted   int64
		wantOK     bool
		wantOwned  bool
		wantInsert bool
	}{
		{name: "incomplete", floors: 4},
		{name: "claim", floors: 5, inserted: 1, wantOK: true, wantInsert: true},
		{name: "idempotent", floors: 5, wantOK: true, wantOwned: true, wantInsert: true},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()

			now := time.Now().UTC()
			campaign := abyssSeasonCampaignAt(now)
			mock.ExpectQuery("SELECT FLOOR").WithArgs("delver", campaign.Start, campaign.End).
				WillReturnRows(sqlmock.NewRows([]string{"week_index", "floors"}).AddRow(0, test.floors))
			if test.wantInsert {
				mock.ExpectExec("INSERT INTO abyss_shop_cosmetics").
					WithArgs("delver", abyssSeasonCosmeticKey(campaign, 1)).
					WillReturnResult(sqlmock.NewResult(0, test.inserted))
			}

			server := &WebServer{bot: &Bot{DB: database}}
			request := httptest.NewRequest("POST", "/api/abyss/season/claim", bytes.NewBufferString(`{"week":1}`))
			response := httptest.NewRecorder()
			server.handleAbyssSeasonClaim(response, request, "delver")

			var body struct {
				OK           bool `json:"ok"`
				AlreadyOwned bool `json:"already_owned"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.OK != test.wantOK || body.AlreadyOwned != test.wantOwned {
				t.Fatalf("response = %s", response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAbyssSeasonAssetsAreWired(t *testing.T) {
	t.Parallel()
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	partial, err := webAssets.ReadFile("webassets/abyss_season.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []struct {
		body []byte
		text string
	}{
		{body: page, text: `{{template "abyssSeasonJourney" .}}`},
		{body: page, text: `/static/abyss_season.css`},
		{body: partial, text: `data-abyss-section="season"`},
		{body: partial, text: `/api/abyss/season/claim`},
		{body: partial, text: `/api/abyss/season/premium/unlock`},
		{body: partial, text: `/api/abyss/season/premium/claim`},
	} {
		if !bytes.Contains(marker.body, []byte(marker.text)) {
			t.Errorf("missing season asset marker %q", marker.text)
		}
	}
}
