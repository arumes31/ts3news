-- Add Gold Economy enhancements to tft_state table
-- This includes battle gold, interest, streaks, and gold caps

-- Add columns to tft_state table for gold economy
ALTER TABLE tft_state ADD COLUMN IF NOT EXISTS battle_gold INTEGER DEFAULT 0;
ALTER TABLE tft_state ADD COLUMN IF NOT EXISTS streak INTEGER DEFAULT 0;
ALTER TABLE tft_state ADD COLUMN IF NOT EXISTS interest_gold INTEGER DEFAULT 0;
ALTER TABLE tft_state ADD COLUMN IF NOT EXISTS gold_cap INTEGER DEFAULT 100;

-- Note: The actual gold values are tracked in the users table for persistence
-- The tft_state holds the current round's battle gold and streak state
