-- Combat System Down Migration
-- Removes combat logs and enhanced combat tracking

-- Drop indexes
DROP INDEX IF EXISTS idx_combat_logs_client_uid;
DROP INDEX IF EXISTS idx_combat_logs_round;
DROP INDEX IF EXISTS idx_combat_logs_created_at;

-- Drop combat logs table
DROP TABLE IF EXISTS combat_logs;

-- Remove combat stats from tft_state
ALTER TABLE tft_state DROP COLUMN IF EXISTS total_damage_dealt;
ALTER TABLE tft_state DROP COLUMN IF EXISTS total_damage_taken;
ALTER TABLE tft_state DROP COLUMN IF EXISTS total_enemies_killed;
ALTER TABLE tft_state DROP COLUMN IF EXISTS combat_count;
