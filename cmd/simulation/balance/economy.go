package main

import "math/rand"

// SimAuctionItem represents an item listed on the auction house.
type SimAuctionItem struct {
	ID        int
	SellerID  int
	Type      string // "gear", "skill", "ultimate", "unique", "ench"
	Name      string
	Gear      *SimGear
	Skill     *SimSkill
	Ult       *SimUltimate
	Price     int64
	ExpiresAt int // fight number when it expires
}

// SimAuctionHouse manages the player-driven auction house.
type SimAuctionHouse struct {
	Items           []SimAuctionItem
	NextID          int
	TotalListed     int
	TotalSold       int
	TotalGoldTraded int64
}

// ListItem adds an item to the auction house.
func (ah *SimAuctionHouse) ListItem(sellerID int, itemType, name string, gear *SimGear, skill *SimSkill, ult *SimUltimate, price int64, expiresAt int) {
	if price < 10 {
		price = 10
	}
	ah.Items = append(ah.Items, SimAuctionItem{
		ID:        ah.NextID,
		SellerID:  sellerID,
		Type:      itemType,
		Name:      name,
		Gear:      gear,
		Skill:     skill,
		Ult:       ult,
		Price:     price,
		ExpiresAt: expiresAt,
	})
	ah.NextID++
	ah.TotalListed++
}

// AutoListUnwantedGear lists gear that's worse than what the player has equipped.
func (ah *SimAuctionHouse) AutoListUnwantedGear(player *SimPlayer, g *SimGear, currentFight int) {
	if g.Rarity < RarityRare {
		return // Only list rare+ items
	}
	price := int64(g.CombatRating()*10+float64(g.Stats.Score())*5) * int64(g.Rarity+1)
	ah.ListItem(player.ID, "gear", g.Name, g, nil, nil, price, currentFight+50) // Expires in ~50 fights
}

// AutoPurchaseUpgrades lets a player buy upgrades from the auction house.
func (ah *SimAuctionHouse) AutoPurchaseUpgrades(player *SimPlayer, currentFight int) {
	for i := 0; i < len(ah.Items); i++ {
		item := ah.Items[i]
		if item.SellerID == player.ID || item.ExpiresAt < currentFight || item.Price > player.Gold {
			continue
		}

		isUpgrade := false
		switch item.Type {
		case "gear":
			if item.Gear != nil && player.EquipGear(item.Gear) {
				isUpgrade = true
			}
		case "skill":
			if len(player.Skills) < 6 {
				isUpgrade = true
			}
		case "ultimate":
			if item.Ult != nil && (player.UltimateSkill == nil || item.Ult.Power > player.UltimateSkill.Power) {
				player.UltimateSkill = item.Ult
				isUpgrade = true
			}
		case "unique":
			if !player.UniqueItems[item.Name] {
				player.UniqueItems[item.Name] = true
				isUpgrade = true
			}
		case "ench":
			isUpgrade = true
		}

		if isUpgrade {
			player.Gold -= item.Price
			ah.TotalGoldTraded += item.Price
			ah.TotalSold++
			// Remove item
			ah.Items = append(ah.Items[:i], ah.Items[i+1:]...)
			i--
		}
	}
}

// RemoveExpired removes expired auction items.
func (ah *SimAuctionHouse) RemoveExpired(currentFight int) {
	var active []SimAuctionItem
	for _, item := range ah.Items {
		if item.ExpiresAt >= currentFight {
			active = append(active, item)
		}
	}
	ah.Items = active
}

// SimGoldEconomy manages gold distribution and inflation control.
type SimGoldEconomy struct {
	TotalSystemGold int64
}

// CalculateGoldReward computes gold for a victorious fight with inflation dampening.
func (ge *SimGoldEconomy) CalculateGoldReward(baseGold int64, params SimParams) int64 {
	mult := 1.0
	if ge.TotalSystemGold > params.InflationThreshold {
		mult = 1.0 / (1.0 + float64(ge.TotalSystemGold-params.InflationThreshold)/float64(params.InflationRate))
	}
	return int64(float64(baseGold) * mult)
}

// AddGold tracks gold entering the system.
func (ge *SimGoldEconomy) AddGold(amount int64) {
	ge.TotalSystemGold += amount
}

// DistributeGold awards gold to players after a victorious fight.
func DistributeGold(rng *rand.Rand, players []*SimPlayer, _ int, totalRewardGold int64, economy *SimGoldEconomy, params SimParams) {
	if len(players) == 0 {
		return
	}

	perPlayer := totalRewardGold / int64(len(players))
	// Apply inflation dampening
	perPlayer = economy.CalculateGoldReward(perPlayer, params)

	// Add some variance
	for _, p := range players {
		variance := 0.75 + rng.Float64()*0.5
		gold := int64(float64(perPlayer) * variance)
		if gold < 1 {
			gold = 1
		}
		p.Gold += gold
		p.TotalGoldEarned += gold
		economy.AddGold(gold)
	}
}
