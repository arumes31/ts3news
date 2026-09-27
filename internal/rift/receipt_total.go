package rift

// TotalBankedItems includes confirmed items no longer present in receipt history.
// Old schema-1 snapshots omitted the total and retained every item name.
func (r *Run) TotalBankedItems() int {
	return max(r.BankedItemsTotal, len(r.BankedItems))
}

// RecordBankedLoot is called only after the inventory insertion succeeds in the
// banking transaction. It records presentation and the authoritative total together.
func (r *Run) RecordBankedLoot(drop Drop) {
	if drop.Gear == nil {
		return
	}
	r.BankedItemsTotal = r.TotalBankedItems() + 1
	r.BankedItems = append(r.BankedItems, drop.Gear.Name)
	r.BankedLoot = append(r.BankedLoot, drop.LootReceipt())
}
