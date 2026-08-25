CREATE TABLE IF NOT EXISTS abyss_boss_cosmetic_loadouts (
    client_uid TEXT PRIMARY KEY REFERENCES users(client_uid) ON DELETE CASCADE,
    mount_key TEXT NOT NULL DEFAULT '',
    banner_key TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
