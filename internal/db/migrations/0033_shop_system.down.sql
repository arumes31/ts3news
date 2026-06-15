-- Remove XP tracking from users table
ALTER TABLE users DROP COLUMN IF EXISTS xp;

-- Remove shop data from tft_state table
ALTER TABLE tft_state DROP COLUMN IF EXISTS shop_data;