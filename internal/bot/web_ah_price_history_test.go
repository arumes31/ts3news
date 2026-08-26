package bot

import (
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestBuildAHPriceHistory(t *testing.T) {
	tests := []struct {
		name      string
		prices    []int64
		want      ahPriceHistoryView
		wantPoint string
	}{
		{name: "empty"},
		{name: "flat", prices: []int64{50, 50}, want: ahPriceHistoryView{Count: 2, Min: 50, Max: 50, Latest: 50, Direction: "flat"}, wantPoint: "0,15 100,15"},
		{name: "rising", prices: []int64{100, 200, 150, 300}, want: ahPriceHistoryView{Count: 4, Min: 100, Max: 300, Latest: 300, Direction: "up"}, wantPoint: "0,27 33,15 66,21 100,3"},
		{name: "falling", prices: []int64{300, 100}, want: ahPriceHistoryView{Count: 2, Min: 100, Max: 300, Latest: 100, Direction: "down"}, wantPoint: "0,3 100,27"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := buildAHPriceHistory(test.prices)
			if got.Count != test.want.Count || got.Min != test.want.Min || got.Max != test.want.Max || got.Latest != test.want.Latest || got.Direction != test.want.Direction {
				t.Fatalf("buildAHPriceHistory(%v) = %+v, want %+v", test.prices, got, test.want)
			}
			if got.Points != test.wantPoint {
				t.Fatalf("points = %q, want %q", got.Points, test.wantPoint)
			}
		})
	}
}

func TestAHPriceHistoriesBatchesVisibleItems(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectQuery(regexp.QuoteMeta("ROW_NUMBER() OVER (PARTITION BY item_id ORDER BY sold_at DESC)")).
		WithArgs(sqlmock.AnyArg(), ahPriceHistoryLimit).
		WillReturnRows(sqlmock.NewRows([]string{"item_id", "price"}).
			AddRow("blade", int64(90)).AddRow("blade", int64(120)).
			AddRow("ring", int64(300)).AddRow("ring", int64(250)))

	views := (&Bot{DB: database}).ahPriceHistories([]ahListingView{
		{ItemID: "blade"}, {ItemID: "blade"}, {ItemID: "ring"}, {ItemID: ""},
	})
	if views["blade"].Count != 2 || views["blade"].Latest != 120 || views["blade"].Direction != "up" {
		t.Fatalf("unexpected blade history: %+v", views["blade"])
	}
	if views["ring"].Count != 2 || views["ring"].Latest != 250 || views["ring"].Direction != "down" {
		t.Fatalf("unexpected ring history: %+v", views["ring"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAHPriceHistoryTemplateContract(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/ah.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/ah_price_history.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{`ge .PriceHistory.Count 2`, `aria-label="Price history: minimum`, `.PriceHistory.Points`, `/static/ah_price_history.css`} {
		if !strings.Contains(string(page), token) {
			t.Fatalf("auction template missing %q", token)
		}
	}
	for _, token := range []string{".ah-spark", ".trend-up", ".trend-down", "@media (max-width: 720px)"} {
		if !strings.Contains(string(styles), token) {
			t.Fatalf("price history stylesheet missing %q", token)
		}
	}
}
