package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"

	"ts3news/internal/content"

	"github.com/DATA-DOG/go-sqlmock"
)

func legacyUnidentifiedAuctionPayload(t *testing.T) []byte {
	t.Helper()
	payload, err := json.Marshal(content.Gear{
		ID: "SECRET_HEAD", Name: "Crown of the Last Star", MaxDurability: 80, Unidentified: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestSettleAbyssHouseAuctionListingPaysWithoutSyntheticBuyer(t *testing.T) {
	for _, tc := range []struct {
		name         string
		price        int64
		payment      int64
		paymentError error
	}{
		{name: "minimum payout", price: 1, payment: 1},
		{name: "large listing", price: 50_000_000, payment: 5},
		{name: "payment failure rolls back sale", price: 50_000_000, payment: 5, paymentError: errors.New("payment unavailable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()

			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT seller_uid,price\s+FROM auction_house\s+WHERE id=\$1.*FOR UPDATE`).
				WithArgs("old-listing").
				WillReturnRows(sqlmock.NewRows([]string{"seller_uid", "price"}).AddRow("seller", tc.price))
			// A vendor has no users row. The nullable buyer preserves the foreign
			// key, while sold_at records the completed sale.
			mock.ExpectExec(`UPDATE auction_house SET buyer_uid = NULL, sold_at = NOW\(\)\s+WHERE id = \$1 AND sold_at IS NULL AND bidder_uid IS NULL`).
				WithArgs("old-listing").WillReturnResult(sqlmock.NewResult(0, 1))
			payment := mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET gold = gold + $1 WHERE client_uid = $2")).
				WithArgs(tc.payment, "seller")
			if tc.paymentError != nil {
				payment.WillReturnError(tc.paymentError)
				mock.ExpectRollback()
			} else {
				payment.WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			}

			err = (&Bot{DB: database}).settleAbyssHouseAuctionListing("old-listing")
			if !errors.Is(err, tc.paymentError) {
				t.Fatalf("settlement error = %v, want %v", err, tc.paymentError)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAutoPurchaseUpgradesExcludesCompletedSales(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	// A vendor sale or a sale whose buyer was deleted has no buyer UID but
	// must remain sold even if its expiry is in the future.
	mock.ExpectQuery(`SELECT id, item_type, item_id, item_name, item_data, price, seller_uid\s+FROM auction_house\s+WHERE sold_at IS NULL AND buyer_uid IS NULL AND expires_at > NOW\(\)`).
		WithArgs(int64(100), "buyer").
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_type", "item_id", "item_name", "item_data", "price", "seller_uid"}))
	if got := (&Bot{DB: database}).autoPurchaseUpgrades("buyer", 100); got != "" {
		t.Fatalf("purchase = %q, want none", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAutoPurchaseUpgradesRollsBackWhenListingWasSold(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	gear := content.Gear{ID: "upgrade", Name: "Upgrade", Slot: content.SlotHead, Stats: content.Stats{STR: 50}}
	payload, err := json.Marshal(gear)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT id, item_type, item_id, item_name, item_data, price, seller_uid").
		WithArgs(int64(100), "buyer").
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_type", "item_id", "item_name", "item_data", "price", "seller_uid"}).
			AddRow("listing", "gear", gear.ID, gear.Name, payload, int64(50), "seller"))
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_gear").
		WithArgs("buyer", string(content.SlotHead)).WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	// House cleanup can complete after discovery and before the claim. The
	// purchase rereads the listing under a row lock and moves no gold when it
	// is no longer available.
	mock.ExpectQuery(`SELECT item_type, item_id, item_name, item_data, price, seller_uid, durability, current_bid, bidder_uid\s+FROM auction_house\s+WHERE id=\$1 AND sold_at IS NULL AND buyer_uid IS NULL AND expires_at > NOW\(\)\s+FOR UPDATE`).
		WithArgs("listing").WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	if got := (&Bot{DB: database}).autoPurchaseUpgrades("buyer", 100); got != "" {
		t.Fatalf("purchase = %q, want no purchase", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupAuctionHouseRecoversHiddenGearBeforeOtherSettlement(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	// sqlmock checks expectations in order. Recovery must run first; the final
	// query must independently exclude any unidentified row recovery could not
	// safely process on this pass.
	mock.ExpectQuery(`SELECT id::text FROM auction_house\s+WHERE sold_at IS NULL AND item_type='gear'.*unidentified`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT id::text FROM auction_house WHERE sold_at IS NULL AND expires_at<=NOW\(\)`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT id::text FROM auction_house\s+WHERE .*unidentified`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	(&Bot{DB: database}).CleanupAuctionHouse()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSettleAbyssHouseAuctionListingDoesNotPayWhenClaimLoses(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT seller_uid,price\s+FROM auction_house\s+WHERE id=\$1.*FOR UPDATE`).
		WithArgs("stale-listing").
		WillReturnRows(sqlmock.NewRows([]string{"seller_uid", "price"}).AddRow("seller", int64(50_000_000)))
	mock.ExpectExec(`UPDATE auction_house SET buyer_uid = NULL, sold_at = NOW\(\)\s+WHERE id = \$1 AND sold_at IS NULL AND bidder_uid IS NULL`).
		WithArgs("stale-listing").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err = (&Bot{DB: database}).settleAbyssHouseAuctionListing("stale-listing")
	if err != nil {
		t.Fatalf("lost House claim: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverLegacyUnidentifiedAuctionListingReturnsUnbidGear(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	payload := legacyUnidentifiedAuctionPayload(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT seller_uid,item_id,item_data,durability,current_bid,bidder_uid").
		WithArgs("listing-1").
		WillReturnRows(sqlmock.NewRows([]string{"seller_uid", "item_id", "item_data", "durability", "current_bid", "bidder_uid"}).
			AddRow("seller", "SECRET_HEAD", payload, nil, int64(0), nil))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO user_inventory (client_uid,gear_id,durability,item_data) VALUES ($1,$2,$3,$4)")).
		WithArgs("seller", "SECRET_HEAD", 80, payload).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM auction_house WHERE id=$1 AND sold_at IS NULL")).
		WithArgs("listing-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = (&Bot{DB: database}).recoverLegacyUnidentifiedAuctionListing("listing-1")
	if err != nil {
		t.Fatalf("recover listing: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverLegacyUnidentifiedAuctionListingRefundsReservedBid(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	payload := legacyUnidentifiedAuctionPayload(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT seller_uid,item_id,item_data,durability,current_bid,bidder_uid").
		WithArgs("listing-2").
		WillReturnRows(sqlmock.NewRows([]string{"seller_uid", "item_id", "item_data", "durability", "current_bid", "bidder_uid"}).
			AddRow("seller", "SECRET_HEAD", payload, int64(37), int64(625), "bidder"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET gold=gold+$1 WHERE client_uid=$2")).
		WithArgs(int64(625), "bidder").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO user_inventory (client_uid,gear_id,durability,item_data) VALUES ($1,$2,$3,$4)")).
		WithArgs("seller", "SECRET_HEAD", 37, payload).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM auction_house WHERE id=$1 AND sold_at IS NULL")).
		WithArgs("listing-2").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO abyss_economy_events").
		WithArgs("bidder", "Bid refunded: an unidentified auction listing was returned to its owner.", int64(625)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = (&Bot{DB: database}).recoverLegacyUnidentifiedAuctionListing("listing-2")
	if err != nil {
		t.Fatalf("recover listing: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverLegacyUnidentifiedAuctionListingRollsBackFailedRefund(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	payload := legacyUnidentifiedAuctionPayload(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT seller_uid,item_id,item_data,durability,current_bid,bidder_uid").
		WithArgs("listing-3").
		WillReturnRows(sqlmock.NewRows([]string{"seller_uid", "item_id", "item_data", "durability", "current_bid", "bidder_uid"}).
			AddRow("seller", "SECRET_HEAD", payload, int64(37), int64(625), "bidder"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET gold=gold+$1 WHERE client_uid=$2")).
		WithArgs(int64(625), "bidder").
		WillReturnError(errors.New("refund unavailable"))
	mock.ExpectRollback()

	err = (&Bot{DB: database}).recoverLegacyUnidentifiedAuctionListing("listing-3")
	if err == nil {
		t.Fatal("recover listing succeeded after refund failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverLegacyUnidentifiedAuctionListingRollsBackRestoredGearIfDeleteMisses(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	payload := legacyUnidentifiedAuctionPayload(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT seller_uid,item_id,item_data,durability,current_bid,bidder_uid").
		WithArgs("listing-missing").
		WillReturnRows(sqlmock.NewRows([]string{"seller_uid", "item_id", "item_data", "durability", "current_bid", "bidder_uid"}).
			AddRow("seller", "SECRET_HEAD", payload, int64(37), int64(0), nil))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO user_inventory (client_uid,gear_id,durability,item_data) VALUES ($1,$2,$3,$4)")).
		WithArgs("seller", "SECRET_HEAD", 37, payload).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM auction_house WHERE id=$1 AND sold_at IS NULL")).
		WithArgs("listing-missing").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err = (&Bot{DB: database}).recoverLegacyUnidentifiedAuctionListing("listing-missing")
	if err == nil || !strings.Contains(err.Error(), "affected 0 rows") {
		t.Fatalf("recover listing error = %v, want failed removal", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverLegacyUnidentifiedAuctionListingRestoresGearWhenBidderWasDeleted(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	payload := legacyUnidentifiedAuctionPayload(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT seller_uid,item_id,item_data,durability,current_bid,bidder_uid").
		WithArgs("listing-4").
		WillReturnRows(sqlmock.NewRows([]string{"seller_uid", "item_id", "item_data", "durability", "current_bid", "bidder_uid"}).
			AddRow("seller", "SECRET_HEAD", payload, nil, int64(625), nil))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO user_inventory (client_uid,gear_id,durability,item_data) VALUES ($1,$2,$3,$4)")).
		WithArgs("seller", "SECRET_HEAD", 80, payload).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM auction_house WHERE id=$1 AND sold_at IS NULL")).
		WithArgs("listing-4").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = (&Bot{DB: database}).recoverLegacyUnidentifiedAuctionListing("listing-4")
	if err != nil {
		t.Fatalf("recover orphaned listing: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyAuctionRecoveryOnlyAcceptsUnidentifiedGear(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	payload, err := json.Marshal(content.Gear{ID: "KNOWN_HEAD", Name: "Known Crown", MaxDurability: 80})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT seller_uid,item_id,item_data,durability,current_bid,bidder_uid").
		WithArgs("listing-5").
		WillReturnRows(sqlmock.NewRows([]string{"seller_uid", "item_id", "item_data", "durability", "current_bid", "bidder_uid"}).
			AddRow("seller", "KNOWN_HEAD", payload, nil, int64(0), nil))
	mock.ExpectRollback()

	err = (&Bot{DB: database}).recoverLegacyUnidentifiedAuctionListing("listing-5")
	if err == nil || !strings.Contains(err.Error(), "not unidentified") {
		t.Fatalf("recover listing error = %v, want non-unidentified rejection", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAutoPurchaseUpgradesSettlesLikeBuyNow(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	gear := content.Gear{ID: "upgrade", Name: "Upgrade", Slot: content.SlotHead, Stats: content.Stats{STR: 50}, MaxDurability: 100}
	payload, err := json.Marshal(gear)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT id, item_type, item_id, item_name, item_data, price, seller_uid").
		WithArgs(int64(100), "buyer").
		WillReturnRows(sqlmock.NewRows([]string{"id", "item_type", "item_id", "item_name", "item_data", "price", "seller_uid"}).
			AddRow("listing", "gear", gear.ID, gear.Name, payload, int64(50), "seller"))
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_gear").
		WithArgs("buyer", string(content.SlotHead)).WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT item_type, item_id, item_name, item_data, price, seller_uid, durability, current_bid, bidder_uid").
		WithArgs("listing").
		WillReturnRows(sqlmock.NewRows([]string{"item_type", "item_id", "item_name", "item_data", "price", "seller_uid", "durability", "current_bid", "bidder_uid"}).
			AddRow("gear", gear.ID, gear.Name, payload, int64(50), "seller", int64(12), int64(30), "bidder"))
	// The displaced bid reservation is refunded.
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET gold = gold + $1 WHERE client_uid = $2")).
		WithArgs(int64(30), "bidder").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1")).
		WithArgs(int64(50), "buyer").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE auction_house SET buyer_uid = $1, sold_at = NOW(), current_bid = 0, bidder_uid = NULL WHERE id = $2 AND sold_at IS NULL AND buyer_uid IS NULL")).
		WithArgs("buyer", "listing").WillReturnResult(sqlmock.NewResult(0, 1))
	// The seller receives the price net of the 5% community tax (ceil(2.5)=3g),
	// and the tax funds the Abyss jackpot.
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET gold = gold + $1 WHERE client_uid = $2")).
		WithArgs(int64(47), "seller").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE arcade_jackpots SET amount=amount+$1,updated_at=NOW() WHERE game_key='abyss'")).
		WithArgs(int64(3)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO abyss_economy_events").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT forge_undo FROM users WHERE client_uid=$1 FOR UPDATE")).
		WithArgs("buyer").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("buyer"))
	mock.ExpectQuery("SELECT gear_id, durability, item_data FROM user_gear").
		WithArgs("buyer", string(content.SlotHead)).WillReturnError(sql.ErrNoRows)
	// The listing's saved durability is preserved, not reset to maximum.
	mock.ExpectExec("INSERT INTO user_gear").
		WithArgs("buyer", string(content.SlotHead), gear.ID, 12, payload).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if got := (&Bot{DB: database}).autoPurchaseUpgrades("buyer", 100); got == "" {
		t.Fatal("expected an automatic purchase")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
