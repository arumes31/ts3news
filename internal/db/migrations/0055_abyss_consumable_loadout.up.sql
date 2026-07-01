-- Per-run consumable loadout. When a player carries more consumables than the
-- Abyss carry cap, they pick a subset to bring; that subset is stored here as a
-- JSON object {cons_id: count}. NULL means "no restriction" (entered under the
-- cap, so every owned consumable is usable this run).
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS consumables JSONB;
