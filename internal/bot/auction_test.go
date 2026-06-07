package bot

import (
	"testing"
	"ts3news/internal/config"
	"ts3news/internal/content"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAutoListUnwantedItems(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	b := &Bot{Cfg: &config.Config{}, DB: db}
	uid := "user1"

	// 1. Common item - should be ignored
	item := content.Gear{Rarity: content.RarityCommon}
	b.autoListUnwantedItems(uid, item)

	// 2. Rare item, but better than current
	item = content.Gear{ID: "NEW_GEAR", Rarity: content.RarityRare, Slot: content.SlotHead}
	mock.ExpectQuery(`SELECT gear_id FROM user_gear`).
		WithArgs(uid, string(content.SlotHead)).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id"}).AddRow("OLD_GEAR"))
	// content.GetGearByID for OLD_GEAR (assume it's worse or we mock it)
	// This is hard to test without fully mocking content, but let's see.
	b.autoListUnwantedItems(uid, item)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %s", err)
	}
}
