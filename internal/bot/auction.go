package bot

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

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

// ListUnwantedItems automatically lists rare+ items that are worse than current gear
func (b *Bot) autoListUnwantedItems(uid string, item interface{}) {
	var g content.Gear
	var itype string

	switch v := item.(type) {
	case content.Gear:
		if v.Rarity < content.RarityRare {
			return
		}
		g = v
		itype = "gear"
	default:
		return
	}

	// Check if player already has better gear in this slot
	var currentID string
	err := b.DB.QueryRow("SELECT gear_id FROM user_gear WHERE client_uid=$1 AND slot=$2", uid, string(g.Slot)).Scan(&currentID)
	switch err {
	case nil:
		if cur, ok := content.GetGearByID(currentID); ok {
			if cur.Rarity >= g.Rarity && cur.CombatRating() >= g.CombatRating() {
				// Item is unwanted, list it!
				// Price based on stats (GS, CR) and Rarity
				price := int64(g.CombatRating()*10+float64(g.Stats.Score())*5) * (int64(g.Rarity) + 1)
				if price < 10 {
					price = 10
				}
				b.listAuctionItem(uid, itype, g.ID, g.Name, g, price)
			}
		}
	case sql.ErrNoRows:
		// Even if slot is empty, we might want to list it if we don't want to equip it
		// (though usually shouldEquip handles this before autoList)
	default:
		// Other error
	}
}

func (b *Bot) listAuctionItem(uid, itype, id, name string, data interface{}, price int64) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal AH item data: %v", err)
		return
	}
	expires := time.Now().Add(24 * time.Hour)

	_, err = b.DB.Exec(`INSERT INTO auction_house (seller_uid, item_type, item_id, item_name, item_data, price, expires_at) 
	                     VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uid, itype, id, name, dataJSON, price, expires)
	if err != nil {
		log.Printf("Failed to list item on AH: %v", err)
	}
}

// GearDropResult describes what happened when a gear item was awarded.
type GearDropResult struct {
	Action   string // "equipped", "listed", "inventoried"
	ItemName string
	Prefix   string // emoji prefix for display
}

// awardGearDrop handles a gear drop from game loot sources.
// It auto-equips upgrades, auto-lists non-upgrade Rare+ items on AH,
// and puts everything else into inventory.
func (b *Bot) awardGearDrop(uid string, g content.Gear) GearDropResult {
	itemName := g.Rarity.String() + " " + g.Name

	if b.shouldEquip(uid, g) {
		// Equip the item directly
		_, err := b.DB.Exec(`INSERT INTO user_gear (client_uid, slot, gear_id, durability)
		                     VALUES ($1, $2, $3, $4)
		                     ON CONFLICT (client_uid, slot) DO UPDATE SET gear_id = EXCLUDED.gear_id, durability = EXCLUDED.durability`,
			uid, string(g.Slot), g.ID, g.MaxDurability)
		if err == nil {
			return GearDropResult{
				Action:   "equipped",
				ItemName: itemName,
				Prefix:   "⬆️ Equipped: ",
			}
		}
		// Fall through to inventory on error
	} else if g.Rarity >= content.RarityRare {
		// List on auction house
		b.autoListUnwantedItems(uid, g)
		return GearDropResult{
			Action:   "listed",
			ItemName: itemName,
			Prefix:   "🏷️ Listed on AH: ",
		}
	}

	// Fallback: insert into inventory
	_, _ = b.DB.Exec("INSERT INTO user_inventory (client_uid, gear_id, durability) VALUES ($1,$2,$3)",
		uid, g.ID, g.MaxDurability)
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
		WHERE buyer_uid IS NULL AND expires_at > NOW() AND price <= $1
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
					res, err = tx.Exec("UPDATE auction_house SET buyer_uid = $1, sold_at = NOW() WHERE id = $2 AND buyer_uid IS NULL", uid, ahID)
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
					_, err = tx.Exec(`INSERT INTO user_gear (client_uid, slot, gear_id, durability) 
					                  VALUES ($1, $2, $3, $4) 
					                  ON CONFLICT (client_uid, slot) DO UPDATE SET gear_id = $3, durability = $4`,
						uid, string(g.Slot), g.ID, g.MaxDurability)
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
