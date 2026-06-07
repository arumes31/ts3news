package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ts3news/internal/content"
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
	if err == nil {
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
	} else if err == sql.ErrNoRows {
		// Even if slot is empty, we might want to list it if we don't want to equip it
		// (though usually shouldEquip handles this before autoList)
	}
}

func (b *Bot) listAuctionItem(uid, itype, id, name string, data interface{}, price int64) {
	dataJSON, _ := json.Marshal(data)
	expires := time.Now().Add(24 * time.Hour)
	
	_, err := b.DB.Exec(`INSERT INTO auction_house (seller_uid, item_type, item_id, item_name, item_data, price, expires_at) 
	                     VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uid, itype, id, name, dataJSON, price, expires)
	if err != nil {
		log.Printf("Failed to list item on AH: %v", err)
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
	defer rows.Close()

	for rows.Next() {
		var ahID, itype, itemID, name, sellerUID string
		var dataJSON []byte
		var price int64
		if err := rows.Scan(&ahID, &itype, &itemID, &name, &dataJSON, &price, &sellerUID); err == nil {
			if itype == "gear" {
				var g content.Gear
				json.Unmarshal(dataJSON, &g)
				if b.shouldEquip(uid, g) {
					// Purchase!
					tx, err := b.DB.Begin()
					if err != nil {
						continue
					}
					
					// 1. Deduct gold
					_, err = tx.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1", price, uid)
					if err != nil {
						tx.Rollback()
						continue
					}
					
					// 2. Mark sold
					_, err = tx.Exec("UPDATE auction_house SET buyer_uid = $1, sold_at = NOW() WHERE id = $2", uid, ahID)
					if err != nil {
						tx.Rollback()
						continue
					}
					
					// 3. Give gold to seller
					_, err = tx.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid = $2", price, sellerUID)
					if err != nil {
						tx.Rollback()
						continue
					}
					
					// 4. Equip item
					_, err = tx.Exec(`INSERT INTO user_gear (client_uid, slot, gear_id, durability) 
					                  VALUES ($1, $2, $3, $4) 
					                  ON CONFLICT (client_uid, slot) DO UPDATE SET gear_id = $3, durability = $4`,
						uid, string(g.Slot), g.ID, g.MaxDurability)
					if err != nil {
						tx.Rollback()
						continue
					}
					
					tx.Commit()
					return fmt.Sprintf("AH Purchase: %s for %s gold!", name, FormatGold(price))
				}
			}
		}
	}
	return ""
}
