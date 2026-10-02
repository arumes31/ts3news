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
	r.BoundReceiptHistory()
}

// ReceiptHistoryLimit bounds recent presentation, never inventory delivery.
const ReceiptHistoryLimit = 200

// BoundReceiptHistory derives the legacy total before dropping old presentation.
// Exact-sized copies release oversized backing arrays from old saves or append growth.
func (r *Run) BoundReceiptHistory() {
	r.BankedItemsTotal = r.TotalBankedItems()
	if len(r.BankedItems) > ReceiptHistoryLimit {
		recent := make([]string, ReceiptHistoryLimit)
		copy(recent, r.BankedItems[len(r.BankedItems)-ReceiptHistoryLimit:])
		r.BankedItems = recent
	}
	if len(r.BankedLoot) > ReceiptHistoryLimit {
		recent := make([]BankedLoot, ReceiptHistoryLimit)
		copy(recent, r.BankedLoot[len(r.BankedLoot)-ReceiptHistoryLimit:])
		r.BankedLoot = recent
	}
}
