-- Floors-since-last-gear-drop streak, distinct from legendary_pity. Boosts loot
-- find odds the longer a run goes without a gear item (potions/consumables don't
-- count and don't reset it).
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_drop_streak INTEGER NOT NULL DEFAULT 0 CHECK (abyss_drop_streak >= 0);
