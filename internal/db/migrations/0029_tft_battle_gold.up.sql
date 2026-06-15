-- Add battle_gold field to tft_state for in-game currency
-- Real gold is only used for end-game rewards
ALTER TABLE tft_state
ADD COLUMN battle_gold INTEGER NOT NULL DEFAULT 0;

-- Initialize existing records with starting battle gold (0, will be set on first game)
UPDATE tft_state SET battle_gold = 0 WHERE battle_gold IS NULL;
