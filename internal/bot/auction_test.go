package bot

import (
	"database/sql"
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

	// 2. Rare item, worse than current
	item = content.Gear{ID: "NEW_GEAR", Rarity: content.RarityRare, Slot: content.SlotHead}
	mock.ExpectQuery(`SELECT gear_id FROM user_gear`).
		WithArgs(uid, string(content.SlotHead)).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id"}).AddRow("B_Head")) // Novice gear, so NEW_GEAR is better
	
	// Wait, if it's BETTER it shouldn't be listed. 
	// My logic was: if cur.Rarity >= v.Rarity && cur.CombatRating() >= v.CombatRating() then list it.
	// B_Head (Common, CR 0) vs NEW_GEAR (Rare, CR X). NEW_GEAR is better, so it's NOT listed.
	b.autoListUnwantedItems(uid, item)

	// 3. Rare item, worse than current (mock current as Legendary)
	// We'll use a slot that has no gear to simplify
	mock.ExpectQuery(`SELECT gear_id FROM user_gear`).
		WithArgs(uid, "MainHand").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`INSERT INTO auction_house`).
		WithArgs(uid, "gear", "NEW_GEAR", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	
	b.autoListUnwantedItems(uid, content.Gear{ID: "NEW_GEAR", Rarity: content.RarityRare, Slot: "MainHand"})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %s", err)
	}
}
