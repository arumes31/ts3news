package bot

import (
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
	mock.ExpectExec(`UPDATE auction_house SET buyer_uid = 'HOUSE', sold_at = NOW\(\)\s+WHERE id = \$1 AND sold_at IS NULL AND bidder_uid IS NULL`).
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
