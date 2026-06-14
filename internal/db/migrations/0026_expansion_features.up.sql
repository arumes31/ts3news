-- 0026_expansion_features.up.sql

-- Add VIP and Daily Spin fields to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS vip_points INTEGER DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_daily_spin TIMESTAMP WITH TIME ZONE;

-- Create Arcade Jackpots table
CREATE TABLE IF NOT EXISTS arcade_jackpots (
    game_key TEXT PRIMARY KEY,
    amount BIGINT DEFAULT 10000,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Initialize default jackpots
INSERT INTO arcade_jackpots (game_key, amount) VALUES
('slots', 25000),
('megawheel', 15000),
('crash', 10000)
ON CONFLICT DO NOTHING;

-- Create a table for global event tracking (Treasure Goblins etc)
CREATE TABLE IF NOT EXISTS world_events (
    id SERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    active BOOLEAN DEFAULT TRUE,
    payload JSONB,
    starts_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);
