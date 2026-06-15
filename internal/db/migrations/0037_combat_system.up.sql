-- Combat System Migration
-- Adds combat logs and enhanced combat tracking

-- Add combat logs table
CREATE TABLE IF NOT EXISTS combat_logs (
    id TEXT PRIMARY KEY,
    client_uid TEXT NOT NULL,
    round_number INTEGER NOT NULL,
    stage_number INTEGER NOT NULL,
    won BOOLEAN NOT NULL,
    damage_dealt INTEGER NOT NULL,
    damage_taken INTEGER NOT NULL,
    units_survived INTEGER NOT NULL DEFAULT 0,
    enemies_killed INTEGER NOT NULL DEFAULT 0,
    frames JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_combat_logs_client_uid ON combat_logs(client_uid);
CREATE INDEX IF NOT EXISTS idx_combat_logs_round ON combat_logs(round_number);
CREATE INDEX IF NOT EXISTS idx_combat_logs_created_at ON combat_logs(created_at);

-- Add combat stats to tft_state
ALTER TABLE tft_state ADD COLUMN IF NOT EXISTS total_damage_dealt INTEGER DEFAULT 0;
ALTER TABLE tft_state ADD COLUMN IF NOT EXISTS total_damage_taken INTEGER DEFAULT 0;
ALTER TABLE tft_state ADD COLUMN IF NOT EXISTS total_enemies_killed INTEGER DEFAULT 0;
ALTER TABLE tft_state ADD COLUMN IF NOT EXISTS combat_count INTEGER DEFAULT 0;
