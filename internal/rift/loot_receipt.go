package rift

// LootReceipt preserves drop provenance independently of later room transitions.
func (d Drop) LootReceipt() BankedLoot {
	if d.Gear == nil {
		return BankedLoot{}
	}
	return BankedLoot{Name: d.Gear.Name, Rarity: int(d.Gear.Rarity), Mission: d.Mission, Tier: d.Tier, Origin: d.Gear.FoundBoss, FoundAt: d.Gear.FoundAt}
}
