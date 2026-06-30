-- Database updates for the Abyss progression and items expansion
ALTER TABLE user_gear ADD COLUMN IF NOT EXISTS item_data JSONB;
ALTER TABLE user_inventory ADD COLUMN IF NOT EXISTS item_data JSONB;
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_upgrades JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE users ADD COLUMN IF NOT EXISTS legendary_pity INTEGER NOT NULL DEFAULT 0;
