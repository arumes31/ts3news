-- Add columns to abyss_active for tracking rest/event floors and modifiers
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS floor_type TEXT NOT NULL DEFAULT 'combat';
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS modifier TEXT NOT NULL DEFAULT '';
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS event_state JSONB;

-- Create table to track unlocked lore fragments
CREATE TABLE IF NOT EXISTS abyss_lore_unlocked (
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    lore_id INTEGER NOT NULL,
    unlocked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, lore_id)
);

-- Create table to track defeated mobs in the Abyss
CREATE TABLE IF NOT EXISTS abyss_bestiary (
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    mob_name TEXT NOT NULL,
    kills INTEGER NOT NULL DEFAULT 1 CHECK (kills > 0),
    first_kill_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, mob_name)
);
