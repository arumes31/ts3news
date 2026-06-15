-- 0028_battle_stats.down.sql
-- Revert battle stats additions

-- Drop indexes
DROP INDEX IF EXISTS idx_battle_stats_wins;
DROP INDEX IF EXISTS idx_battle_stats_streak;
DROP INDEX IF EXISTS idx_battle_stats_wave;
DROP INDEX IF EXISTS idx_battle_history_wave;

-- Drop battle_stats table
DROP TABLE IF EXISTS battle_stats;

-- Remove columns from battle_history
ALTER TABLE battle_history DROP COLUMN IF EXISTS wave_number;
ALTER TABLE battle_history DROP COLUMN IF EXISTS highest_wave;
ALTER TABLE battle_history DROP COLUMN IF EXISTS damage_dealt;
ALTER TABLE battle_history DROP COLUMN IF EXISTS turns_survived;
