package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"ts3news/internal/clientquery"
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
	var itype, id, name string
	var price int64

	switch v := item.(type) {
	case content.Gear:
		if v.Rarity < content.RarityRare {
			return
		}
		itype = "gear"
		id = v.ID
		name = v.Name
		// Check if player already has better gear in this slot
		var currentID string
		err := b.DB.QueryRow("SELECT gear_id FROM user_gear WHERE client_uid=$1 AND slot=$2", uid, string(v.Slot)).Scan(&currentID)
		if err == nil {
			if cur, ok := content.GetGearByID(currentID); ok {
				if cur.Rarity >= v.Rarity && cur.CombatRating() >= v.CombatRating() {
					price = int64(v.CombatRating()*10+float64(v.Stats.Score())*5) * (int64(v.Rarity) + 1)
				} else {
					return // It's an upgrade, shouldn't be here
				}
			}
		} else if err == sql.ErrNoRows {
			price = int64(v.CombatRating()*10+float64(v.Stats.Score())*5) * (int64(v.Rarity) + 1)
		} else {
			return
		}

	case content.Skill:
		if v.Rarity < content.RarityRare {
			return
		}
		itype = "skill"
		id = v.ID
		name = v.Name
		price = int64(v.Power*100) * (int64(v.Rarity) + 1)

	case content.UltimateSkill:
		if v.Rarity < content.RarityRare {
			return
		}
		itype = "ultimate_skill"
		id = v.ID
		name = v.Name
		price = int64(v.Power*200) * (int64(v.Rarity) + 1)

	case content.UniqueItem:
		if v.Rarity < content.RarityRare {
			return
		}
		itype = "unique_item"
		id = v.Name // Unique items use name as ID
		name = v.Name
		price = int64(v.Power*500) * (int64(v.Rarity) + 1)

	case content.Enchantment:
		if v.Rarity < content.RarityRare {
			return
		}
		itype = "enchantment"
		id = v.ID
		name = v.Name
		price = int64(v.XPMultiplier*1000 + float64(v.Stats.Score())*10) * (int64(v.Rarity) + 1)

	default:
		return
	}

	if price < 10 {
		price = 10
	}
	b.listAuctionItem(uid, itype, id, name, item, price)
}

func (b *Bot) listAuctionItem(uid, itype, id, name string, data interface{}, price int64) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal AH item data: %v", err)
		return
	}
	// Items stay listed for 7 days
	expires := time.Now().Add(7 * 24 * time.Hour)
	
	_, err = b.DB.Exec(`INSERT INTO auction_house (seller_uid, item_type, item_id, item_name, item_data, price, expires_at) 
	                     VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uid, itype, id, name, dataJSON, price, expires)
	if err != nil {
		log.Printf("Failed to list item on AH: %v", err)
	}
}

// ProcessExpiredAHItems finds unsold items older than 7 days and "sells" them to a vendor for 0.1% price
func (b *Bot) processExpiredAHItems(c *clientquery.Client) {
	rows, err := b.DB.Query(`
		SELECT id, seller_uid, item_name, price 
		FROM auction_house 
		WHERE buyer_uid IS NULL AND expires_at < NOW()`)
	if err != nil {
		log.Printf("Failed to query expired AH items: %v", err)
		return
	}
	defer func() { _ = rows.Close() }()

	clients, _ := c.ClientList()

	for rows.Next() {
		var id, sellerUID, itemName string
		var price int64
		if err := rows.Scan(&id, &sellerUID, &itemName, &price); err == nil {
			vendorPrice := price / 1000
			if vendorPrice < 1 && price > 0 {
				vendorPrice = 1
			}

			tx, err := b.DB.Begin()
			if err != nil {
				continue
			}

			// 1. Give gold to seller
			_, err = tx.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid = $2", vendorPrice, sellerUID)
			if err != nil {
				_ = tx.Rollback()
				continue
			}

			// 2. Mark as vendor sold (or just delete/mark sold_at)
			_, err = tx.Exec("UPDATE auction_house SET buyer_uid = 'VENDOR', sold_at = NOW() WHERE id = $1", id)
			if err != nil {
				_ = tx.Rollback()
				continue
			}

			if err := tx.Commit(); err == nil {
				// Notify seller if online
				for _, cl := range clients {
					if cl.UID == sellerUID {
						_ = c.SendPrivateMessage(cl.CLID, fmt.Sprintf("🏪 [b]AH Vendor Sale:[/b] Your item [b]%s[/b] didn't sell and was bought by a vendor for %s (0.1%% value).", 
							itemName, FormatGold(vendorPrice)))
					}
				}
			}
		}
	}
}

func (b *Bot) isAHItemUpgrade(uid string, itype string, dataJSON []byte) bool {
	switch itype {
	case "gear":
		var g content.Gear
		if err := json.Unmarshal(dataJSON, &g); err == nil && b.shouldEquip(uid, g) {
			return true
		}
	case "skill":
		var s content.Skill
		if err := json.Unmarshal(dataJSON, &s); err == nil {
			// Check if we have an empty slot or a worse skill
			rows, err := b.DB.Query("SELECT skill_id FROM user_skills WHERE client_uid = $1", uid)
			if err != nil {
				return false
			}
			defer func() { _ = rows.Close() }()
			count := 0
			hasWorse := false
			for rows.Next() {
				var sid string
				if err := rows.Scan(&sid); err == nil {
					count++
					if cur, ok := content.GetSkillByID(sid); ok && s.Rarity > cur.Rarity {
						hasWorse = true
					}
				}
			}
			if count < 5 || hasWorse {
				return true
			}
		}
	case "ultimate_skill":
		var us content.UltimateSkill
		if err := json.Unmarshal(dataJSON, &us); err == nil {
			var exists bool
			_ = b.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM user_ultimate_skills WHERE client_uid=$1 AND ultimate_id=$2)", uid, us.ID).Scan(&exists)
			if !exists {
				return true
			}
		}
	case "unique_item":
		var ui content.UniqueItem
		if err := json.Unmarshal(dataJSON, &ui); err == nil {
			var exists bool
			_ = b.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM user_unique_items WHERE client_uid=$1 AND item_name=$2)", uid, ui.Name).Scan(&exists)
			if !exists {
				return true
			}
		}
	case "enchantment":
		var ench content.Enchantment
		if err := json.Unmarshal(dataJSON, &ench); err == nil {
			// Any rare+ enchantment is an "upgrade" (consumable)
			return true
		}
	}
	return false
}

func (b *Bot) ResolveGlobalAH(c *clientquery.Client, onlineClients []clientquery.ClientInfo) {
	// 1. Get all available items
	rows, err := b.DB.Query(`
		SELECT id, item_type, item_id, item_name, item_data, price, seller_uid 
		FROM auction_house 
		WHERE buyer_uid IS NULL AND expires_at > NOW()
		ORDER BY price DESC`)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()

	type ahItem struct {
		id, itype, itemID, name, sellerUID string
		dataJSON                           []byte
		price                              int64
	}
	var items []ahItem
	for rows.Next() {
		var it ahItem
		if err := rows.Scan(&it.id, &it.itype, &it.itemID, &it.name, &it.dataJSON, &it.price, &it.sellerUID); err == nil {
			items = append(items, it)
		}
	}

	// 2. Track online player gold (in-memory snapshot to prevent overspending)
	playerGold := make(map[string]int64)
	for _, cl := range onlineClients {
		if cl.UID == "" {
			continue
		}
		var gold int64
		_ = b.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", cl.UID).Scan(&gold)
		playerGold[cl.UID] = gold
	}

	// 3. Process each item
	for _, it := range items {
		var interested []clientquery.ClientInfo
		for _, cl := range onlineClients {
			if cl.UID == "" || cl.UID == it.sellerUID {
				continue
			}
			// Preliminary gold check (normal price)
			if playerGold[cl.UID] < it.price {
				continue
			}
			if b.isAHItemUpgrade(cl.UID, it.itype, it.dataJSON) {
				interested = append(interested, cl)
			}
		}

		if len(interested) == 0 {
			continue
		}

		// Roll the buyer
		// #nosec G404
		rand.Shuffle(len(interested), func(i, j int) {
			interested[i], interested[j] = interested[j], interested[i]
		})

		fee := int64(0)
		if len(interested) > 1 {
			fee = it.price / 10 // 10% fee for contention
		}

		success := false
		for _, buyer := range interested {
			totalPrice := it.price + fee
			if playerGold[buyer.UID] < totalPrice {
				continue // Fallback to next buyer
			}

			// Execute Transaction
			tx, err := b.DB.Begin()
			if err != nil {
				continue
			}

			// a. Deduct gold
			res, err := tx.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid = $2 AND gold >= $1", totalPrice, buyer.UID)
			if err != nil {
				_ = tx.Rollback()
				continue
			}
			if n, _ := res.RowsAffected(); n == 0 {
				_ = tx.Rollback()
				playerGold[buyer.UID] = 0 // Update cache
				continue
			}

			// b. Mark sold
			res, err = tx.Exec("UPDATE auction_house SET buyer_uid = $1, sold_at = NOW() WHERE id = $2 AND buyer_uid IS NULL", buyer.UID, it.id)
			if err != nil {
				_ = tx.Rollback()
				continue
			}
			if n, _ := res.RowsAffected(); n == 0 {
				_ = tx.Rollback() // Item already bought?
				break
			}

			// c. Give gold to seller
			_, err = tx.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid = $2", it.price, it.sellerUID)
			if err != nil {
				_ = tx.Rollback()
				continue
			}

			if err := tx.Commit(); err != nil {
				_ = tx.Rollback()
				continue
			}

			// d. Apply item (Side effect: equip/learn)
			b.applyAHItem(buyer.UID, it.itype, it.dataJSON)
			
			// e. Notifications
			playerGold[buyer.UID] -= totalPrice
			_ = c.SendPrivateMessage(buyer.CLID, fmt.Sprintf("🎁 [b]AH Purchase![/b] You bought [b]%s[/b] for %s!%s", 
				it.name, FormatGold(totalPrice), func() string {
					if fee > 0 { return fmt.Sprintf(" (Includes %s contention fee)", FormatGold(fee)) }
					return ""
				}()))
			
			// Notify seller
			for _, scl := range onlineClients {
				if scl.UID == it.sellerUID {
					_ = c.SendPrivateMessage(scl.CLID, fmt.Sprintf("💰 [b]AH Sale![/b] Your item [b]%s[/b] was bought by [b]%s[/b] for %s!", 
						it.name, buyer.Nickname, FormatGold(it.price)))
				}
			}

			success = true
			break
		}
		if success {
			// Item handled
		}
	}
}

func (b *Bot) applyAHItem(uid, itype string, dataJSON []byte) {
	switch itype {
	case "gear":
		var g content.Gear
		if err := json.Unmarshal(dataJSON, &g); err == nil {
			_, _ = b.DB.Exec(`INSERT INTO user_gear (client_uid, slot, gear_id, durability) 
			                  VALUES ($1, $2, $3, $4) 
			                  ON CONFLICT (client_uid, slot) DO UPDATE SET gear_id = $3, durability = $4`,
				uid, string(g.Slot), g.ID, g.MaxDurability)
		}
	case "skill":
		var s content.Skill
		if err := json.Unmarshal(dataJSON, &s); err == nil {
			_, _ = b.equipSkill(uid, s)
		}
	case "ultimate_skill":
		var us content.UltimateSkill
		if err := json.Unmarshal(dataJSON, &us); err == nil {
			_, _ = b.DB.Exec("INSERT INTO user_ultimate_skills (client_uid, ultimate_id) VALUES ($1, $2)", uid, us.ID)
		}
	case "unique_item":
		var ui content.UniqueItem
		if err := json.Unmarshal(dataJSON, &ui); err == nil {
			_, _ = b.DB.Exec("INSERT INTO user_unique_items (client_uid, item_name, rarity, power) VALUES ($1, $2, $3, $4)", uid, ui.Name, ui.Rarity, ui.Power)
		}
	case "enchantment":
		var ench content.Enchantment
		if err := json.Unmarshal(dataJSON, &ench); err == nil {
			_, _ = b.applyEnchantment(uid, ench)
		}
	}
}
