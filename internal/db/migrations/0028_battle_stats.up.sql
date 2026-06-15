-- 0028_battle_stats.up.sql
-- Add wave tracking and battle statistics to battle_history table

-- Add wave tracking columns
ALTER TABLE battle_history ADD COLUMN IF NOT EXISTS wave_number INTEGER DEFAULT 1;
ALTER TABLE battle_history ADD COLUMN IF NOT EXISTS highest_wave INTEGER DEFAULT 1;
ALTER TABLE battle_history ADD COLUMN IF NOT EXISTS damage_dealt INTEGER DEFAULT 0;
ALTER TABLE battle_history ADD COLUMN IF NOT EXISTS turns_survived INTEGER DEFAULT 0;

-- Add index for wave-based queries
CREATE INDEX IF NOT EXISTS idx_battle_history_wave ON battle_history (client_uid, wave_number DESC);

-- Create battle_stats table for tracking cumulative statistics per player
-- This enables leaderboards for total battles, win streaks, highest wave, etc.
CREATE TABLE IF NOT EXISTS battle_stats (
    client_uid       TEXT PRIMARY KEY REFERENCES users(client_uid) ON DELETE CASCADE,
    total_battles    INTEGER DEFAULT 0,
    total_wins       INTEGER DEFAULT 0,
    total_losses     INTEGER DEFAULT 0,
    current_streak   INTEGER DEFAULT 0,
    best_streak      INTEGER DEFAULT 0,
    highest_wave     INTEGER DEFAULT 1,
    total_damage     INTEGER DEFAULT 0,
    total_turns      INTEGER DEFAULT 0,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Initialize battle_stats for existing users with battle history
INSERT INTO battle_stats (client_uid, total_battles, total_wins, highest_wave, updated_at)
SELECT 
    client_uid,
    COUNT(*) as total_battles,
    COUNT(*) FILTER (WHERE victory = true) as total_wins,
    MAX(COALESCE(wave_number, 1)) as highest_wave,
    NOW() as updated_at
FROM battle_history
GROUP BY client_uid
ON CONFLICT (client_uid) DO NOTHING;

-- Create index for leaderboard queries
CREATE INDEX IF NOT EXISTS idx_battle_stats_wins ON battle_stats (total_wins DESC);
CREATE INDEX IF NOT EXISTS idx_battle_stats_streak ON battle_stats (best_streak DESC);
CREATE INDEX IF NOT EXISTS idx_battle_stats_wave ON battle_stats (highest_wave DESC);
