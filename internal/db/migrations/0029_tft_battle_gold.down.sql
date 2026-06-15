-- Remove battle_gold field from tft_state
ALTER TABLE tft_state DROP COLUMN IF EXISTS battle_gold;
