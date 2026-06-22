-- Pity counters for high rarity loot drops
ALTER TABLE users ADD COLUMN IF NOT EXISTS artifact_pity INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS ultimate_pity INTEGER NOT NULL DEFAULT 0;
