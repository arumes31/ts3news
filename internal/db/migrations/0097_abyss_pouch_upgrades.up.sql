CREATE TABLE IF NOT EXISTS abyss_consumable_pouches (
    client_uid TEXT PRIMARY KEY REFERENCES users(client_uid) ON DELETE CASCADE,
    level INTEGER NOT NULL DEFAULT 0 CHECK (level BETWEEN 0 AND 3),
    upgraded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
