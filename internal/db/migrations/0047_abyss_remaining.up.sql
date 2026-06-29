ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS last_action_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS coop_uid TEXT REFERENCES users(client_uid) ON DELETE SET NULL;

ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_prestige INTEGER NOT NULL DEFAULT 0 CHECK (abyss_prestige >= 0);

CREATE TABLE IF NOT EXISTS abyss_boss_kills (
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    boss_name TEXT NOT NULL,
    depth INTEGER NOT NULL CHECK (depth >= 0),
    kill_time_ms INTEGER NOT NULL CHECK (kill_time_ms >= 0),
    killed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
