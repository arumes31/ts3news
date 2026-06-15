-- UI/UX System Migration
-- Adds player settings storage

-- Add player settings table
CREATE TABLE IF NOT EXISTS player_settings (
    client_uid TEXT PRIMARY KEY,
    settings JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_player_settings_client_uid ON player_settings(client_uid);
