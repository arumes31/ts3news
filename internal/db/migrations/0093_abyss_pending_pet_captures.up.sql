CREATE TABLE IF NOT EXISTS abyss_pending_pet_captures (
    client_uid TEXT PRIMARY KEY REFERENCES users(client_uid) ON DELETE CASCADE,
    name TEXT NOT NULL,
    mob_type TEXT NOT NULL,
    level INTEGER NOT NULL CHECK (level >= 0),
    hp INTEGER NOT NULL CHECK (hp > 0),
    max_hp INTEGER NOT NULL CHECK (max_hp > 0),
    str INTEGER NOT NULL,
    def INTEGER NOT NULL,
    spd INTEGER NOT NULL,
    loyalty INTEGER NOT NULL CHECK (loyalty BETWEEN 1 AND 100),
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
