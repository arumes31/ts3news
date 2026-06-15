-- Remove Gold Economy enhancements from tft_state table
-- This removes the gold economy columns

-- Drop the columns added for gold economy
ALTER TABLE tft_state DROP COLUMN IF EXISTS battle_gold;
ALTER TABLE tft_state DROP COLUMN IF EXISTS streak;
ALTER TABLE tft_state DROP COLUMN IF EXISTS interest_gold;
ALTER TABLE tft_state DROP COLUMN IF EXISTS gold_cap;
