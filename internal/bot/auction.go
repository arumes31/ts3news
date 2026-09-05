package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

// AuctionItem is one listing row from the auction_house table, as returned to
// the web portal.
type AuctionItem struct {
	ID        string          `json:"id"`
	SellerUID string          `json:"seller_uid"`
	ItemType  string          `json:"item_type"`
	ItemID    string          `json:"item_id"`
	ItemName  string          `json:"item_name"`
	ItemData  json.RawMessage `json:"item_data"`
	Price     int64           `json:"price"`
	ListedAt  time.Time       `json:"listed_at"`
	ExpiresAt time.Time       `json:"expires_at"`
}

// autoListUnwantedItems automatically lists items that are worse than current loadout
func (b *Bot) autoListUnwantedItems(uid string, item interface{}) bool {
	var id, name, itype string
	var price int64
	var data interface{}

	switch v := item.(type) {
	case content.Gear:
		if v.Unidentified || v.Attuned {
			return false
		}
		itype = "gear"
		id, name, data = v.ID, v.Name, v
		// Price based on stats (GS, CR) and Rarity
		price = int64(v.CombatRating()*10+float64(v.Stats.Score())*5) * (int64(v.Rarity) + 1)

		// Check if player already has better gear in this slot
		var currentID string
		err := b.DB.QueryRow("SELECT gear_id FROM user_gear WHERE client_uid=$1 AND slot=$2", uid, string(v.Slot)).Scan(&currentID)
		if err == nil {
			if cur, ok := content.GetGearByID(currentID); ok {
				if !(cur.Rarity > v.Rarity || (cur.Rarity == v.Rarity && cur.CombatRating() >= v.CombatRating())) {
					return false // This is actually an upgrade or should have been equipped
				}
				// Otherwise the currently-equipped item is unneeded gear — fall
				// through and price it fairly for listing below.
			}
		}
	case content.Skill:
		// Do not list normal skills on the Auction House
		return false
	case content.UltimateSkill:
		itype = "ultimate"
		id, name, data = v.ID, v.Name, v
		price = int64(100 + int(v.Rarity)*100)
	case content.UniqueItem:
		itype = "unique"
		id, name, data = v.Name, v.Name, v
		price = int64(250 + int(v.Rarity)*250)
	case content.Enchantment:
		itype = "enchantment"
		id, name, data = v.ID, v.Name, v
		price = int64(50 + int(v.Rarity)*50)
	default:
		return false
	}

	if price < 10 {
		price = 10
	}
	return b.listAuctionItem(uid, itype, id, name, data, price)
}

func (b *Bot) listAuctionItem(uid, itype, id, name string, data interface{}, price int64) bool {
	if gear, ok := data.(content.Gear); ok && (gear.Unidentified || gear.Attuned) {
		return false
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal AH item data: %v", err)
		return false
	}
	expires := time.Now().Add(24 * time.Hour)

	_, err = b.DB.Exec(`INSERT INTO auction_house (seller_uid, item_type, item_id, item_name, item_data, price, expires_at) 
	                     VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uid, itype, id, name, dataJSON, price, expires)
	if err != nil {
		log.Printf("Failed to list item on AH: %v", err)
		return false
	}
	b.notifyAbyssAHListing(uid, id, name, price)
	return true
}

// CleanupAuctionHouse performs maintenance on the Auction House.
// Items older than 7 days are bought by 'The House' for 0.00001% of their price (min 1g).
func (b *Bot) CleanupAuctionHouse() {
	// Legacy unidentified listings are hidden from every buyer-facing path. Return
	// them before bid settlement or House cleanup can transfer any value.
	b.recoverLegacyUnidentifiedAuctionListings()
	// Player bids reserve real gold, so settle the remaining eligible listings
	// before the legacy House cleanup sees expired buy-now listings.
	b.settleAbyssAuctionBids()
	rows, err := b.DB.Query(`
		SELECT id::text
		FROM auction_house 
		WHERE sold_at IS NULL AND bidder_uid IS NULL AND listed_at < NOW() - INTERVAL '7 days'
		  AND (item_type <> 'gear' OR LOWER(COALESCE(item_data->>'unidentified','false')) <> 'true')`)
	if err != nil {
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return
	}
	_ = rows.Close()

	for _, id := range ids {
		if err := b.settleAbyssHouseAuctionListing(id); err != nil {
			log.Printf("Failed to settle House auction listing %s: %v", id, err)
		}
	}
}

// settleAbyssHouseAuctionListing revalidates and claims one stale listing while
// holding its row lock. Seller payment happens only after this transaction wins
// the claim, so overlapping cleanup passes cannot pay twice or overwrite a sale.
func (b *Bot) settleAbyssHouseAuctionListing(id string) error {
	tx, err := b.DB.Begin()
	if err != nil {
		return fmt.Errorf("begin house auction settlement: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var sellerUID string
	var price int64
	err = tx.QueryRow(`SELECT seller_uid,price
		FROM auction_house
		WHERE id=$1 AND sold_at IS NULL AND bidder_uid IS NULL
		  AND listed_at < NOW() - INTERVAL '7 days'
		  AND (item_type <> 'gear' OR LOWER(COALESCE(item_data->>'unidentified','false')) <> 'true')
		FOR UPDATE`, id).Scan(&sellerUID, &price)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lock house auction listing: %w", err)
	}

	// The House is a vendor, not a users row. Record the sale without inventing
	// a buyer UID that would violate the auction's foreign key.
	result, err := tx.Exec(`UPDATE auction_house SET buyer_uid = NULL, sold_at = NOW()
		WHERE id = $1 AND sold_at IS NULL AND bidder_uid IS NULL`, id)
	if err != nil {
		return fmt.Errorf("claim house auction listing: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check house auction claim: %w", err)
	}
	if affected != 1 {
		return nil
	}

	// The House pays 0.00001% of the listing price, with an exact 1g floor.
	housePrice := price / 10_000_000
	if housePrice < 1 {
		housePrice = 1
	}
	result, err = tx.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid = $2", housePrice, sellerUID)
	if err != nil {
		return fmt.Errorf("pay house auction seller: %w", err)
	}
	affected, err = result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check house auction seller payment: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("pay house auction seller: affected %d users", affected)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit house auction settlement: %w", err)
	}
	return nil
}

// recoverLegacyUnidentifiedAuctionListings drains listings created before
// unidentified gear was barred from the Auction House. Candidate discovery is
// separate from recovery so no result set remains open while transactions run.
func (b *Bot) recoverLegacyUnidentifiedAuctionListings() {
	rows, err := b.DB.Query(`SELECT id::text FROM auction_house
		WHERE sold_at IS NULL AND item_type='gear'
		  AND LOWER(COALESCE(item_data->>'unidentified','false'))='true'
		ORDER BY listed_at LIMIT 25`)
	if err != nil {
		log.Printf("Failed to find legacy unidentified auction listings: %v", err)
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			log.Printf("Failed to scan legacy unidentified auction listing: %v", err)
			return
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		log.Printf("Failed to read legacy unidentified auction listings: %v", err)
		return
	}
	_ = rows.Close()

	for _, id := range ids {
		if err := b.recoverLegacyUnidentifiedAuctionListing(id); err != nil {
			log.Printf("Failed to recover legacy unidentified auction listing %s: %v", id, err)
		}
	}
}

// recoverLegacyUnidentifiedAuctionListing atomically refunds reserved gold,
// restores the exact listed gear to its owner, and removes the hidden listing.
func (b *Bot) recoverLegacyUnidentifiedAuctionListing(id string) error {
	tx, err := b.DB.Begin()
	if err != nil {
		return fmt.Errorf("begin recovery: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var sellerUID, itemID string
	var itemData []byte
	var durability sql.NullInt64
	var currentBid int64
	var bidderUID sql.NullString
	err = tx.QueryRow(`SELECT seller_uid,item_id,item_data,durability,current_bid,bidder_uid
		FROM auction_house
		WHERE id=$1 AND sold_at IS NULL AND item_type='gear'
		FOR UPDATE`, id).Scan(&sellerUID, &itemID, &itemData, &durability, &currentBid, &bidderUID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lock listing: %w", err)
	}

	var gear content.Gear
	if err := json.Unmarshal(itemData, &gear); err != nil {
		return fmt.Errorf("decode listed gear: %w", err)
	}
	if gear.ID == "" || gear.ID != itemID {
		return errors.New("listed gear ID does not match listing")
	}
	if !gear.Unidentified {
		return errors.New("listed gear is not unidentified")
	}
	if currentBid < 0 {
		return errors.New("listing has a negative reserved bid")
	}
	// bidder_uid uses ON DELETE SET NULL. If that account was deleted while its
	// bid was reserved, there is no surviving owner to refund; minting the amount
	// elsewhere would duplicate value. Return the seller's gear and remove the
	// hidden listing, but issue a refund only when the bidder still exists.
	refundBid := currentBid > 0 && bidderUID.Valid && bidderUID.String != ""
	if refundBid {
		result, err := tx.Exec("UPDATE users SET gold=gold+$1 WHERE client_uid=$2", currentBid, bidderUID.String)
		if err != nil {
			return fmt.Errorf("refund reserved bid: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("check reserved-bid refund: %w", err)
		}
		if affected != 1 {
			return fmt.Errorf("refund reserved bid: affected %d users", affected)
		}
	}

	dur := gear.MaxDurability
	if durability.Valid {
		dur = int(durability.Int64)
	}
	if _, err := tx.Exec("INSERT INTO user_inventory (client_uid,gear_id,durability,item_data) VALUES ($1,$2,$3,$4)", sellerUID, itemID, dur, itemData); err != nil {
		return fmt.Errorf("restore listed gear: %w", err)
	}
	result, err := tx.Exec("DELETE FROM auction_house WHERE id=$1 AND sold_at IS NULL", id)
	if err != nil {
		return fmt.Errorf("remove recovered listing: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check recovered-listing removal: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("remove recovered listing: affected %d rows", affected)
	}
	if refundBid {
		if _, err := tx.Exec(`INSERT INTO abyss_economy_events (client_uid,kind,message,amount)
			VALUES ($1,'bid_refund',$2,$3)`, bidderUID.String,
			"Bid refunded: an unidentified auction listing was returned to its owner.", currentBid); err != nil {
			return fmt.Errorf("record reserved-bid refund: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit recovery: %w", err)
	}
	return nil
}

// GearDropResult describes what happened when a gear item was awarded.
type GearDropResult struct {
	Action   string // "equipped", "listed", "inventoried"
	ItemName string
	Prefix   string // emoji prefix for display
}

// awardGearDrop handles a gear drop from game loot sources.
// It auto-equips upgrades, auto-lists non-upgrade items on AH,
// and puts everything else into inventory.
func (b *Bot) awardGearDrop(uid string, g content.Gear) GearDropResult {
	itemName := g.Rarity.String() + " " + g.Name
	itemDataBytes, _ := json.Marshal(g)

	if !g.Unidentified && b.shouldEquip(uid, g) {
		// Equip atomically: equipGear displaces the current piece to inventory then
		// upserts the new one — two writes. On a bare b.DB call a failure between them
		// would persist the displace but not the upsert, and the inventory fall-through
		// below would then also inventory the new drop, duplicating the old piece. A tx
		// makes it all-or-nothing.
		if tx, err := b.DB.Begin(); err == nil {
			if equipErr := b.equipGear(tx, uid, g, g.MaxDurability, string(itemDataBytes)); equipErr == nil {
				if err := tx.Commit(); err == nil {
					return GearDropResult{
						Action:   "equipped",
						ItemName: itemName,
						Prefix:   "⬆️ Equipped: ",
					}
				}
			} else {
				_ = tx.Rollback()
			}
		}
		// Fall through to inventory on any tx/equip error (nothing partial persisted).
	} else if !g.Unidentified {
		// List on auction house
		if b.autoListUnwantedItems(uid, g) {
			return GearDropResult{
				Action:   "listed",
				ItemName: itemName,
				Prefix:   "🏷️ Listed on AH: ",
			}
		}
	}

	// Fallback: insert into inventory
	_, _ = b.DB.Exec("INSERT INTO user_inventory (client_uid, gear_id, durability, item_data) VALUES ($1,$2,$3,$4)",
		uid, g.ID, g.MaxDurability, string(itemDataBytes))
	return GearDropResult{
		Action:   "inventoried",
		ItemName: itemName,
		Prefix:   "🎒 ",
	}
}

// AutoPurchaseUpgrades checks AH for upgrades the user can afford
func (b *Bot) autoPurchaseUpgrades(uid string, gold int64) string {
	// Find top 5 affordable upgrades
	rows, err := b.DB.Query(`
		SELECT id, item_type, item_id, item_name, item_data, price, seller_uid 
		FROM auction_house 
		WHERE sold_at IS NULL AND buyer_uid IS NULL AND expires_at > NOW() AND price <= $1
		  AND (item_type <> 'gear' OR LOWER(COALESCE(item_data->>'unidentified','false')) <> 'true')
		ORDER BY price DESC LIMIT 5`, gold)
	if err != nil {
		return ""
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var ahID, itype, itemID, name, sellerUID string
		var dataJSON []byte
		var price int64
		if err := rows.Scan(&ahID, &itype, &itemID, &name, &dataJSON, &price, &sellerUID); err == nil {
			if itype == "gear" {
				var g content.Gear
				if err := json.Unmarshal(dataJSON, &g); err != nil {
					log.Printf("Failed to unmarshal AH item: %v", err)
					continue
				}
				if g.Unidentified || g.Attuned {
					continue
				}
				if b.shouldEquip(uid, g) {
					// Purchase!
					tx, err := b.DB.Begin()
					if err != nil {
						continue
					}

					// 1. Deduct gold
					res, err := tx.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1", price, uid)
					if err != nil {
						_ = tx.Rollback()
						continue
					}
					rowsAffected, _ := res.RowsAffected()
					if rowsAffected == 0 {
						_ = tx.Rollback()
						continue
					}

					// 2. Mark sold (ensure it wasn't bought concurrently)
					res, err = tx.Exec("UPDATE auction_house SET buyer_uid = $1, sold_at = NOW() WHERE id = $2 AND sold_at IS NULL AND buyer_uid IS NULL", uid, ahID)
					if err != nil {
						_ = tx.Rollback()
						continue
					}
					rowsAffected, _ = res.RowsAffected()
					if rowsAffected == 0 {
						_ = tx.Rollback()
						continue
					}

					// 3. Give gold to seller
					_, err = tx.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid = $2", price, sellerUID)
					if err != nil {
						_ = tx.Rollback()
						continue
					}

					// 4. Equip item
					err = b.equipGear(tx, uid, g, g.MaxDurability, dataJSON)
					if err != nil {
						_ = tx.Rollback()
						continue
					}

					if err := tx.Commit(); err != nil {
						log.Printf("Failed to commit AH purchase: %v", err)
						_ = tx.Rollback()
						continue
					}
					return i18n.T("ah.purchase", name, FormatGold(price), "")
				}
			}
		}
	}
	return ""
}
