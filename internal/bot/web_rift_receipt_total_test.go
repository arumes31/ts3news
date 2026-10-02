package bot

import (
	"context"
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"strconv"
	"testing"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

func TestRiftReceiptTotalIndependentOfPresentation(t *testing.T) {
	r := riftVitalsFixture()
	r.Practice = nil
	r.BankedItems = []string{"Recent sword"}
	r.PastExpeditions.Gear = 9
	raw, err := encodeRift(r)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err = json.Unmarshal([]byte(raw), &fields); err != nil {
		t.Fatal(err)
	}
	fields["banked_items_total"] = 1234
	data, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := decodeRift(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if saved.RecordedTotals().Gear != 1243 {
		t.Fatalf("career total uses presentation length: %d", saved.RecordedTotals().Gear)
	}
	saved.RecordEncounterSummary("cleared")
	if saved.LastEncounter.BankedItemsCount != 1234 {
		t.Fatal("encounter summary lost older item total")
	}
	encoded, err := encodeRift(saved)
	if err != nil {
		t.Fatal(err)
	}
	again, err := decodeRift(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if again.RecordedTotals().Gear != 1243 || len(again.BankedItems) != 1 {
		t.Fatal("save changed total or receipt")
	}
}

func TestRiftBankIncrementsIndependentTotalOnce(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	r := riftVitalsFixture()
	r.Practice = nil
	r.BankedItems = []string{"Recent sword"}
	r.BankedItemsTotal = 1234
	for i := 0; i < 250; i++ {
		r.Drops = append(r.Drops, rift.Drop{ID: strconv.Itoa(i), Collected: true, Gear: &content.Gear{ID: "test", Name: "New sword", MaxDurability: 80}})
	}
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config").WithArgs("rift_brawl", "bank", r.ID, "").WillReturnResult(sqlmock.NewResult(0, 1))
	for i := 0; i < 250; i++ {
		mock.ExpectExec("INSERT INTO user_inventory").WithArgs("owner", "test", 80, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectExec("SELECT set_config").WithArgs("rift_brawl", "bank", r.ID, "").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = bankRift(context.Background(), tx, "owner", "bank", r); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if r.BankedItemsTotal != 1484 || r.TotalBankedItems() != 1484 || len(r.BankedItems) != rift.ReceiptHistoryLimit || !r.Drops[0].Banked {
		t.Fatal("banked total or delivery replayed")
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
