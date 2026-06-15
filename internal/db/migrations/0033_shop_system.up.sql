-- Add XP tracking to users table for shop system
ALTER TABLE users ADD COLUMN IF NOT EXISTS xp INTEGER DEFAULT 0;

-- Add shop data to tft_state table
ALTER TABLE tft_state ADD COLUMN IF NOT EXISTS shop_data JSONB DEFAULT '{}';

-- Note: The shop data will be stored in the JSONB column with structure:
-- {
--   "units": ["brute", "archer", "mage", "knight", "rogue"],
--   "xp": 0
-- }