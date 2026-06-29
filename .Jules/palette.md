## 2026-06-29 - Contextual Action Buttons in Item Lists
**Learning:** The item listing components (Inventory, Shop, Auction House) rely on standard action buttons like "Equip", "Vendor", "Buy", and "List". For screen reader users, iterating through these lists results in meaningless repetitive labels that lack context of the item being interacted with.
**Action:** When working on lists or table structures containing actions per row/item, always ensure that dynamic context is included in the action's `aria-label` attribute (e.g., `aria-label="Buy {{.Name}}"`).
