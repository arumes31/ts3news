package bot

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestArmoryEquippedItemsKeepsInstanceAndDurabilityAtomic(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	gear := content.Gear{
		ID: "READINESS_TEST", Name: "Readiness Blade", Slot: content.SlotMainHand,
		Rarity: content.RarityLegendary, MaxDurability: 100,
		Stats: content.Stats{STR: 42},
	}
	itemData, err := json.Marshal(gear)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT slot, gear_id, item_data, durability FROM user_gear").WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"slot", "gear_id", "item_data", "durability"}).
			AddRow(content.SlotMainHand, gear.ID, string(itemData), 37))

	items, err := (&Bot{DB: database}).armoryEquippedItems("player")
	if err != nil {
		t.Fatal(err)
	}
	views := armorySlotViews(items)
	var equipped *gearView
	for i := range views {
		if views[i].Slot == string(content.SlotMainHand) {
			equipped = &views[i]
			break
		}
	}
	if equipped == nil || equipped.Name != gear.Name || equipped.Durability != 37 || equipped.MaxDurability != 100 {
		t.Fatalf("MainHand readiness view = %+v", equipped)
	}
	if len(views) != len(content.AllSlots) {
		t.Fatalf("slot count = %d, want %d", len(views), len(content.AllSlots))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestArmoryEquippedItemsReturnsIterationError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	readErr := errors.New("connection interrupted")
	mock.ExpectQuery("SELECT slot, gear_id, item_data, durability FROM user_gear").WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"slot", "gear_id", "item_data", "durability"}).
			AddRow(content.SlotMainHand, "READINESS_TEST", `{}`, 37).
			RowError(0, readErr))

	items, err := (&Bot{DB: database}).armoryEquippedItems("player")
	if err == nil || !strings.Contains(err.Error(), readErr.Error()) {
		t.Fatalf("armoryEquippedItems() error = %v, want wrapped %v", err, readErr)
	}
	if items != nil {
		t.Fatalf("armoryEquippedItems() items = %#v, want nil after partial read", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
